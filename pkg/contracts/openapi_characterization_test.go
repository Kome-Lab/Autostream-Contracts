package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const pinnedRedoclyVersion = "2.39.0"

type openAPIFingerprintManifest struct {
	FormatVersion  int                     `json:"format_version"`
	RedoclyVersion string                  `json:"redocly_version"`
	APIs           []openAPIFingerprintAPI `json:"apis"`
}

type openAPIFingerprintAPI struct {
	API                        string `json:"api"`
	NormalizedFile             string `json:"normalized_file"`
	NormalizedSHA256           string `json:"normalized_sha256"`
	RefLayoutIndependentSHA256 string `json:"ref_layout_independent_sha256"`
}

type openAPISourceInventory struct {
	FormatVersion       int                 `json:"format_version"`
	EntrypointCount     int                 `json:"entrypoint_count"`
	Entrypoints         []string            `json:"entrypoints"`
	SourceFileCount     int                 `json:"source_file_count"`
	SourceFiles         []openAPISourceFile `json:"source_files"`
	Refs                []map[string]any    `json:"refs"`
	ExternalNetworkRefs []map[string]any    `json:"external_network_refs"`
	MissingLocalRefs    []map[string]any    `json:"missing_local_refs"`
}

type openAPISourceFile struct {
	Path                   string `json:"path"`
	NormalizedSourceSHA256 string `json:"normalized_source_sha256"`
}

type openAPILintBaseline struct {
	FormatVersion  int              `json:"format_version"`
	RedoclyVersion string           `json:"redocly_version"`
	APIs           []openAPILintAPI `json:"apis"`
}

type openAPILintAPI struct {
	API          string           `json:"api"`
	LintExitCode int              `json:"lint_exit_code"`
	ErrorCount   int              `json:"error_count"`
	WarningCount int              `json:"warning_count"`
	FindingCount int              `json:"finding_count"`
	Findings     []map[string]any `json:"findings"`
}

func TestOpenAPICharacterizationArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "characterization", "generated")
	var fingerprints openAPIFingerprintManifest
	readCharacterizationJSON(t, filepath.Join(root, "openapi", "fingerprints.json"), &fingerprints)
	if fingerprints.RedoclyVersion != pinnedRedoclyVersion {
		t.Fatalf("Redocly version=%q, want %q", fingerprints.RedoclyVersion, pinnedRedoclyVersion)
	}
	if len(fingerprints.APIs) != 4 {
		t.Fatalf("OpenAPI fingerprint count=%d, want 4", len(fingerprints.APIs))
	}

	var sourceInventory openAPISourceInventory
	readCharacterizationJSON(t, filepath.Join(root, "openapi", "source-inventory.json"), &sourceInventory)
	verifyOpenAPISourceInventory(t, sourceInventory)

	for _, api := range fingerprints.APIs {
		if !validSHA256(api.NormalizedSHA256) || !validSHA256(api.RefLayoutIndependentSHA256) {
			t.Fatalf("%s has invalid bundle fingerprint metadata", api.API)
		}
		normalizedPath := filepath.Join(root, filepath.FromSlash(api.NormalizedFile))
		var normalized any
		readCharacterizationJSON(t, normalizedPath, &normalized)
		compact := canonicalCharacterizationJSON(t, normalized)
		digest := sha256.Sum256(compact)
		if got := hex.EncodeToString(digest[:]); got != api.NormalizedSHA256 {
			t.Fatalf("%s normalized SHA-256=%s, want %s", api.API, got, api.NormalizedSHA256)
		}
	}

	semanticPath := filepath.Join(root, "openapi", "semantic-inventory.json")
	var semanticInventory struct {
		FormatVersion  int              `json:"format_version"`
		RedoclyVersion string           `json:"redocly_version"`
		APIs           []map[string]any `json:"apis"`
	}
	readCharacterizationJSON(t, semanticPath, &semanticInventory)
	if semanticInventory.RedoclyVersion != pinnedRedoclyVersion || len(semanticInventory.APIs) != 4 {
		t.Fatalf("invalid semantic inventory header: version=%q APIs=%d", semanticInventory.RedoclyVersion, len(semanticInventory.APIs))
	}
	for _, api := range semanticInventory.APIs {
		verifyOpenAPISemanticInventory(t, api)
	}
	verifyControlAPISemanticProjection(t, root, semanticInventory.APIs)
	verifyBundledJobGenerationContract(t, root)

	var lintBaseline openAPILintBaseline
	readCharacterizationJSON(t, filepath.Join(root, "openapi", "lint-baseline.json"), &lintBaseline)
	verifyOpenAPILintBaseline(t, lintBaseline)
}

