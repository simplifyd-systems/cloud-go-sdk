package cloud

import (
	"context"
	"fmt"
)

// ProjectClient is scoped to a single project.
// Obtain one via client.Workspace(ws).Project(slug).
type ProjectClient struct {
	client    *Client
	workspace string
	slug      string
}

func (p *ProjectClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s", p.workspace, p.slug)
}

// Get returns the project details.
func (p *ProjectClient) Get(ctx context.Context) (*Project, error) {
	var proj Project
	if err := p.client.get(ctx, p.base(), &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

// Update renames the project.
func (p *ProjectClient) Update(ctx context.Context, name string) (*Project, error) {
	var proj Project
	if err := p.client.put(ctx, p.base(), createNameRequest{Name: name}, &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

// ── environments ──────────────────────────────────────────────────────────────

// ListEnvs returns all environments in the project.
func (p *ProjectClient) ListEnvs(ctx context.Context) ([]Env, error) {
	var envs []Env
	if err := p.client.get(ctx, p.base()+"/envs", &envs); err != nil {
		return nil, err
	}
	return envs, nil
}

// CreateEnv creates a new environment in the project.
func (p *ProjectClient) CreateEnv(ctx context.Context, name string) (*Env, error) {
	var env Env
	if err := p.client.post(ctx, p.base()+"/envs", createNameRequest{Name: name}, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Env returns an EnvClient scoped to the given environment slug.
func (p *ProjectClient) Env(slug string) *EnvClient {
	return &EnvClient{
		client:    p.client,
		workspace: p.workspace,
		project:   p.slug,
		slug:      slug,
	}
}

// ── tokens ────────────────────────────────────────────────────────────────────

// Tokens returns the TokensClient for this project.
// Tokens are scoped to a project and tied to a specific environment.
func (p *ProjectClient) Tokens() *TokensClient {
	return &TokensClient{client: p.client, workspace: p.workspace, project: p.slug}
}

// TokensClient manages API tokens for a project.
type TokensClient struct {
	client    *Client
	workspace string
	project   string
}

func (t *TokensClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/tokens", t.workspace, t.project)
}

// List returns all tokens for the project.
func (t *TokensClient) List(ctx context.Context) ([]Token, error) {
	var tokens []Token
	if err := t.client.get(ctx, t.base(), &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// Create creates a new project token scoped to the given environment slug.
// The full token key is only returned on creation — store it securely.
func (t *TokensClient) Create(ctx context.Context, name, envSlug string) (*Token, error) {
	var token Token
	if err := t.client.post(ctx, t.base(), createTokenRequest{Name: name, Env: envSlug}, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Delete revokes a token.
func (t *TokensClient) Delete(ctx context.Context, tokenSlug string) error {
	return t.client.delete(ctx, t.base()+"/"+tokenSlug, nil)
}
