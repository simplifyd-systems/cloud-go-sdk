package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ServicesClient manages services within an environment.
// Obtain one via client.Workspace(ws).Project(proj).Env(env).Services().
type ServicesClient struct {
	client    *Client
	workspace string
	project   string
	env       string
}

func (s *ServicesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs", s.workspace, s.project, s.env)
}

func (s *ServicesClient) svcPath(svcSlug string) string {
	return s.base() + "/" + svcSlug
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// List returns all services in the environment.
func (s *ServicesClient) List(ctx context.Context) ([]Service, error) {
	var svcs []Service
	if err := s.client.get(ctx, s.base(), &svcs); err != nil {
		return nil, err
	}
	return svcs, nil
}

// Get returns a service by its slug.
func (s *ServicesClient) Get(ctx context.Context, svcSlug string) (*Service, error) {
	var svc Service
	if err := s.client.get(ctx, s.svcPath(svcSlug), &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

// Create creates a new service in the environment.
func (s *ServicesClient) Create(ctx context.Context, in CreateServiceInput) (*Service, error) {
	var svc Service
	if err := s.client.post(ctx, s.base(), in, &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

// Update patches a service (image, resources, name, registry credentials, etc.).
func (s *ServicesClient) Update(ctx context.Context, svcSlug string, in UpdateServiceInput) (*Service, error) {
	var svc Service
	if err := s.client.patch(ctx, s.svcPath(svcSlug), in, &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

// Delete permanently deletes a service and all its data.
func (s *ServicesClient) Delete(ctx context.Context, svcSlug string) error {
	return s.client.delete(ctx, s.svcPath(svcSlug), nil)
}

// ── deployments ───────────────────────────────────────────────────────────────

// Deploy creates a new deployment (first deploy or deploy after config changes).
//
// Pass DeployOptions{AutoApproveChangeSets: true} to automatically approve any
// pending changesets before deploying. Without it, the call returns an error if
// pending changesets exist.
func (s *ServicesClient) Deploy(ctx context.Context, svcSlug string, opts ...DeployOptions) (*Deployment, error) {
	path := s.svcPath(svcSlug) + "/deployments"
	if len(opts) > 0 && opts[0].AutoApproveChangeSets {
		path += "?auto_approve_change_sets=true"
	}
	var dep Deployment
	if err := s.client.post(ctx, path, nil, &dep); err != nil {
		return nil, err
	}
	if dep.Slug == "" {
		return s.ActiveDeployment(ctx, svcSlug)
	}
	return &dep, nil
}

// Redeploy re-deploys the currently active deployment (no config changes required).
//
// Pass DeployOptions{AutoApproveChangeSets: true} to automatically approve any
// pending changesets before redeploying. Without it, the call returns an error if
// pending changesets exist.
func (s *ServicesClient) Redeploy(ctx context.Context, svcSlug string, opts ...DeployOptions) (*Deployment, error) {
	path := s.svcPath(svcSlug) + "/deployments"
	if len(opts) > 0 && opts[0].AutoApproveChangeSets {
		path += "?auto_approve_change_sets=true"
	}
	var dep Deployment
	if err := s.client.put(ctx, path, nil, &dep); err != nil {
		return nil, err
	}
	if dep.Slug == "" {
		return s.ActiveDeployment(ctx, svcSlug)
	}
	return &dep, nil
}

// Undeploy stops the running service without deleting it.
func (s *ServicesClient) Undeploy(ctx context.Context, svcSlug string) error {
	return s.client.delete(ctx, s.svcPath(svcSlug)+"/deployments", nil)
}

// ListDeployments returns the deployment history for a service.
func (s *ServicesClient) ListDeployments(ctx context.Context, svcSlug string) ([]Deployment, error) {
	var resp listDeploymentsResponse
	if err := s.client.get(ctx, s.svcPath(svcSlug)+"/deployments", &resp); err != nil {
		return nil, err
	}
	deps := make([]Deployment, 0, len(resp.Deployments)+1)
	if resp.Active.Slug != "" {
		deps = append(deps, resp.Active)
	}
	deps = append(deps, resp.Deployments...)
	return deps, nil
}

// ActiveDeployment returns the currently active deployment for a service.
func (s *ServicesClient) ActiveDeployment(ctx context.Context, svcSlug string) (*Deployment, error) {
	var resp listDeploymentsResponse
	if err := s.client.get(ctx, s.svcPath(svcSlug)+"/deployments", &resp); err != nil {
		return nil, err
	}
	if resp.Active.Slug == "" {
		return nil, fmt.Errorf("service %s has no active deployment", svcSlug)
	}
	return &resp.Active, nil
}

// GetDeployment returns a single deployment by its slug.
func (s *ServicesClient) GetDeployment(ctx context.Context, svcSlug, deploySlug string) (*Deployment, error) {
	var dep Deployment
	if err := s.client.get(ctx, s.svcPath(svcSlug)+"/deployments/"+deploySlug, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// GetLogs fetches up to maxLines log lines from a deployment's SSE log stream,
// stopping early when the stream ends or ctx is cancelled. The logs endpoint
// only streams, so callers wanting a bounded snapshot should pass a context
// with a timeout.
func (s *ServicesClient) GetLogs(ctx context.Context, svcSlug, deploySlug string, maxLines int) ([]string, error) {
	if maxLines <= 0 {
		maxLines = 1000
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lines []string
	err := s.StreamLogs(ctx, svcSlug, deploySlug, func(line string) {
		if len(lines) < maxLines {
			lines = append(lines, line)
			if len(lines) == maxLines {
				cancel()
			}
		}
	})
	return lines, err
}

// StreamLogs streams SSE log lines from a deployment, calling lineFunc for each
// line. Blocks until the stream ends or ctx is cancelled.
func (s *ServicesClient) StreamLogs(ctx context.Context, svcSlug, deploySlug string, lineFunc func(string)) error {
	path := s.svcPath(svcSlug) + "/deployments/" + deploySlug + "/logs"
	return s.client.streamLines(ctx, path, lineFunc)
}

// DiscardChangeset discards any pending (un-deployed) changes on the service.
func (s *ServicesClient) DiscardChangeset(ctx context.Context, svcSlug string) error {
	return s.client.delete(ctx, s.svcPath(svcSlug)+"/changeset", nil)
}

// ApproveChangeset approves the service's pending changeset, applying the
// staged changes.
func (s *ServicesClient) ApproveChangeset(ctx context.Context, svcSlug string) error {
	return s.client.post(ctx, s.svcPath(svcSlug)+"/changeset/approve", nil, nil)
}

// ── TCP proxy ─────────────────────────────────────────────────────────────────

// AddTCPProxy exposes a container port externally via a shared-IP TCP proxy.
// Returns the provider response including the assigned public address/port.
// Optional allowedSourceRanges restrict which client IPs/CIDRs may connect
// (bare IPs are treated as /32); none means open to all.
func (s *ServicesClient) AddTCPProxy(ctx context.Context, svcSlug string, port uint, allowedSourceRanges ...string) (map[string]interface{}, error) {
	var resp map[string]interface{}
	body := struct {
		Port                uint     `json:"port"`
		AllowedSourceRanges []string `json:"allowed_source_ranges,omitempty"`
	}{Port: port, AllowedSourceRanges: allowedSourceRanges}
	if err := s.client.post(ctx, s.svcPath(svcSlug)+"/proxy", body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteTCPProxy removes the TCP proxy for the given container port.
func (s *ServicesClient) DeleteTCPProxy(ctx context.Context, svcSlug string, port uint) error {
	return s.client.delete(ctx, fmt.Sprintf("%s/proxy/%d", s.svcPath(svcSlug), port), nil)
}

// ── convenience methods ───────────────────────────────────────────────────────

// DeployImage updates the Docker image (and optional tag) on a service and
// triggers a new deployment. It is equivalent to calling Update followed by
// Deploy.
//
// Returns the new Deployment. If the service already runs that image:tag the
// update still triggers a fresh deployment.
//
// Pass DeployOptions{AutoApproveChangeSets: true} to automatically approve any
// pending changesets before deploying.
func (s *ServicesClient) DeployImage(ctx context.Context, svcSlug, image, tag string, opts ...DeployOptions) (*Deployment, error) {
	if _, err := s.Update(ctx, svcSlug, UpdateServiceInput{
		Action: "image",
		Image:  image,
		Tag:    tag,
	}); err != nil {
		return nil, fmt.Errorf("updating image: %w", err)
	}

	dep, err := s.Deploy(ctx, svcSlug, opts...)
	if err != nil {
		// Already have a running deployment — try a redeploy instead.
		dep, err = s.Redeploy(ctx, svcSlug, opts...)
		if err != nil {
			return nil, fmt.Errorf("deploying: %w", err)
		}
	}
	return dep, nil
}

// WaitForDeployment polls until the given deployment reaches a terminal status
// (running, failed, stopped, sleeping) or ctx is cancelled.
//
// interval controls how often the status is checked (minimum 2s is enforced).
// Returns the final Deployment state.
func (s *ServicesClient) WaitForDeployment(ctx context.Context, svcSlug, deploySlug string, interval time.Duration) (*Deployment, error) {
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			deps, err := s.ListDeployments(ctx, svcSlug)
			if err != nil {
				return nil, fmt.Errorf("polling deployments: %w", err)
			}
			for i := range deps {
				if deps[i].Slug == deploySlug {
					switch deps[i].Status {
					case DeploymentStatusRunning,
						DeploymentStatusFailed,
						DeploymentStatusStopped,
						DeploymentStatusSleeping:
						return &deps[i], nil
					}
				}
			}
		}
	}
}

// ── sub-resource accessors ────────────────────────────────────────────────────

// Variables returns a SvcVariablesClient for managing environment variables
// on the given service.
func (s *ServicesClient) Variables(svcSlug string) *SvcVariablesClient {
	return &SvcVariablesClient{
		client:    s.client,
		workspace: s.workspace,
		project:   s.project,
		env:       s.env,
		svc:       svcSlug,
	}
}

// Configs returns a ConfigsClient for managing static config file mounts on
// the given service.
func (s *ServicesClient) Configs(svcSlug string) *ConfigsClient {
	return &ConfigsClient{
		client:    s.client,
		workspace: s.workspace,
		project:   s.project,
		env:       s.env,
		svc:       svcSlug,
	}
}

// Ingress returns an IngressClient for managing ingress ports on the given service.
func (s *ServicesClient) Ingress(svcSlug string) *IngressClient {
	return &IngressClient{
		client:    s.client,
		workspace: s.workspace,
		project:   s.project,
		env:       s.env,
		svc:       svcSlug,
	}
}

// ── shell ─────────────────────────────────────────────────────────────────────

// ConnectShell opens an interactive WebSocket shell to a running service
// container. stdin and stdout are connected to the remote PTY. resizeCh
// delivers terminal resize events; close it (or cancel ctx) to end the session.
//
// Blocks until the connection is closed, the context is cancelled, or an error
// occurs.
func (s *ServicesClient) ConnectShell(
	ctx context.Context,
	svcSlug string,
	stdin io.Reader,
	stdout io.Writer,
	resizeCh <-chan TerminalSize,
) error {
	return s.ConnectShellWithOptions(ctx, svcSlug, stdin, stdout, resizeCh, ShellOptions{TTY: readerIsTerminal(stdin)})
}

// readerIsTerminal recognizes the file shape used by terminal stdin without
// adding a platform-specific terminal dependency to the SDK. Pipes and other
// readers deliberately select non-TTY mode.
func readerIsTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// ShellOptions controls how the remote shell is attached.
type ShellOptions struct {
	// TTY allocates a terminal for interactive use. Set it to false when stdin
	// is a pipe so terminal probes cannot consume command input.
	TTY bool
}

// ConnectShellWithOptions opens a WebSocket shell with an explicit terminal
// mode. Unlike ConnectShell, this can run a non-interactive, non-TTY session.
func (s *ServicesClient) ConnectShellWithOptions(
	ctx context.Context,
	svcSlug string,
	stdin io.Reader,
	stdout io.Writer,
	resizeCh <-chan TerminalSize,
	options ShellOptions,
) error {
	wsURL, err := s.shellWSURL(svcSlug, options.TTY)
	if err != nil {
		return fmt.Errorf("building shell URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("connecting to shell: %w", err)
	}
	defer conn.Close()

	errc := make(chan error, 3)
	stdinEOFCapable := make(chan struct{})
	var capabilityOnce sync.Once
	var writeMu sync.Mutex
	writeMessage := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	// server → stdout
	go func() {
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if msgType == websocket.BinaryMessage {
				if _, err := stdout.Write(data); err != nil {
					errc <- err
					return
				}
			} else if msgType == websocket.TextMessage {
				var msg struct {
					Type     string `json:"type"`
					StdinEOF bool   `json:"stdin_eof"`
				}
				if json.Unmarshal(data, &msg) == nil && msg.Type == "shell_ready" && msg.StdinEOF {
					capabilityOnce.Do(func() { close(stdinEOFCapable) })
				}
			}
		}
	}()

	// stdin → server (binary frames). EOF half-closes only remote stdin; the
	// read pump remains active until all remote output has arrived.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				if werr := writeMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					errc <- werr
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					select {
					case <-stdinEOFCapable:
						data, _ := json.Marshal(map[string]string{"type": "stdin_eof"})
						if werr := writeMessage(websocket.TextMessage, data); werr != nil {
							errc <- werr
						}
					case <-time.After(500 * time.Millisecond):
						// Older servers do not support half-close. Preserve the
						// previous EOF behavior instead of leaving the session hung.
						errc <- nil
					}
					return
				}
				errc <- err
				return
			}
		}
	}()

	// resize channel → server (text JSON frames)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errc <- nil
				return
			case sz, ok := <-resizeCh:
				if !ok {
					errc <- nil
					return
				}
				msg := shellResizeMsg{Type: "resize", Cols: sz.Cols, Rows: sz.Rows}
				data, _ := json.Marshal(msg)
				if err := writeMessage(websocket.TextMessage, data); err != nil {
					errc <- err
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		_ = writeMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return nil
	case err := <-errc:
		_ = writeMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return nil
		}
		return err
	}
}

// shellWSURL builds the wss:// (or ws://) URL for the shell endpoint, with the
// auth token passed as a query parameter (browsers and WebSocket clients cannot
// set custom headers during the upgrade).
func (s *ServicesClient) shellWSURL(svcSlug string, tty bool) (string, error) {
	base := s.client.baseURL
	var wsBase string
	switch {
	case strings.HasPrefix(base, "https://"):
		wsBase = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		wsBase = "ws://" + strings.TrimPrefix(base, "http://")
	default:
		wsBase = "wss://" + base
	}

	path := fmt.Sprintf("%s/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/shell",
		wsBase, s.workspace, s.project, s.env, svcSlug)

	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("jwt", s.client.token)
	q.Set("tty", strconv.FormatBool(tty))
	u.RawQuery = q.Encode()
	return u.String(), nil
}