func TestCharacterizationReportsAreMachineReadable(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "characterization")
	var initial map[string]any
	readCharacterizationJSON(t, filepath.Join(root, "initial-state.json"), &initial)
	if initial["expected_base_matched"] != true || fmt.Sprint(initial["exported_package_identifier_count"]) != "479" {
		t.Fatalf("invalid initial-state characterization: %v", initial)
	}

	var authority map[string]any
	readCharacterizationJSON(t, filepath.Join(root, "authority.json"), &authority)
	if authority["classification"] != "RAW_SOURCE_AUTHORITY" || authority["authority_changed_by_characterization_task"] != false {
		t.Fatalf("invalid authority characterization: %v", authority)
	}

	var consumers map[string]any
	readCharacterizationJSON(t, filepath.Join(root, "consumer-usage.json"), &consumers)
	repositories, ok := consumers["repositories"].([]any)
	if !ok || len(repositories) != 5 || consumers["workspace_external_consumer_absence_proven"] != false {
		t.Fatalf("invalid consumer characterization: %v", consumers)
	}
}

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
		encoded, err := json.Marshal(operation)
		if err != nil {
			t.Fatal(err)
		}
		for _, schemaName := range required {
			if !strings.Contains(string(encoded), schemaName) {
				t.Fatalf("%s does not expose mixed-fleet schema %s: %s", pathName, schemaName, encoded)
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

func TestV2CoreContractNegativeSensitivity(t *testing.T) {
	contractError := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/V2ContractError")
	assertV2SchemaFixture(t, contractError, map[string]any{
		"contract_major": 2, "code": "contract_major_unsupported",
		"message": "supported contract major is 2", "retryable": false, "expected_major": 2,
	}, true)
	assertV2SchemaFixture(t, contractError, map[string]any{
		"contract_major": 1, "code": "contract_major_unsupported",
		"message": "supported contract major is 2", "retryable": false, "expected_major": 2,
	}, false)

	capability := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/V2CapabilityNegotiation")
	readyCapability := map[string]any{
		"protocol_version": 2, "contract_major": 2,
		"capabilities":         []any{map[string]any{"name": "host.update", "required": true, "supported": true}},
		"unknown_capabilities": []any{}, "readiness": "ready", "revision": 3,
	}
	assertV2SchemaFixture(t, capability, readyCapability, true)
	unknownCapability := map[string]any{
		"protocol_version": 2, "contract_major": 2,
		"capabilities":         []any{map[string]any{"name": "future.capability", "required": true, "supported": false}},
		"unknown_capabilities": []any{"future.capability"}, "readiness": "not_ready", "revision": 3,
		"safe_error": map[string]any{
			"contract_major": 2, "code": "semantic_validation_failed",
			"message": "required capability is unknown", "retryable": false,
		},
	}
	assertV2SchemaFixture(t, capability, unknownCapability, true)
	unknownMarkedReady := cloneV2Fixture(t, unknownCapability)
	unknownMarkedReady["readiness"] = "ready"
	delete(unknownMarkedReady, "safe_error")
	assertV2SchemaFixture(t, capability, unknownMarkedReady, false)

	command := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterCommandEnvelope")
	validCommand := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "idempotency_key": "idem-1",
		"issuer": map[string]any{
			"service_id": "control-panel-1", "service_type": "control_panel",
			"authentication": "assignment_bound_rotating_service_identity", "permission": "updates.authorize",
		},
		"canonical_payload_digest": "sha256:" + strings.Repeat("a", 64),
		"audit_correlation_id":     "audit-1",
		"desired_operation": map[string]any{
			"operation": "software_update",
			"software_update": map[string]any{
				"expected_current_version": "v2.0.0", "target_version": "v2.0.1", "strategy": "when_idle",
			},
		},
		"mutation_authorization": map[string]any{
			"authorization_id": "authorization-1", "nonce_id": "nonce-12345678901234567890",
			"job_id": "job-1", "updater_id": "updater-1", "host_id": "host-1", "action_type": "host.update",
			"target": map[string]any{
				"target_kind": "application", "service_id": "worker-1", "service_type": "worker", "deployment_mode": "systemd",
				"expected_config_revision": 8,
			},
			"canonical_argument_digest": "sha256:" + strings.Repeat("b", 64),
			"desired_revision":          9, "fence": 4, "expires_at": "2026-08-31T01:00:00Z",
			"required_capability": "host.update", "one_time": true,
		},
	}
	assertV2SchemaFixture(t, command, validCommand, true)
	for name, mutate := range map[string]func(map[string]any){
		"unsupported_protocol": func(value map[string]any) { value["protocol_version"] = 1 },
		"arbitrary_shell":      func(value map[string]any) { value["shell"] = "powershell" },
		"shared_database":      func(value map[string]any) { value["database_credentials"] = "forbidden" },
		"missing_fence": func(value map[string]any) {
			delete(value["mutation_authorization"].(map[string]any), "fence")
		},
		"missing_identity": func(value map[string]any) {
			authorization := value["mutation_authorization"].(map[string]any)
			delete(authorization["target"].(map[string]any), "service_id")
		},
		"mismatched_capability": func(value map[string]any) {
			value["mutation_authorization"].(map[string]any)["required_capability"] = "host.port"
		},
		"transplanted_host": func(value map[string]any) {
			delete(value["mutation_authorization"].(map[string]any), "host_id")
		},
		"missing_desired_operation": func(value map[string]any) {
			delete(value, "desired_operation")
		},
		"generic_desired_token": func(value map[string]any) {
			value["desired_operation"].(map[string]any)["token"] = "forbidden"
		},
		"multiple_desired_variants": func(value map[string]any) {
			value["desired_operation"].(map[string]any)["bootstrap"] = map[string]any{
				"expected_state": "absent", "target_version": "v2.0.1",
			}
		},
	} {
		t.Run("updater_command_"+name, func(t *testing.T) {
			fixture := cloneV2Fixture(t, validCommand)
			mutate(fixture)
			assertV2SchemaFixture(t, command, fixture, false)
		})
	}

	result := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterResultEnvelope")
	validResult := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "job_id": "job-1",
		"updater_id": "updater-1", "host_id": "host-1", "lease_id": "lease-1", "lease_generation": 3, "idempotency_key": "idem-1",
		"canonical_payload_digest": "sha256:" + strings.Repeat("a", 64), "authorization_id": "authorization-1",
		"desired_revision": 9, "applied_revision": 8, "fence": 4,
		"outcome": "ambiguous", "status": "reconciling", "automatic_resend_allowed": false,
		"audit_correlation_id": "audit-1",
		"safe_error": map[string]any{
			"code": "outcome_ambiguous", "message": "updater outcome requires reconciliation", "retryable": false,
		},
		"evidence": []any{map[string]any{
			"evidence_code": "outcome_ambiguous", "observed_at": "2026-08-31T00:30:00Z", "observed_revision": 8,
		}},
	}
	assertV2SchemaFixture(t, result, validResult, true)
	resultWithCallerText := cloneV2Fixture(t, validResult)
	resultWithCallerText["safe_error"].(map[string]any)["message"] = "/var/lib/autostream/runtime-token"
	assertV2SchemaFixture(t, result, resultWithCallerText, false)
	ambiguousReportedSucceeded := cloneV2Fixture(t, validResult)
	ambiguousReportedSucceeded["status"] = "succeeded"
	assertV2SchemaFixture(t, result, ambiguousReportedSucceeded, false)
	resultWithRawOutput := cloneV2Fixture(t, validResult)
	resultWithRawOutput["evidence"].([]any)[0].(map[string]any)["stdout"] = "raw output"
	assertV2SchemaFixture(t, result, resultWithRawOutput, false)
	contradictoryTerminal := cloneV2Fixture(t, validResult)
	contradictoryTerminal["outcome"] = "succeeded"
	contradictoryTerminal["status"] = "failed"
	delete(contradictoryTerminal, "safe_error")
	assertV2SchemaFixture(t, result, contradictoryTerminal, false)
	failedWithoutSafeError := cloneV2Fixture(t, validResult)
	failedWithoutSafeError["outcome"] = "failed"
	failedWithoutSafeError["status"] = "failed"
	delete(failedWithoutSafeError, "safe_error")
	assertV2SchemaFixture(t, result, failedWithoutSafeError, false)

	lease := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterLeaseEnvelope")
	validLease := map[string]any{
		"protocol_version": 2, "lease_id": "lease-1", "lease_generation": 3,
		"lease_expires_at": "2026-08-31T00:45:00Z",
		"command":          validCommand,
	}
	assertV2SchemaFixture(t, lease, validLease, true)
	staleLease := cloneV2Fixture(t, validLease)
	staleLease["lease_generation"] = 0
	assertV2SchemaFixture(t, lease, staleLease, false)

	progress := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterProgressEnvelope")
	validProgress := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "job_id": "job-1",
		"updater_id": "updater-1", "host_id": "host-1", "lease_id": "lease-1", "lease_generation": 3, "sequence": 2,
		"phase": "executing", "progress": 50, "desired_revision": 9, "fence": 4,
		"audit_correlation_id": "audit-1", "observed_at": "2026-08-31T00:30:00Z",
	}
	assertV2SchemaFixture(t, progress, validProgress, true)
	progressWithLog := cloneV2Fixture(t, validProgress)
	progressWithLog["raw_log"] = "forbidden"
	assertV2SchemaFixture(t, progress, progressWithLog, false)

	grant := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/RemediationGrant")
	validGrant := map[string]any{
		"authorization_id": "authorization-1", "authorization_nonce_id": "nonce-12345678901234567890",
		"proposal_id": "proposal-1",
		"request_origin": map[string]any{
			"origin_type": "service", "principal_id": "observability-1", "service_id": "observability-1",
			"service_type": "observability", "permission": "remediation.execute",
		},
		"executor": map[string]any{
			"service_id": "updater-1", "authority": "updater", "execution_scope": "host_system",
		},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "idempotency_key": "idem-1",
		"canonical_argument_digest": "sha256:" + strings.Repeat("c", 64), "desired_revision": 9,
		"fence": 4, "expires_at": "2026-08-31T01:00:00Z", "capability": "host.update",
		"one_time": true, "audit_correlation_id": "audit-1",
	}
	assertV2SchemaFixture(t, grant, validGrant, true)
	observabilityExecutor := cloneV2Fixture(t, validGrant)
	observabilityExecutor["executor"].(map[string]any)["authority"] = "observability"
	assertV2SchemaFixture(t, grant, observabilityExecutor, false)
	grantWithCredential := cloneV2Fixture(t, validGrant)
	grantWithCredential["credentials"] = "forbidden"
	assertV2SchemaFixture(t, grant, grantWithCredential, false)
	wrongPermission := cloneV2Fixture(t, validGrant)
	wrongPermission["request_origin"].(map[string]any)["permission"] = "updates.authorize"
	assertV2SchemaFixture(t, grant, wrongPermission, false)
	mismatchedGrantCapability := cloneV2Fixture(t, validGrant)
	mismatchedGrantCapability["capability"] = "host.port"
	assertV2SchemaFixture(t, grant, mismatchedGrantCapability, false)
	applicationGrant := cloneV2Fixture(t, validGrant)
	applicationGrant["action_type"] = "application.retry_package_remux"
	applicationGrant["capability"] = "application.retry_package_remux"
	applicationGrant["executor"] = map[string]any{
		"authority": "target_application", "execution_scope": "application_local",
	}
	applicationGrant["target"] = map[string]any{
		"service_id": "encoder-1", "service_type": "encoder_recorder", "incident_id": "incident-1",
	}
	assertV2SchemaFixture(t, grant, applicationGrant, true)
	contradictoryApplicationExecutor := cloneV2Fixture(t, applicationGrant)
	contradictoryApplicationExecutor["executor"].(map[string]any)["service_id"] = "worker-1"
	assertV2SchemaFixture(t, grant, contradictoryApplicationExecutor, false)
	wrongApplicationTarget := cloneV2Fixture(t, applicationGrant)
	wrongApplicationTarget["target"].(map[string]any)["service_type"] = "worker"
	assertV2SchemaFixture(t, grant, wrongApplicationTarget, false)
	hostWithApplicationExecutor := cloneV2Fixture(t, validGrant)
	hostWithApplicationExecutor["executor"] = applicationGrant["executor"]
	assertV2SchemaFixture(t, grant, hostWithApplicationExecutor, false)

	remediationResult := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/RemediationResultEvidence")
	validRemediationResult := map[string]any{
		"authorization_id": "authorization-1", "proposal_id": "proposal-1",
		"executor": map[string]any{
			"service_id": "updater-1", "authority": "updater", "execution_scope": "host_system",
		},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "idempotency_key": "idem-1",
		"canonical_argument_digest": "sha256:" + strings.Repeat("c", 64), "desired_revision": 9,
		"applied_revision": 8, "fence": 4, "result": "ambiguous", "reconciliation_required": true,
		"automatic_resend_allowed": false,
		"evidence": []any{map[string]any{
			"evidence_code": "outcome_ambiguous", "observed_at": "2026-08-31T00:30:00Z", "observed_revision": 8,
		}},
		"audit_correlation_id": "audit-1", "completed_at": "2026-08-31T00:31:00Z",
	}
	assertV2SchemaFixture(t, remediationResult, validRemediationResult, true)
	blindResend := cloneV2Fixture(t, validRemediationResult)
	blindResend["automatic_resend_allowed"] = true
	assertV2SchemaFixture(t, remediationResult, blindResend, false)
	falseReconciliation := cloneV2Fixture(t, validRemediationResult)
	falseReconciliation["reconciliation_required"] = false
	assertV2SchemaFixture(t, remediationResult, falseReconciliation, false)

	proposal := compileNormalizedOpenAPISchema(t, "observability-api.json", "/components/schemas/RemediationProposal")
	validProposal := map[string]any{
		"proposal_id": "proposal-1", "incident_id": "incident-1",
		"detector":    map[string]any{"service_id": "observability-1", "service_type": "observability"},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "proposal_revision": 2, "required_capability": "host.update",
		"evidence": []any{map[string]any{
			"evidence_code": "host_symptom_confirmed", "observed_at": "2026-08-31T00:15:00Z", "observed_revision": 2,
		}},
		"audit_correlation_id": "audit-1", "observed_at": "2026-08-31T00:15:00Z",
		"control_panel_authorization_required": true,
	}
	assertV2SchemaFixture(t, proposal, validProposal, true)
	proposalWithExecutor := cloneV2Fixture(t, validProposal)
	proposalWithExecutor["executor"] = map[string]any{"authority": "updater"}
	assertV2SchemaFixture(t, proposal, proposalWithExecutor, false)
	wrongDetector := cloneV2Fixture(t, validProposal)
	wrongDetector["detector"].(map[string]any)["service_type"] = "worker"
	assertV2SchemaFixture(t, proposal, wrongDetector, false)
	proposalCapabilityMismatch := cloneV2Fixture(t, validProposal)
	proposalCapabilityMismatch["required_capability"] = "host.port"
	assertV2SchemaFixture(t, proposal, proposalCapabilityMismatch, false)

	localTransition := compileNormalizedOpenAPISchema(t, "observability-api.json", "/components/schemas/ObservabilityLocalRemediationTransition")
	validLocalTransition := map[string]any{
		"action_id": "action-1", "action_type": "rerun_diagnostics", "expected_revision": 2,
		"fence": 2, "idempotency_key": "idem-1", "audit_correlation_id": "audit-1",
	}
	assertV2SchemaFixture(t, localTransition, validLocalTransition, true)
	hostActionInObservability := cloneV2Fixture(t, validLocalTransition)
	hostActionInObservability["action_type"] = "host.update"
	assertV2SchemaFixture(t, localTransition, hostActionInObservability, false)

	probeCases := []struct {
		bundle      string
		serviceType string
	}{
		{bundle: "control-api.json", serviceType: "control_panel"},
		{bundle: "discord-bot-api.json", serviceType: "discord_bot"},
		{bundle: "encoder-recorder-api.json", serviceType: "encoder_recorder"},
		{bundle: "observability-api.json", serviceType: "observability"},
	}
	for _, test := range probeCases {
		t.Run("application_probe_"+test.serviceType, func(t *testing.T) {
			probe := compileNormalizedOpenAPISchema(t, test.bundle,
				"/paths/~1updater~1version/get/responses/200/content/application~1json/schema")
			validProbe := map[string]any{
				"version": "v2.0.0", "service_id": test.serviceType + "-1",
				"service_type": test.serviceType, "config_revision": 7,
			}
			assertV2SchemaFixture(t, probe, validProbe, true)
			missingRevision := cloneV2Fixture(t, validProbe)
			delete(missingRevision, "config_revision")
			assertV2SchemaFixture(t, probe, missingRevision, false)
			wrongIdentity := cloneV2Fixture(t, validProbe)
			wrongIdentity["service_type"] = "updater"
			assertV2SchemaFixture(t, probe, wrongIdentity, false)
			updaterHealthSubstitution := cloneV2Fixture(t, validProbe)
			delete(updaterHealthSubstitution, "service_id")
			updaterHealthSubstitution["updater_health"] = map[string]any{"status": "ready"}
			assertV2SchemaFixture(t, probe, updaterHealthSubstitution, false)
		})
	}
	workerProbe := compileNormalizedOpenAPISchema(t, "control-api.json",
		"/components/pathItems/WorkerApplicationRuntimeIdentityProbe/get/responses/200/content/application~1json/schema")
	validWorkerProbe := map[string]any{
		"version": "v2.0.0", "service_id": "worker-1", "service_type": "worker", "config_revision": 7,
	}
	assertV2SchemaFixture(t, workerProbe, validWorkerProbe, true)
	workerHealthSubstitution := cloneV2Fixture(t, validWorkerProbe)
	delete(workerHealthSubstitution, "service_id")
	workerHealthSubstitution["updater_health"] = map[string]any{"status": "ready"}
	assertV2SchemaFixture(t, workerProbe, workerHealthSubstitution, false)
}

