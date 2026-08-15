package cloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectDeleteHitsScopedPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if err := client.Workspace("ws").Project("storefront").Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if want := "/v1/workspaces/ws/projects/storefront"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
}

func TestEnvDeleteHitsScopedPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if err := client.Workspace("ws").Project("storefront").Env("production").Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if want := "/v1/workspaces/ws/projects/storefront/envs/production"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
}

// The API refuses a project's last environment with a 409 whose body is
// {"success":false,"message":...} rather than the usual error envelope. Callers
// branch on IsConflict, so that shape has to survive decoding.
func TestEnvDeleteLastEnvironmentIsConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"success":false,"message":"ErrLastEnvironment: delete the project instead"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	err := client.Workspace("ws").Project("storefront").Env("production").Delete(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsConflict(err) {
		t.Fatalf("expected a conflict, got %v", err)
	}
	if IsNotFound(err) {
		t.Error("409 must not read as a 404: destroy would silently treat it as gone")
	}
	if err.Error() != "ErrLastEnvironment: delete the project instead" {
		t.Errorf("message not surfaced: %q", err.Error())
	}
}

// A non-owner gets 403. Distinct from 409 because the caller cannot fix it by
// deleting something else first.
func TestProjectDeleteForbiddenForNonOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"only a workspace owner can delete a project"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	err := client.Workspace("ws").Project("storefront").Delete(context.Background())
	if !IsForbidden(err) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if IsConflict(err) {
		t.Error("403 must not read as a conflict")
	}
}
