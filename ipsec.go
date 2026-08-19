package cloud

import (
	"context"
	"fmt"
)

// IPsecClient manages the tunnels on a site-to-site VPN gateway service.
// Obtain one via env.Services().IPsec(svcSlug).
type IPsecClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (i *IPsecClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/ipsec/connections",
		i.workspace, i.project, i.env, i.svc)
}

// List returns the gateway's tunnels. They are carried on the service itself,
// so this reads the service rather than a list endpoint of its own.
func (i *IPsecClient) List(ctx context.Context) ([]IPsecConnection, error) {
	svc, err := (&ServicesClient{
		client:    i.client,
		workspace: i.workspace,
		project:   i.project,
		env:       i.env,
	}).Get(ctx, i.svc)
	if err != nil {
		return nil, err
	}
	if svc.IPsecGateway == nil {
		return nil, nil
	}
	return svc.IPsecGateway.Connections, nil
}

// Add creates a tunnel. A connection without a pre-shared key comes up and
// fails authentication, so set one here or with RotatePSK before deploying.
func (i *IPsecClient) Add(ctx context.Context, in IPsecConnectionInput) (*IPsecConnection, error) {
	var conn IPsecConnection
	if err := i.client.post(ctx, i.base(), in, &conn); err != nil {
		return nil, err
	}
	return &conn, nil
}

// Update changes a tunnel's settings. The pre-shared key is not touched: any
// PSK on the input is ignored by the API, and rotating it is RotatePSK's job.
func (i *IPsecClient) Update(ctx context.Context, connSlug string, in IPsecConnectionInput) (*IPsecConnection, error) {
	var conn IPsecConnection
	if err := i.client.put(ctx, i.base()+"/"+connSlug, in, &conn); err != nil {
		return nil, err
	}
	return &conn, nil
}

// RotatePSK replaces a tunnel's pre-shared key. The new key reaches the gateway
// on the next deploy, when the Kubernetes Secret is rewritten and the pod
// restarts — it is not live on return.
func (i *IPsecClient) RotatePSK(ctx context.Context, connSlug, psk string) error {
	body := struct {
		PSK string `json:"psk"`
	}{PSK: psk}
	return i.client.put(ctx, i.base()+"/"+connSlug+"/psk", body, nil)
}

// Delete removes a tunnel by its slug.
func (i *IPsecClient) Delete(ctx context.Context, connSlug string) error {
	return i.client.delete(ctx, i.base()+"/"+connSlug, nil)
}