func compileNormalizedOpenAPISchema(t *testing.T, bundleName, fragment string) *jsonschema.Schema {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "characterization", "generated", "openapi", "normalized", bundleName)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse normalized %s: %v", bundleName, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.UseRegexpEngine(compileECMAContractSchemaRegexp)
	if err := compiler.AddResource(bundleName, document); err != nil {
		t.Fatalf("register normalized %s: %v", bundleName, err)
	}
	compiled, err := compiler.Compile(bundleName + "#" + fragment)
	if err != nil {
		t.Fatalf("compile normalized %s#%s: %v", bundleName, fragment, err)
	}
	return compiled
}

func assertGoJSONFields(t *testing.T, typ reflect.Type, fields ...string) {
	t.Helper()
	present := make(map[string]struct{}, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		name := strings.Split(typ.Field(index).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			present[name] = struct{}{}
		}
	}
	for _, field := range fields {
		if _, ok := present[field]; !ok {
			t.Fatalf("%s omits JSON field %q", typ.Name(), field)
		}
	}
}

func assertExactGoJSONSchemaParity(t *testing.T, typ reflect.Type, schema map[string]any) {
	t.Helper()
	if schema["additionalProperties"] != false {
		t.Fatalf("%s OpenAPI schema is not strict", typ.Name())
	}
	properties := requireCharacterizationMap(t, schema, "properties")
	required := requireCharacterizationStringSet(t, schema, "required")
	goFields := make(map[string]bool, typ.NumField())
	for index := 0; index < typ.NumField(); index++ {
		tag := typ.Field(index).Tag.Get("json")
		parts := strings.Split(tag, ",")
		if parts[0] == "" || parts[0] == "-" {
			continue
		}
		omitEmpty := false
		for _, option := range parts[1:] {
			if option == "omitempty" {
				omitEmpty = true
			}
		}
		goFields[parts[0]] = omitEmpty
	}
	if len(goFields) != len(properties) {
		t.Fatalf("%s Go/OpenAPI field count differs: go=%v openapi=%v", typ.Name(), goFields, properties)
	}
	for name, omitEmpty := range goFields {
		if _, ok := properties[name]; !ok {
			t.Fatalf("%s Go field %q is absent from OpenAPI", typ.Name(), name)
		}
		_, isRequired := required[name]
		if omitEmpty == isRequired {
			t.Fatalf("%s field %q omitempty=%t OpenAPI-required=%t", typ.Name(), name, omitEmpty, isRequired)
		}
	}
	for name := range properties {
		if _, ok := goFields[name]; !ok {
			t.Fatalf("%s OpenAPI field %q is absent from Go DTO", typ.Name(), name)
		}
	}
}

