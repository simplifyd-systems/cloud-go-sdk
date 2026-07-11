package cloud

import (
	"context"
	"fmt"
)

// WorkspaceClient is scoped to a single workspace.
// Obtain one via client.Workspace(slug).
type WorkspaceClient struct {
	client *Client
	slug   string
}

func (w *WorkspaceClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s", w.slug)
}

// Get returns the workspace details.
func (w *WorkspaceClient) Get(ctx context.Context) (*Workspace, error) {
	var ws Workspace
	if err := w.client.get(ctx, w.base(), &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// Update renames the workspace.
func (w *WorkspaceClient) Update(ctx context.Context, name string) (*Workspace, error) {
	var ws Workspace
	if err := w.client.put(ctx, w.base(), createNameRequest{Name: name}, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// MyRole returns the calling user's role in this workspace.
// Role is one of "owner", "developer", or "billing".
func (w *WorkspaceClient) MyRole(ctx context.Context) (string, error) {
	var resp struct {
		Role string `json:"role"`
	}
	if err := w.client.get(ctx, w.base()+"/my-role", &resp); err != nil {
		return "", err
	}
	return resp.Role, nil
}

// GetStats returns resource count summaries (services, deployments, members)
// for the workspace.
func (w *WorkspaceClient) GetStats(ctx context.Context) (*WorkspaceStats, error) {
	var stats WorkspaceStats
	if err := w.client.get(ctx, w.base()+"/usage", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// ── billing ───────────────────────────────────────────────────────────────────

// Usage returns the current-month billing summary for the workspace
// (requires the owner or billing role).
func (w *WorkspaceClient) Usage(ctx context.Context) (*WorkspaceUsage, error) {
	var u WorkspaceUsage
	if err := w.client.get(ctx, w.base()+"/usage", &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Transactions returns the workspace's wallet transactions
// (requires the owner or billing role).
func (w *WorkspaceClient) Transactions(ctx context.Context) ([]Transaction, error) {
	var txns []Transaction
	if err := w.client.get(ctx, w.base()+"/txns", &txns); err != nil {
		return nil, err
	}
	return txns, nil
}

// Fund initiates a wallet top-up. method is "paystack", "stripe", or
// "bank_transfer"; amount is in the smallest currency unit (kobo or cents).
// Returns the provider response: a payment URL for paystack/stripe, or bank
// account details for bank_transfer.
func (w *WorkspaceClient) Fund(ctx context.Context, method string, amount int64) (map[string]interface{}, error) {
	var resp map[string]interface{}
	if err := w.client.post(ctx, w.base()+"/fund", fundWorkspaceRequest{Method: method, Amount: amount}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ── projects ──────────────────────────────────────────────────────────────────

// ListProjects returns all projects in the workspace.
func (w *WorkspaceClient) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	if err := w.client.get(ctx, w.base()+"/projects", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// CreateProject creates a new project in the workspace.
func (w *WorkspaceClient) CreateProject(ctx context.Context, name string) (*Project, error) {
	var p Project
	if err := w.client.post(ctx, w.base()+"/projects", createNameRequest{Name: name}, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Project returns a ProjectClient scoped to the given project slug.
func (w *WorkspaceClient) Project(slug string) *ProjectClient {
	return &ProjectClient{client: w.client, workspace: w.slug, slug: slug}
}

// ── members ───────────────────────────────────────────────────────────────────

// Members returns the MembersClient for managing workspace membership.
func (w *WorkspaceClient) Members() *MembersClient {
	return &MembersClient{client: w.client, workspace: w.slug}
}

// MembersClient manages members of a workspace.
type MembersClient struct {
	client    *Client
	workspace string
}

func (m *MembersClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/members", m.workspace)
}

// List returns all members of the workspace with their roles.
func (m *MembersClient) List(ctx context.Context) ([]WorkspaceMember, error) {
	var members []WorkspaceMember
	if err := m.client.get(ctx, m.base(), &members); err != nil {
		return nil, err
	}
	return members, nil
}

// Add invites users by email to the workspace with the default role (developer).
func (m *MembersClient) Add(ctx context.Context, emails []string) error {
	return m.client.post(ctx, m.base(), addMembersRequest{Emails: emails}, nil)
}

// AddWithRole invites users by email to the workspace with the given role.
// Role must be one of "owner", "developer", or "billing".
func (m *MembersClient) AddWithRole(ctx context.Context, emails []string, role string) error {
	return m.client.post(ctx, m.base(), addMembersRequest{Emails: emails, Role: role}, nil)
}

// Remove removes a member from the workspace.
func (m *MembersClient) Remove(ctx context.Context, memberSlug string) error {
	return m.client.delete(ctx, m.base()+"/"+memberSlug, nil)
}

// UpdateRole changes a member's role.
// Role must be one of "owner", "developer", or "billing".
func (m *MembersClient) UpdateRole(ctx context.Context, memberSlug, role string) error {
	return m.client.patch(ctx, m.base()+"/"+memberSlug+"/role", updateMemberRoleRequest{Role: role}, nil)
}

// ── registry ──────────────────────────────────────────────────────────────────

// Registry returns the RegistryClient for the workspace container registry.
func (w *WorkspaceClient) Registry() *RegistryClient {
	return &RegistryClient{client: w.client, workspace: w.slug}
}

// RegistryClient manages the workspace container registry (Harbor).
type RegistryClient struct {
	client    *Client
	workspace string
}

func (r *RegistryClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/registry", r.workspace)
}

// Get returns the workspace registry details.
func (r *RegistryClient) Get(ctx context.Context) (*Registry, error) {
	var reg Registry
	if err := r.client.get(ctx, r.base(), &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// Credentials returns short-lived push/pull credentials for the registry.
func (r *RegistryClient) Credentials(ctx context.Context) (*RegistryCredentials, error) {
	var creds RegistryCredentials
	if err := r.client.get(ctx, r.base()+"/credentials", &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// ListRepos returns all repositories in the workspace registry.
func (r *RegistryClient) ListRepos(ctx context.Context) ([]RegistryRepo, error) {
	var repos []RegistryRepo
	if err := r.client.get(ctx, r.base()+"/repos", &repos); err != nil {
		return nil, err
	}
	return repos, nil
}
