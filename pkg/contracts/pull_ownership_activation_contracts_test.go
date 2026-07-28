package contracts

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const pullOwnershipPolicyDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func validatePullOwnershipActivationJSON(t *testing.T, schema *jsonschema.Schema, value any, wantValid bool) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&instance); err != nil {
		t.Fatal(err)
	}
	err = schema.Validate(instance)
	if wantValid && err != nil {
		t.Fatalf("expected valid pull ownership activation contract for %s: %v", body, err)
	}
	if !wantValid && err == nil {
		t.Fatalf("expected pull ownership activation contract rejection for %s", body)
	}
}

func pullOwnershipActivationRequestFixture() map[string]any {
	return map[string]any{
		"expected_execution_host_id":              "host-a",
		"expected_ownership_epoch":                int64(0),
		"expected_source_policy_revision":         int64(7),
		"expected_projection_revision":            int64(11),
		"expected_local_executor_policy_revision": int64(9),
		"expected_local_executor_policy_sha256":   pullOwnershipPolicyDigest,
	}
}

func pullOwnershipActivationResponseFixture() map[string]any {
	return map[string]any{
		"updater_id":                     "host-agent-a",
		"execution_host_id":              "host-a",
		"transport_mode":                 "pull_v2",
		"agent_service_id":               "host-agent-a",
		"ownership_epoch":                int64(1),
		"source_policy_revision":         int64(7),
		"projection_revision":            int64(11),
		"local_executor_policy_revision": int64(9),
		"local_executor_policy_sha256":   pullOwnershipPolicyDigest,
	}
}

