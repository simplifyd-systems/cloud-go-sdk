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
			_ = json.NewEncoder(w).Encode(TemplateDeployResult{Services: []Service{{Slug: "svc-1", Name: "supabase_db"}, {Slug: "svc-2", Name: "supabase_gateway"}}})
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

	result, err := client.Workspace("ws").Templates().Deploy(ctx, "tmpl-1", DeployTemplateInput{Project: "proj", Env: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if deployed != (DeployTemplateInput{Project: "proj", Env: "prod"}) {
		t.Errorf("deploy body = %#v", deployed)
	}
	if len(result.Services) != 2 || result.Services[1].Name != "supabase_gateway" {
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
