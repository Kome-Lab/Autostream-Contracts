package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type contractSchemaNode struct {
	AdditionalProperties any                           `json:"additionalProperties"`
	Required             []string                      `json:"required"`
	Properties           map[string]contractSchemaNode `json:"properties"`
	Items                *contractSchemaNode           `json:"items"`
	AllOf                []contractSchemaNode          `json:"allOf"`
	AnyOf                []contractSchemaNode          `json:"anyOf"`
	OneOf                []contractSchemaNode          `json:"oneOf"`
	Not                  *contractSchemaNode           `json:"not"`
	Then                 *contractSchemaNode           `json:"then"`
	Enum                 []string                      `json:"enum"`
	Const                any                           `json:"const"`
	Pattern              string                        `json:"pattern"`
	MinLength            int                           `json:"minLength"`
	MaxLength            int                           `json:"maxLength"`
	Minimum              float64                       `json:"minimum"`
	Maximum              float64                       `json:"maximum"`
	MinItems             int                           `json:"minItems"`
	MaxItems             int                           `json:"maxItems"`
	WriteOnly            bool                          `json:"writeOnly"`
}

func readContractSchema(t *testing.T, name string) contractSchemaNode {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
	if err != nil {
		t.Fatal(err)
	}
	var doc contractSchemaNode
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return doc
}

func compileContractJSONSchema(t *testing.T, name string, dependencies ...string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	resources := append([]string{name}, dependencies...)
	for _, resourceName := range resources {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", resourceName))
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", resourceName, err)
		}
		if err := compiler.AddResource(resourceName, document); err != nil {
			t.Fatalf("%s: %v", resourceName, err)
		}
	}
	compiled, err := compiler.Compile(name)
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func contractSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func requireContractFields(t *testing.T, actual []string, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if !contractSliceHas(actual, field) {
			t.Fatalf("required fields %#v are missing %q", actual, field)
		}
	}
}

func TestSystemUpdatePublicSchemasMatchControlPanelWireShape(t *testing.T) {
	create := readContractSchema(t, "system-update-create-request.schema.json")
	requireContractFields(t, create.Required, "target_id", "strategy", "idempotency_key")
	idempotency := create.Properties["idempotency_key"]
	if idempotency.MinLength != 1 || idempotency.MaxLength != 128 || idempotency.Pattern == "" {
		t.Fatalf("idempotency key must match Control Panel validation: %#v", idempotency)
	}

	target := readContractSchema(t, "system-update-target.schema.json")
	requireContractFields(t, target.Required,
		"target_id", "target_type", "name", "update_available",
		"updater_online", "eligible", "busy",
	)
	for _, optional := range []string{
		"host_id", "current_version", "latest_version", "deployment_mode", "updater_id",
		"blocked_reason", "current_stream_id", "update_check_source", "update_check_error",
	} {
		if _, ok := target.Properties[optional]; !ok {
			t.Fatalf("target schema is missing optional field %q", optional)
		}
		if contractSliceHas(target.Required, optional) {
			t.Fatalf("target field %q must remain optional", optional)
		}
	}

	job := readContractSchema(t, "system-update-job.schema.json")
	requireContractFields(t, job.Required,
		"id", "target_id", "target_type", "host_id", "deployment_mode", "current_version", "target_version",
		"strategy", "status", "idempotency_key",
		"lease_generation", "sequence", "progress", "created_at", "updated_at",
	)
	for _, optional := range []string{
		"updater_id", "requested_by", "lease_expires_at",
		"code", "message", "artifact_digest", "previous_digest",
		"claimed_at", "completed_at", "canceled_at",
	} {
		if _, ok := job.Properties[optional]; !ok {
			t.Fatalf("job schema is missing optional field %q", optional)
		}
		if contractSliceHas(job.Required, optional) {
			t.Fatalf("job field %q must remain optional", optional)
		}
	}
	for _, hidden := range []string{"agent_service_id", "execution_host_id", "requested_by_user_id"} {
		if _, ok := job.Properties[hidden]; ok {
			t.Fatalf("job must not expose internal field %q", hidden)
		}
	}
	if !contractSliceHas(job.Properties["status"].Enum, "reconciling") {
		t.Fatal("job status enum is missing reconciling")
	}
	for _, field := range []string{"artifact_digest", "previous_digest"} {
		if job.Properties[field].Pattern != "^sha256:[a-f0-9]{64}$" {
			t.Fatalf("%s must be a canonical OCI-style SHA-256 digest", field)
		}
	}
	if job.Properties["code"].MaxLength != 128 || job.Properties["code"].Pattern != "^[a-z0-9_.-]+$" {
		t.Fatal("job code constraints differ from Control Panel validation")
	}

	openAPIBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openAPI := string(openAPIBody)
	start := strings.Index(openAPI, "    SystemUpdateJob:")
	end := strings.Index(openAPI, "    SystemUpdatesResponse:")
	if start < 0 || end <= start {
		t.Fatal("control OpenAPI is missing the SystemUpdateJob component")
	}
	jobComponent := openAPI[start:end]
	for _, field := range []string{"updater_id:", "host_id:"} {
		if !strings.Contains(jobComponent, field) {
			t.Fatalf("control OpenAPI job is missing %s", field)
		}
	}
	for _, hidden := range []string{"agent_service_id:", "execution_host_id:", "requested_by_user_id:"} {
		if strings.Contains(jobComponent, hidden) {
			t.Fatalf("control OpenAPI job exposes internal field %q", hidden)
		}
	}
}

