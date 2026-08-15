package cloud

import "time"

// ── Auth ──────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned by Login.
type LoginResponse struct {
	Token           string `json:"jwt"`
	ActiveWorkspace string `json:"active_workspace"`
	ActiveProject   string `json:"active_project"`
	ActiveEnv       string `json:"active_env"`
}

// User represents an authenticated Simplifyd Cloud account.
type User struct {
	Slug      string    `json:"slug"`
	Email     string    `json:"username"`
	Name      string    `json:"name"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Workspace ─────────────────────────────────────────────────────────────────

// Workspace is a billing and collaboration boundary.
type Workspace struct {
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	WalletBalance int64     `json:"wallet_balance"` // kobo (1/100 Naira)
	CreatedAt     time.Time `json:"created_at"`
}

// WorkspaceMember is a user with a role in a workspace.
type WorkspaceMember struct {
	Slug  string `json:"slug"`
	Email string `json:"username"`
	Name  string `json:"name"`
	// Role is one of "owner", "developer", or "billing".
	Role string `json:"role"`
}

// Registry is the container image registry for a workspace.
type Registry struct {
	Name            string `json:"name"`
	HarborProjectID int    `json:"harbor_project_id"`
	RegistryURL     string `json:"registry_url"`
}

// RegistryCredentials contains push/pull credentials for the workspace registry.
type RegistryCredentials struct {
	RegistryURL string `json:"registry_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	// Cred is a base64-encoded Docker RegistryAuth JSON blob.
	Cred string `json:"cred"`
}

// RegistryRepo is a repository within the workspace registry.
type RegistryRepo struct {
	Name string `json:"name"`
}

// RegistryTag is a single tag pointing at an artifact in a repository.
type RegistryTag struct {
	Name      string `json:"name"`
	PushTime  string `json:"push_time"`
	PullTime  string `json:"pull_time"`
	Immutable bool   `json:"immutable"`
}

// RegistryArtifact is an image manifest in a repository, along with the tags
// pointing at it. An artifact with no tags is an untagged leftover manifest.
type RegistryArtifact struct {
	Digest   string        `json:"digest"`
	Size     int64         `json:"size"`
	Type     string        `json:"type"`
	PushTime string        `json:"push_time"`
	PullTime string        `json:"pull_time"`
	Tags     []RegistryTag `json:"tags"`
}

// ── Project ───────────────────────────────────────────────────────────────────

// Project groups environments under a workspace.
type Project struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Workspace   string    `json:"workspace"`
	NetworkSlug string    `json:"network_slug,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Token is a scoped project API token (sk_proj_*).
type Token struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Scope   string `json:"scope"`
	Project string `json:"project"`
	Env     *Env   `json:"env,omitempty"`
	// Key is the full token value — only present on creation.
	Key string `json:"key,omitempty"`
	// MaskedKey is the partially-redacted key for display.
	MaskedKey string    `json:"masked_key,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// RegistryPush reports whether the token may read registry credentials.
	RegistryPush bool `json:"registry_push"`
}

// ── Environment ───────────────────────────────────────────────────────────────

