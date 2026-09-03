package contracts

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type contractSchemaNode struct {
	AdditionalProperties any                           `json:"additionalProperties"`
	Type                 string                        `json:"type"`
	Required             []string                      `json:"required"`
	Properties           map[string]contractSchemaNode `json:"properties"`
	PropertyNames        *contractSchemaNode           `json:"propertyNames"`
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
	canonicalRootID := ""
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
		var identity struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(body, &identity); err != nil {
			t.Fatalf("%s: %v", resourceName, err)
		}
		if identity.ID != "" && identity.ID != resourceName {
			if err := compiler.AddResource(identity.ID, document); err != nil {
				t.Fatalf("%s canonical id: %v", resourceName, err)
			}
		}
		if resourceName == name {
			canonicalRootID = identity.ID
			continue
		}
		if canonicalRootID == "" {
			continue
		}
		rootURL, err := url.Parse(canonicalRootID)
		if err != nil {
			t.Fatalf("%s canonical id: %v", name, err)
		}
		relativeURL, err := url.Parse(resourceName)
		if err != nil {
			t.Fatalf("%s resource name: %v", resourceName, err)
		}
		resolvedID := rootURL.ResolveReference(relativeURL).String()
		if resolvedID != resourceName && resolvedID != identity.ID {
			if err := compiler.AddResource(resolvedID, document); err != nil {
				t.Fatalf("%s resolved canonical id: %v", resourceName, err)
			}
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

func systemUpdateAutomaticResendDisabledFixture() *bool {
	allowed := false
	return &allowed
}

func systemUpdateV2JobFixture(now time.Time) SystemUpdateJob {
	return SystemUpdateJob{
		ProtocolVersion: 2, ID: "job-1", TargetID: "control-panel", TargetType: SystemUpdateTargetControlPanel,
		ExecutionHostID: "host-1", TransportMode: UpdateTransportPullV2, OwnershipEpoch: 2, PolicyRevision: 3,
		DeploymentMode: SystemUpdateDeploymentSystemd, CurrentVersion: "v1.6.5", TargetVersion: "v1.6.6",
		Strategy: SystemUpdateWhenIdle, Status: SystemUpdateQueued, Outcome: "pending",
		IdempotencyKey: "request-1", UpdaterID: "updater-1", AuthorizationID: "authorization-1",
		CanonicalPayloadDigest: portContractConfigDigest, DesiredRevision: 4, Fence: 5,
		RequiredCapability: UpdaterCapabilityUpdate, LeaseGeneration: 1,
		AutomaticResendAllowed: systemUpdateAutomaticResendDisabledFixture(), CreatedAt: now, UpdatedAt: now,
	}
}

func TestSystemUpdatePublicSchemasMatchControlPanelWireShape(t *testing.T) {
	create := readContractSchema(t, "system-update-create-request.schema.json")
	requireContractFields(t, create.Required, "protocol_version", "target_id", "idempotency_key", "desired_revision", "fence", "required_capability")
	if _, ok := create.Properties["strategy"]; !ok {
		t.Fatal("software-update create request is missing strategy")
	}
	idempotency := create.Properties["idempotency_key"]
	if idempotency.MinLength != 1 || idempotency.MaxLength != 128 || idempotency.Pattern == "" {
		t.Fatalf("idempotency key must match Control Panel validation: %#v", idempotency)
	}

	target := readContractSchema(t, "system-update-target.schema.json")
	requireContractFields(t, target.Required,
		"protocol_version", "host_id", "updater_id", "capabilities", "desired_revision",
		"applied_revision", "fence", "updater_health", "application_probe",
		"target_id", "target_type", "name", "update_available",
		"updater_online", "eligible", "busy",
	)
	for _, optional := range []string{
		"current_version", "latest_version", "deployment_mode",
		"blocked_reason", "current_stream_id", "update_check_source",
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
		"protocol_version", "transport_mode", "updater_id", "desired_revision", "fence", "outcome",
		"required_capability", "authorization_id", "canonical_payload_digest", "automatic_resend_allowed",
		"id", "target_id", "target_type", "host_id", "deployment_mode", "current_version", "target_version",
		"strategy", "status", "idempotency_key",
		"lease_generation", "sequence", "progress", "created_at", "updated_at",
	)
	for _, optional := range []string{
		"requested_by", "lease_expires_at", "artifact_digest", "previous_digest",
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

	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	components := requireCharacterizationMap(t, control, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")
	jobComponent := resolveCharacterizationSchema(t, control, schemas["SystemUpdateJob"])
	jobProperties := requireCharacterizationMap(t, jobComponent, "properties")
	for _, field := range []string{"updater_id", "host_id"} {
		if _, ok := jobProperties[field]; !ok {
			t.Fatalf("control OpenAPI job is missing %s", field)
		}
	}
	for _, hidden := range []string{"agent_service_id", "execution_host_id", "requested_by_user_id"} {
		if _, ok := jobProperties[hidden]; ok {
			t.Fatalf("control OpenAPI job exposes internal field %q", hidden)
		}
	}
}

func TestUpdateAgentLeaseRecoverySchemasAreExplicit(t *testing.T) {
	claimRequest := readContractSchema(t, "update-agent-claim-request.schema.json")
	requireContractFields(t, claimRequest.Required, "updater_id", "host_id", "lease_generation", "fence")
	if contractSliceHas(claimRequest.Required, "active_job_id") {
		t.Fatal("active_job_id must remain optional for recovery")
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
	requireContractFields(t, report.Required, "updater_id", "host_id", "lease_token", "lease_generation", "fence", "sequence", "status")
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
	requireContractFields(t, authorize.Required, "updater_id", "host_id", "lease_token", "lease_generation", "fence", "target_id", "target_version", "deployment_mode")
	if authorize.AdditionalProperties != false || !authorize.Properties["lease_token"].WriteOnly || authorize.Properties["lease_token"].MinLength != 32 || authorize.Properties["lease_token"].MaxLength != 256 {
		t.Fatalf("authorize lease token constraints changed: %#v", authorize.Properties["lease_token"])
	}
	if authorize.Properties["lease_generation"].Minimum != 1 || authorize.Properties["host_id"].MaxLength != 191 || authorize.Properties["target_id"].MaxLength != 128 || authorize.Properties["target_version"].MaxLength != 128 {
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
	valid := `{"updater_id":"updater-1","host_id":"host-1","lease_token":"` + strings.Repeat("a", 43) + `","lease_generation":2,"fence":3,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`
	assertValidation(valid, true)
	assertValidation(`{"updater_id":"updater-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"fence":3,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, false)
	assertValidation(`{"updater_id":"updater-1","host_id":"host-1","lease_token":"short","lease_generation":2,"fence":3,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, false)
	assertValidation(`{"updater_id":"updater-1","host_id":"host-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":0,"fence":3,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"systemd"}`, false)
	assertValidation(`{"updater_id":"updater-1","host_id":"host-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"fence":3,"target_id":"control-panel","target_version":"v1.6.6","deployment_mode":"kubernetes"}`, false)
	assertValidation(strings.TrimSuffix(valid, "}")+`,"command":"systemctl restart"}`, false)
	assertValidation(`{"updater_id":"updater-1","host_id":"host-1","lease_token":"`+strings.Repeat("a", 43)+`","lease_generation":2,"fence":3,"target_id":"control-panel","target_version":"v1.6.6"}`, false)
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

	validClaim := `{"updater_id":"updater-1","host_id":"host-1","lease_generation":2,"fence":3}`
	assertValidation(requestSchema, validClaim, true)
	assertValidation(requestSchema, strings.TrimSuffix(validClaim, "}")+`,"active_job_id":"job-1"}`, true)
	assertValidation(requestSchema, `{"service_id":"updater-1","host_id":"host-1","lease_generation":2,"fence":3}`, false)
	assertValidation(requestSchema, `{"updater_id":"updater-1","host_id":"","lease_generation":2,"fence":3}`, false)
	assertValidation(requestSchema, strings.TrimSuffix(validClaim, "}")+`,"active_job_id":""}`, false)
	assertValidation(requestSchema, strings.TrimSuffix(validClaim, "}")+`,"active_job_id":"   "}`, false)
	assertValidation(requestSchema, strings.TrimSuffix(validClaim, "}")+`,"active_job_id":"`+strings.Repeat("a", 65)+`"}`, false)

	now := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	job := systemUpdateV2JobFixture(now)
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
	job := systemUpdateV2JobFixture(now)
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"protocol_version", "transport_mode", "updater_id", "desired_revision", "fence", "outcome",
		"required_capability", "authorization_id", "canonical_payload_digest", "automatic_resend_allowed",
		"id", "target_id", "target_type", "host_id", "deployment_mode", "current_version", "target_version",
		"strategy", "status", "idempotency_key",
		"lease_generation", "sequence", "progress", "created_at", "updated_at",
	} {
		if _, ok := wire[field]; !ok {
			t.Fatalf("Go job JSON omitted required field %q: %s", field, body)
		}
	}
	for _, field := range []string{
		"requested_by", "lease_expires_at",
		"artifact_digest", "previous_digest", "claimed_at", "completed_at", "canceled_at",
	} {
		if _, ok := wire[field]; ok {
			t.Fatalf("Go job JSON emitted absent optional field %q: %s", field, body)
		}
	}
	for _, marker := range []string{`"sequence":0`, `"progress":0`, `"automatic_resend_allowed":false`} {
		if !strings.Contains(string(body), marker) {
			t.Fatalf("Go job JSON omitted required zero/false marker %s", marker)
		}
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-job.schema.json"), string(body), true)

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

	requestBody, err := json.Marshal(UpdateAgentClaimRequest{UpdaterID: "updater-1", HostID: "host-1", LeaseGeneration: 1, Fence: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(requestBody), "active_job_id") {
		t.Fatalf("empty active_job_id must be omitted: %s", requestBody)
	}
	requestBody, err = json.Marshal(UpdateAgentClaimRequest{UpdaterID: "updater-1", HostID: "host-1", LeaseGeneration: 2, Fence: 3, ActiveJobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`"updater_id":"updater-1"`, `"host_id":"host-1"`, `"lease_generation":2`, `"fence":3`, `"active_job_id":"job-1"`} {
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
		UpdaterID: "updater-1", HostID: "host-1", LeaseToken: strings.Repeat("a", 43), LeaseGeneration: 2, Fence: 3,
		TargetID: "control-panel", TargetVersion: "v1.6.6", DeploymentMode: SystemUpdateDeploymentSystemd,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`"updater_id":"updater-1"`, `"lease_generation":2`, `"fence":3`, `"host_id":"host-1"`, `"target_id":"control-panel"`, `"target_version":"v1.6.6"`, `"deployment_mode":"systemd"`} {
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
	requireContractFields(t, updater.Required, "protocol_version", "updater_id", "host_id", "service_id", "name", "transport_mode", "authentication", "status", "online", "version", "heartbeat_sequence", "capabilities", "desired_revision", "applied_revision", "fence")
	updaterOptional := []string{
		"execution_host_id", "ownership_epoch", "last_heartbeat_at", "policy_status", "policy_error_code",
		"bootstrap_encryption_public_key", "bootstrap_encryption_key_fingerprint",
	}
	for _, optional := range updaterOptional {
		if contractSliceHas(updater.Required, optional) {
			t.Fatalf("updater status field %q must remain optional", optional)
		}
	}
	if updater.Properties["transport_mode"].Const != "pull_v2" {
		t.Fatalf("updater transport_mode must be pull_v2, got %#v", updater.Properties["transport_mode"].Const)
	}
	for _, field := range []string{"ownership_epoch", "applied_revision"} {
		property := updater.Properties[field]
		if property.Type != "integer" || property.Minimum != 0 {
			t.Fatalf("updater %s must be a non-negative integer: %#v", field, property)
		}
	}
	if desired := updater.Properties["desired_revision"]; desired.Type != "integer" || desired.Minimum != 1 {
		t.Fatalf("updater desired_revision must be a positive integer: %#v", desired)
	}
	for _, value := range []string{"applied", "pending", "failed"} {
		if !contractSliceHas(updater.Properties["policy_status"].Enum, value) {
			t.Fatalf("updater policy_status enum is missing %q", value)
		}
	}
	for _, removed := range []string{"ssh_client_public_keys", "ssh_client_key_fingerprints"} {
		if _, ok := updater.Properties[removed]; ok {
			t.Fatalf("updater v2 status retained removed SSH field %q", removed)
		}
	}
	for field, wantPattern := range map[string]string{
		"bootstrap_encryption_public_key":      "^B[A-Za-z0-9_-]{86}$",
		"bootstrap_encryption_key_fingerprint": "^SHA256:[A-Za-z0-9+/]{43}$",
	} {
		if updater.Properties[field].Pattern != wantPattern {
			t.Fatalf("updater %s has the wrong cryptographic format constraint: %q", field, updater.Properties[field].Pattern)
		}
	}

	host := readContractSchema(t, "system-update-host-status.schema.json")
	if host.AdditionalProperties != false {
		t.Fatal("host status must reject unknown fields")
	}
	requireContractFields(t, host.Required, "protocol_version", "host_id", "name", "updater_id", "reachability", "updater_health", "application_probe")
	for _, value := range []string{"reachable", "unreachable", "unknown"} {
		if !contractSliceHas(host.Properties["reachability"].Enum, value) {
			t.Fatalf("host reachability enum is missing %q", value)
		}
	}
	for _, optional := range []string{
		"reachability_checked_at", "reachability_code",
	} {
		if contractSliceHas(host.Required, optional) {
			t.Fatalf("host status field %q must remain optional", optional)
		}
	}
	for _, removed := range []string{"ssh_client_public_key", "ssh_client_key_fingerprint"} {
		if _, ok := host.Properties[removed]; ok {
			t.Fatalf("host v2 status retained removed SSH field %q", removed)
		}
	}

	response := readContractSchema(t, "system-updates-response.schema.json")
	if response.AdditionalProperties != false {
		t.Fatal("system update GET response must reject undeclared top-level fields")
	}
	requireContractFields(t, response.Required, "updaters", "hosts", "targets", "jobs")

	now := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	inventoryJob := systemUpdateV2JobFixture(now)
	inventoryJob.TargetID = "worker-1"
	inventoryJob.TargetType = SystemUpdateTargetWorker
	instance := SystemUpdatesResponse{
		Updaters: []SystemUpdateAgentStatus{{
			ProtocolVersion: 2, UpdaterID: "updater-1", HostID: "host-1", ServiceID: "updater-service-1",
			Authentication: "assignment_bound_rotating_service_identity", Name: "Independent Updater", Status: "online",
			TransportMode: UpdateTransportPullV2, ExecutionHostID: "host-1", OwnershipEpoch: 2,
			Online: true, Version: "v2.0.0", LastHeartbeatAt: &now,
			HeartbeatSequence: 8, Capabilities: []UpdaterCapability{UpdaterCapabilityUpdate},
			DesiredRevision: 4, AppliedRevision: 3, Fence: 5, PolicyStatus: "pending", PolicyErrorCode: "active_job_pending",
			BootstrapEncryptionPublicKey:      "BG_wO5SSQc4drdQ1GeaWDgqFtBppoFwygQOqK84VlMoWPE91OlW_AdxT9sCwx-7ni0DG_30lqW4igrmJzvccFEo",
			BootstrapEncryptionKeyFingerprint: "SHA256:JWsb4ydF0dHJ1JEhzIe8RRxPLfp1bYm0yRCvrrYliuA",
		}},
		Hosts: []SystemUpdateHostStatus{{
			ProtocolVersion: 2, HostID: "host-1", Name: "Encoder Host", UpdaterID: "updater-1",
			Reachability: SystemUpdateReachable, ReachabilityCheckedAt: &now,
			UpdaterHealth: &UpdaterHealth{Status: "ready", Revision: 4},
			ApplicationProbe: &SystemUpdateHostApplicationProbe{
				Status:                          "ready",
				ApplicationRuntimeIdentityProbe: ApplicationRuntimeIdentityProbe{Version: "v2.0.0", ServiceID: "encoder-1", ServiceType: SystemUpdateTargetEncoderRecorder, ConfigRevision: 3},
			},
		}},
		Targets: []SystemUpdateTarget{{
			ProtocolVersion: 2, UpdaterID: "updater-1", Capabilities: []UpdaterCapability{UpdaterCapabilityUpdate},
			DesiredRevision: 4, AppliedRevision: 0, Fence: 5,
			UpdaterHealth:    &UpdaterHealth{Status: "ready", Revision: 4},
			ApplicationProbe: &ApplicationRuntimeIdentityProbe{Version: "v2.0.0", ServiceID: "worker-1", ServiceType: SystemUpdateTargetWorker, ConfigRevision: 3},
			TargetID:         "worker-1", TargetType: SystemUpdateTargetWorker, Name: "Worker",
			HostID: "host-1", UpdateAvailable: false, UpdaterOnline: true, Eligible: false, Busy: false,
		}},
		Jobs: []SystemUpdateJob{inventoryJob},
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
	root, ok := document.(map[string]any)
	if !ok {
		t.Fatalf("system update GET fixture is not an object: %T", document)
	}
	cloneDocument := func() map[string]any {
		t.Helper()
		encoded, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		var cloned map[string]any
		if err := json.Unmarshal(encoded, &cloned); err != nil {
			t.Fatal(err)
		}
		return cloned
	}
	updaterAt := func(value map[string]any) map[string]any {
		return value["updaters"].([]any)[0].(map[string]any)
	}
	hostAt := func(value map[string]any) map[string]any {
		return value["hosts"].([]any)[0].(map[string]any)
	}
	invalidCases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "unknown_updater_field", mutate: func(value map[string]any) {
			updaterAt(value)["credential"] = "must-not-be-accepted"
		}},
		{name: "unknown_host_field", mutate: func(value map[string]any) {
			hostAt(value)["ssh_port"] = 22
		}},
		{name: "transport_mode", mutate: func(value map[string]any) {
			updaterAt(value)["transport_mode"] = "grpc_v3"
		}},
		{name: "ownership_epoch", mutate: func(value map[string]any) {
			updaterAt(value)["ownership_epoch"] = -1
		}},
		{name: "desired_revision", mutate: func(value map[string]any) {
			updaterAt(value)["desired_revision"] = -1
		}},
		{name: "applied_revision", mutate: func(value map[string]any) {
			updaterAt(value)["applied_revision"] = -1
		}},
		{name: "removed_ssh_key_map", mutate: func(value map[string]any) {
			updaterAt(value)["ssh_client_public_keys"] = map[string]any{"host-1": "removed"}
		}},
		{name: "bootstrap_public_key", mutate: func(value map[string]any) {
			updaterAt(value)["bootstrap_encryption_public_key"] = "not-a-p256-public-key"
		}},
		{name: "bootstrap_fingerprint", mutate: func(value map[string]any) {
			updaterAt(value)["bootstrap_encryption_key_fingerprint"] = "sha256:wrong-case"
		}},
		{name: "removed_host_ssh_public_key", mutate: func(value map[string]any) {
			hostAt(value)["ssh_client_public_key"] = "removed"
		}},
	}
	for _, testCase := range invalidCases {
		t.Run("reject_"+testCase.name, func(t *testing.T) {
			invalid := cloneDocument()
			testCase.mutate(invalid)
			if err := schema.Validate(invalid); err == nil {
				t.Fatalf("system update GET schema accepted invalid %s", testCase.name)
			}
		})
	}

	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	paths := requireCharacterizationMap(t, control, "paths")
	operation := requireCharacterizationMap(t,
		requireCharacterizationMap(t, paths, "/system-updates"), "get")
	responses := requireCharacterizationMap(t, operation, "responses")
	success := resolveCharacterizationSchema(t, control, responses["200"])
	content := requireCharacterizationMap(t, success, "content")
	jsonContent := requireCharacterizationMap(t, content, "application/json")
	responseSchema := resolveCharacterizationSchema(t, control, jsonContent["schema"])
	components := requireCharacterizationMap(t, control, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")
	responseComponent := resolveCharacterizationSchema(t, control, schemas["SystemUpdatesResponse"])
	if !reflect.DeepEqual(responseSchema, responseComponent) {
		t.Fatal("GET /system-updates does not use the canonical SystemUpdatesResponse component")
	}
	assertExactCharacterizationProperties(t, responseSchema, []string{"updaters", "hosts", "targets", "jobs"})
	responseProperties := requireCharacterizationMap(t, responseSchema, "properties")
	for property, componentName := range map[string]string{
		"updaters": "SystemUpdateAgentStatus",
		"hosts":    "SystemUpdateHostStatus",
		"targets":  "SystemUpdateTarget",
		"jobs":     "SystemUpdateJob",
	} {
		arraySchema := requireCharacterizationMap(t, responseProperties, property)
		if arraySchema["type"] != "array" {
			t.Fatalf("GET /system-updates %s is %v, want array", property, arraySchema["type"])
		}
		itemSchema := resolveCharacterizationSchema(t, control, arraySchema["items"])
		componentSchema := resolveCharacterizationSchema(t, control, schemas[componentName])
		if !reflect.DeepEqual(itemSchema, componentSchema) {
			t.Fatalf("GET /system-updates %s does not use %s", property, componentName)
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
	if err := compileContractJSONSchema(t, "service-token-create-request.schema.json", "encoder-output-relay-capabilities.schema.json").Validate(issuedTokenInstance); err != nil {
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
	if contractSliceHas(manifest.Required, "minimum_agent_version") || contractSliceHas(manifest.Required, "protocol_major") {
		t.Fatal("release compatibility fields must be required by channel, not across both channels")
	}
	if !contractSliceHas(manifest.Properties["channel"].Enum, "host") ||
		!contractSliceHas(manifest.Properties["channel"].Enum, "docker") {
		t.Fatal("release manifest must support host and docker channels")
	}
	if manifest.Properties["schema_version"].Const != float64(2) {
		t.Fatalf("release manifest schema version must be 2, got %#v", manifest.Properties["schema_version"].Const)
	}
	for _, removed := range []string{"bundle_version", "generated_at"} {
		if _, ok := manifest.Properties[removed]; ok {
			t.Fatalf("release manifest retained removed alias %q", removed)
		}
	}
	if len(manifest.AllOf) != 2 || manifest.AllOf[0].Then == nil || manifest.AllOf[1].Then == nil {
		t.Fatalf("release manifest channel conditions changed: %#v", manifest.AllOf)
	}

	docker := manifest.AllOf[0].Then
	requireContractFields(t, docker.Required, "protocol_major")
	if manifest.Properties["protocol_major"].Const != float64(2) || docker.Not == nil || !contractSliceHas(docker.Not.Required, "minimum_agent_version") {
		t.Fatal("Docker releases must use protocol-major 2 and forbid version-based compatibility")
	}
	requireContractFields(t, manifest.AllOf[1].Then.Required, "minimum_agent_version")
	if manifest.Properties["minimum_agent_version"].Pattern != "^v[0-9]+\\.[0-9]+\\.[0-9]+$" {
		t.Fatal("minimum_agent_version must use the canonical release version format")
	}
	dockerComponents := docker.Properties["components"]
	if dockerComponents.MinItems != 6 || dockerComponents.MaxItems != 6 || dockerComponents.Items == nil || len(dockerComponents.Items.OneOf) != 2 {
		t.Fatalf("docker manifest must contain five images and one independent Updater metadata component: %#v", dockerComponents)
	}
	imageComponent := dockerComponents.Items.OneOf[0]
	updaterComponent := dockerComponents.Items.OneOf[1]
	requireContractFields(t, imageComponent.Required,
		"service", "source_version", "commit", "image", "manifest_digest", "platform_digests",
		"rollback_compatible", "database_schema",
	)
	requireContractFields(t, updaterComponent.Required, "service", "commit", "protocol_major")
	if len(updaterComponent.Required) != 3 || updaterComponent.Properties["service"].Const != "updater" || updaterComponent.Properties["protocol_major"].Const != float64(2) {
		t.Fatal("independent Updater component must contain only source identity and protocol-major metadata")
	}
	if len(dockerComponents.AllOf) != 6 {
		t.Fatalf("docker manifest must contain each known service exactly once: %#v", dockerComponents.AllOf)
	}
	if !strings.Contains(imageComponent.Properties["image"].Pattern, "ghcr\\.io/kome-lab/autostream-docker") {
		t.Fatal("docker images must use the fixed lowercase GHCR namespace")
	}
	if imageComponent.Properties["rollback_compatible"].Const != true {
		t.Fatal("Docker releases must explicitly allow rollback")
	}
	if imageComponent.Not == nil {
		t.Fatal("Docker components must reject host-only fields")
	}
	for _, forbidden := range imageComponent.Not.AnyOf {
		for _, policyField := range []string{"rollback_compatible", "database_schema"} {
			if contractSliceHas(forbidden.Required, policyField) {
				t.Fatalf("Docker schema both requires and forbids %q", policyField)
			}
		}
	}
	if schemas := imageComponent.Properties["database_schema"].Enum; !contractSliceHas(schemas, "none") || !contractSliceHas(schemas, "backward_compatible") || contractSliceHas(schemas, "irreversible") {
		t.Fatalf("Docker database_schema policy is unsafe: %#v", schemas)
	}
	expectedSchemas := []string{"backward_compatible", "none", "none", "backward_compatible", "none"}
	for i, expected := range expectedSchemas {
		if imageComponent.AllOf[i].Then == nil || imageComponent.AllOf[i].Then.Properties["database_schema"].Const != expected {
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
