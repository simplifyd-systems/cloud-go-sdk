package cloud

import (
	"context"
	"fmt"
	"time"
)

// Grants returns a client for the authenticated user's connected applications:
// third-party apps, such as an AI agent connected over MCP, that the user has
// authorized to act on their behalf via OAuth.
//
// These are account-level and not scoped to a workspace.
func (c *Client) Grants() *GrantsClient {
	return &GrantsClient{client: c}
}

// GrantsClient manages connected applications.
type GrantsClient struct {
	client *Client
}

// OAuthGrantWorkspace is a workspace a grant covers.
type OAuthGrantWorkspace struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// OAuthGrant is a standing authorization the user gave to one application.
type OAuthGrant struct {
	// Slug identifies the grant, and is what Revoke takes.
	Slug string `json:"slug"`
	// ClientID identifies the application that holds the grant.
	ClientID   string `json:"client_id"`
	ClientName string `json:"client_name"`
	ClientURI  string `json:"client_uri,omitempty"`
	LogoURI    string `json:"logo_uri,omitempty"`
	// Scopes granted: identity, workspace, offline_access.
	Scopes []string `json:"scopes"`
	// AllWorkspaces reports whether the grant covers every workspace,
	// including ones created after it was given. When false, Workspaces lists
	// exactly what the application can reach.
	AllWorkspaces bool                  `json:"all_workspaces"`
	Workspaces    []OAuthGrantWorkspace `json:"workspaces"`
	CreatedAt     time.Time             `json:"created_at"`
	LastUsedAt    *time.Time            `json:"last_used_at,omitempty"`
}

// HasScope reports whether the grant carries the named scope.
func (g OAuthGrant) HasScope(scope string) bool {
	for _, granted := range g.Scopes {
		if granted == scope {
			return true
		}
	}
	return false
}

// grantsEnvelope unwraps the {success, data} response these endpoints use.
// Most of the API returns bare payloads, so this is local rather than shared.
type grantsEnvelope struct {
	Success bool         `json:"success"`
	Data    []OAuthGrant `json:"data"`
}

// List returns the applications the authenticated user has connected.
//
// Requires a user session token. A project token has no user to speak for, and
// an OAuth access token is refused: an application must not be able to inspect
// or revoke the user's other connections.
func (g *GrantsClient) List(ctx context.Context) ([]OAuthGrant, error) {
	var resp grantsEnvelope
	if err := g.client.get(ctx, "/v1/oauth/grants", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// Revoke disconnects an application by grant slug.
//
// It takes effect immediately rather than when the application's current access
// token would have expired: the grant is re-checked on every request, and its
// refresh tokens are revoked at the same time.
func (g *GrantsClient) Revoke(ctx context.Context, grant string) error {
	if grant == "" {
		return fmt.Errorf("grant slug is required")
	}
	return g.client.delete(ctx, "/v1/oauth/grants/"+grant, nil)
}
