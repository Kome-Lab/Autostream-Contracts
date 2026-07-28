package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	hostSelfUpdateSHA256       = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hostSelfUpdatePolicySHA256 = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hostSelfUpdateCommit       = "cccccccccccccccccccccccccccccccccccccccc"
)

func TestHostSelfUpdateRequestSchemasAreStrictAndBounded(t *testing.T) {
	create := compileContractJSONSchema(t, "host-self-update-create-request.schema.json")
	retry := compileContractJSONSchema(t, "host-self-update-retry-request.schema.json")
	cancel := compileContractJSONSchema(t, "host-self-update-cancel-request.schema.json")

	validateServiceTransportInstance(t, create,
		`{"target_version":"v1.8.0","idempotency_key":"host-a-v1.8.0"}`, true)
	validateServiceTransportInstance(t, create,
		`{"target_version":"v1.8.0","idempotency_key":"host-a-v1.8.0","host_id":"host-a"}`, false)
	validateServiceTransportInstance(t, create,
		`{"target_version":"latest","idempotency_key":"host-a-v1.8.0"}`, false)
	validateServiceTransportInstance(t, create,
		`{"target_version":"v1.8.0","idempotency_key":"`+strings.Repeat("a", 129)+`"}`, false)

	validateServiceTransportInstance(t, retry,
		`{"idempotency_key":"host-a-v1.8.0-retry-1"}`, true)
	validateServiceTransportInstance(t, retry,
		`{"idempotency_key":"host-a-v1.8.0-retry-1","target_version":"v1.8.1"}`, false)

	validateServiceTransportInstance(t, cancel, `{"expected_revision":3}`, true)
	validateServiceTransportInstance(t, cancel, `{"expected_revision":0}`, false)
	validateServiceTransportInstance(t, cancel,
		`{"expected_revision":3,"force":true}`, false)
}

func TestHostSelfUpdateLifecycleSchemaHasExactStatusAndNoSecretOrURL(t *testing.T) {
	schema := compileContractJSONSchema(t,
		"host-self-update.schema.json",
		"host-self-update-release.schema.json",
	)
	valid := validHostSelfUpdateJSON("staging")
	validateServiceTransportInstance(t, schema, valid, true)

	for _, status := range []string{
		"queued", "staging", "activating", "verifying", "rolling_back",
		"cancel_requested", "succeeded", "rolled_back", "failed", "canceled",
	} {
		validateServiceTransportInstance(
			t, schema, validHostSelfUpdateJSON(status), true,
		)
	}
	validateServiceTransportInstance(
		t, schema, validHostSelfUpdateJSON("installing"), false,
	)
	validateServiceTransportInstance(t, schema,
		strings.Replace(valid, `"updated_at":`, `"token":"secret","updated_at":`, 1),
		false,
	)
	validateServiceTransportInstance(t, schema,
		strings.Replace(valid, `"updated_at":`, `"download_url":"https://example.invalid/archive","updated_at":`, 1),
		false,
	)

	for _, file := range []string{
		"host-self-update-release.schema.json",
		"host-self-update-release-binding.schema.json",
		"host-agent-self-update-directive.schema.json",
		"host-self-update.schema.json",
		"host-agent-policy-response.schema.json",
		"host-self-update-grant.schema.json",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := strings.ToLower(string(body))
		for _, forbidden := range []string{
			`"url"`, `"download_url"`, `"token"`, `"github_token"`,
		} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("%s exposes forbidden durable field %s", file, forbidden)
			}
		}
	}
}

