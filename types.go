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
	ServiceTypeKafka         ServiceType = "kafka"
	ServiceTypeIPsecGateway  ServiceType = "ipsec_gateway"
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

	Docker       *DockerConfig       `json:"docker_image_svc,omitempty"`
	Postgres     *PostgresConfig     `json:"postgres_svc,omitempty"`
	Redis        *RedisConfig        `json:"redis_svc,omitempty"`
	Kafka        *KafkaConfig        `json:"kafka_svc,omitempty"`
	HTTPGateway  *HTTPGatewayConfig  `json:"http_gateway_svc,omitempty"`
	IPsecGateway *IPsecGatewayConfig `json:"ipsec_gateway_svc,omitempty"`

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

// PostgresDatabase is a database on a managed PostgreSQL service, alongside the
// default "app" database that always exists and is not listed.
type PostgresDatabase struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
	// Extensions are the extensions declared in this database. An entry with
	// Ensure "absent" is a pending removal, not an installed extension.
	Extensions []PostgresExtension `json:"extensions,omitempty"`
}

// PostgresExtension is an extension managed inside a database.
type PostgresExtension struct {
	Name string `json:"name"`
	// Ensure is "present" or "absent". An absent entry is retained so the
	// platform can finish dropping the extension.
	Ensure string `json:"ensure"`
}

// PostgresExtensions is the extension state of one database, together with the
// platform allowlist of extensions that may be installed.
type PostgresExtensions struct {
	Extensions []PostgresExtension `json:"extensions"`
	Supported  []string            `json:"supported,omitempty"`
}

// Installed returns the names of the extensions actually installed, excluding
// the "absent" entries that represent pending removals.
func (p *PostgresExtensions) Installed() []string {
	names := make([]string, 0, len(p.Extensions))
	for _, e := range p.Extensions {
		if e.Ensure == PostgresExtensionAbsent {
			continue
		}
		names = append(names, e.Name)
	}
	return names
}

// Ensure values for PostgresExtension.
const (
	PostgresExtensionPresent = "present"
	PostgresExtensionAbsent  = "absent"
)

// CreatePostgresDatabaseInput creates a database on a Postgres service.
type CreatePostgresDatabaseInput struct {
	Name string `json:"name"`
	// Owner is the role that owns the database, granted ALL privileges on it.
	// Defaults to "app" when empty.
	Owner string `json:"owner,omitempty"`
}

// SetPostgresExtensionsInput replaces the extensions declared on a database.
type SetPostgresExtensionsInput struct {
	Extensions []string `json:"extensions"`
}

// PostgresUser is an additional login role on a managed PostgreSQL service.
type PostgresUser struct {
	Username string `json:"username"`
	// Password is returned only by Create and RotatePassword — it is a one-time
	// reveal and cannot be read back afterwards.
	Password      string `json:"password,omitempty"`
	ConnectionURL string `json:"conn_url,omitempty"`
	// Replication reports whether the role may create replication slots and
	// stream WAL, as an external logical-replication subscriber needs.
	Replication bool `json:"replication"`
	// InRoles are the role memberships the platform enforces declaratively:
	// memberships granted manually via SQL but absent here are revoked on the
	// next reconcile.
	InRoles []string `json:"in_roles,omitempty"`
}

// CreatePostgresUserInput creates a login role on a Postgres service.
type CreatePostgresUserInput struct {
	Username    string   `json:"username"`
	Replication bool     `json:"replication,omitempty"`
	InRoles     []string `json:"in_roles,omitempty"`
}

