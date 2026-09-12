package contracts

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestV2CleanBreakContractPolicy(t *testing.T) {
	bundle := readNormalizedOpenAPICharacterization(t, "control-api.json")
	policy := requireCharacterizationMap(t, bundle, "x-autostream-contract-policy")

	if fmt.Sprint(policy["contract_major"]) != "2" || policy["final_state"] != "v2_only" {
		t.Fatalf("clean-break major/final state=%v, want major 2 and v2_only", policy)
	}
	if fmt.Sprint(policy["unsupported_contract_major_status"]) != "426" ||
		fmt.Sprint(policy["invalid_payload_status"]) != "400" {
		t.Fatalf("clean-break major/payload status behavior drifted: %v", policy)
	}
	supportedMajors, ok := policy["supported_contract_majors"].([]any)
	if !ok || len(supportedMajors) != 1 || fmt.Sprint(supportedMajors[0]) != "2" {
		t.Fatalf("supported contract majors=%v, want only 2", supportedMajors)
	}
	if policy["uri_strategy"] != "preserve_existing_paths_no_v2_prefix" ||
		policy["unknown_field_behavior"] != "reject_fail_closed" ||
		policy["unknown_capability_behavior"] != "non_ready" {
		t.Fatalf("clean-break fail-closed policy drifted: %v", policy)
	}
	if policy["request_header"] != "X-AutoStream-Contract-Major" ||
		policy["response_header"] != "X-AutoStream-Contract-Major" ||
		policy["request_header_value"] != "2" || policy["response_header_value"] != "2" {
		t.Fatalf("contract-major header authority drifted: %v", policy)
	}

	errors := requireCharacterizationStringSet(t, policy, "stable_error_codes")
	for _, code := range []string{
		"contract_major_unsupported", "protocol_version_unsupported", "request_schema_invalid",
		"revision_conflict", "stale_generation", "stale_fence", "semantic_validation_failed",
	} {
		if _, ok := errors[code]; !ok {
			t.Fatalf("clean-break stable errors omit %q: %v", code, policy["stable_error_codes"])
		}
	}
	compatibility := requireCharacterizationMap(t, policy, "temporary_compatibility")
	if compatibility["owner"] != "V2-COMPAT-EOL-CONTRACTS-001" ||
		compatibility["removal_wave"] != "Execution Bundle 8" ||
		compatibility["final_state"] != "absent" {
		t.Fatalf("temporary compatibility metadata drifted: %v", compatibility)
	}

	components := requireCharacterizationMap(t, bundle, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")
	for _, name := range []string{"V2ContractError", "V2CapabilityNegotiation"} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("control API is missing clean-break component %s", name)
		}
	}

	paths := requireCharacterizationMap(t, bundle, "paths")
	methods := map[string]struct{}{
		"get": {}, "put": {}, "post": {}, "delete": {}, "options": {}, "head": {}, "patch": {}, "trace": {},
	}
	for pathName, rawPathItem := range paths {
		pathItem := resolveCharacterizationSchema(t, bundle, rawPathItem)
		if pathName == "/health" {
			health := resolveCharacterizationSchema(t, bundle, pathItem["get"])
			if health["x-autostream-contract-major-exempt"] != "health_non_json" {
				t.Fatalf("health exception is not exact: %v", health)
			}
			continue
		}
		parameters, ok := pathItem["parameters"].([]any)
		if !ok {
			t.Fatalf("%s has no standard contract-major path parameter", pathName)
		}
		headerFound := false
		for _, rawParameter := range parameters {
			parameter := resolveCharacterizationSchema(t, bundle, rawParameter)
			if parameter["name"] == "X-AutoStream-Contract-Major" && parameter["in"] == "header" && parameter["required"] == true {
				schema := requireCharacterizationMap(t, parameter, "schema")
				headerFound = schema["const"] == "2"
			}
		}
		if !headerFound {
			t.Fatalf("%s does not require exact contract major 2", pathName)
		}
		for method, rawOperation := range pathItem {
			if _, ok := methods[method]; !ok {
				continue
			}
			operation := resolveCharacterizationSchema(t, bundle, rawOperation)
			responses := requireCharacterizationMap(t, operation, "responses")
			if _, ok := responses["426"]; !ok {
				t.Fatalf("%s %s omits typed 426", method, pathName)
			}
			for status, rawResponse := range responses {
				response := resolveCharacterizationSchema(t, bundle, rawResponse)
				headers := requireCharacterizationMap(t, response, "headers")
				rawHeader, ok := headers["X-AutoStream-Contract-Major"]
				if !ok {
					t.Fatalf("%s %s response %s does not echo contract major", method, pathName, status)
				}
				header := resolveCharacterizationSchema(t, bundle, rawHeader)
				if requireCharacterizationMap(t, header, "schema")["const"] != "2" {
					t.Fatalf("%s %s response %s contract major is not exact 2", method, pathName, status)
				}
			}
		}
	}
}