// Env is a deployment environment (e.g. "production", "staging").
type Env struct {
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Variable is a key/value pair available to services in an environment.
type Variable struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ── Service ───────────────────────────────────────────────────────────────────

// ServiceType enumerates supported service kinds.
type ServiceType string

const (
	ServiceTypeDocker        ServiceType = "docker"
	ServiceTypePostgres      ServiceType = "postgres"
	ServiceTypeRedis         ServiceType = "redis"
	ServiceTypeHTTPGateway   ServiceType = "http_gateway"
	ServiceTypeS3Bucket      ServiceType = "s3_bucket"
	ServiceTypeZerodataProxy ServiceType = "zerodata_proxy"
	ServiceTypeStaticSite    ServiceType = "static_site"
)

// ServiceStatus is the current lifecycle state of a service.
type ServiceStatus string

const (
	ServiceStatusPending       ServiceStatus = "pending"
	ServiceStatusDeploying     ServiceStatus = "deploying"
	ServiceStatusRunning       ServiceStatus = "running"
	ServiceStatusOffline       ServiceStatus = "offline"
	ServiceStatusFailed        ServiceStatus = "failed"
	ServiceStatusRolloutFailed ServiceStatus = "rollout_failed"
)

// Service is a deployable unit running inside an environment.
type Service struct {
	Slug     string        `json:"slug"`
	Name     string        `json:"name"`
	Type     ServiceType   `json:"type"`
	VCPUs    uint          `json:"vcpus"`
	Memory   uint          `json:"memory"` // MiB
	Replicas uint          `json:"replicas"`
	Region   string        `json:"region"`
	Status   ServiceStatus `json:"status"`

	Docker   *DockerConfig   `json:"docker_image_svc,omitempty"`
	Postgres *PostgresConfig `json:"postgres_svc,omitempty"`
	Redis    *RedisConfig    `json:"redis_svc,omitempty"`

	Variables           []Variable           `json:"variables,omitempty"`
	Ingress             []IngressPort        `json:"ingress_ports,omitempty"`
	Configs             []ServiceConfig      `json:"configs,omitempty"`
	Changeset           []ChangesetEntry     `json:"changeset,omitempty"`
	PrivateHostname     string               `json:"private_hostname,omitempty"`
	PrivateAccessGrants []PrivateAccessGrant `json:"private_access_grants,omitempty"`
	LivenessProbe       *ServiceProbe        `json:"liveness_probe,omitempty"`
	ReadinessProbe      *ServiceProbe        `json:"readiness_probe,omitempty"`
	StartupProbe        *ServiceProbe        `json:"startup_probe,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// PrivateAccessGrant permits services in one consumer project to connect to a
// specific private port on this service.
type PrivateAccessGrant struct {
	Slug                string `json:"slug"`
	ConsumerProjectSlug string `json:"consumer_project_slug"`
	ConsumerProjectName string `json:"consumer_project_name"`
	Protocol            string `json:"protocol"`
	Port                uint   `json:"port"`
}

type CreatePrivateAccessGrantInput struct {
	ConsumerProject string `json:"consumer_project"`
	Protocol        string `json:"protocol"`
	Port            uint   `json:"port"`
}

// DockerConfig holds configuration for a Docker service.
type DockerConfig struct {
	Image                  string   `json:"image"`
	Tag                    string   `json:"tag"`
	RegistryUsername       string   `json:"registry_username,omitempty"`
	HasRegistryCredentials bool     `json:"has_registry_credentials,omitempty"`
	StartCommand           string   `json:"start_command,omitempty"`
	StartCommandArgs       []string `json:"start_command_args,omitempty"`
}

// PostgresConfig holds configuration for a managed PostgreSQL service.
type PostgresConfig struct {
	Image          string                 `json:"image"`
	Tag            string                 `json:"tag"`
	ConnectionInfo PostgresConnectionInfo `json:"connection_info"`
	Parameters     map[string]string      `json:"parameters,omitempty"`
}

// PostgresParameters describes customer-controlled PostgreSQL server settings
// and the parameter names accepted by the platform.
type PostgresParameters struct {
	Parameters map[string]string `json:"parameters"`
	Supported  []string          `json:"supported,omitempty"`
}

type UpdatePostgresParametersInput struct {
	Parameters map[string]string `json:"parameters"`
}

// PostgresConnectionInfo contains the credentials for a PostgreSQL service.
type PostgresConnectionInfo struct {
	User          string `json:"user"`
	Password      string `json:"password"`
	ConnectionURL string `json:"connection_url"`
}

// RedisConfig holds configuration for a managed Redis service.
type RedisConfig struct {
	// Mode is one of "standalone", "replication", or "cluster".
	Mode     string `json:"mode"`
	Replicas int    `json:"replicas"`
}

// IngressPort is an external network endpoint for a service.
type IngressPort struct {
	Slug        string `json:"slug"`
	Protocol    string `json:"protocol"` // "HTTP", "gRPC", "TCP"
	Port        uint   `json:"port"`
	VanityFQDN  string `json:"vanity_fqdn,omitempty"`
	CustomFQDNs []FQDN `json:"custom_fqdns,omitempty"`
	// AllowedSourceRanges is the client IP allowlist (CIDRs) enforced on the
	// port's public LoadBalancer. Empty means open to all. TCP/UDP ports only.
	AllowedSourceRanges []string `json:"allowed_source_ranges,omitempty"`
}

// FQDN is a custom domain attached to a service ingress port.
type FQDN struct {
	Slug       string `json:"slug"`
	FQDN       string `json:"fqdn"`
	CNAME      string `json:"cname,omitempty"`
	Verified   bool   `json:"verified"`
	CertStatus string `json:"cert_status,omitempty"`
}

// ServiceConfig is a static file mounted into a service container.
type ServiceConfig struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	MountPath string `json:"mount_path"`
}

// ChangesetEntry describes a pending (un-deployed) change on a service.
type ChangesetEntry struct {
	Type          string `json:"type"`
	Action        string `json:"action"`
	Name          string `json:"name"`
	PreviousValue string `json:"previous_value"`
	NewValue      string `json:"new_value"`
}

// ── Create / Update inputs ────────────────────────────────────────────────────

// CreateServiceInput is the request body for creating a new service.
type CreateServiceInput struct {
	Name       string           `json:"name"`
	Type       ServiceType      `json:"type"`
	VCPUs      uint             `json:"vcpus,omitempty"`
	Memory     uint             `json:"memory,omitempty"`
	Docker     *DockerInput     `json:"docker_image_svc,omitempty"`
	Postgres   *PostgresInput   `json:"postgres_svc,omitempty"`
	Redis      *RedisInput      `json:"redis_svc,omitempty"`
	S3Bucket   *S3BucketInput   `json:"s3_bucket_svc,omitempty"`
	StaticSite *StaticSiteInput `json:"static_site_svc,omitempty"`
}

// StaticSiteInput configures a static site service on creation. A static site
// serves files straight from object storage: there is no container, so no
// image, resources or replicas apply.
type StaticSiteInput struct {
	Name string `json:"name,omitempty"`
	// IndexDocument is the object served for a directory request, default
	// index.html.
	IndexDocument string `json:"index_document,omitempty"`
	// ErrorDocument is the object served for a request that matches nothing,
	// defaulting to IndexDocument. Pointing it at the index is what makes a
	// client-side router work on deep links.
	ErrorDocument string `json:"error_document,omitempty"`
}

// StaticSite describes a static site service.
type StaticSite struct {
	DisplayName   string `json:"display_name"`
	BucketName    string `json:"bucket_name"`
	Status        string `json:"status"`
	IndexDocument string `json:"index_document"`
	ErrorDocument string `json:"error_document"`
	// DefaultURL is the always-available platform URL for the site.
	DefaultURL string `json:"default_url,omitempty"`
	// CustomDomain is served once its DNS points at DomainCNAMETarget.
	CustomDomain      string     `json:"custom_domain,omitempty"`
	DomainStatus      string     `json:"domain_status,omitempty"`
	DomainCNAMETarget string     `json:"domain_cname_target,omitempty"`
	BytesUsed         int64      `json:"bytes_used"`
	FileCount         int64      `json:"file_count"`
	LastPublishedAt   *time.Time `json:"last_published_at,omitempty"`
}

// StaticSiteFile is one file in a site publish. Content travels inline, so a
// caller with no filesystem and no way to make an out-of-band upload can
// publish a complete site in a single request.
type StaticSiteFile struct {
	// Path is the file's path within the site, e.g. "assets/app.js".
	Path string `json:"path"`
	// Content is the file body: UTF-8 text, or base64 when Encoding says so.
	Content string `json:"content"`
	// Encoding is "utf8" (default) or "base64". Binary files must use base64.
	Encoding string `json:"encoding,omitempty"`
	// ContentType overrides the type inferred from the path extension.
	ContentType string `json:"content_type,omitempty"`
}

// PublishStaticSiteInput is the request body for publishing site files.
type PublishStaticSiteInput struct {
	Files []StaticSiteFile `json:"files"`
	// Prune removes objects not in Files, making the publish a full replace.
	// Defaults to true server-side when nil.
	Prune *bool `json:"prune,omitempty"`
}

// StaticSitePublishResult reports what a publish changed.
type StaticSitePublishResult struct {
	FilesUploaded int      `json:"files_uploaded"`
	FilesDeleted  int      `json:"files_deleted"`
	BytesUploaded int64    `json:"bytes_uploaded"`
	URL           string   `json:"url,omitempty"`
	Paths         []string `json:"paths,omitempty"`
}

// StaticSiteObject describes one published file. A listing carries no content;
// use StaticSitesClient.Fetch for the bytes.
type StaticSiteObject struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type,omitempty"`
	LastModified time.Time `json:"last_modified"`
}

// StaticSiteFileList is the result of listing a site's files.
type StaticSiteFileList struct {
	Files      []StaticSiteObject `json:"files"`
	TotalBytes int64              `json:"total_bytes"`
	Truncated  bool               `json:"truncated"`
}

// FetchStaticSiteFilesInput names the files to download inline.
type FetchStaticSiteFilesInput struct {
	Paths []string `json:"paths"`
}

// StaticSiteFetchResult carries file contents back in the same shape Publish
// accepts, so a fetched file can be edited and published back unchanged.
type StaticSiteFetchResult struct {
	Files []StaticSiteFile `json:"files"`
}

// UpdateStaticSiteDocumentsInput sets the index and error documents.
type UpdateStaticSiteDocumentsInput struct {
	IndexDocument string `json:"index_document,omitempty"`
	ErrorDocument string `json:"error_document,omitempty"`
}

// SetStaticSiteDomainInput attaches a custom domain; an empty value detaches
// the current one.
type SetStaticSiteDomainInput struct {
	CustomDomain string `json:"custom_domain"`
}

// S3BucketInput configures an S3-compatible bucket service on creation.
type S3BucketInput struct {
	Name   string `json:"name,omitempty"`
	Region string `json:"region,omitempty"`
}

// DockerInput configures a Docker service on creation.
type DockerInput struct {
	Image          string        `json:"image"`
	Tag            string        `json:"tag,omitempty"`
	ReadinessProbe *ServiceProbe `json:"readiness_probe,omitempty"`
}

// ServiceProbe configures an HTTP health probe for a Docker service.
type ServiceProbe struct {
	Path                string `json:"path"`
	Port                uint   `json:"port"`
	InitialDelaySeconds int32  `json:"initial_delay_seconds"`
	PeriodSeconds       int32  `json:"period_seconds"`
	TimeoutSeconds      int32  `json:"timeout_seconds"`
	FailureThreshold    int32  `json:"failure_threshold"`
	SuccessThreshold    int32  `json:"success_threshold"`
}

// PostgresInput configures a PostgreSQL service on creation.
type PostgresInput struct {
	StorageGB uint64 `json:"storage_gb,omitempty"`
	// Mode is one of "standalone" or "replication".
	Mode string `json:"mode,omitempty"`
}

// RedisInput configures a Redis service on creation.
type RedisInput struct {
	StorageGB uint64 `json:"storage_gb,omitempty"`
	// Mode is one of "standalone", "replication", or "cluster".
	Mode     string `json:"mode,omitempty"`
	Replicas int    `json:"replicas,omitempty"`
}

// UpdateServiceInput is the request body for patching a service.
// Set Action to what is changing: "name", "vcpus", "replicas", "memory", "image",
// "start_command", "registry_credentials", "*_probe", or "delete_*_probe".
type UpdateServiceInput struct {
	Action           string        `json:"action"`
	Name             string        `json:"name,omitempty"`
	VCPUs            uint          `json:"vcpus,omitempty"`
	Replicas         uint          `json:"replicas,omitempty"`
	Memory           uint          `json:"memory,omitempty"`
	Image            string        `json:"image,omitempty"`
	Tag              string        `json:"tag,omitempty"`
	RegistryUsername string        `json:"registry_username,omitempty"`
	RegistryPassword string        `json:"registry_password,omitempty"`
	StartCommand     string        `json:"start_command,omitempty"`
	StartCommandArgs []string      `json:"start_command_args,omitempty"`
	Probe            *ServiceProbe `json:"probe,omitempty"`
}

// AddIngressInput is the request body for adding an ingress port.
type AddIngressInput struct {
	// Protocol is one of "HTTP", "gRPC", or "TCP".
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	CustomFQDN string `json:"custom_fqdn,omitempty"`
	// AllowedSourceRanges restricts which client IPs/CIDRs may connect
	// (TCP/UDP only). Bare IPs are treated as /32. Empty means open to all.
	AllowedSourceRanges []string `json:"allowed_source_ranges,omitempty"`
}

// CreateConfigInput is the request body for creating a config file mount.
type CreateConfigInput struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	MountPath string `json:"mount_path"`
}

