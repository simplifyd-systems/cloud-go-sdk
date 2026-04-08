package cloud

import (
	"context"
	"fmt"
)

// EnvClient is scoped to a single environment.
// Obtain one via client.Workspace(ws).Project(proj).Env(slug).
type EnvClient struct {
	client    *Client
	workspace string
	project   string
	slug      string
}

func (e *EnvClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s", e.workspace, e.project, e.slug)
}

// Get returns the environment details.
func (e *EnvClient) Get(ctx context.Context) (*Env, error) {
	var env Env
	if err := e.client.get(ctx, e.base(), &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Update renames the environment.
func (e *EnvClient) Update(ctx context.Context, name string) (*Env, error) {
	var env Env
	if err := e.client.put(ctx, e.base(), createNameRequest{Name: name}, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// ── env-level variables ───────────────────────────────────────────────────────

// Variables returns the EnvVariablesClient for environment-level shared variables.
func (e *EnvClient) Variables() *EnvVariablesClient {
	return &EnvVariablesClient{client: e.client, workspace: e.workspace, project: e.project, env: e.slug}
}

// EnvVariablesClient manages shared variables at the environment level.
// These variables are available to all services in the environment.
type EnvVariablesClient struct {
	client    *Client
	workspace string
	project   string
	env       string
}

func (v *EnvVariablesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/variables", v.workspace, v.project, v.env)
}

// List returns all environment-level variables.
func (v *EnvVariablesClient) List(ctx context.Context) ([]Variable, error) {
	var vars []Variable
	if err := v.client.get(ctx, v.base(), &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

// Set creates or updates a variable.
func (v *EnvVariablesClient) Set(ctx context.Context, name, value string) (*Variable, error) {
	var variable Variable
	if err := v.client.post(ctx, v.base(), setVariableRequest{Name: name, Value: value}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// Update replaces the value of an existing variable identified by its slug.
func (v *EnvVariablesClient) Update(ctx context.Context, varSlug, value string) (*Variable, error) {
	var variable Variable
	if err := v.client.put(ctx, v.base()+"/"+varSlug, setVariableRequest{Value: value}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// Delete removes a variable by its slug.
func (v *EnvVariablesClient) Delete(ctx context.Context, varSlug string) error {
	return v.client.delete(ctx, v.base()+"/"+varSlug, nil)
}

// ── services ──────────────────────────────────────────────────────────────────

// Services returns the ServicesClient for managing services in this environment.
func (e *EnvClient) Services() *ServicesClient {
	return &ServicesClient{
		client:    e.client,
		workspace: e.workspace,
		project:   e.project,
		env:       e.slug,
	}
}
