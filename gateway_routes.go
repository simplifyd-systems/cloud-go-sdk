package cloud

import (
	"context"
	"fmt"
)

// GatewayRoutesClient manages the routing table of an HTTP gateway service.
// Obtain one via env.Services().GatewayRoutes(svcSlug).
type GatewayRoutesClient struct {
	client    *Client
	workspace string
	project   string
	env       string
	svc       string
}

func (g *GatewayRoutesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/projects/%s/envs/%s/svcs/%s/gateway-routes",
		g.workspace, g.project, g.env, g.svc)
}

// List returns the gateway's routes. There is no list endpoint of its own — the
// routes live on the service — so this reads the service and returns its table.
func (g *GatewayRoutesClient) List(ctx context.Context) ([]GatewayRoute, error) {
	svc, err := (&ServicesClient{
		client:    g.client,
		workspace: g.workspace,
		project:   g.project,
		env:       g.env,
	}).Get(ctx, g.svc)
	if err != nil {
		return nil, err
	}
	if svc.HTTPGateway == nil {
		return nil, nil
	}
	return svc.HTTPGateway.Routes, nil
}

// Add creates a route on the gateway.
func (g *GatewayRoutesClient) Add(ctx context.Context, in GatewayRouteInput) (*GatewayRoute, error) {
	var route GatewayRoute
	if err := g.client.post(ctx, g.base(), in, &route); err != nil {
		return nil, err
	}
	return &route, nil
}

// Update replaces a route's settings. Every field is sent, so this is a full
// replacement rather than a partial patch.
func (g *GatewayRoutesClient) Update(ctx context.Context, routeSlug string, in GatewayRouteInput) (*GatewayRoute, error) {
	var route GatewayRoute
	if err := g.client.put(ctx, g.base()+"/"+routeSlug, in, &route); err != nil {
		return nil, err
	}
	return &route, nil
}

// Delete removes a route by its slug.
func (g *GatewayRoutesClient) Delete(ctx context.Context, routeSlug string) error {
	return g.client.delete(ctx, g.base()+"/"+routeSlug, nil)
}
