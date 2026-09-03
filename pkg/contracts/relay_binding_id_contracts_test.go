package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validRelayBindingID = "relay-01234567-89ab-4def-8123-456789abcdef"

func TestRelayBindingIDFormatContracts(t *testing.T) {
	const wantPattern = `^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	if RelayBindingIDPattern != wantPattern {
		t.Fatalf("RelayBindingIDPattern = %q, want %q", RelayBindingIDPattern, wantPattern)
	}

	for _, test := range []struct {
		name         string
		schema       string
		dependencies []string
		valid        string
		invalid      string
	}{
		{
			name:    "encoder capability",
			schema:  "encoder-output-relay-capabilities.schema.json",
			valid:   `{"output_relay_mode":"live_api_relay_static","output_relay_binding_id":"` + validRelayBindingID + `"}`,
			invalid: `{"output_relay_mode":"live_api_relay_static","output_relay_binding_id":"relay-binding-1"}`,
		},
		{
			name:    "YouTube output write",
			schema:  "youtube-output-write.schema.json",
			valid:   `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"` + validRelayBindingID + `","reusable_live_stream_id":"live-stream-1"}`,
			invalid: `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-binding-1","reusable_live_stream_id":"live-stream-1"}`,
		},
		{
			name:    "YouTube output read",
			schema:  "youtube-output.schema.json",
			valid:   `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"` + validRelayBindingID + `","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			invalid: `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"relay-binding-1","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
		},
		{
			name:    "prepared YouTube runtime",
			schema:  "youtube-runtime-config.schema.json",
			valid:   `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"` + validRelayBindingID + `","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}`,
			invalid: `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-binding-1","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}`,
		},
		{
			name:         "service runtime config",
			schema:       "service-runtime-config.schema.json",
			dependencies: []string{"registered-service.schema.json", "encoder-output-relay-capabilities.schema.json", "service-assignment.schema.json", "profile.schema.json"},
			valid: `{
"service":{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","ssl_enabled":false,"version":"v1","status":"online","capabilities":{},"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z","public_url":"http://encoder.example.com"},
"assignments":[],"profiles":{},"stream_youtube_configs":[{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"` + validRelayBindingID + `","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}}]}`,
			invalid: `{
"service":{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","ssl_enabled":false,"version":"v1","status":"online","capabilities":{},"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z","public_url":"http://encoder.example.com"},
"assignments":[],"profiles":{},"stream_youtube_configs":[{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-binding-1","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}}]}`,
		},
		{
			name:    "static recovery response",
			schema:  "youtube-relay-static-recovery-resolve-response.schema.json",
			valid:   `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"` + validRelayBindingID + `"}`,
			invalid: `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-binding-1"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			validator := compileContractJSONSchema(t, test.schema, test.dependencies...)
			validateEncoderOutputRelayModeInstance(t, validator, test.valid, true)
			validateEncoderOutputRelayModeInstance(t, validator, test.invalid, false)
		})
	}

	legacyStreamKey := compileContractJSONSchema(t, "youtube-output-write.schema.json")
	validateEncoderOutputRelayModeInstance(t, legacyStreamKey, `{"name":"legacy key output","mode":"stream_key"}`, true)

	capabilities := compileContractJSONSchema(t, "encoder-output-relay-capabilities.schema.json")
	for _, malformedBindingID := range []string{
		"relay-01234567-89ab-4def-8123-456789abcde",
		"relay-01234567-89AB-4def-8123-456789abcdef",
		"binding-01234567-89ab-4def-8123-456789abcdef",
	} {
		validateEncoderOutputRelayModeInstance(t, capabilities, `{"output_relay_mode":"live_api_relay_static","output_relay_binding_id":"`+malformedBindingID+`"}`, false)
	}

	for _, file := range []string{
		"encoder-output-relay-capabilities.schema.json",
		"youtube-output-write.schema.json",
		"youtube-output.schema.json",
		"youtube-runtime-config.schema.json",
		"service-runtime-config.schema.json",
		"youtube-relay-static-recovery-resolve-response.schema.json",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"pattern": "`+RelayBindingIDPattern+`"`) {
			t.Fatalf("%s is missing relay binding pattern", file)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(openAPI), `pattern: "`+RelayBindingIDPattern+`"`); got != 8 {
		t.Fatalf("control-api.yaml has %d canonical relay binding patterns, want 8", got)
	}
}

func TestEncoderStartStreamAllowsCanonicalStaticRuntimeWatchURL(t *testing.T) {
	validator := compileContractJSONSchema(t,
		"encoder-start-stream-request.schema.json",
		"youtube-runtime-config.schema.json",
		visualCatalogSchema,
	)

	valid := `{"stream_id":"stream-1","name":"fixed relay","rtmp_url":"rtmp://encoder.internal/live","youtube_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"` + validRelayBindingID + `","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}}`
	validateEncoderOutputRelayModeInstance(t, validator, valid, true)

	invalid := `{"stream_id":"stream-1","name":"fixed relay","rtmp_url":"rtmp://encoder.internal/live","youtube_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"` + validRelayBindingID + `","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://example.com/watch?v=abc12345"}}`
	validateEncoderOutputRelayModeInstance(t, validator, invalid, false)
}
