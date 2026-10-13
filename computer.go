package cloud

import (
	"context"
	"time"
)

// ComputerClient drives a desktop machine: a VM created from VMImageDesktop.
// It is what an AI agent uses as its computer — it looks at the screen with
// Screenshot and acts with the mouse, keyboard and shell.
//
// Coordinates are pixels on the machine's display, origin top left; Display
// reports its size. Screenshots are taken at that same size, so a point an
// agent picks out of a screenshot is the point it clicks.
//
// Every call returns an *APIError with StatusCode 503 while the machine is
// stopped or still starting; retry after a few seconds.
//
// Obtain one via client.Workspace(ws).Project(p).Env(e).Services().Computer(slug).
type ComputerClient struct {
	client *Client
	path   string
}

// Computer returns a ComputerClient for the desktop machine svcSlug.
func (s *ServicesClient) Computer(svcSlug string) *ComputerClient {
	return &ComputerClient{client: s.client, path: s.svcPath(svcSlug) + "/computer"}
}

// MouseButton is a mouse button for Click and Drag.
type MouseButton string

const (
	MouseLeft   MouseButton = "left"
	MouseMiddle MouseButton = "middle"
	MouseRight  MouseButton = "right"
)

// ScrollDirection is a direction for Scroll.
type ScrollDirection string

const (
	ScrollUp    ScrollDirection = "up"
	ScrollDown  ScrollDirection = "down"
	ScrollLeft  ScrollDirection = "left"
	ScrollRight ScrollDirection = "right"
)

// DisplaySize is the machine's screen size in pixels.
type DisplaySize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// CommandResult is the outcome of Bash or Exec.
type CommandResult struct {
	// Output is stdout and stderr interleaved, as a terminal would show them.
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out"`
	// Truncated is set when the output passed 1 MB; Output then holds the
	// last megabyte, which is where an error usually is.
	Truncated bool `json:"truncated"`
}

// Screenshot returns the screen as a PNG.
func (c *ComputerClient) Screenshot(ctx context.Context) ([]byte, error) {
	return c.client.getRaw(ctx, c.path+"/screenshot")
}

// Display returns the screen size.
func (c *ComputerClient) Display(ctx context.Context) (*DisplaySize, error) {
	var out DisplaySize
	if err := c.client.get(ctx, c.path+"/display", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Click moves the pointer to (x, y) and clicks button.
func (c *ComputerClient) Click(ctx context.Context, x, y int, button MouseButton) error {
	return c.action(ctx, "click", map[string]any{"x": x, "y": y, "button": button})
}

// DoubleClick moves the pointer to (x, y) and double-clicks the left button.
func (c *ComputerClient) DoubleClick(ctx context.Context, x, y int) error {
	return c.action(ctx, "click", map[string]any{"x": x, "y": y, "button": MouseLeft, "double": true})
}

// MouseMove moves the pointer to (x, y) without clicking.
func (c *ComputerClient) MouseMove(ctx context.Context, x, y int) error {
	return c.action(ctx, "mouse-move", map[string]any{"x": x, "y": y})
}

// Drag presses the left button at (fromX, fromY), moves to (toX, toY) and
// releases.
func (c *ComputerClient) Drag(ctx context.Context, fromX, fromY, toX, toY int) error {
	return c.action(ctx, "drag", map[string]any{
		"start_x": fromX, "start_y": fromY, "end_x": toX, "end_y": toY,
	})
}

// Type types text into whatever has keyboard focus.
func (c *ComputerClient) Type(ctx context.Context, text string) error {
	return c.action(ctx, "type", map[string]any{"text": text})
}

// Key presses a key or chord in xdotool syntax: "Return", "ctrl+l",
// "ctrl+shift+t". Several space-separated keys are pressed in turn.
func (c *ComputerClient) Key(ctx context.Context, keys string) error {
	return c.action(ctx, "key", map[string]any{"key": keys})
}

// Scroll scrolls by amount notches (1–50) in direction, with the pointer at
// (x, y).
func (c *ComputerClient) Scroll(ctx context.Context, x, y int, direction ScrollDirection, amount int) error {
	return c.action(ctx, "scroll", map[string]any{"x": x, "y": y, "direction": direction, "amount": amount})
}

// Wait pauses on the machine for up to 30 seconds — for a page to load, say —
// before the caller's next action.
func (c *ComputerClient) Wait(ctx context.Context, d time.Duration) error {
	return c.action(ctx, "wait", map[string]any{"seconds": d.Seconds()})
}

// Bash runs command with bash as the desktop user, in their home directory
// and with access to the display, so it can launch programs onto the screen.
// timeout is capped at 10 minutes; zero means one minute.
//
// A program started in the background (`google-chrome &`) keeps running after
// Bash returns.
func (c *ComputerClient) Bash(ctx context.Context, command string, timeout time.Duration) (*CommandResult, error) {
	return c.command(ctx, "bash", map[string]any{"command": command}, timeout)
}

// Exec runs Python 3 code the same way Bash runs a command.
func (c *ComputerClient) Exec(ctx context.Context, code string, timeout time.Duration) (*CommandResult, error) {
	return c.command(ctx, "exec", map[string]any{"code": code}, timeout)
}

func (c *ComputerClient) command(ctx context.Context, action string, body map[string]any, timeout time.Duration) (*CommandResult, error) {
	if timeout > 0 {
		body["timeout"] = int(timeout.Round(time.Second) / time.Second)
	}
	var out CommandResult
	if err := c.client.post(ctx, c.path+"/"+action, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ComputerClient) action(ctx context.Context, action string, body map[string]any) error {
	return c.client.post(ctx, c.path+"/"+action, body, nil)
}