func readNormalizedOpenAPICharacterization(t *testing.T, name string) map[string]any {
	t.Helper()
	root := filepath.Join("..", "..", "testdata", "characterization", "generated", "openapi", "normalized")
	var bundle map[string]any
	readCharacterizationJSON(t, filepath.Join(root, name), &bundle)
	return bundle
}

func requireCharacterizationMap(t *testing.T, parent map[string]any, field string) map[string]any {
	t.Helper()
	value, ok := parent[field].(map[string]any)
	if !ok {
		t.Fatalf("%s is %T, want object", field, parent[field])
	}
	return value
}

func resolveCharacterizationSchema(t *testing.T, root map[string]any, raw any) map[string]any {
	t.Helper()
	value, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("schema/response is %T, want object", raw)
	}
	ref, _ := value["$ref"].(string)
	if ref == "" {
		return value
	}
	if !strings.HasPrefix(ref, "#/") {
		t.Fatalf("unsupported non-local characterization ref %q", ref)
	}
	current := any(root)
	for _, rawToken := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("ref %q traversed non-object at %q", ref, token)
		}
		current, ok = object[token]
		if !ok {
			t.Fatalf("ref %q is unresolved at %q", ref, token)
		}
	}
	resolved, ok := current.(map[string]any)
	if !ok {
		t.Fatalf("ref %q resolved to %T, want object", ref, current)
	}
	return resolved
}