func TestUpdateAgentLeaseRecoverySchemasAreExplicit(t *testing.T) {
	claimRequest := readContractSchema(t, "update-agent-claim-request.schema.json")
	requireContractFields(t, claimRequest.Required, "service_id")
	for _, optional := range []string{"host_id", "active_job_id"} {
		if contractSliceHas(claimRequest.Required, optional) {
			t.Fatalf("%s must remain optional for legacy per-host agents", optional)
		}
	}
	hostID := claimRequest.Properties["host_id"]
	if hostID.MinLength != 1 || hostID.MaxLength != 191 || hostID.Pattern == "" {
		t.Fatalf("claim host_id must be a bounded safe execution-host identity when present: %#v", hostID)
	}
	activeJobID := claimRequest.Properties["active_job_id"]
	if activeJobID.MinLength != 1 || activeJobID.MaxLength != 64 || activeJobID.Pattern != "\\S" {
		t.Fatalf("active_job_id must be a non-empty bounded job ID when present: %#v", activeJobID)
	}

	claim := readContractSchema(t, "update-agent-claim-response.schema.json")
	if len(claim.OneOf) != 2 {
		t.Fatalf("claim response must expose exactly lease and clear variants: %#v", claim.OneOf)
	}
	var leaseClaim, clearClaim contractSchemaNode
	for _, variant := range claim.OneOf {
		if _, ok := variant.Properties["job"]; ok {
			leaseClaim = variant
		}
		if _, ok := variant.Properties["clear_active_job_id"]; ok {
			clearClaim = variant
		}
	}
	requireContractFields(t, leaseClaim.Required,
		"job", "lease_token", "lease_expires_at", "lease_generation",
		"report_sequence", "recovery_required", "last_status",
	)
	if !leaseClaim.Properties["lease_token"].WriteOnly {
		t.Fatal("claim lease token must remain writeOnly")
	}
	if leaseClaim.Properties["lease_generation"].Minimum != 1 || leaseClaim.Properties["report_sequence"].Minimum != 1 {
		t.Fatal("claim generations and report sequences start at one")
	}
	if !contractSliceHas(leaseClaim.Properties["last_status"].Enum, "reconciling") {
		t.Fatal("claim last_status enum is missing reconciling")
	}
	requireContractFields(t, clearClaim.Required, "clear_active_job_id")
	if clearClaim.Properties["clear_active_job_id"].Const != true {
		t.Fatal("clear_active_job_id response must carry the explicit true sentinel")
	}
	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"active_job_id:",
		"UpdateAgentClearActiveJobResponse:",
		"active_job_id was omitted and no eligible job is currently available",
		"never claims a different job",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing active-job recovery marker %q", marker)
		}
	}

	report := readContractSchema(t, "update-agent-report-request.schema.json")
	requireContractFields(t, report.Required, "service_id", "lease_token", "lease_generation", "sequence", "status")
	if !report.Properties["lease_token"].WriteOnly || report.Properties["lease_generation"].Minimum != 1 {
		t.Fatal("report must bind its write-only token to a positive lease generation")
	}
	if !contractSliceHas(report.Properties["status"].Enum, "reconciling") {
		t.Fatal("report status enum is missing reconciling")
	}
	if report.Properties["code"].MaxLength != 128 || report.Properties["code"].Pattern != "^[a-z0-9_.-]+$" {
		t.Fatal("report code constraints differ from Control Panel validation")
	}

	authorize := readContractSchema(t, "update-agent-authorize-request.schema.json")
	requireContractFields(t, authorize.Required, "service_id", "lease_token", "lease_generation", "target_id", "target_version", "deployment_mode")
	if contractSliceHas(authorize.Required, "host_id") {
		t.Fatal("legacy authorize host_id must remain optional")
	}
	if authorize.AdditionalProperties != false || !authorize.Properties["lease_token"].WriteOnly || authorize.Properties["lease_token"].MinLength != 32 || authorize.Properties["lease_token"].MaxLength != 256 {
		t.Fatalf("authorize lease token constraints changed: %#v", authorize.Properties["lease_token"])
	}
	if authorize.Properties["lease_generation"].Minimum != 1 || authorize.Properties["host_id"].MaxLength != 191 || authorize.Properties["target_id"].MaxLength != 191 || authorize.Properties["target_version"].MaxLength != 128 {
		t.Fatal("authorize plan constraints differ from Control Panel validation")
	}
	if !contractSliceHas(authorize.Properties["deployment_mode"].Enum, "systemd") || !contractSliceHas(authorize.Properties["deployment_mode"].Enum, "docker") {
		t.Fatal("authorize deployment_mode must support exact systemd and docker plans")
	}
	for _, marker := range []string{"/services/update-jobs/{id}/authorize:", "updates.authorize", "legacy_system_update_authorization_disabled", `"410":`} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing disabled legacy authorization marker %q", marker)
		}
	}
	legacyPathStart := strings.Index(string(openAPI), "  /services/update-jobs/{id}/authorize:")
	grantPathStart := strings.Index(string(openAPI), "  /services/update-jobs/{id}/mutation-grants:")
	if legacyPathStart < 0 || grantPathStart <= legacyPathStart || strings.Contains(string(openAPI)[legacyPathStart:grantPathStart], `"204":`) || strings.Contains(string(openAPI)[legacyPathStart:grantPathStart], "UpdateAgentAuthorizeRequest") {
		t.Fatal("legacy authorization endpoint still advertises a successful mutation path")
	}
}

