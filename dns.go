package cloud

import (
	"context"
	"fmt"
	"time"
)

// DNS hosting on Simplifyd Cloud's nameservers.
//
// A zone is a whole registrable domain — example.com, not www.example.com —
// and starts "pending": its records can be set up, but nothing is served until
// the domain's nameservers are changed at its registrar to the ones the zone
// lists. The zone is checked in the background and turns "active" once they
// are; CheckZone checks it now. A domain bought through Domains() gets an
// active zone straight away.
//
// Some records are written by the platform itself — a custom domain's route to
// its service, an email domain's DKIM key. Their Owner is "platform", and they
// can be seen but not changed; the feature that wrote one removes it.

// DNS zone statuses.
const (
	DNSZonePending = "pending"
	DNSZoneActive  = "active"
)

// DNSZone is a zone a workspace hosts.
type DNSZone struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// Nameservers are what the domain must be delegated to.
	Nameservers []string   `json:"nameservers"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	// LastCheckedAt is when a pending zone's delegation was last looked up.
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DNSZoneList is a workspace's zones and the nameservers they are served by.
type DNSZoneList struct {
	Zones       []DNSZone `json:"zones"`
	Nameservers []string  `json:"nameservers"`
}

// DNSRecord is one value at one name in a zone.
type DNSRecord struct {
	Slug string `json:"slug"`
	// Name is relative to the zone: "@" for the domain itself, "www", "*.api".
	Name string `json:"name"`
	// Type is A, AAAA, CNAME, ALIAS, MX, TXT, SRV, CAA or NS.
	Type string `json:"type"`
	TTL  uint32 `json:"ttl"`
	// Value is an address, a hostname, TXT text unquoted, "weight port target"
	// for an SRV, or `flags tag "value"` for a CAA.
	Value string `json:"value"`
	// Priority is an MX's preference or an SRV's priority.
	Priority *uint16 `json:"priority,omitempty"`
	// Owner is "user" or "platform"; platform records cannot be changed.
	Owner string `json:"owner"`
	// ManagedBy names what wrote a platform record.
	ManagedBy string    `json:"managed_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DNSRecordInput creates or replaces a record.
type DNSRecordInput struct {
	// Name is relative ("www"), "@" or empty for the domain itself, or in
	// full ("www.example.com").
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	// TTL is in seconds, from 60 to 86400; 300 when zero.
	TTL uint32 `json:"ttl,omitempty"`
	// Priority is required for MX and SRV records, and ignored otherwise.
	Priority *uint16 `json:"priority,omitempty"`
}

// DNSClient manages a workspace's DNS zones and records.
// Obtain one via Workspace(slug).DNS().
type DNSClient struct {
	client *Client
	ws     string
}

// DNS returns a client for the workspace's DNS zones.
func (w *WorkspaceClient) DNS() *DNSClient {
	return &DNSClient{client: w.client, ws: w.slug}
}

func (d *DNSClient) zones() string {
	return fmt.Sprintf("/v1/workspaces/%s/dns/zones", d.ws)
}

// Zones lists the workspace's zones.
func (d *DNSClient) Zones(ctx context.Context) (*DNSZoneList, error) {
	var out DNSZoneList
	if err := d.client.get(ctx, d.zones(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateZone adds a zone for a domain the workspace owns elsewhere. It is
// pending until the domain's nameservers point at the zone's Nameservers.
func (d *DNSClient) CreateZone(ctx context.Context, name string) (*DNSZone, error) {
	var out struct {
		Zone DNSZone `json:"zone"`
	}
	if err := d.client.post(ctx, d.zones(), map[string]string{"name": name}, &out); err != nil {
		return nil, err
	}
	return &out.Zone, nil
}

// Zone returns one zone.
func (d *DNSClient) Zone(ctx context.Context, zoneSlug string) (*DNSZone, error) {
	var out struct {
		Zone DNSZone `json:"zone"`
	}
	if err := d.client.get(ctx, d.zones()+"/"+zoneSlug, &out); err != nil {
		return nil, err
	}
	return &out.Zone, nil
}

// DeleteZone stops hosting a zone and deletes its records.
func (d *DNSClient) DeleteZone(ctx context.Context, zoneSlug string) error {
	return d.client.delete(ctx, d.zones()+"/"+zoneSlug, nil)
}

// CheckZone looks up a pending zone's delegation now. A zone whose nameservers
// do not point at ours yet stays pending; nameserver changes can take hours to
// be seen.
func (d *DNSClient) CheckZone(ctx context.Context, zoneSlug string) (*DNSZone, error) {
	var out struct {
		Zone DNSZone `json:"zone"`
	}
	if err := d.client.post(ctx, d.zones()+"/"+zoneSlug+"/check", nil, &out); err != nil {
		return nil, err
	}
	return &out.Zone, nil
}

// Records lists a zone's records, the platform's included.
func (d *DNSClient) Records(ctx context.Context, zoneSlug string) ([]DNSRecord, error) {
	var out struct {
		Records []DNSRecord `json:"records"`
	}
	if err := d.client.get(ctx, d.zones()+"/"+zoneSlug+"/records", &out); err != nil {
		return nil, err
	}
	return out.Records, nil
}

// CreateRecord adds a record to a zone.
func (d *DNSClient) CreateRecord(ctx context.Context, zoneSlug string, input DNSRecordInput) (*DNSRecord, error) {
	var out struct {
		Record DNSRecord `json:"record"`
	}
	if err := d.client.post(ctx, d.zones()+"/"+zoneSlug+"/records", input, &out); err != nil {
		return nil, err
	}
	return &out.Record, nil
}

// UpdateRecord replaces a record: every field is set from input.
func (d *DNSClient) UpdateRecord(ctx context.Context, zoneSlug, recordSlug string, input DNSRecordInput) (*DNSRecord, error) {
	var out struct {
		Record DNSRecord `json:"record"`
	}
	if err := d.client.put(ctx, d.zones()+"/"+zoneSlug+"/records/"+recordSlug, input, &out); err != nil {
		return nil, err
	}
	return &out.Record, nil
}

// DeleteRecord removes a record from a zone.
func (d *DNSClient) DeleteRecord(ctx context.Context, zoneSlug, recordSlug string) error {
	return d.client.delete(ctx, d.zones()+"/"+zoneSlug+"/records/"+recordSlug, nil)
}
