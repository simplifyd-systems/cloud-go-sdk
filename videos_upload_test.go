package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

const uploadBasePath = "/v1/workspaces/ws/projects/p/envs/dev/svcs/lib/video"

// uploadFixture stands in for the API and the object store together, because
// the interesting behaviour of an upload is the conversation between them: a
// part is signed by one and stored by the other, and resuming depends on the
// store's account of what it already holds.
type uploadFixture struct {
	t        *testing.T
	partSize int64
	size     int64

	api   *httptest.Server
	store *httptest.Server

	mu sync.Mutex
	// stored is the ETag per part number the store has accepted.
	stored map[int]string
	// fail is how many more times a given part should be rejected, and with
	// what status, before it is allowed through.
	fail       map[int]int
	failStatus int
	// attempts counts every PUT, so a test can assert a part was not re-sent.
	attempts  map[int]int
	completed []VideoUploadedPart
	deleted   bool
	signed    int
}

func newUploadFixture(t *testing.T, size, partSize int64) *uploadFixture {
	t.Helper()
	f := &uploadFixture{
		t: t, size: size, partSize: partSize,
		stored: map[int]string{}, fail: map[int]int{}, attempts: map[int]int{},
		failStatus: http.StatusInternalServerError,
	}

	f.store = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var number int
		if _, err := fmt.Sscanf(r.URL.Path, "/part/%d", &number); err != nil {
			http.NotFound(w, r)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.attempts[number]++
		if f.fail[number] > 0 {
			f.fail[number]--
			w.WriteHeader(f.failStatus)
			return
		}
		// A stale signature is rejected the way the store would reject it.
		if r.URL.Query().Get("stale") == "1" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		f.stored[number] = fmt.Sprintf("etag-%d", number)
		w.Header().Set("ETag", `"`+f.stored[number]+`"`)
	}))
	t.Cleanup(f.store.Close)

	f.api = httptest.NewServer(http.HandlerFunc(f.serveAPI))
	t.Cleanup(f.api.Close)
	return f
}

func (f *uploadFixture) partCount() int {
	return int((f.size + f.partSize - 1) / f.partSize)
}

func (f *uploadFixture) partLen(number int) int64 {
	if remaining := f.size - int64(number-1)*f.partSize; remaining < f.partSize {
		return remaining
	}
	return f.partSize
}

func (f *uploadFixture) presign(from, to int) []VideoUploadPart {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signed++
	var parts []VideoUploadPart
	for n := from; n <= to && n <= f.partCount(); n++ {
		parts = append(parts, VideoUploadPart{
			PartNumber: n,
			URL:        fmt.Sprintf("%s/part/%d", f.store.URL, n),
		})
	}
	return parts
}

func (f *uploadFixture) serveAPI(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == uploadBasePath+"/videos":
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(VideoUploadPlan{
			VideoSlug: "vid-1", UploadID: "up-1", Key: "originals/vid-1/source.mp4",
			PartSize: f.partSize, PartCount: f.partCount(), Parts: f.presign(1, videoPartBatch),
		})

	case r.Method == http.MethodPost && r.URL.Path == uploadBasePath+"/videos/vid-1/parts":
		var in PresignVideoPartsInput
		_ = json.NewDecoder(r.Body).Decode(&in)
		_ = json.NewEncoder(w).Encode(map[string]any{"parts": f.presign(in.From, in.To)})

	case r.Method == http.MethodGet && r.URL.Path == uploadBasePath+"/videos/vid-1/parts":
		f.mu.Lock()
		uploaded := make([]VideoUploadedPart, 0, len(f.stored))
		for number, etag := range f.stored {
			uploaded = append(uploaded, VideoUploadedPart{
				PartNumber: number, ETag: etag, Size: f.partLen(number),
			})
		}
		f.mu.Unlock()
		sort.Slice(uploaded, func(i, j int) bool { return uploaded[i].PartNumber < uploaded[j].PartNumber })
		_ = json.NewEncoder(w).Encode(VideoUploadState{
			VideoSlug: "vid-1", Key: "originals/vid-1/source.mp4", Size: f.size,
			PartSize: f.partSize, PartCount: f.partCount(), Uploaded: uploaded,
		})

	case r.Method == http.MethodPost && r.URL.Path == uploadBasePath+"/videos/vid-1/complete":
		var in CompleteVideoUploadInput
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.completed = in.Parts
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(Video{Slug: "vid-1", Status: "queued", SourceBytes: f.size})

	case r.Method == http.MethodDelete && r.URL.Path == uploadBasePath+"/videos/vid-1":
		f.mu.Lock()
		f.deleted = true
		f.mu.Unlock()

	default:
		http.NotFound(w, r)
	}
}

func (f *uploadFixture) client() *VideosClient {
	return NewClient(WithBaseURL(f.api.URL)).
		Workspace("ws").Project("p").Env("dev").Services().Video("lib")
}

