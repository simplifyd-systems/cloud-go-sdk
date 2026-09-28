package cloud

import (
	"context"
	"time"
)

// Email services send through Why.email.
//
// Creating one sets up a Why.email account and an API key. Nothing is sent
// until a domain you own is added and verified: AddDomain returns the DNS
// records to publish, and VerifyDomain checks them. Other services in the
// environment send with the variables the email service publishes —
// ${{email.SMTP_HOST}}, ${{email.SMTP_PASSWORD}} and the rest, named after the
// service — so the key never has to be copied anywhere.

// EmailInput configures an email service on creation.
type EmailInput struct {
	Name string `json:"name,omitempty"`
}

// EmailConfig is an email service as the API returns it.
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	APIURL       string `json:"api_url"`
	// APIKeyPrefix identifies the key without revealing it. The key itself
	// only ever reaches sibling services, as SMTP_PASSWORD and
	// WHY_EMAIL_API_KEY.
	APIKeyPrefix string `json:"api_key_prefix"`
	// HasVerifiedDomain reports whether the service can send yet.
	HasVerifiedDomain bool `json:"has_verified_domain"`
	// Suspended is set while the workspace is suspended for lack of credit.
	Suspended bool          `json:"suspended"`
	Domains   []EmailDomain `json:"domains,omitempty"`
}

// EmailDomain is a domain an email service sends from.
type EmailDomain struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	// Status is "pending" until the domain's records are verified, then
	// "verified".
	Status     string           `json:"status"`
	VerifiedAt *time.Time       `json:"verified_at,omitempty"`
	Records    []EmailDNSRecord `json:"records"`
}

// Verified reports whether mail can be sent from the domain.
func (d EmailDomain) Verified() bool { return d.Status == "verified" }

// EmailDNSRecord is one record to publish at the domain's DNS provider.
// Purpose is "verification", "dkim", "spf" or "dmarc".
type EmailDNSRecord struct {
	Purpose string `json:"purpose"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Value   string `json:"value"`
}

// EmailVariables are the variables every email service publishes to its
// siblings, referenced as ${{<service name>.<variable>}}.
var EmailVariables = []string{
	"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD",
	"WHY_EMAIL_API_KEY", "WHY_EMAIL_API_URL",
}

// EmailClient manages an email service's domains and key.
// Obtain one via Services().Email(svcSlug).
type EmailClient struct {
	services *ServicesClient
	svcSlug  string
}

// Email returns a client for an email service.
func (s *ServicesClient) Email(svcSlug string) *EmailClient {
	return &EmailClient{services: s, svcSlug: svcSlug}
}

func (c *EmailClient) base() string {
	return c.services.svcPath(c.svcSlug) + "/email"
}

// Domains lists the service's sending domains, with each one's DNS records.
func (c *EmailClient) Domains(ctx context.Context) ([]EmailDomain, error) {
	var out struct {
		Domains []EmailDomain `json:"domains"`
	}
	if err := c.services.client.get(ctx, c.base()+"/domains", &out); err != nil {
		return nil, err
	}
	return out.Domains, nil
}

// AddDomain registers a domain to send from. It is pending until the records
// it returns are published and VerifyDomain succeeds.
func (c *EmailClient) AddDomain(ctx context.Context, domain string) (*EmailDomain, error) {
	var out struct {
		Domain EmailDomain `json:"domain"`
	}
	if err := c.services.client.post(ctx, c.base()+"/domains", map[string]string{"domain": domain}, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// VerifyDomain checks a domain's DNS records now. Records that are not visible
// yet leave it pending, or fail with an error explaining which; DNS changes
// can take a while to propagate, so trying again later is expected.
func (c *EmailClient) VerifyDomain(ctx context.Context, domainID string) (*EmailDomain, error) {
	var out struct {
		Domain EmailDomain `json:"domain"`
	}
	if err := c.services.client.post(ctx, c.base()+"/domains/"+domainID+"/verify", nil, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// DeleteDomain stops the service sending from a domain.
func (c *EmailClient) DeleteDomain(ctx context.Context, domainID string) error {
	return c.services.client.delete(ctx, c.base()+"/domains/"+domainID, nil)
}

// RotateKey replaces the service's API key and revokes the old one. Services
// that send with its variables keep the old key until they are redeployed.
func (c *EmailClient) RotateKey(ctx context.Context) (*EmailConfig, error) {
	var out struct {
		Email EmailConfig `json:"email_svc"`
	}
	if err := c.services.client.post(ctx, c.base()+"/key/rotate", nil, &out); err != nil {
		return nil, err
	}
	return &out.Email, nil
}
