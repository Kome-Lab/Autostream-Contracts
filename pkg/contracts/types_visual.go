package contracts

import (
	"time"
)

type VisualReadiness string

const (
	VisualReadinessReady    VisualReadiness = "ready"
	VisualReadinessNotReady VisualReadiness = "not_ready"
	VisualReadinessUnknown  VisualReadiness = "unknown"
)

type VisualSafeErrorCode string

const (
	VisualErrorInvalidThemeID               VisualSafeErrorCode = "invalid_theme_id"
	VisualErrorMediaAssetFormatUnsupported  VisualSafeErrorCode = "media_asset_format_unsupported"
	VisualErrorMediaAssetTooLarge           VisualSafeErrorCode = "media_asset_too_large"
	VisualErrorMediaAssetDecodeFailed       VisualSafeErrorCode = "media_asset_decode_failed"
	VisualErrorMediaAssetAspectRatioInvalid VisualSafeErrorCode = "media_asset_aspect_ratio_invalid"
	VisualErrorMediaAssetVariantProcessing  VisualSafeErrorCode = "media_asset_variant_processing"
	VisualErrorMediaAssetVariantFailed      VisualSafeErrorCode = "media_asset_variant_failed"
	VisualErrorMediaAssetUnauthorized       VisualSafeErrorCode = "media_asset_unauthorized"
	VisualErrorMediaAssetNotFound           VisualSafeErrorCode = "media_asset_not_found"
	VisualErrorMediaAssetHashMismatch       VisualSafeErrorCode = "media_asset_hash_mismatch"
	VisualErrorMediaAssetDimensionMismatch  VisualSafeErrorCode = "media_asset_dimension_mismatch"
	VisualErrorMediaAssetTimeout            VisualSafeErrorCode = "media_asset_timeout"
	VisualErrorDiscordTargetInvalid         VisualSafeErrorCode = "discord_target_invalid"
	VisualErrorPresetNotFound               VisualSafeErrorCode = "preset_not_found"
	VisualErrorPresetRevisionConflict       VisualSafeErrorCode = "preset_revision_conflict"
	VisualErrorStaleJobGeneration           VisualSafeErrorCode = "stale_job_generation"
	VisualErrorStaleCoverGeneration         VisualSafeErrorCode = "stale_cover_generation"
	VisualErrorStaleCoverRevision           VisualSafeErrorCode = "stale_cover_revision"
	VisualErrorIdempotencyConflict          VisualSafeErrorCode = "idempotency_conflict"
	VisualErrorCoverApplyAmbiguous          VisualSafeErrorCode = "cover_apply_ambiguous"
	VisualErrorCoverGraphUnavailable        VisualSafeErrorCode = "cover_graph_unavailable"
	VisualErrorRevisionPayloadConflict      VisualSafeErrorCode = "revision_payload_conflict"
	VisualErrorCapabilityRequired           VisualSafeErrorCode = "capability_required"
)

type VisualSafeError struct {
	Code      VisualSafeErrorCode `json:"code"`
	RequestID string              `json:"request_id,omitempty"`
}

// MediaAssetDescriptor is safe metadata for one immutable processed variant.
// It never carries a storage key, filesystem path, external URL, or raw bytes.
type MediaAssetDescriptor struct {
	AssetID             string           `json:"asset_id"`
	VariantID           string           `json:"variant_id"`
	Usage               string           `json:"usage"`
	MediaType           string           `json:"media_type"`
	Width               int              `json:"width"`
	Height              int              `json:"height"`
	ByteSize            int64            `json:"byte_size"`
	PixelCount          int64            `json:"pixel_count"`
	Animated            bool             `json:"animated"`
	AspectRatioErrorPPM *int             `json:"aspect_ratio_error_ppm,omitempty"`
	Opaque              *bool            `json:"opaque,omitempty"`
	SHA256              string           `json:"sha256"`
	Revision            uint64           `json:"revision"`
	Readiness           VisualReadiness  `json:"readiness"`
	Error               *VisualSafeError `json:"error,omitempty"`
}

// SceneAppearance is the immutable Control Panel-owned start snapshot. Its
// Generation shares the Video Cover JobGeneration epoch for this stream start;
// it is intentionally distinct from the Worker-issued job generation.
type SceneAppearance struct {
	Generation      uint64                `json:"generation"`
	Revision        uint64                `json:"revision"`
	Capability      string                `json:"capability"`
	Readiness       VisualReadiness       `json:"readiness"`
	BackgroundMode  string                `json:"background_mode"`
	Background      *MediaAssetDescriptor `json:"background,omitempty"`
	HeaderTitleMode string                `json:"header_title_mode"`
	CustomTitle     string                `json:"custom_title,omitempty"`
	Error           *VisualSafeError      `json:"error,omitempty"`
}

