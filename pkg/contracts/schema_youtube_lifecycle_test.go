package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYouTubeSchemasDocumentLiveAPILifecycle(t *testing.T) {
	for _, file := range []string{"youtube-output.schema.json", "youtube-output-write.schema.json", "youtube-runtime-config.schema.json", "service-runtime-config.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"live_api_dry_run",
			"live_api",
			"live_api_relay_static",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing YouTube Live API mode marker %q", file, want)
			}
		}
	}

	runtimeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeRaw := string(runtimeBody)
	for _, want := range []string{
		"broadcast_id",
		"live_stream_id",
		"relay_binding_id",
		"reusable_live_stream_id",
		"watch_url",
		"dry_run",
		"complete_on_stop",
	} {
		if !strings.Contains(runtimeRaw, want) {
			t.Fatalf("youtube-runtime-config.schema.json is missing runtime lifecycle field %q", want)
		}
	}

	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-output-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeRaw := string(writeBody)
	for _, want := range []string{
		"writeOnly",
		"enable_auto_start",
		"enable_auto_stop",
		"complete_on_stop",
		"Required for live_api, live_api_dry_run, and live_api_relay_static modes.",
	} {
		if !strings.Contains(writeRaw, want) {
			t.Fatalf("youtube-output-write.schema.json is missing Live API write marker %q", want)
		}
	}
}

func TestYouTubeRelayStaticWriteRejectsIngestFields(t *testing.T) {
	validator := compileContractJSONSchema(t, "youtube-output-write.schema.json")
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "static output without ingest fields",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}`,
			valid: true,
		},
		{
			name:  "static output with rtmp url",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","rtmp_url":"rtmps://a.rtmps.youtube.com/live2"}`,
			valid: false,
		},
		{
			name:  "static output with stream key",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","stream_key":"not-a-real-key"}`,
			valid: false,
		},
		{
			name:  "static output with watch url",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","watch_url":"https://www.youtube.com/watch?v=abc12345"}`,
			valid: false,
		},
	}
	for _, test := range tests {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := validator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	outputValidator := compileContractJSONSchema(t, "youtube-output.schema.json")
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "incomplete static output read remains repairable",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: true,
		},
		{
			name:  "static output read with rtmp url",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","rtmp_url":"rtmps://a.rtmps.youtube.com/live2","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: false,
		},
		{
			name:  "static output read with manually supplied watch url",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","watch_url":"https://www.youtube.com/watch?v=abc12345","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: false,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := outputValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}
}

