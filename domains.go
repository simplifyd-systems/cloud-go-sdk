package cloud

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Domain names bought through Simplifyd Cloud.
//
// Buying a name charges the workspace wallet, VAT included, before it is
// registered. The registrant is the domain's legal owner, and the registrar
// emails them to confirm their address. A workspace saves its registrant once
// (SetRegistrant, or the console's settings) and every purchase uses it. A
// bought domain gets a DNS zone with us straight away; its records are managed
// with DNS().
//
// Prices are in centiKobo (1/10000 Naira), like the rest of the billing API.
// FormatNaira renders one for display.

// Domain statuses.
const (
	DomainRegistering = "registering"
	DomainActive      = "active"
	DomainFailed      = "failed"
	DomainExpired     = "expired"
)

// Domain is a domain a workspace bought.
type Domain struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	// Status is "registering" while the registrar has yet to confirm a paid
	// registration, then "active"; "failed" if it was refused and refunded.
	Status     string        `json:"status"`
	ExpiresAt  *time.Time    `json:"expires_at,omitempty"`
	AutoRenew  bool          `json:"auto_renew"`
	Registrant DomainContact `json:"registrant"`
	// RenewalError says why the last automatic renewal could not be made,
	// e.g. the wallet could not cover it.
	RenewalError   string     `json:"renewal_error,omitempty"`
	RenewalErrorAt *time.Time `json:"renewal_error_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// DomainDetail is a domain with its registrar settings and orders.
type DomainDetail struct {
	Domain
	// Locked is the registrar transfer lock: while it is on, the domain cannot
	// be moved to another registrar.
	Locked      bool     `json:"locked"`
	Nameservers []string `json:"nameservers,omitempty"`
	// TransferLockedUntil is the registry's own lock after a registration or
	// transfer, which no setting lifts.
	TransferLockedUntil *time.Time    `json:"transfer_locked_until,omitempty"`
	Orders              []DomainOrder `json:"orders"`
}

// DomainOrder is one purchase: a registration or a renewal.
type DomainOrder struct {
	Slug string `json:"slug"`
	// Kind is "registration" or "renewal".
	Kind   string `json:"kind"`
	Years  int    `json:"years"`
	Amount int64  `json:"amount"` // centiKobo, before VAT
	VAT    int64  `json:"vat"`    // centiKobo
	// Status is "pending", "completed" or "refunded".
	Status    string    `json:"status"`
	Failure   string    `json:"failure,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// DomainContact is a domain's registrant: its legal owner.
