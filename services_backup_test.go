package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreatePostgresBackup(t *testing.T) {
	const path = "/v1/workspaces/ws/projects/project/envs/prod/svcs/db/postgres/backups"
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != path {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, err := json.Marshal(BackupRun{
			Name:   "prod-db-pg-manual-abc12",
			Method: "plugin",
			Phase:  "pending",
		})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Request:    r,
		}, nil
	})

	client := NewClient(
		WithBaseURL("https://api.example.test"),
		WithHTTPClient(&http.Client{Transport: transport}),
	)
	run, err := client.Workspace("ws").Project("project").Env("prod").Services().
		CreatePostgresBackup(context.Background(), "db")
	if err != nil {
		t.Fatal(err)
	}
	if run.Name != "prod-db-pg-manual-abc12" || run.Method != "plugin" || run.Phase != "pending" {
		t.Fatalf("unexpected backup run: %#v", run)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