// VideoCoverStartSnapshot binds the desired Cover state and visual epoch at
// stream start. New producers send both active and inactive snapshots; whole-
// field omission remains only for legacy callers during compatibility.
type VideoCoverStartSnapshot struct {
	JobGeneration  uint64                `json:"job_generation"`
	Revision       uint64                `json:"revision"`
	Active         bool                  `json:"active"`
	IdempotencyKey string                `json:"idempotency_key"`
	CoverAsset     *MediaAssetDescriptor `json:"cover_asset,omitempty"`
}

// EncoderVideoCoverApplyRequest is a fenced mutation of the Cover layer only.
// Watermark state is intentionally not representable. HideConfirmed is
// required by the schema for inactive requests because an ambiguous hide must
// be reconciled with a read instead of automatically resent. Consumers must
// validate raw bytes with ValidateEncoderVideoCoverApplyRequest before
// using this presence-losing decoded DTO.
type EncoderVideoCoverApplyRequest struct {
	StreamID           string                `json:"stream_id"`
	JobGeneration      uint64                `json:"job_generation"`
	ExpectedGeneration uint64                `json:"expected_generation"`
	Revision           uint64                `json:"revision"`
	Active             bool                  `json:"active"`
	IdempotencyKey     string                `json:"idempotency_key"`
	CoverAsset         *MediaAssetDescriptor `json:"cover_asset,omitempty"`
	HideConfirmed      bool                  `json:"hide_confirmed,omitempty"`
}

type VideoCoverDesiredState struct {
	Active    bool   `json:"active"`
	Revision  uint64 `json:"revision"`
	Source    string `json:"source"`
	VariantID string `json:"variant_id,omitempty"`
}

// VideoCoverAppliedState uses pointers for conditionally present known-state
// fields so a known inactive state can still encode active=false explicitly.
type VideoCoverAppliedState struct {
	State     string `json:"state"`
	Active    *bool  `json:"active,omitempty"`
	Revision  uint64 `json:"revision,omitempty"`
	VariantID string `json:"variant_id,omitempty"`
}

type VideoVisualLayerState struct {
	Enabled   bool   `json:"enabled"`
	Revision  uint64 `json:"revision"`
	VariantID string `json:"variant_id,omitempty"`
}

type VisualAudioContinuity struct {
	ProcessRestart           int `json:"process_restart"`
	AudioEncoderRestart      int `json:"audio_encoder_restart"`
	AudioMuxRestart          int `json:"audio_mux_restart"`
	GraphRebuild             int `json:"graph_rebuild"`
	Reconnect                int `json:"reconnect"`
	SequenceLoss             int `json:"sequence_loss"`
	TimestampDiscontinuity   int `json:"timestamp_discontinuity"`
	IntentionalMuteInsertion int `json:"intentional_mute_insertion"`
}

type VisualPipelineInvariant struct {
	Layers                    []string              `json:"layers"`
	WatermarkTopmost          bool                  `json:"watermark_topmost"`
	CoverWatermarkIndependent bool                  `json:"cover_watermark_independent"`
	OutputParity              []string              `json:"output_parity"`
	AudioContinuity           VisualAudioContinuity `json:"audio_continuity"`
}

// VideoCoverAppliedWitness exists only after the Encoder graph applied the
// state. Request acceptance and asset validation cannot produce this witness.
type VideoCoverAppliedWitness struct {
	GraphApplied bool                    `json:"graph_applied"`
	Generation   uint64                  `json:"generation"`
	Revision     uint64                  `json:"revision"`
	Active       bool                    `json:"active"`
	Cover        VideoVisualLayerState   `json:"cover"`
	Watermark    VideoVisualLayerState   `json:"watermark"`
	Pipeline     VisualPipelineInvariant `json:"pipeline"`
}

