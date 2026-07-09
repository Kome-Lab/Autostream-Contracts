package contracts

import "time"

type StreamStatus string

const (
	StreamCreated   StreamStatus = "created"
	StreamStarting  StreamStatus = "starting"
	StreamLive      StreamStatus = "live"
	StreamStopping  StreamStatus = "stopping"
	StreamCompleted StreamStatus = "completed"
	StreamFailed    StreamStatus = "failed"
)

type ServiceType string

const (
	ServiceDiscordBot      ServiceType = "discord_bot"
	ServiceEncoderRecorder ServiceType = "encoder_recorder"
	ServiceWorker          ServiceType = "worker"
	ServiceObservability   ServiceType = "observability"
)

type ServiceStatus string

const (
	ServiceStatusUnknown          ServiceStatus = "unknown"
	ServiceStatusPending          ServiceStatus = "pending"
	ServiceStatusRegistered       ServiceStatus = "registered"
	ServiceStatusAssigned         ServiceStatus = "assigned"
	ServiceStatusRestartRequested ServiceStatus = "restart_requested"
	ServiceStatusOnline           ServiceStatus = "online"
	ServiceStatusDegraded         ServiceStatus = "degraded"
	ServiceStatusOffline          ServiceStatus = "offline"
)

const (
	AssignmentRolePrimary = "primary"
	AssignmentRoleStandby = "standby"
)

type ServiceScope string

const (
	ScopeServiceRegister      ServiceScope = "service.register"
	ScopeServiceHeartbeat     ServiceScope = "service.heartbeat"
	ScopeServiceLogsWrite     ServiceScope = "service.logs.write"
	ScopeServiceStatusWrite   ServiceScope = "service.status.write"
	ScopeServiceConfigRead    ServiceScope = "service.config.read"
	ScopeServiceSecretResolve ServiceScope = "service.secret.resolve"
	ScopeWorkerEventsWrite    ServiceScope = "worker.events.write"
	ScopeEncoderStatusWrite   ServiceScope = "encoder.status.write"
	ScopeDiscordStatusWrite   ServiceScope = "discord.status.write"
	ScopeObservabilityIngest  ServiceScope = "observability.ingest"
	ScopeRemediationExecute   ServiceScope = "remediation.execute"
)

type ErrorResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type PasskeyCredential struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	CredentialIDHash string   `json:"credential_id_hash"`
	SignCount        uint32   `json:"sign_count"`
	Transports       []string `json:"transports,omitempty"`
	AAGUID           string   `json:"aaguid,omitempty"`
	BackupEligible   bool     `json:"backup_eligible,omitempty"`
	BackedUp         bool     `json:"backed_up,omitempty"`
	CreatedAt        string   `json:"created_at,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	LastUsedAt       string   `json:"last_used_at,omitempty"`
}

type PasskeyRegistrationStartRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}

type PasskeyRegistrationStartResponse struct {
	RegistrationToken string                 `json:"registration_token"`
	ExpiresAt         string                 `json:"expires_at"`
	PublicKey         PasskeyCreationOptions `json:"public_key"`
}

type PasskeyRegistrationFinishRequest struct {
	RegistrationToken string         `json:"registration_token"`
	Name              string         `json:"name,omitempty"`
	Credential        map[string]any `json:"credential"`
}

type PasskeyLoginStartRequest struct {
	Username string `json:"username,omitempty"`
}

type PasskeyLoginStartResponse struct {
	ChallengeToken string         `json:"challenge_token"`
	ExpiresAt      string         `json:"expires_at"`
	PublicKey      map[string]any `json:"public_key"`
}

type PasskeyLoginFinishRequest struct {
	ChallengeToken string         `json:"challenge_token"`
	Credential     map[string]any `json:"credential"`
}

type PasskeyCreationOptions struct {
	Challenge              string                        `json:"challenge"`
	RP                     PasskeyRelyingParty           `json:"rp"`
	User                   PasskeyUser                   `json:"user"`
	PubKeyCredParams       []PasskeyCredentialParameter  `json:"pubKeyCredParams"`
	Timeout                int                           `json:"timeout"`
	Attestation            string                        `json:"attestation"`
	AuthenticatorSelection PasskeyAuthenticatorSelection `json:"authenticatorSelection"`
}

type PasskeyRelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasskeyUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type PasskeyCredentialParameter struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type PasskeyAuthenticatorSelection struct {
	ResidentKey      string `json:"residentKey"`
	UserVerification string `json:"userVerification"`
}

type ServiceRegistration struct {
	ServiceID    string         `json:"service_id"`
	ServiceType  ServiceType    `json:"service_type"`
	ServiceName  string         `json:"service_name"`
	PublicURL    string         `json:"public_url"`
	Version      string         `json:"version"`
	Capabilities map[string]any `json:"capabilities"`
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

type RegisteredService struct {
	ServiceID       string         `json:"service_id"`
	ServiceType     ServiceType    `json:"service_type"`
	ServiceName     string         `json:"service_name"`
	PublicURL       string         `json:"public_url"`
	Version         string         `json:"version"`
	Status          ServiceStatus  `json:"status"`
	AssignmentRole  string         `json:"assignment_role,omitempty"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at,omitempty"`
	HealthStatus    string         `json:"health_status,omitempty"`
	HeartbeatStale  bool           `json:"heartbeat_stale,omitempty"`
	HeartbeatAgeSec *int64         `json:"heartbeat_age_sec,omitempty"`
	CurrentStreamID string         `json:"current_stream_id,omitempty"`
	Capabilities    map[string]any `json:"capabilities"`
	TokenID         string         `json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
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
	CaptionAudioURL  string `json:"caption_audio_url,omitempty"`
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
	ServiceID       string             `json:"service_id"`
	CurrentStreamID string             `json:"current_stream_id,omitempty"`
	Status          string             `json:"status"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
	Timestamp       time.Time          `json:"timestamp,omitempty"`
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