func TestHostSelfUpdateReleaseBindingIsExactAndCredentialFree(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"host-self-update-release-binding.schema.json",
	)
	valid := validHostSelfUpdateReleaseBindingJSON()
	validateServiceTransportInstance(t, schema, valid, true)

	var binding map[string]any
	if err := json.Unmarshal([]byte(valid), &binding); err != nil {
		t.Fatal(err)
	}
	requiredFields := []string{
		"tag", "commit", "published_at",
		"manifest_asset_id", "manifest_asset_name", "manifest_sha256",
		"manifest_checksum_asset_id", "manifest_checksum_sha256",
		"archive_asset_id", "archive_asset_name", "archive_size", "archive_sha256",
		"archive_checksum_asset_id", "archive_checksum_sha256",
		"arch", "agent_protocol_version", "executor_protocol_version",
		"mutation_protocol_version", "recovery_protocol_version",
		"minimum_panel_version",
	}
	for _, field := range requiredFields {
		t.Run("missing_"+field, func(t *testing.T) {
			candidate := cloneJSONDocument(t, binding)
			delete(candidate, field)
			assertHostAgentPolicyDocument(t, schema, candidate, false)
		})
	}

	for _, field := range []string{
		"url", "download_url", "token", "github_token",
		"attestation_verified_at", "manifest_checksum_asset_name",
		"archive_checksum_asset_name",
	} {
		t.Run("forbidden_"+field, func(t *testing.T) {
			candidate := cloneJSONDocument(t, binding)
			candidate[field] = "must-not-cross-the-boundary"
			assertHostAgentPolicyDocument(t, schema, candidate, false)
		})
	}

	for _, field := range []string{
		"manifest_sha256",
		"manifest_checksum_sha256",
		"archive_sha256",
		"archive_checksum_sha256",
	} {
		t.Run("prefixed_"+field, func(t *testing.T) {
			candidate := cloneJSONDocument(t, binding)
			candidate[field] = "sha256:" + hostSelfUpdateSHA256
			assertHostAgentPolicyDocument(t, schema, candidate, false)
		})
	}

	fieldMismatches := map[string]any{
		"tag":                       "latest",
		"commit":                    strings.Repeat("A", 40),
		"published_at":              "2026-07-28T09:00:00+09:00",
		"manifest_asset_id":         float64(0),
		"manifest_asset_name":       "other-manifest.json",
		"archive_asset_id":          float64(0),
		"archive_asset_name":        "host-agent.tar.gz",
		"archive_size":              float64(268435457),
		"arch":                      "386",
		"agent_protocol_version":    float64(0),
		"executor_protocol_version": float64(0),
		"mutation_protocol_version": float64(0),
		"recovery_protocol_version": float64(1),
		"minimum_panel_version":     "v1",
	}
	for field, mismatched := range fieldMismatches {
		t.Run("mismatched_"+field, func(t *testing.T) {
			candidate := cloneJSONDocument(t, binding)
			candidate[field] = mismatched
			assertHostAgentPolicyDocument(t, schema, candidate, false)
		})
	}
}

func TestHostSelfUpdateReleaseBindingGoTypeHasExactWireFields(t *testing.T) {
	want := map[string]struct{}{
		"tag": {}, "commit": {}, "published_at": {},
		"manifest_asset_id": {}, "manifest_asset_name": {}, "manifest_sha256": {},
		"manifest_checksum_asset_id": {}, "manifest_checksum_sha256": {},
		"archive_asset_id": {}, "archive_asset_name": {}, "archive_size": {},
		"archive_sha256": {}, "archive_checksum_asset_id": {},
		"archive_checksum_sha256": {}, "arch": {},
		"agent_protocol_version": {}, "executor_protocol_version": {},
		"mutation_protocol_version": {}, "recovery_protocol_version": {},
		"minimum_panel_version": {},
	}
	typ := reflect.TypeOf(HostSelfUpdateReleaseBinding{})
	if typ.NumField() != len(want) {
		t.Fatalf(
			"HostSelfUpdateReleaseBinding has %d fields, want %d",
			typ.NumField(), len(want),
		)
	}
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if _, ok := want[tag]; !ok {
			t.Fatalf(
				"HostSelfUpdateReleaseBinding exposes unexpected field %q",
				tag,
			)
		}
		delete(want, tag)
	}
	if len(want) != 0 {
		t.Fatalf("HostSelfUpdateReleaseBinding is missing fields %#v", want)
	}
}

