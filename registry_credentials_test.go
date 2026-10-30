package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistryCredentialsLifecycle(t *testing.T) {
	const basePath = "/v1/workspaces/ws/registry-credentials"
	var created CreateRegistryCredentialInput
	var updated map[string]any
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == basePath:
			_ = json.NewEncoder(w).Encode([]WorkspaceRegistryCredential{{Slug: "cred-1", RegistryHost: "ghcr.io", Username: "octocat"}})
		case r.Method == http.MethodPost && r.URL.Path == basePath:
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(WorkspaceRegistryCredential{Slug: "cred-1", RegistryHost: "ghcr.io", Username: created.Username})
		case r.Method == http.MethodPatch && r.URL.Path == basePath+"/cred-1":
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(WorkspaceRegistryCredential{Slug: "cred-1", RegistryHost: "ghcr.io", Username: "octocat"})
		case r.Method == http.MethodDelete && r.URL.Path == basePath+"/cred-1":
			deleted = true
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	credentials := NewClient(WithBaseURL(server.URL)).Workspace("ws").RegistryCredentials()

	list, err := credentials.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].RegistryHost != "ghcr.io" || list[0].Username != "octocat" {
		t.Fatalf("unexpected list: %#v", list)
	}

	credential, err := credentials.Create(ctx, CreateRegistryCredentialInput{RegistryHost: "ghcr.io", Username: "octocat", Password: "ghp_token"})
	if err != nil {
		t.Fatal(err)
	}
	if created != (CreateRegistryCredentialInput{RegistryHost: "ghcr.io", Username: "octocat", Password: "ghp_token"}) {
		t.Fatalf("unexpected create body: %#v", created)
	}
	if credential.Slug != "cred-1" {
		t.Fatalf("unexpected credential: %#v", credential)
	}

	if _, err := credentials.Update(ctx, "cred-1", UpdateRegistryCredentialInput{Password: "ghp_rotated"}); err != nil {
		t.Fatal(err)
	}
	// An unchanged username is left out, so the API keeps it.
	if len(updated) != 1 || updated["password"] != "ghp_rotated" {
		t.Fatalf("unexpected update body: %#v", updated)
	}

	if err := credentials.Delete(ctx, "cred-1"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("delete was not sent")
	}
}

func TestRegistryCredentialsValidateInput(t *testing.T) {
	ctx := context.Background()
	credentials := NewClient().Workspace("ws").RegistryCredentials()
	for _, in := range []CreateRegistryCredentialInput{
		{Username: "u", Password: "p"},
		{RegistryHost: " ", Username: "u", Password: "p"},
		{RegistryHost: "ghcr.io", Password: "p"},
		{RegistryHost: "ghcr.io", Username: "u"},
	} {
		if _, err := credentials.Create(ctx, in); err == nil {
			t.Errorf("expected validation error for %#v", in)
		}
	}
	if _, err := credentials.Update(ctx, "cred-1", UpdateRegistryCredentialInput{}); err == nil {
		t.Error("expected validation error for an empty update")
	}

	services := NewClient().Workspace("ws").Project("p").Env("dev").Services()
	for _, pair := range [][2]string{{"", "p"}, {"u", ""}} {
		if _, err := services.SetRegistryCredentials(ctx, "api", pair[0], pair[1]); err == nil {
			t.Errorf("expected validation error for username %q, password %q", pair[0], pair[1])
		}
	}
}

func TestServiceRegistryCredentials(t *testing.T) {
	const svcPath = "/v1/workspaces/ws/projects/p/envs/dev/svcs/api"
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != svcPath {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(Service{Slug: "api", Docker: &DockerConfig{
			Image:                    "ghcr.io/acme/api",
			RegistryHost:             "ghcr.io",
			RegistryCredentialSource: RegistryCredentialSourceWorkspace,
		}})
	}))
	defer server.Close()

	ctx := context.Background()
	services := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("dev").Services()

	if _, err := services.SetRegistryCredentials(ctx, "api", "octocat", "ghp_token"); err != nil {
		t.Fatal(err)
	}
	svc, err := services.ClearRegistryCredentials(ctx, "api")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Docker.RegistryHost != "ghcr.io" || svc.Docker.RegistryCredentialSource != RegistryCredentialSourceWorkspace {
		t.Fatalf("registry fields not decoded: %#v", svc.Docker)
	}

	if len(bodies) != 2 {
		t.Fatalf("got %d requests, want 2", len(bodies))
	}
	set, clear := bodies[0], bodies[1]
	if set["action"] != "registry_credentials" || set["registry_username"] != "octocat" || set["registry_password"] != "ghp_token" {
		t.Fatalf("unexpected set body: %#v", set)
	}
	// Clearing sends neither field, which the API stores as no credentials.
	if len(clear) != 1 || clear["action"] != "registry_credentials" {
		t.Fatalf("unexpected clear body: %#v", clear)
	}
}
