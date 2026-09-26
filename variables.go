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

// SetSealed creates a variable, sealed or not. Set creates it sealed: its value
// is given to the service but never returned. Pass sealed false for a value
// people need to read back with Reveal.
func (v *SvcVariablesClient) SetSealed(ctx context.Context, name, value string, sealed bool) (*Variable, error) {
	var variable Variable
	if err := v.client.post(ctx, v.base(), setVariableRequest{Name: name, Value: value, Sealed: &sealed}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// UpdateSealed replaces a variable's value and sets whether it is sealed. It
// is the only way to unseal a variable: a sealed value is never revealed, so
// it can only be made revealable together with a new value.
func (v *SvcVariablesClient) UpdateSealed(ctx context.Context, varSlug, value string, sealed bool) (*Variable, error) {
	var variable Variable
	if err := v.client.put(ctx, v.base()+"/"+varSlug, setVariableRequest{Value: value, Sealed: &sealed}, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// Reveal returns an unsealed variable's value, by slug or name, references
// unresolved. It fails for a sealed variable, for workspace members who cannot
// manage services, and for project tokens: revealing is for people.
func (v *SvcVariablesClient) Reveal(ctx context.Context, variable string) (string, error) {
	var out struct {
		Value string `json:"value"`
	}
	if err := v.client.post(ctx, v.base()+"/"+variable+"/reveal", nil, &out); err != nil {
		return "", err
	}
	return out.Value, nil
}

// Seal seals a variable, by slug or name, so its current value can never be
// revealed. It cannot be undone; a new value can be set unsealed.
func (v *SvcVariablesClient) Seal(ctx context.Context, variable string) error {
	return v.client.post(ctx, v.base()+"/"+variable+"/seal", nil, nil)
}

// Update replaces the value of an existing variable identified by its slug.
// The variable stays sealed or not as it was.
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
