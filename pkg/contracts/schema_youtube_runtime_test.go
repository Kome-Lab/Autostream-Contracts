package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYouTubeRelayStaticRuntimeConfigExcludesIngestFields(t *testing.T) {
	validator := compileContractJSONSchema(t, "youtube-runtime-config.schema.json")
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "fixed relay carries all prepared non-secret lifecycle fields",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}`,
			valid: true,
		},
		{
			name:  "fixed relay requires prepared identity and lifecycle fields",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1"}`,
			valid: false,
		},
		{
			name:  "fixed relay must complete on stop",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":false}`,
			valid: false,
		},
		{
			name:  "fixed relay cannot carry rtmp endpoint",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"rtmp_url":"rtmps://a.rtmps.youtube.com/live2"}`,
			valid: false,
		},
		{
			name:  "fixed relay cannot carry key secret reference",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"stream_key_secret_name":"youtube_stream_key_runtime_1"}`,
			valid: false,
		},
		{
			name:  "fixed relay runtime can carry canonical public watch url",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}`,
			valid: true,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := validator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	serviceValidator := compileContractJSONSchema(t,
		"service-runtime-config.schema.json",
		"registered-service.schema.json",
		"encoder-output-relay-capabilities.schema.json",
		"service-assignment.schema.json",
		"profile.schema.json",
	)
	serviceRuntimePayload := func(streamYouTubeConfig string) string {
		return `{
"service":{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","ssl_enabled":false,"version":"v1","status":"online","capabilities":{},"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z","public_url":"http://encoder.example.com"},
"assignments":[],"profiles":{},
"stream_youtube_configs":[` + streamYouTubeConfig + `]
}`
	}
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "ready fixed relay service config requires identity and lifecycle fields",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"},"active_runtime":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}}`),
			valid: false,
		},
		{
			name:  "not ready fixed relay config may expose incomplete profile for repair",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":false,"readiness_code":"youtube_output_invalid_config","readiness_message":"selected YouTube output is incomplete","youtube_config":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}}`),
			valid: true,
		},
		{
			name:  "not ready fixed relay config cannot expose a raw stream key",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":false,"readiness_code":"youtube_output_invalid_config","readiness_message":"selected YouTube output is incomplete","youtube_config":{"mode":"live_api_relay_static","stream_key":"not-a-real-key"}}`),
			valid: false,
		},
		{
			name:  "ready fixed relay config does not require an active runtime before preparation",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime requires broadcast and live stream ids",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","complete_on_stop":true}}`),
			valid: false,
		},
		{
			name:  "prepared fixed relay active runtime carries provider identifiers",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime validates relay binding format",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","relay_binding_id":"relay-binding-1","complete_on_stop":true}}`),
			valid: false,
		},
		{
			name:  "prepared fixed relay active runtime can carry canonical public watch url",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime cannot expose a raw stream key",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"stream_key":"not-a-real-key"}}`),
			valid: false,
		},
		{
			name:  "fixed relay service config can carry canonical public watch url",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}}`),
			valid: true,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := serviceValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	for _, file := range []struct {
		name    string
		path    string
		markers []string
	}{
		{
			name: "service-runtime-config.schema.json",
			path: filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"),
			markers: []string{
				"live_api_relay_static", `"required": ["rtmp_url"]`, `"required": ["stream_key_secret_name"]`,
			},
		},
		{
			name: "control-api.yaml",
			path: filepath.Join("..", "..", "openapi", "control-api.yaml"),
			markers: []string{
				"live_api_relay_static",
			},
		},
	} {
		body, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range file.markers {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing fixed-relay runtime exclusion marker %q", file.name, want)
			}
		}
	}
	requireControlOpenAPIRelayStaticExclusions(t)
}

func TestYouTubeRelayBindingMutationErrorsArePublicContracts(t *testing.T) {
	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{name: "claim check", got: ErrorCodeYouTubeRelayBindingClaimCheckFailed, want: "youtube_relay_binding_claim_check_failed"},
		{name: "release pending", got: ErrorCodeYouTubeRelayBindingReleasePending, want: "youtube_relay_binding_release_pending"},
	} {
		if test.got != test.want {
			t.Fatalf("%s relay binding mutation error = %q, want %q", test.name, test.got, test.want)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(openAPI)
	errorResponseStart := strings.Index(raw, "    ErrorResponse:\n")
	if errorResponseStart < 0 {
		t.Fatal("control-api.yaml is missing ErrorResponse")
	}
	for _, code := range []string{
		"youtube_relay_binding_claim_check_failed",
		"youtube_relay_binding_release_pending",
	} {
		if !strings.Contains(raw, "\n            - "+code+"\n") {
			t.Fatalf("ErrorResponse enum is missing %q", code)
		}
	}

	operationSection := func(path, method string) string {
		t.Helper()
		pathMarker := "  " + path + ":\n"
		pathStart := strings.Index(raw, pathMarker)
		if pathStart < 0 {
			t.Fatalf("control-api.yaml is missing %s", path)
		}
		pathSection := raw[pathStart+len(pathMarker):]
		if end := strings.Index(pathSection, "\n  /"); end >= 0 {
			pathSection = pathSection[:end]
		}
		methodMarker := "    " + method + ":\n"
		methodStart := strings.Index(pathSection, methodMarker)
		if methodStart < 0 {
			t.Fatalf("%s is missing %s", path, method)
		}
		section := pathSection[methodStart+len(methodMarker):]
		for _, nextMethod := range []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"} {
			if end := strings.Index(section, "\n    "+nextMethod+":\n"); end >= 0 {
				section = section[:end]
			}
		}
		return section
	}
	responseSection := func(operation, status string) string {
		t.Helper()
		statusMarker := "        \"" + status + "\":\n"
		statusStart := strings.Index(operation, statusMarker)
		if statusStart < 0 {
			t.Fatalf("operation is missing HTTP %s response", status)
		}
		section := operation[statusStart+len(statusMarker):]
		if end := strings.Index(section, "\n        \""); end >= 0 {
			section = section[:end]
		}
		return section
	}

	for _, test := range []struct {
		path   string
		method string
		status string
		code   string
	}{
		{path: "/youtube/outputs/{id}", method: "put", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/youtube/outputs/{id}", method: "put", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/youtube/outputs/{id}", method: "delete", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/youtube/outputs/{id}", method: "delete", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/streams/{id}", method: "delete", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/streams/{id}", method: "delete", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/streams/{id}/settings", method: "put", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/streams/{id}/settings", method: "put", status: "500", code: "youtube_relay_binding_claim_check_failed"},
	} {
		response := responseSection(operationSection(test.path, test.method), test.status)
		if !strings.Contains(response, test.code) {
			t.Fatalf("%s %s HTTP %s is missing %q", test.method, test.path, test.status, test.code)
		}
		if !strings.Contains(response, `#/components/schemas/ErrorResponse`) {
			t.Fatalf("%s %s HTTP %s must return ErrorResponse", test.method, test.path, test.status)
		}
	}
}

func TestYouTubeOutputWatchURLContractIsBackwardCompatible(t *testing.T) {
	for _, file := range []string{"youtube-output.schema.json", "youtube-output-write.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "watch_url") {
			t.Fatalf("%s is missing watch_url", file)
		}
	}

	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-output-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	for _, required := range doc.Required {
		if required == "watch_url" {
			t.Fatal("watch_url must remain optional in the API for existing stream_key profiles")
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"watch_url:", "New UI-created profiles require it", "existing API profiles without this field remain compatible"} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing YouTube watch URL compatibility marker %q", want)
		}
	}
}