type StreamArtifact struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	SizeBytes    int64  `json:"size_bytes"`
}

type ServiceArtifactReport struct {
	ServiceID string           `json:"service_id"`
	StreamID  string           `json:"stream_id"`
	Artifacts []StreamArtifact `json:"artifacts"`
}

type WorkerEventType string

const (
	WorkerEventCurrentTime   WorkerEventType = "overlay.current_time"
	WorkerEventParticipants  WorkerEventType = "overlay.participants"
	WorkerEventActiveSpeaker WorkerEventType = "overlay.active_speaker"
	WorkerEventDiscordChat   WorkerEventType = "overlay.discord_chat"
	WorkerEventCaptionTelop  WorkerEventType = "caption.telop"
	WorkerEventCaptionFinal  WorkerEventType = "caption.final"
)

type WorkerEvent struct {
	ID        string          `json:"id"`
	StreamID  string          `json:"stream_id"`
	ServiceID string          `json:"service_id,omitempty"`
	Type      WorkerEventType `json:"type"`
	Payload   map[string]any  `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type WorkerStreamContext struct {
	StreamID           string `json:"stream_id"`
	StreamName         string `json:"stream_name,omitempty"`
	EncoderRecorderURL string `json:"encoder_recorder_url,omitempty"`
	StreamIngestToken  string `json:"stream_ingest_token,omitempty"`
	OverlayProfileID   string `json:"overlay_profile_id,omitempty"`
	CaptionProfileID   string `json:"caption_profile_id,omitempty"`
}

type DiscordVoiceJob struct {
	StreamID          string `json:"stream_id"`
	GuildID           string `json:"guild_id"`
	VoiceChannelID    string `json:"voice_channel_id"`
	TextChannelID     string `json:"text_channel_id,omitempty"`
	EncoderAudioURL   string `json:"encoder_audio_url,omitempty"`
	CaptionAudioURL   string `json:"caption_audio_url,omitempty"`
	StreamIngestToken string `json:"stream_ingest_token,omitempty"`
	WorkerEventsURL   string `json:"worker_events_url,omitempty"`
	WorkerEventsToken string `json:"worker_events_token,omitempty"`
}

type DiscordBotStartJobRequest = DiscordVoiceJob

type EncoderInputMode string

const (
	EncoderInputModeExternal       EncoderInputMode = "external"
	EncoderInputModeDiscordOpusRTP EncoderInputMode = "discord_opus_rtp"
)

type DiscordOpusPacket struct {
	SSRC       uint32    `json:"ssrc"`
	UserID     string    `json:"user_id,omitempty"`
	Sequence   uint16    `json:"sequence"`
	Timestamp  uint32    `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`
	OpusBase64 string    `json:"opus_base64"`
}

type DiscordOpusIngestRequest struct {
	StreamID string              `json:"stream_id"`
	Source   string              `json:"source"`
	Packets  []DiscordOpusPacket `json:"packets"`
}

type DiscordOpusIngestResponse struct {
	Accepted      bool   `json:"accepted"`
	StreamID      string `json:"stream_id"`
	AcceptedCount int    `json:"accepted_count"`
	RTPForwarded  int    `json:"rtp_forwarded"`
	LogsPath      string `json:"logs_path,omitempty"`
}

type DiscordAudioBridgeStatus struct {
	StreamID          string    `json:"stream_id"`
	BridgeActive      bool      `json:"bridge_active"`
	StartedAt         time.Time `json:"started_at"`
	LastPacketAt      time.Time `json:"last_packet_at,omitempty"`
	PacketsTotal      int64     `json:"packets_total"`
	RTPForwardedTotal int64     `json:"rtp_forwarded"`
	LastPacketAgeSec  float64   `json:"last_packet_age_sec"`
}

type StartStreamRequest struct {
	DiscordConfigID       string `json:"discord_config_id,omitempty"`
	DiscordGuildID        string `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string `json:"discord_text_channel_id,omitempty"`
	EncoderInputURL       string `json:"encoder_input_url,omitempty"`
	EncoderRTMPURL        string `json:"encoder_rtmp_url,omitempty"`
	EncoderProfileID      string `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string `json:"overlay_profile_id,omitempty"`
	ArchiveProfileID      string `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string `json:"youtube_output_id,omitempty"`
}

