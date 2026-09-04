package cloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VideosClient manages a video library: what has been uploaded into it, what
// the platform made of each file, and how it is being watched.
// Obtain one via Services().Video(svcSlug).
type VideosClient struct {
	services *ServicesClient
	svcSlug  string
	// httpClient uploads parts straight to object storage. It is separate from
	// the API client on purpose: a presigned URL carries its own authorisation
	// in the query string, and attaching the caller's API token to it would
	// send that credential to a host that has no business seeing it.
	httpClient *http.Client
}

// Video returns a client for a video library service.
func (s *ServicesClient) Video(svcSlug string) *VideosClient {
	return &VideosClient{
		services: s,
		svcSlug:  svcSlug,
		// Generous, because one request here is a 16 MiB part going out over
		// whatever connection the caller has.
		httpClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *VideosClient) base() string {
	return c.services.svcPath(c.svcSlug) + "/video"
}

// Get returns the library's settings, counts and playback host.
func (c *VideosClient) Get(ctx context.Context) (*VideoLibrary, error) {
	var lib VideoLibrary
	if err := c.services.client.get(ctx, c.base(), &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// Update changes the ladder ceiling and whether originals are retained.
//
// Both take effect on the next encode. Raising the ceiling does not
// retrospectively add rungs to what is already stored, and lowering it does not
// remove them — use Reprocess for that, which bills again.
func (c *VideosClient) Update(ctx context.Context, in UpdateVideoLibraryInput) (*VideoLibrary, error) {
	var lib VideoLibrary
	if err := c.services.client.put(ctx, c.base(), in, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// SetPlaybackDomain attaches a custom playback domain, or detaches the current
// one when the domain is empty.
//
// After attaching, point the domain's CNAME at the returned DomainCNAMETarget.
// Until it resolves, playback stays on the platform hostname — which is on the
// same zero-rated address, so nothing is metered in the meantime.
func (c *VideosClient) SetPlaybackDomain(ctx context.Context, domain string) (*VideoLibrary, error) {
	var lib VideoLibrary
	if err := c.services.client.put(ctx, c.base()+"/domain",
		SetVideoPlaybackDomainInput{PlaybackDomain: domain}, &lib); err != nil {
		return nil, err
	}
	return &lib, nil
}

// List returns the library's videos, most recent first.
func (c *VideosClient) List(ctx context.Context, limit, offset int) (*VideoList, error) {
	path := c.base() + "/videos"
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", fmt.Sprint(limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprint(offset))
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var list VideoList
	if err := c.services.client.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetVideo returns one video with its renditions, captions and job progress.
func (c *VideosClient) GetVideo(ctx context.Context, videoSlug string) (*Video, error) {
	var v Video
	if err := c.services.client.get(ctx, c.videoPath(videoSlug), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *VideosClient) videoPath(videoSlug string) string {
	return c.base() + "/videos/" + videoSlug
}

// Upload puts a file into the library and returns the video it created.
//
// The bytes go straight to object storage using presigned URLs; they never pass
// through the API, which is what makes uploading a multi-gigabyte file from a
// script no different from uploading a small one. The returned video is in
// "queued" — encoding happens afterwards, so use WaitUntilReady to block on it.
//
// Progress, when non-nil, is called after every part with the bytes sent so far
// and the total.
func (c *VideosClient) Upload(ctx context.Context, path, title string, progress func(sent, total int64)) (*Video, error) {
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
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	plan, err := c.Register(ctx, RegisterVideoUploadInput{
		Title:    title,
		Filename: filepath.Base(path),
		Size:     info.Size(),
	})
	if err != nil {
		return nil, err
	}

	parts, err := c.uploadParts(ctx, f, plan, info.Size(), progress)
	if err != nil {
		// The upload is abandoned rather than left in flight: an incomplete
		// multipart upload keeps its parts in the bucket, where they accrue
		// storage the customer is billed for and cannot see.
		_ = c.Delete(context.WithoutCancel(ctx), plan.VideoSlug)
		return nil, err
	}
	return c.Complete(ctx, plan.VideoSlug, parts)
}

// uploadParts sends the file part by part, asking for presigned URLs in batches
// as it goes rather than minting every URL up front — a multi-gigabyte upload
// would otherwise need hundreds of credentials, most of which expire unused.
func (c *VideosClient) uploadParts(
	ctx context.Context,
	r io.ReaderAt,
	plan *VideoUploadPlan,
	size int64,
	progress func(sent, total int64),
) ([]VideoUploadedPart, error) {
	uploaded := make([]VideoUploadedPart, 0, plan.PartCount)
	urls := make(map[int]string, len(plan.Parts))
	for _, p := range plan.Parts {
		urls[p.PartNumber] = p.URL
	}

	var sent int64
	for number := 1; number <= plan.PartCount; number++ {
		if _, ok := urls[number]; !ok {
			batch, err := c.PresignParts(ctx, plan.VideoSlug, number, number+videoPartBatch-1)
			if err != nil {
				return nil, err
			}
			for _, p := range batch {
				urls[p.PartNumber] = p.URL
			}
		}
		signed, ok := urls[number]
		if !ok {
			return nil, fmt.Errorf("no upload URL was issued for part %d", number)
		}

		offset := int64(number-1) * plan.PartSize
		length := plan.PartSize
		if remaining := size - offset; remaining < length {
			length = remaining
		}
		etag, err := c.putPart(ctx, signed, io.NewSectionReader(r, offset, length), length)
		if err != nil {
			return nil, fmt.Errorf("uploading part %d of %d: %w", number, plan.PartCount, err)
		}
		uploaded = append(uploaded, VideoUploadedPart{PartNumber: number, ETag: etag})

		sent += length
		if progress != nil {
			progress(sent, size)
		}
	}
	return uploaded, nil
}

// videoPartBatch matches the API's own batch size, so one presign call covers
// the next fifty parts.
const videoPartBatch = 50

// putPart sends one part and returns the ETag the store assigned it. Every ETag
// is needed, in order, to assemble the object.
func (c *VideosClient) putPart(ctx context.Context, signedURL string, body io.Reader, length int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, body)
	if err != nil {
		return "", err
	}
	req.ContentLength = length

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("storage returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	etag := strings.Trim(resp.Header.Get("ETag"), `"`)
	if etag == "" {
		return "", fmt.Errorf("storage accepted the part but returned no ETag")
	}
	return etag, nil
}

// Register begins an upload and returns the plan for it. Most callers want
// Upload, which does this and the rest of the handshake.
func (c *VideosClient) Register(ctx context.Context, in RegisterVideoUploadInput) (*VideoUploadPlan, error) {
	var plan VideoUploadPlan
	if err := c.services.client.post(ctx, c.base()+"/videos", in, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// PresignParts asks for the next run of part URLs, inclusive at both ends.
func (c *VideosClient) PresignParts(ctx context.Context, videoSlug string, from, to int) ([]VideoUploadPart, error) {
	var out struct {
		Parts []VideoUploadPart `json:"parts"`
	}
	if err := c.services.client.post(ctx, c.videoPath(videoSlug)+"/parts",
		PresignVideoPartsInput{From: from, To: to}, &out); err != nil {
		return nil, err
	}
	return out.Parts, nil
}

// Complete finishes an upload and queues the video for encoding.
func (c *VideosClient) Complete(ctx context.Context, videoSlug string, parts []VideoUploadedPart) (*Video, error) {
	var v Video
	if err := c.services.client.post(ctx, c.videoPath(videoSlug)+"/complete",
		CompleteVideoUploadInput{Parts: parts}, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateVideo changes a video's title and description.
func (c *VideosClient) UpdateVideo(ctx context.Context, videoSlug string, in UpdateVideoInput) (*Video, error) {
	var v Video
	if err := c.services.client.patch(ctx, c.videoPath(videoSlug), in, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// SetPoster chooses the frame shown before playback starts.
//
// The frame is decoded from the retained original, so a library that does not
// keep originals cannot change its posters. The video stays playable throughout
// and the returned poster URL is unchanged — the image behind it is replaced a
// few seconds later.
func (c *VideosClient) SetPoster(ctx context.Context, videoSlug string, offsetMS int64) (*Video, error) {
	var v Video
	if err := c.services.client.put(ctx, c.videoPath(videoSlug)+"/poster",
		SetVideoPosterInput{PosterOffsetMS: offsetMS}, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Reprocess encodes a video again from its retained original, which is what
// picks up a changed ladder ceiling without a re-upload. It is chargeable work
// and bills again.
func (c *VideosClient) Reprocess(ctx context.Context, videoSlug string) (*Video, error) {
	var v Video
	if err := c.services.client.post(ctx, c.videoPath(videoSlug)+"/reprocess", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Delete removes a video and every object derived from it.
func (c *VideosClient) Delete(ctx context.Context, videoSlug string) error {
	return c.services.client.delete(ctx, c.videoPath(videoSlug), nil)
}

// AddCaptions attaches a WebVTT caption track. The file travels inline, so a
// caller with no filesystem can add captions in a single request.
func (c *VideosClient) AddCaptions(ctx context.Context, videoSlug, language, label string, vtt []byte) (*VideoTrack, error) {
	var track VideoTrack
	if err := c.services.client.post(ctx, c.videoPath(videoSlug)+"/tracks", AddVideoTrackInput{
		Language: language,
		Label:    label,
		Content:  base64.StdEncoding.EncodeToString(vtt),
	}, &track); err != nil {
		return nil, err
	}
	return &track, nil
}

// DeleteCaptions removes a caption track and its object.
func (c *VideosClient) DeleteCaptions(ctx context.Context, videoSlug, trackSlug string) error {
	return c.services.client.delete(ctx, c.videoPath(videoSlug)+"/tracks/"+trackSlug, nil)
}

// Analytics returns playback figures for one video over a date range.
//
// Dates are "YYYY-MM-DD" and both are optional; empty means the last four
// weeks ending today.
func (c *VideosClient) Analytics(ctx context.Context, videoSlug, from, to string) (*VideoAnalytics, error) {
	var stats VideoAnalytics
	if err := c.services.client.get(ctx, withRange(c.videoPath(videoSlug)+"/analytics", from, to), &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// LibraryAnalytics returns the same figures across the whole library, plus a
// ranking of what is being watched.
func (c *VideosClient) LibraryAnalytics(ctx context.Context, from, to string) (*VideoAnalytics, error) {
	var stats VideoAnalytics
	if err := c.services.client.get(ctx, withRange(c.base()+"/analytics", from, to), &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func withRange(path, from, to string) string {
	q := url.Values{}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

// WaitUntilReady blocks until a video finishes encoding, polling at interval.
//
// Returns the video once it is ready, or an error carrying the encoder's own
// account of what went wrong — "this file is AV1, which we cannot decode" is
// something a caller can act on, and a generic failure is not.
func (c *VideosClient) WaitUntilReady(ctx context.Context, videoSlug string, interval time.Duration) (*Video, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		v, err := c.GetVideo(ctx, videoSlug)
		if err != nil {
			return nil, err
		}
		switch v.Status {
		case "ready":
			return v, nil
		case "failed":
			return v, fmt.Errorf("encoding failed: %s", v.StatusMessage)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