func TestHostSelfUpdateGrantSchemasRequireExactBinding(t *testing.T) {
	issue := compileContractJSONSchema(t,
		"host-self-update-grant-issue-request.schema.json",
	)
	grantDependencies := []string{
		"host-self-update-grant.schema.json",
		"host-self-update-release-binding.schema.json",
	}
	issued := compileContractJSONSchema(t,
		"host-self-update-grant-issue-response.schema.json",
		grantDependencies...,
	)
	consume := compileContractJSONSchema(t,
		"host-self-update-grant-consume-request.schema.json",
		grantDependencies...,
	)
	consumed := compileContractJSONSchema(t,
		"host-self-update-grant-consume-response.schema.json",
		grantDependencies...,
	)

	validateServiceTransportInstance(t, issue, `{
		"expected_revision":3,
		"operation":"stage",
		"plan_sha256":"`+hostSelfUpdateSHA256+`",
		"session_id":"session-host-self-update-0001"
	}`, true)
	validateServiceTransportInstance(t, issue, `{
		"expected_revision":3,
		"operation":"apply",
		"plan_sha256":"`+hostSelfUpdateSHA256+`",
		"session_id":"session-host-self-update-0001"
	}`, false)

	grant := validHostSelfUpdateGrantJSON()
	validateServiceTransportInstance(t, issued,
		`{"grant":`+grant+`,"token":"`+strings.Repeat("A", 43)+`","issued":true}`,
		true,
	)
	validateServiceTransportInstance(t, consume,
		`{"token":"`+strings.Repeat("A", 43)+`","binding":`+grant+`}`,
		true,
	)
	consumedGrant := validConsumedHostSelfUpdateGrantJSON("stage")
	validateServiceTransportInstance(t, consumed,
		`{"grant":`+consumedGrant+`,"consumed":true}`,
		true,
	)
	validateServiceTransportInstance(t, consumed,
		`{"grant":`+consumedGrant+`,"consumed":false}`,
		true,
	)
	validateServiceTransportInstance(t, issued,
		`{"grant":`+consumedGrant+`,"issued":false}`,
		false,
	)
	validateServiceTransportInstance(t, consume,
		`{"token":"`+strings.Repeat("A", 43)+`","binding":`+consumedGrant+`}`,
		false,
	)
	validateServiceTransportInstance(t, consumed,
		`{"grant":`+grant+`,"consumed":true}`,
		false,
	)

	var binding map[string]any
	if err := json.Unmarshal([]byte(grant), &binding); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"id", "self_update_id", "attempt_generation", "operation",
		"execution_host_id", "agent_service_id",
		"expected_self_update_revision", "expected_ownership_epoch",
		"expected_source_policy_revision", "expected_projection_revision",
		"expected_local_executor_policy_revision",
		"expected_local_executor_policy_sha256", "agent_version",
		"executor_version", "release_commit", "artifact_sha256",
		"agent_protocol_version", "executor_protocol_version",
		"mutation_protocol_version", "recovery_protocol_version",
		"release",
		"directive_issued_at", "plan_sha256", "session_id", "revision",
		"issued_at", "expires_at", "created_at", "updated_at",
	} {
		candidate := make(map[string]any, len(binding))
		for key, value := range binding {
			candidate[key] = value
		}
		delete(candidate, required)
		body, err := json.Marshal(map[string]any{
			"token":   strings.Repeat("A", 43),
			"binding": candidate,
		})
		if err != nil {
			t.Fatal(err)
		}
		validateServiceTransportInstance(t, consume, string(body), false)
	}
}

func TestHostSelfUpdateGrantStageClaimReceiptIsConditional(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"host-self-update-grant.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	var unconsumedStage map[string]any
	if err := json.Unmarshal(
		[]byte(validHostSelfUpdateGrantJSON()),
		&unconsumedStage,
	); err != nil {
		t.Fatal(err)
	}
	assertHostAgentPolicyDocument(t, schema, unconsumedStage, true)

	var consumedStage map[string]any
	if err := json.Unmarshal(
		[]byte(validConsumedHostSelfUpdateGrantJSON("stage")),
		&consumedStage,
	); err != nil {
		t.Fatal(err)
	}
	assertHostAgentPolicyDocument(t, schema, consumedStage, true)
	for _, required := range []string{
		"stage_claim_revision",
		"stage_claimed_at",
	} {
		candidate := cloneJSONDocument(t, consumedStage)
		delete(candidate, required)
		assertHostAgentPolicyDocument(t, schema, candidate, false)
	}
	zeroRevision := cloneJSONDocument(t, consumedStage)
	zeroRevision["stage_claim_revision"] = float64(0)
	assertHostAgentPolicyDocument(t, schema, zeroRevision, false)

	unconsumedWithClaim := cloneJSONDocument(t, unconsumedStage)
	unconsumedWithClaim["stage_claim_revision"] = float64(4)
	unconsumedWithClaim["stage_claimed_at"] = "2026-07-28T00:03:30Z"
	assertHostAgentPolicyDocument(t, schema, unconsumedWithClaim, false)

	reconcile := cloneJSONDocument(t, unconsumedStage)
	reconcile["operation"] = "reconcile"
	assertHostAgentPolicyDocument(t, schema, reconcile, true)

	var consumedReconcile map[string]any
	if err := json.Unmarshal(
		[]byte(validConsumedHostSelfUpdateGrantJSON("reconcile")),
		&consumedReconcile,
	); err != nil {
		t.Fatal(err)
	}
	assertHostAgentPolicyDocument(t, schema, consumedReconcile, true)
	for _, forbidden := range []string{
		"stage_claim_revision",
		"stage_claimed_at",
	} {
		candidate := cloneJSONDocument(t, consumedReconcile)
		if forbidden == "stage_claim_revision" {
			candidate[forbidden] = float64(4)
		} else {
			candidate[forbidden] = "2026-07-28T00:03:30Z"
		}
		assertHostAgentPolicyDocument(t, schema, candidate, false)
	}
}

