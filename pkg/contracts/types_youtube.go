package contracts

import (
	"time"
)

type YouTubeOutputMode string

const (
	YouTubeOutputModeStreamKey          YouTubeOutputMode = "stream_key"
	YouTubeOutputModeLiveAPIDryRun      YouTubeOutputMode = "live_api_dry_run"
	YouTubeOutputModeLiveAPI            YouTubeOutputMode = "live_api"
	YouTubeOutputModeLiveAPIRelayStatic YouTubeOutputMode = "live_api_relay_static"
)

// EncoderOutputRelayMode describes the non-secret output routing capability
// advertised by an Encoder/Recorder. It is intentionally distinct from
// YouTubeOutputMode, which selects the Control Panel output profile behavior.
type EncoderOutputRelayMode string

const (
	EncoderOutputRelayModeDirect             EncoderOutputRelayMode = "direct"
	EncoderOutputRelayModeLiveAPIRelayStatic EncoderOutputRelayMode = "live_api_relay_static"
)

// RelayBindingIDPattern is the exact format of a non-secret fixed relay
// binding identity. It is intentionally not an ingest URL or stream key.
const RelayBindingIDPattern = `^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

// EncoderOutputRelayCapabilities is the historical shared service-capability
// DTO name. Its non-secret fields now also cover negotiated Worker visual and
// Discord Bot start-envelope behavior. A live_api_relay_static relay requires an
// OutputRelayBindingID matching RelayBindingIDPattern.
type EncoderOutputRelayCapabilities struct {
	OutputRelayMode           EncoderOutputRelayMode `json:"output_relay_mode,omitempty"`
	OutputRelayBindingID      string                 `json:"output_relay_binding_id,omitempty"`
	SceneFramesMJPEGSRT       bool                   `json:"scene_frames_mjpeg_srt,omitempty"`
	WorkerFrameIngestMJPEGSRT bool                   `json:"worker_frame_ingest_mjpeg_srt,omitempty"`
	SceneAppearanceV1         bool                   `json:"scene_appearance_v1,omitempty"`
	LiveVideoCoverV1          bool                   `json:"live_video_cover_v1,omitempty"`
	DiscordResolvedTargetV2   bool                   `json:"discord_resolved_target_v2,omitempty"`
}

// ErrorCodeYouTubeRelayStaticConfigChangedReload is returned when a fixed
// relay output or its stream assignment changed after a start request read its
// configuration. Callers must reload before starting again; no provider side
// effect was accepted for that stale configuration.
const (
	// ErrorCodeYouTubeRelayBindingClaimCheckFailed means Control Panel could not
	// determine whether a fixed relay binding is still claimed, so the mutation
	// is rejected rather than risking release while its ingress may be active.
	ErrorCodeYouTubeRelayBindingClaimCheckFailed = "youtube_relay_binding_claim_check_failed"
	// ErrorCodeYouTubeRelayBindingReleasePending means a fixed relay binding is
	// still claimed and must be released through its safe lifecycle before its
	// output or stream association can change.
	ErrorCodeYouTubeRelayBindingReleasePending        = "youtube_relay_binding_release_pending"
	ErrorCodeYouTubeRelayStaticConfigChangedReload    = "youtube_relay_static_config_changed_reload"
	ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay = "live_api_requires_managed_output_relay"
	// ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream prevents a
	// fixed-relay binding from being released while its stream might still own
	// the relay ingress.
	ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream = "youtube_relay_static_completion_requires_completed_stream"
	// ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable is returned
	// when recovery cannot find the primary Encoder required to prove the
	// fixed relay is no longer in use.
	ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable = "youtube_relay_static_recovery_encoder_stop_unavailable"
	// ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed is returned
	// when a possibly-dispatched fixed-relay start cannot be proven stopped.
	ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed = "youtube_relay_static_recovery_encoder_stop_unconfirmed"
	// ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown requires explicit
	// operator investigation because a possibly-dispatched relay claim has no
	// trustworthy YouTube Broadcast identifier to complete.
	ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown = "youtube_relay_static_recovery_broadcast_unknown"
	// ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid is returned for a
	// corrupted or unsupported durable recovery phase; the binding remains
	// fenced.
	ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid = "youtube_relay_static_recovery_dispatch_state_invalid"
	// ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed retains the fixed relay
	// claim when YouTube completion after a confirmed Encoder stop fails.
	ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed = "youtube_relay_static_recovery_complete_failed"
)

type YouTubeOutput struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Mode                   YouTubeOutputMode `json:"mode"`
	RTMPURL                string            `json:"rtmp_url,omitempty"`
	StreamKeyConfigured    bool              `json:"stream_key_configured,omitempty"`
	StreamKeyFingerprint   string            `json:"stream_key_fingerprint,omitempty"`
	WatchURL               string            `json:"watch_url,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	RelayBindingID         string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID   string            `json:"reusable_live_stream_id,omitempty"`
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
	WatchURL               string            `json:"watch_url,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	RelayBindingID         string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID   string            `json:"reusable_live_stream_id,omitempty"`
	BroadcastTitleTemplate string            `json:"broadcast_title_template,omitempty"`
	BroadcastDescription   string            `json:"broadcast_description,omitempty"`
	PrivacyStatus          string            `json:"privacy_status,omitempty"`
	LatencyPreference      string            `json:"latency_preference,omitempty"`
	EnableAutoStart        *bool             `json:"enable_auto_start,omitempty"`
	EnableAutoStop         *bool             `json:"enable_auto_stop,omitempty"`
	CompleteOnStop         *bool             `json:"complete_on_stop,omitempty"`
}

type YouTubeRuntimeConfig struct {
	Mode                 YouTubeOutputMode `json:"mode"`
	OutputID             string            `json:"output_id,omitempty"`
	OAuthAccountID       string            `json:"oauth_account_id,omitempty"`
	BroadcastID          string            `json:"broadcast_id,omitempty"`
	LiveStreamID         string            `json:"live_stream_id,omitempty"`
	RelayBindingID       string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID string            `json:"reusable_live_stream_id,omitempty"`
	StreamKeySecretName  string            `json:"stream_key_secret_name,omitempty"`
	WatchURL             string            `json:"watch_url,omitempty"`
	DryRun               bool              `json:"dry_run,omitempty"`
	CompleteOnStop       bool              `json:"complete_on_stop,omitempty"`
	CompleteRetryCount   int               `json:"complete_retry_count,omitempty"`
	CompleteNextRetryAt  string            `json:"complete_next_retry_at,omitempty"`
	CompleteLastError    string            `json:"complete_last_error,omitempty"`
}

type YouTubeRelayStaticRecoveryResolveRequest struct {
	ConfirmExternalCleanup bool `json:"confirm_external_cleanup"`
}

type YouTubeRelayStaticRecoveryResolveResponse struct {
	Resolved       bool   `json:"resolved"`
	Cleanup        string `json:"cleanup"`
	RelayBindingID string `json:"relay_binding_id"`
}