func assertExactCharacterizationProperties(t *testing.T, schema map[string]any, want []string) {
	t.Helper()
	if schema["additionalProperties"] != false {
		t.Fatalf("schema is not strict: %v", schema)
	}
	properties := requireCharacterizationMap(t, schema, "properties")
	got := make([]string, 0, len(properties))
	for name := range properties {
		got = append(got, name)
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema properties=%v, want exact %v", got, want)
	}
	assertCharacterizationRequired(t, schema, want...)
}

func assertCharacterizationRequired(t *testing.T, schema map[string]any, fields ...string) {
	t.Helper()
	required := characterizationStringSlice(t, schema["required"])
	set := make(map[string]struct{}, len(required))
	for _, field := range required {
		set[field] = struct{}{}
	}
	for _, field := range fields {
		if _, ok := set[field]; !ok {
			t.Fatalf("required=%v omits %q", required, field)
		}
	}
}

func requireCharacterizationStringSet(t *testing.T, parent map[string]any, field string) map[string]struct{} {
	t.Helper()
	values := characterizationStringSlice(t, parent[field])
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func characterizationStringSlice(t *testing.T, raw any) []string {
	t.Helper()
	values, ok := raw.([]any)
	if !ok {
		t.Fatalf("value is %T, want array", raw)
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("array item is %T, want string", value)
		}
		result = append(result, text)
	}
	return result
}