type StreamSettingsWriteRequest struct {
	DiscordConfigID       string `json:"discord_config_id,omitempty"`
	DiscordGuildID        string `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string `json:"discord_text_channel_id,omitempty"`
	AutoStartTrigger      string `json:"auto_start_trigger,omitempty"`
	EncoderProfileID      string `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string `json:"overlay_profile_id,omitempty"`
	ArchiveProfileID      string `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string `json:"youtube_output_id,omitempty"`
	EncoderInputURL       string `json:"encoder_input_url,omitempty"`
}

type StreamWriteRequest struct {
	Name string `json:"name"`
	StreamSettingsWriteRequest
}

type ArchiveRuntimeConfig struct {
	DriveDestinationID     string `json:"drive_destination_id,omitempty"`
	ArchiveProfileID       string `json:"archive_profile_id,omitempty"`
	AuthMode               string `json:"auth_mode,omitempty"`
	OAuthAccountID         string `json:"oauth_account_id,omitempty"`
	OAuthProviderID        string `json:"oauth_provider_id,omitempty"`
	FolderIDSecretName     string `json:"folder_id_secret_name,omitempty"`
	BasePath               string `json:"base_path,omitempty"`
	SharedDrive            bool   `json:"shared_drive,omitempty"`
	ClientID               string `json:"client_id,omitempty"`
	ClientSecretSecretName string `json:"client_secret_secret_name,omitempty"`
	RefreshTokenSecretName string `json:"refresh_token_secret_name,omitempty"`
}

type YouTubeOutputMode string

const (
	YouTubeOutputModeStreamKey     YouTubeOutputMode = "stream_key"
	YouTubeOutputModeLiveAPIDryRun YouTubeOutputMode = "live_api_dry_run"
	YouTubeOutputModeLiveAPI       YouTubeOutputMode = "live_api"
)

type DiscordConfig struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	ServiceID            string    `json:"service_id,omitempty"`
	GuildID              string    `json:"guild_id,omitempty"`
	VoiceChannelID       string    `json:"voice_channel_id,omitempty"`
	TextChannelID        string    `json:"text_channel_id,omitempty"`
	BotTokenConfigured   bool      `json:"bot_token_configured,omitempty"`
	BotTokenFingerprint  string    `json:"bot_token_fingerprint,omitempty"`
	CaptionEnabled       bool      `json:"caption_enabled,omitempty"`
	STTProfileID         string    `json:"stt_profile_id,omitempty"`
	ReconnectEnabled     bool      `json:"reconnect_enabled,omitempty"`
	ReconnectMaxAttempts int       `json:"reconnect_max_attempts,omitempty"`
	ReconnectBaseDelay   string    `json:"reconnect_base_delay,omitempty"`
	ReconnectMaxDelay    string    `json:"reconnect_max_delay,omitempty"`
	AudioForwardEnabled  bool      `json:"audio_forward_enabled,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type DiscordConfigWriteRequest struct {
	Name                 string `json:"name"`
	ServiceID            string `json:"service_id,omitempty"`
	GuildID              string `json:"guild_id,omitempty"`
	VoiceChannelID       string `json:"voice_channel_id,omitempty"`
	TextChannelID        string `json:"text_channel_id,omitempty"`
	BotToken             string `json:"bot_token,omitempty"`
	CaptionEnabled       *bool  `json:"caption_enabled,omitempty"`
	STTProfileID         string `json:"stt_profile_id,omitempty"`
	ReconnectEnabled     *bool  `json:"reconnect_enabled,omitempty"`
	ReconnectMaxAttempts int    `json:"reconnect_max_attempts,omitempty"`
	ReconnectBaseDelay   string `json:"reconnect_base_delay,omitempty"`
	ReconnectMaxDelay    string `json:"reconnect_max_delay,omitempty"`
	AudioForwardEnabled  *bool  `json:"audio_forward_enabled,omitempty"`
}

type YouTubeOutput struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Mode                   YouTubeOutputMode `json:"mode"`
	RTMPURL                string            `json:"rtmp_url,omitempty"`
	StreamKeyConfigured    bool              `json:"stream_key_configured,omitempty"`
	StreamKeyFingerprint   string            `json:"stream_key_fingerprint,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	BroadcastTitleTemplate string            `json:"broadcast_title_template,omitempty"`
	BroadcastDescription   string            `json:"broadcast_description,omitempty"`
	PrivacyStatus          string            `json:"privacy_status,omitempty"`
	LatencyPreference      string            `json:"latency_preference,omitempty"`
	EnableAutoStart        bool              `json:"enable_auto_start,omitempty"`
	EnableAutoStop         bool              `json:"enable_auto_stop,omitempty"`
	CompleteOnStop         bool              `json:"complete_on_stop"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