// UpdatePostgresUserRolesInput replaces a user's role memberships. The list is
// authoritative — omitted memberships are revoked.
type UpdatePostgresUserRolesInput struct {
	InRoles []string `json:"in_roles"`
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

// KafkaConfig holds configuration for a managed Kafka service. Kafka runs in
// KRaft mode: a "standalone" service is one node carrying both roles, while a
// "cluster" splits brokers and controllers into separate pools.
type KafkaConfig struct {
	StorageGB uint `json:"storage_gb,omitempty"`
	// Mode is one of "standalone" or "cluster".
	Mode string `json:"mode"`
	// Brokers and Controllers are the node counts of each pool. Both are 1 in
	// standalone mode, where a single node carries both roles.
	Brokers     int    `json:"brokers"`
	Controllers int    `json:"controllers"`
	Version     string `json:"version"`
}

// HTTPGatewayConfig holds configuration for an HTTP gateway service. Its routes
// are managed with the GatewayRoutes sub-client, not by replacing this struct.
type HTTPGatewayConfig struct {
	Routes []GatewayRoute `json:"routes,omitempty"`
}

// GatewayRoute forwards requests matching a path prefix to a backend service in
// the same environment.
type GatewayRoute struct {
	Slug       string `json:"slug"`
	PathPrefix string `json:"path_prefix"`
	// BackendSlug is the slug of the target service in the same environment.
	BackendSlug string `json:"backend_slug"`
	BackendPort uint   `json:"backend_port"`
	// StripPrefix removes PathPrefix from the request path before forwarding.
	StripPrefix bool `json:"strip_prefix"`
	// Priority orders overlapping prefixes; higher wins.
	Priority int `json:"priority"`
}

// GatewayRouteInput is the request body for creating or updating a route.
type GatewayRouteInput struct {
	PathPrefix  string `json:"path_prefix"`
	BackendSlug string `json:"backend_slug"`
	BackendPort uint   `json:"backend_port"`
	StripPrefix bool   `json:"strip_prefix"`
	Priority    int    `json:"priority"`
}

// IPsecGatewayConfig holds configuration for a site-to-site VPN gateway. The
// gateway terminates IKEv2 on a fixed public address, because counterparties
// pin the peer address in firewall rules that outlive any pod.
type IPsecGatewayConfig struct {
	PublicIPSlug string `json:"public_ip_slug,omitempty"`
	PublicIP     string `json:"public_ip,omitempty"`
	// LocalSubnets are the ranges this environment presents to counterparties.
	// Empty on creation seeds it with the gateway's own address.
	LocalSubnets []string `json:"local_subnets,omitempty"`
	Image        string   `json:"image,omitempty"`
	// VNI is the gateway's overlay identifier, allocated by the platform and
	// fixed for the gateway's life.
	VNI         int32             `json:"vni,omitempty"`
	Connections []IPsecConnection `json:"connections,omitempty"`
}

// IPsecConnection is one tunnel to a counterparty.
//
// The pre-shared key is write-only: it is stored encrypted and never read back,
// so this struct reports only whether one is set.
type IPsecConnection struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	RemoteGateway string   `json:"remote_gateway"`
	RemoteSubnets []string `json:"remote_subnets"`
	// LocalSubnets narrows the gateway's set for this connection. Empty inherits.
	LocalSubnets []string  `json:"local_subnets,omitempty"`
	LocalID      string    `json:"local_id,omitempty"`
	RemoteID     string    `json:"remote_id,omitempty"`
	IKEProposal  string    `json:"ike_proposal,omitempty"`
	ESPProposal  string    `json:"esp_proposal,omitempty"`
	IKELifetime  string    `json:"ike_lifetime,omitempty"`
	Lifetime     string    `json:"lifetime,omitempty"`
	StartAction  string    `json:"start_action,omitempty"`
	HasPSK       bool      `json:"has_psk"`
	CreatedAt    time.Time `json:"created_at"`
}

