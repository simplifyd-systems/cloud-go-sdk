package cloud

import (
	"context"
	"fmt"
)

// IngressClient manages ingress ports for a service.
// Obtain one via env.Services().Ingress(svcSlug).
type IngressClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (i *IngressClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/ingress",
		i.workspace, i.project, i.env, i.svc)
}

// Add creates a new ingress port on the service.
// Protocol must be one of "HTTP", "gRPC", or "TCP".
// A vanity FQDN is automatically assigned; optionally provide a CustomFQDN.
func (i *IngressClient) Add(ctx context.Context, in AddIngressInput) (*IngressPort, error) {
	var port IngressPort
	if err := i.client.post(ctx, i.base(), in, &port); err != nil {
		return nil, err
	}
	if port.Slug == "" && port.VanityFQDN == "" {
		service, err := (&ServicesClient{
			client:    i.client,
			workspace: i.workspace,
			project:   i.project,
			env:       i.env,
		}).Get(ctx, i.svc)
		if err != nil {
			return nil, fmt.Errorf("getting service after adding ingress: %w", err)
		}
		for _, ingress := range service.Ingress {
			if ingress.Protocol == in.Protocol && ingress.Port == uint(in.Port) {
				port = ingress
				break
			}
		}
		if port.Slug == "" && port.VanityFQDN == "" {
			for _, ingress := range service.Ingress {
				if ingress.VanityFQDN != "" {
					port = ingress
					break
				}
			}
		}
	}
	return &port, nil
}

// Delete removes an ingress port by its slug.
func (i *IngressClient) Delete(ctx context.Context, ingressSlug string) error {
	return i.client.delete(ctx, i.base()+"/"+ingressSlug, nil)
}

// DeleteFQDN removes a custom FQDN (vanity domain) from a service.
// fqdn is the domain string (e.g. "api.example.com"), not a slug.
func (i *IngressClient) DeleteFQDN(ctx context.Context, fqdn string) error {
	path := fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/ingress/%s",
		i.workspace, i.project, i.env, i.svc, fqdn)
	return i.client.delete(ctx, path, nil)
}
