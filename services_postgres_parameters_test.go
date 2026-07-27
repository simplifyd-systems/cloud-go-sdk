package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUpdatePostgresParameters(t *testing.T) {
	const path = "/v1/workspaces/ws/projects/project/envs/prod/svcs/db/postgres/parameters"
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != path {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var input UpdatePostgresParametersInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.Parameters["statement_timeout"] != "30s" {
			t.Fatalf("unexpected input: %#v", input)
		}
		body := `{"parameters":{"statement_timeout":"30s"},"supported":["statement_timeout"]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	client := NewClient(WithBaseURL("https://api.example.test"), WithHTTPClient(&http.Client{Transport: transport}))
	got, err := client.Workspace("ws").Project("project").Env("prod").Services().UpdatePostgresParameters(
		context.Background(),
		"db",
		UpdatePostgresParametersInput{Parameters: map[string]string{"statement_timeout": "30s"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Parameters["statement_timeout"] != "30s" {
		t.Fatalf("unexpected response: %#v", got)
	}
}