func TestHostSelfUpdateGrantGoTypeMatchesReleaseBoundSchema(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"host-self-update-grant.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	issuedAt := time.Date(2026, 7, 28, 0, 3, 0, 0, time.UTC)
	grant := HostSelfUpdateGrant{
		ID:                                  "grant-1",
		SelfUpdateID:                        "self-update-1",
		AttemptGeneration:                   "11111111-1111-4111-8111-111111111111",
		Operation:                           "stage",
		ExecutionHostID:                     "host-a",
		AgentServiceID:                      "host-agent-a",
		ExpectedSelfUpdateRevision:          3,
		ExpectedOwnershipEpoch:              2,
		ExpectedSourcePolicyRevision:        5,
		ExpectedProjectionRevision:          7,
		ExpectedLocalExecutorPolicyRevision: 3,
		ExpectedLocalExecutorPolicySHA256:   hostSelfUpdatePolicySHA256,
		AgentVersion:                        "v1.8.0",
		ExecutorVersion:                     "v1.8.0",
		ReleaseCommit:                       hostSelfUpdateCommit,
		ArtifactSHA256:                      "sha256:" + hostSelfUpdateSHA256,
		AgentProtocolVersion:                2,
		ExecutorProtocolVersion:             2,
		MutationProtocolVersion:             2,
		RecoveryProtocolVersion:             2,
		Release:                             validHostSelfUpdateReleaseBinding(),
		DirectiveIssuedAt:                   issuedAt.Add(-time.Minute),
		PlanSHA256:                          hostSelfUpdateSHA256,
		SessionID:                           "session-host-self-update-0001",
		Revision:                            1,
		IssuedAt:                            issuedAt,
		ExpiresAt:                           issuedAt.Add(90 * time.Second),
		CreatedAt:                           issuedAt,
		UpdatedAt:                           issuedAt,
	}
	body, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	validateServiceTransportInstance(t, schema, string(body), true)

	consumedAt := issuedAt.Add(30 * time.Second)
	grant.ConsumedAt = &consumedAt
	grant.StageClaimRevision = grant.ExpectedSelfUpdateRevision + 1
	grant.StageClaimedAt = &consumedAt
	body, err = json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	validateServiceTransportInstance(t, schema, string(body), true)
}

func TestHostAgentPolicySelfUpdateOverlayIsAllOrNothing(t *testing.T) {
	schema := compileContractJSONSchema(
		t,
		"host-agent-policy-response.schema.json",
		"host-agent-self-update-directive.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	valid := validHostAgentSelfUpdatePolicyJSON()
	validateServiceTransportInstance(t, schema, valid, true)

	var document map[string]any
	if err := json.Unmarshal([]byte(valid), &document); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"self_update_id", "self_update_revision", "self_update_status",
		"runtime_requirement", "self_update",
	} {
		candidate := make(map[string]any, len(document))
		for key, value := range document {
			candidate[key] = value
		}
		delete(candidate, field)
		assertHostAgentPolicyDocument(t, schema, candidate, false)
	}
	for _, field := range []string{"release", "staged_at"} {
		candidate := cloneJSONDocument(t, document)
		selfUpdate := candidate["self_update"].(map[string]any)
		delete(selfUpdate, field)
		assertHostAgentPolicyDocument(t, schema, candidate, false)
	}
	offsetStagedAt := cloneJSONDocument(t, document)
	offsetStagedAt["self_update"].(map[string]any)["staged_at"] =
		"2026-07-28T09:02:00+09:00"
	assertHostAgentPolicyDocument(t, schema, offsetStagedAt, false)
}

