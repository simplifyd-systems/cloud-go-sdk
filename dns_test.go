package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDNSClient(t *testing.T) {
	const base = "/v1/workspaces/ws/dns/zones"
	zone := DNSZone{Slug: "z1", Name: "acme.com", Status: DNSZonePending, Nameservers: []string{"taiwo.simplifyd.net", "kehinde.simplifyd.net"}}
	mx := uint16(10)
	var created, replaced map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(map[string]any{"zones": []DNSZone{zone}, "nameservers": zone.Nameservers})
		case r.Method == http.MethodPost && r.URL.Path == base:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"zone": zone})
		case r.Method == http.MethodGet && r.URL.Path == base+"/z1":
			_ = json.NewEncoder(w).Encode(map[string]any{"zone": zone})
		case r.Method == http.MethodPost && r.URL.Path == base+"/z1/check":
			active := zone
			active.Status = DNSZoneActive
			_ = json.NewEncoder(w).Encode(map[string]any{"zone": active})
		case r.Method == http.MethodDelete && r.URL.Path == base+"/z1":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodGet && r.URL.Path == base+"/z1/records":
			_ = json.NewEncoder(w).Encode(map[string]any{"records": []DNSRecord{
				{Slug: "r1", Name: "@", Type: "MX", Value: "mx.acme.com", Priority: &mx, Owner: "user"},
				{Slug: "r2", Name: "api", Type: "A", Value: "102.221.184.10", Owner: "platform", ManagedBy: "ingress:x"},
			}})
		case r.Method == http.MethodPost && r.URL.Path == base+"/z1/records":
			_ = json.NewDecoder(r.Body).Decode(&created)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"record": DNSRecord{Slug: "r3", Name: "www", Type: "CNAME", Value: "acme.com", TTL: 300}})
		case r.Method == http.MethodPut && r.URL.Path == base+"/z1/records/r3":
			_ = json.NewDecoder(r.Body).Decode(&replaced)
			_ = json.NewEncoder(w).Encode(map[string]any{"record": DNSRecord{Slug: "r3", Name: "www", Type: "CNAME", Value: "acme.net", TTL: 600}})
		case r.Method == http.MethodDelete && r.URL.Path == base+"/z1/records/r3":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	dns := NewClient(WithBaseURL(server.URL)).Workspace("ws").DNS()

	if list, err := dns.Zones(ctx); err != nil || len(list.Zones) != 1 || len(list.Nameservers) != 2 {
		t.Fatalf("Zones = %+v, %v", list, err)
	}
	if z, err := dns.CreateZone(ctx, "acme.com"); err != nil || z.Status != DNSZonePending {
		t.Fatalf("CreateZone = %+v, %v", z, err)
	}
	if z, err := dns.Zone(ctx, "z1"); err != nil || z.Name != "acme.com" {
		t.Fatalf("Zone = %+v, %v", z, err)
	}
	if z, err := dns.CheckZone(ctx, "z1"); err != nil || z.Status != DNSZoneActive {
		t.Fatalf("CheckZone = %+v, %v", z, err)
	}

	records, err := dns.Records(ctx, "z1")
	if err != nil || len(records) != 2 || *records[0].Priority != 10 || records[1].Owner != "platform" {
		t.Fatalf("Records = %+v, %v", records, err)
	}

	if r, err := dns.CreateRecord(ctx, "z1", DNSRecordInput{Name: "www", Type: "CNAME", Value: "acme.com"}); err != nil || r.Slug != "r3" {
		t.Fatalf("CreateRecord = %+v, %v", r, err)
	}
	// A zero TTL and nil priority are left out, so the API's defaults apply.
	if _, sent := created["ttl"]; sent {
		t.Errorf("zero TTL was sent: %v", created)
	}
	if _, sent := created["priority"]; sent {
		t.Errorf("nil priority was sent: %v", created)
	}

	if r, err := dns.UpdateRecord(ctx, "z1", "r3", DNSRecordInput{Name: "www", Type: "CNAME", Value: "acme.net", TTL: 600}); err != nil || r.TTL != 600 {
		t.Fatalf("UpdateRecord = %+v, %v", r, err)
	}
	if replaced["value"] != "acme.net" || replaced["ttl"] != float64(600) {
		t.Errorf("update sent %v", replaced)
	}
	if err := dns.DeleteRecord(ctx, "z1", "r3"); err != nil {
		t.Fatal(err)
	}
	if err := dns.DeleteZone(ctx, "z1"); err != nil {
		t.Fatal(err)
	}
}
