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
	ServiceTypeMySQL         ServiceType = "mysql"
	// ServiceTypeVideo is a video library: uploads are transcoded into an
	// adaptive ladder and served from the platform's zero-rated address, so
	// watching costs the viewer no data on supported networks.
	ServiceTypeVideo ServiceType = "video"
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
	MySQL        *MySQLConfig        `json:"mysql_svc,omitempty"`
	HTTPGateway  *HTTPGatewayConfig  `json:"http_gateway_svc,omitempty"`
	IPsecGateway *IPsecGatewayConfig `json:"ipsec_gateway_svc,omitempty"`

	Variables           []Variable           `json:"variables,omitempty"`
	Ingress             []IngressPort        `json:"ingress_ports,omitempty"`
	Configs             []ServiceConfig      `json:"configs,omitempty"`
	Volumes             []ServiceVolume      `json:"persistent_storages,omitempty"`
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
	Image string `json:"image"`
	Tag   string `json:"tag"`
	// StorageGB is the provisioned volume size. Mode has no counterpart here:
	// the API accepts it on creation but never reports it back.
	StorageGB      uint                   `json:"storage_gb,omitempty"`
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
	Mode      string `json:"mode"`
	Replicas  int    `json:"replicas"`
	StorageGB uint   `json:"storage_gb,omitempty"`
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

// MySQLConfig holds configuration for a managed MySQL service. MySQL runs as an
// InnoDB Cluster: Instances server pods fronted by RouterInstances stateless
// router pods. Clients connect through the router, never to a server directly.
type MySQLConfig struct {
	StorageGB uint `json:"storage_gb,omitempty"`
	// Instances is the number of MySQL server pods. One is a valid single-member
	// cluster with no high availability.
	Instances int `json:"instances"`
	// RouterInstances is the number of router pods. They are ordinary pods and
	// are billed as such, so a 3-server cluster with 1 router is 4 billed pods.
	RouterInstances int    `json:"router_instances"`
	Version         string `json:"version"`

	ConnectionInfo *MySQLConnectionInfo `json:"connection_info,omitempty"`

	// Backup is the scheduled-backup configuration, nil when backups are off.
	Backup *MySQLBackupConfig `json:"backup,omitempty"`

	// Restore records the dump this service was seeded from, nil if it was not.
	Restore *MySQLRestoreConfig `json:"restore,omitempty"`
}

// MySQLBackupFrequency is how often a scheduled backup runs. The platform
// offers presets rather than cron expressions; times are UTC.
type MySQLBackupFrequency string

const (
	// MySQLBackupDaily runs at 02:00 UTC.
	MySQLBackupDaily MySQLBackupFrequency = "daily"
	// MySQLBackupTwiceDaily runs at 02:00 and 14:00 UTC.
	MySQLBackupTwiceDaily MySQLBackupFrequency = "twice_daily"
	// MySQLBackupWeekly runs on Sunday at 02:00 UTC.
	MySQLBackupWeekly MySQLBackupFrequency = "weekly"
)

