package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrivateAccessGrantLifecycle(t *testing.T) {
	const basePath = "/v1/workspaces/ws/projects/provider/envs/dev/svcs/db"
	var created CreatePrivateAccessGrantInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == basePath+"/private-access-grants":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == basePath:
			_ = json.NewEncoder(w).Encode(Service{Slug: "db", PrivateHostname: "db.slug.simplifyd.internal", PrivateAccessGrants: []PrivateAccessGrant{{Slug: "grant-1", ConsumerProjectSlug: created.ConsumerProject, ConsumerProjectName: "payments", Protocol: created.Protocol, Port: created.Port}}})
		case r.Method == http.MethodDelete && r.URL.Path == basePath+"/private-access-grants/grant-1":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	services := client.Workspace("ws").Project("provider").Env("dev").Services()
	grant, err := services.CreatePrivateAccessGrant(context.Background(), "db", CreatePrivateAccessGrantInput{ConsumerProject: "consumer", Protocol: "tcp", Port: 5432})
	if err != nil {
		t.Fatal(err)
	}
	if grant.Slug != "grant-1" || grant.Protocol != "TCP" || grant.Port != 5432 {
		t.Fatalf("unexpected grant: %#v", grant)
	}
	if err := services.DeletePrivateAccessGrant(context.Background(), "db", grant.Slug); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePrivateAccessGrantValidatesInput(t *testing.T) {
	services := NewClient().Workspace("ws").Project("provider").Env("dev").Services()
	for _, input := range []CreatePrivateAccessGrantInput{{Protocol: "TCP", Port: 80}, {ConsumerProject: "p", Protocol: "HTTP", Port: 80}, {ConsumerProject: "p", Protocol: "TCP", Port: 0}} {
		if _, err := services.CreatePrivateAccessGrant(context.Background(), "db", input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}