// VideoCoverRuntimeState returns the actual Encoder-owned graph generation and
// keeps the independently observed Cover and Watermark layer states separate.
// Consumers must validate raw GET bytes with
// ValidateEncoderVideoCoverRuntimeState before using this decoded DTO.
type VideoCoverRuntimeState struct {
	StreamID          string                    `json:"stream_id"`
	JobGeneration     uint64                    `json:"job_generation"`
	Generation        uint64                    `json:"generation"`
	Capability        string                    `json:"capability"`
	Readiness         VisualReadiness           `json:"readiness"`
	Desired           VideoCoverDesiredState    `json:"desired"`
	Applied           VideoCoverAppliedState    `json:"applied"`
	Cover             VideoVisualLayerState     `json:"cover"`
	CoverAsset        *MediaAssetDescriptor     `json:"cover_asset,omitempty"`
	Watermark         VideoVisualLayerState     `json:"watermark"`
	Pipeline          VisualPipelineInvariant   `json:"pipeline"`
	AppliedWitness    *VideoCoverAppliedWitness `json:"applied_witness,omitempty"`
	NoAutomaticResend bool                      `json:"no_automatic_resend"`
	LastGoodApplied   *VideoCoverAppliedState   `json:"last_good_applied,omitempty"`
	Error             *VisualSafeError          `json:"error,omitempty"`
}

type EncoderVideoCoverApplyOutcome string

const (
	EncoderVideoCoverApplyOutcomeApplied   EncoderVideoCoverApplyOutcome = "applied"
	EncoderVideoCoverApplyOutcomeRejected  EncoderVideoCoverApplyOutcome = "rejected"
	EncoderVideoCoverApplyOutcomeAmbiguous EncoderVideoCoverApplyOutcome = "ambiguous"
)

// EncoderVideoCoverApplyResponse is the decoded response DTO. Consumers must
// validate raw bytes with ValidateEncoderVideoCoverApplyResponse first so
// required false booleans cannot be confused with omitted fields.
type EncoderVideoCoverApplyResponse struct {
	StreamID          string                        `json:"stream_id"`
	JobGeneration     uint64                        `json:"job_generation"`
	RequestedRevision uint64                        `json:"requested_revision"`
	ActualGeneration  uint64                        `json:"actual_generation"`
	Accepted          bool                          `json:"accepted"`
	Rejected          bool                          `json:"rejected"`
	Applied           bool                          `json:"applied"`
	Outcome           EncoderVideoCoverApplyOutcome `json:"outcome"`
	Actual            VideoCoverRuntimeState        `json:"actual"`
	Error             *VisualSafeError              `json:"error,omitempty"`
}

// EncoderVideoCoverUnavailableResponse is returned only when a running legacy
// stream has no negotiated Video Cover runtime epoch. It intentionally carries
// no request-derived generation or revision fields.
type EncoderVideoCoverUnavailableResponse struct {
	Code VisualSafeErrorCode `json:"code"`
}

type EncoderStartStreamRequest struct {
	StreamID               string                   `json:"stream_id"`
	ArchiveRunID           string                   `json:"archive_run_id,omitempty"`
	Name                   string                   `json:"name"`
	InputURL               string                   `json:"input_url,omitempty"`
	InputMode              string                   `json:"input_mode,omitempty"`
	WorkerVideoIngest      bool                     `json:"worker_video_ingest,omitempty"`
	WorkerVideoIngestToken string                   `json:"worker_video_ingest_token,omitempty"`
	RTMPURL                string                   `json:"rtmp_url"`
	StreamKeySecretName    string                   `json:"stream_key_secret_name,omitempty"`
	EncoderProfileID       string                   `json:"encoder_profile_id,omitempty"`
	OverlayProfileID       string                   `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB     float64                  `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID       string                   `json:"archive_profile_id,omitempty"`
	StartedAt              time.Time                `json:"started_at,omitempty"`
	YouTubeRuntime         YouTubeRuntimeConfig     `json:"youtube_runtime,omitempty"`
	VideoCoverStart        *VideoCoverStartSnapshot `json:"video_cover_start,omitempty"`
	DryRun                 bool                     `json:"dry_run,omitempty"`
}

type EncoderRuntimeSettingsRequest struct {
	EncoderAudioGainDB float64 `json:"encoder_audio_gain_db"`
	OverlayProfileID   string  `json:"overlay_profile_id"`
}

type EncoderVideoIngest struct {
	URL        string `json:"url"`
	Passphrase string `json:"passphrase"`
	PBKeyLen   int    `json:"pbkeylen"`
}

// EncoderStartStreamResponse is an internal service-to-service response. The
// VideoIngest field must be consumed by the dispatcher and removed before any
// audit, status, or public API serialization.
type EncoderStartStreamResponse struct {
	StreamID     string              `json:"stream_id"`
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	StartedAtJST string              `json:"started_at_jst"`
	StoppedAtJST string              `json:"stopped_at_jst,omitempty"`
	Archive      map[string]string   `json:"archive"`
	Error        string              `json:"error,omitempty"`
	VideoIngest  *EncoderVideoIngest `json:"video_ingest,omitempty"`
}
