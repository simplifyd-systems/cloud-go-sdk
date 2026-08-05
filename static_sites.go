package cloud

import "context"

// StaticSitesClient manages a static site service's content and domain.
// Obtain one via Services().StaticSite(svcSlug).
type StaticSitesClient struct {
	services *ServicesClient
	svcSlug  string
}

// StaticSite returns a client for a static site service's content and domain.
func (s *ServicesClient) StaticSite(svcSlug string) *StaticSitesClient {
	return &StaticSitesClient{services: s, svcSlug: svcSlug}
}

func (c *StaticSitesClient) base() string {
	return c.services.svcPath(c.svcSlug) + "/site"
}

// Get returns the site's configuration and serving URLs.
func (c *StaticSitesClient) Get(ctx context.Context) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.get(ctx, c.base(), &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Publish uploads the given files to the site. File contents travel inline in
// the request, so no separate upload step or filesystem access is needed.
//
// With Prune left nil (the default), the publish is a full replace: files not
// in the request are removed, so publishing the same set twice converges on
// exactly those files. Set Prune to false to patch individual files instead.
func (c *StaticSitesClient) Publish(ctx context.Context, in PublishStaticSiteInput) (*StaticSitePublishResult, error) {
	var result StaticSitePublishResult
	if err := c.services.client.post(ctx, c.base()+"/publish", in, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetDocuments changes which objects serve as the directory index and the 404
// body.
func (c *StaticSitesClient) SetDocuments(ctx context.Context, in UpdateStaticSiteDocumentsInput) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.put(ctx, c.base()+"/documents", in, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// SetCustomDomain attaches a custom domain to the site, or detaches the
// current one when the domain is empty. After attaching, point the domain's
// CNAME at the returned DomainCNAMETarget and deploy the service so the
// platform starts routing it.
func (c *StaticSitesClient) SetCustomDomain(ctx context.Context, domain string) (*StaticSite, error) {
	var site StaticSite
	if err := c.services.client.put(ctx, c.base()+"/domain", SetStaticSiteDomainInput{CustomDomain: domain}, &site); err != nil {
		return nil, err
	}
	return &site, nil
}