func verifyOpenAPISourceInventory(t *testing.T, inventory openAPISourceInventory) {
	t.Helper()

	if inventory.EntrypointCount != len(inventory.Entrypoints) || inventory.EntrypointCount != 4 {
		t.Fatalf("OpenAPI entrypoint count=%d/%d, want 4", inventory.EntrypointCount, len(inventory.Entrypoints))
	}
	actualEntries, err := os.ReadDir(filepath.Join("..", "..", "openapi"))
	if err != nil {
		t.Fatalf("read OpenAPI directory: %v", err)
	}
	var actualEntrypoints []string
	for _, entry := range actualEntries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		actualEntrypoints = append(actualEntrypoints, "openapi/"+entry.Name())
	}
	sort.Strings(actualEntrypoints)
	if !reflect.DeepEqual(actualEntrypoints, inventory.Entrypoints) {
		t.Fatalf("OpenAPI entrypoint inventory drifted: got=%v want=%v", actualEntrypoints, inventory.Entrypoints)
	}
	if inventory.SourceFileCount != len(inventory.SourceFiles) || inventory.SourceFileCount == 0 {
		t.Fatalf("OpenAPI source file count=%d/%d", inventory.SourceFileCount, len(inventory.SourceFiles))
	}
	if len(inventory.ExternalNetworkRefs) != 0 || len(inventory.MissingLocalRefs) != 0 {
		t.Fatalf("OpenAPI source inventory contains external or missing refs: external=%v missing=%v",
			inventory.ExternalNetworkRefs, inventory.MissingLocalRefs)
	}
	for _, source := range inventory.SourceFiles {
		if filepath.IsAbs(source.Path) || strings.Contains(source.Path, "..") {
			t.Fatalf("unsafe OpenAPI source path %q", source.Path)
		}
		body, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(source.Path)))
		if err != nil {
			t.Fatalf("read OpenAPI source dependency %s: %v", source.Path, err)
		}
		normalized := bytes.ReplaceAll(bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
		digest := sha256.Sum256(normalized)
		if got := hex.EncodeToString(digest[:]); got != source.NormalizedSourceSHA256 {
			t.Fatalf("OpenAPI source dependency %s drifted: got=%s want=%s; run the pinned characterization verifier",
				source.Path, got, source.NormalizedSourceSHA256)
		}
	}
}

