package cloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServiceMetricsHitsScopedPathWithRange(t *testing.T) {
	var gotPath, gotRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotRange = r.URL.Path, r.URL.Query().Get("range")
		_, _ = w.Write([]byte(`{
			"range": "1d",
			"vcpus": 2,
			"memory_mib": 1024,
			"has_volume": true,
			"points": [{
				"timestamp": "2026-10-14T09:00:00Z",
				"cpu_millicores": 1500,
				"memory_mib": 700,
				"pod_count": 1,
				"disk_used_mib": 4096,
				"disk_capacity_mib": 10240
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	m, err := client.Workspace("ws").Project("shop").Env("production").Services().
		Metrics(context.Background(), "api", MetricsRange1d)
	if err != nil {
		t.Fatal(err)
	}
	if want := "/v1/workspaces/ws/projects/shop/envs/production/svcs/api/metrics"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
	if gotRange != "1d" {
		t.Errorf("range = %q, want 1d", gotRange)
	}
	if m.Range != MetricsRange1d || m.VCPUs != 2 || m.MemoryMiB != 1024 || !m.HasVolume {
		t.Errorf("metrics = %+v", m)
	}
	if len(m.Points) != 1 || m.Points[0].CPUMillicores != 1500 || m.Points[0].DiskCapacityMiB != 10240 {
		t.Errorf("points = %+v", m.Points)
	}
}

func TestServiceMetricsOmitsEmptyRange(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"range":"1h","points":[]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if _, err := client.Workspace("ws").Project("shop").Env("production").Services().
		Metrics(context.Background(), "api", ""); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want none", gotQuery)
	}
}
