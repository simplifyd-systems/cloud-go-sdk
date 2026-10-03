package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A site's domains are added and removed one at a time, so attaching a second
// never takes down the first.
func TestStaticSiteAddAndRemoveDomain(t *testing.T) {
	const base = "/v1/workspaces/ws/projects/p/envs/prod/svcs/web/site/domains"
	site := StaticSite{DomainCNAMETarget: "site.simplifyd.app"}
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.EscapedPath())
		switch {
		case r.Method == http.MethodPost && r.URL.Path == base:
			var in AddStaticSiteDomainInput
			_ = json.NewDecoder(r.Body).Decode(&in)
			site.CustomDomains = append(site.CustomDomains, StaticSiteDomain{
				Slug: "slug-" + in.Domain, Domain: in.Domain, Status: "active",
			})
		case r.Method == http.MethodDelete && r.URL.Path == base+"/example.com":
			site.CustomDomains = site.CustomDomains[1:]
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(site)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("prod").Services().StaticSite("web")
	ctx := context.Background()

	if _, err := client.AddDomain(ctx, "example.com"); err != nil {
		t.Fatalf("add apex: %v", err)
	}
	got, err := client.AddDomain(ctx, "www.example.com")
	if err != nil {
		t.Fatalf("add www: %v", err)
	}
	if len(got.CustomDomains) != 2 {
		t.Fatalf("expected both domains, got %+v", got.CustomDomains)
	}
	if d := got.Domain("WWW.example.com."); d == nil || d.Slug != "slug-www.example.com" {
		t.Fatalf("Domain lookup = %+v", d)
	}
	if got.Domain("other.example.com") != nil {
		t.Fatal("Domain should be nil for a domain the site does not have")
	}

	got, err = client.RemoveDomain(ctx, "example.com")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(got.CustomDomains) != 1 || got.CustomDomains[0].Domain != "www.example.com" {
		t.Fatalf("after remove: %+v", got.CustomDomains)
	}
	if calls[len(calls)-1] != "DELETE "+base+"/example.com" {
		t.Errorf("remove called %q", calls[len(calls)-1])
	}
}
