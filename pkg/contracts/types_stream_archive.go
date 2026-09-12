package contracts

import (
	"time"
)

type StreamArtifact struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	SizeBytes    int64  `json:"size_bytes"`
}

type ServiceArtifactReport struct {
	ServiceID        string           `json:"service_id"`
	StreamID         string           `json:"stream_id"`
	ArchiveRunID     string           `json:"archive_run_id"`
	ArchiveStartedAt *time.Time       `json:"archive_started_at"`
	Artifacts        []StreamArtifact `json:"artifacts"`
}

type StartStreamRequest struct {
	DiscordConfigID       string  `json:"discord_config_id,omitempty"`
	DiscordGuildID        string  `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string  `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string  `json:"discord_text_channel_id,omitempty"`
	EncoderInputURL       string  `json:"encoder_input_url,omitempty"`
	EncoderRTMPURL        string  `json:"encoder_rtmp_url,omitempty"`
	EncoderProfileID      string  `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string  `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string  `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB    float64 `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID      string  `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string  `json:"youtube_output_id,omitempty"`
}

type StreamSettingsWriteRequest struct {
	DiscordConfigID       string  `json:"discord_config_id,omitempty"`
	DiscordGuildID        string  `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string  `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string  `json:"discord_text_channel_id,omitempty"`
	AutoStartTrigger      string  `json:"auto_start_trigger,omitempty"`
	EncoderProfileID      string  `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string  `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string  `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB    float64 `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID      string  `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string  `json:"youtube_output_id,omitempty"`
	EncoderInputURL       string  `json:"encoder_input_url,omitempty"`
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
	SharedDrive            bool   `json:"shared_drive,omitempty"`
	ClientID               string `json:"client_id,omitempty"`
	ClientSecretSecretName string `json:"client_secret_secret_name,omitempty"`
	RefreshTokenSecretName string `json:"refresh_token_secret_name,omitempty"`
}

type EncoderPackageStreamRequest struct {
	StreamID     string    `json:"stream_id"`
	ArchiveRunID string    `json:"archive_run_id"`
	Name         string    `json:"name"`
	StartedAt    time.Time `json:"started_at"`
	DryRun       bool      `json:"dry_run,omitempty"`
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
	ReadinessIssueMissingStreamAssignment              = "missing_stream_assignment"
	ReadinessIssueServiceCallTokenMissing              = "service_call_token_missing"
	ReadinessIssueServicePublicURLInvalid              = "service_public_url_invalid"
	ReadinessIssueServicePublicURLBlocked              = "service_public_url_blocked"
	ReadinessIssueEncoderPublicURLMissing              = "encoder_public_url_missing"
	ReadinessIssueEncoderPublicURLInvalid              = "encoder_public_url_invalid"
	ReadinessIssueEncoderPublicURLBlocked              = "encoder_public_url_blocked"
	ReadinessIssueServiceOffline                       = "service_offline"
	ReadinessIssueServiceHeartbeatStale                = "service_heartbeat_stale"
	ReadinessIssueDiscordAudioForwardUnavailable       = "discord_audio_forward_unavailable"
	ReadinessIssueDiscordAudioCaptureUnavailable       = "discord_audio_capture_unavailable"
	ReadinessIssueDiscordConfigRequired                = "discord_config_required"
	ReadinessIssueDiscordConfigNotFound                = "discord_config_not_found"
	ReadinessIssueDiscordConfigInvalid                 = "discord_config_invalid"
	ReadinessIssueDiscordConfigServiceMismatch         = "discord_config_service_mismatch"
	ReadinessIssueYouTubeOutputNotFound                = "youtube_output_not_found"
	ReadinessIssueYouTubeOutputInvalidConfig           = "youtube_output_invalid_config"
	ReadinessIssueYouTubeStreamKeyUnavailable          = "youtube_stream_key_unavailable"
	ReadinessIssueYouTubeLiveAPIUnavailable            = "youtube_live_api_unavailable"
	ReadinessIssueYouTubeOAuthAccountUnavailable       = "youtube_oauth_account_unavailable"
	ReadinessIssueYouTubeRelayStaticUnavailable        = "youtube_relay_static_unavailable"
	ReadinessIssueYouTubeRelayStaticBindingUnavailable = "youtube_relay_static_binding_unavailable"
	ReadinessIssueYouTubeRelayBindingStoreUnavailable  = "youtube_relay_binding_store_unavailable"
	ReadinessIssueYouTubeRelayBindingInUse             = "youtube_relay_binding_in_use"
	ReadinessIssueYouTubeRelayStaticRecoveryRequired   = "youtube_relay_static_recovery_required"
	ReadinessIssueArchiveProfileNotFound               = "archive_profile_not_found"
	ReadinessIssueArchiveProfileInvalidConfig          = "archive_profile_invalid_config"
	ReadinessIssueDriveDestinationNotFound             = "drive_destination_not_found"
	ReadinessIssueDriveDestinationUnavailable          = "drive_destination_unavailable"
	ReadinessIssueDriveOAuthAccountUnavailable         = "drive_oauth_account_unavailable"
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
	ReadinessIssueYouTubeRelayStaticUnavailable,
	ReadinessIssueYouTubeRelayStaticBindingUnavailable,
	ReadinessIssueYouTubeRelayBindingStoreUnavailable,
	ReadinessIssueYouTubeRelayBindingInUse,
	ReadinessIssueYouTubeRelayStaticRecoveryRequired,
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
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Status            StreamStatus `json:"status"`
	ArchiveRunID      string       `json:"archive_run_id,omitempty"`
	ArchiveStartedAt  *time.Time   `json:"archive_started_at,omitempty"`
	ArchiveReportedAt *time.Time   `json:"archive_reported_at,omitempty"`
	DiscordConfigID   string       `json:"discord_config_id,omitempty"`
	DiscordGuildID    string       `json:"discord_guild_id,omitempty"`
	DiscordVoiceID    string       `json:"discord_voice_channel_id,omitempty"`
	DiscordTextID     string       `json:"discord_text_channel_id,omitempty"`
	EncoderProfileID  string       `json:"encoder_profile_id,omitempty"`
	CaptionProfileID  string       `json:"caption_profile_id,omitempty"`
	OverlayProfileID  string       `json:"overlay_profile_id,omitempty"`
	ArchiveProfileID  string       `json:"archive_profile_id,omitempty"`
	YouTubeOutputID   string       `json:"youtube_output_id,omitempty"`
	EncoderInputMode  string       `json:"encoder_input_mode,omitempty"`
	StartedAt         *time.Time   `json:"started_at,omitempty"`
	CompletedAt       *time.Time   `json:"completed_at,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

type StreamPreviewLink struct {
	StreamID  string    `json:"stream_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
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

type DriveDestination struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	AuthMode            string    `json:"auth_mode"`
	OAuthAccountID      string    `json:"oauth_account_id,omitempty"`
	FolderIDConfigured  bool      `json:"folder_id_configured"`
	FolderIDFingerprint string    `json:"folder_id_fingerprint,omitempty"`
	MaskedFolderID      string    `json:"masked_folder_id,omitempty"`
	SharedDrive         bool      `json:"shared_drive"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type DriveDestinationWriteRequest struct {
	Name           string `json:"name"`
	AuthMode       string `json:"auth_mode"`
	OAuthAccountID string `json:"oauth_account_id,omitempty"`
	FolderID       string `json:"folder_id,omitempty"`
	SharedDrive    bool   `json:"shared_drive"`
}
