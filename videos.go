package cloud

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// UploadOptions tunes how a file is sent. The zero value is the sensible
// default: four parts in flight, five attempts each, no progress reporting.
type UploadOptions struct {
	// Title defaults to the file name with its extension removed.
	Title string

	// Concurrency is how many parts are in flight at once. Four is a
	// deliberate default rather than a large number: on a home or mobile
	// connection the link is the bottleneck, and past a handful of streams
	// more concurrency only adds retransmissions.
	Concurrency int

	// Retries is how many times a single part is attempted before the upload
	// gives up. A transient 500 from the store must not cost the whole file.
	Retries int

	// Progress, when non-nil, is called with the bytes stored so far and the
	// total. It is called from several goroutines, serialised by the client,
	// and may be called out of order in the sense that the figure only ever
	// rises.
	Progress func(sent, total int64)

	// Resume names a video whose upload was interrupted. The client asks the
	// store which parts already arrived and sends only the rest. The file
	// passed must be the same one — its size is checked against what the
	// upload was registered with.
	Resume string

	// AbandonOnFailure discards the video and the parts already stored when
	// the upload fails, rather than leaving them to be resumed.
	//
	// Off by default, which is the opposite of what a small upload wants and
	// exactly what a large one needs: throwing away 2.8 GB of successfully
	// stored parts because the last one timed out is the behaviour that makes
	// moving a large library impossible. The abandoned video is visible in the
	// library as "uploading" and can be removed like any other.
	AbandonOnFailure bool
}

func (o UploadOptions) concurrency() int {
	if o.Concurrency <= 0 {
		return 4
	}
	return o.Concurrency
}

func (o UploadOptions) retries() int {
	if o.Retries <= 0 {
		return 5
	}
	return o.Retries
}

// UploadError reports an upload that did not finish, and carries the video slug
// needed to resume it. Callers should surface that slug: it is the difference
// between retrying the remaining parts and sending the whole file again.
type UploadError struct {
	VideoSlug string
	// StoredParts is how many parts the store accepted before the failure.
	StoredParts int
	PartCount   int
	Abandoned   bool
	Err         error
}

func (e *UploadError) Error() string {
	if e.Abandoned {
		return fmt.Sprintf("upload failed after %d of %d parts and was discarded: %v",
			e.StoredParts, e.PartCount, e.Err)
	}
	return fmt.Sprintf("upload failed after %d of %d parts: %v (resume it with video %s)",
		e.StoredParts, e.PartCount, e.Err, e.VideoSlug)
}

func (e *UploadError) Unwrap() error { return e.Err }

// Upload puts a file into the library and returns the video it created, with
// the default options. See UploadFile for the rest.
func (c *VideosClient) Upload(ctx context.Context, path, title string, progress func(sent, total int64)) (*Video, error) {
	return c.UploadFile(ctx, path, UploadOptions{Title: title, Progress: progress})
}

// UploadFile puts a file into the library and returns the video it created.
//
// The bytes go straight to object storage using presigned URLs; they never pass
// through the API, which is what makes uploading a multi-gigabyte file from a
// script no different from uploading a small one. Parts are sent concurrently
// and each is retried on its own, so a single dropped connection costs one part
// rather than the file. The returned video is in "queued" — encoding happens
// afterwards, so use WaitUntilReady to block on it.
//
// On failure the parts already stored are kept and the error is an *UploadError
// carrying the video slug, so the same file can be finished by calling again
// with Resume set to that slug.
func (c *VideosClient) UploadFile(ctx context.Context, path string, opts UploadOptions) (*Video, error) {
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

	slug, partSize, partCount, done, err := c.beginOrResume(ctx, path, info.Size(), opts)
	if err != nil {
		return nil, err
	}

	stored, err := c.uploadParts(ctx, f, slug, info.Size(), partSize, partCount, done, opts)
	if err != nil {
		abandoned := false
		if opts.AbandonOnFailure {
			// context.WithoutCancel so an upload cancelled by the caller still
			// cleans up: parts left in flight accrue storage the customer is
			// billed for.
			abandoned = c.Delete(context.WithoutCancel(ctx), slug) == nil
		}
		return nil, &UploadError{
			VideoSlug:   slug,
			StoredParts: len(stored),
			PartCount:   partCount,
			Abandoned:   abandoned,
			Err:         err,
		}
	}
	return c.Complete(ctx, slug, stored)
}