func TestUpdateAgentAuthorizeSchemaValidatesOnlyTheExactMutationPlan(t *testing.T) {
	schema := compileContractJSONSchema(t, "update-agent-authorize-request.schema.json")
	assertValidation := func(body string, wantValid bool) {
		t.Helper()
		var instance any
		if err := json.Unmarshal([]byte(body), &instance); err != nil {
			t.Fatal(err)
		}
		err := schema.Validate(instance)
		if wantValid && err != nil {
			t.Fatalf("expected valid authorization contract for %s: %v", body, err)
		}
		if !wantValid && err == nil {
			t.Fatalf("expected authorization contract rejection for %s", body)
		}
	}
	valid := `{"service_id":"updater-1","lease_token":"` + strings.Repeat("a", 43) + `","lease_generation":2,"host_id":"host-1","target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`
	assertValidation(valid, true)
	assertValidation(`{"service_id":"updater-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, true)
	assertValidation(`{"service_id":"updater-1","lease_token":"short","lease_generation":2,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, false)
	assertValidation(`{"service_id":"updater-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":0,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, false)
	assertValidation(`{"service_id":"updater-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"kubernetes"}`, false)
	assertValidation(strings.TrimSuffix(valid, "}")+`,"command":"systemctl restart"}`, false)
	assertValidation(`{"service_id":"updater-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"target_id":"control-panel","target_version":"v1.6.6"}`, false)
}

func TestUpdateAgentClaimSchemasValidateExactRecoveryShapes(t *testing.T) {
	requestSchema := compileContractJSONSchema(t, "update-agent-claim-request.schema.json")
	responseSchema := compileContractJSONSchema(t, "update-agent-claim-response.schema.json", "system-update-job.schema.json")

	assertValidation := func(schema *jsonschema.Schema, body string, wantValid bool) {
		t.Helper()
		var instance any
		if err := json.Unmarshal([]byte(body), &instance); err != nil {
			t.Fatal(err)
		}
		err := schema.Validate(instance)
		if wantValid && err != nil {
			t.Fatalf("expected valid contract for %s: %v", body, err)
		}
		if !wantValid && err == nil {
			t.Fatalf("expected contract rejection for %s", body)
		}
	}

	assertValidation(requestSchema, `{"service_id":"updater-1"}`, true)
	assertValidation(requestSchema, `{"service_id":"updater-1","host_id":"host-1"}`, true)
	assertValidation(requestSchema, `{"service_id":"updater-1","active_job_id":"job-1"}`, true)
	assertValidation(requestSchema, `{"service_id":"updater-1","host_id":""}`, false)
	assertValidation(requestSchema, `{"service_id":"updater-1","active_job_id":""}`, false)
	assertValidation(requestSchema, `{"service_id":"updater-1","active_job_id":"   "}`, false)
	assertValidation(requestSchema, `{"service_id":"updater-1","active_job_id":"`+strings.Repeat("a", 65)+`"}`, false)

	now := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	job := SystemUpdateJob{
		ID: "job-1", TargetID: "control-panel", TargetType: SystemUpdateTargetControlPanel,
		ExecutionHostID: "host-1",
		DeploymentMode:  SystemUpdateDeploymentSystemd, CurrentVersion: "v1.6.5", TargetVersion: "v1.6.6",
		Strategy: SystemUpdateWhenIdle, Status: SystemUpdateQueued,
		IdempotencyKey: "request-1", CreatedAt: now, UpdatedAt: now,
	}
	leaseBody, err := json.Marshal(UpdateAgentClaimResponse{
		Job: job, LeaseToken: strings.Repeat("a", 43), LeaseExpiresAt: now.Add(time.Minute),
		LeaseGeneration: 1, ReportSequence: 1, RecoveryRequired: false, LastStatus: SystemUpdateQueued,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertValidation(responseSchema, string(leaseBody), true)
	assertValidation(responseSchema, `{"clear_active_job_id":true}`, true)
	assertValidation(responseSchema, `{"clear_active_job_id":false}`, false)
	assertValidation(responseSchema, `{"clear_active_job_id":true,"lease_token":"`+strings.Repeat("a", 43)+`"}`, false)
	assertValidation(responseSchema, `{}`, false)
}

func TestSystemUpdateGoTypesPreserveRequiredZerosAndOmitOptionalFields(t *testing.T) {
	now := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	job := SystemUpdateJob{
		ID: "job-1", TargetID: "control-panel", TargetType: SystemUpdateTargetControlPanel,
		ExecutionHostID: "host-1",
		DeploymentMode:  SystemUpdateDeploymentSystemd, CurrentVersion: "v1.6.5", TargetVersion: "v1.6.6",
		Strategy: SystemUpdateWhenIdle, Status: SystemUpdateQueued,
		IdempotencyKey: "request-1",
		CreatedAt:      now, UpdatedAt: now,
	}
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"id", "target_id", "target_type", "host_id", "deployment_mode", "current_version", "target_version",
		"strategy", "status", "idempotency_key",
		"lease_generation", "sequence", "progress", "created_at", "updated_at",
	} {
		if _, ok := wire[field]; !ok {
			t.Fatalf("Go job JSON omitted required field %q: %s", field, body)
		}
	}
	for _, field := range []string{
		"updater_id", "requested_by", "lease_expires_at",
		"artifact_digest", "previous_digest", "claimed_at", "completed_at", "canceled_at",
	} {
		if _, ok := wire[field]; ok {
			t.Fatalf("Go job JSON emitted absent optional field %q: %s", field, body)
		}
	}

	claimBody, err := json.Marshal(UpdateAgentClaimResponse{
		Job: job, LeaseToken: strings.Repeat("a", 43), LeaseExpiresAt: now.Add(time.Minute),
		LeaseGeneration: 1, ReportSequence: 1, RecoveryRequired: false, LastStatus: SystemUpdateQueued,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		`"lease_generation":1`, `"report_sequence":1`,
		`"recovery_required":false`, `"last_status":"queued"`,
	} {
		if !strings.Contains(string(claimBody), marker) {
			t.Fatalf("claim JSON is missing %s: %s", marker, claimBody)
		}
	}

	requestBody, err := json.Marshal(UpdateAgentClaimRequest{ServiceID: "updater-1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(requestBody), "active_job_id") {
		t.Fatalf("empty active_job_id must be omitted: %s", requestBody)
	}
	requestBody, err = json.Marshal(UpdateAgentClaimRequest{ServiceID: "updater-1", HostID: "host-1", ActiveJobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`"host_id":"host-1"`, `"active_job_id":"job-1"`} {
		if !strings.Contains(string(requestBody), marker) {
			t.Fatalf("recovery claim is missing %s: %s", marker, requestBody)
		}
	}
	clearBody, err := json.Marshal(UpdateAgentClearActiveJobResponse{ClearActiveJobID: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(clearBody) != `{"clear_active_job_id":true}` {
		t.Fatalf("clear response must contain only the explicit sentinel: %s", clearBody)
	}
	authorizeBody, err := json.Marshal(UpdateAgentAuthorizeRequest{
		ServiceID: "updater-1", LeaseToken: strings.Repeat("a", 43), LeaseGeneration: 2,
		ExecutionHostID: "host-1", TargetID: "control-panel", TargetVersion: "v1.6.6", DeploymentMode: SystemUpdateDeploymentSystemd,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`"service_id":"updater-1"`, `"lease_generation":2`, `"host_id":"host-1"`, `"target_id":"control-panel"`, `"target_version":"v1.6.6"`, `"deployment_mode":"systemd"`} {
		if !strings.Contains(string(authorizeBody), marker) {
			t.Fatalf("authorize JSON is missing %s: %s", marker, authorizeBody)
		}
	}
}

func TestSystemUpdateInventoryHostContractsMatchGETShape(t *testing.T) {
	updater := readContractSchema(t, "system-update-agent-status.schema.json")
	if updater.AdditionalProperties != false {
		t.Fatal("updater status must reject unknown fields")
	}
	requireContractFields(t, updater.Required, "updater_id", "name", "status", "online", "version")
	if contractSliceHas(updater.Required, "last_heartbeat_at") {
		t.Fatal("updater last_heartbeat_at must remain optional before the first heartbeat")
	}

	host := readContractSchema(t, "system-update-host-status.schema.json")
	if host.AdditionalProperties != false {
		t.Fatal("host status must reject unknown fields")
	}
	requireContractFields(t, host.Required, "host_id", "name", "updater_id", "reachability")
	for _, value := range []string{"reachable", "unreachable", "unknown"} {
		if !contractSliceHas(host.Properties["reachability"].Enum, value) {
			t.Fatalf("host reachability enum is missing %q", value)
		}
	}
	for _, optional := range []string{"reachability_checked_at", "reachability_code"} {
		if contractSliceHas(host.Required, optional) {
			t.Fatalf("host status field %q must remain optional", optional)
		}
	}

	response := readContractSchema(t, "system-updates-response.schema.json")
	if response.AdditionalProperties != false {
		t.Fatal("system update GET response must reject undeclared top-level fields")
	}
	requireContractFields(t, response.Required, "updaters", "hosts", "targets", "jobs")

	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	instance := SystemUpdatesResponse{
		Updaters: []SystemUpdateAgentStatus{{
			UpdaterID: "updater-1", Name: "Central Updater", Status: "online",
			Online: true, Version: "v1.7.0", LastHeartbeatAt: &now,
		}},
		Hosts: []SystemUpdateHostStatus{{
			HostID: "host-1", Name: "Encoder Host", UpdaterID: "updater-1",
			Reachability: SystemUpdateReachable, ReachabilityCheckedAt: &now,
		}},
		Targets: []SystemUpdateTarget{{
			TargetID: "worker-1", TargetType: SystemUpdateTargetWorker, Name: "Worker",
			HostID: "host-1", UpdateAvailable: false, UpdaterOnline: true, Eligible: false, Busy: false,
		}},
		Jobs: []SystemUpdateJob{{
			ID: "job-1", TargetID: "worker-1", TargetType: SystemUpdateTargetWorker,
			ExecutionHostID: "host-1", DeploymentMode: SystemUpdateDeploymentSystemd,
			CurrentVersion: "v1.0.0", TargetVersion: "v1.0.1", Strategy: SystemUpdateWhenIdle, Status: SystemUpdateQueued,
			IdempotencyKey: "request-1", CreatedAt: now, UpdatedAt: now,
		}},
	}
	body, err := json.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	schema := compileContractJSONSchema(t,
		"system-updates-response.schema.json",
		"system-update-agent-status.schema.json",
		"system-update-host-status.schema.json",
		"system-update-target.schema.json",
		"system-update-job.schema.json",
	)
	if err := schema.Validate(document); err != nil {
		t.Fatalf("Go system update GET response violates schema: %v body=%s", err, body)
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"SystemUpdateAgentStatus:", "SystemUpdateHostStatus:", "required: [updaters, hosts, targets, jobs]",
		"enum: [reachable, unreachable, unknown]", "reachability_checked_at:", "reachability_code:",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing inventory-host marker %q", marker)
		}
	}
}

func TestMutationGrantContractsBindExactRemoteSessionAndRejectUnknownFields(t *testing.T) {
	issue := readContractSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	issueFields := []string{
		"service_id", "lease_token", "lease_generation", "host_id", "target_id",
		"target_version", "deployment_mode", "operation", "plan_sha256", "session_id",
	}
	requireContractFields(t, issue.Required, issueFields...)
	if issue.AdditionalProperties != false || !issue.Properties["lease_token"].WriteOnly {
		t.Fatal("grant issue request must reject unknown fields and protect the lease token")
	}

	issued := readContractSchema(t, "update-agent-mutation-grant-issue-response.schema.json")
	requireContractFields(t, issued.Required, "grant_token", "expires_at")
	if issued.AdditionalProperties != false || !issued.Properties["grant_token"].WriteOnly ||
		issued.Properties["grant_token"].MinLength != 32 || issued.Properties["grant_token"].MaxLength != 256 {
		t.Fatal("issued mutation grant must be a bounded one-time secret")
	}
	if _, exists := issued.Properties["grant_id"]; exists {
		t.Fatal("public mutation grant response must not expose the internal grant ID")
	}

	consume := readContractSchema(t, "update-agent-mutation-grant-consume-request.schema.json")
	consumeFields := []string{
		"lease_generation", "host_id", "target_id", "target_version",
		"deployment_mode", "operation", "plan_sha256", "session_id",
	}
	requireContractFields(t, consume.Required, consumeFields...)
	if consume.AdditionalProperties != false {
		t.Fatal("grant consume request must reject unknown fields")
	}
	for _, forbidden := range []string{"job_id", "grant_token", "ssh_address", "ssh_user", "ssh_path", "path", "command"} {
		if _, exists := issue.Properties[forbidden]; exists {
			t.Fatalf("grant issue body exposes forbidden field %q", forbidden)
		}
		if _, exists := consume.Properties[forbidden]; exists {
			t.Fatalf("grant consume body exposes forbidden field %q", forbidden)
		}
	}
	for _, forbidden := range []string{"service_id", "lease_token"} {
		if _, exists := consume.Properties[forbidden]; exists {
			t.Fatalf("grant consume body exposes service-lease credential field %q", forbidden)
		}
	}
	for _, schema := range []contractSchemaNode{issue, consume} {
		if schema.Properties["plan_sha256"].Pattern != "^[a-f0-9]{64}$" {
			t.Fatal("mutation grant plan must use an exact lowercase SHA-256 binding")
		}
		if schema.Properties["session_id"].MinLength != 16 || schema.Properties["session_id"].MaxLength != 128 ||
			schema.Properties["session_id"].Pattern != "^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$" {
			t.Fatal("mutation grant session must be a bounded high-entropy identifier")
		}
		for _, operation := range []string{"apply", "reconcile"} {
			if !contractSliceHas(schema.Properties["operation"].Enum, operation) {
				t.Fatalf("mutation grant operation enum is missing %q", operation)
			}
		}
	}

	issueSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	consumeSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-consume-request.schema.json")
	responseSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-response.schema.json")
	assertValidation := func(schema *jsonschema.Schema, body string, wantValid bool) {
		t.Helper()
		var instance any
		if err := json.Unmarshal([]byte(body), &instance); err != nil {
			t.Fatal(err)
		}
		err := schema.Validate(instance)
		if wantValid && err != nil {
			t.Fatalf("expected valid mutation grant contract for %s: %v", body, err)
		}
		if !wantValid && err == nil {
			t.Fatalf("expected mutation grant contract rejection for %s", body)
		}
	}
	hash := strings.Repeat("a", 64)
	lease := strings.Repeat("l", 43)
	grant := strings.Repeat("g", 43)
	validIssue := `{"service_id":"updater-1","lease_token":"` + lease + `","lease_generation":2,"host_id":"host-1","target_id":"worker-1","target_version":"v1.2.0","deployment_mode":"systemd","operation":"apply","plan_sha256":"` + hash + `","session_id":"session-contract-0001"}`
	validConsume := `{"lease_generation":2,"host_id":"host-1","target_id":"worker-1","target_version":"v1.2.0","deployment_mode":"systemd","operation":"apply","plan_sha256":"` + hash + `","session_id":"session-contract-0001"}`
	assertValidation(issueSchema, validIssue, true)
	assertValidation(consumeSchema, validConsume, true)
	assertValidation(responseSchema, `{"grant_token":"`+grant+`","expires_at":"2026-07-19T00:01:00Z"}`, true)
	assertValidation(issueSchema, strings.TrimSuffix(validIssue, "}")+`,"command":"systemctl restart x"}`, false)
	assertValidation(consumeSchema, strings.TrimSuffix(validConsume, "}")+`,"grant_token":"`+grant+`"}`, false)
	assertValidation(consumeSchema, strings.Replace(validConsume, hash, strings.ToUpper(hash), 1), false)
	assertValidation(consumeSchema, strings.Replace(validConsume, `"operation":"apply"`, `"operation":"probe"`, 1), false)
	assertValidation(consumeSchema, strings.Replace(validConsume, "session-contract-0001", "short", 1), false)
	assertValidation(responseSchema, `{"grant_token":"`+grant+`","expires_at":"2026-07-19T00:01:00Z","session_id":"session-contract-0001"}`, false)

	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	issueBody, err := json.Marshal(UpdateAgentMutationGrantIssueRequest{
		ServiceID: "updater-1", LeaseToken: lease, LeaseGeneration: 2, ExecutionHostID: "host-1",
		TargetID: "worker-1", TargetVersion: "v1.2.0", DeploymentMode: SystemUpdateDeploymentSystemd,
		Operation: SystemUpdateMutationApply, PlanSHA256: hash, SessionID: "session-contract-0001",
	})
	if err != nil {
		t.Fatal(err)
	}
	consumeBody, err := json.Marshal(UpdateAgentMutationGrantConsumeRequest{
		LeaseGeneration: 2, ExecutionHostID: "host-1", TargetID: "worker-1",
		TargetVersion: "v1.2.0", DeploymentMode: SystemUpdateDeploymentSystemd,
		Operation: SystemUpdateMutationApply, PlanSHA256: hash, SessionID: "session-contract-0001",
	})
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := json.Marshal(UpdateAgentMutationGrantIssueResponse{GrantToken: grant, ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	for schema, body := range map[*jsonschema.Schema][]byte{
		issueSchema: issueBody, consumeSchema: consumeBody, responseSchema: responseBody,
	} {
		var instance any
		if err := json.Unmarshal(body, &instance); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("Go mutation grant JSON violates schema: %v body=%s", err, body)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"/services/update-jobs/{id}/mutation-grants:",
		"/services/update-jobs/{id}/mutation-grants/consume:",
		"UpdateAgentMutationGrantIssueRequest:",
		"UpdateAgentMutationGrantIssueResponse:",
		"UpdateAgentMutationGrantConsumeRequest:",
		`"201":`, "Always no-store because the response contains a one-time grant credential.",
		"supplied as the Bearer credential", "exact same grant, session, and binding", "different reuse returns 409",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing mutation-grant marker %q", marker)
		}
	}
	if strings.Contains(string(openAPI), "grant_id:") {
		t.Fatal("control OpenAPI exposes the internal mutation grant ID")
	}
}

func TestUpdaterRegistrationScopesHeartbeatAndEmailHTMLContracts(t *testing.T) {
	if ServiceUpdateAgent != "update_agent" || ServiceStatusUpdating != "updating" {
		t.Fatal("update agent service type or updating status changed")
	}
	if ScopeUpdatesClaim != "updates.claim" || ScopeUpdatesReport != "updates.report" || ScopeUpdatesAuthorize != "updates.authorize" {
		t.Fatal("update agent scopes changed")
	}
	if SystemUpdateReconciling != "reconciling" ||
		SystemUpdateDeploymentSystemd != "systemd" || SystemUpdateDeploymentDocker != "docker" {
		t.Fatal("system update recovery or deployment values changed")
	}

	heartbeat := readContractSchema(t, "heartbeat.schema.json")
	if !contractSliceHas(heartbeat.Properties["status"].Enum, "updating") {
		t.Fatal("heartbeat status enum is missing updating")
	}
	for _, name := range []string{"service-token-create-request.schema.json", "service-token.schema.json"} {
		token := readContractSchema(t, name)
		if !contractSliceHas(token.Properties["service_type"].Enum, "update_agent") {
			t.Fatalf("%s is missing update_agent", name)
		}
		for _, scope := range []string{"updates.claim", "updates.report", "updates.authorize"} {
			if !contractSliceHas(token.Properties["scopes"].Items.Enum, scope) {
				t.Fatalf("%s is missing %s", name, scope)
			}
		}
	}
	issuedTokenRequest, err := json.Marshal(ServiceTokenCreateRequest{
		ServiceType: ServiceUpdateAgent,
		Scopes:      []ServiceScope{ScopeServiceRegister, ScopeServiceHeartbeat, ScopeServiceConfigRead, ScopeServiceLogsWrite, ScopeServiceStatusWrite, ScopeUpdatesClaim, ScopeUpdatesReport, ScopeUpdatesAuthorize},
	})
	if err != nil {
		t.Fatal(err)
	}
	var issuedTokenInstance any
	if err := json.Unmarshal(issuedTokenRequest, &issuedTokenInstance); err != nil {
		t.Fatal(err)
	}
	if err := compileContractJSONSchema(t, "service-token-create-request.schema.json").Validate(issuedTokenInstance); err != nil {
		t.Fatalf("Control Panel-issued update_agent token violates the contract: %v body=%s", err, issuedTokenRequest)
	}

	email := readContractSchema(t, "service-notification-email-request.schema.json")
	requireContractFields(t, email.Required, "recipients", "subject", "text")
	if contractSliceHas(email.Required, "html") {
		t.Fatal("email html must remain optional")
	}
	html := email.Properties["html"]
	if html.MaxLength != 65536 || html.Pattern != "^[^\\u0000]*$" {
		t.Fatalf("unexpected email html limits: %#v", html)
	}
	body, err := json.Marshal(ServiceNotificationEmailRequest{
		Recipients: []string{"ops@example.jp"}, Subject: "更新", Text: "plain fallback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"html"`) {
		t.Fatalf("empty optional html must be omitted: %s", body)
	}
	openAPIBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"optional HTML alternative", "html:", "maxLength: 65536"} {
		if !strings.Contains(string(openAPIBody), marker) {
			t.Fatalf("control OpenAPI is missing email HTML marker %q", marker)
		}
	}
}

func TestReleaseManifestSchemasSeparateHostAndDockerChannels(t *testing.T) {
	rawManifest, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"release-manifest.json.sha256", "64 lowercase hexadecimal characters", "two spaces", "trailing newline"} {
		if !strings.Contains(string(rawManifest), marker) {
			t.Fatalf("release manifest sidecar contract is missing %q", marker)
		}
	}
	manifest := readContractSchema(t, "release-manifest.schema.json")
	requireContractFields(t, manifest.Required, "schema_version", "release_id", "channel", "published_at", "components")
	requireContractFields(t, manifest.Required, "minimum_agent_version")
	if !contractSliceHas(manifest.Properties["channel"].Enum, "host") ||
		!contractSliceHas(manifest.Properties["channel"].Enum, "docker") {
		t.Fatal("release manifest must support host and docker channels")
	}
	for _, alias := range []string{"bundle_version", "generated_at"} {
		if !strings.Contains(manifest.Properties[alias].Pattern+readReleaseDescription(t, alias), "must equal") {
			t.Fatalf("%s must document its canonical identity alias", alias)
		}
	}
	if len(manifest.AllOf) != 2 || manifest.AllOf[0].Then == nil || manifest.AllOf[1].Then == nil {
		t.Fatalf("release manifest channel conditions changed: %#v", manifest.AllOf)
	}

	docker := manifest.AllOf[0].Then
	requireContractFields(t, docker.Required, "bundle_version", "generated_at", "minimum_agent_version")
	if manifest.Properties["minimum_agent_version"].Pattern != "^v[0-9]+\\.[0-9]+\\.[0-9]+$" {
		t.Fatal("minimum_agent_version must use the canonical release version format")
	}
	dockerComponents := docker.Properties["components"]
	if dockerComponents.MinItems != 5 || dockerComponents.MaxItems != 5 || dockerComponents.Items == nil {
		t.Fatalf("docker manifest must contain exactly five components: %#v", dockerComponents)
	}
	requireContractFields(t, dockerComponents.Items.Required,
		"service", "source_version", "image", "manifest_digest", "platform_digests",
		"rollback_compatible", "database_schema",
	)
	if len(dockerComponents.AllOf) != 5 {
		t.Fatalf("docker manifest must contain each known service exactly once: %#v", dockerComponents.AllOf)
	}
	if !strings.Contains(dockerComponents.Items.Properties["image"].Pattern, "ghcr\\.io/kome-lab/autostream-docker") {
		t.Fatal("docker images must use the fixed lowercase GHCR namespace")
	}
	if dockerComponents.Items.Properties["rollback_compatible"].Const != true {
		t.Fatal("Docker releases must explicitly allow rollback")
	}
	if dockerComponents.Items.Not == nil {
		t.Fatal("Docker components must reject host-only fields")
	}
	for _, forbidden := range dockerComponents.Items.Not.AnyOf {
		for _, policyField := range []string{"rollback_compatible", "database_schema"} {
			if contractSliceHas(forbidden.Required, policyField) {
				t.Fatalf("Docker schema both requires and forbids %q", policyField)
			}
		}
	}
	if schemas := dockerComponents.Items.Properties["database_schema"].Enum; !contractSliceHas(schemas, "none") || !contractSliceHas(schemas, "backward_compatible") || contractSliceHas(schemas, "irreversible") {
		t.Fatalf("Docker database_schema policy is unsafe: %#v", schemas)
	}
	expectedSchemas := []string{"backward_compatible", "none", "none", "backward_compatible", "none"}
	for i, expected := range expectedSchemas {
		if dockerComponents.Items.AllOf[i].Then == nil || dockerComponents.Items.AllOf[i].Then.Properties["database_schema"].Const != expected {
			t.Fatalf("Docker service policy %d must require database_schema %q", i, expected)
		}
	}
	componentJSON, err := json.Marshal(ReleaseManifestComponent{})
	if err != nil {
		t.Fatal(err)
	}
	var componentWire map[string]any
	if err := json.Unmarshal(componentJSON, &componentWire); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"rollback_compatible", "database_schema"} {
		if _, ok := componentWire[required]; !ok {
			t.Fatalf("ReleaseManifestComponent omits required field %q", required)
		}
	}

	platforms := manifest.Properties["components"].Items.Properties["platform_digests"]
	requireContractFields(t, platforms.Required, "linux/amd64", "linux/arm64")
	if platforms.AdditionalProperties != false {
		t.Fatal("platform_digests must contain exactly amd64 and arm64")
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		if platforms.Properties[platform].Pattern != "^sha256:[a-f0-9]{64}$" {
			t.Fatalf("%s digest is not canonical", platform)
		}
	}

	hostComponents := manifest.AllOf[1].Then.Properties["components"]
	if hostComponents.MinItems != 1 || hostComponents.MaxItems != 1 || hostComponents.Items == nil {
		t.Fatal("a host release manifest must describe exactly its repository component")
	}
	requireContractFields(t, hostComponents.Items.Required,
		"service", "source_version", "commit", "artifacts",
		"rollback_compatible", "database_schema",
	)
	if hostComponents.Items.Properties["rollback_compatible"].Const != true {
		t.Fatal("host releases must explicitly allow rollback")
	}
	hostArtifacts := hostComponents.Items.Properties["artifacts"]
	if hostArtifacts.MinItems != 2 || hostArtifacts.MaxItems != 2 {
		t.Fatal("host releases must contain exactly amd64 and arm64 artifacts")
	}
	artifacts := manifest.Properties["components"].Items.Properties["artifacts"]
	if artifacts.Items == nil || artifacts.Items.Properties["sha256"].Pattern != "^[a-f0-9]{64}$" ||
		artifacts.Items.Properties["size"].Maximum != 268435456 {
		t.Fatal("host artifact checksum or size limit differs from the updater")
	}
}

func TestDockerReleaseManifestGeneratorShapeValidatesAgainstSchema(t *testing.T) {
	// This fixture is generated with autostream-docker's release manifest
	// generator. Its producer-side tests separately pin the exact field set.
	instanceJSON, err := os.ReadFile(filepath.Join("..", "..", "testdata", "release-manifest.docker.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		t.Fatal(err)
	}

	schemaJSON, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	schemaDocument, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err := compiler.AddResource("release-manifest.schema.json", schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("release-manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(instance); err != nil {
		t.Fatalf("generator-shaped Docker manifest violates shared schema: %v", err)
	}

	instanceMap := instance.(map[string]any)
	componentMaps := instanceMap["components"].([]any)
	first := componentMaps[0].(map[string]any)
	first["rollback_compatible"] = false
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted rollback_compatible=false")
	}
	delete(first, "rollback_compatible")
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted Docker component without rollback_compatible")
	}
	first["rollback_compatible"] = true
	delete(first, "database_schema")
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted Docker component without database_schema")
	}
	first["database_schema"] = "none"
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted unsafe control-panel database_schema")
	}
}

func readReleaseDescription(t *testing.T, property string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Properties[property].Description
}
