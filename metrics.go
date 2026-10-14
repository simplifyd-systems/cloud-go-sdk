package cloud

import (
	"context"
	"net/url"
	"time"
)

// MetricsRange is how far back a metrics query reaches.
type MetricsRange string

const (
	MetricsRange1h  MetricsRange = "1h"
	MetricsRange6h  MetricsRange = "6h"
	MetricsRange1d  MetricsRange = "1d"
	MetricsRange7d  MetricsRange = "7d"
	MetricsRange30d MetricsRange = "30d"
)

// MetricPoint is one resource sample for a service. The platform takes one a
// minute while the service is running.
//
// CPU and memory are summed across every instance the service was running when
// the sample was taken, so read them against the per-instance allocation times
// PodCount, not against the allocation alone. Disk is the service's volume and
// is zero for a service that mounts none.
type MetricPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	CPUMillicores   int       `json:"cpu_millicores"`
	MemoryMiB       int       `json:"memory_mib"`
	PodCount        int       `json:"pod_count"`
	DiskUsedMiB     int       `json:"disk_used_mib"`
	DiskCapacityMiB int       `json:"disk_capacity_mib"`
}

// ServiceMetrics is a service's resource samples over a range, with the
// allocation each instance is given.
type ServiceMetrics struct {
	// Range is the range the API answered for, which is 1h when the requested
	// one was not recognised.
	Range  MetricsRange  `json:"range"`
	Points []MetricPoint `json:"points"`
	// VCPUs and MemoryMiB are what each instance is allocated, not the total.
	VCPUs     int `json:"vcpus"`
	MemoryMiB int `json:"memory_mib"`
	// HasVolume is whether any sample in the range saw a mounted volume.
	HasVolume bool `json:"has_volume"`
}

// Metrics returns CPU, memory and disk samples for a service over a range,
// oldest first. An empty rng means the last hour.
func (s *ServicesClient) Metrics(ctx context.Context, svcSlug string, rng MetricsRange) (*ServiceMetrics, error) {
	path := s.svcPath(svcSlug) + "/metrics"
	if rng != "" {
		path += "?range=" + url.QueryEscape(string(rng))
	}
	var m ServiceMetrics
	if err := s.client.get(ctx, path, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