func TestControlOpenAPIDocumentsDedicatedHostSelfUpdatePlane(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	normalizedOpenAPI := strings.ReplaceAll(raw, "\r\n", "\n")
	for _, marker := range []string{
		"/system-updates/host-self-updates:",
		"/system-updates/hosts/{host_id}/self-updates:",
		"/system-updates/host-self-updates/{id}/retry:",
		"/system-updates/host-self-updates/{id}/cancel:",
		"/services/host-agent/self-updates/{id}/grants:",
		"/services/host-agent/self-update-grants/consume:",
		"HostSelfUpdateCreateRequest:",
		"HostSelfUpdateRelease:",
		"HostSelfUpdate:",
		"HostSelfUpdateGrant:",
		"HostSelfUpdateGrantIssueRequest:",
		"HostSelfUpdateGrantConsumeRequest:",
		"HostAgentRuntimeRequirement:",
		"HostAgentSelfUpdateDirective:",
		"HostSelfUpdateReleaseBinding:",
	} {
		if !strings.Contains(raw, marker) {
			t.Fatalf("control-api.yaml is missing host self-update marker %q", marker)
		}
	}

	start := strings.Index(raw, "    HostSelfUpdateReleaseBinding:")
	endOffset := strings.Index(raw[start+1:], "\n    ServiceEndpoint:")
	if start < 0 || endOffset < 0 {
		t.Fatal("control-api.yaml has no bounded HostSelfUpdateReleaseBinding component")
	}
	releaseBinding := raw[start : start+1+endOffset]
	const releaseBindingSchemaRef = `$ref: "../schemas/host-self-update-release-binding.schema.json"`
	if strings.Count(releaseBinding, releaseBindingSchemaRef) != 1 ||
		strings.Contains(releaseBinding, "properties:") ||
		strings.Contains(releaseBinding, "additionalProperties:") {
		t.Fatal(
			"OpenAPI HostSelfUpdateReleaseBinding must use only the canonical JSON Schema",
		)
	}
	grantStart := strings.Index(raw, "    HostSelfUpdateGrant:")
	grantEndOffset := strings.Index(
		raw[grantStart+1:],
		"\n    HostSelfUpdateGrantIssueRequest:",
	)
	if grantStart < 0 || grantEndOffset < 0 {
		t.Fatal("control-api.yaml has no bounded HostSelfUpdateGrant component")
	}
	grantComponent := raw[grantStart : grantStart+1+grantEndOffset]
	const grantSchemaRef = `$ref: "../schemas/host-self-update-grant.schema.json"`
	if strings.Count(grantComponent, grantSchemaRef) != 1 ||
		strings.Contains(grantComponent, "properties:") ||
		strings.Contains(grantComponent, "additionalProperties:") {
		t.Fatal(
			"OpenAPI HostSelfUpdateGrant must use only the canonical JSON Schema",
		)
	}
	for component, schemaFile := range map[string]string{
		"HostAgentSelfUpdateDirective":       "host-agent-self-update-directive.schema.json",
		"HostSelfUpdateGrantIssueResponse":   "host-self-update-grant-issue-response.schema.json",
		"HostSelfUpdateGrantConsumeRequest":  "host-self-update-grant-consume-request.schema.json",
		"HostSelfUpdateGrantConsumeResponse": "host-self-update-grant-consume-response.schema.json",
	} {
		want := "    " + component + ":\n" +
			`      $ref: "../schemas/` + schemaFile + `"`
		if !strings.Contains(normalizedOpenAPI, want) {
			t.Fatalf(
				"OpenAPI %s must use canonical schema %s",
				component,
				schemaFile,
			)
		}
	}
}

func TestHostSelfUpdateOpenAPIExternalSchemasCompile(t *testing.T) {
	compileContractJSONSchema(
		t,
		"host-agent-self-update-directive.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	compileContractJSONSchema(
		t,
		"host-self-update-grant.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	for _, schemaName := range []string{
		"host-self-update-grant-issue-response.schema.json",
		"host-self-update-grant-consume-request.schema.json",
		"host-self-update-grant-consume-response.schema.json",
	} {
		compileContractJSONSchema(
			t,
			schemaName,
			"host-self-update-grant.schema.json",
			"host-self-update-release-binding.schema.json",
		)
	}
}