// UpdateConfigInput is the request body for updating a config file mount.
type UpdateConfigInput struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	MountPath string `json:"mount_path"`
}

// ── Deployment ────────────────────────────────────────────────────────────────

// DeployOptions controls behaviour of a Deploy or Redeploy call.
type DeployOptions struct {
	// AutoApproveChangeSets, when true, allows deploying even when the service
	// has pending changesets. When false (the default), a deploy or redeploy
	// is rejected if any pending changesets exist.
	AutoApproveChangeSets bool
}

// DeploymentStatus is the lifecycle state of a single deployment.
type DeploymentStatus string

const (
	DeploymentStatusPending  DeploymentStatus = "pending"
	DeploymentStatusStarting DeploymentStatus = "starting"
	DeploymentStatusRunning  DeploymentStatus = "running"
	DeploymentStatusFailed   DeploymentStatus = "failed"
	DeploymentStatusStopped  DeploymentStatus = "stopped"
	DeploymentStatusSleeping DeploymentStatus = "sleeping"
)

// Deployment is a single roll-out of a service.
type Deployment struct {
	Slug       string           `json:"slug"`
	Status     DeploymentStatus `json:"status"`
	Active     bool             `json:"active"`
	CreatedAt  time.Time        `json:"created_at"`
	DeployedAt time.Time        `json:"deployed_at,omitempty"`
}