func TestYouTubeRelayStaticRecoveryContractsRequireExplicitSafeResolution(t *testing.T) {
	if YouTubeOutputModeLiveAPIRelayStatic != "live_api_relay_static" {
		t.Fatalf("YouTube relay static mode wire value = %q", YouTubeOutputModeLiveAPIRelayStatic)
	}
	if ErrorCodeYouTubeRelayStaticConfigChangedReload != "youtube_relay_static_config_changed_reload" {
		t.Fatalf("YouTube relay static config-change error wire value = %q", ErrorCodeYouTubeRelayStaticConfigChangedReload)
	}
	if ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay != "live_api_requires_managed_output_relay" {
		t.Fatalf("YouTube dynamic relay error wire value = %q", ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay)
	}
	if ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream != "youtube_relay_static_completion_requires_completed_stream" {
		t.Fatalf("YouTube relay static completion guard wire value = %q", ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable != "youtube_relay_static_recovery_encoder_stop_unavailable" {
		t.Fatalf("YouTube relay static recovery unavailable wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed != "youtube_relay_static_recovery_encoder_stop_unconfirmed" {
		t.Fatalf("YouTube relay static recovery unconfirmed wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown != "youtube_relay_static_recovery_broadcast_unknown" {
		t.Fatalf("YouTube relay static recovery unknown broadcast wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid != "youtube_relay_static_recovery_dispatch_state_invalid" {
		t.Fatalf("YouTube relay static recovery invalid dispatch state wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed != "youtube_relay_static_recovery_complete_failed" {
		t.Fatalf("YouTube relay static recovery complete failure wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed)
	}

	requestBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-relay-static-recovery-resolve-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var requestSchema struct {
		AdditionalProperties bool                       `json:"additionalProperties"`
		Required             []string                   `json:"required"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(requestBody, &requestSchema); err != nil {
		t.Fatal(err)
	}
	if requestSchema.AdditionalProperties || len(requestSchema.Required) != 1 || requestSchema.Required[0] != "confirm_external_cleanup" || len(requestSchema.Properties) != 1 {
		t.Fatal("relay static recovery request must accept only required confirm_external_cleanup")
	}
	var confirmation struct {
		Type  string `json:"type"`
		Const bool   `json:"const"`
	}
	if err := json.Unmarshal(requestSchema.Properties["confirm_external_cleanup"], &confirmation); err != nil {
		t.Fatal(err)
	}
	if confirmation.Type != "boolean" || !confirmation.Const {
		t.Fatal("relay static recovery must require confirm_external_cleanup=true")
	}
	requestValidator := compileContractJSONSchema(t, "youtube-relay-static-recovery-resolve-request.schema.json")
	for _, test := range []struct {
		body  string
		valid bool
	}{
		{body: `{"confirm_external_cleanup":true}`, valid: true},
		{body: `{"confirm_external_cleanup":false}`, valid: false},
		{body: `{"confirm_external_cleanup":true,"broadcast_id":"must-not-be-accepted"}`, valid: false},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := requestValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("relay static recovery request valid=%t for %s: %v", test.valid, test.body, err)
		}
	}
	if got, err := json.Marshal(YouTubeRelayStaticRecoveryResolveRequest{ConfirmExternalCleanup: true}); err != nil || string(got) != `{"confirm_external_cleanup":true}` {
		t.Fatalf("relay static recovery request wire shape = %s, err=%v", got, err)
	}

	responseBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-relay-static-recovery-resolve-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var responseSchema struct {
		AdditionalProperties bool                       `json:"additionalProperties"`
		Required             []string                   `json:"required"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(responseBody, &responseSchema); err != nil {
		t.Fatal(err)
	}
	if responseSchema.AdditionalProperties || len(responseSchema.Required) != 3 || len(responseSchema.Properties) != 3 {
		t.Fatal("relay static recovery response must expose only resolved, cleanup, and relay_binding_id")
	}
	var cleanup struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(responseSchema.Properties["cleanup"], &cleanup); err != nil {
		t.Fatal(err)
	}
	if strings.Join(cleanup.Enum, ",") != "provider_delete,provider_complete,operator_confirmed_unknown_broadcast,operator_confirmed_provider_cleanup" {
		t.Fatalf("relay static cleanup values = %v", cleanup.Enum)
	}
	responseValidator := compileContractJSONSchema(t, "youtube-relay-static-recovery-resolve-response.schema.json")
	for _, test := range []struct {
		body  string
		valid bool
	}{
		{body: `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"provider_complete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"operator_confirmed_unknown_broadcast","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"operator_confirmed_provider_cleanup","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":false,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: false},
		{body: `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","broadcast_id":"must-not-be-returned"}`, valid: false},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := responseValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("relay static recovery response valid=%t for %s: %v", test.valid, test.body, err)
		}
	}
	if got, err := json.Marshal(YouTubeRelayStaticRecoveryResolveResponse{
		Resolved:       true,
		Cleanup:        "provider_delete",
		RelayBindingID: "relay-01234567-89ab-4def-8123-456789abcdef",
	}); err != nil || string(got) != `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}` {
		t.Fatalf("relay static recovery response wire shape = %s, err=%v", got, err)
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"youtube_relay_static_config_changed_reload",
		"live_api_requires_managed_output_relay",
		"youtube_relay_static_completion_requires_completed_stream",
	} {
		if !strings.Contains(string(openAPI), want) {
			t.Fatalf("control-api.yaml is missing relay start error code %q", want)
		}
	}
	pathMarker := "  /streams/{id}/youtube/relay-static/recovery/resolve:\n"
	start := strings.Index(string(openAPI), pathMarker)
	if start < 0 {
		t.Fatal("control-api.yaml is missing relay static recovery route")
	}
	pathSection := string(openAPI)[start+len(pathMarker):]
	if end := strings.Index(pathSection, "\n  /"); end >= 0 {
		pathSection = pathSection[:end]
	}
	for _, want := range []string{
		"operationId: resolveYouTubeRelayStaticRecovery",
		"Requires streams.stop.",
		"YouTubeRelayStaticRecoveryResolveRequest",
		"YouTubeRelayStaticRecoveryResolveResponse",
		"youtube_relay_static_external_cleanup_confirmation_required",
		"youtube_relay_static_recovery_not_found",
		"stream_relay_recovery_not_safe_while_active",
		"youtube_relay_static_recovery_not_required",
		"youtube_relay_static_recovery_encoder_stop_unavailable",
		"youtube_relay_static_recovery_encoder_stop_unconfirmed",
		"youtube_relay_static_recovery_broadcast_unknown",
		"youtube_relay_static_recovery_dispatch_state_invalid",
		"youtube_relay_static_recovery_complete_failed",
		"youtube_relay_static_recovery_cleanup_failed",
		"youtube_relay_static_recovery_cleanup_unavailable",
		"youtube_relay_static_recovery_credentials_failed",
		"resolve_youtube_relay_static_recovery_failed",
	} {
		if !strings.Contains(pathSection, want) {
			t.Fatalf("relay static recovery path is missing %q", want)
		}
	}
}
