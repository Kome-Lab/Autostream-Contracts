package contracts

import (
	"time"
)

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
	StreamID              string           `json:"stream_id"`
	StreamName            string           `json:"stream_name,omitempty"`
	EncoderRecorderURL    string           `json:"encoder_recorder_url,omitempty"`
	StreamIngestToken     string           `json:"stream_ingest_token,omitempty"`
	OverlayProfileID      string           `json:"overlay_profile_id,omitempty"`
	CaptionProfileID      string           `json:"caption_profile_id,omitempty"`
	SceneAppearance       *SceneAppearance `json:"scene_appearance,omitempty"`
	EncoderProfileID      string           `json:"encoder_profile_id,omitempty"`
	VideoWidth            int              `json:"video_width,omitempty"`
	VideoHeight           int              `json:"video_height,omitempty"`
	VideoFPS              int              `json:"video_fps,omitempty"`
	VideoIngestURL        string           `json:"video_ingest_url,omitempty"`
	VideoIngestPassphrase string           `json:"video_ingest_passphrase,omitempty"`
	VideoIngestPBKeyLen   int              `json:"video_ingest_pbkeylen,omitempty"`
}

type WorkerStartJobRequest = WorkerStreamContext

type WorkerCaptionRuntimeSettingsRequest struct {
	CaptionProfileID string `json:"caption_profile_id"`
}

type ResolvedDiscordTarget struct {
	GuildID        string `json:"guild_id"`
	TextChannelID  string `json:"text_channel_id"`
	VoiceChannelID string `json:"voice_channel_id"`
}

// DiscordTargetSnapshot is the server-resolved v2 target. Preset and manual
// selection inputs are intentionally absent from the Bot-facing DTO.
type DiscordTargetSnapshot struct {
	Revision uint64                `json:"revision"`
	Resolved ResolvedDiscordTarget `json:"resolved"`
}

type DiscordVoiceJob struct {
	StreamID                    string `json:"stream_id"`
	JobGeneration               uint64 `json:"job_generation"`
	GuildID                     string `json:"guild_id"`
	VoiceChannelID              string `json:"voice_channel_id"`
	TextChannelID               string `json:"text_channel_id,omitempty"`
	EncoderAudioURL             string `json:"encoder_audio_url,omitempty"`
	CaptionAudioURL             string `json:"caption_audio_url,omitempty"`
	CaptionAudioToken           string `json:"caption_audio_token,omitempty"`
	CaptionAudioFlushMS         int    `json:"caption_audio_flush_ms,omitempty"`
	CaptionAudioMaxBatchPackets int    `json:"caption_audio_max_batch_packets,omitempty"`
	UnresolvedSSRCBufferMS      int    `json:"unresolved_ssrc_buffer_ms,omitempty"`
	StreamIngestToken           string `json:"stream_ingest_token,omitempty"`
	WorkerEventsURL             string `json:"worker_events_url,omitempty"`
	WorkerEventsToken           string `json:"worker_events_token,omitempty"`
}

type DiscordBotStartJobRequest = DiscordVoiceJob

// DiscordBotStartJobV2Request is the strict additive v2 wire DTO. The existing
// DiscordVoiceJob and DiscordBotStartJobRequest remain source- and wire-stable
// for the legacy compatibility branch through Execution Bundle 8.
type DiscordBotStartJobV2Request struct {
	SchemaVersion               int                   `json:"schema_version"`
	StreamID                    string                `json:"stream_id"`
	JobGeneration               uint64                `json:"job_generation"`
	DiscordTarget               DiscordTargetSnapshot `json:"discord_target"`
	EncoderAudioURL             string                `json:"encoder_audio_url,omitempty"`
	CaptionAudioURL             string                `json:"caption_audio_url,omitempty"`
	CaptionAudioToken           string                `json:"caption_audio_token,omitempty"`
	CaptionAudioFlushMS         int                   `json:"caption_audio_flush_ms,omitempty"`
	CaptionAudioMaxBatchPackets int                   `json:"caption_audio_max_batch_packets,omitempty"`
	UnresolvedSSRCBufferMS      *int                  `json:"unresolved_ssrc_buffer_ms,omitempty"`
	StreamIngestToken           string                `json:"stream_ingest_token,omitempty"`
	WorkerEventsURL             string                `json:"worker_events_url,omitempty"`
	WorkerEventsToken           string                `json:"worker_events_token,omitempty"`
}

type EncoderInputMode string

const (
	EncoderInputModeExternal             EncoderInputMode = "external"
	EncoderInputModeDiscordOpusRTP       EncoderInputMode = "discord_opus_rtp"
	EncoderInputModeWorkerSceneSRTLegacy EncoderInputMode = "worker_scene_srt"
	EncoderInputModeWorkerSceneFramesSRT EncoderInputMode = "worker_scene_frames_srt"
)

type DiscordOpusPacket struct {
	SSRC                 uint32    `json:"ssrc"`
	UserID               string    `json:"user_id,omitempty"`
	JobGeneration        uint64    `json:"job_generation,omitempty"`
	ConnectionGeneration uint64    `json:"connection_generation,omitempty"`
	Sequence             uint16    `json:"sequence"`
	Timestamp            uint32    `json:"timestamp"`
	ReceivedAt           time.Time `json:"received_at"`
	OpusBase64           string    `json:"opus_base64"`
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

type CaptionProfileConfig struct {
	Provider                    string   `json:"provider"`
	Model                       string   `json:"model"`
	Language                    string   `json:"language"`
	APIKeySecretName            string   `json:"api_key_secret_name"`
	EndpointingMS               int      `json:"endpointing_ms"`
	UtteranceEndMS              int      `json:"utterance_end_ms"`
	LocalFinalizeMS             int      `json:"local_finalize_ms"`
	SpeakerIdleCloseSeconds     int      `json:"speaker_idle_close_seconds"`
	KeepAliveIntervalSeconds    int      `json:"keepalive_interval_seconds"`
	InterimResults              bool     `json:"interim_results"`
	SmartFormat                 bool     `json:"smart_format"`
	Keyterms                    []string `json:"keyterms,omitempty"`
	MIPOptOut                   bool     `json:"mip_opt_out"`
	ReplayBufferMaxMS           int      `json:"replay_buffer_max_ms"`
	DelayMS                     int      `json:"delay_ms"`
	CaptionAudioFlushMS         int      `json:"caption_audio_flush_ms"`
	CaptionAudioMaxBatchPackets int      `json:"caption_audio_max_batch_packets"`
	UnresolvedSSRCBufferMS      int      `json:"unresolved_ssrc_buffer_ms"`
	ConversationMaxItems        int      `json:"conversation_max_items"`
	ConversationReorderWindowMS int      `json:"conversation_reorder_window_ms"`
	VoiceInterimTTLSeconds      int      `json:"voice_interim_ttl_seconds"`
	VoiceFinalTTLSeconds        int      `json:"voice_final_ttl_seconds"`
	ShowVoiceTranscripts        bool     `json:"show_voice_transcripts"`
	ShowLegacyCaptionBar        bool     `json:"show_legacy_caption_bar"`
}