func validHostSelfUpdateJSON(status string) string {
	return `{
		"id":"self-update-1",
		"execution_host_id":"host-a",
		"agent_service_id":"host-agent-a",
		"target_version":"v1.8.0",
		"status":"` + status + `",
		"revision":3,
		"idempotency_key":"host-a-v1.8.0",
		"requested_by":"admin",
		"attempt_generation":"11111111-1111-4111-8111-111111111111",
		"expected_ownership_epoch":2,
		"expected_source_policy_revision":5,
		"expected_projection_revision":7,
		"expected_local_executor_policy_revision":3,
		"expected_local_executor_policy_sha256":"` + hostSelfUpdatePolicySHA256 + `",
		"previous_agent_version":"v1.7.9",
		"previous_executor_version":"v1.7.9",
		"previous_agent_protocol_version":2,
		"previous_executor_protocol_version":2,
		"previous_mutation_protocol_version":2,
		"previous_recovery_protocol_version":1,
		"release":{
			"tag":"v1.8.0",
			"commit":"` + hostSelfUpdateCommit + `",
			"published_at":"2026-07-28T00:00:00Z",
			"manifest_asset_id":101,
			"manifest_asset_name":"host-agent-manifest.json",
			"manifest_sha256":"` + hostSelfUpdateSHA256 + `",
			"manifest_checksum_asset_id":102,
			"manifest_checksum_sha256":"` + hostSelfUpdateSHA256 + `",
			"archive_asset_id":103,
			"archive_asset_name":"autostream-host-agent_v1.8.0_linux_amd64.tar.gz",
			"archive_size":4096,
			"archive_sha256":"` + hostSelfUpdateSHA256 + `",
			"archive_checksum_asset_id":104,
			"archive_checksum_sha256":"` + hostSelfUpdateSHA256 + `",
			"arch":"amd64",
			"agent_protocol_version":2,
			"executor_protocol_version":2,
			"mutation_protocol_version":2,
			"recovery_protocol_version":2,
			"minimum_panel_version":"v1.8.0",
			"attestation_verified_at":"2026-07-28T00:01:00Z"
		},
		"issued_at":"2026-07-28T00:02:00Z",
		"observation_state":"known",
		"created_at":"2026-07-28T00:02:00Z",
		"updated_at":"2026-07-28T00:02:00Z"
	}`
}

func validHostSelfUpdateGrantJSON() string {
	return `{
		"id":"grant-1",
		"self_update_id":"self-update-1",
		"attempt_generation":"11111111-1111-4111-8111-111111111111",
		"operation":"stage",
		"execution_host_id":"host-a",
		"agent_service_id":"host-agent-a",
		"expected_self_update_revision":3,
		"expected_ownership_epoch":2,
		"expected_source_policy_revision":5,
		"expected_projection_revision":7,
		"expected_local_executor_policy_revision":3,
		"expected_local_executor_policy_sha256":"` + hostSelfUpdatePolicySHA256 + `",
		"agent_version":"v1.8.0",
		"executor_version":"v1.8.0",
		"release_commit":"` + hostSelfUpdateCommit + `",
		"artifact_sha256":"sha256:` + hostSelfUpdateSHA256 + `",
		"agent_protocol_version":2,
		"executor_protocol_version":2,
		"mutation_protocol_version":2,
		"recovery_protocol_version":2,
		"release":` + validHostSelfUpdateReleaseBindingJSON() + `,
		"directive_issued_at":"2026-07-28T00:02:00Z",
		"plan_sha256":"` + hostSelfUpdateSHA256 + `",
		"session_id":"session-host-self-update-0001",
		"revision":1,
		"issued_at":"2026-07-28T00:03:00Z",
		"expires_at":"2026-07-28T00:04:30Z",
		"created_at":"2026-07-28T00:03:00Z",
		"updated_at":"2026-07-28T00:03:00Z"
	}`
}

