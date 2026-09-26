package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTemplates(t *testing.T) {
	supabase := Template{
		Slug: "tmpl-1", Name: "Supabase", Icon: "supabase", Status: "published",
		Services: []TemplateService{
			{Key: "supabase_db", Type: ServiceTypeDocker, VCPUs: 2, Memory: 2048,
				PersistentStorages: []TemplateServiceStorage{{Name: "data", MountPath: "/var/lib/postgresql", SizeGB: 20}}},
			{Key: "supabase_gateway", Type: ServiceTypeDocker, VCPUs: 1, Memory: 256,
				Ingress: []TemplateServiceIngress{{Port: 8000, Protocol: "HTTP"}}},
		},
	}
	var deployed DeployTemplateInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/templates":
			_ = json.NewEncoder(w).Encode([]Template{supabase})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/templates/tmpl-1":
			_ = json.NewEncoder(w).Encode(supabase)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/workspaces/ws/templates":
			_ = json.NewEncoder(w).Encode([]Template{})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces/ws/templates/tmpl-1/deploy":
			if err := json.NewDecoder(r.Body).Decode(&deployed); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(TemplateDeployResult{
				Services: []Service{{Slug: "svc-1", Name: "supabase_db"}, {Slug: "svc-2", Name: "supabase_gateway"}},
				Outputs:  []TemplateDeployOutput{{Service: "supabase_gateway", Name: "ANON_KEY", Value: "eyJ"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	ctx := context.Background()

	catalog, err := client.Templates().List(ctx)
	if err != nil || len(catalog) != 1 || catalog[0].Icon != "supabase" {
		t.Fatalf("catalog = %#v, err = %v", catalog, err)
	}
	if _, err := client.Templates().Get(ctx, "tmpl-1"); err != nil {
		t.Fatal(err)
	}
	own, err := client.Workspace("ws").Templates().List(ctx)
	if err != nil || len(own) != 0 {
		t.Fatalf("own = %#v, err = %v", own, err)
	}

	result, err := client.Workspace("ws").Templates().Deploy(ctx, "tmpl-1", DeployTemplateInput{Project: "proj", Env: "prod", InitSQL: "create table t ();"})
	if err != nil {
		t.Fatal(err)
	}
	if deployed != (DeployTemplateInput{Project: "proj", Env: "prod", InitSQL: "create table t ();"}) {
		t.Errorf("deploy body = %#v", deployed)
	}
	if len(result.Services) != 2 || result.Services[1].Name != "supabase_gateway" || len(result.Outputs) != 1 || result.Outputs[0].Value != "eyJ" {
		t.Errorf("result = %#v", result)
	}
}

func TestTemplateSize(t *testing.T) {
	tmpl := Template{Services: []TemplateService{
		{Key: "sized", VCPUs: 1, Memory: 512, PersistentStorages: []TemplateServiceStorage{{SizeGB: 20}}},
		{Key: "defaults"}, // the API's defaults: 2 vCPU, 4096 MiB
		{Key: "pg", VCPUs: 1, Memory: 1024, Postgres: &TemplatePostgresService{StorageGB: 10}},
	}}
	vcpus, memory, disk := tmpl.Size()
	if vcpus != 4 || memory != 512+4096+1024 || disk != 30 {
		t.Fatalf("Size() = %d vCPU, %d MiB, %d GB", vcpus, memory, disk)
	}
	if tmpl.Services[1].Public() {
		t.Error("service without ingress reported public")
	}
}

func TestPublishableVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/workspaces/ws/projects/p/envs/e/publishable-variables" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[{"service_slug":"s","service":"supabase_gateway","name":"ANON_KEY","value":"eyJ"}]`))
	}))
	defer server.Close()

	got, err := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("e").PublishableVariables(context.Background())
	if err != nil || len(got) != 1 || got[0].Name != "ANON_KEY" || got[0].Value != "eyJ" {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

func TestSealedVariables(t *testing.T) {
	base := "/v1/workspaces/ws/projects/p/envs/e/svcs/s/variables"
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		switch r.Method + " " + r.URL.Path {
		case "POST " + base, "PUT " + base + "/v1", "POST " + base + "/PW/seal":
			_, _ = w.Write([]byte(`{}`))
		case "POST " + base + "/PW/reveal":
			_, _ = w.Write([]byte(`{"name":"PW","value":"hunter2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	vars := NewClient(WithBaseURL(server.URL)).Workspace("ws").Project("p").Env("e").Services().Variables("s")
	ctx := context.Background()

	if _, err := vars.Set(ctx, "A", "1"); err != nil {
		t.Fatal(err)
	}
	if _, err := vars.SetSealed(ctx, "PW", "x", false); err != nil {
		t.Fatal(err)
	}
	if _, err := vars.UpdateSealed(ctx, "v1", "y", false); err != nil {
		t.Fatal(err)
	}
	if _, present := bodies[0]["sealed"]; present {
		t.Errorf("Set sent sealed: %v", bodies[0])
	}
	if bodies[1]["sealed"] != false || bodies[2]["sealed"] != false {
		t.Errorf("bodies = %v", bodies)
	}
	if got, err := vars.Reveal(ctx, "PW"); err != nil || got != "hunter2" {
		t.Errorf("Reveal = %q, %v", got, err)
	}
	if err := vars.Seal(ctx, "PW"); err != nil {
		t.Error(err)
	}
}
