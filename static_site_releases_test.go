package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStaticReleaseAPIAndMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing API auth")
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases") && r.Method == http.MethodGet:
			w.Write([]byte(`[{"slug":"r1","status":"ready","active":false}]`))
		case strings.HasSuffix(r.URL.Path, "/releases") && r.Method == http.MethodPost:
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if string(payload["preview"]) != "true" || string(payload["branch"]) != `"feature"` || len(payload["files"]) == 0 {
				t.Errorf("release fields were nested or lost: %v", payload)
			}
			w.WriteHeader(201)
			w.Write([]byte(`{"url":"https://preview.example.com","release":{"slug":"r1","status":"ready"}}`))
		case strings.HasSuffix(r.URL.Path, "/r1/promote") && r.Method == http.MethodPost:
			w.Write([]byte(`{"slug":"r1","status":"ready","active":true}`))
		case strings.HasSuffix(r.URL.Path, "/r1") && r.Method == http.MethodDelete:
			w.Write([]byte(`{"success":true}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	client := NewClient(WithBaseURL(server.URL), WithToken("token")).Workspace("w").Project("p").Env("e").Services().StaticSite("s")
	ctx := context.Background()
	result, err := client.PublishRelease(ctx, PublishStaticSiteReleaseInput{PublishStaticSiteInput: PublishStaticSiteInput{Files: []StaticSiteFile{{Path: "index.html", Content: "app"}}}, StaticSiteReleaseOptions: StaticSiteReleaseOptions{Preview: true, Branch: "feature"}})
	if err != nil || result.Release == nil || result.Release.Slug != "r1" {
		t.Fatalf("publish: %+v %v", result, err)
	}
	rows, err := client.Releases(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("history: %+v %v", rows, err)
	}
	r, err := client.PromoteRelease(ctx, "r1")
	if err != nil || !r.Active {
		t.Fatalf("promotion: %+v %v", r, err)
	}
	if err := client.DeleteRelease(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
}
