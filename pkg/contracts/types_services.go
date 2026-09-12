package contracts

import (
	"time"
)

type ServiceRegistration struct {
	ServiceID     string              `json:"service_id"`
	ServiceType   ServiceType         `json:"service_type"`
	ServiceName   string              `json:"service_name"`
	TransportMode UpdateTransportMode `json:"transport_mode,omitempty"`
	Description   string              `json:"description,omitempty"`
	Host          string              `json:"host,omitempty"`
	Port          int                 `json:"port,omitempty"`
	SSLEnabled    bool                `json:"ssl_enabled"`
	PublicURL     string              `json:"public_url,omitempty"`
	Version       string              `json:"version"`
	Commit        string              `json:"commit,omitempty"`
	BuildDate     string              `json:"build_date,omitempty"`
	Capabilities  map[string]any      `json:"capabilities"`
	Hostname      string              `json:"hostname,omitempty"`
	OS            string              `json:"os,omitempty"`
	Arch          string              `json:"arch,omitempty"`
}

type ServiceToken struct {
	ID          string      `json:"id"`
	ServiceType ServiceType `json:"service_type"`
	Scopes      []string    `json:"scopes"`
	RevokedAt   *time.Time  `json:"revoked_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`

	// Token is returned only once on creation or rotation. List/get responses must omit it.
	Token string `json:"token,omitempty"`
}

type ServiceTokenCreateRequest struct {
	ServiceType  ServiceType    `json:"service_type"`
	Scopes       []ServiceScope `json:"scopes"`
	ServiceID    string         `json:"service_id,omitempty"`
	ServiceName  string         `json:"service_name,omitempty"`
	PublicURL    string         `json:"public_url,omitempty"`
	Version      string         `json:"version,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
}

type ServiceEndpoint struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"ssl_enabled"`
	PublicURL  string `json:"public_url"`
}