func verifyOpenAPISemanticInventory(t *testing.T, api map[string]any) {
	t.Helper()

	name, _ := api["api"].(string)
	if name == "" {
		t.Fatal("semantic inventory API name is empty")
	}
	for _, field := range []string{
		"path_count", "operation_count", "method_counts", "schema_count", "security_scheme_count",
		"operation_id_count", "duplicate_operation_ids", "unresolved_ref_count", "response_status_distribution",
		"request_body_count", "explicit_security_count", "inherited_security_count", "undefined_security_count",
		"public_operations", "content_type_inventory", "duplicate_method_paths", "operations_without_security_definition",
		"operations_without_responses", "operations_without_success_response", "operations_without_4xx_response",
		"inline_schema_count", "operations", "normalized_bundle_sha256", "ref_layout_independent_sha256",
	} {
		if _, ok := api[field]; !ok {
			t.Fatalf("%s semantic inventory is missing %s", name, field)
		}
	}
	if countAsInt(t, api, "unresolved_ref_count") != 0 {
		t.Fatalf("%s has unresolved bundled refs: %v", name, api["unresolved_refs"])
	}
	for _, field := range []string{
		"duplicate_operation_ids", "duplicate_method_paths", "operations_without_security_definition",
		"operations_without_responses",
	} {
		if values, ok := api[field].([]any); !ok || len(values) != 0 {
			t.Fatalf("%s %s=%v, want empty", name, field, api[field])
		}
	}
	operations, ok := api["operations"].([]any)
	if !ok || len(operations) != countAsInt(t, api, "operation_count") {
		t.Fatalf("%s operation inventory count does not match summary", name)
	}
	seen := make(map[string]struct{}, len(operations))
	for _, raw := range operations {
		operation, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("%s has invalid operation record %T", name, raw)
		}
		for _, field := range []string{
			"method", "path", "operation_id", "tags", "security_source", "effective_security",
			"exposure_classification", "request_body_content_types", "request_schemas", "responses",
			"deprecated", "summary_present",
		} {
			if _, ok := operation[field]; !ok {
				t.Fatalf("%s operation is missing %s: %v", name, field, operation)
			}
		}
		identity := fmt.Sprintf("%s %s", operation["method"], operation["path"])
		if _, exists := seen[identity]; exists {
			t.Fatalf("%s duplicates method/path %s", name, identity)
		}
		seen[identity] = struct{}{}
	}
}

func verifyControlAPISemanticProjection(t *testing.T, root string, APIs []map[string]any) {
	t.Helper()

	var control map[string]any
	readCharacterizationJSON(t, filepath.Join(root, "openapi", "control-api-semantics.json"), &control)
	for _, api := range APIs {
		if api["api"] == "openapi/control-api.yaml" {
			if !reflect.DeepEqual(api, control) {
				t.Fatal("control-api-semantics.json differs from the all-API semantic inventory")
			}
			return
		}
	}
	t.Fatal("semantic inventory has no control-api entry")
}

func verifyBundledJobGenerationContract(t *testing.T, root string) {
	t.Helper()

	var bundle map[string]any
	readCharacterizationJSON(t, filepath.Join(root, "openapi", "normalized", "discord-bot-api.json"), &bundle)
	components, _ := bundle["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	for name, raw := range schemas {
		schema, _ := raw.(map[string]any)
		properties, _ := schema["properties"].(map[string]any)
		jobGeneration, ok := properties["job_generation"].(map[string]any)
		if !ok {
			continue
		}
		required, _ := schema["required"].([]any)
		if !containsJSONText(required, "job_generation") || jobGeneration["type"] != "integer" ||
			fmt.Sprint(jobGeneration["minimum"]) != "1" {
			t.Fatalf("bundled %s job_generation lost required integer minimum 1: %v", name, jobGeneration)
		}
		if _, exists := jobGeneration["maximum"]; exists {
			t.Fatalf("bundled %s job_generation gained an unsafe maximum", name)
		}
		return
	}
	t.Fatal("Discord Bot bundle has no schema containing job_generation")
}

