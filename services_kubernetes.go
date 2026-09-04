package cloud

import (
	"context"
	"time"
)

// ExecCredential is the credential a Kubernetes client expects back from a
// credential plugin. The field names and apiVersion are fixed by kubectl, not
// chosen by us.
type ExecCredential struct {
	Kind       string               `json:"kind"`
	APIVersion string               `json:"apiVersion"`
	Status     ExecCredentialStatus `json:"status"`
}

// ExecCredentialStatus carries the token and the moment it stops being valid.
type ExecCredentialStatus struct {
	Token string `json:"token"`
	// ExpirationTimestamp is what lets kubectl cache the credential rather than
	// invoking the plugin on every request.
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
}

// CreateClusterToken mints a short-lived credential for a managed Kubernetes
// cluster, in the form a Kubernetes client expects from a credential plugin.
//
// Scoped to the one cluster named here: the same issuer serves every cluster, so
// the token is bound to this one and no other will accept it.
func (s *ServicesClient) CreateClusterToken(ctx context.Context, svcSlug string) (*ExecCredential, error) {
	var cred ExecCredential
	if err := s.client.post(ctx, s.svcPath(svcSlug)+"/kubernetes/token", nil, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// GetClusterKubeconfig returns the cluster's kubeconfig as raw YAML.
func (s *ServicesClient) GetClusterKubeconfig(ctx context.Context, svcSlug string) ([]byte, error) {
	return s.client.getRaw(ctx, s.svcPath(svcSlug)+"/kubernetes/kubeconfig")
}
