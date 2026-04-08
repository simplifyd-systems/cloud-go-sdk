package cloud

import (
	"context"
	"fmt"
)

// ConfigsClient manages static config file mounts for a service.
// Obtain one via env.Services().Configs(svcSlug).
type ConfigsClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (c *ConfigsClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/configs",
		c.workspace, c.project, c.env, c.svc)
}

// Create adds a new static config file mount to the service.
// The file at MountPath will contain Content when the container runs.
// Use the ${{VAR_NAME}} syntax in Content to interpolate variables at deploy time.
func (c *ConfigsClient) Create(ctx context.Context, in CreateConfigInput) (*ServiceConfig, error) {
	var cfg ServiceConfig
	if err := c.client.post(ctx, c.base(), in, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Update replaces an existing config file mount.
func (c *ConfigsClient) Update(ctx context.Context, configSlug string, in UpdateConfigInput) (*ServiceConfig, error) {
	var cfg ServiceConfig
	if err := c.client.put(ctx, c.base()+"/"+configSlug, in, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Delete removes a config file mount from the service.
func (c *ConfigsClient) Delete(ctx context.Context, configSlug string) error {
	return c.client.delete(ctx, c.base()+"/"+configSlug, nil)
}
