package contracts

import (
	"time"
)

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
	MetricStreamStatus                      MetricName = "stream.status"
	MetricStreamStartDurationMS             MetricName = "stream.start_duration_ms"
	MetricStreamLiveDurationSec             MetricName = "stream.live_duration_sec"
	MetricStreamStopDurationMS              MetricName = "stream.stop_duration_ms"
	MetricStreamRestartCount                MetricName = "stream.restart_count"
	MetricEncoderProcessAlive               MetricName = "encoder.process_alive"
	MetricEncoderOutputFPS                  MetricName = "encoder.output_fps"
	MetricEncoderOutputBitrateKbps          MetricName = "encoder.output_bitrate_kbps"
	MetricEncoderDroppedFramesTotal         MetricName = "encoder.dropped_frames_total"
	MetricEncoderEncodeLagMS                MetricName = "encoder.encode_lag_ms"
	MetricEncoderAudioLevelDB               MetricName = "encoder.audio_level_db"
	MetricEncoderAudioSilenceSec            MetricName = "encoder.audio_silence_sec"
	MetricEncoderAudioClippingTotal         MetricName = "encoder.audio_clipping_total"
	MetricEncoderRTMPReconnectCount         MetricName = "encoder.rtmp_reconnect_count"
	MetricRecorderFileSizeBytes             MetricName = "recorder.file_size_bytes"
	MetricRecorderWriteBitrateKbps          MetricName = "recorder.write_bitrate_kbps"
	MetricRecorderDiskFreeBytes             MetricName = "recorder.disk_free_bytes"
	MetricRecorderRemuxDurationMS           MetricName = "recorder.remux_duration_ms"
	MetricSRTPacketLossPercent              MetricName = "srt.packet_loss_percent"
	MetricSRTRTTMS                          MetricName = "srt.rtt_ms"
	MetricSRTJitterMS                       MetricName = "srt.jitter_ms"
	MetricSRTBandwidthMbps                  MetricName = "srt.bandwidth_mbps"
	MetricRTPPacketLossPercent              MetricName = "rtp.packet_loss_percent"
	MetricRTPJitterMS                       MetricName = "rtp.jitter_ms"
	MetricMediaInputBitrateKbps             MetricName = "media.input_bitrate_kbps"
	MetricMediaInputTimeoutSec              MetricName = "media.input_timeout_sec"
	MetricDiscordGatewayConnected           MetricName = "discord.gateway_connected"
	MetricDiscordVoiceConnected             MetricName = "discord.voice_connected"
	MetricDiscordAudioReceiving             MetricName = "discord.audio_receiving"
	MetricDiscordAudioPacketsTotal          MetricName = "discord.audio_packets_total"
	MetricDiscordAudioForwardedTotal        MetricName = "discord.audio_forwarded_total"
	MetricDiscordAudioForwardErrors         MetricName = "discord.audio_forward_errors_total"
	MetricDiscordAudioLastPacketAge         MetricName = "discord.audio_last_packet_age_sec"
	MetricDiscordAudioLastForwardAge        MetricName = "discord.audio_last_forward_age_sec"
	MetricDiscordParticipantCount           MetricName = "discord.participant_count"
	MetricDiscordWorkerEventFailures        MetricName = "discord.worker_event_publish_failures_total"
	MetricDiscordReconnectCount             MetricName = "discord.reconnect_count"
	MetricDiscordVoiceDisconnects           MetricName = "discord.voice_disconnect_count"
	MetricDiscordCaptionAudioPacketsTotal   MetricName = "discord.caption_audio_packets_total"
	MetricDiscordCaptionAudioForwardedTotal MetricName = "discord.caption_audio_forwarded_total"
	MetricDiscordCaptionAudioForwardErrors  MetricName = "discord.caption_audio_forward_errors_total"
	MetricDiscordCaptionAudioLastForwardAge MetricName = "discord.caption_audio_last_forward_age_sec"
	MetricDiscordCaptionUnresolvedSSRC      MetricName = "discord.caption_unresolved_ssrc_total"
	MetricWorkerHeartbeatAgeSec             MetricName = "worker.heartbeat_age_sec"
	MetricWorkerOverlayEventsTotal          MetricName = "worker.overlay_events_total"
	MetricWorkerCaptionEventsTotal          MetricName = "worker.caption_events_total"
	MetricWorkerSceneUpdatesTotal           MetricName = "worker.scene_updates_total"
	MetricWorkerEventSendFailures           MetricName = "worker.event_send_failures_total"
	MetricWorkerCaptionInterimTotal         MetricName = "worker.caption_interim_total"
	MetricWorkerCaptionFinalTotal           MetricName = "worker.caption_final_total"
	MetricWorkerCaptionProviderErrors       MetricName = "worker.caption_provider_errors_total"
	MetricWorkerCaptionReconnectsTotal      MetricName = "worker.caption_reconnects_total"
	MetricWorkerCaptionFinalizeTotal        MetricName = "worker.caption_finalize_total"
	MetricWorkerCaptionUtteranceEndTotal    MetricName = "worker.caption_utterance_end_total"
	MetricWorkerCaptionAudioToInterimMS     MetricName = "worker.caption_audio_to_interim_ms"
	MetricWorkerCaptionAudioToFinalMS       MetricName = "worker.caption_audio_to_final_ms"
	MetricEncoderCaptionArchiveFinalTotal   MetricName = "encoder.caption_archive_final_total"
	MetricEncoderCaptionArchiveDedupedTotal MetricName = "encoder.caption_archive_deduped_total"
	MetricArchiveFinalMKVExists             MetricName = "archive.final_mkv_exists"
	MetricArchiveFinalMP4Exists             MetricName = "archive.final_mp4_exists"
	MetricArchivePackageStatus              MetricName = "archive.package_status"
	MetricGDriveUploadStatus                MetricName = "gdrive.upload_status"
	MetricGDriveUploadProgress              MetricName = "gdrive.upload_progress_percent"
	MetricGDriveUploadRetryCount            MetricName = "gdrive.upload_retry_count"
	MetricGDriveUploadDurationSec           MetricName = "gdrive.upload_duration_sec"
	MetricGDriveUploadFileCount             MetricName = "gdrive.upload_file_count"
	MetricGDriveUploadFolderProof           MetricName = "gdrive.upload_folder_fingerprint_present"
	MetricGDriveUploadFinalMP4Proof         MetricName = "gdrive.upload_final_mp4_fingerprint_present"
	MetricGDriveUploadMetadataProof         MetricName = "gdrive.upload_metadata_fingerprint_present"
	MetricHostCPUPercent                    MetricName = "host.cpu_percent"
	MetricHostMemoryPercent                 MetricName = "host.memory_percent"
	MetricHostDiskFreeBytes                 MetricName = "host.disk_free_bytes"
	MetricHostNetworkTxBPS                  MetricName = "host.network_tx_bps"
	MetricHostNetworkRxBPS                  MetricName = "host.network_rx_bps"
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

type DiagnosticRerunOutcome string

const (
	DiagnosticRerunOutcomeEvaluated    DiagnosticRerunOutcome = "evaluated"
	DiagnosticRerunOutcomeInconclusive DiagnosticRerunOutcome = "inconclusive"
)

type DiagnosticRerunReason string

const (
	DiagnosticRerunReasonSavedSignalMissing         DiagnosticRerunReason = "saved_signal_missing"
	DiagnosticRerunReasonSavedSignalNotFound        DiagnosticRerunReason = "saved_signal_not_found"
	DiagnosticRerunReasonSavedSignalNoLongerMatches DiagnosticRerunReason = "saved_signal_no_longer_matches_rule"
	DiagnosticRerunReasonIncidentUpdatedDuringRerun DiagnosticRerunReason = "incident_updated_during_rerun"
)

// DiagnosticRerunResponse reports a diagnostic-only re-evaluation. Incident
// lifecycle and remediation state are intentionally unchanged.
type DiagnosticRerunResponse struct {
	Incident Incident               `json:"incident"`
	Outcome  DiagnosticRerunOutcome `json:"outcome"`
	Reason   DiagnosticRerunReason  `json:"reason,omitempty"`
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