// IPsecConnectionInput is the request body for creating or updating a tunnel.
// PSK is accepted here on creation only; changing an existing key is a separate
// call (RotatePSK), so a settings edit cannot blank a key by omitting it.
type IPsecConnectionInput struct {
	Name          string   `json:"name"`
	RemoteGateway string   `json:"remote_gateway"`
	RemoteSubnets []string `json:"remote_subnets"`
	LocalSubnets  []string `json:"local_subnets,omitempty"`
	LocalID       string   `json:"local_id,omitempty"`
	RemoteID      string   `json:"remote_id,omitempty"`
	PSK           string   `json:"psk,omitempty"`
	IKEProposal   string   `json:"ike_proposal,omitempty"`
	ESPProposal   string   `json:"esp_proposal,omitempty"`
	IKELifetime   string   `json:"ike_lifetime,omitempty"`
	Lifetime      string   `json:"lifetime,omitempty"`
	StartAction   string   `json:"start_action,omitempty"`
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
	Kafka      *KafkaInput      `json:"kafka_svc,omitempty"`
	// IPsecGateway configures a VPN gateway. A gateway is billed per tunnel and
	// has a fixed footprint, so VCPUs and Memory do not apply to it.
	IPsecGateway *IPsecGatewayInput `json:"ipsec_gateway_svc,omitempty"`
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

// KafkaInput configures a Kafka service on creation.
type KafkaInput struct {
	StorageGB uint64 `json:"storage_gb,omitempty"`
	// Mode is one of "standalone" or "cluster". Standalone pins both node
	// counts to 1; the platform ignores whatever is sent for them.
	Mode string `json:"mode,omitempty"`
	// Brokers and Controllers apply to cluster mode only. Controllers should be
	// odd so the KRaft quorum can tolerate a failure.
	Brokers     int    `json:"brokers,omitempty"`
	Controllers int    `json:"controllers,omitempty"`
	Version     string `json:"version,omitempty"`
}

// IPsecGatewayInput configures a VPN gateway on creation. The strongSwan image
// is platform-supplied and deliberately not accepted from the client.
type IPsecGatewayInput struct {
	// LocalSubnets is optional; empty seeds it with the gateway's own allocated
	// address.
	LocalSubnets []string `json:"local_subnets,omitempty"`
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
	Name string `json:"name"`
	Env  string `json:"env,omitempty"`
}

// CreateTokenOptions describes a token to mint.
type CreateTokenOptions struct {
	Name string
	Env  string
}

// ── Workspace / Project / Env create inputs ───────────────────────────────────

type createNameRequest struct {
	Name string `json:"name"`
}

// ── Registry retention ────────────────────────────────────────────────────────

// Retention policy kinds. Rules are unioned: a tag survives if any matching
// rule keeps it.
const (
	// RetentionKeepRecentN keeps the N most recently pushed images.
	RetentionKeepRecentN = "keep_recent_n"
	// RetentionKeepPushedWithin keeps images pushed in the last N days.
	RetentionKeepPushedWithin = "keep_pushed_within"
	// RetentionKeepPulledWithin keeps images pulled in the last N days. An
	// image that was never pulled is not kept by this rule.
	RetentionKeepPulledWithin = "keep_pulled_within"
)

// RetentionRule is one clause of a retention policy. Patterns are globs, where
// "**" also crosses "/" so it can cover nested repository paths.
type RetentionRule struct {
	RepositoryPattern string   `json:"repository_pattern"`
	TagPattern        string   `json:"tag_pattern"`
	TagExclude        []string `json:"tag_exclude,omitempty"`
	PolicyKind        string   `json:"policy_kind"`
	Threshold         int      `json:"threshold"`
}

// RetentionPolicy is a workspace registry's retention configuration. It is
// disabled by default and deletes nothing until it is enabled.
type RetentionPolicy struct {
	Enabled bool `json:"enabled"`
	// ProtectMovingTags shields tags like "latest" and "main" from deletion.
	ProtectMovingTags bool            `json:"protect_moving_tags"`
	Rules             []RetentionRule `json:"rules"`
}

// RetentionRunItem is one tag's outcome in a retention run. Action is one of
// "deleted", "would_delete", "skipped_in_use" or "failed"; an image a service
// is deployed from is always skipped, whatever the rules say.
type RetentionRunItem struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"size_bytes"`
	Action     string `json:"action"`
	Reason     string `json:"reason"`
}

// RetentionRun is one execution of a retention policy, or a preview of one.
type RetentionRun struct {
	Slug             string     `json:"slug"`
	Trigger          string     `json:"trigger"`
	DryRun           bool       `json:"dry_run"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	Status           string     `json:"status"`
	Error            string     `json:"error,omitempty"`
	ArtifactsDeleted int        `json:"artifacts_deleted"`
	TagsDeleted      int        `json:"tags_deleted"`
	// BytesFreedEstimate sums whole-image sizes, so shared layers are counted
	// once per image and the figure overstates the real saving.
	BytesFreedEstimate int64              `json:"bytes_freed_estimate"`
	Items              []RetentionRunItem `json:"items,omitempty"`
}