type DomainContact struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	CompanyName string `json:"company_name,omitempty"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2,omitempty"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
	// Country is an ISO 3166-1 alpha-2 code, e.g. "NG".
	Country string `json:"country"`
	Email   string `json:"email"`
	// Phone is in E.164 form, e.g. "+2348012345678".
	Phone string `json:"phone"`
}

// DomainQuote is what buying or renewing a name costs.
type DomainQuote struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	// Reason is why a name cannot be bought, when it cannot.
	Reason string `json:"reason,omitempty"`
	Years  int    `json:"years"`
	// Price is the net charge for Years, VAT is on top, and Total is what
	// leaves the wallet. RenewalPrice is the net price of a year's renewal.
	Price        int64 `json:"price"` // centiKobo
	VAT          int64 `json:"vat"`
	Total        int64 `json:"total"`
	RenewalPrice int64 `json:"renewal_price"`
}

// DomainList is a workspace's domains.
type DomainList struct {
	Domains []Domain `json:"domains"`
	// SalesEnabled reports whether new domains can be bought right now.
	SalesEnabled bool `json:"sales_enabled"`
}

// RegisterDomainInput buys a domain.
type RegisterDomainInput struct {
	Name string `json:"name"`
	// Years defaults to 1, and can be at most 10.
	Years int `json:"years,omitempty"`
	// AutoRenew defaults to on: the domain is renewed from the wallet 30 days
	// before it expires.
	AutoRenew *bool `json:"auto_renew,omitempty"`
	// Registrant overrides the workspace's saved registrant for this purchase.
	// Leave it nil to use the saved one, which is what the console does.
	Registrant *DomainContact `json:"registrant,omitempty"`
}

// UpdateDomainInput changes a domain's settings. Nil fields are left alone.
type UpdateDomainInput struct {
	AutoRenew *bool `json:"auto_renew,omitempty"`
	Locked    *bool `json:"locked,omitempty"`
}

// DomainsClient buys and manages a workspace's domains.
// Obtain one via Workspace(slug).Domains().
type DomainsClient struct {
	client *Client
	ws     string
}

// Domains returns a client for the workspace's domains.
func (w *WorkspaceClient) Domains() *DomainsClient {
	return &DomainsClient{client: w.client, ws: w.slug}
}

func (d *DomainsClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/domains", d.ws)
}

// Search looks up names matching query — a word, or a full name like
// acme.com — with whether each can be bought and its price for a year.
func (d *DomainsClient) Search(ctx context.Context, query string) ([]DomainQuote, error) {
	var out struct {
		Results []DomainQuote `json:"results"`
	}
	if err := d.client.get(ctx, d.base()+"/search?q="+url.QueryEscape(query), &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

// Quote prices registering name for years (1 when zero).
func (d *DomainsClient) Quote(ctx context.Context, name string, years int) (*DomainQuote, error) {
	q := url.Values{"name": {name}}
	if years > 0 {
		q.Set("years", strconv.Itoa(years))
	}
	var out struct {
		Quote DomainQuote `json:"quote"`
	}
	if err := d.client.get(ctx, d.base()+"/quote?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out.Quote, nil
}

// List returns the workspace's domains.
func (d *DomainsClient) List(ctx context.Context) (*DomainList, error) {
	var out DomainList
	if err := d.client.get(ctx, d.base(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Registrant returns the registrant the workspace buys domains as, or nil if
// none has been saved.
func (d *DomainsClient) Registrant(ctx context.Context) (*DomainContact, error) {
	var out struct {
		Registrant *DomainContact `json:"registrant"`
	}
	if err := d.client.get(ctx, d.base()+"/registrant", &out); err != nil {
		return nil, err
	}
	return out.Registrant, nil
}

// SetRegistrant saves the registrant the workspace buys domains as. It is
// checked as the registrar would, and returned as stored (country upper-cased,
// phone without spaces). Domains already bought keep their own registrant.
func (d *DomainsClient) SetRegistrant(ctx context.Context, c DomainContact) (*DomainContact, error) {
	var out struct {
		Registrant *DomainContact `json:"registrant"`
	}
	if err := d.client.put(ctx, d.base()+"/registrant", c, &out); err != nil {
		return nil, err
	}
	return out.Registrant, nil
}

// Register buys a domain, charging the workspace wallet. Without a Registrant
// in input it is registered to the workspace's saved one, and fails, charging
// nothing, if none is saved. The returned domain
// may still be "registering": it is paid for, and the registration completes
// in the background. If the registrar refuses it, the wallet is refunded.
func (d *DomainsClient) Register(ctx context.Context, input RegisterDomainInput) (*Domain, error) {
	var out struct {
		Domain Domain `json:"domain"`
	}
	if err := d.client.post(ctx, d.base(), input, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// Get returns a domain with its registrar settings and orders.
func (d *DomainsClient) Get(ctx context.Context, slug string) (*DomainDetail, error) {
	var out struct {
		Domain DomainDetail `json:"domain"`
	}
	if err := d.client.get(ctx, d.base()+"/"+slug, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// Update changes a domain's auto-renewal or transfer lock.
func (d *DomainsClient) Update(ctx context.Context, slug string, input UpdateDomainInput) (*DomainDetail, error) {
	var out struct {
		Domain DomainDetail `json:"domain"`
	}
	if err := d.client.patch(ctx, d.base()+"/"+slug, input, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// Renew extends a domain by years (1 when zero), charging the wallet now.
func (d *DomainsClient) Renew(ctx context.Context, slug string, years int) (*Domain, error) {
	var out struct {
		Domain Domain `json:"domain"`
	}
	body := map[string]int{"years": years}
	if err := d.client.post(ctx, d.base()+"/"+slug+"/renew", body, &out); err != nil {
		return nil, err
	}
	return &out.Domain, nil
}

// AuthCode returns the code that moves a domain to another registrar.
// Only workspace owners can see it.
func (d *DomainsClient) AuthCode(ctx context.Context, slug string) (string, error) {
	var out struct {
		AuthCode string `json:"auth_code"`
	}
	if err := d.client.get(ctx, d.base()+"/"+slug+"/auth-code", &out); err != nil {
		return "", err
	}
	return out.AuthCode, nil
}

// FormatNaira renders an amount in centiKobo as naira, e.g. "₦16,125.00".
func FormatNaira(centiKobo int64) string {
	sign := ""
	if centiKobo < 0 {
		sign, centiKobo = "-", -centiKobo
	}
	// Round to the nearest kobo.
	kobo := (centiKobo + 50) / 100
	whole := strconv.FormatInt(kobo/100, 10)
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return fmt.Sprintf("%s₦%s.%02d", sign, b.String(), kobo%100)
}
