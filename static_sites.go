package cloud

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaticSitesClient manages a static site service's content and domain.
// Obtain one via Services().StaticSite(svcSlug).
type StaticSitesClient struct {
	services *ServicesClient
	svcSlug  string
	// httpClient uploads an archive straight to object storage. It is separate
	// from the API client on purpose: a presigned URL carries its own
	// authorisation in the query string, and attaching the caller's API token
	// to it would send that credential to a host that has no business seeing
	// it.
	httpClient *http.Client
}

// StaticSite returns a client for a static site service's content and domain.
func (s *ServicesClient) StaticSite(svcSlug string) *StaticSitesClient {
	return &StaticSitesClient{
		services: s,
		svcSlug:  svcSlug,
		// Generous, because one request here is a whole site archive going out
		// over whatever connection the caller has.
		httpClient: &http.Client{Timeout: 30 * time.Minute},
	}
}

func (c *StaticSitesClient) base() string {
	return c.services.svcPath(c.svcSlug) + "/site"
}

// Get returns the site's configuration and serving URLs.
func (c *StaticSitesClient) Get(ctx context.Context) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.get(ctx, c.base(), &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Publish uploads the given files to the site. File contents travel inline in
// the request, so no separate upload step or filesystem access is needed.
//
// With Prune left nil (the default), the publish is a full replace: files not
// in the request are removed, so publishing the same set twice converges on
// exactly those files. Set Prune to false to patch individual files instead.
func (c *StaticSitesClient) Publish(ctx context.Context, in PublishStaticSiteInput) (*StaticSitePublishResult, error) {
	var result StaticSitePublishResult
	if err := c.services.client.post(ctx, c.base()+"/publish", in, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublishArchive publishes a whole site from a local .zip in one call.
//
// The archive goes straight to object storage using a presigned URL and is
// expanded server-side, so — unlike Publish, whose files travel inline on the
// API request — the size of the site is not bounded by the size of a request.
// A build output is usually zipped as the directory rather than its contents;
// a single wrapping directory is removed on expansion, so dist/index.html
// still serves at the site root.
//
// The publish is a full replace unless opts.Prune says otherwise.
func (c *StaticSitesClient) PublishArchive(ctx context.Context, path string, opts ArchivePublishOptions) (*StaticSitePublishResult, error) {
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		return nil, fmt.Errorf("%s is not a .zip archive", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	// A fresh key per publish: two deploys running at once must not stage over
	// each other, and the server deletes the archive as soon as it has read it.
	key, err := stagedArchiveKey()
	if err != nil {
		return nil, err
	}
	signed, err := c.presignUpload(ctx, key)
	if err != nil {
		return nil, err
	}

	var body io.Reader = f
	if opts.Progress != nil {
		body = &progressReader{r: f, total: info.Size(), report: opts.Progress}
	}
	if err := c.putArchive(ctx, signed.URL, body, info.Size()); err != nil {
		return nil, err
	}

	return c.PublishStagedArchive(ctx, PublishStaticSiteArchiveInput{Key: key, Prune: opts.Prune})
}

// PublishStagedArchive expands an archive already uploaded to the site's own
// bucket. PublishArchive covers the usual case of publishing a local file; this
// is for a caller that put the archive there itself, such as a build that
// writes straight to storage.
func (c *StaticSitesClient) PublishStagedArchive(ctx context.Context, in PublishStaticSiteArchiveInput) (*StaticSitePublishResult, error) {
	var result StaticSitePublishResult
	if err := c.services.client.post(ctx, c.base()+"/publish-archive", in, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// staticSiteArchiveStagingPrefix keeps staged archives out of the way of the
// site's own files. Nothing is left there — the server deletes the archive once
// it has read it — but a publish that fails before then should not look like
// part of the site.
const staticSiteArchiveStagingPrefix = ".uploads/"

func stagedArchiveKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return staticSiteArchiveStagingPrefix + hex.EncodeToString(b[:]) + ".zip", nil
}

func (c *StaticSitesClient) presignUpload(ctx context.Context, key string) (*PresignedUpload, error) {
	var out PresignedUpload
	path := c.services.svcPath(c.svcSlug) + "/s3/objects/presign-upload"
	if err := c.services.client.post(ctx, path, map[string]string{"key": key}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *StaticSitesClient) putArchive(ctx context.Context, signedURL string, body io.Reader, length int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, body)
	if err != nil {
		return err
	}
	req.ContentLength = length
	req.Header.Set("Content-Type", "application/zip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return &storeError{status: resp.StatusCode, body: strings.TrimSpace(string(msg))}
	}
	return nil
}

// progressReader reports how much of the archive has gone out. The count is the
// bytes handed to the connection, which on a slow link runs ahead of what the
// store has acknowledged.
type progressReader struct {
	r      io.Reader
	sent   int64
	total  int64
	report func(sent, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.sent += int64(n)
		p.report(p.sent, p.total)
	}
	return n, err
}

// ListFiles returns every published file, recursively, under an optional
// prefix. Contents are not included — use Fetch for those.
func (c *StaticSitesClient) ListFiles(ctx context.Context, prefix string) (*StaticSiteFileList, error) {
	path := c.base() + "/files"
	if prefix != "" {
		path += "?prefix=" + url.QueryEscape(prefix)
	}
	var list StaticSiteFileList
	if err := c.services.client.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Fetch downloads the named files with their contents inline, mirroring what
// Publish accepts: a caller can fetch a file, edit it and publish it straight
// back. Text files come back as utf8 and binary files as base64.
func (c *StaticSitesClient) Fetch(ctx context.Context, paths []string) (*StaticSiteFetchResult, error) {
	var result StaticSiteFetchResult
	if err := c.services.client.post(ctx, c.base()+"/files/fetch", FetchStaticSiteFilesInput{Paths: paths}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDocuments changes which objects serve as the directory index and the 404
// body.
func (c *StaticSitesClient) SetDocuments(ctx context.Context, in UpdateStaticSiteDocumentsInput) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.put(ctx, c.base()+"/documents", in, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// SetCustomDomain attaches a custom domain to the site, or detaches the
// current one when the domain is empty. After attaching, point the domain's
// CNAME at the returned DomainCNAMETarget and deploy the service so the
// platform starts routing it.
func (c *StaticSitesClient) SetCustomDomain(ctx context.Context, domain string) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.put(ctx, c.base()+"/domain", SetStaticSiteDomainInput{CustomDomain: domain}, &site); err != nil {
		return nil, err
	}
	return &site, nil
}