type RegisteredService struct {
	ServiceID             string              `json:"service_id"`
	ServiceType           ServiceType         `json:"service_type"`
	ServiceName           string              `json:"service_name"`
	TransportMode         UpdateTransportMode `json:"transport_mode,omitempty"`
	ExecutionHostID       string              `json:"execution_host_id,omitempty"`
	OwnershipEpoch        int64               `json:"ownership_epoch,omitempty"`
	PublicURL             string              `json:"public_url,omitempty"`
	DesiredEndpoint       *ServiceEndpoint    `json:"desired_endpoint,omitempty"`
	AppliedEndpoint       *ServiceEndpoint    `json:"applied_endpoint,omitempty"`
	ReportedEndpoint      *ServiceEndpoint    `json:"reported_endpoint,omitempty"`
	EndpointRevision      int64               `json:"endpoint_revision,omitempty"`
	EndpointStatus        string              `json:"endpoint_status,omitempty"`
	AppliedConfigRevision int64               `json:"applied_config_revision,omitempty"`
	AppliedConfigSHA256   string              `json:"applied_config_sha256,omitempty"`
	Version               string              `json:"version"`
	Status                ServiceStatus       `json:"status"`
	AssignmentRole        string              `json:"assignment_role,omitempty"`
	LastHeartbeatAt       *time.Time          `json:"last_heartbeat_at,omitempty"`
	HealthStatus          string              `json:"health_status,omitempty"`
	HeartbeatStale        bool                `json:"heartbeat_stale,omitempty"`
	HeartbeatAgeSec       *int64              `json:"heartbeat_age_sec,omitempty"`
	CurrentStreamID       string              `json:"current_stream_id,omitempty"`
	Capabilities          map[string]any      `json:"capabilities"`
	ReportedCapabilities  map[string]any      `json:"reported_capabilities,omitempty"`
	TokenID               string              `json:"-"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type StreamServiceAssignment struct {
	StreamID       string      `json:"stream_id"`
	ServiceID      string      `json:"service_id"`
	ServiceType    ServiceType `json:"service_type"`
	AssignmentRole string      `json:"assignment_role"`
	AssignedAt     time.Time   `json:"assigned_at"`
}

type ServiceAssignmentWriteRequest struct {
	StreamID       string `json:"stream_id"`
	AssignmentRole string `json:"assignment_role,omitempty"`
}

type ServiceRuntimeConfig struct {
	Service              RegisteredService             `json:"service"`
	Assignments          []StreamServiceAssignment     `json:"assignments"`
	Profiles             map[ProfileKind][]Profile     `json:"profiles"`
	StreamDiscordConfigs []ServiceRuntimeDiscordConfig `json:"stream_discord_configs,omitempty"`
	StreamArchiveConfigs []ServiceRuntimeArchiveConfig `json:"stream_archive_configs,omitempty"`
	StreamYouTubeConfigs []ServiceRuntimeYouTubeConfig `json:"stream_youtube_configs,omitempty"`
}

type ServiceRuntimeDiscordConfig struct {
	StreamID         string `json:"stream_id"`
	AssignmentRole   string `json:"assignment_role"`
	DiscordConfigID  string `json:"discord_config_id"`
	GuildID          string `json:"guild_id"`
	VoiceChannelID   string `json:"voice_channel_id"`
	TextChannelID    string `json:"text_channel_id,omitempty"`
	AutoStartTrigger string `json:"auto_start_trigger,omitempty"`
}

type ServiceRuntimeArchiveConfig struct {
	StreamID         string         `json:"stream_id"`
	AssignmentRole   string         `json:"assignment_role"`
	ArchiveProfileID string         `json:"archive_profile_id"`
	Ready            bool           `json:"ready"`
	ReadinessCode    string         `json:"readiness_code,omitempty"`
	ReadinessMessage string         `json:"readiness_message,omitempty"`
	ArchiveConfig    map[string]any `json:"archive_config,omitempty"`
}

type ServiceRuntimeYouTubeConfig struct {
	StreamID         string         `json:"stream_id"`
	AssignmentRole   string         `json:"assignment_role"`
	YouTubeOutputID  string         `json:"youtube_output_id"`
	Ready            bool           `json:"ready"`
	ReadinessCode    string         `json:"readiness_code,omitempty"`
	ReadinessMessage string         `json:"readiness_message,omitempty"`
	YouTubeConfig    map[string]any `json:"youtube_config,omitempty"`
	ActiveRuntime    map[string]any `json:"active_runtime,omitempty"`
}

type ServiceRuntimeSecretResolveRequest struct {
	ServiceID        string `json:"service_id"`
	StreamID         string `json:"stream_id,omitempty"`
	ArchiveProfileID string `json:"archive_profile_id,omitempty"`
	SecretName       string `json:"secret_name"`
}

type ServiceRuntimeSecretResolveResponse struct {
	SecretName   string `json:"secret_name"`
	Value        string `json:"value"`
	ExpiresInSec int    `json:"expires_in_sec"`
}

type Heartbeat struct {
	ServiceID       string         `json:"service_id"`
	NodeID          string         `json:"nodeId,omitempty"`
	NodeIDSnake     string         `json:"node_id,omitempty"`
	CurrentStreamID string         `json:"current_stream_id,omitempty"`
	Status          string         `json:"status"`
	Version         string         `json:"version,omitempty"`
	Commit          string         `json:"commit,omitempty"`
	BuildDate       string         `json:"build_date,omitempty"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	Hostname        string         `json:"hostname,omitempty"`
	OS              string         `json:"os,omitempty"`
	Arch            string         `json:"arch,omitempty"`
	API             *NodeAgentAPI  `json:"api,omitempty"`
	Metrics         map[string]any `json:"metrics,omitempty"`
	Timestamp       *time.Time     `json:"timestamp,omitempty"`
}

type NodeAgentAPI struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"sslEnabled"`
}

type ServicePreflightCheck struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ServicePreflightResponse struct {
	CheckedAt time.Time               `json:"checked_at"`
	Ready     bool                    `json:"ready"`
	Checks    []ServicePreflightCheck `json:"checks"`
	Summary   map[string]any          `json:"summary,omitempty"`
}

type ServiceStreamEvent struct {
	ServiceID string         `json:"service_id"`
	StreamID  string         `json:"stream_id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}