func TestApplicationRuntimeIdentityProbeContract(t *testing.T) {
	tests := []struct {
		bundle      string
		serviceType string
		errorStatus string
	}{
		{bundle: "control-api.json", serviceType: "control_panel", errorStatus: "500"},
		{bundle: "discord-bot-api.json", serviceType: "discord_bot", errorStatus: "503"},
		{bundle: "encoder-recorder-api.json", serviceType: "encoder_recorder", errorStatus: "503"},
		{bundle: "observability-api.json", serviceType: "observability", errorStatus: "503"},
	}
	for _, test := range tests {
		t.Run(test.serviceType, func(t *testing.T) {
			bundle := readNormalizedOpenAPICharacterization(t, test.bundle)
			paths := requireCharacterizationMap(t, bundle, "paths")
			pathItem := requireCharacterizationMap(t, paths, "/updater/version")
			operation := requireCharacterizationMap(t, pathItem, "get")
			if operation["x-autostream-semantic-name"] != "application_runtime_identity_probe" ||
				operation["x-autostream-current-source-cache-control"] != "no_explicit_route_level_no_store" ||
				operation["x-autostream-v2-target-cache-control"] != "no-store" {
				t.Fatalf("application probe current/target distinction drifted: %v", operation)
			}
			responses := requireCharacterizationMap(t, operation, "responses")
			for _, status := range []string{"200", test.errorStatus, "405"} {
				response := resolveCharacterizationSchema(t, bundle, responses[status])
				headers := requireCharacterizationMap(t, response, "headers")
				cache := resolveCharacterizationSchema(t, bundle, headers["Cache-Control"])
				cacheSchema := requireCharacterizationMap(t, cache, "schema")
				if cacheSchema["const"] != "no-store" {
					t.Fatalf("%s %s Cache-Control=%v, want exact no-store", test.serviceType, status, cacheSchema)
				}
			}
			success := resolveCharacterizationSchema(t, bundle, responses["200"])
			content := requireCharacterizationMap(t, success, "content")
			jsonContent := requireCharacterizationMap(t, content, "application/json")
			probe := resolveCharacterizationSchema(t, bundle, jsonContent["schema"])
			assertExactCharacterizationProperties(t, probe,
				[]string{"version", "service_id", "service_type", "config_revision"})
			properties := requireCharacterizationMap(t, probe, "properties")
			serviceType := requireCharacterizationMap(t, properties, "service_type")
			if serviceType["const"] != test.serviceType {
				t.Fatalf("%s probe service_type=%v", test.serviceType, serviceType)
			}
		})
	}

	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	components := requireCharacterizationMap(t, control, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")
	worker := resolveCharacterizationSchema(t, control, schemas["WorkerApplicationRuntimeIdentityProbe"])
	assertExactCharacterizationProperties(t, worker,
		[]string{"version", "service_id", "service_type", "config_revision"})
	workerProperties := requireCharacterizationMap(t, worker, "properties")
	if requireCharacterizationMap(t, workerProperties, "service_type")["const"] != "worker" {
		t.Fatalf("Worker application probe is not independently frozen: %v", worker)
	}
	if _, exists := workerProperties["updater_id"]; exists {
		t.Fatal("Updater health identity must not substitute for the Worker application probe")
	}
	pathItems := requireCharacterizationMap(t, components, "pathItems")
	workerPath := resolveCharacterizationSchema(t, control, pathItems["WorkerApplicationRuntimeIdentityProbe"])
	workerOperation := resolveCharacterizationSchema(t, control, workerPath["get"])
	if workerOperation["x-autostream-current-source-cache-control"] != "no_explicit_route_level_no_store" ||
		workerOperation["x-autostream-v2-target-cache-control"] != "no-store" {
		t.Fatalf("Worker current/target cache contract drifted: %v", workerOperation)
	}
	workerResponses := requireCharacterizationMap(t, workerOperation, "responses")
	for _, status := range []string{"200", "503", "405"} {
		response := resolveCharacterizationSchema(t, control, workerResponses[status])
		headers := requireCharacterizationMap(t, response, "headers")
		cache := resolveCharacterizationSchema(t, control, headers["Cache-Control"])
		if requireCharacterizationMap(t, cache, "schema")["const"] != "no-store" {
			t.Fatalf("Worker %s response lacks exact no-store", status)
		}
	}
}

func TestV2UpdaterProtocolAndRemediationAuthority(t *testing.T) {
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	components := requireCharacterizationMap(t, control, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")

	command := resolveCharacterizationSchema(t, control, schemas["UpdaterCommandEnvelope"])
	assertCharacterizationRequired(t, command,
		"protocol_version", "command_id", "idempotency_key", "issuer", "canonical_payload_digest",
		"mutation_authorization", "desired_operation", "audit_correlation_id")
	commandProperties := requireCharacterizationMap(t, command, "properties")
	if fmt.Sprint(requireCharacterizationMap(t, commandProperties, "protocol_version")["const"]) != "2" {
		t.Fatalf("Updater command is not protocol major 2: %v", commandProperties["protocol_version"])
	}
	for _, forbidden := range []string{"shell", "command", "argv", "environment", "path", "url", "database_credentials", "credentials", "token", "stdout", "stderr"} {
		if _, exists := commandProperties[forbidden]; exists {
			t.Fatalf("Updater command exposes forbidden arbitrary/secret field %q", forbidden)
		}
	}
	authorization := resolveCharacterizationSchema(t, control, schemas["UpdaterMutationAuthorization"])
	assertCharacterizationRequired(t, authorization,
		"authorization_id", "nonce_id", "job_id", "updater_id", "host_id", "action_type", "target",
		"canonical_argument_digest", "desired_revision", "fence", "expires_at", "required_capability", "one_time")
	authorizationProperties := requireCharacterizationMap(t, authorization, "properties")
	actionType := requireCharacterizationMap(t, authorizationProperties, "action_type")
	actions := characterizationStringSlice(t, actionType["enum"])
	if !reflect.DeepEqual(actions, []string{
		"host.systemd", "host.docker", "host.update", "host.bootstrap", "host.port", "host.self_update",
	}) {
		t.Fatalf("Updater command allowlist=%v", actions)
	}

	result := resolveCharacterizationSchema(t, control, schemas["UpdaterResultEnvelope"])
	assertCharacterizationRequired(t, result,
		"protocol_version", "command_id", "job_id", "updater_id", "host_id", "lease_id", "lease_generation", "idempotency_key",
		"canonical_payload_digest", "authorization_id", "desired_revision", "fence", "outcome", "status",
		"automatic_resend_allowed", "audit_correlation_id", "evidence")
	resultProperties := requireCharacterizationMap(t, result, "properties")
	outcomeSchema := requireCharacterizationMap(t, resultProperties, "outcome")
	outcomes := requireCharacterizationStringSet(t, outcomeSchema, "enum")
	if _, ok := outcomes["ambiguous"]; !ok {
		t.Fatalf("Updater result omits ambiguous outcome: %v", resultProperties["outcome"])
	}
	for _, name := range []string{
		"UpdaterDesiredOperation", "UpdaterLeaseEnvelope", "UpdaterProgressEnvelope", "UpdaterHeartbeat",
		"UpdaterSafeError", "UpdaterLocalJournalBoundary", "UpdaterRuntimeTokenRotationCredentialClaimRequest",
		"UpdaterMutationGrantBinding", "UpdaterMutationGrantIssueRequest", "UpdaterMutationGrantIssueResponse",
		"UpdaterMutationGrantConsumeRequest",
	} {
		if _, exists := schemas[name]; !exists {
			t.Fatalf("control API is missing Updater component %s", name)
		}
	}
	lease := resolveCharacterizationSchema(t, control, schemas["UpdaterLeaseEnvelope"])
	assertExactCharacterizationProperties(t, lease, []string{"protocol_version", "lease_id", "lease_generation", "lease_expires_at", "command"})
	progress := resolveCharacterizationSchema(t, control, schemas["UpdaterProgressEnvelope"])
	assertCharacterizationRequired(t, progress, "lease_id", "lease_generation", "sequence", "progress")
	target := resolveCharacterizationSchema(t, control, schemas["UpdaterTargetIdentity"])
	assertCharacterizationRequired(t, target, "target_kind", "service_id", "service_type", "deployment_mode")
	claimRequest := resolveCharacterizationSchema(t, control, schemas["UpdaterRuntimeTokenRotationCredentialClaimRequest"])
	assertExactCharacterizationProperties(t, claimRequest, []string{"expected_revision", "claim_id"})
	grantBinding := resolveCharacterizationSchema(t, control, schemas["UpdaterMutationGrantBinding"])
	assertExactCharacterizationProperties(t, grantBinding, []string{"lease", "operation", "session_id"})
	grantOperation := requireCharacterizationMap(t, requireCharacterizationMap(t, grantBinding, "properties"), "operation")
	if !reflect.DeepEqual(characterizationStringSlice(t, grantOperation["enum"]), []string{
		"apply", "reconcile", "port_reconfigure", "port_reconfigure_reconcile", "bootstrap", "bootstrap_reconcile",
		"host_self_update_stage", "host_self_update_activate", "host_self_update_reconcile",
	}) {
		t.Fatalf("Updater Local Executor operation allowlist drifted: %v", grantOperation)
	}
	grantOperationMap := requireCharacterizationMap(t, grantBinding, "x-autostream-operation-desired-map")
	if grantOperationMap["apply"] != "software_update" || grantOperationMap["bootstrap"] != "bootstrap" ||
		grantOperationMap["port_reconfigure"] != "port_reconfigure" ||
		grantOperationMap["host_self_update_activate"] != "host_self_update" {
		t.Fatalf("Updater Local Executor operation/desired map drifted: %v", grantOperationMap)
	}
	grantResponse := resolveCharacterizationSchema(t, control, schemas["UpdaterMutationGrantIssueResponse"])
	assertExactCharacterizationProperties(t, grantResponse, []string{"grant_token", "expires_at"})
	updaterAuthority := requireCharacterizationMap(t, control, "x-autostream-updater-v2-authority")
	if _, exists := updaterAuthority["embedded_legacy_transition"]; exists {
		t.Fatal("Updater v2 authority retained removed embedded-runtime transition metadata")
	}
	digestAuthority := requireCharacterizationMap(t, updaterAuthority, "canonical_command_digest")
	if digestAuthority["serialization"] != "RFC8785_JCS" ||
		digestAuthority["timestamp_lexical_form"] != "canonical_utc_rfc3339nano_no_trailing_fractional_zeroes" ||
		digestAuthority["payload_and_argument_digests_must_match"] != true {
		t.Fatalf("Updater JCS digest authority drifted: %v", digestAuthority)
	}
	rotationAuthority := requireCharacterizationMap(t, updaterAuthority, "runtime_token_rotation")
	if rotationAuthority["raw_replacement_token_surface"] != "credential_claim_response_only" ||
		rotationAuthority["cache_control"] != "no-store" || rotationAuthority["command_allowed"] != false ||
		rotationAuthority["local_journal_allowed"] != false {
		t.Fatalf("Updater runtime-token rotation boundary drifted: %v", rotationAuthority)
	}
	grantAuthority := requireCharacterizationMap(t, updaterAuthority, "local_executor_mutation_grant")
	if grantAuthority["opaque_grant_surface"] != "issue_response_and_authorization_header_only" ||
		grantAuthority["cache_control"] != "no-store" || grantAuthority["command_allowed"] != false ||
		grantAuthority["local_journal_allowed"] != false {
		t.Fatalf("Updater Local Executor grant boundary drifted: %v", grantAuthority)
	}

	authority := requireCharacterizationMap(t, control, "x-autostream-remediation-authority")
	if authority["authorization_orchestration_audit"] != "control_panel" ||
		authority["host_system_execution"] != "updater" ||
		authority["observability_role"] != "detect_propose_evidence" {
		t.Fatalf("remediation authority drifted: %v", authority)
	}
	grant := resolveCharacterizationSchema(t, control, schemas["RemediationGrant"])
	assertCharacterizationRequired(t, grant,
		"authorization_id", "authorization_nonce_id", "proposal_id", "request_origin", "executor", "target",
		"action_type", "idempotency_key", "canonical_argument_digest", "desired_revision", "fence",
		"expires_at", "capability", "one_time", "audit_correlation_id")
	grantProperties := requireCharacterizationMap(t, grant, "properties")
	for _, forbidden := range []string{"shell", "command", "argv", "token", "credentials", "database_credentials"} {
		if _, exists := grantProperties[forbidden]; exists {
			t.Fatalf("remediation grant exposes forbidden field %q", forbidden)
		}
	}

	observability := readNormalizedOpenAPICharacterization(t, "observability-api.json")
	observabilityAuthority := requireCharacterizationMap(t, observability, "x-autostream-remediation-authority")
	if observabilityAuthority["may_mint_cross_service_grant"] != false ||
		observabilityAuthority["may_execute_host_runtime"] != false ||
		observabilityAuthority["may_call_updater_directly"] != false {
		t.Fatalf("Observability gained remediation execution authority: %v", observabilityAuthority)
	}
}

func TestV2UpdaterCrossPlaneAuthority(t *testing.T) {
	create := compileNormalizedOpenAPISchema(t, "control-api.json",
		"/paths/~1system-updates/post/requestBody/content/application~1json/schema")
	validCreate := map[string]any{
		"protocol_version": 2, "operation": "software_update", "target_id": "worker-1",
		"strategy": "maintenance", "idempotency_key": "idem-1", "desired_revision": 12,
		"fence": 4, "required_capability": "host.update",
	}
	assertV2SchemaFixture(t, create, validCreate, true)
	wrongCapability := cloneV2Fixture(t, validCreate)
	wrongCapability["required_capability"] = "host.port"
	assertV2SchemaFixture(t, create, wrongCapability, false)

	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	paths := requireCharacterizationMap(t, control, "paths")
	heartbeat := resolveCharacterizationSchema(t, control,
		requireCharacterizationMap(t, requireCharacterizationMap(t, paths, "/services/heartbeat"), "post"))
	claim := resolveCharacterizationSchema(t, control,
		requireCharacterizationMap(t, requireCharacterizationMap(t, paths, "/services/update-jobs/claim"), "post"))
	report := resolveCharacterizationSchema(t, control,
		requireCharacterizationMap(t, requireCharacterizationMap(t, paths, "/services/update-jobs/{id}/report"), "post"))
	for name, value := range map[string]any{"heartbeat": heartbeat, "claim": claim, "report": report} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		text := string(encoded)
		for _, required := range map[string][]string{
			"heartbeat": {"UpdaterHeartbeat"},
			"claim":     {"UpdaterLeaseEnvelope", "UpdateAgentClearActiveJobResponse"},
			"report":    {"UpdaterProgressEnvelope", "UpdaterResultEnvelope"},
		}[name] {
			if !strings.Contains(text, required) {
				t.Fatalf("%s endpoint does not expose %s: %s", name, required, text)
			}
		}
		if name == "claim" && strings.Contains(text, "UpdateAgentLeaseClaimResponse") {
			t.Fatalf("v2 claim endpoint still accepts the legacy lease response: %s", text)
		}
	}
	for pathName, required := range map[string][]string{
		"/services/update-jobs/{id}/mutation-grants":         {"UpdaterMutationGrantIssueRequest", "UpdateAgentMutationGrantIssueRequest"},
		"/services/update-jobs/{id}/mutation-grants/consume": {"UpdaterMutationGrantConsumeRequest", "UpdateAgentMutationGrantConsumeRequest"},
	} {
		operation := resolveCharacterizationSchema(t, control,
			requireCharacterizationMap(t, requireCharacterizationMap(t, paths, pathName), "post"))
		requestBody := requireCharacterizationMap(t, operation, "requestBody")
		content := requireCharacterizationMap(t, requestBody, "content")
		media := requireCharacterizationMap(t, content, "application/json")
		requestSchema := requireCharacterizationMap(t, media, "schema")
		alternatives, ok := requestSchema["oneOf"].([]any)
		if !ok || len(alternatives) != len(required) {
			t.Fatalf("%s must expose exactly %d mixed-fleet request alternatives", pathName, len(required))
		}
		schemas := requireCharacterizationMap(t, requireCharacterizationMap(t, control, "components"), "schemas")
		for _, schemaName := range required {
			// Bundling may replace a named component alias with its canonical ref.
			// Require each expected schema once after resolving those references.
			expected := resolveCharacterizationSchema(t, control, schemas[schemaName])
			matches := 0
			for _, alternative := range alternatives {
				if reflect.DeepEqual(resolveCharacterizationSchema(t, control, alternative), expected) {
					matches++
				}
			}
			if matches != 1 {
				t.Errorf("%s exposes mixed-fleet schema %s %d times, want exactly once", pathName, schemaName, matches)
			}
		}
	}

	assertGoJSONFields(t, reflect.TypeOf(SystemUpdateCreateRequest{}),
		"protocol_version", "desired_revision", "fence", "required_capability")
	assertGoJSONFields(t, reflect.TypeOf(SystemUpdateTarget{}),
		"protocol_version", "capabilities", "desired_revision", "applied_revision", "fence",
		"updater_health", "application_probe", "safe_error")
	assertGoJSONFields(t, reflect.TypeOf(SystemUpdateJob{}),
		"protocol_version", "authorization_id", "canonical_payload_digest", "desired_revision", "fence",
		"outcome", "required_capability", "automatic_resend_allowed", "safe_error")
	assertGoJSONFields(t, reflect.TypeOf(SystemUpdateAgentStatus{}),
		"protocol_version", "host_id", "service_id", "authentication", "heartbeat_sequence", "capabilities", "fence")
	assertGoJSONFields(t, reflect.TypeOf(SystemUpdateHostStatus{}),
		"protocol_version", "updater_health", "application_probe")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterCommandEnvelope{}),
		"protocol_version", "command_id", "issuer", "idempotency_key", "canonical_payload_digest",
		"mutation_authorization", "desired_operation", "audit_correlation_id")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterTargetIdentity{}),
		"target_kind", "service_id", "service_type", "deployment_mode", "expected_config_revision", "execution_host_id")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterLeaseEnvelope{}),
		"protocol_version", "lease_id", "lease_generation", "lease_expires_at", "command")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterProgressEnvelope{}), "lease_id", "lease_generation")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterResultEnvelope{}), "lease_id", "lease_generation")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterRuntimeTokenRotationCredentialClaimRequest{}), "expected_revision", "claim_id")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterMutationGrantBinding{}), "lease", "operation", "session_id")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterMutationGrantIssueRequest{}), "binding")
	assertGoJSONFields(t, reflect.TypeOf(UpdaterMutationGrantConsumeRequest{}), "binding")
	assertGoJSONFields(t, reflect.TypeOf(UpdateAgentClearActiveJobResponse{}), "clear_active_job_id")
	heartbeatSchema := resolveCharacterizationSchema(t, control,
		requireCharacterizationMap(t, requireCharacterizationMap(t, control, "components"), "schemas")["UpdaterHeartbeat"])
	assertExactGoJSONSchemaParity(t, reflect.TypeOf(UpdaterHeartbeat{}), heartbeatSchema)
}
