package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHostAgentPolicyRequestCarriesNoServerOwnedBinding(t *testing.T) {
	schema := compileContractJSONSchema(t, "host-agent-policy-request.schema.json")

	for _, tt := range []struct {
		name      string
		body      string
		wantValid bool
	}{
		{
			name:      "initial policy fetch",
			body:      `{"service_id":"host-agent-a","current_revision":0}`,
			wantValid: true,
		},
		{
			name:      "negative revision",
			body:      `{"service_id":"host-agent-a","current_revision":-1}`,
			wantValid: false,
		},
		{
			name:      "invalid service identity",
			body:      `{"service_id":" host-agent-a ","current_revision":0}`,
			wantValid: false,
		},
		{
			name:      "server owned binding is rejected",
			body:      `{"service_id":"host-agent-a","current_revision":0,"execution_host_id":"host-a","ownership_epoch":1}`,
			wantValid: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			validateServiceTransportInstance(t, schema, tt.body, tt.wantValid)
		})
	}
}

func TestHostAgentPolicyResponseIsServerBoundAndEndpointAware(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"host-agent-policy-response.schema.json",
		"host-agent-self-update-directive.schema.json",
		"host-self-update-release-binding.schema.json",
	)

	const validPolicy = `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":2,
		"revision":7,
		"source_policy_revision":5,
		"local_executor_policy_revision":3,
		"local_executor_policy_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"observe_only":false,
		"targets":[{
			"service_id":"worker-a",
			"service_type":"worker",
			"deployment_mode":"systemd",
			"desired_endpoint":{"host":"worker.example.com","port":18084,"ssl_enabled":false,"public_url":"http://worker.example.com:18084"},
			"applied_endpoint":{"host":"worker.example.com","port":8084,"ssl_enabled":false,"public_url":"http://worker.example.com:8084"},
			"local_listen_endpoint":{"host":"127.0.0.1","port":8084,"ssl_enabled":false,"public_url":"http://127.0.0.1:8084"},
			"local_health_endpoint":{"host":"127.0.0.1","port":8084,"ssl_enabled":false,"public_url":"http://127.0.0.1:8084/health"}
		}]
	}`

	validateServiceTransportInstance(t, schema, validPolicy, true)
	validateServiceTransportInstance(t, schema, `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":1,
		"revision":1,
		"source_policy_revision":1,
		"local_executor_policy_revision":1,
		"local_executor_policy_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"observe_only":false,
		"targets":[{
			"service_id":"worker-a",
			"service_type":"worker",
			"deployment_mode":"docker"
		}]
	}`, true)

	var document map[string]any
	if err := json.Unmarshal([]byte(validPolicy), &document); err != nil {
		t.Fatal(err)
	}

	document["transport_mode"] = "ssh_v1"
	assertHostAgentPolicyDocument(t, schema, document, false)
	document["transport_mode"] = "pull_v2"

	document["observe_only"] = true
	assertHostAgentPolicyDocument(t, schema, document, false)
	document["observe_only"] = false

	target := document["targets"].([]any)[0].(map[string]any)
	target["service_type"] = "update_agent"
	assertHostAgentPolicyDocument(t, schema, document, false)
	target["service_type"] = "worker"

	endpoint := target["local_listen_endpoint"].(map[string]any)
	endpoint["port"] = float64(0)
	assertHostAgentPolicyDocument(t, schema, document, false)
	endpoint["port"] = float64(8084)

	target["command"] = "unsafe"
	assertHostAgentPolicyDocument(t, schema, document, false)
}

