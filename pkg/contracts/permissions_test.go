package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPermissionExistsFailsClosed(t *testing.T) {
	if !PermissionExists("streams.start") {
		t.Fatal("expected streams.start permission")
	}
	if PermissionExists("secrets.read_raw") {
		t.Fatal("raw secret permission must never exist")
	}
	if PermissionExists("unknown.permission") {
		t.Fatal("unknown permissions must fail closed")
	}
}

func TestCoreWireConstants(t *testing.T) {
	if StreamCreated != "created" || StreamLive != "live" || StreamFailed != "failed" {
		t.Fatal("stream status wire values changed")
	}
	if ServiceStatusPending != "pending" || ServiceStatusRegistered != "registered" || ServiceStatusAssigned != "assigned" || ServiceStatusRestartRequested != "restart_requested" {
		t.Fatal("service status wire values changed")
	}
	if ServiceDiscordBot != "discord_bot" || ServiceEncoderRecorder != "encoder_recorder" || ServiceWorker != "worker" {
		t.Fatal("service type wire values changed")
	}
	if AssignmentRolePrimary != "primary" || AssignmentRoleStandby != "standby" {
		t.Fatal("service assignment role wire values changed")
	}
	if ScopeServiceRegister != "service.register" || ScopeServiceConfigRead != "service.config.read" || ScopeServiceSecretResolve != "service.secret.resolve" || ScopeObservabilityIngest != "observability.ingest" || ScopeRemediationExecute != "remediation.execute" {
		t.Fatal("service scope wire values changed")
	}
	if SignalMetric != "metric" || SignalDiagnosticReport != "diagnostic_report" {
		t.Fatal("signal type wire values changed")
	}
	if WorkerEventParticipants != "overlay.participants" || WorkerEventDiscordChat != "overlay.discord_chat" || WorkerEventCaptionTelop != "caption.telop" {
		t.Fatal("worker event wire values changed")
	}
	if EncoderInputModeExternal != "external" || EncoderInputModeDiscordOpusRTP != "discord_opus_rtp" {
		t.Fatal("encoder input mode wire values changed")
	}
	if MetricWorkerEventSendFailures != "worker.event_send_failures_total" || RuleWorkerEventSendFailed != "worker_event_send_failed" {
		t.Fatal("worker observability wire values changed")
	}
	if MetricDiscordAudioReceiving != "discord.audio_receiving" || MetricDiscordAudioPacketsTotal != "discord.audio_packets_total" || RuleDiscordAudioNotReceiving != "discord_audio_not_receiving" {
		t.Fatal("discord audio observability wire values changed")
	}
	if MetricDiscordAudioForwardedTotal != "discord.audio_forwarded_total" || MetricDiscordAudioForwardErrors != "discord.audio_forward_errors_total" || RuleDiscordAudioForwardFailed != "discord_audio_forward_failed" {
		t.Fatal("discord audio forward observability wire values changed")
	}
	if MetricDiscordWorkerEventFailures != "discord.worker_event_publish_failures_total" {
		t.Fatal("discord worker event observability wire values changed")
	}
	if MetricDiscordVoiceConnected != "discord.voice_connected" || MetricDiscordReconnectCount != "discord.reconnect_count" || MetricDiscordVoiceDisconnects != "discord.voice_disconnect_count" {
		t.Fatal("discord connection observability wire values changed")
	}
	if RuleDiscordReconnectLoop != "discord_reconnect_loop" || RuleDiscordVoiceDisconnected != "discord_voice_disconnected" {
		t.Fatal("discord connection incident rule wire values changed")
	}
	if MetricEncoderAudioClippingTotal != "encoder.audio_clipping_total" || RuleAudioClipping != "audio_clipping" {
		t.Fatal("audio clipping observability wire values changed")
	}
	if MetricMediaInputTimeoutSec != "media.input_timeout_sec" || RuleMediaInputTimeout != "media_input_timeout" {
		t.Fatal("media input timeout wire values changed")
	}
	if MetricArchivePackageStatus != "archive.package_status" || MetricArchiveFinalMP4Exists != "archive.final_mp4_exists" {
		t.Fatal("archive metric wire values changed")
	}
	if MetricGDriveUploadStatus != "gdrive.upload_status" || MetricGDriveUploadRetryCount != "gdrive.upload_retry_count" || MetricGDriveUploadDurationSec != "gdrive.upload_duration_sec" {
		t.Fatal("gdrive metric wire values changed")
	}
	if MetricGDriveUploadFileCount != "gdrive.upload_file_count" || MetricGDriveUploadFolderProof != "gdrive.upload_folder_fingerprint_present" || MetricGDriveUploadFinalMP4Proof != "gdrive.upload_final_mp4_fingerprint_present" || MetricGDriveUploadMetadataProof != "gdrive.upload_metadata_fingerprint_present" {
		t.Fatal("gdrive upload proof metric wire values changed")
	}
	if RuleArchivePackageFailed != "archive_package_failed" || RuleArchiveRemuxSlow != "archive_remux_slow" || RuleGDriveUploadRetryHigh != "gdrive_upload_retry_high" {
		t.Fatal("archive observability wire values changed")
	}
	if ProfileEncoder != "encoder" || ProfileDiscordConfig != "discord_config" || ProfileYouTubeOutput != "youtube_output" {
		t.Fatal("profile kind wire values changed")
	}
	if IncidentOpen != "open" || IncidentResolved != "resolved" {
		t.Fatal("incident status wire values changed")
	}
	if RemediationSuggestOnly != "suggest_only" || RemediationSafeAuto != "safe_auto" {
		t.Fatal("remediation mode wire values changed")
	}
	if NotificationArchiveUploadFailed != "archive.upload.failed" || NotificationServiceRecovered != "service.recovered" {
		t.Fatal("notification event wire values changed")
	}
	body, err := json.Marshal(MissingStreamAssignmentsResponse{Code: "missing_stream_assignments", MissingServiceTypes: []string{"discord_bot", "encoder_recorder"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"missing_service_types"`) || strings.Contains(string(body), "MissingServiceTypes") {
		t.Fatalf("missing assignment response JSON changed: %s", body)
	}
	readinessBody, err := json.Marshal(StartReadinessResponse{StreamID: "stream-01", MissingServiceTypes: []string{"worker"}, Issues: []ReadinessIssue{{ServiceType: "worker", Code: ReadinessIssueMissingStreamAssignment, Message: "assign worker"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readinessBody), `"missing_service_types"`) || !strings.Contains(string(readinessBody), `"issues"`) || strings.Contains(string(readinessBody), "ReadinessIssue") {
		t.Fatalf("start readiness response JSON changed: %s", readinessBody)
	}
	for _, want := range []string{
		ReadinessIssueDiscordConfigServiceMismatch,
		ReadinessIssueYouTubeStreamKeyUnavailable,
		ReadinessIssueYouTubeOAuthAccountUnavailable,
		ReadinessIssueDriveDestinationUnavailable,
		ReadinessIssueDriveOAuthAccountUnavailable,
	} {
		if !containsString(KnownStartReadinessIssueCodes, want) {
			t.Fatalf("readiness issue code %q is not in KnownStartReadinessIssueCodes", want)
		}
	}
	preflightBody, err := json.Marshal(ServicePreflightResponse{Ready: false, Checks: []ServicePreflightCheck{{ID: "ffmpeg_binary", Status: "not_found", Severity: "critical", Message: "missing"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(preflightBody), `"ready"`) || !strings.Contains(string(preflightBody), `"checks"`) || strings.Contains(string(preflightBody), "ServicePreflightCheck") {
		t.Fatalf("service preflight response JSON changed: %s", preflightBody)
	}
	audioBody, err := json.Marshal(DiscordAudioBridgeStatus{StreamID: "stream-01", BridgeActive: true, PacketsTotal: 1, RTPForwardedTotal: 1, LastPacketAgeSec: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(audioBody), `"bridge_active"`) || !strings.Contains(string(audioBody), `"rtp_forwarded"`) {
		t.Fatalf("audio bridge status JSON changed: %s", audioBody)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
