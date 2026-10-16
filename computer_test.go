package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestComputerClientRequests(t *testing.T) {
	type call struct {
		method, path string
		body         map[string]any
	}
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if data, _ := io.ReadAll(r.Body); len(data) > 0 {
			_ = json.Unmarshal(data, &body)
		}
		calls = append(calls, call{r.Method, r.URL.Path, body})
		switch r.URL.Path {
		case "/v1/workspaces/ws/projects/p/envs/e/svcs/m/computer/screenshot":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNG"))
		case "/v1/workspaces/ws/projects/p/envs/e/svcs/m/computer/bash":
			_, _ = w.Write([]byte(`{"output":"hi\n","exit_code":0}`))
		case "/v1/workspaces/ws/projects/p/envs/e/svcs/m/computer/key":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"type":"FATAL_ERROR","message":"The machine is not ready."}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	computer := NewClient(WithBaseURL(srv.URL), WithToken("t")).Workspace("ws").Project("p").Env("e").Services().Computer("m")

	png, err := computer.Screenshot(ctx)
	if err != nil || string(png) != "PNG" {
		t.Fatalf("screenshot = %q, %v", png, err)
	}
	if err := computer.Click(ctx, 10, 20, MouseRight); err != nil {
		t.Fatal(err)
	}
	if err := computer.DoubleClick(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	res, err := computer.Bash(ctx, "echo hi", 90*time.Second)
	if err != nil || res.Output != "hi\n" {
		t.Fatalf("bash = %+v, %v", res, err)
	}

	var apiErr *APIError
	if err := computer.Key(ctx, "ctrl+l"); !errors.As(err, &apiErr) || apiErr.StatusCode != 503 {
		t.Fatalf("key err = %v, want a 503 APIError", err)
	}

	if calls[1].path != "/v1/workspaces/ws/projects/p/envs/e/svcs/m/computer/click" ||
		calls[1].body["x"] != float64(10) || calls[1].body["button"] != "right" {
		t.Errorf("click call = %+v", calls[1])
	}
	if calls[2].body["double"] != true {
		t.Errorf("double click call = %+v", calls[2])
	}
	if calls[3].body["command"] != "echo hi" || calls[3].body["timeout"] != float64(90) {
		t.Errorf("bash call = %+v", calls[3])
	}
}

func TestVMStartStop(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	svcs := NewClient(WithBaseURL(srv.URL), WithToken("t")).Workspace("ws").Project("p").Env("e").Services()
	if err := svcs.StopVM(ctx, "m"); err != nil {
		t.Fatal(err)
	}
	if err := svcs.StartVM(ctx, "m"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /v1/workspaces/ws/projects/p/envs/e/svcs/m/vm/stop",
		"POST /v1/workspaces/ws/projects/p/envs/e/svcs/m/vm/start",
	}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("calls = %v", paths)
	}
}
