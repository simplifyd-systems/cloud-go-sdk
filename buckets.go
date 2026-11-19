package cloud

import (
	"context"
	"fmt"
	"net/url"
)

// BucketsClient manages an S3-compatible bucket service's public access and
// custom domains. Obtain one via Services().Bucket(svcSlug).
type BucketsClient struct {
	services *ServicesClient
	svcSlug  string
}

// Bucket returns a client for a bucket service's public access and domains.
func (s *ServicesClient) Bucket(svcSlug string) *BucketsClient {
	return &BucketsClient{services: s, svcSlug: svcSlug}
}

func (c *BucketsClient) base() string {
	return c.services.svcPath(c.svcSlug) + "/s3"
}

// Get returns the bucket's configuration, read from the service.
func (c *BucketsClient) Get(ctx context.Context) (*S3Bucket, error) {
	svc, err := c.services.Get(ctx, c.svcSlug)
	if err != nil {
		return nil, err
	}
	if svc.Type != ServiceTypeS3Bucket || svc.S3Bucket == nil {
		return nil, fmt.Errorf("service %s is not a bucket (type %s)", c.svcSlug, svc.Type)
	}
	return svc.S3Bucket, nil
}

// SetPublicAccess turns anonymous (unauthenticated) reads of the bucket's
// objects on or off. A public bucket serves its objects on PublicURL, and on
// any custom domains it has. A bucket with custom domains cannot be made
// private: remove them first.
func (c *BucketsClient) SetPublicAccess(ctx context.Context, public bool) (*S3Bucket, error) {
	var b S3Bucket
	if err := c.services.client.put(ctx, c.base()+"/public-access", SetS3BucketPublicAccessInput{Public: public}, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// AddDomain serves a public bucket's objects on a domain you own, alongside
// its public URL and any other domains; attaching one it already has checks
// its verification again and changes nothing else. A bucket can have up to
// 20. Only reads are served on the domain: signed requests keep using the
// bucket's endpoint URL.
//
// After attaching, publish the TXT record in the domain's Verification and
// point its CNAME at the returned DomainCNAMETarget — unless its entry in
// CustomDomains has a DNSZone, in which case the records were written for
// you. The domain is not served until it is verified, and its routing takes
// effect when the bucket is next deployed: call ServicesClient.Deploy with
// AutoApproveChangeSets.
func (c *BucketsClient) AddDomain(ctx context.Context, domain string) (*S3Bucket, error) {
	var b S3Bucket
	if err := c.services.client.post(ctx, c.base()+"/domains", AddS3BucketDomainInput{Domain: domain}, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// RemoveDomain stops serving the bucket on one domain, named by the domain
// itself or by its slug. The bucket keeps serving on its public URL and any
// other domains; deploy it to take down the routing for this one.
func (c *BucketsClient) RemoveDomain(ctx context.Context, domainOrSlug string) (*S3Bucket, error) {
	var b S3Bucket
	if err := c.services.client.delete(ctx, c.base()+"/domains/"+url.PathEscape(domainOrSlug), &b); err != nil {
		return nil, err
	}
	return &b, nil
}