// beginOrResume either registers a new upload or reads back the state of one
// that was interrupted, returning the parts already stored.
func (c *VideosClient) beginOrResume(
	ctx context.Context, path string, size int64, opts UploadOptions,
) (slug string, partSize int64, partCount int, done map[int]VideoUploadedPart, err error) {
	if opts.Resume != "" {
		state, err := c.UploadState(ctx, opts.Resume)
		if err != nil {
			return "", 0, 0, nil, fmt.Errorf("reading what has already been uploaded: %w", err)
		}
		// A resumed upload assembles parts the store is holding from the
		// earlier attempt. If this is a different file, the object that comes
		// out is a splice of two — so the size is checked rather than trusted.
		if state.Size != size {
			return "", 0, 0, nil, fmt.Errorf(
				"video %s was registered for a %d-byte file and %s is %d bytes; "+
					"resume the upload with the same file, or upload this one fresh",
				opts.Resume, state.Size, filepath.Base(path), size)
		}
		done = make(map[int]VideoUploadedPart, len(state.Uploaded))
		for _, p := range state.Uploaded {
			// A part the store recorded short is one whose PUT was cut off
			// mid-body; sending it again is cheaper than discovering at
			// completion that the assembled file is corrupt.
			if expected := partLength(p.PartNumber, state.PartSize, state.Size); p.Size != expected {
				continue
			}
			done[p.PartNumber] = VideoUploadedPart{PartNumber: p.PartNumber, ETag: p.ETag}
		}
		return state.VideoSlug, state.PartSize, state.PartCount, done, nil
	}

	title := opts.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	plan, err := c.Register(ctx, RegisterVideoUploadInput{
		Title:    title,
		Filename: filepath.Base(path),
		Size:     size,
	})
	if err != nil {
		return "", 0, 0, nil, err
	}
	return plan.VideoSlug, plan.PartSize, plan.PartCount, map[int]VideoUploadedPart{}, nil
}

// partLength is how many bytes part number n holds, the last one being short.
func partLength(number int, partSize, size int64) int64 {
	offset := int64(number-1) * partSize
	if remaining := size - offset; remaining < partSize {
		return remaining
	}
	return partSize
}

