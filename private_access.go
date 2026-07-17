package cloud

import (
	"context"
	"fmt"
	"strings"
)

// ListPrivateAccessGrants returns the projects currently allowed to connect to
// a service over its private network hostname.
func (s *ServicesClient) ListPrivateAccessGrants(ctx context.Context, svcSlug string) ([]PrivateAccessGrant, error) {
	svc, err := s.Get(ctx, svcSlug)
	if err != nil {
		return nil, err
	}
	return svc.PrivateAccessGrants, nil
}

// CreatePrivateAccessGrant allows a project in the same workspace to connect
// to one TCP or UDP port. The change applies to a running service immediately.
func (s *ServicesClient) CreatePrivateAccessGrant(ctx context.Context, svcSlug string, in CreatePrivateAccessGrantInput) (*PrivateAccessGrant, error) {
	in.Protocol = strings.ToUpper(in.Protocol)
	if in.ConsumerProject == "" {
		return nil, fmt.Errorf("consumer project is required")
	}
	if in.Protocol != "TCP" && in.Protocol != "UDP" {
		return nil, fmt.Errorf("protocol must be TCP or UDP")
	}
	if in.Port == 0 || in.Port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	if err := s.client.post(ctx, s.svcPath(svcSlug)+"/private-access-grants", in, nil); err != nil {
		return nil, err
	}
	grants, err := s.ListPrivateAccessGrants(ctx, svcSlug)
	if err != nil {
		return nil, err
	}
	for i := range grants {
		if grants[i].ConsumerProjectSlug == in.ConsumerProject && grants[i].Protocol == in.Protocol && grants[i].Port == in.Port {
			return &grants[i], nil
		}
	}
	return nil, fmt.Errorf("private access grant was created but was not returned by the service API")
}

// DeletePrivateAccessGrant revokes a grant and removes its live network policy.
func (s *ServicesClient) DeletePrivateAccessGrant(ctx context.Context, svcSlug, grantSlug string) error {
	if grantSlug == "" {
		return fmt.Errorf("grant slug is required")
	}
	return s.client.delete(ctx, s.svcPath(svcSlug)+"/private-access-grants/"+grantSlug, nil)
}
