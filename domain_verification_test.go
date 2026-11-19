package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const pendingVerificationJSON = `{
	"hostname": "cdn.example.com",
	"status": "pending",
	"record_type": "TXT",
	"record_name": "_simplifyd-verify.cdn.example.com",
	"record_value": "simplifyd-verify=3f9c",
	"last_checked_at": "2026-11-19T12:00:01Z",
	"last_error": "No TXT record found at _simplifyd-verify.cdn.example.com yet.",
	"auto_check_until": "2026-12-03T12:00:00Z"
}`

// Every kind of custom domain carries the TXT record that verifies it.
func TestDomainVerificationDecodes(t *testing.T) {
	var svc Service
	body := `{
		"type": "s3_bucket",
		"ingress_ports": [{"slug": "p", "custom_fqdns": [{"slug": "f", "fqdn": "cdn.example.com", "verification": ` + pendingVerificationJSON + `}]}],
		"s3_bucket_svc": {"public": true, "domain_cname_target": "edge.simplifyd.app",
			"custom_domains": [{"slug": "d", "domain": "cdn.example.com", "status": "pending", "verification": ` + pendingVerificationJSON + `}]}
	}`
	if err := json.Unmarshal([]byte(body), &svc); err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]*DomainVerification{
		"ingress": svc.Ingress[0].CustomFQDNs[0].Verification,
		"bucket":  svc.S3Bucket.Domain("CDN.example.com.").Verification,
	} {
		if v == nil {
			t.Fatalf("%s: no verification", name)
		}
		if v.Status != DomainVerificationPending || v.Verified() {
			t.Errorf("%s: status %q, Verified %v", name, v.Status, v.Verified())
		}
		if v.RecordType != "TXT" || v.RecordName != "_simplifyd-verify.cdn.example.com" || v.RecordValue != "simplifyd-verify=3f9c" {
			t.Errorf("%s: record %s %s %s", name, v.RecordType, v.RecordName, v.RecordValue)
		}
		if v.LastError == "" || v.LastCheckedAt == nil || v.AutoCheckUntil == nil || v.VerifiedAt != nil {
			t.Errorf("%s: %+v", name, v)
		}
	}

	var lib VideoLibrary
	if err := json.Unmarshal([]byte(`{"playback_domain": "cdn.example.com", "verification": `+pendingVerificationJSON+`}`), &lib); err != nil {
		t.Fatal(err)
	}
	if lib.Verification == nil || lib.Verification.Hostname != "cdn.example.com" {
		t.Errorf("video library verification = %+v", lib.Verification)
	}

	// An API that predates verification serves every domain, so a missing
	// verification is not reported as pending.
	var none *DomainVerification
	if !none.Verified() {
		t.Error("nil verification should count as verified")
	}
}

func TestCheckDomainVerification(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = io.WriteString(w, strings.Replace(pendingVerificationJSON, `"pending"`, `"verified"`, 1))
	}))
	defer server.Close()

	svcs := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services()
	v, err := svcs.CheckDomainVerification(context.Background(), "web", "cdn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "POST /v1/workspaces/ws/projects/p/envs/prod/svcs/web/domains/verify" {
		t.Errorf("called %s", gotPath)
	}
	if strings.TrimSpace(gotBody) != `{"domain":"cdn.example.com"}` {
		t.Errorf("body %s", gotBody)
	}
	if !v.Verified() {
		t.Errorf("status %q", v.Status)
	}
}

// A bucket is made public, then given domains one at a time.
func TestBucketPublicAccessAndDomains(t *testing.T) {
	const base = "/v1/workspaces/ws/projects/p/envs/prod/svcs/assets/s3"
	bucket := S3Bucket{DomainCNAMETarget: "edge.simplifyd.app"}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.EscapedPath())
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/workspaces/ws/projects/p/envs/prod/svcs/assets":
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "s3_bucket", "s3_bucket_svc": bucket})
			return
		case r.Method == http.MethodPut && r.URL.Path == base+"/public-access":
			var in SetS3BucketPublicAccessInput
			_ = json.NewDecoder(r.Body).Decode(&in)
			bucket.Public = in.Public
		case r.Method == http.MethodPost && r.URL.Path == base+"/domains":
			if !bucket.Public {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"message": "the bucket is private"}`)
				return
			}
			var in AddS3BucketDomainInput
			_ = json.NewDecoder(r.Body).Decode(&in)
			bucket.CustomDomains = append(bucket.CustomDomains, StaticSiteDomain{
				Slug: "slug-" + in.Domain, Domain: in.Domain, Status: "pending",
				Verification: &DomainVerification{Hostname: in.Domain, Status: DomainVerificationPending, RecordType: "TXT"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == base+"/domains/cdn.example.com":
			bucket.CustomDomains = bucket.CustomDomains[1:]
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(bucket)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services().Bucket("assets")
	ctx := context.Background()

	if _, err := client.AddDomain(ctx, "cdn.example.com"); err == nil {
		t.Fatal("adding a domain to a private bucket should fail")
	}
	b, err := client.SetPublicAccess(ctx, true)
	if err != nil || !b.Public {
		t.Fatalf("set public: %v %+v", err, b)
	}
	if _, err := client.AddDomain(ctx, "cdn.example.com"); err != nil {
		t.Fatalf("add: %v", err)
	}
	b, err = client.AddDomain(ctx, "img.example.com")
	if err != nil {
		t.Fatalf("add second: %v", err)
	}
	if d := b.Domain("img.example.com"); d == nil || d.Verification.Verified() {
		t.Fatalf("new domain should be pending verification: %+v", d)
	}
	got, err := client.Get(ctx)
	if err != nil || len(got.CustomDomains) != 2 || got.DomainCNAMETarget != "edge.simplifyd.app" {
		t.Fatalf("get: %v %+v", err, got)
	}
	b, err = client.RemoveDomain(ctx, "cdn.example.com")
	if err != nil || len(b.CustomDomains) != 1 || b.CustomDomains[0].Domain != "img.example.com" {
		t.Fatalf("remove: %v %+v", err, b)
	}
}

// Get refuses a service that is not a bucket rather than returning the empty
// bucket object the API sends for every service.
func TestBucketGetRejectsOtherServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"type": "docker", "s3_bucket_svc": {"public": false}}`)
	}))
	defer server.Close()
	_, err := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services().Bucket("web").Get(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not a bucket") {
		t.Fatalf("err = %v", err)
	}
}
