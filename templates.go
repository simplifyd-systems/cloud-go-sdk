package cloud

import (
	"context"
	"fmt"
	"time"
)

// Templates returns the client for the templates Simplifyd lists for every
// workspace. Deploy one with Workspace(slug).Templates().Deploy.
func (c *Client) Templates() *CatalogTemplatesClient {
	return &CatalogTemplatesClient{client: c}
}

// CatalogTemplatesClient reads the template catalog: templates Simplifyd
// maintains, such as Supabase, which any workspace can deploy.
type CatalogTemplatesClient struct {
	client *Client
}

// List returns every template in the catalog.
func (c *CatalogTemplatesClient) List(ctx context.Context) ([]Template, error) {
	var out []Template
	if err := c.client.get(ctx, "/v1/templates", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one catalog template.
func (c *CatalogTemplatesClient) Get(ctx context.Context, templateSlug string) (*Template, error) {
	var t Template
	if err := c.client.get(ctx, "/v1/templates/"+templateSlug, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Templates returns the client for the workspace's own templates, and for
// deploying any template, the workspace's or the catalog's, into it.
func (w *WorkspaceClient) Templates() *TemplatesClient {
	return &TemplatesClient{client: w.client, workspace: w.slug}
}

// TemplatesClient reads a workspace's templates and deploys templates into the
// workspace's environments.
type TemplatesClient struct {
	client    *Client
	workspace string
}

func (c *TemplatesClient) base() string {
	return fmt.Sprintf("/v1/workspaces/%s/templates", c.workspace)
}

// List returns the workspace's own templates, drafts included. Catalog
// templates are listed by Client.Templates().List.
func (c *TemplatesClient) List(ctx context.Context) ([]Template, error) {
	var out []Template
	if err := c.client.get(ctx, c.base(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one of the workspace's own templates.
func (c *TemplatesClient) Get(ctx context.Context, templateSlug string) (*Template, error) {
	var t Template
	if err := c.client.get(ctx, c.base()+"/"+templateSlug, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Deploy creates every service in a template in one environment of the
// workspace. templateSlug may name one of the workspace's own templates or a
// catalog template.
//
// The services are created, not started: deploy each as with any new service.
// Secrets the template generates are generated here, once, for this
// deployment; the ones the template marks publishable are returned in
// Outputs. Either every service is created or none is. It fails if a service
// would share a name with one already in the environment.
func (c *TemplatesClient) Deploy(ctx context.Context, templateSlug string, in DeployTemplateInput) (*TemplateDeployResult, error) {
	var out TemplateDeployResult
	if err := c.client.post(ctx, c.base()+"/"+templateSlug+"/deploy", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeployTemplateInput is where a template is deployed.
type DeployTemplateInput struct {
	Project string `json:"project"`
	Env     string `json:"env"`
	// InitSQL, for a template that takes it (see Template.TakesInitSQL), runs
	// once, when the template's database is first created: all of it or, on
	// any error, none of it, in which case the database's logs say why.
	InitSQL string `json:"init_sql,omitempty"`
}

// TemplateDeployResult lists the services a deploy created.
type TemplateDeployResult struct {
	Services []Service `json:"services"`
	// Outputs are the values the template marks publishable, such as a
	// Supabase URL and anon key: safe to put in a browser app. Every other
	// value is only shown in the dashboard.
	Outputs []TemplateDeployOutput `json:"outputs,omitempty"`
}

// TemplateDeployOutput is a publishable variable of a created service.
type TemplateDeployOutput struct {
	Service     string `json:"service"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// Template is a set of services deployed together.
type Template struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Icon is a key such as "supabase" naming the icon the console shows.
	Icon       string            `json:"icon,omitempty"`
	Visibility string            `json:"visibility"`
	Status     string            `json:"status"` // draft, published or archived
	Services   []TemplateService `json:"services"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// TemplateService is one service a template creates. Only what describes the
// service is included; its variables and files are applied by the API.
type TemplateService struct {
	// Key is the name the service is created with.
	Key                string                   `json:"key"`
	Type               ServiceType              `json:"type"`
	VCPUs              uint                     `json:"vcpus,omitempty"`
	Memory             uint                     `json:"memory,omitempty"` // MiB
	Docker             *TemplateDockerService   `json:"docker,omitempty"`
	Postgres           *TemplatePostgresService `json:"postgres,omitempty"`
	Ingress            []TemplateServiceIngress `json:"ingress,omitempty"`
	PersistentStorages []TemplateServiceStorage `json:"persistent_storages,omitempty"`
	// InitSQLPath is set on the service a deploy's init SQL is run on.
	InitSQLPath string `json:"init_sql_path,omitempty"`
}

// TemplateDockerService is the image a docker service runs.
type TemplateDockerService struct {
	Image string `json:"image"`
	Tag   string `json:"tag,omitempty"`
}

// TemplatePostgresService sizes a managed Postgres service.
type TemplatePostgresService struct {
	StorageGB uint `json:"storage_gb"`
}

// TemplateServiceIngress is a port the service exposes publicly.
type TemplateServiceIngress struct {
	Port     uint   `json:"port"`
	Protocol string `json:"protocol"` // HTTP, gRPC, TCP or UDP
}

// TemplateServiceStorage is a disk that outlives the service's pod.
type TemplateServiceStorage struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SizeGB    int    `json:"size_gb"`
}

// The size a template service is given when it does not set one; the API
// applies the same defaults.
const (
	DefaultTemplateVCPUs  = 2
	DefaultTemplateMemory = 4096 // MiB
)

// Public reports whether the service is reachable from the internet.
func (s TemplateService) Public() bool {
	return len(s.Ingress) > 0
}

// Size returns what the service is created with, defaults applied.
func (s TemplateService) Size() (vcpus, memoryMiB uint, diskGB int) {
	vcpus, memoryMiB = s.VCPUs, s.Memory
	if vcpus == 0 {
		vcpus = DefaultTemplateVCPUs
	}
	if memoryMiB == 0 {
		memoryMiB = DefaultTemplateMemory
	}
	for _, st := range s.PersistentStorages {
		diskGB += st.SizeGB
	}
	if s.Postgres != nil {
		diskGB += int(s.Postgres.StorageGB)
	}
	return vcpus, memoryMiB, diskGB
}

// TakesInitSQL reports whether a deploy of the template accepts InitSQL, and
// the service it is run on.
func (t Template) TakesInitSQL() (service string, ok bool) {
	for _, s := range t.Services {
		if s.InitSQLPath != "" {
			return s.Key, true
		}
	}
	return "", false
}

// Size returns the total the template's services are created with.
func (t Template) Size() (vcpus, memoryMiB uint, diskGB int) {
	for _, s := range t.Services {
		v, m, d := s.Size()
		vcpus += v
		memoryMiB += m
		diskGB += d
	}
	return vcpus, memoryMiB, diskGB
}
