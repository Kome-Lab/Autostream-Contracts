package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func validateEncoderOutputRelayModeInstance(t *testing.T, schema *jsonschema.Schema, body string, wantValid bool) {
	t.Helper()

	var instance any
	if err := json.Unmarshal([]byte(body), &instance); err != nil {
		t.Fatalf("decode test instance: %v", err)
	}
	err := schema.Validate(instance)
	if wantValid && err != nil {
		t.Fatalf("expected valid instance, got %v\n%s", err, body)
	}
	if !wantValid && err == nil {
		t.Fatalf("expected invalid instance\n%s", body)
	}
}

func encoderOutputRelayCapabilitiesJSON(mode string, binding bool) string {
	if binding {
		return fmt.Sprintf(`{"output_relay_mode":%q,"output_relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, mode)
	}
	return fmt.Sprintf(`{"output_relay_mode":%q}`, mode)
}

func TestEncoderOutputRelayModeVocabulary(t *testing.T) {
	for _, test := range []struct {
		name string
		got  EncoderOutputRelayMode
		want string
	}{
		{name: "direct", got: EncoderOutputRelayModeDirect, want: "direct"},
		{name: "live API relay static", got: EncoderOutputRelayModeLiveAPIRelayStatic, want: "live_api_relay_static"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := string(test.got); got != test.want {
				t.Fatalf("Encoder output relay mode wire value = %q, want %q", got, test.want)
			}
		})
	}

	payload, err := json.Marshal(RegisteredService{
		Capabilities: map[string]any{
			"output_relay_mode": string(EncoderOutputRelayModeLiveAPIRelayStatic),
		},
		ReportedCapabilities: map[string]any{
			"output_relay_mode": string(EncoderOutputRelayModeDirect),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"reported_capabilities":{"output_relay_mode":"direct"}`) {
		t.Fatalf("RegisteredService must expose reported relay capabilities: %s", payload)
	}
}

func TestEncoderOutputRelayModeSchemaCompatibility(t *testing.T) {
	const capabilitiesSchema = "encoder-output-relay-capabilities.schema.json"
	registration := compileContractJSONSchema(t, "service-registration.schema.json", capabilitiesSchema)
	registered := compileContractJSONSchema(t, "registered-service.schema.json", capabilitiesSchema)
	heartbeat := compileContractJSONSchema(t, "heartbeat.schema.json", capabilitiesSchema)
	tokenCreate := compileContractJSONSchema(t, "service-token-create-request.schema.json", capabilitiesSchema)
	readiness := compileContractJSONSchema(t, "start-readiness-response.schema.json", capabilitiesSchema)

	registrationBody := func(capabilities string) string {
		return `{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","public_url":"http://encoder.example.com","version":"v1","capabilities":` + capabilities + `}`
	}
	registeredBody := func(capabilities, reportedCapabilities string) string {
		return `{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","ssl_enabled":false,"public_url":"http://encoder.example.com","version":"v1","status":"online","capabilities":` + capabilities + `,"reported_capabilities":` + reportedCapabilities + `,"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`
	}
	heartbeatBody := func(capabilities string) string {
		return `{"service_id":"encoder-1","status":"online","capabilities":` + capabilities + `}`
	}
	tokenCreateBody := func(capabilities string) string {
		return `{"service_type":"encoder_recorder","scopes":["service.register"],"capabilities":` + capabilities + `}`
	}
	readinessBody := func(capabilities string) string {
		return `{"stream_id":"stream-1","ready":true,"missing_service_types":[],"issues":[],"assigned_service_count":1,"primary_service_count":1,"assignments":[{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","public_url":"http://encoder.example.com","version":"v1","status":"online","capabilities":` + capabilities + `,"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}]}`
	}

	for _, test := range []struct {
		name         string
		capabilities string
		valid        bool
	}{
		{name: "direct is canonical", capabilities: encoderOutputRelayCapabilitiesJSON("direct", false), valid: true},
		{name: "live API relay static carries binding", capabilities: encoderOutputRelayCapabilitiesJSON("live_api_relay_static", true), valid: true},
		{name: "removed legacy stream key is rejected", capabilities: encoderOutputRelayCapabilitiesJSON("legacy_stream_key", false), valid: false},
		{name: "removed live API static is rejected", capabilities: encoderOutputRelayCapabilitiesJSON("live_api_static", true), valid: false},
		{name: "removed static alias is rejected", capabilities: encoderOutputRelayCapabilitiesJSON("static", false), valid: false},
		{name: "live API relay static requires binding", capabilities: encoderOutputRelayCapabilitiesJSON("live_api_relay_static", false), valid: false},
		{name: "unknown mode is rejected fail closed", capabilities: encoderOutputRelayCapabilitiesJSON("managed", false), valid: false},
		{name: "empty live API relay binding is rejected", capabilities: `{"output_relay_mode":"live_api_relay_static","output_relay_binding_id":""}`, valid: false},
		{name: "capability without relay mode remains valid", capabilities: `{"ffmpeg":true}`, valid: true},
		{name: "preserved visual and frame capabilities", capabilities: `{"scene_frames_mjpeg_srt":true,"worker_frame_ingest_mjpeg_srt":true,"scene_appearance_v1":true,"live_video_cover_v1":true,"live_encoder_runtime_settings":true}`, valid: true},
		{name: "removed scene video flag true", capabilities: `{"scene_video_srt":true}`, valid: false},
		{name: "removed scene video flag false", capabilities: `{"scene_video_srt":false}`, valid: false},
		{name: "removed ingest flag true", capabilities: `{"worker_video_ingest_srt":true}`, valid: false},
		{name: "removed ingest flag false", capabilities: `{"worker_video_ingest_srt":false}`, valid: false},
		{name: "canonical flags cannot rescue removed aliases", capabilities: `{"scene_frames_mjpeg_srt":true,"worker_frame_ingest_mjpeg_srt":true,"scene_video_srt":true,"worker_video_ingest_srt":true}`, valid: false},
	} {
		t.Run("registration/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, registration, registrationBody(test.capabilities), test.valid)
		})
		t.Run("heartbeat/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, heartbeat, heartbeatBody(test.capabilities), test.valid)
		})
		t.Run("token create/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, tokenCreate, tokenCreateBody(test.capabilities), test.valid)
		})
		t.Run("registered capabilities/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, registered, registeredBody(test.capabilities, `{}`), test.valid)
		})
		t.Run("registered reported capabilities/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, registered, registeredBody(`{}`, test.capabilities), test.valid)
		})
		t.Run("readiness/"+test.name, func(t *testing.T) {
			validateEncoderOutputRelayModeInstance(t, readiness, readinessBody(test.capabilities), test.valid)
		})
	}

	validateEncoderOutputRelayModeInstance(t, registered, registeredBody(
		encoderOutputRelayCapabilitiesJSON("live_api_relay_static", true),
		encoderOutputRelayCapabilitiesJSON("direct", false),
	), true)
	validateEncoderOutputRelayModeInstance(t, registered, registeredBody(
		encoderOutputRelayCapabilitiesJSON("direct", false),
		encoderOutputRelayCapabilitiesJSON("unknown", false),
	), false)
	validateEncoderOutputRelayModeInstance(t, readiness, readinessBody(encoderOutputRelayCapabilitiesJSON("live_api_relay_static", true)), true)
	validateEncoderOutputRelayModeInstance(t, readiness, readinessBody(encoderOutputRelayCapabilitiesJSON("static", false)), false)
	validateEncoderOutputRelayModeInstance(t, readiness, readinessBody(encoderOutputRelayCapabilitiesJSON("unknown", false)), false)

	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", capabilitiesSchema))
	if err != nil {
		t.Fatal(err)
	}
	var capabilityDocument struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(body, &capabilityDocument); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"output_relay_url", "rtmp_url", "stream_key", "stream_key_secret_name"} {
		if _, ok := capabilityDocument.Properties[forbidden]; ok {
			t.Fatalf("Encoder relay capability schema must not expose %q", forbidden)
		}
	}
}

