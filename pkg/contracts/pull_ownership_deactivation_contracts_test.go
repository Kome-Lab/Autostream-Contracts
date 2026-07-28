package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func validatePullOwnershipDeactivationJSON(
	t *testing.T,
	schema *jsonschema.Schema,
	value any,
	wantValid bool,
) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	if err := json.Unmarshal(body, &instance); err != nil {
		t.Fatal(err)
	}
	err = schema.Validate(instance)
	if wantValid && err != nil {
		t.Fatalf("expected valid pull ownership deactivation contract for %s: %v", body, err)
	}
	if !wantValid && err == nil {
		t.Fatalf("expected pull ownership deactivation contract rejection for %s", body)
	}
}

func pullOwnershipDeactivationRequestFixture() map[string]any {
	return map[string]any{
		"expected_execution_host_id":              "host-a",
		"expected_ownership_epoch":                int64(13),
		"expected_source_policy_revision":         int64(7),
		"expected_projection_revision":            int64(11),
		"expected_local_executor_policy_revision": int64(9),
		"expected_local_executor_policy_sha256":   pullOwnershipPolicyDigest,
	}
}

func pullOwnershipDeactivationResponseFixture() map[string]any {
	return map[string]any{
		"updater_id":                     "host-agent-a",
		"execution_host_id":              "host-a",
		"transport_mode":                 "ssh_v1",
		"agent_service_id":               "legacy-updater-a",
		"ownership_epoch":                int64(14),
		"agent_ownership_epoch":          int64(0),
		"source_policy_revision":         int64(7),
		"projection_revision":            int64(11),
		"local_executor_policy_revision": int64(9),
		"local_executor_policy_sha256":   pullOwnershipPolicyDigest,
	}
}

func TestPullOwnershipDeactivationRequestIsStrictCompareAndSwap(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"system-update-pull-ownership-deactivate-request.schema.json",
	)
	valid := pullOwnershipDeactivationRequestFixture()
	validatePullOwnershipDeactivationJSON(t, schema, valid, true)

	for field := range valid {
		candidate := clonePullOwnershipFixture(t, valid)
		delete(candidate, field)
		validatePullOwnershipDeactivationJSON(t, schema, candidate, false)
	}
	for field, value := range map[string]any{
		"legacy_agent_service_id":                 "attacker-selected-owner",
		"agent_service_id":                        "attacker-selected-owner",
		"transport_mode":                          "ssh_v1",
		"runtime_token":                           "secret",
		"mutation_grant":                          "secret",
		"expected_execution_host_ownership_epoch": int64(13),
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipDeactivationJSON(t, schema, candidate, false)
	}

	for field, value := range map[string]any{
		"expected_ownership_epoch":                int64(0),
		"expected_source_policy_revision":         int64(0),
		"expected_projection_revision":            int64(0),
		"expected_local_executor_policy_revision": int64(0),
		"expected_local_executor_policy_sha256":   "sha256:ABCDEF",
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipDeactivationJSON(t, schema, candidate, false)
	}
}

func TestPullOwnershipDeactivationResponseIsExactAndSecretFree(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"system-update-pull-ownership-deactivate-response.schema.json",
	)
	valid := pullOwnershipDeactivationResponseFixture()
	validatePullOwnershipDeactivationJSON(t, schema, valid, true)

	for field := range valid {
		candidate := clonePullOwnershipFixture(t, valid)
		delete(candidate, field)
		validatePullOwnershipDeactivationJSON(t, schema, candidate, false)
	}
	for field, value := range map[string]any{
		"legacy_agent_service_id": "legacy-updater-a",
		"runtime_token":           "secret",
		"mutation_grant":          "secret",
		"lease_token":             "secret",
		"release_token":           "secret",
		"policy":                  map[string]any{},
	} {
		candidate := clonePullOwnershipFixture(t, valid)
		candidate[field] = value
		validatePullOwnershipDeactivationJSON(t, schema, candidate, false)
	}

	wrongTransport := clonePullOwnershipFixture(t, valid)
	wrongTransport["transport_mode"] = "pull_v2"
	validatePullOwnershipDeactivationJSON(t, schema, wrongTransport, false)
	activeAgent := clonePullOwnershipFixture(t, valid)
	activeAgent["agent_ownership_epoch"] = int64(14)
	validatePullOwnershipDeactivationJSON(t, schema, activeAgent, false)
	zeroHostEpoch := clonePullOwnershipFixture(t, valid)
	zeroHostEpoch["ownership_epoch"] = int64(0)
	validatePullOwnershipDeactivationJSON(t, schema, zeroHostEpoch, false)
}

func TestPullOwnershipDeactivationGoTypesMatchExactSchemas(t *testing.T) {
	requestSchema := compileContractJSONSchema(
		t,
		"system-update-pull-ownership-deactivate-request.schema.json",
	)
	responseSchema := compileContractJSONSchema(
		t,
		"system-update-pull-ownership-deactivate-response.schema.json",
	)

	request := SystemUpdatePullOwnershipDeactivateRequest{
		ExpectedExecutionHostID:             "host-a",
		ExpectedOwnershipEpoch:              13,
		ExpectedSourcePolicyRevision:        7,
		ExpectedProjectionRevision:          11,
		ExpectedLocalExecutorPolicyRevision: 9,
		ExpectedLocalExecutorPolicySHA256:   pullOwnershipPolicyDigest,
	}
	response := SystemUpdatePullOwnershipDeactivateResponse{
		UpdaterID:                   "host-agent-a",
		ExecutionHostID:             "host-a",
		TransportMode:               UpdateTransportSSHV1,
		AgentServiceID:              "legacy-updater-a",
		OwnershipEpoch:              14,
		AgentOwnershipEpoch:         0,
		SourcePolicyRevision:        7,
		ProjectionRevision:          11,
		LocalExecutorPolicyRevision: 9,
		LocalExecutorPolicySHA256:   pullOwnershipPolicyDigest,
	}
	validatePullOwnershipDeactivationJSON(t, requestSchema, request, true)
	validatePullOwnershipDeactivationJSON(t, responseSchema, response, true)
}

func TestControlOpenAPIDocumentsPullOwnershipDeactivationPathAndExactSchemas(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	const pathMarker = "  /system-updates/updaters/{id}/pull-ownership/deactivate:\n"
	start := strings.Index(raw, pathMarker)
	if start < 0 {
		t.Fatal("control-api.yaml is missing the pull ownership deactivation path")
	}
	pathSection := raw[start+len(pathMarker):]
	if end := strings.Index(pathSection, "\n  /"); end >= 0 {
		pathSection = pathSection[:end]
	}
	for _, want := range []string{
		"operationId: deactivateSystemUpdatePullOwnership",
		"#/components/schemas/SystemUpdatePullOwnershipDeactivateRequest",
		"#/components/schemas/SystemUpdatePullOwnershipDeactivateResponse",
		`"200":`,
		`"400":`,
		`"403":`,
		`"404":`,
		`"409":`,
		`"500":`,
		"updater_policy_revision_conflict",
		"system_update_ownership_conflict",
		"host_lifecycle_busy",
		"host_agent_not_ready",
		"update_agent_inactive",
	} {
		if !strings.Contains(pathSection, want) {
			t.Fatalf("pull ownership deactivation path is missing %q", want)
		}
	}

	for _, component := range []string{
		"SystemUpdatePullOwnershipDeactivateRequest:",
		"SystemUpdatePullOwnershipDeactivateResponse:",
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
	}
	if strings.Contains(pathSection, "legacy_agent_service_id") {
		t.Fatal("client-selected legacy owner leaked into the deactivation request contract")
	}
}