func clonePullOwnershipFixture(t *testing.T, source map[string]any) map[string]any {
	t.Helper()
	body, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(body, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func TestPullOwnershipActivationRequestIsStrictCompareAndSwap(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-pull-ownership-activate-request.schema.json")
	valid := pullOwnershipActivationRequestFixture()
	validatePullOwnershipActivationJSON(t, schema, valid, true)

	for field := range valid {
		candidate := clonePullOwnershipFixture(t, valid)
		delete(candidate, field)
		validatePullOwnershipActivationJSON(t, schema, candidate, false)
	}
	for field, value := range map[string]any{
		"expected_execution_host_ownership_epoch": int64(0),
		"ownership_epoch":                         int64(1),
		"agent_service_id":                        "attacker",
		"runtime_token":                           "secret",
		"release_url":                             "https://attacker.example/release",
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipActivationJSON(t, schema, candidate, false)
	}

	maximumEpoch := clonePullOwnershipFixture(t, valid)
	maximumEpoch["expected_ownership_epoch"] = int64(math.MaxInt64 - 1)
	validatePullOwnershipActivationJSON(t, schema, maximumEpoch, true)
	overflowingEpoch := clonePullOwnershipFixture(t, valid)
	overflowingEpoch["expected_ownership_epoch"] = int64(math.MaxInt64)
	validatePullOwnershipActivationJSON(t, schema, overflowingEpoch, false)

	for field, value := range map[string]any{
		"expected_ownership_epoch":                int64(-1),
		"expected_source_policy_revision":         int64(0),
		"expected_projection_revision":            int64(0),
		"expected_local_executor_policy_revision": int64(0),
		"expected_local_executor_policy_sha256":   "sha256:ABCDEF",
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipActivationJSON(t, schema, candidate, false)
	}
}

func TestPullOwnershipActivationResponseIsExactAndSecretFree(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-pull-ownership-activate-response.schema.json")
	valid := pullOwnershipActivationResponseFixture()
	validatePullOwnershipActivationJSON(t, schema, valid, true)

	for field := range valid {
		candidate := clonePullOwnershipFixture(t, valid)
		delete(candidate, field)
		validatePullOwnershipActivationJSON(t, schema, candidate, false)
	}
	for field, value := range map[string]any{
		"runtime_token":  "secret",
		"mutation_grant": "secret",
		"lease_token":    "secret",
		"release_token":  "secret",
		"policy":         map[string]any{},
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipActivationJSON(t, schema, candidate, false)
	}

	wrongTransport := clonePullOwnershipFixture(t, valid)
	wrongTransport["transport_mode"] = "ssh_v1"
	validatePullOwnershipActivationJSON(t, schema, wrongTransport, false)
	zeroEpoch := clonePullOwnershipFixture(t, valid)
	zeroEpoch["ownership_epoch"] = int64(0)
	validatePullOwnershipActivationJSON(t, schema, zeroEpoch, false)
}

func TestPullOwnershipActivationGoTypesMatchExactSchemas(t *testing.T) {
	requestSchema := compileContractJSONSchema(t, "system-update-pull-ownership-activate-request.schema.json")
	responseSchema := compileContractJSONSchema(t, "system-update-pull-ownership-activate-response.schema.json")

	request := SystemUpdatePullOwnershipActivateRequest{
		ExpectedExecutionHostID:             "host-a",
		ExpectedOwnershipEpoch:              0,
		ExpectedSourcePolicyRevision:        7,
		ExpectedProjectionRevision:          11,
		ExpectedLocalExecutorPolicyRevision: 9,
		ExpectedLocalExecutorPolicySHA256:   pullOwnershipPolicyDigest,
	}
	response := SystemUpdatePullOwnershipActivateResponse{
		UpdaterID:                   "host-agent-a",
		ExecutionHostID:             "host-a",
		TransportMode:               UpdateTransportPullV2,
		AgentServiceID:              "host-agent-a",
		OwnershipEpoch:              1,
		SourcePolicyRevision:        7,
		ProjectionRevision:          11,
		LocalExecutorPolicyRevision: 9,
		LocalExecutorPolicySHA256:   pullOwnershipPolicyDigest,
	}
	validatePullOwnershipActivationJSON(t, requestSchema, request, true)
	validatePullOwnershipActivationJSON(t, responseSchema, response, true)
}

func TestControlOpenAPIDocumentsPullOwnershipActivationPathAndExactSchemas(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	const pathMarker = "  /system-updates/updaters/{id}/pull-ownership/activate:\n"
	start := strings.Index(raw, pathMarker)
	if start < 0 {
		t.Fatal("control-api.yaml is missing the pull ownership activation path")
	}
	pathSection := raw[start+len(pathMarker):]
	if end := strings.Index(pathSection, "\n  /"); end >= 0 {
		pathSection = pathSection[:end]
	}
	for _, want := range []string{
		"operationId: activateSystemUpdatePullOwnership",
		"#/components/schemas/SystemUpdatePullOwnershipActivateRequest",
		"#/components/schemas/SystemUpdatePullOwnershipActivateResponse",
		`"200":`,
		`"400":`,
		`"403":`,
		`"404":`,
		`"409":`,
		`"500":`,
		"updater_policy_revision_conflict",
		"system_update_ownership_conflict",
		"system_update_execution_host_busy",
		"host_agent_not_ready",
		"update_agent_inactive",
	} {
		if !strings.Contains(pathSection, want) {
			t.Fatalf("pull ownership activation path is missing %q", want)
		}
	}

	for _, component := range []string{
		"SystemUpdatePullOwnershipActivateRequest:",
		"SystemUpdatePullOwnershipActivateResponse:",
	} {
		componentStart := strings.Index(raw, "    "+component)
		if componentStart < 0 {
			t.Fatalf("control-api.yaml is missing component %s", component)
		}
		section := raw[componentStart:]
		if end := strings.Index(section[1:], "\n    SystemUpdate"); end >= 0 {
			section = section[:end+1]
		}
		if !strings.Contains(section, "additionalProperties: false") {
			t.Fatalf("%s is not strict", component)
		}
		if component == "SystemUpdatePullOwnershipActivateRequest:" &&
			!strings.Contains(section, "maximum: 9223372036854775806") {
			t.Fatal("activation request ownership epoch must reject int64 overflow")
		}
	}
	for _, want := range []string{
		"expected_ownership_epoch",
		"expected_source_policy_revision",
		"expected_projection_revision",
		"expected_local_executor_policy_revision",
		"expected_local_executor_policy_sha256",
		"projection_revision",
		"local_executor_policy_sha256",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing pull ownership field %q", want)
		}
	}
	if strings.Contains(pathSection, "expected_execution_host_ownership_epoch") {
		t.Fatal("deprecated epoch field leaked into the activation path contract")
	}
}