func TestEncoderOutputRelayModeOpenAPIContract(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	component := func(name string) string {
		t.Helper()
		marker := "    " + name + ":\n"
		start := strings.Index(raw, marker)
		if start < 0 {
			t.Fatalf("control-api.yaml is missing %s", name)
		}
		section := raw[start+len(marker):]
		for offset := 0; ; {
			next := strings.Index(section[offset:], "\n    ")
			if next < 0 {
				break
			}
			next += offset
			afterIndent := next + len("\n    ")
			if afterIndent < len(section) && section[afterIndent] != ' ' && section[afterIndent] != '\t' {
				section = section[:next]
				break
			}
			offset = next + 1
		}
		return section
	}

	mode := component("EncoderOutputRelayMode")
	for _, want := range []string{"direct", "live_api_relay_static", "fail closed"} {
		if !strings.Contains(mode, want) {
			t.Fatalf("EncoderOutputRelayMode is missing %q", want)
		}
	}
	for _, removed := range []string{"legacy_stream_key", "live_api_static", "static alias"} {
		if strings.Contains(mode, removed) {
			t.Fatalf("EncoderOutputRelayMode retained removed value %q", removed)
		}
	}

	capabilities := component("EncoderOutputRelayCapabilities")
	for _, want := range []string{"output_relay_mode", "output_relay_binding_id", `pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"`, "live_api_relay_static", "stream key", "RTMPS"} {
		if !strings.Contains(capabilities, want) {
			t.Fatalf("EncoderOutputRelayCapabilities is missing %q", want)
		}
	}

	for _, test := range []struct {
		name string
		refs int
	}{
		{name: "ServiceRegistrationRequest", refs: 1},
		{name: "Heartbeat", refs: 1},
		{name: "RegisteredService", refs: 2},
		{name: "ServiceTokenCreateRequest", refs: 1},
	} {
		section := component(test.name)
		if got := strings.Count(section, "#/components/schemas/EncoderOutputRelayCapabilities"); got != test.refs {
			t.Fatalf("%s references EncoderOutputRelayCapabilities %d times, want %d", test.name, got, test.refs)
		}
	}
}