func TestHostAgentPolicyGoTypesMatchSchemas(t *testing.T) {
	requestSchema := compileContractJSONSchema(t, "host-agent-policy-request.schema.json")
	responseSchema := compileContractJSONSchema(
		t,
		"host-agent-policy-response.schema.json",
		"host-agent-self-update-directive.schema.json",
		"host-self-update-release-binding.schema.json",
	)

	requestBody, err := json.Marshal(HostAgentPolicyRequest{
		ServiceID:       "host-agent-a",
		CurrentRevision: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	var request any
	if err := json.Unmarshal(requestBody, &request); err != nil {
		t.Fatal(err)
	}
	if err := requestSchema.Validate(request); err != nil {
		t.Fatalf("HostAgentPolicyRequest type does not match schema: %v", err)
	}

	endpoint := &ServiceEndpoint{
		Host:       "127.0.0.1",
		Port:       8084,
		SSLEnabled: false,
		PublicURL:  "http://127.0.0.1:8084",
	}
	responseBody, err := json.Marshal(HostAgentPolicyResponse{
		ServiceID:                   "host-agent-a",
		TransportMode:               UpdateTransportPullV2,
		ExecutionHostID:             "host-a",
		OwnershipEpoch:              1,
		Revision:                    1,
		SourcePolicyRevision:        1,
		LocalExecutorPolicyRevision: 1,
		LocalExecutorPolicySHA256:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ObserveOnly:                 false,
		RuntimeRequirement: &HostAgentRuntimeRequirement{
			MinimumAgentVersion:     "v1.8.0",
			MinimumExecutorVersion:  "v1.8.0",
			AgentProtocolVersion:    2,
			ExecutorProtocolVersion: 2,
			MutationProtocolVersion: 2,
			RecoveryProtocolVersion: 2,
		},
		SelfUpdate: &HostAgentSelfUpdateDirective{
			Generation:              "11111111-1111-4111-8111-111111111111",
			AgentVersion:            "v1.8.0",
			ExecutorVersion:         "v1.8.0",
			Commit:                  hostSelfUpdateCommit,
			ArtifactSHA256:          "sha256:" + hostSelfUpdateSHA256,
			AgentProtocolVersion:    2,
			ExecutorProtocolVersion: 2,
			MutationProtocolVersion: 2,
			RecoveryProtocolVersion: 2,
			Release:                 validHostSelfUpdateReleaseBinding(),
			StagedAt:                time.Date(2026, 7, 28, 0, 2, 0, 0, time.UTC),
		},
		SelfUpdateID:       "self-update-1",
		SelfUpdateRevision: 3,
		SelfUpdateStatus:   "staging",
		Targets: []HostAgentPolicyTarget{{
			ServiceID:           "worker-a",
			ServiceType:         "worker",
			DeploymentMode:      "systemd",
			DesiredEndpoint:     endpoint,
			AppliedEndpoint:     endpoint,
			LocalListenEndpoint: endpoint,
			LocalHealthEndpoint: endpoint,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		t.Fatal(err)
	}
	if err := responseSchema.Validate(response); err != nil {
		t.Fatalf("HostAgentPolicyResponse type does not match schema: %v", err)
	}
	selfUpdate := response.(map[string]any)["self_update"].(map[string]any)
	release := selfUpdate["release"].(map[string]any)
	if got := release["published_at"]; got != time.Date(
		2026, 7, 28, 0, 0, 0, 0, time.UTC,
	).Format(time.RFC3339) {
		t.Fatalf("published_at=%v", got)
	}
}

func assertHostAgentPolicyDocument(t *testing.T, schema interface{ Validate(any) error }, document any, wantValid bool) {
	t.Helper()
	err := schema.Validate(document)
	if wantValid && err != nil {
		t.Fatalf("expected valid host-agent policy, got %v", err)
	}
	if !wantValid && err == nil {
		t.Fatal("expected invalid host-agent policy")
	}
}

func TestControlOpenAPIIncludesDedicatedHostAgentPolicy(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"/services/host-agent/policy:",
		"operationId: fetchHostAgentPolicy",
		"#/components/schemas/HostAgentPolicyRequest",
		"#/components/schemas/HostAgentPolicyResponse",
		"HostAgentPolicyTarget:",
		"local_listen_endpoint:",
		"local_health_endpoint:",
		"Server-owned host binding fields are never accepted from the request.",
		"Cache-Control",
		"no-store",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing host-agent policy marker %q", want)
		}
	}
}