func verifyOpenAPILintBaseline(t *testing.T, baseline openAPILintBaseline) {
	t.Helper()

	if baseline.RedoclyVersion != pinnedRedoclyVersion || len(baseline.APIs) != 4 {
		t.Fatalf("invalid lint baseline header: version=%q APIs=%d", baseline.RedoclyVersion, len(baseline.APIs))
	}
	windowsAbsolute := regexp.MustCompile(`(?i)^[a-z]:[\\/]`)
	for _, api := range baseline.APIs {
		if api.FindingCount != len(api.Findings) {
			t.Fatalf("%s lint finding count=%d/%d", api.API, api.FindingCount, len(api.Findings))
		}
		errors := 0
		warnings := 0
		for _, finding := range api.Findings {
			if _, containsMessage := finding["message"]; containsMessage {
				t.Fatalf("%s lint baseline retains volatile message text", api.API)
			}
			severity, _ := finding["severity"].(string)
			switch severity {
			case "error":
				errors++
			case "warn", "warning":
				warnings++
			}
			fingerprint, _ := finding["fingerprint"].(string)
			delete(finding, "fingerprint")
			canonical := canonicalCharacterizationJSON(t, finding)
			digest := sha256.Sum256(canonical)
			finding["fingerprint"] = fingerprint
			if got := hex.EncodeToString(digest[:]); got != fingerprint {
				t.Fatalf("%s lint finding fingerprint=%s, want %s", api.API, fingerprint, got)
			}
			locations, _ := finding["locations"].([]any)
			for _, rawLocation := range locations {
				location, _ := rawLocation.(map[string]any)
				source, _ := location["source"].(string)
				if filepath.IsAbs(source) || windowsAbsolute.MatchString(source) {
					t.Fatalf("%s lint baseline contains absolute source path %q", api.API, source)
				}
			}
		}
		if errors != api.ErrorCount || warnings != api.WarningCount {
			t.Fatalf("%s lint counts drifted: errors=%d/%d warnings=%d/%d",
				api.API, errors, api.ErrorCount, warnings, api.WarningCount)
		}
	}
}

func readCharacterizationJSON(t *testing.T, path string, destination any) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func canonicalCharacterizationJSON(t *testing.T, value any) []byte {
	t.Helper()

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		t.Fatalf("encode canonical characterization JSON: %v", err)
	}
	return normalizeJavaScriptJSONLineSeparators(bytes.TrimSuffix(buffer.Bytes(), []byte("\n")))
}

func TestCanonicalCharacterizationJSONMatchesJavaScriptLineSeparatorEncoding(t *testing.T) {
	got := string(canonicalCharacterizationJSON(t, map[string]any{
		"actual":  "\u2028\u2029",
		"literal": `\u2028`,
	}))
	want := "{\"actual\":\"\u2028\u2029\",\"literal\":\"\\\\u2028\"}"
	if got != want {
		t.Fatalf("canonical JSON = %q, want %q", got, want)
	}
}

// JavaScript JSON.stringify emits U+2028 and U+2029 literally, while Go's
// encoding/json escapes them for JSONP safety even when HTML escaping is off.
// The pinned characterization generator is JavaScript, so normalize only
// unescaped instances without corrupting a literal "\\u2028" string.
func normalizeJavaScriptJSONLineSeparators(encoded []byte) []byte {
	normalized := make([]byte, 0, len(encoded))
	for index := 0; index < len(encoded); {
		if index+1 < len(encoded) && encoded[index] == '\\' && encoded[index+1] == '\\' {
			normalized = append(normalized, encoded[index], encoded[index+1])
			index += 2
			continue
		}
		if index+6 <= len(encoded) && bytes.Equal(encoded[index:index+6], []byte(`\u2028`)) {
			normalized = append(normalized, []byte("\u2028")...)
			index += 6
			continue
		}
		if index+6 <= len(encoded) && bytes.Equal(encoded[index:index+6], []byte(`\u2029`)) {
			normalized = append(normalized, []byte("\u2029")...)
			index += 6
			continue
		}
		normalized = append(normalized, encoded[index])
		index++
	}
	return normalized
}

func countAsInt(t *testing.T, object map[string]any, field string) int {
	t.Helper()

	number, ok := object[field].(json.Number)
	if !ok {
		t.Fatalf("%s is %T, want JSON number", field, object[field])
	}
	value, err := number.Int64()
	if err != nil {
		t.Fatalf("parse %s=%q: %v", field, number, err)
	}
	return int(value)
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func containsJSONText(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
