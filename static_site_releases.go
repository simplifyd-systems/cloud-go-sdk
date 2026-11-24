package cloud

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"
)

type StaticSiteReleaseOptions struct {
	Preview        bool   `json:"preview"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Repository     string `json:"repository,omitempty"`
	Message        string `json:"message,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type StaticSiteRelease struct {
	Slug          string     `json:"slug"`
	Status        string     `json:"status"`
	Active        bool       `json:"active"`
	PreviewURL    string     `json:"preview_url,omitempty"`
	CommitSHA     string     `json:"commit_sha,omitempty"`
	Branch        string     `json:"branch,omitempty"`
	Repository    string     `json:"repository,omitempty"`
	Message       string     `json:"message,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	FileCount     int64      `json:"file_count"`
	BytesUploaded int64      `json:"bytes_uploaded"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	PromotedAt    *time.Time `json:"promoted_at,omitempty"`
}

type PublishStaticSiteReleaseInput struct {
	PublishStaticSiteInput
	StaticSiteReleaseOptions
}

type PublishStaticSiteReleaseArchiveInput struct {
	PublishStaticSiteArchiveInput
	StaticSiteReleaseOptions
}

// PublishRelease uploads a complete immutable build, or clones and patches the
// current release when Prune is false. Preview never changes production.
func (c *StaticSitesClient) PublishRelease(ctx context.Context, in PublishStaticSiteReleaseInput) (*StaticSitePublishResult, error) {
	var result StaticSitePublishResult
	if err := c.services.client.post(ctx, c.base()+"/releases", in, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *StaticSitesClient) PublishStagedReleaseArchive(ctx context.Context, in PublishStaticSiteReleaseArchiveInput) (*StaticSitePublishResult, error) {
	var result StaticSitePublishResult
	if err := c.services.client.post(ctx, c.base()+"/releases/archive", in, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublishReleaseArchive stages a zip using a signed object-storage request,
// then validates and publishes it through the release API. For larger sites,
// configure the API client's timeout or supply a context deadline appropriate
// to the build size.
func (c *StaticSitesClient) PublishReleaseArchive(ctx context.Context, filename string, opts StaticSiteReleaseOptions) (*StaticSitePublishResult, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 5<<30 {
		return nil, fmt.Errorf("provide a nonempty archive of at most 5 GiB")
	}
	stage, err := c.StageArchiveUpload(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.putArchive(ctx, stage.URL, f, info.Size()); err != nil {
		return nil, err
	}
	return c.PublishStagedReleaseArchive(ctx, PublishStaticSiteReleaseArchiveInput{PublishStaticSiteArchiveInput: PublishStaticSiteArchiveInput{Key: stage.Key}, StaticSiteReleaseOptions: opts})
}

func (c *StaticSitesClient) Releases(ctx context.Context) ([]StaticSiteRelease, error) {
	var rows []StaticSiteRelease
	if err := c.services.client.get(ctx, c.base()+"/releases", &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// PromoteRelease also performs rollback: it selects an existing complete
// release for every production domain without rebuilding or copying files.
func (c *StaticSitesClient) PromoteRelease(ctx context.Context, id string) (*StaticSiteRelease, error) {
	var result StaticSiteRelease
	if err := c.services.client.post(ctx, c.base()+"/releases/"+url.PathEscape(id)+"/promote", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *StaticSitesClient) DeleteRelease(ctx context.Context, id string) error {
	return c.services.client.delete(ctx, c.base()+"/releases/"+url.PathEscape(id), nil)
}