// uploadParts sends every part that is not already stored, several at a time,
// and returns all of them in ascending order ready for completion.
//
// Presigned URLs are fetched in batches as the upload advances rather than
// minted up front: a multi-gigabyte upload would otherwise need hundreds of
// credentials, most of which expire unused.
func (c *VideosClient) uploadParts(
	ctx context.Context,
	r io.ReaderAt,
	slug string,
	size, partSize int64,
	partCount int,
	done map[int]VideoUploadedPart,
	opts UploadOptions,
) ([]VideoUploadedPart, error) {
	stored := make([]VideoUploadedPart, partCount)
	var sent int64
	for _, p := range done {
		stored[p.PartNumber-1] = p
		sent += partLength(p.PartNumber, partSize, size)
	}

	urls := &partURLs{client: c, slug: slug, urls: map[int]string{}}
	var mu sync.Mutex // serialises the progress callback and the sent total
	report := func(n int64) {
		mu.Lock()
		defer mu.Unlock()
		sent += n
		if opts.Progress != nil {
			opts.Progress(sent, size)
		}
	}
	if opts.Progress != nil && sent > 0 {
		opts.Progress(sent, size)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numbers := make(chan int)
	go func() {
		defer close(numbers)
		for n := 1; n <= partCount; n++ {
			if _, ok := done[n]; ok {
				continue
			}
			select {
			case numbers <- n:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	fail := func(err error) {
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr == nil {
			firstErr = err
			// Stop the other workers: once one part has exhausted its retries
			// the upload cannot complete, and the remaining parts would only
			// spend the caller's bandwidth failing too.
			cancel()
		}
	}

	for i := 0; i < opts.concurrency(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for number := range numbers {
				length := partLength(number, partSize, size)
				etag, err := c.putPartWithRetry(ctx, urls, number,
					io.NewSectionReader(r, int64(number-1)*partSize, length), length, opts.retries())
				if err != nil {
					if ctx.Err() == nil {
						fail(fmt.Errorf("uploading part %d of %d: %w", number, partCount, err))
					}
					return
				}
				stored[number-1] = VideoUploadedPart{PartNumber: number, ETag: etag}
				report(length)
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return compactParts(stored), firstErr
	}
	if err := ctx.Err(); err != nil {
		return compactParts(stored), err
	}
	for n, p := range stored {
		if p.ETag == "" {
			return compactParts(stored), fmt.Errorf("part %d was never stored", n+1)
		}
	}
	return stored, nil
}

// compactParts drops the gaps left by parts that were never sent, so a partial
// result can still be counted and reported.
func compactParts(parts []VideoUploadedPart) []VideoUploadedPart {
	out := parts[:0:0]
	for _, p := range parts {
		if p.ETag != "" {
			out = append(out, p)
		}
	}
	return out
}

// putPartWithRetry sends one part, retrying on anything transient.
//
// A presigned URL is valid for a couple of hours, which a slow connection can
// outlast on a large file, so an authorisation failure re-signs that part
// rather than failing the upload — the one error here that is fixed by asking
// the API again rather than by waiting.
func (c *VideosClient) putPartWithRetry(
	ctx context.Context, urls *partURLs, number int, body *io.SectionReader, length int64, attempts int,
) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		signed, err := urls.get(ctx, number, attempt > 1 && isAuthError(lastErr))
		if err != nil {
			return "", err
		}
		if _, err := body.Seek(0, io.SeekStart); err != nil {
			return "", err
		}

		etag, err := c.putPart(ctx, signed, body, length)
		if err == nil {
			return etag, nil
		}
		lastErr = err
		if ctx.Err() != nil || !isRetryable(err) {
			return "", err
		}
		// Exponential backoff, capped: the store being briefly unavailable is
		// worth waiting out, and hammering it is not.
		delay := time.Duration(1<<uint(attempt-1)) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}
	}
	return "", fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}

// partURLs hands out presigned part URLs, fetching them a batch at a time and
// re-signing individual parts whose URL expired mid-upload.
type partURLs struct {
	client *VideosClient
	slug   string
	mu     sync.Mutex
	urls   map[int]string
}

func (p *partURLs) get(ctx context.Context, number int, refresh bool) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !refresh {
		if u, ok := p.urls[number]; ok {
			return u, nil
		}
	}
	// A batch starting at this part rather than at the next multiple: with
	// several parts in flight they are not claimed in order, and a batch
	// anchored to the part actually being sent always covers it.
	batch, err := p.client.PresignParts(ctx, p.slug, number, number+videoPartBatch-1)
	if err != nil {
		return "", err
	}
	for _, part := range batch {
		p.urls[part.PartNumber] = part.URL
	}
	u, ok := p.urls[number]
	if !ok {
		return "", fmt.Errorf("no upload URL was issued for part %d", number)
	}
	return u, nil
}

// storeError is an HTTP status the object store returned for a part.
type storeError struct {
	status int
	body   string
}

func (e *storeError) Error() string {
	return fmt.Sprintf("storage returned %d: %s", e.status, e.body)
}

// isRetryable reports whether sending the part again could succeed. A network
// error always could; a status only if the store said it was busy, broken or
// no longer willing to accept the signature.
func isRetryable(err error) bool {
	var se *storeError
	if !errors.As(err, &se) {
		return true
	}
	return se.status == http.StatusRequestTimeout ||
		se.status == http.StatusTooManyRequests ||
		se.status >= http.StatusInternalServerError ||
		isAuthError(err)
}

// isAuthError reports a rejected signature, which usually means the presigned
// URL expired rather than that the caller lacks access.
func isAuthError(err error) bool {
	var se *storeError
	if !errors.As(err, &se) {
		return false
	}
	return se.status == http.StatusForbidden || se.status == http.StatusUnauthorized
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
		return "", &storeError{status: resp.StatusCode, body: strings.TrimSpace(string(msg))}
	}

	etag := strings.Trim(resp.Header.Get("ETag"), `"`)
	if etag == "" {
		return "", fmt.Errorf("storage accepted the part but returned no ETag")
	}
	return etag, nil
}

// UploadState reports which parts of an interrupted upload the store already
// holds, and the shape the upload was registered with.
func (c *VideosClient) UploadState(ctx context.Context, videoSlug string) (*VideoUploadState, error) {
	var state VideoUploadState
	if err := c.services.client.get(ctx, c.videoPath(videoSlug)+"/parts", &state); err != nil {
		return nil, err
	}
	return &state, nil
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
