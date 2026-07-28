package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func validateServiceTransportInstance(t *testing.T, schema *jsonschema.Schema, body string, wantValid bool) {
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

func TestServiceRegistrationTransportModeCompatibility(t *testing.T) {
	schema := compileContractJSONSchema(t, "service-registration.schema.json")

	tests := []struct {
		name      string
		body      string
		wantValid bool
	}{
		{
			name: "legacy worker retains endpoint requirement",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"public_url":"http://worker-a.example.com",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: true,
		},
		{
			name: "legacy updater is ssh v1 compatible",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"public_url":"http://updater-a.example.com",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: true,
		},
		{
			name: "pull updater is endpointless",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"transport_mode":"pull_v2",
				"version":"v2.0.0",
				"capabilities":{}
			}`,
			wantValid: true,
		},
		{
			name: "pull updater rejects advertised endpoint",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"transport_mode":"pull_v2",
				"host":"127.0.0.1",
				"port":8090,
				"public_url":"http://127.0.0.1:8090",
				"version":"v2.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "pull updater rejects server owned binding",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"transport_mode":"pull_v2",
				"execution_host_id":"host-a",
				"ownership_epoch":1,
				"version":"v2.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "explicit ssh updater still requires endpoint",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"transport_mode":"ssh_v1",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "legacy updater without endpoint stays invalid",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "normal node cannot select pull transport",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"transport_mode":"pull_v2",
				"public_url":"http://worker-a.example.com",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "normal node cannot claim execution host ownership",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"execution_host_id":"host-a",
				"ownership_epoch":1,
				"public_url":"http://worker-a.example.com",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
		{
			name: "unknown transport is rejected",
			body: `{
				"service_id":"updater-a",
				"service_type":"update_agent",
				"service_name":"Updater A",
				"transport_mode":"push_v3",
				"public_url":"http://updater-a.example.com",
				"version":"v1.0.0",
				"capabilities":{}
			}`,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateServiceTransportInstance(t, schema, tt.body, tt.wantValid)
		})
	}
}

func TestPullRegistrationGoTypeCarriesNoServerOwnedBinding(t *testing.T) {
	payload, err := json.Marshal(ServiceRegistration{
		ServiceID:     "host-agent-a",
		ServiceType:   ServiceUpdateAgent,
		ServiceName:   "Host Agent A",
		TransportMode: UpdateTransportPullV2,
		Version:       "v2.0.0",
		Capabilities:  map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(payload)
	for _, forbidden := range []string{"execution_host_id", "ownership_epoch", "public_url", `"host"`, `"port"`} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("runtime pull registration exposes forbidden field %q: %s", forbidden, raw)
		}
	}

	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	if err := compileContractJSONSchema(t, "service-registration.schema.json").Validate(document); err != nil {
		t.Fatalf("runtime pull registration type does not match schema: %v\n%s", err, raw)
	}
}

func TestRegisteredServiceTransportModeCompatibility(t *testing.T) {
	schema := compileContractJSONSchema(t, "registered-service.schema.json")

	const common = `
		"service_id":"updater-a",
		"service_type":"update_agent",
		"service_name":"Updater A",
		"ssl_enabled":false,
		"version":"v2.0.0",
		"status":"online",
		"capabilities":{},
		"created_at":"2026-07-28T00:00:00Z",
		"updated_at":"2026-07-28T00:00:00Z"`

	tests := []struct {
		name      string
		body      string
		wantValid bool
	}{
		{
			name: "normal node response carries desired applied and reported endpoints",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"ssl_enabled":false,
				"public_url":"http://worker-a.example.com:8084",
				"desired_endpoint":{"host":"worker-a.example.com","port":9084,"ssl_enabled":false,"public_url":"http://worker-a.example.com:9084"},
				"applied_endpoint":{"host":"worker-a.example.com","port":8084,"ssl_enabled":false,"public_url":"http://worker-a.example.com:8084"},
				"reported_endpoint":{"host":"127.0.0.1","port":8084,"ssl_enabled":false,"public_url":"http://127.0.0.1:8084"},
				"endpoint_revision":2,
				"endpoint_status":"pending",
				"version":"v2.0.0",
				"status":"online",
				"capabilities":{},
				"created_at":"2026-07-28T00:00:00Z",
				"updated_at":"2026-07-28T00:00:00Z"
			}`,
			wantValid: true,
		},
		{
			name: "endpoint port is bounded",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"ssl_enabled":false,
				"public_url":"http://worker-a.example.com:8084",
				"applied_endpoint":{"host":"worker-a.example.com","port":0,"ssl_enabled":false,"public_url":"http://worker-a.example.com:8084"},
				"version":"v2.0.0",
				"status":"online",
				"capabilities":{},
				"created_at":"2026-07-28T00:00:00Z",
				"updated_at":"2026-07-28T00:00:00Z"
			}`,
			wantValid: false,
		},
		{
			name: "endpoint rejects unknown fields",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"ssl_enabled":false,
				"public_url":"http://worker-a.example.com:8084",
				"applied_endpoint":{"host":"worker-a.example.com","port":8084,"ssl_enabled":false,"public_url":"http://worker-a.example.com:8084","command":"unsafe"},
				"version":"v2.0.0",
				"status":"online",
				"capabilities":{},
				"created_at":"2026-07-28T00:00:00Z",
				"updated_at":"2026-07-28T00:00:00Z"
			}`,
			wantValid: false,
		},
		{
			name: "endpoint revision must be positive",
			body: `{
				"service_id":"worker-a",
				"service_type":"worker",
				"service_name":"Worker A",
				"ssl_enabled":false,
				"public_url":"http://worker-a.example.com:8084",
				"endpoint_revision":0,
				"version":"v2.0.0",
				"status":"online",
				"capabilities":{},
				"created_at":"2026-07-28T00:00:00Z",
				"updated_at":"2026-07-28T00:00:00Z"
			}`,
			wantValid: false,
		},
		{
			name:      "pull updater response is endpointless",
			body:      `{` + common + `,"transport_mode":"pull_v2","execution_host_id":"host-a","ownership_epoch":1}`,
			wantValid: true,
		},
		{
			name:      "pull updater response rejects advertised endpoint",
			body:      `{` + common + `,"transport_mode":"pull_v2","execution_host_id":"host-a","ownership_epoch":1,"public_url":"http://updater-a.example.com"}`,
			wantValid: false,
		},
		{
			name:      "pull updater response rejects endpoint state",
			body:      `{` + common + `,"transport_mode":"pull_v2","execution_host_id":"host-a","ownership_epoch":1,"reported_endpoint":{"host":"127.0.0.1","port":8090,"ssl_enabled":false,"public_url":"http://127.0.0.1:8090"}}`,
			wantValid: false,
		},
		{
			name:      "pull updater response rejects endpoint revision metadata",
			body:      `{` + common + `,"transport_mode":"pull_v2","execution_host_id":"host-a","ownership_epoch":1,"endpoint_revision":2,"endpoint_status":"applied"}`,
			wantValid: false,
		},
		{
			name:      "pull updater response rejects node config revision metadata",
			body:      `{` + common + `,"transport_mode":"pull_v2","execution_host_id":"host-a","ownership_epoch":1,"applied_config_revision":2,"applied_config_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
			wantValid: false,
		},
		{
			name:      "pull updater response requires ownership binding",
			body:      `{` + common + `,"transport_mode":"pull_v2"}`,
			wantValid: false,
		},
		{
			name:      "legacy updater response retains endpoint requirement",
			body:      `{` + common + `}`,
			wantValid: false,
		},
		{
			name:      "explicit ssh updater response retains endpoint requirement",
			body:      `{` + common + `,"transport_mode":"ssh_v1"}`,
			wantValid: false,
		},
		{
			name:      "legacy updater response remains valid with endpoint",
			body:      `{` + common + `,"public_url":"http://updater-a.example.com"}`,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateServiceTransportInstance(t, schema, tt.body, tt.wantValid)
		})
	}
}

func TestControlOpenAPIModelsEndpointlessPullUpdater(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)

	for _, want := range []string{
		"transport_mode:",
		"enum: [ssh_v1, pull_v2]",
		"default: ssh_v1",
		"ServiceEndpoint:",
		"desired_endpoint:",
		"applied_endpoint:",
		"reported_endpoint:",
		"endpoint_revision:",
		"endpoint_status:",
		"Existing payloads that omit transport_mode are interpreted as ssh_v1.",
		"update_agent with transport_mode pull_v2 must omit host, port, and public_url.",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing pull updater compatibility marker %q", want)
		}
	}

	registrationStart := strings.Index(raw, "    ServiceRegistrationRequest:\n")
	registrationEnd := strings.Index(raw[registrationStart+1:], "\n    Heartbeat:\n")
	if registrationStart < 0 || registrationEnd < 0 {
		t.Fatal("control-api.yaml service registration schema boundaries are missing")
	}
	registration := raw[registrationStart : registrationStart+1+registrationEnd]
	for _, forbidden := range []string{"execution_host_id:", "ownership_epoch:"} {
		if strings.Contains(registration, forbidden) {
			t.Fatalf("ServiceRegistrationRequest must not accept server-owned field %q", forbidden)
		}
	}
}
