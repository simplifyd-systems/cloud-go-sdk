package cloud

import (
	"context"
	"fmt"
)

// VolumesClient manages persistent volumes for a service.
// Obtain one via env.Services().Volumes(svcSlug).
//
// A volume outlives the pod: it survives a restart, a redeploy and a node
// failure, which is what separates it from an ephemeral storage attachment.
// Two things follow from that and are worth knowing before attaching one,
// because neither is a setting the caller can turn off:
//
//   - the service is pinned to a single replica. The volume is ReadWriteOnce,
//     so exactly one pod can attach it; a second would stay Pending for ever.
//   - the service is rolled out by stopping the old pod before starting the
//     new one, so a deploy costs a few seconds of downtime. The alternative
//     would be a rollout that waits for a volume it will never get.
type VolumesClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (c *VolumesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/volumes",
		c.workspace, c.project, c.env, c.svc)
}

// Create attaches a new persistent volume to the service. It takes effect on
// the next deploy.
func (c *VolumesClient) Create(ctx context.Context, in CreateVolumeInput) (*ServiceVolume, error) {
	var v ServiceVolume
	if err := c.client.post(ctx, c.base(), in, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Update changes an attached volume.
//
// The size may be raised and applies on the next deploy. It may not be
// lowered: Kubernetes cannot shrink a volume, so a smaller value is recorded
// and then ignored rather than destroying data to honour it.
func (c *VolumesClient) Update(ctx context.Context, volumeSlug string, in UpdateVolumeInput) (*ServiceVolume, error) {
	var v ServiceVolume
	if err := c.client.patch(ctx, c.base()+"/"+volumeSlug, in, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Delete detaches a volume from the service.
//
// It does not destroy the data. The underlying claim is removed with the
// service, so a volume detached by mistake can be reattached at the same mount
// path and will still hold what was there.
func (c *VolumesClient) Delete(ctx context.Context, volumeSlug string) error {
	return c.client.delete(ctx, c.base()+"/"+volumeSlug, nil)
}
