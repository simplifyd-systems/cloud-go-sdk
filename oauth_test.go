package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The exact body /v1/oauth/grants returns: a {success, data} envelope, unlike
// most of the API. Decoding this correctly is the whole point of the binding.
const grantsResponse = `{
  "success": true,
  "data": [
    {
      "slug": "019fced6-919d-78e8-85db-640f9b044995",
      "client_id": "J7ScwoycRQQWdjjSPmuNia",
      "client_name": "Claude",
      "scopes": ["identity", "workspace", "offline_access"],
      "all_workspaces": true,
      "workspaces": [],
      "created_at": "2026-08-04T18:19:46Z",
      "last_used_at": "2026-08-05T09:02:11Z"
    },
    {
      "slug": "019fced6-0000-78e8-85db-640f9b044996",
      "client_id": "OtherClient",
      "client_name": "Some Editor",
      "scopes": ["identity", "workspace"],
      "all_workspaces": false,
      "workspaces": [{"slug": "ws-a", "name": "Production"}],
      "created_at": "2026-07-01T10:00:00Z"
    }
  ]
}`

func TestGrantsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/oauth/grants" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(grantsResponse))
	}))
	defer server.Close()

	grants, err := NewClient(WithBaseURL(server.URL)).Grants().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 2 {
		t.Fatalf("got %d grants, want 2", len(grants))
	}

	first := grants[0]
	if first.ClientName != "Claude" {
		t.Errorf("client_name = %q", first.ClientName)
	}
	if !first.AllWorkspaces {
		t.Error("all_workspaces should be true")
	}
	if !first.HasScope("offline_access") || first.HasScope("admin") {
		t.Errorf("scopes = %v", first.Scopes)
	}
	if first.LastUsedAt == nil || !first.LastUsedAt.Equal(time.Date(2026, 8, 5, 9, 2, 11, 0, time.UTC)) {
		t.Errorf("last_used_at = %v", first.LastUsedAt)
	}

	// A "selected workspaces" grant must expose exactly what it covers, so a
	// caller can show the user which workspaces are actually shared.
	second := grants[1]
	if second.AllWorkspaces {
		t.Error("second grant should not be all_workspaces")
	}
	if len(second.Workspaces) != 1 || second.Workspaces[0].Name != "Production" {
		t.Errorf("workspaces = %#v", second.Workspaces)
	}
	// Never used: the pointer distinguishes "never" from a zero timestamp.
	if second.LastUsedAt != nil {
		t.Errorf("last_used_at should be nil, got %v", second.LastUsedAt)
	}
}

func TestGrantsListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer server.Close()

	grants, err := NewClient(WithBaseURL(server.URL)).Grants().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 0 {
		t.Fatalf("got %d grants, want 0", len(grants))
	}
}

func TestGrantsRevoke(t *testing.T) {
	var deleted string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = r.URL.Path
			_, _ = w.Write([]byte(`{"success":true,"message":"application disconnected"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	err := NewClient(WithBaseURL(server.URL)).Grants().Revoke(context.Background(), "grant-1")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != "/v1/oauth/grants/grant-1" {
		t.Fatalf("deleted %q", deleted)
	}
}

func TestGrantsRevokeRequiresSlug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server without a slug")
	}))
	defer server.Close()

	if err := NewClient(WithBaseURL(server.URL)).Grants().Revoke(context.Background(), ""); err == nil {
		t.Fatal("expected an error for an empty grant slug")
	}
}

func TestGrantsRevokeSurfacesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false, "message": "connected application not found",
		})
	}))
	defer server.Close()

	err := NewClient(WithBaseURL(server.URL)).Grants().Revoke(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
}