func validConsumedHostSelfUpdateGrantJSON(operation string) string {
	grant := validHostSelfUpdateGrantJSON()
	grant = strings.Replace(
		grant,
		`"operation":"stage"`,
		`"operation":"`+operation+`"`,
		1,
	)
	receipt := `"consumed_at":"2026-07-28T00:03:30Z",`
	if operation == "stage" {
		receipt += `"stage_claim_revision":4,` +
			`"stage_claimed_at":"2026-07-28T00:03:30Z",`
	}
	return strings.Replace(
		grant,
		`"created_at":"2026-07-28T00:03:00Z"`,
		receipt+`"created_at":"2026-07-28T00:03:00Z"`,
		1,
	)
}

func validHostAgentSelfUpdatePolicyJSON() string {
	return `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":2,
		"revision":7,
		"source_policy_revision":5,
		"local_executor_policy_revision":3,
		"local_executor_policy_sha256":"` + hostSelfUpdatePolicySHA256 + `",
		"observe_only":false,
		"self_update_id":"self-update-1",
		"self_update_revision":3,
		"self_update_status":"staging",
		"runtime_requirement":{
			"minimum_agent_version":"v1.8.0",
			"minimum_executor_version":"v1.8.0",
			"agent_protocol_version":2,
			"executor_protocol_version":2,
			"mutation_protocol_version":2,
			"recovery_protocol_version":2
		},
		"self_update":{
			"generation":"11111111-1111-4111-8111-111111111111",
			"agent_version":"v1.8.0",
			"executor_version":"v1.8.0",
			"commit":"` + hostSelfUpdateCommit + `",
			"artifact_sha256":"sha256:` + hostSelfUpdateSHA256 + `",
			"agent_protocol_version":2,
			"executor_protocol_version":2,
			"mutation_protocol_version":2,
			"recovery_protocol_version":2,
			"release":` + validHostSelfUpdateReleaseBindingJSON() + `,
			"staged_at":"2026-07-28T00:02:00Z"
		},
		"targets":[{
			"service_id":"worker-a",
			"service_type":"worker",
			"deployment_mode":"systemd",
			"applied_config_revision":1
		}]
	}`
}

func validHostSelfUpdateReleaseBindingJSON() string {
	return `{
		"tag":"v1.8.0",
		"commit":"` + hostSelfUpdateCommit + `",
		"published_at":"2026-07-28T00:00:00Z",
		"manifest_asset_id":101,
		"manifest_asset_name":"host-agent-manifest.json",
		"manifest_sha256":"` + hostSelfUpdateSHA256 + `",
		"manifest_checksum_asset_id":102,
		"manifest_checksum_sha256":"` + hostSelfUpdateSHA256 + `",
		"archive_asset_id":103,
		"archive_asset_name":"autostream-host-agent_v1.8.0_linux_amd64.tar.gz",
		"archive_size":4096,
		"archive_sha256":"` + hostSelfUpdateSHA256 + `",
		"archive_checksum_asset_id":104,
		"archive_checksum_sha256":"` + hostSelfUpdateSHA256 + `",
		"arch":"amd64",
		"agent_protocol_version":2,
		"executor_protocol_version":2,
		"mutation_protocol_version":2,
		"recovery_protocol_version":2,
		"minimum_panel_version":"v1.8.0"
	}`
}

func validHostSelfUpdateReleaseBinding() HostSelfUpdateReleaseBinding {
	return HostSelfUpdateReleaseBinding{
		Tag:                     "v1.8.0",
		Commit:                  hostSelfUpdateCommit,
		PublishedAt:             time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		ManifestAssetID:         101,
		ManifestAssetName:       "host-agent-manifest.json",
		ManifestSHA256:          hostSelfUpdateSHA256,
		ManifestChecksumAssetID: 102,
		ManifestChecksumSHA256:  hostSelfUpdateSHA256,
		ArchiveAssetID:          103,
		ArchiveAssetName:        "autostream-host-agent_v1.8.0_linux_amd64.tar.gz",
		ArchiveSize:             4096,
		ArchiveSHA256:           hostSelfUpdateSHA256,
		ArchiveChecksumAssetID:  104,
		ArchiveChecksumSHA256:   hostSelfUpdateSHA256,
		Arch:                    "amd64",
		AgentProtocolVersion:    2,
		ExecutorProtocolVersion: 2,
		MutationProtocolVersion: 2,
		RecoveryProtocolVersion: 2,
		MinimumPanelVersion:     "v1.8.0",
	}
}

func cloneJSONDocument(t *testing.T, source map[string]any) map[string]any {
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
