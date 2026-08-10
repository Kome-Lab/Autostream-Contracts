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
		{name: "legacy stream key", got: EncoderOutputRelayModeLegacyStreamKey, want: "legacy_stream_key"},
		{name: "live API static", got: EncoderOutputRelayModeLiveAPIStatic, want: "live_api_static"},
		{name: "legacy static alias", got: EncoderOutputRelayModeStaticLegacyAlias, want: "static"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := string(test.got); got != test.want {
				t.Fatalf("Encoder output relay mode wire value = %q, want %q", got, test.want)
			}
		})
	}

	payload, err := json.Marshal(RegisteredService{
		Capabilities: map[string]any{
			"output_relay_mode": string(EncoderOutputRelayModeLiveAPIStatic),
		},
		ReportedCapabilities: map[string]any{
			"output_relay_mode": string(EncoderOutputRelayModeStaticLegacyAlias),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"reported_capabilities":{"output_relay_mode":"static"}`) {
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
		{name: "legacy stream key is canonical", capabilities: encoderOutputRelayCapabilitiesJSON("legacy_stream_key", false), valid: true},
		{name: "live API static carries binding", capabilities: encoderOutputRelayCapabilitiesJSON("live_api_static", true), valid: true},
		{name: "historical static alias remains accepted", capabilities: encoderOutputRelayCapabilitiesJSON("static", false), valid: true},
		{name: "live API static requires binding", capabilities: encoderOutputRelayCapabilitiesJSON("live_api_static", false), valid: false},
		{name: "unknown mode is rejected fail closed", capabilities: encoderOutputRelayCapabilitiesJSON("managed", false), valid: false},
		{name: "empty live API static binding is rejected", capabilities: `{"output_relay_mode":"live_api_static","output_relay_binding_id":""}`, valid: false},
		{name: "legacy capability without relay mode remains compatible", capabilities: `{"ffmpeg":true}`, valid: true},
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
	}

	validateEncoderOutputRelayModeInstance(t, registered, registeredBody(
		encoderOutputRelayCapabilitiesJSON("live_api_static", true),
		encoderOutputRelayCapabilitiesJSON("static", false),
	), true)
	validateEncoderOutputRelayModeInstance(t, registered, registeredBody(
		encoderOutputRelayCapabilitiesJSON("direct", false),
		encoderOutputRelayCapabilitiesJSON("unknown", false),
	), false)
	validateEncoderOutputRelayModeInstance(t, readiness, readinessBody(encoderOutputRelayCapabilitiesJSON("static", false)), true)
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
	for _, want := range []string{"direct", "legacy_stream_key", "live_api_static", "static", "canonical", "legacy", "fail closed"} {
		if !strings.Contains(mode, want) {
			t.Fatalf("EncoderOutputRelayMode is missing %q", want)
		}
	}

	capabilities := component("EncoderOutputRelayCapabilities")
	for _, want := range []string{"output_relay_mode", "output_relay_binding_id", `pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"`, "live_api_static", "stream key", "RTMPS"} {
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
