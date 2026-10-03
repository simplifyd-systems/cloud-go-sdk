package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDomainsClient(t *testing.T) {
	const base = "/v1/workspaces/ws/domains"
	active := Domain{Slug: "d1", Name: "acme.com", Status: DomainActive, AutoRenew: true}
	var registered RegisterDomainInput
	var savedRegistrant *DomainContact
	var updated map[string]any
	var renewed map[string]int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base+"/search":
			if r.URL.Query().Get("q") != "acme shop" {
				t.Errorf("q = %q", r.URL.Query().Get("q"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []DomainQuote{{Name: "acme.com", Available: true, Years: 1, Total: 161250000}}})
		case r.Method == http.MethodGet && r.URL.Path == base+"/quote":
			if r.URL.Query().Get("name") != "acme.com" || r.URL.Query().Get("years") != "2" {
				t.Errorf("quote query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"quote": DomainQuote{Name: "acme.com", Available: true, Years: 2}})
		case r.Method == http.MethodGet && r.URL.Path == base:
			_ = json.NewEncoder(w).Encode(map[string]any{"domains": []Domain{active}, "sales_enabled": true})
		case r.Method == http.MethodPost && r.URL.Path == base:
			_ = json.NewDecoder(r.Body).Decode(&registered)
			pending := active
			pending.Status = DomainRegistering
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": pending})
		case r.Method == http.MethodGet && r.URL.Path == base+"/registrant":
			_ = json.NewEncoder(w).Encode(map[string]any{"registrant": savedRegistrant})
		case r.Method == http.MethodPut && r.URL.Path == base+"/registrant":
			var c DomainContact
			_ = json.NewDecoder(r.Body).Decode(&c)
			c.Country = "NG"
			savedRegistrant = &c
			_ = json.NewEncoder(w).Encode(map[string]any{"registrant": c})
		case r.Method == http.MethodGet && r.URL.Path == base+"/d1":
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": DomainDetail{Domain: active, Locked: true, Orders: []DomainOrder{{Kind: "registration"}}}})
		case r.Method == http.MethodPatch && r.URL.Path == base+"/d1":
			_ = json.NewDecoder(r.Body).Decode(&updated)
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": DomainDetail{Domain: active}})
		case r.Method == http.MethodPost && r.URL.Path == base+"/d1/renew":
			_ = json.NewDecoder(r.Body).Decode(&renewed)
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": active})
		case r.Method == http.MethodGet && r.URL.Path == base+"/d1/auth-code":
			_ = json.NewEncoder(w).Encode(map[string]any{"auth_code": "x9-secret"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	domains := NewClient(WithBaseURL(server.URL)).Workspace("ws").Domains()

	if results, err := domains.Search(ctx, "acme shop"); err != nil || len(results) != 1 || results[0].Total != 161250000 {
		t.Fatalf("Search = %+v, %v", results, err)
	}
	if q, err := domains.Quote(ctx, "acme.com", 2); err != nil || q.Years != 2 {
		t.Fatalf("Quote = %+v, %v", q, err)
	}
	if list, err := domains.List(ctx); err != nil || !list.SalesEnabled || len(list.Domains) != 1 {
		t.Fatalf("List = %+v, %v", list, err)
	}

	if r, err := domains.Registrant(ctx); err != nil || r != nil {
		t.Fatalf("no saved registrant = %+v, %v", r, err)
	}
	saved, err := domains.SetRegistrant(ctx, DomainContact{FirstName: "Ada", LastName: "Obi", Country: "ng", Phone: "+2348012345678"})
	if err != nil || saved.Country != "NG" {
		t.Fatalf("SetRegistrant = %+v, %v", saved, err)
	}
	if r, err := domains.Registrant(ctx); err != nil || r == nil || r.FirstName != "Ada" {
		t.Fatalf("Registrant = %+v, %v", r, err)
	}

	// Without a registrant, none is sent, so the saved one is used.
	off := false
	d, err := domains.Register(ctx, RegisterDomainInput{Name: "acme.com", Years: 2, AutoRenew: &off})
	if err != nil || d.Status != DomainRegistering {
		t.Fatalf("Register = %+v, %v", d, err)
	}
	if registered.Name != "acme.com" || registered.Years != 2 || registered.AutoRenew == nil || *registered.AutoRenew ||
		registered.Registrant != nil {
		t.Errorf("registered %+v", registered)
	}

	if detail, err := domains.Get(ctx, "d1"); err != nil || !detail.Locked || detail.Name != "acme.com" || len(detail.Orders) != 1 {
		t.Fatalf("Get = %+v, %v", detail, err)
	}

	// Only the settings given are sent, so a PATCH never resets the other.
	on := true
	if _, err := domains.Update(ctx, "d1", UpdateDomainInput{Locked: &on}); err != nil {
		t.Fatal(err)
	}
	if _, sent := updated["auto_renew"]; sent || updated["locked"] != true {
		t.Errorf("update sent %v", updated)
	}

	if _, err := domains.Renew(ctx, "d1", 3); err != nil || renewed["years"] != 3 {
		t.Fatalf("Renew sent %v, %v", renewed, err)
	}
	if code, err := domains.AuthCode(ctx, "d1"); err != nil || code != "x9-secret" {
		t.Fatalf("AuthCode = %q, %v", code, err)
	}
}

func TestFormatNaira(t *testing.T) {
	for in, want := range map[int64]string{
		0:           "₦0.00",
		10_000:      "₦1.00",
		161_250_000: "₦16,125.00",
		123_456_789: "₦12,345.68",
		-25_000:     "-₦2.50",
		100_000_000: "₦10,000.00",
	} {
		if got := FormatNaira(in); got != want {
			t.Errorf("FormatNaira(%d) = %q, want %q", in, got, want)
		}
	}
}