// BackupRun describes a CloudNativePG base-backup operation.
type BackupRun struct {
	Name      string     `json:"name"`
	BackupID  string     `json:"backup_id,omitempty"`
	Method    string     `json:"method,omitempty"`
	Phase     string     `json:"phase,omitempty"`
	Error     string     `json:"error,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
}

type listDeploymentsResponse struct {
	Active      Deployment   `json:"active"`
	Deployments []Deployment `json:"deployments"`
}

// WorkspaceStats holds resource count summaries for a workspace.
type WorkspaceStats struct {
	Services    int `json:"services"`
	Deployments int `json:"deployments"`
	Members     int `json:"members"`
}

// UsageCosts is a cost breakdown in centiKobo (1/10000 Naira).
type UsageCosts struct {
	TotalCPUCost      int `json:"total_cpu_cost"`
	TotalMemoryCost   int `json:"total_memory_cost"`
	TotalStorageCost  int `json:"total_storage_cost"`
	TotalDataCost     int `json:"total_data_cost"`
	TotalZeroDataCost int `json:"total_zerodata_cost"`
	TotalNetworkCost  int `json:"total_network_cost"`
	TotalCost         int `json:"total_cost"`
}

// WorkspaceUsage is the current-month billing summary for a workspace.
type WorkspaceUsage struct {
	CurrentUsage                  UsageCosts `json:"current_usage"`
	EstimatedUsage                UsageCosts `json:"estimated_usage"`
	EstimatedMonthlyBurn          int64      `json:"estimated_monthly_burn"` // centiKobo
	DaysOfRunwayLeft              int        `json:"days_of_runway_left"`    // -1 means unknown
	WalletBalance                 int64      `json:"wallet_balance"`         // centiKobo
	BillingSuspended              bool       `json:"billing_suspended"`
	BillingNegativeBalanceAllowed bool       `json:"billing_negative_balance_allowed"`
	Period                        string     `json:"period"`
}

// Transaction is a wallet transaction (funding or billing charge).
type Transaction struct {
	Slug              string    `json:"slug"`
	Reference         string    `json:"reference"`
	ProviderReference string    `json:"provider_reference"`
	Status            string    `json:"status"`
	Type              string    `json:"type"`
	Amount            int64     `json:"amount"` // smallest currency unit
	Currency          string    `json:"currency"`
	Processor         string    `json:"processor"`
	CreatedAt         time.Time `json:"created_at"`
}

type fundWorkspaceRequest struct {
	Method string `json:"method"` // "paystack", "stripe", or "bank_transfer"
	Amount int64  `json:"amount"` // smallest currency unit (kobo or cents)
}

// ── Workspace members inputs ──────────────────────────────────────────────────

type addMembersRequest struct {
	Emails []string `json:"emails"`
	Role   string   `json:"role,omitempty"` // "owner", "developer" (default), or "billing"
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

// ── Variable inputs ───────────────────────────────────────────────────────────

type setVariableRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bulkSetVariablesRequest struct {
	Variables []setVariableRequest `json:"variables"`
}

// ── Shell ─────────────────────────────────────────────────────────────────────

// TerminalSize describes the dimensions of a terminal window.
type TerminalSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// shellResizeMsg is the JSON frame sent to the server on terminal resize.
type shellResizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// ── Token inputs ──────────────────────────────────────────────────────────────

type createTokenRequest struct {
	Name         string `json:"name"`
	Env          string `json:"env,omitempty"`
	RegistryPush bool   `json:"registry_push,omitempty"`
}

// CreateTokenOptions describes a token to mint. RegistryPush is an explicit
// opt-in: without it the token cannot read registry credentials and so cannot
// push images.
type CreateTokenOptions struct {
	Name         string
	Env          string
	RegistryPush bool
}

// ── Workspace / Project / Env create inputs ───────────────────────────────────

type createNameRequest struct {
	Name string `json:"name"`
}
