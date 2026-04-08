package cloud

import (
	"context"
	"fmt"
)

// SvcVariablesClient manages environment variables on a specific service.
// Obtain one via env.Services().Variables(svcSlug).
type SvcVariablesClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (v *SvcVariablesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/variables",
		v.workspace, v.project, v.env, v.svc)
}

// List returns all environment variables set on the service.
func (v *SvcVariablesClient) List(ctx context.Context) ([]Variable, error) {
	var vars []Variable
	if err := v.client.get(ctx, v.base(), &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

// Set creates or updates a single variable on the service.
func (v *SvcVariablesClient) Set(ctx context.Context, name, value string) (*Variable, error) {
	var variable Variable
	if err := v.client.post(ctx, v.base(), setVariableRequest{Name: name, Value: value}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// Update replaces the value of an existing variable identified by its slug.
func (v *SvcVariablesClient) Update(ctx context.Context, varSlug, value string) (*Variable, error) {
	var variable Variable
	if err := v.client.put(ctx, v.base()+"/"+varSlug, setVariableRequest{Value: value}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// BulkSet replaces all service variables in one request.
// The provided map (name → value) becomes the complete set of variables.
func (v *SvcVariablesClient) BulkSet(ctx context.Context, vars map[string]string) error {
	items := make([]setVariableRequest, 0, len(vars))
	for k, val := range vars {
		items = append(items, setVariableRequest{Name: k, Value: val})
	}
	return v.client.put(ctx, v.base(), bulkSetVariablesRequest{Variables: items}, nil)
}

// Delete removes a variable by its slug.
func (v *SvcVariablesClient) Delete(ctx context.Context, varSlug string) error {
	return v.client.delete(ctx, v.base()+"/"+varSlug, nil)
}

// AddShared copies a shared environment-level variable (identified by slug)
// into this service's variable set.
func (v *SvcVariablesClient) AddShared(ctx context.Context, envVarSlug string) error {
	path := fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/shared-variables/%s",
		v.workspace, v.project, v.env, v.svc, envVarSlug)
	return v.client.post(ctx, path, nil, nil)
}
