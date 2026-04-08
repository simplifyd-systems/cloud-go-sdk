// Package cloud is the official Go SDK for the Simplifyd Cloud API.
// It lets you manage workspaces, projects, environments, and services
// programmatically — ideal for CI/CD pipelines and infrastructure automation.
//
// # Quick start
//
//	client := cloud.NewClient(cloud.WithToken("sk_proj_..."))
//
//	svc, err := client.
//	    Workspace("my-ws").
//	    Project("my-proj").
//	    Env("production").
//	    Services().
//	    Get(ctx, "api-server")
package cloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the Simplifyd Cloud API endpoint.
const DefaultBaseURL = "https://api.cloud.simplifyd.com"

// Client is the Simplifyd Cloud API client.
// Use NewClient to construct one.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithToken sets the authentication token.
// Accepts both JWT bearer tokens (from Login) and project tokens (sk_proj_*).
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient replaces the default http.Client used for all requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithTimeout sets the HTTP request timeout (default: 30s).
// Has no effect if WithHTTPClient was also provided.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// NewClient creates a Simplifyd Cloud client.
//
//	client := cloud.NewClient(
//	    cloud.WithToken(os.Getenv("CLOUD_TOKEN")),
//	    cloud.WithBaseURL("https://api.cloud.simplifyd.com"),
//	)
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ── top-level resource accessors ──────────────────────────────────────────────

// Workspace returns a WorkspaceClient scoped to the given workspace slug.
func (c *Client) Workspace(slug string) *WorkspaceClient {
	return &WorkspaceClient{client: c, slug: slug}
}

// ── auth & top-level operations ───────────────────────────────────────────────

// Login authenticates with email + password and returns a JWT plus the caller's
// active workspace/project/env slugs. The JWT can be passed to WithToken for
// subsequent client calls.
func (c *Client) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	var resp LoginResponse
	err := c.post(ctx, "/v1/auth/accounts/login", loginRequest{Username: email, Password: password}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Me returns the currently authenticated user.
func (c *Client) Me(ctx context.Context) (*User, error) {
	var u User
	if err := c.get(ctx, "/v1/auth/me", &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ListWorkspaces returns all workspaces the authenticated user belongs to.
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var ws []Workspace
	if err := c.get(ctx, "/v1/workspaces", &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// CreateWorkspace creates a new workspace.
func (c *Client) CreateWorkspace(ctx context.Context, name string) (*Workspace, error) {
	var ws Workspace
	if err := c.post(ctx, "/v1/workspaces", createNameRequest{Name: name}, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// ── transport ─────────────────────────────────────────────────────────────────

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func (c *Client) doRequest(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Message != "" {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
		}
	}

	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

func (c *Client) post(ctx context.Context, path string, body, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

func (c *Client) put(ctx context.Context, path string, body, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

func (c *Client) patch(ctx context.Context, path string, body, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

func (c *Client) delete(ctx context.Context, path string, out interface{}) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

// streamLines performs a GET with Accept: text/event-stream and calls fn for
// every "data: ..." line until the stream ends or ctx is cancelled.
func (c *Client) streamLines(ctx context.Context, path string, fn func(string)) error {
	// Use a separate client with no timeout for streaming responses.
	sc := &Client{
		baseURL:    c.baseURL,
		token:      c.token,
		httpClient: &http.Client{},
	}
	req, err := sc.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var apiErr APIError
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Message != "" {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	done := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				fn(strings.TrimPrefix(line, "data: "))
			}
		}
		done <- scanner.Err()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-done:
		return err
	}
}
