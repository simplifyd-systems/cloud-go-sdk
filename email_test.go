package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailClient(t *testing.T) {
	const base = "/v1/workspaces/ws/projects/p/envs/prod/svcs/email/email"
	pending := EmailDomain{ID: "d1", Domain: "mail.acme.com", Status: "pending", Records: []EmailDNSRecord{
		{Purpose: "dkim", Type: "TXT", Name: "smail._domainkey.mail.acme.com", Value: "v=DKIM1; p=abc"},
	}}
	var added map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base+"/domains":
			_ = json.NewDecoder(r.Body).Decode(&added)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": pending})
		case r.Method == http.MethodGet && r.URL.Path == base+"/domains":
			_ = json.NewEncoder(w).Encode(map[string]any{"domains": []EmailDomain{pending}})
		case r.Method == http.MethodPost && r.URL.Path == base+"/domains/d1/verify":
			verified := pending
			verified.Status = "verified"
			_ = json.NewEncoder(w).Encode(map[string]any{"domain": verified})
		case r.Method == http.MethodDelete && r.URL.Path == base+"/domains/d1":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodPost && r.URL.Path == base+"/key/rotate":
			_ = json.NewEncoder(w).Encode(map[string]any{"email_svc": EmailConfig{APIKeyPrefix: "sk_live_new"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	email := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services().Email("email")

	d, err := email.AddDomain(ctx, "mail.acme.com")
	if err != nil || d.ID != "d1" || d.Verified() || len(d.Records) != 1 {
		t.Fatalf("AddDomain = %+v, %v", d, err)
	}
	if added["domain"] != "mail.acme.com" {
		t.Errorf("sent %v", added)
	}
	domains, err := email.Domains(ctx)
	if err != nil || len(domains) != 1 {
		t.Fatalf("Domains = %+v, %v", domains, err)
	}
	if v, err := email.VerifyDomain(ctx, "d1"); err != nil || !v.Verified() {
		t.Fatalf("VerifyDomain = %+v, %v", v, err)
	}
	if err := email.DeleteDomain(ctx, "d1"); err != nil {
		t.Fatal(err)
	}
	if cfg, err := email.RotateKey(ctx); err != nil || cfg.APIKeyPrefix != "sk_live_new" {
		t.Fatalf("RotateKey = %+v, %v", cfg, err)
	}
}