// MySQLBackupInput configures scheduled backups for a MySQL service.
//
// Backups are logical dumps taken on a schedule, not continuous archiving:
// recovery points are the dumps that have run, and there is no point-in-time
// recovery. Restoring creates a new service rather than rewinding this one.
//
// Give exactly one destination — either BucketSvcSlug or an explicit bucket
// with credentials. An empty Frequency turns backups off and clears the stored
// credentials; dumps already written are left alone.
type MySQLBackupInput struct {
	Frequency MySQLBackupFrequency `json:"frequency"`

	// BucketSvcSlug names a Simplifyd bucket service to write to. It must be in
	// the same project and environment as the database. Its credentials are read
	// at each run, so rotating the bucket's keys does not break backups.
	BucketSvcSlug string `json:"bucket_svc_slug,omitempty"`

	// Explicit S3-compatible destination, used when BucketSvcSlug is empty.
	BucketName      string `json:"bucket_name,omitempty"`
	EndpointURL     string `json:"endpoint_url,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// MySQLRestoreInput seeds a new MySQL service from an existing dump.
//
// It can only be given at creation. The operator loads the dump when the
// cluster bootstraps and ignores the field afterwards, so there is no way to
// restore into a database that already exists — recovering means creating a new
// service from a backup and moving traffic to it.
//
// Give exactly one source: either BucketSvcSlug or an explicit bucket with
// credentials.
type MySQLRestoreInput struct {
	// Path is the path to one dump, not the backup root. A schedule writes each
	// dump under its own sub-path; pointing at the root loads nothing and
	// reports success.
	Path string `json:"path"`

	// BucketSvcSlug names a Simplifyd bucket service in the same project and
	// environment. This is the usual case — restoring from the bucket the
	// database was backed up into.
	BucketSvcSlug string `json:"bucket_svc_slug,omitempty"`

	BucketName      string `json:"bucket_name,omitempty"`
	EndpointURL     string `json:"endpoint_url,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// MySQLRestoreConfig reports what a service was seeded from, if anything. It is
// historical: the dump was loaded at bootstrap and changing it does nothing.
// Credentials are never returned.
type MySQLRestoreConfig struct {
	BucketName     string `json:"bucket_name,omitempty"`
	Path           string `json:"path,omitempty"`
	BucketSvcSlug  string `json:"bucket_svc_slug,omitempty"`
	EndpointURL    string `json:"endpoint_url,omitempty"`
	Region         string `json:"region,omitempty"`
	HasCredentials bool   `json:"has_credentials"`
}

// MySQLBackupConfig is the backup configuration reported on a MySQL service.
// Credentials are never returned; HasCredentials reports whether any are stored.
type MySQLBackupConfig struct {
	Frequency       MySQLBackupFrequency `json:"frequency,omitempty"`
	BucketSvcSlug   string               `json:"bucket_svc_slug,omitempty"`
	DestinationPath string               `json:"destination_path,omitempty"`
	EndpointURL     string               `json:"endpoint_url,omitempty"`
	Region          string               `json:"region,omitempty"`
	HasCredentials  bool                 `json:"has_credentials"`
}

// MySQLConnectionInfo carries the credentials for a MySQL service. The URL
// targets the router's read-write port.
type MySQLConnectionInfo struct {
	User          string `json:"user"`
	Password      string `json:"password"`
	ConnectionURL string `json:"connection_url"`
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

// ServiceVolume is a persistent volume attached to a service. Unlike an
// ephemeral storage attachment its contents survive the pod. See VolumesClient
// for the two constraints that come with one.
type ServiceVolume struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SizeGB    int    `json:"size_gb"`
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
	MySQL      *MySQLInput      `json:"mysql_svc,omitempty"`
	// IPsecGateway configures a VPN gateway. A gateway is billed per tunnel and
	// has a fixed footprint, so VCPUs and Memory do not apply to it.
	IPsecGateway *IPsecGatewayInput `json:"ipsec_gateway_svc,omitempty"`
	Video        *VideoInput        `json:"video_svc,omitempty"`
}

// VideoInput configures a video library on creation. Like a static site it runs
// no container, so image, resources and replicas do not apply — the work is
// done by encoder jobs, billed per source minute of what is uploaded.
type VideoInput struct {
	Name string `json:"name,omitempty"`
	// MaxHeight caps the encoding ladder. Left at zero it defaults to 720p,
	// which is the right ceiling for the devices and networks this serves:
	// 1080p nearly doubles the stored bytes per video for a rung almost nobody
	// on a cheap handset will select.
	MaxHeight int `json:"max_height,omitempty"`
	// KeepOriginal retains the uploaded master, at roughly double the storage,
	// so a video can be encoded again — or given a new poster frame — without
	// being uploaded a second time. Defaults to true.
	KeepOriginal *bool `json:"keep_original,omitempty"`
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

// MySQLInput configures a MySQL service on creation.
type MySQLInput struct {
	StorageGB uint64 `json:"storage_gb,omitempty"`
	// Instances is the number of MySQL server pods. Group Replication needs a
	// majority to accept writes, so an even count costs a pod without buying
	// failure tolerance — use 1, 3, 5, 7 or 9.
	Instances int `json:"instances,omitempty"`
	// RouterInstances is the number of router pods, defaulting to 1. Each is a
	// billed pod.
	RouterInstances int    `json:"router_instances,omitempty"`
	Version         string `json:"version,omitempty"`

	// Restore seeds the new database from an existing dump. Creation-time only.
	Restore *MySQLRestoreInput `json:"restore,omitempty"`
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

// CreateVolumeInput is the request body for attaching a persistent volume.
type CreateVolumeInput struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SizeGB    int    `json:"size_gb"`
}

// UpdateVolumeInput is the request body for changing an attached volume. The
// size may only be raised; see VolumesClient.Update.
type UpdateVolumeInput struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SizeGB    int    `json:"size_gb"`
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

// ── Video libraries ───────────────────────────────────────────────────────────

// VideoLibrary is a video service: two buckets, a playback host, and the
// settings that govern what the encoder produces.
type VideoLibrary struct {
	DisplayName string `json:"display_name"`
	// SourceBucket holds the originals and is never served anonymously.
	SourceBucket string `json:"source_bucket"`
	// HLSBucket holds the encoder's output and is anonymously readable, which
	// is what lets a player fetch it with no credentials.
	HLSBucket string `json:"hls_bucket"`
	Status    string `json:"status"`
	// PlaybackURL is the origin every embed and playlist URL is built from. It
	// resolves to the zero-rated ingress address, which is what makes watching
	// free to the viewer.
	PlaybackURL string `json:"playback_url,omitempty"`
	// PlaybackDomain is served once its DNS points at DomainCNAMETarget.
	PlaybackDomain    string `json:"playback_domain,omitempty"`
	DomainStatus      string `json:"domain_status,omitempty"`
	DomainCNAMETarget string `json:"domain_cname_target,omitempty"`
	// MaxHeight is the tallest rung the ladder may produce, 720 by default.
	MaxHeight int `json:"max_height"`
	// KeepOriginal decides whether a video can be re-encoded, or its poster
	// changed, without being uploaded again.
	KeepOriginal bool      `json:"keep_original"`
	VideoCount   int64     `json:"video_count"`
	BytesUsed    int64     `json:"bytes_used"`
	CreatedAt    time.Time `json:"created_at"`
}

// UpdateVideoLibraryInput changes a library's encoding settings.
type UpdateVideoLibraryInput struct {
	MaxHeight    int  `json:"max_height"`
	KeepOriginal bool `json:"keep_original"`
}

// SetVideoPlaybackDomainInput attaches a custom playback hostname. An empty
// domain detaches the current one.
type SetVideoPlaybackDomainInput struct {
	PlaybackDomain string `json:"playback_domain"`
}

// Video is one uploaded file and everything derived from it.
type Video struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Status is uploading, queued, processing, ready or failed.
	Status string `json:"status"`
	// StatusMessage carries the encoder's own words when something failed.
	StatusMessage    string           `json:"status_message,omitempty"`
	DurationMS       int64            `json:"duration_ms"`
	Width            int              `json:"width"`
	Height           int              `json:"height"`
	SourceBytes      int64            `json:"source_bytes"`
	HLSBytes         int64            `json:"hls_bytes"`
	PosterURL        string           `json:"poster_url,omitempty"`
	PosterOffsetMS   int64            `json:"poster_offset_ms,omitempty"`
	StoryboardURL    string           `json:"storyboard_url,omitempty"`
	StoryboardVTTURL string           `json:"storyboard_vtt_url,omitempty"`
	PlaybackURL      string           `json:"playback_url,omitempty"`
	EmbedURL         string           `json:"embed_url,omitempty"`
	IframeSnippet    string           `json:"iframe_snippet,omitempty"`
	ScriptSnippet    string           `json:"script_snippet,omitempty"`
	Renditions       []VideoRendition `json:"renditions,omitempty"`
	Tracks           []VideoTrack     `json:"tracks,omitempty"`
	// Progress is the current encode's percentage, 0 when nothing is running.
	Progress int `json:"progress_pct"`
	// HasSource reports whether the original is still retained, which decides
	// whether the video can be re-encoded or given a new poster.
	HasSource   bool       `json:"has_source"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// VideoList is a page of a library's videos.
type VideoList struct {
	Videos []Video `json:"videos"`
	Total  int64   `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// VideoRendition is one rung the encoder actually produced.
type VideoRendition struct {
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	VideoKbps int    `json:"video_kbps"`
	AudioKbps int    `json:"audio_kbps"`
	Bytes     int64  `json:"bytes"`
	// MegabytesPerHour is what an hour at this rung costs the viewer, which on
	// a metered connection is more useful than the pixel height.
	MegabytesPerHour int `json:"megabytes_per_hour"`
}

// VideoTrack is a caption or subtitle file attached to a video.
type VideoTrack struct {
	Slug      string `json:"slug"`
	Kind      string `json:"kind"`
	Language  string `json:"language"`
	Label     string `json:"label"`
	URL       string `json:"url,omitempty"`
	IsDefault bool   `json:"is_default"`
}

// RegisterVideoUploadInput begins an upload. The size is required up front so
// the file can be rejected before any bytes move if it is over the limit.
type RegisterVideoUploadInput struct {
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// VideoUploadPlan is what a client needs to put a file into the source bucket
// without the bytes passing through the API.
type VideoUploadPlan struct {
	VideoSlug string            `json:"video_slug"`
	UploadID  string            `json:"upload_id"`
	Key       string            `json:"key"`
	PartSize  int64             `json:"part_size"`
	PartCount int               `json:"part_count"`
	Parts     []VideoUploadPart `json:"parts"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// VideoUploadPart is one presigned PUT. PartNumber is 1-based.
type VideoUploadPart struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

// PresignVideoPartsInput asks for a run of part URLs, inclusive at both ends.
type PresignVideoPartsInput struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// VideoUploadedPart is the ETag the store returned for one part. All of them,
// in order, are needed to assemble the object.
type VideoUploadedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

// CompleteVideoUploadInput finishes an upload and queues encoding.
type CompleteVideoUploadInput struct {
	Parts []VideoUploadedPart `json:"parts"`
}

// UpdateVideoInput changes what a video is called.
type UpdateVideoInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SetVideoPosterInput names the moment to take the poster frame from.
type SetVideoPosterInput struct {
	PosterOffsetMS int64 `json:"poster_offset_ms"`
}

// AddVideoTrackInput uploads a WebVTT caption file inline.
type AddVideoTrackInput struct {
	Language string `json:"language"`
	Label    string `json:"label"`
	// Content is the WebVTT file, base64-encoded.
	Content string `json:"content"`
}

// VideoAnalytics is the whole answer to "how is this video doing".
type VideoAnalytics struct {
	VideoSlug string `json:"video_slug,omitempty"`
	From      string `json:"from"`
	To        string `json:"to"`

	Totals VideoAnalyticsTotals `json:"totals"`
	// Daily is the series with no gaps: a day with no views is a zero row
	// rather than a missing one.
	Daily []VideoDailyStat `json:"daily"`
	// Retention is the drop-off curve, 100 points.
	Retention []VideoRetentionPoint `json:"retention,omitempty"`
	// Carriers is the dimension no other video host can report: what share of
	// watch time was free to the people who watched it, by carrier.
	Carriers  []VideoCarrierStat  `json:"carriers,omitempty"`
	Rungs     []VideoRungStat     `json:"rungs,omitempty"`
	Referrers []VideoReferrerStat `json:"referrers,omitempty"`
	TopVideos []VideoTopStat      `json:"top_videos,omitempty"`
}

// VideoAnalyticsTotals is the headline row.
type VideoAnalyticsTotals struct {
	Views         int64 `json:"views"`
	UniqueViewers int64 `json:"unique_viewers"`
	WatchSeconds  int64 `json:"watch_seconds"`
	Completions   int64 `json:"completions"`
	Stalls        int64 `json:"stalls"`
	StallSeconds  int64 `json:"stall_seconds"`
	Errors        int64 `json:"errors"`
	// EstimatedBytes is rung bitrate x watch time, not a socket measurement.
	EstimatedBytes        int64   `json:"estimated_bytes"`
	ZeroRatedWatchSeconds int64   `json:"zero_rated_watch_seconds"`
	ZeroRatedSharePct     float64 `json:"zero_rated_share_pct"`
	CompletionRatePct     float64 `json:"completion_rate_pct"`
	// StallRatePct is stalls per hundred views.
	StallRatePct        float64 `json:"stall_rate_pct"`
	AverageWatchSeconds float64 `json:"average_watch_seconds"`
}

// VideoDailyStat is one day of the series.
type VideoDailyStat struct {
	Day                   string `json:"day"` // YYYY-MM-DD
	Views                 int64  `json:"views"`
	UniqueViewers         int64  `json:"unique_viewers"`
	WatchSeconds          int64  `json:"watch_seconds"`
	Completions           int64  `json:"completions"`
	Stalls                int64  `json:"stalls"`
	StallSeconds          int64  `json:"stall_seconds"`
	Errors                int64  `json:"errors"`
	EstimatedBytes        int64  `json:"estimated_bytes"`
	ZeroRatedWatchSeconds int64  `json:"zero_rated_watch_seconds"`
}

// VideoRetentionPoint is one percentage point of the runtime.
type VideoRetentionPoint struct {
	Pct     int     `json:"pct"`
	Viewers int64   `json:"viewers"`
	Share   float64 `json:"share_pct"`
}

// VideoCarrierStat is one carrier's slice of the audience.
type VideoCarrierStat struct {
	Tier          string `json:"tier"`
	Name          string `json:"name,omitempty"`
	ZeroRated     bool   `json:"zero_rated"`
	Views         int64  `json:"views"`
	UniqueViewers int64  `json:"unique_viewers"`
	WatchSeconds  int64  `json:"watch_seconds"`
	Stalls        int64  `json:"stalls"`
	StallSeconds  int64  `json:"stall_seconds"`
	// StallRatePct is stalls per hundred views on this carrier — the figure
	// that says whether the ladder is tuned for the networks people are on.
	StallRatePct   float64 `json:"stall_rate_pct"`
	EstimatedBytes int64   `json:"estimated_bytes"`
	SharePct       float64 `json:"share_pct"`
}

// VideoRungStat is how much watch time each rung carried.
type VideoRungStat struct {
	Rung             string  `json:"rung"`
	WatchSeconds     int64   `json:"watch_seconds"`
	SharePct         float64 `json:"share_pct"`
	MegabytesPerHour int     `json:"megabytes_per_hour,omitempty"`
}

// VideoReferrerStat is one embedding site.
type VideoReferrerStat struct {
	Host  string `json:"host"`
	Views int64  `json:"views"`
}

// VideoTopStat is one video in a library-wide ranking.
type VideoTopStat struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	PosterURL     string `json:"poster_url,omitempty"`
	Views         int64  `json:"views"`
	UniqueViewers int64  `json:"unique_viewers"`
	WatchSeconds  int64  `json:"watch_seconds"`
}