// sourceFile writes a file of the fixture's size, so section reads and part
// lengths are exercised against real bytes rather than a stub reader.
func (f *uploadFixture) sourceFile() string {
	f.t.Helper()
	path := filepath.Join(f.t.TempDir(), "talk.mp4")
	body := make([]byte, f.size)
	for i := range body {
		body[i] = byte(i)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func TestUploadSendsEveryPartInOrder(t *testing.T) {
	f := newUploadFixture(t, 1000, 128) // 8 parts, the last one short
	path := f.sourceFile()

	var lastSent int64
	video, err := f.client().UploadFile(context.Background(), path, UploadOptions{
		Concurrency: 4,
		Progress:    func(sent, total int64) { lastSent = sent },
	})
	if err != nil {
		t.Fatal(err)
	}
	if video.Slug != "vid-1" {
		t.Fatalf("unexpected video: %#v", video)
	}
	if len(f.completed) != f.partCount() {
		t.Fatalf("completed with %d parts, want %d", len(f.completed), f.partCount())
	}
	// The store assembles parts in the order they are listed, so an upload sent
	// concurrently must still be completed in ascending part order.
	for i, p := range f.completed {
		if p.PartNumber != i+1 {
			t.Fatalf("part %d of the completion list is number %d", i, p.PartNumber)
		}
		if p.ETag != fmt.Sprintf("etag-%d", i+1) {
			t.Fatalf("part %d carries etag %q", p.PartNumber, p.ETag)
		}
	}
	if lastSent != f.size {
		t.Fatalf("progress ended at %d bytes, want %d", lastSent, f.size)
	}
}

func TestUploadRetriesATransientlyFailingPart(t *testing.T) {
	f := newUploadFixture(t, 512, 128) // 4 parts
	f.fail[3] = 2                      // part 3 fails twice, then succeeds
	path := f.sourceFile()

	if _, err := f.client().UploadFile(context.Background(), path, UploadOptions{
		Concurrency: 2, Retries: 4,
	}); err != nil {
		t.Fatal(err)
	}
	if len(f.completed) != 4 {
		t.Fatalf("completed with %d parts, want 4", len(f.completed))
	}
	if f.attempts[3] != 3 {
		t.Fatalf("part 3 was attempted %d times, want 3", f.attempts[3])
	}
	if f.attempts[1] != 1 {
		t.Fatalf("part 1 was attempted %d times; a failure elsewhere must not re-send it", f.attempts[1])
	}
}

func TestUploadKeepsStoredPartsForResume(t *testing.T) {
	f := newUploadFixture(t, 512, 128)
	f.fail[4] = 99 // part 4 never succeeds
	path := f.sourceFile()

	_, err := f.client().UploadFile(context.Background(), path, UploadOptions{
		Concurrency: 1, Retries: 2,
	})
	if err == nil {
		t.Fatal("expected the upload to fail")
	}
	var interrupted *UploadError
	if !errors.As(err, &interrupted) {
		t.Fatalf("error is %T, want *UploadError", err)
	}
	if interrupted.VideoSlug != "vid-1" {
		t.Fatalf("error names video %q", interrupted.VideoSlug)
	}
	if interrupted.StoredParts != 3 || interrupted.PartCount != 4 {
		t.Fatalf("reported %d of %d parts stored, want 3 of 4",
			interrupted.StoredParts, interrupted.PartCount)
	}
	// The point of the change: what arrived is still there to be resumed.
	if f.deleted {
		t.Fatal("the video was deleted, discarding the parts already stored")
	}
	if interrupted.Abandoned {
		t.Fatal("the upload reported itself abandoned")
	}
}

func TestUploadAbandonsOnFailureWhenAsked(t *testing.T) {
	f := newUploadFixture(t, 256, 128)
	f.fail[2] = 99
	path := f.sourceFile()

	_, err := f.client().UploadFile(context.Background(), path, UploadOptions{Retries: 1, AbandonOnFailure: true})
	if err == nil {
		t.Fatal("expected the upload to fail")
	}
	if !f.deleted {
		t.Fatal("the video was kept despite AbandonOnFailure")
	}
}

func TestResumeSendsOnlyTheMissingParts(t *testing.T) {
	f := newUploadFixture(t, 512, 128)
	f.fail[4] = 99
	path := f.sourceFile()
	client := f.client()

	if _, err := client.UploadFile(context.Background(), path, UploadOptions{Concurrency: 1, Retries: 1}); err == nil {
		t.Fatal("expected the first attempt to fail")
	}

	f.mu.Lock()
	f.fail[4] = 0
	before := map[int]int{}
	for n, a := range f.attempts {
		before[n] = a
	}
	f.mu.Unlock()

	if _, err := client.UploadFile(context.Background(), path, UploadOptions{Resume: "vid-1"}); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		if f.attempts[n] != before[n] {
			t.Fatalf("part %d was sent again on resume (%d attempts, was %d)", n, f.attempts[n], before[n])
		}
	}
	if len(f.completed) != 4 {
		t.Fatalf("completed with %d parts, want 4", len(f.completed))
	}
}

func TestResumeRefusesADifferentFile(t *testing.T) {
	f := newUploadFixture(t, 512, 128)
	path := f.sourceFile()
	f.size = 1024 // the upload was registered for a larger file

	_, err := f.client().UploadFile(context.Background(), path, UploadOptions{Resume: "vid-1"})
	if err == nil {
		t.Fatal("expected resuming with a different file to be refused")
	}
	if len(f.completed) != 0 {
		t.Fatal("a mismatched resume completed the upload")
	}
}

func TestUploadResignsAnExpiredPartURL(t *testing.T) {
	f := newUploadFixture(t, 128, 128) // one part
	// The first signature comes back stale, as an expired presigned URL would.
	f.mu.Lock()
	f.signed = 0
	f.mu.Unlock()

	f.failStatus = http.StatusForbidden
	f.fail[1] = 1
	path := f.sourceFile()

	if _, err := f.client().UploadFile(context.Background(), path, UploadOptions{Retries: 3}); err != nil {
		t.Fatal(err)
	}
	// Two signing calls: the plan, then a fresh URL after the rejection.
	if f.signed < 2 {
		t.Fatalf("the part was signed %d times; a rejected signature must be renewed", f.signed)
	}
}
