package cloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// ── registry credentials ──────────────────────────────────────────────────────

// RegistryCredentials returns the client for the workspace's pull credentials
// for private container registries such as ghcr.io or docker.io.
//
// These are not the workspace's own registry: see Registry for that, and
// Registry().Credentials for its push and pull credentials.
func (w *WorkspaceClient) RegistryCredentials() *RegistryCredentialsClient {
	return &RegistryCredentialsClient{client: w.client, workspace: w.slug}
}

// RegistryCredentialsClient manages the credentials a workspace's Docker image
// services pull private images with. A workspace holds at most one credential
// per registry host, and every Docker image service whose image is on that host
// pulls with it, unless the service has registry credentials of its own (see
// ServicesClient.SetRegistryCredentials). A change takes effect on the
// services' next image pull, without a redeploy, except for a service last
// deployed before 30 October 2026, which picks credentials up on its next
// deploy.
//
// Listing is open to every workspace member; creating, updating and deleting
// require the owner role.
type RegistryCredentialsClient struct {
	client    *Client
	workspace string
}

func (r *RegistryCredentialsClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/registry-credentials", r.workspace)
}

// List returns the workspace's registry credentials. Passwords are never
// returned.
func (r *RegistryCredentialsClient) List(ctx context.Context) ([]WorkspaceRegistryCredential, error) {
	var credentials []WorkspaceRegistryCredential
	if err := r.client.get(ctx, r.base(), &credentials); err != nil {
		return nil, err
	}
	return credentials, nil
}

// Create adds credentials for a registry host. The host may be given as a URL
// or with a path after it ("https://ghcr.io/org"); the API reduces it to the
// host, and treats Docker Hub's aliases (docker.io, index.docker.io,
// registry-1.docker.io, ...) as one. A host that already has credentials is
// rejected: update those instead. sdcr.io is rejected too, since its images
// always pull with platform credentials.
func (r *RegistryCredentialsClient) Create(ctx context.Context, in CreateRegistryCredentialInput) (*WorkspaceRegistryCredential, error) {
	if strings.TrimSpace(in.RegistryHost) == "" {
		return nil, fmt.Errorf("registry host is required")
	}
	if in.Username == "" || in.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	var credential WorkspaceRegistryCredential
	if err := r.client.post(ctx, r.base(), in, &credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

// Update changes a credential's username, password, or both. An empty field
// keeps its current value. The registry host cannot be changed: delete the
// credential and create one for the new host.
func (r *RegistryCredentialsClient) Update(ctx context.Context, slug string, in UpdateRegistryCredentialInput) (*WorkspaceRegistryCredential, error) {
	if in.Username == "" && in.Password == "" {
		return nil, fmt.Errorf("a username or password is required")
	}
	var credential WorkspaceRegistryCredential
	if err := r.client.patch(ctx, r.base()+"/"+url.PathEscape(slug), in, &credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

// Delete removes a credential. Services that pulled with it fall back to
// pulling their images without credentials.
func (r *RegistryCredentialsClient) Delete(ctx context.Context, slug string) error {
	return r.client.delete(ctx, r.base()+"/"+url.PathEscape(slug), nil)
}

// ── per-service registry credentials ──────────────────────────────────────────

// SetRegistryCredentials gives a Docker image service registry credentials of
// its own, which it pulls with instead of the workspace's credentials for its
// registry. Both the username and the password are required, since the API
// replaces the pair. It takes effect on the service's next deploy.
func (s *ServicesClient) SetRegistryCredentials(ctx context.Context, svcSlug, username, password string) (*Service, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	return s.Update(ctx, svcSlug, UpdateServiceInput{
		Action:           "registry_credentials",
		RegistryUsername: username,
		RegistryPassword: password,
	})
}

// ClearRegistryCredentials removes a Docker image service's own registry
// credentials, so that it pulls with the workspace's credentials for its
// registry, or without credentials when the workspace has none. It takes
// effect on the service's next deploy.
func (s *ServicesClient) ClearRegistryCredentials(ctx context.Context, svcSlug string) (*Service, error) {
	return s.Update(ctx, svcSlug, UpdateServiceInput{Action: "registry_credentials"})
}