type YouTubeOutputWriteRequest struct {
	Name                   string            `json:"name"`
	Mode                   YouTubeOutputMode `json:"mode"`
	RTMPURL                string            `json:"rtmp_url,omitempty"`
	StreamKey              string            `json:"stream_key,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	BroadcastTitleTemplate string            `json:"broadcast_title_template,omitempty"`
	BroadcastDescription   string            `json:"broadcast_description,omitempty"`
	PrivacyStatus          string            `json:"privacy_status,omitempty"`
	LatencyPreference      string            `json:"latency_preference,omitempty"`
	EnableAutoStart        *bool             `json:"enable_auto_start,omitempty"`
	EnableAutoStop         *bool             `json:"enable_auto_stop,omitempty"`
	CompleteOnStop         *bool             `json:"complete_on_stop,omitempty"`
}

type YouTubeRuntimeConfig struct {
	Mode                YouTubeOutputMode `json:"mode"`
	OutputID            string            `json:"output_id,omitempty"`
	OAuthAccountID      string            `json:"oauth_account_id,omitempty"`
	BroadcastID         string            `json:"broadcast_id,omitempty"`
	LiveStreamID        string            `json:"live_stream_id,omitempty"`
	StreamKeySecretName string            `json:"stream_key_secret_name,omitempty"`
	WatchURL            string            `json:"watch_url,omitempty"`
	DryRun              bool              `json:"dry_run,omitempty"`
	CompleteOnStop      bool              `json:"complete_on_stop,omitempty"`
	CompleteRetryCount  int               `json:"complete_retry_count,omitempty"`
	CompleteNextRetryAt string            `json:"complete_next_retry_at,omitempty"`
	CompleteLastError   string            `json:"complete_last_error,omitempty"`
}

type EncoderStartStreamRequest struct {
	StreamID            string               `json:"stream_id"`
	Name                string               `json:"name"`
	InputURL            string               `json:"input_url,omitempty"`
	InputMode           string               `json:"input_mode,omitempty"`
	RTMPURL             string               `json:"rtmp_url"`
	StreamKey           string               `json:"stream_key,omitempty"`
	StreamKeySecretName string               `json:"stream_key_secret_name,omitempty"`
	EncoderProfileID    string               `json:"encoder_profile_id,omitempty"`
	ArchiveProfileID    string               `json:"archive_profile_id,omitempty"`
	YouTubeRuntime      YouTubeRuntimeConfig `json:"youtube_runtime,omitempty"`
	ArchiveConfig       ArchiveRuntimeConfig `json:"archive_config,omitempty"`
	DryRun              bool                 `json:"dry_run,omitempty"`
}

type EncoderPackageStreamRequest struct {
	StreamID      string               `json:"stream_id"`
	Name          string               `json:"name"`
	StartedAt     time.Time            `json:"started_at,omitempty"`
	DryRun        bool                 `json:"dry_run,omitempty"`
	ArchiveConfig ArchiveRuntimeConfig `json:"archive_config,omitempty"`
}

type MissingStreamAssignmentsResponse struct {
	Code                string   `json:"code"`
	MissingServiceTypes []string `json:"missing_service_types"`
}

type ReadinessIssue struct {
	ServiceID   string `json:"service_id,omitempty"`
	ServiceType string `json:"service_type,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

const (
	ReadinessIssueMissingStreamAssignment        = "missing_stream_assignment"
	ReadinessIssueServiceCallTokenMissing        = "service_call_token_missing"
	ReadinessIssueServicePublicURLInvalid        = "service_public_url_invalid"
	ReadinessIssueServicePublicURLBlocked        = "service_public_url_blocked"
	ReadinessIssueEncoderPublicURLMissing        = "encoder_public_url_missing"
	ReadinessIssueEncoderPublicURLInvalid        = "encoder_public_url_invalid"
	ReadinessIssueEncoderPublicURLBlocked        = "encoder_public_url_blocked"
	ReadinessIssueServiceOffline                 = "service_offline"
	ReadinessIssueServiceHeartbeatStale          = "service_heartbeat_stale"
	ReadinessIssueDiscordAudioForwardUnavailable = "discord_audio_forward_unavailable"
	ReadinessIssueDiscordAudioCaptureUnavailable = "discord_audio_capture_unavailable"
	ReadinessIssueDiscordConfigRequired          = "discord_config_required"
	ReadinessIssueDiscordConfigNotFound          = "discord_config_not_found"
	ReadinessIssueDiscordConfigInvalid           = "discord_config_invalid"
	ReadinessIssueDiscordConfigServiceMismatch   = "discord_config_service_mismatch"
	ReadinessIssueYouTubeOutputNotFound          = "youtube_output_not_found"
	ReadinessIssueYouTubeOutputInvalidConfig     = "youtube_output_invalid_config"
	ReadinessIssueYouTubeStreamKeyUnavailable    = "youtube_stream_key_unavailable"
	ReadinessIssueYouTubeLiveAPIUnavailable      = "youtube_live_api_unavailable"
	ReadinessIssueYouTubeOAuthAccountUnavailable = "youtube_oauth_account_unavailable"
	ReadinessIssueArchiveProfileNotFound         = "archive_profile_not_found"
	ReadinessIssueArchiveProfileInvalidConfig    = "archive_profile_invalid_config"
	ReadinessIssueDriveDestinationNotFound       = "drive_destination_not_found"
	ReadinessIssueDriveDestinationUnavailable    = "drive_destination_unavailable"
	ReadinessIssueDriveOAuthAccountUnavailable   = "drive_oauth_account_unavailable"
)

var KnownStartReadinessIssueCodes = []string{
	ReadinessIssueMissingStreamAssignment,
	ReadinessIssueServiceCallTokenMissing,
	ReadinessIssueServicePublicURLInvalid,
	ReadinessIssueServicePublicURLBlocked,
	ReadinessIssueEncoderPublicURLMissing,
	ReadinessIssueEncoderPublicURLInvalid,
	ReadinessIssueEncoderPublicURLBlocked,
	ReadinessIssueServiceOffline,
	ReadinessIssueServiceHeartbeatStale,
	ReadinessIssueDiscordAudioForwardUnavailable,
	ReadinessIssueDiscordAudioCaptureUnavailable,
	ReadinessIssueDiscordConfigRequired,
	ReadinessIssueDiscordConfigNotFound,
	ReadinessIssueDiscordConfigInvalid,
	ReadinessIssueDiscordConfigServiceMismatch,
	ReadinessIssueYouTubeOutputNotFound,
	ReadinessIssueYouTubeOutputInvalidConfig,
	ReadinessIssueYouTubeStreamKeyUnavailable,
	ReadinessIssueYouTubeLiveAPIUnavailable,
	ReadinessIssueYouTubeOAuthAccountUnavailable,
	ReadinessIssueArchiveProfileNotFound,
	ReadinessIssueArchiveProfileInvalidConfig,
	ReadinessIssueDriveDestinationNotFound,
	ReadinessIssueDriveDestinationUnavailable,
	ReadinessIssueDriveOAuthAccountUnavailable,
}

type StartReadinessResponse struct {
	StreamID             string              `json:"stream_id"`
	Ready                bool                `json:"ready"`
	MissingServiceTypes  []string            `json:"missing_service_types"`
	Issues               []ReadinessIssue    `json:"issues"`
	AssignedServiceCount int                 `json:"assigned_service_count"`
	PrimaryServiceCount  int                 `json:"primary_service_count"`
	Assignments          []RegisteredService `json:"assignments,omitempty"`
}

type StreamJob struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Status           StreamStatus `json:"status"`
	DiscordConfigID  string       `json:"discord_config_id,omitempty"`
	DiscordGuildID   string       `json:"discord_guild_id,omitempty"`
	DiscordVoiceID   string       `json:"discord_voice_channel_id,omitempty"`
	DiscordTextID    string       `json:"discord_text_channel_id,omitempty"`
	EncoderProfileID string       `json:"encoder_profile_id,omitempty"`
	CaptionProfileID string       `json:"caption_profile_id,omitempty"`
	OverlayProfileID string       `json:"overlay_profile_id,omitempty"`
	ArchiveProfileID string       `json:"archive_profile_id,omitempty"`
	YouTubeOutputID  string       `json:"youtube_output_id,omitempty"`
	EncoderInputMode string       `json:"encoder_input_mode,omitempty"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type EncoderProfile struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Width               int    `json:"width"`
	Height              int    `json:"height"`
	FPS                 int    `json:"fps"`
	VideoCodec          string `json:"video_codec"`
	AudioCodec          string `json:"audio_codec"`
	VideoBitrateKbps    int    `json:"video_bitrate_kbps"`
	AudioBitrateKbps    int    `json:"audio_bitrate_kbps"`
	AudioSampleRateHz   int    `json:"audio_sample_rate_hz"`
	KeyframeIntervalSec int    `json:"keyframe_interval_sec"`
}

type ProfileKind string

const (
	ProfileEncoder       ProfileKind = "encoder"
	ProfileArchive       ProfileKind = "archive"
	ProfileCaption       ProfileKind = "caption"
	ProfileOverlay       ProfileKind = "overlay"
	ProfileDiscordConfig ProfileKind = "discord_config"
	ProfileYouTubeOutput ProfileKind = "youtube_output"
)

type Profile struct {
	ID        string         `json:"id"`
	Kind      ProfileKind    `json:"kind"`
	Name      string         `json:"name"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ProfileWriteRequest struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

type SecuritySettings struct {
	PasswordMinLength        int      `json:"password_min_length"`
	PasswordHash             string   `json:"password_hash"`
	LoginLockoutThreshold    int      `json:"login_lockout_threshold"`
	SessionIdleTimeoutMin    int      `json:"session_idle_timeout_min"`
	SessionAbsoluteLifetimeH int      `json:"session_absolute_lifetime_h"`
	RememberMeEnabled        bool     `json:"remember_me_enabled"`
	MFAMode                  string   `json:"mfa_mode"`
	MFARequiredRoles         []string `json:"mfa_required_roles,omitempty"`
	MFASupportedMethods      []string `json:"mfa_supported_methods,omitempty"`
	PasskeyStatus            string   `json:"passkey_status,omitempty"`
	UpdatedAt                string   `json:"updated_at,omitempty"`
}

type SecretStatus struct {
	Name        string `json:"name"`
	Configured  bool   `json:"configured"`
	Fingerprint string `json:"fingerprint,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type SecretUpdateRequest struct {
	Value string `json:"value"`
}

type ArchiveUploadResult struct {
	DryRun              bool              `json:"dry_run"`
	FolderIDConfigured  bool              `json:"folder_id_configured,omitempty"`
	FolderIDFingerprint string            `json:"folder_id_fingerprint,omitempty"`
	FileCount           int               `json:"file_count,omitempty"`
	FileFingerprints    map[string]string `json:"file_fingerprints,omitempty"`
	Attempts            int               `json:"attempts"`
}

type ArchiveMetadata struct {
	StreamID     string              `json:"stream_id"`
	Name         string              `json:"name"`
	StartedAtJST string              `json:"started_at_jst"`
	Archive      map[string]string   `json:"archive"`
	Upload       ArchiveUploadResult `json:"upload"`
	Commands     []map[string]any    `json:"commands,omitempty"`
	Extra        map[string]any      `json:"extra,omitempty"`
}

type AuditEvent struct {
	ID            string         `json:"id"`
	Timestamp     time.Time      `json:"timestamp"`
	ActorUserID   string         `json:"actor_user_id,omitempty"`
	ActorUsername string         `json:"actor_username,omitempty"`
	ActorIP       string         `json:"actor_ip,omitempty"`
	UserAgent     string         `json:"user_agent,omitempty"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id,omitempty"`
	Result        string         `json:"result"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	RequestID     string         `json:"request_id"`
}

type SignalType string

const (
	SignalHeartbeat         SignalType = "heartbeat"
	SignalMetric            SignalType = "metric"
	SignalLog               SignalType = "log"
	SignalEvent             SignalType = "event"
	SignalWarning           SignalType = "warning"
	SignalError             SignalType = "error"
	SignalIncident          SignalType = "incident"
	SignalDiagnosticReport  SignalType = "diagnostic_report"
	SignalRemediationAction SignalType = "remediation_action"
	SignalNotificationEvent SignalType = "notification_event"
)

type ObservabilitySignal struct {
	Type        SignalType         `json:"type"`
	Name        string             `json:"name"`
	ServiceID   string             `json:"service_id"`
	ServiceType ServiceType        `json:"service_type"`
	StreamID    string             `json:"stream_id,omitempty"`
	Status      string             `json:"status,omitempty"`
	Value       *float64           `json:"value,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Attributes  map[string]any     `json:"attributes,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

type MetricName string

const (
	MetricStreamStatus               MetricName = "stream.status"
	MetricStreamStartDurationMS      MetricName = "stream.start_duration_ms"
	MetricStreamLiveDurationSec      MetricName = "stream.live_duration_sec"
	MetricStreamStopDurationMS       MetricName = "stream.stop_duration_ms"
	MetricStreamRestartCount         MetricName = "stream.restart_count"
	MetricEncoderProcessAlive        MetricName = "encoder.process_alive"
	MetricEncoderOutputFPS           MetricName = "encoder.output_fps"
	MetricEncoderOutputBitrateKbps   MetricName = "encoder.output_bitrate_kbps"
	MetricEncoderDroppedFramesTotal  MetricName = "encoder.dropped_frames_total"
	MetricEncoderEncodeLagMS         MetricName = "encoder.encode_lag_ms"
	MetricEncoderAudioLevelDB        MetricName = "encoder.audio_level_db"
	MetricEncoderAudioSilenceSec     MetricName = "encoder.audio_silence_sec"
	MetricEncoderAudioClippingTotal  MetricName = "encoder.audio_clipping_total"
	MetricEncoderRTMPReconnectCount  MetricName = "encoder.rtmp_reconnect_count"
	MetricRecorderFileSizeBytes      MetricName = "recorder.file_size_bytes"
	MetricRecorderWriteBitrateKbps   MetricName = "recorder.write_bitrate_kbps"
	MetricRecorderDiskFreeBytes      MetricName = "recorder.disk_free_bytes"
	MetricRecorderRemuxDurationMS    MetricName = "recorder.remux_duration_ms"
	MetricSRTPacketLossPercent       MetricName = "srt.packet_loss_percent"
	MetricSRTRTTMS                   MetricName = "srt.rtt_ms"
	MetricSRTJitterMS                MetricName = "srt.jitter_ms"
	MetricSRTBandwidthMbps           MetricName = "srt.bandwidth_mbps"
	MetricRTPPacketLossPercent       MetricName = "rtp.packet_loss_percent"
	MetricRTPJitterMS                MetricName = "rtp.jitter_ms"
	MetricMediaInputBitrateKbps      MetricName = "media.input_bitrate_kbps"
	MetricMediaInputTimeoutSec       MetricName = "media.input_timeout_sec"
	MetricDiscordGatewayConnected    MetricName = "discord.gateway_connected"
	MetricDiscordVoiceConnected      MetricName = "discord.voice_connected"
	MetricDiscordAudioReceiving      MetricName = "discord.audio_receiving"
	MetricDiscordAudioPacketsTotal   MetricName = "discord.audio_packets_total"
	MetricDiscordAudioForwardedTotal MetricName = "discord.audio_forwarded_total"
	MetricDiscordAudioForwardErrors  MetricName = "discord.audio_forward_errors_total"
	MetricDiscordAudioLastPacketAge  MetricName = "discord.audio_last_packet_age_sec"
	MetricDiscordAudioLastForwardAge MetricName = "discord.audio_last_forward_age_sec"
	MetricDiscordParticipantCount    MetricName = "discord.participant_count"
	MetricDiscordWorkerEventFailures MetricName = "discord.worker_event_publish_failures_total"
	MetricDiscordReconnectCount      MetricName = "discord.reconnect_count"
	MetricDiscordVoiceDisconnects    MetricName = "discord.voice_disconnect_count"
	MetricWorkerHeartbeatAgeSec      MetricName = "worker.heartbeat_age_sec"
	MetricWorkerOverlayEventsTotal   MetricName = "worker.overlay_events_total"
	MetricWorkerCaptionEventsTotal   MetricName = "worker.caption_events_total"
	MetricWorkerSceneUpdatesTotal    MetricName = "worker.scene_updates_total"
	MetricWorkerEventSendFailures    MetricName = "worker.event_send_failures_total"
	MetricArchiveFinalMKVExists      MetricName = "archive.final_mkv_exists"
	MetricArchiveFinalMP4Exists      MetricName = "archive.final_mp4_exists"
	MetricArchivePackageStatus       MetricName = "archive.package_status"
	MetricGDriveUploadStatus         MetricName = "gdrive.upload_status"
	MetricGDriveUploadProgress       MetricName = "gdrive.upload_progress_percent"
	MetricGDriveUploadRetryCount     MetricName = "gdrive.upload_retry_count"
	MetricGDriveUploadDurationSec    MetricName = "gdrive.upload_duration_sec"
	MetricGDriveUploadFileCount      MetricName = "gdrive.upload_file_count"
	MetricGDriveUploadFolderProof    MetricName = "gdrive.upload_folder_fingerprint_present"
	MetricGDriveUploadFinalMP4Proof  MetricName = "gdrive.upload_final_mp4_fingerprint_present"
	MetricGDriveUploadMetadataProof  MetricName = "gdrive.upload_metadata_fingerprint_present"
	MetricHostCPUPercent             MetricName = "host.cpu_percent"
	MetricHostMemoryPercent          MetricName = "host.memory_percent"
	MetricHostDiskFreeBytes          MetricName = "host.disk_free_bytes"
	MetricHostNetworkTxBPS           MetricName = "host.network_tx_bps"
	MetricHostNetworkRxBPS           MetricName = "host.network_rx_bps"
)

type IncidentRule string

const (
	RuleHeartbeatTimeout          IncidentRule = "heartbeat_timeout"
	RuleEncoderProcessExited      IncidentRule = "encoder_process_exited"
	RuleRecorderNotWriting        IncidentRule = "recorder_not_writing"
	RuleArchiveRemuxSlow          IncidentRule = "archive_remux_slow"
	RuleArchivePackageFailed      IncidentRule = "archive_package_failed"
	RuleGDriveUploadFailed        IncidentRule = "gdrive_upload_failed"
	RuleGDriveUploadRetryHigh     IncidentRule = "gdrive_upload_retry_high"
	RuleHighPacketLoss            IncidentRule = "high_packet_loss"
	RuleRTMPSReconnectLoop        IncidentRule = "rtmps_reconnect_loop"
	RuleEncoderLowFPS             IncidentRule = "encoder_low_fps"
	RuleEncoderBitrateLow         IncidentRule = "encoder_bitrate_low"
	RuleEncoderDroppedFrames      IncidentRule = "encoder_dropped_frames_high"
	RuleAudioSilence              IncidentRule = "audio_silence"
	RuleAudioClipping             IncidentRule = "audio_clipping"
	RuleDiscordAudioNotReceiving  IncidentRule = "discord_audio_not_receiving"
	RuleDiscordAudioForwardFailed IncidentRule = "discord_audio_forward_failed"
	RuleDiscordReconnectLoop      IncidentRule = "discord_reconnect_loop"
	RuleDiscordVoiceDisconnected  IncidentRule = "discord_voice_disconnected"
	RuleMediaInputTimeout         IncidentRule = "media_input_timeout"
	RuleDiskLow                   IncidentRule = "disk_low"
	RuleStreamStartTimeout        IncidentRule = "stream_start_timeout"
	RuleStreamStopTimeout         IncidentRule = "stream_stop_timeout"
	RuleUnexpectedStopped         IncidentRule = "unexpected_stopped"
	RuleWorkerEventSendFailed     IncidentRule = "worker_event_send_failed"
)

type IncidentSeverity string

const (
	SeverityInfo     IncidentSeverity = "info"
	SeverityWarning  IncidentSeverity = "warning"
	SeverityError    IncidentSeverity = "error"
	SeverityCritical IncidentSeverity = "critical"
)

type IncidentStatus string

const (
	IncidentOpen          IncidentStatus = "open"
	IncidentAcknowledged  IncidentStatus = "acknowledged"
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentMitigated     IncidentStatus = "mitigated"
	IncidentResolved      IncidentStatus = "resolved"
	IncidentIgnored       IncidentStatus = "ignored"
)

type Incident struct {
	ID         string           `json:"id"`
	Rule       IncidentRule     `json:"rule"`
	Severity   IncidentSeverity `json:"severity"`
	Status     IncidentStatus   `json:"status"`
	SummaryJA  string           `json:"summary_ja"`
	ServiceID  string           `json:"service_id"`
	StreamID   string           `json:"stream_id,omitempty"`
	SignalID   string           `json:"signal_id,omitempty"`
	Report     DiagnosticReport `json:"diagnostic_report"`
	OpenedAt   time.Time        `json:"opened_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`
}

type DiagnosticReport struct {
	Summary            string   `json:"summary"`
	LikelyCause        string   `json:"likely_cause"`
	Confidence         float64  `json:"confidence"`
	Evidence           []string `json:"evidence"`
	Impact             string   `json:"impact"`
	RecommendedActions []string `json:"recommended_actions"`
	SafeAutoCandidates []string `json:"safe_auto_remediation_candidates"`
	ApprovalRequired   []string `json:"actions_requiring_approval"`
}

type RemediationMode string

const (
	RemediationDisabled       RemediationMode = "disabled"
	RemediationSuggestOnly    RemediationMode = "suggest_only"
	RemediationSafeAuto       RemediationMode = "safe_auto"
	RemediationManualApproval RemediationMode = "manual_approval"
)

type RemediationAction struct {
	ID               string          `json:"id"`
	IncidentID       string          `json:"incident_id"`
	Action           string          `json:"action"`
	Mode             RemediationMode `json:"mode"`
	Status           string          `json:"status"`
	SafeAuto         bool            `json:"safe_auto"`
	RequiresApproval bool            `json:"requires_approval"`
	Result           string          `json:"result,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ExecutedAt       *time.Time      `json:"executed_at,omitempty"`
}

type NotificationEventType string

const (
	NotificationStreamStarted              NotificationEventType = "stream.started"
	NotificationStreamLive                 NotificationEventType = "stream.live"
	NotificationStreamCompleted            NotificationEventType = "stream.completed"
	NotificationStreamFailed               NotificationEventType = "stream.failed"
	NotificationStreamWarning              NotificationEventType = "stream.warning"
	NotificationStreamError                NotificationEventType = "stream.error"
	NotificationIncidentOpened             NotificationEventType = "incident.opened"
	NotificationIncidentUpdated            NotificationEventType = "incident.updated"
	NotificationIncidentResolved           NotificationEventType = "incident.resolved"
	NotificationDiagnosticCreated          NotificationEventType = "diagnostic.created"
	NotificationRemediationPendingApproval NotificationEventType = "remediation.pending_approval"
	NotificationRemediationExecuted        NotificationEventType = "remediation.executed"
	NotificationArchiveUploadCompleted     NotificationEventType = "archive.upload.completed"
	NotificationArchiveUploadFailed        NotificationEventType = "archive.upload.failed"
	NotificationServiceOffline             NotificationEventType = "service.offline"
	NotificationServiceRecovered           NotificationEventType = "service.recovered"
)

type NotificationChannel struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Type                   string    `json:"type"`
	Enabled                bool      `json:"enabled"`
	MaskedWebhookURL       string    `json:"masked_webhook_url,omitempty"`
	SMTPPasswordConfigured bool      `json:"smtp_password_configured,omitempty"`
	MaskedEmailTarget      string    `json:"masked_email_target,omitempty"`
	SeverityFilter         []string  `json:"severity_filter,omitempty"`
	EventTypeFilter        []string  `json:"event_type_filter,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type NotificationChannelWriteRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	SMTPHost        string   `json:"smtp_host,omitempty"`
	SMTPPort        int      `json:"smtp_port,omitempty"`
	SMTPTLS         bool     `json:"smtp_tls,omitempty"`
	SMTPFrom        string   `json:"smtp_from,omitempty"`
	SMTPUsername    string   `json:"smtp_username,omitempty"`
	SMTPPassword    string   `json:"smtp_password,omitempty"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	EventTypeFilter []string `json:"event_type_filter,omitempty"`
}

type OAuthProvider struct {
	ID                     string    `json:"id"`
	ProviderType           string    `json:"provider_type"`
	Name                   string    `json:"name"`
	Enabled                bool      `json:"enabled"`
	ClientID               string    `json:"client_id"`
	ClientSecretConfigured bool      `json:"client_secret_configured"`
	Scopes                 []string  `json:"scopes"`
	AllowedDomains         []string  `json:"allowed_domains"`
	AutoProvision          bool      `json:"auto_provision"`
	DefaultRoleIDs         []string  `json:"default_role_ids,omitempty"`
	RedirectURI            string    `json:"redirect_uri"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type OAuthProviderWriteRequest struct {
	ProviderType   string   `json:"provider_type"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	AutoProvision  bool     `json:"auto_provision,omitempty"`
	DefaultRoleIDs []string `json:"default_role_ids,omitempty"`
	// RedirectURI must match AUTOSTREAM_PUBLIC_URL scheme/host in production.
	// /auth/oauth/callback is used for login providers.
	// /integrations/oauth-accounts/callback is only for Google Drive/YouTube connected accounts.
	RedirectURI string `json:"redirect_uri"`
}

type OAuthAccount struct {
	ID                     string    `json:"id"`
	ProviderID             string    `json:"provider_id"`
	ProviderType           string    `json:"provider_type"`
	AccountLabel           string    `json:"account_label"`
	Subject                string    `json:"subject,omitempty"`
	Email                  string    `json:"email,omitempty"`
	Scopes                 []string  `json:"scopes"`
	RefreshTokenConfigured bool      `json:"refresh_token_configured"`
	TokenFingerprint       string    `json:"token_fingerprint,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type OAuthAccountWriteRequest struct {
	ProviderID   string   `json:"provider_id"`
	ProviderType string   `json:"provider_type"`
	AccountLabel string   `json:"account_label"`
	Subject      string   `json:"subject,omitempty"`
	Email        string   `json:"email,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
}

type OAuthAccountConnectionStartRequest struct {
	ProviderID    string `json:"provider_id"`
	AccountLabel  string `json:"account_label,omitempty"`
	RedirectAfter string `json:"redirect_after,omitempty"`
}

type OAuthAccountConnectionStartResponse struct {
	Provider         OAuthLoginProvider `json:"provider"`
	AuthorizationURL string             `json:"authorization_url"`
	State            string             `json:"state"`
	Nonce            string             `json:"nonce"`
	ExpiresAt        time.Time          `json:"expires_at"`
	AccountLabel     string             `json:"account_label,omitempty"`
}

type OAuthAccountConnectionCallbackRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	State        string `json:"state"`
	Code         string `json:"code"`
	AccountLabel string `json:"account_label,omitempty"`
}

type DriveDestination struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	AuthMode            string    `json:"auth_mode"`
	OAuthAccountID      string    `json:"oauth_account_id,omitempty"`
	FolderIDConfigured  bool      `json:"folder_id_configured"`
	FolderIDFingerprint string    `json:"folder_id_fingerprint,omitempty"`
	MaskedFolderID      string    `json:"masked_folder_id,omitempty"`
	SharedDrive         bool      `json:"shared_drive"`
	BasePath            string    `json:"base_path"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type DriveDestinationWriteRequest struct {
	Name           string `json:"name"`
	AuthMode       string `json:"auth_mode"`
	OAuthAccountID string `json:"oauth_account_id,omitempty"`
	FolderID       string `json:"folder_id,omitempty"`
	SharedDrive    bool   `json:"shared_drive"`
	BasePath       string `json:"base_path"`
}

type OAuthLoginProvider struct {
	ID           string   `json:"id"`
	ProviderType string   `json:"provider_type"`
	Name         string   `json:"name"`
	Scopes       []string `json:"scopes"`
	RedirectURI  string   `json:"redirect_uri"`
}

type OAuthLoginStartRequest struct {
	RedirectAfter string `json:"redirect_after,omitempty"`
}

type OAuthLoginStartResponse struct {
	Provider         OAuthLoginProvider `json:"provider"`
	AuthorizationURL string             `json:"authorization_url"`
	State            string             `json:"state"`
	Nonce            string             `json:"nonce"`
	ExpiresAt        time.Time          `json:"expires_at"`
}

type OAuthLoginCallbackRequest struct {
	ProviderID string `json:"provider_id,omitempty"`
	State      string `json:"state"`
	Code       string `json:"code"`
}

type OAuthUserLink struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProviderID   string    `json:"provider_id"`
	ProviderType string    `json:"provider_type"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OAuthUserLinkWriteRequest struct {
	ProviderID   string `json:"provider_id"`
	ProviderType string `json:"provider_type,omitempty"`
	Subject      string `json:"subject"`
	Email        string `json:"email,omitempty"`
}

type NotificationDelivery struct {
	ID         string                `json:"id"`
	EventType  NotificationEventType `json:"event_type"`
	Channel    string                `json:"channel"`
	Target     string                `json:"target"`
	IncidentID string                `json:"incident_id,omitempty"`
	Status     string                `json:"status"`
	Error      string                `json:"error,omitempty"`
	Metadata   map[string]any        `json:"metadata,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
}

type ServiceRemediationExecuteRequest struct {
	ActionID   string `json:"action_id"`
	Action     string `json:"action"`
	IncidentID string `json:"incident_id"`
	StreamID   string `json:"stream_id"`
}
