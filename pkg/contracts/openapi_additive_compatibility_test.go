package contracts

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const controlOpenAPITestMutantEnvironment = "AUTOSTREAM_CONTRACT_TEST_MUTANT"

func TestControlOpenAPIAdditiveV2CompatibilitySemantics(t *testing.T) {
	for _, marker := range []string{
		"new_advertised_port:",
		"new_published_port:",
		"new_container_port:",
		"expected_endpoint_revision:",
		"eligible_operations:",
		"operation_blocked_reasons:",
		"enum: [primary, standby]",
		"enum: [disabled, totp, passkey]",
		"enum: [discord, slack, generic, email]",
		"required: [rtmp_url]",
		"required: [stream_key_secret_name]",
		"enum: [ssh_v1, pull_v2]",
	} {
		if !requireControlOpenAPISemanticMarker(t, marker) {
			t.Fatalf("semantic marker %q has no structural assertion", marker)
		}
	}

	for _, name := range []string{
		"SystemUpdatePullOwnershipActivateRequest",
		"SystemUpdatePullOwnershipActivateResponse",
		"SystemUpdatePullOwnershipDeactivateRequest",
		"SystemUpdatePullOwnershipDeactivateResponse",
	} {
		requireControlOpenAPIPullOwnershipComponent(t, name)
	}

	activate := compileNormalizedOpenAPISchema(t, "control-api.json",
		normalizedOpenAPIRequestSchemaFragment(
			"/system-updates/updaters/{id}/pull-ownership/activate", "post"))
	activateValid := pullOwnershipActivationRequestFixture()
	validatePullOwnershipActivationJSON(t, activate, activateValid, true)
	activateUnknown := clonePullOwnershipFixture(t, activateValid)
	activateUnknown["unknown_property"] = true
	validatePullOwnershipActivationJSON(t, activate, activateUnknown, false)

	deactivate := compileNormalizedOpenAPISchema(t, "control-api.json",
		normalizedOpenAPIRequestSchemaFragment(
			"/system-updates/updaters/{id}/pull-ownership/deactivate", "post"))
	deactivateValid := pullOwnershipDeactivationRequestFixture()
	validatePullOwnershipDeactivationJSON(t, deactivate, deactivateValid, true)
	deactivateUnknown := clonePullOwnershipFixture(t, deactivateValid)
	deactivateUnknown["unknown_property"] = true
	validatePullOwnershipDeactivationJSON(t, deactivate, deactivateUnknown, false)
}

func TestControlOpenAPIAdditiveV2CompatibilityMutationSensitivity(t *testing.T) {
	for _, test := range []struct {
		name        string
		wantFailure string
	}{
		{name: "new_advertised_port_removed", wantFailure: "new_advertised_port"},
		{name: "eligible_operations_removed", wantFailure: "eligible_operations"},
		{name: "activation_unknown_allowed", wantFailure: "SystemUpdatePullOwnershipActivateRequest is not a strict object"},
		{name: "deactivation_unknown_allowed", wantFailure: "SystemUpdatePullOwnershipDeactivateRequest is not a strict object"},
		{name: "standby_removed", wantFailure: "enum=[primary]"},
		{name: "passkey_removed", wantFailure: "enum=[disabled totp]"},
		{name: "email_removed", wantFailure: "enum=[discord slack generic]"},
		{name: "relay_rtmp_exclusion_removed", wantFailure: "rtmp_url"},
		{name: "ssh_v1_removed", wantFailure: "enum=[pull_v2]"},
		{name: "pull_v2_removed", wantFailure: "enum=[ssh_v1]"},
		{name: "marker_only_unreferenced_schema", wantFailure: "POST /system-updates request does not reference SystemUpdateCreateRequest"},
		{name: "email_marker_only_unreferenced_schema", wantFailure: "enum=[discord slack generic]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(executable,
				"-test.run=^TestControlOpenAPIAdditiveV2CompatibilitySemantics$",
				"-test.count=1",
			)
			command.Env = append(os.Environ(), controlOpenAPITestMutantEnvironment+"="+test.name)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("mutant %s was not rejected:\n%s", test.name, output)
			}
			if !strings.Contains(string(output), test.wantFailure) {
				t.Fatalf("mutant %s failed for the wrong reason; wanted %q:\n%s",
					test.name, test.wantFailure, output)
			}
		})
	}
}

func requireControlOpenAPISemanticMarker(t *testing.T, marker string) bool {
	t.Helper()
	switch marker {
	case "new_advertised_port:", "new_published_port:", "new_container_port:", "expected_endpoint_revision:":
		requireControlOpenAPISystemUpdateCreateRequest(t)
	case "eligible_operations:", "operation_blocked_reasons:":
		requireControlOpenAPISystemUpdateTargetProjection(t)
	case "enum: [primary, standby]":
		requireControlOpenAPIServiceAssignmentRole(t)
	case "enum: [disabled, totp, passkey]":
		requireControlOpenAPIMFAPolicy(t)
	case "enum: [discord, slack, generic, email]":
		requireControlOpenAPINotificationEmail(t)
	case "required: [rtmp_url]", "required: [stream_key_secret_name]":
		requireControlOpenAPIRelayStaticExclusions(t)
	case "enum: [ssh_v1, pull_v2]":
		requireControlOpenAPITransportModes(t)
	default:
		return false
	}
	return true
}

func requireControlOpenAPISystemUpdateCreateRequest(t *testing.T) map[string]any {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	schema := requireControlOpenAPIRequestComponentSchema(
		t, control, "/system-updates", "post", "SystemUpdateCreateRequest")
	requireControlOpenAPIIntegerProperty(t, schema, "SystemUpdateCreateRequest", "new_advertised_port", 1, 65535)
	requireControlOpenAPIIntegerProperty(t, schema, "SystemUpdateCreateRequest", "new_published_port", 1024, 65535)
	requireControlOpenAPIIntegerProperty(t, schema, "SystemUpdateCreateRequest", "new_container_port", 1024, 65535)
	property := requireControlOpenAPIProperty(t, schema, "SystemUpdateCreateRequest", "expected_endpoint_revision")
	if property["type"] != "integer" || fmt.Sprint(property["minimum"]) != "1" {
		t.Fatalf("SystemUpdateCreateRequest.expected_endpoint_revision=%v", property)
	}
	return schema
}

func requireControlOpenAPISystemUpdateTargetProjection(t *testing.T) map[string]any {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	response := requireControlOpenAPIResponseComponentSchema(
		t, control, "/system-updates", "get", "200", "SystemUpdatesResponse")
	properties := requireCharacterizationMap(t, response, "properties")
	targets := requireCharacterizationMap(t, properties, "targets")
	target := requireControlOpenAPIComponentReference(
		t, control, targets["items"], "SystemUpdateTarget", "GET /system-updates response targets[]")

	eligible := requireControlOpenAPIProperty(t, target, "SystemUpdateTarget", "eligible_operations")
	if eligible["type"] != "array" || eligible["uniqueItems"] != true {
		t.Fatalf("SystemUpdateTarget.eligible_operations=%v", eligible)
	}
	assertControlOpenAPIExactEnum(t, requireCharacterizationMap(t, eligible, "items"),
		[]string{"software_update", "port_reconfigure"})

	reasons := requireControlOpenAPIProperty(t, target, "SystemUpdateTarget", "operation_blocked_reasons")
	if reasons["type"] != "object" || reasons["additionalProperties"] != false {
		t.Fatalf("SystemUpdateTarget.operation_blocked_reasons=%v", reasons)
	}
	assertExactControlOpenAPIKeys(t, requireCharacterizationMap(t, reasons, "properties"),
		[]string{"software_update", "port_reconfigure"})
	return target
}

func requireControlOpenAPIServiceAssignmentRole(t *testing.T) {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	schema := requireControlOpenAPIRequestComponentSchema(
		t, control, "/services/{id}/assign", "post", "ServiceAssignmentWriteRequest")
	assertControlOpenAPIExactEnum(t,
		requireControlOpenAPIProperty(t, schema, "ServiceAssignmentWriteRequest", "assignment_role"),
		[]string{"primary", "standby"})
}

func requireControlOpenAPIMFAPolicy(t *testing.T) {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	request := requireControlOpenAPIRequestComponentSchema(
		t, control, "/security/settings", "put", "SecuritySettings")
	response := requireControlOpenAPIResponseComponentSchema(
		t, control, "/security/settings", "get", "200", "SecuritySettings")
	for label, schema := range map[string]map[string]any{"request": request, "response": response} {
		assertControlOpenAPIExactEnum(t,
			requireControlOpenAPIProperty(t, schema, "SecuritySettings "+label, "mfa_mode"),
			[]string{"disabled", "totp", "passkey"})
	}
}

func requireControlOpenAPINotificationEmail(t *testing.T) {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	request := requireControlOpenAPIRequestComponentSchema(
		t, control, "/observability/notification-channels/{id}", "put",
		"ControlNotificationChannelUpdateRequest")
	response := requireControlOpenAPIResponseComponentSchema(
		t, control, "/observability/notification-channels/{id}", "put", "200", "NotificationChannel")
	for label, schema := range map[string]map[string]any{
		"ControlNotificationChannelUpdateRequest": request,
		"NotificationChannel":                     response,
	} {
		assertControlOpenAPIExactEnum(t,
			requireControlOpenAPIProperty(t, schema, label, "type"),
			[]string{"discord", "slack", "generic", "email"})
	}
}

func requireControlOpenAPIRelayStaticExclusions(t *testing.T) {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	runtime := requireControlOpenAPIResponseComponentSchema(
		t, control, "/services/runtime-config", "get", "200", "ServiceRuntimeConfig")
	notSchema := findRelayStaticExclusion(t, control, runtime)
	requireRelayStaticExclusionRequirement(t, notSchema, "rtmp_url")
	requireRelayStaticExclusionRequirement(t, notSchema, "stream_key_secret_name")
}

func requireControlOpenAPITransportModes(t *testing.T) {
	t.Helper()
	control := readMutatedControlOpenAPI(t)
	schema := requireControlOpenAPIRequestComponentSchema(
		t, control, "/services/register", "post", "ServiceRegistrationRequest")
	assertControlOpenAPIExactEnum(t,
		requireControlOpenAPIProperty(t, schema, "ServiceRegistrationRequest", "transport_mode"),
		[]string{"ssh_v1", "pull_v2"})
}

func readMutatedControlOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	applyControlOpenAPITestMutant(t, control, os.Getenv(controlOpenAPITestMutantEnvironment))
	return control
}

func controlOpenAPIComponent(t *testing.T, control map[string]any, name string) (map[string]any, map[string]any) {
	t.Helper()
	components := requireCharacterizationMap(t, control, "components")
	schemas := requireCharacterizationMap(t, components, "schemas")
	raw, ok := schemas[name].(map[string]any)
	if !ok {
		t.Fatalf("control OpenAPI component %s is %T, want schema object", name, schemas[name])
	}
	return raw, resolveCharacterizationSchema(t, control, raw)
}

func controlOpenAPIPathOperation(t *testing.T, control map[string]any, path string, method string) map[string]any {
	t.Helper()
	paths := requireCharacterizationMap(t, control, "paths")
	pathItem := resolveCharacterizationSchema(t, control, paths[path])
	operation, ok := pathItem[method]
	if !ok {
		t.Fatalf("control OpenAPI path %s omits %s", path, strings.ToUpper(method))
	}
	return resolveCharacterizationSchema(t, control, operation)
}

func controlOpenAPIRequestSchemaSlot(t *testing.T, control map[string]any, path string, method string) map[string]any {
	t.Helper()
	operation := controlOpenAPIPathOperation(t, control, path, method)
	requestBody := resolveCharacterizationSchema(t, control, operation["requestBody"])
	content := requireCharacterizationMap(t, requestBody, "content")
	return requireCharacterizationMap(t, content, "application/json")
}

func controlOpenAPIResponseSchemaSlot(t *testing.T, control map[string]any, path string, method string, status string) map[string]any {
	t.Helper()
	operation := controlOpenAPIPathOperation(t, control, path, method)
	responses := requireCharacterizationMap(t, operation, "responses")
	response := resolveCharacterizationSchema(t, control, responses[status])
	content := requireCharacterizationMap(t, response, "content")
	return requireCharacterizationMap(t, content, "application/json")
}

func requireControlOpenAPIRequestComponentSchema(t *testing.T, control map[string]any, path string, method string, componentName string) map[string]any {
	t.Helper()
	slot := controlOpenAPIRequestSchemaSlot(t, control, path, method)
	label := fmt.Sprintf("%s %s request", strings.ToUpper(method), path)
	return requireControlOpenAPIComponentReference(t, control, slot["schema"], componentName, label)
}

func requireControlOpenAPIResponseComponentSchema(t *testing.T, control map[string]any, path string, method string, status string, componentName string) map[string]any {
	t.Helper()
	slot := controlOpenAPIResponseSchemaSlot(t, control, path, method, status)
	label := fmt.Sprintf("%s %s response %s", strings.ToUpper(method), path, status)
	return requireControlOpenAPIComponentReference(t, control, slot["schema"], componentName, label)
}

func requireControlOpenAPIComponentReference(t *testing.T, control map[string]any, raw any, componentName string, label string) map[string]any {
	t.Helper()
	rawSchema, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("%s schema is %T, want object", label, raw)
	}
	gotRef, _ := rawSchema["$ref"].(string)
	componentRaw, component := controlOpenAPIComponent(t, control, componentName)
	allowedRefs := map[string]struct{}{"#/components/schemas/" + componentName: {}}
	if alias, _ := componentRaw["$ref"].(string); alias != "" {
		allowedRefs[alias] = struct{}{}
	}
	if _, ok := allowedRefs[gotRef]; !ok {
		t.Fatalf("%s does not reference %s: ref=%q", label, componentName, gotRef)
	}
	resolved := resolveCharacterizationSchema(t, control, rawSchema)
	if !reflect.DeepEqual(resolved, component) {
		t.Fatalf("%s does not resolve to canonical %s", label, componentName)
	}
	return resolved
}

func applyControlOpenAPITestMutant(t *testing.T, control map[string]any, mutant string) {
	t.Helper()
	if mutant == "" {
		return
	}
	component := func(name string) map[string]any {
		_, resolved := controlOpenAPIComponent(t, control, name)
		return resolved
	}
	property := func(componentName string) map[string]any {
		return requireCharacterizationMap(t, component(componentName), "properties")
	}
	setEnum := func(componentName, propertyName string, values ...string) {
		properties := property(componentName)
		schema, ok := properties[propertyName].(map[string]any)
		if !ok {
			t.Fatalf("mutant target property %s.%s is missing", componentName, propertyName)
		}
		enum := make([]any, len(values))
		for index, value := range values {
			enum[index] = value
		}
		schema["enum"] = enum
	}

	switch mutant {
	case "new_advertised_port_removed":
		delete(property("SystemUpdateCreateRequest"), "new_advertised_port")
	case "eligible_operations_removed":
		delete(property("SystemUpdateTarget"), "eligible_operations")
	case "activation_unknown_allowed":
		component("SystemUpdatePullOwnershipActivateRequest")["additionalProperties"] = true
	case "deactivation_unknown_allowed":
		component("SystemUpdatePullOwnershipDeactivateRequest")["additionalProperties"] = true
	case "standby_removed":
		setEnum("ServiceAssignmentWriteRequest", "assignment_role", "primary")
	case "passkey_removed":
		setEnum("SecuritySettings", "mfa_mode", "disabled", "totp")
	case "email_removed":
		setEnum("ControlNotificationChannelUpdateRequest", "type", "discord", "slack", "generic")
	case "relay_rtmp_exclusion_removed":
		runtime := requireControlOpenAPIResponseComponentSchema(
			t, control, "/services/runtime-config", "get", "200", "ServiceRuntimeConfig")
		notSchema := findRelayStaticExclusion(t, control, runtime)
		alternatives := notSchema["anyOf"].([]any)
		kept := make([]any, 0, len(alternatives))
		for _, rawAlternative := range alternatives {
			alternative, _ := rawAlternative.(map[string]any)
			required := characterizationStringSlice(t, alternative["required"])
			if reflect.DeepEqual(required, []string{"rtmp_url"}) {
				continue
			}
			kept = append(kept, rawAlternative)
		}
		notSchema["anyOf"] = kept
	case "ssh_v1_removed":
		setEnum("ServiceRegistrationRequest", "transport_mode", "pull_v2")
	case "pull_v2_removed":
		setEnum("ServiceRegistrationRequest", "transport_mode", "ssh_v1")
	case "marker_only_unreferenced_schema":
		slot := controlOpenAPIRequestSchemaSlot(t, control, "/system-updates", "post")
		slot["schema"] = map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]any{"operation": map[string]any{"type": "string"}},
		}
		components := requireCharacterizationMap(t, control, "components")
		schemas := requireCharacterizationMap(t, components, "schemas")
		schemas["UnreferencedMarkerOnly"] = map[string]any{
			"description": "new_advertised_port: marker-only text",
			"type":        "object",
			"properties": map[string]any{
				"new_advertised_port": map[string]any{"type": "integer"},
			},
		}
	case "email_marker_only_unreferenced_schema":
		setEnum("ControlNotificationChannelUpdateRequest", "type", "discord", "slack", "generic")
		requestType := requireCharacterizationMap(t, property("ControlNotificationChannelUpdateRequest"), "type")
		requestType["description"] = "enum: [discord, slack, generic, email] marker-only text"
		components := requireCharacterizationMap(t, control, "components")
		schemas := requireCharacterizationMap(t, components, "schemas")
		schemas["UnreferencedNotificationEmailMarker"] = map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{"enum": []any{"discord", "slack", "generic", "email"}},
			},
		}
	default:
		t.Fatalf("unknown control OpenAPI test mutant %q", mutant)
	}
}

func requireControlOpenAPIProperty(t *testing.T, schema map[string]any, schemaName string, propertyName string) map[string]any {
	t.Helper()
	properties := requireCharacterizationMap(t, schema, "properties")
	property, ok := properties[propertyName].(map[string]any)
	if !ok {
		t.Fatalf("%s.%s is %T, want schema object", schemaName, propertyName, properties[propertyName])
	}
	return property
}

func requireControlOpenAPIIntegerProperty(t *testing.T, schema map[string]any, schemaName string, propertyName string, minimum float64, maximum float64) {
	t.Helper()
	property := requireControlOpenAPIProperty(t, schema, schemaName, propertyName)
	if property["type"] != "integer" || fmt.Sprint(property["minimum"]) != fmt.Sprint(minimum) || fmt.Sprint(property["maximum"]) != fmt.Sprint(maximum) {
		t.Fatalf("%s.%s=%v", schemaName, propertyName, property)
	}
}

func assertControlOpenAPIExactEnum(t *testing.T, schema map[string]any, want []string) {
	t.Helper()
	got := characterizationStringSlice(t, schema["enum"])
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enum=%v, want %v", got, want)
	}
}

func assertExactControlOpenAPIKeys(t *testing.T, values map[string]any, want []string) {
	t.Helper()
	got := make([]string, 0, len(values))
	for name := range values {
		got = append(got, name)
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys=%v, want %v", got, want)
	}
}

func findRelayStaticExclusion(t *testing.T, control map[string]any, schema map[string]any) map[string]any {
	t.Helper()
	seenRefs := make(map[string]bool)
	var walk func(any) map[string]any
	walk = func(value any) map[string]any {
		switch typed := value.(type) {
		case map[string]any:
			if ref, _ := typed["$ref"].(string); ref != "" {
				if !seenRefs[ref] {
					seenRefs[ref] = true
					if found := walk(resolveCharacterizationSchema(t, control, typed)); found != nil {
						return found
					}
				}
			}
			if condition, ok := typed["if"].(map[string]any); ok {
				properties, _ := condition["properties"].(map[string]any)
				mode, _ := properties["mode"].(map[string]any)
				thenSchema, _ := typed["then"].(map[string]any)
				notSchema, _ := thenSchema["not"].(map[string]any)
				if mode["const"] == "live_api_relay_static" && notSchema["anyOf"] != nil {
					return notSchema
				}
			}
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if key == "$ref" {
					continue
				}
				if found := walk(typed[key]); found != nil {
					return found
				}
			}
		case []any:
			for _, child := range typed {
				if found := walk(child); found != nil {
					return found
				}
			}
		}
		return nil
	}
	found := walk(schema)
	if found == nil {
		t.Fatal("GET /services/runtime-config omits the reachable live_api_relay_static exclusion")
	}
	return found
}

func requireRelayStaticExclusionRequirement(t *testing.T, notSchema map[string]any, field string) {
	t.Helper()
	alternatives, ok := notSchema["anyOf"].([]any)
	if !ok {
		t.Fatalf("live_api_relay_static exclusion anyOf is %T", notSchema["anyOf"])
	}
	for _, rawAlternative := range alternatives {
		alternative, ok := rawAlternative.(map[string]any)
		if !ok {
			continue
		}
		required := characterizationStringSlice(t, alternative["required"])
		if reflect.DeepEqual(required, []string{field}) {
			return
		}
	}
	t.Fatalf("reachable live_api_relay_static exclusion omits required marker %q", field)
}

func requireControlOpenAPIPullOwnershipComponent(t *testing.T, name string) map[string]any {
	t.Helper()
	type specification struct {
		path       string
		status     string
		properties []string
		required   []string
	}
	requestProperties := []string{
		"protocol_version", "idempotency_key", "desired_revision", "fence", "required_capability",
		"expected_execution_host_id", "expected_ownership_epoch", "expected_source_policy_revision",
		"expected_projection_revision", "expected_local_executor_policy_revision",
		"expected_local_executor_policy_sha256",
	}
	requestRequired := []string{
		"expected_execution_host_id", "expected_ownership_epoch", "expected_source_policy_revision",
		"expected_projection_revision", "expected_local_executor_policy_revision",
		"expected_local_executor_policy_sha256",
	}
	specifications := map[string]specification{
		"SystemUpdatePullOwnershipActivateRequest": {
			path: "/system-updates/updaters/{id}/pull-ownership/activate", properties: requestProperties, required: requestRequired,
		},
		"SystemUpdatePullOwnershipActivateResponse": {
			path: "/system-updates/updaters/{id}/pull-ownership/activate", status: "200",
			properties: []string{
				"protocol_version", "idempotency_key", "desired_revision", "applied_revision", "fence", "capability",
				"updater_id", "execution_host_id", "transport_mode", "agent_service_id", "ownership_epoch",
				"source_policy_revision", "projection_revision", "local_executor_policy_revision", "local_executor_policy_sha256",
			},
			required: []string{
				"updater_id", "execution_host_id", "transport_mode", "agent_service_id", "ownership_epoch",
				"source_policy_revision", "projection_revision", "local_executor_policy_revision", "local_executor_policy_sha256",
			},
		},
		"SystemUpdatePullOwnershipDeactivateRequest": {
			path: "/system-updates/updaters/{id}/pull-ownership/deactivate", properties: requestProperties, required: requestRequired,
		},
		"SystemUpdatePullOwnershipDeactivateResponse": {
			path: "/system-updates/updaters/{id}/pull-ownership/deactivate", status: "200",
			properties: []string{
				"protocol_version", "idempotency_key", "desired_revision", "applied_revision", "fence", "capability",
				"rollback_transition", "updater_id", "execution_host_id", "transport_mode", "agent_service_id",
				"ownership_epoch", "agent_ownership_epoch", "source_policy_revision", "projection_revision",
				"local_executor_policy_revision", "local_executor_policy_sha256",
			},
			required: []string{
				"updater_id", "execution_host_id", "transport_mode", "agent_service_id", "ownership_epoch",
				"agent_ownership_epoch", "source_policy_revision", "projection_revision",
				"local_executor_policy_revision", "local_executor_policy_sha256",
			},
		},
	}
	spec, ok := specifications[name]
	if !ok {
		t.Fatalf("unknown pull-ownership component %q", name)
	}
	control := readMutatedControlOpenAPI(t)
	var schema map[string]any
	if spec.status == "" {
		schema = requireControlOpenAPIRequestComponentSchema(t, control, spec.path, "post", name)
	} else {
		schema = requireControlOpenAPIResponseComponentSchema(t, control, spec.path, "post", spec.status, name)
	}
	strict := requireControlOpenAPIStrictSchema(t, name, schema, spec.properties, spec.required)
	if name == "SystemUpdatePullOwnershipActivateRequest" {
		canonical := compileContractJSONSchema(t, "system-update-pull-ownership-activate-request.schema.json")
		overflow := pullOwnershipActivationRequestFixture()
		overflow["expected_ownership_epoch"] = int64(math.MaxInt64)
		validatePullOwnershipActivationJSON(t, canonical, overflow, false)
	}
	if name == "SystemUpdatePullOwnershipDeactivateRequest" {
		canonical := compileContractJSONSchema(t, "system-update-pull-ownership-deactivate-request.schema.json")
		zeroEpoch := pullOwnershipDeactivationRequestFixture()
		zeroEpoch["expected_ownership_epoch"] = int64(0)
		validatePullOwnershipDeactivationJSON(t, canonical, zeroEpoch, false)
	}
	return strict
}

func requireControlOpenAPIStrictSchema(t *testing.T, name string, schema map[string]any, wantProperties []string, wantRequired []string) map[string]any {
	t.Helper()
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Fatalf("%s is not a strict object: %v", name, schema)
	}
	assertExactControlOpenAPIKeys(t, requireCharacterizationMap(t, schema, "properties"), wantProperties)
	required := characterizationStringSlice(t, schema["required"])
	sort.Strings(required)
	wantRequired = append([]string(nil), wantRequired...)
	sort.Strings(wantRequired)
	if !reflect.DeepEqual(required, wantRequired) {
		t.Fatalf("%s required=%v, want %v", name, required, wantRequired)
	}
	requireNoReopenedAdditionalProperties(t, name, schema)
	return schema
}

func requireNoReopenedAdditionalProperties(t *testing.T, path string, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		if typed["additionalProperties"] == true {
			t.Fatalf("%s reopens additionalProperties", path)
		}
		for key, child := range typed {
			requireNoReopenedAdditionalProperties(t, path+"."+key, child)
		}
	case []any:
		for _, child := range typed {
			requireNoReopenedAdditionalProperties(t, path+"[]", child)
		}
	}
}

func normalizedOpenAPIRequestSchemaFragment(path string, method string) string {
	pathToken := strings.ReplaceAll(strings.ReplaceAll(path, "~", "~0"), "/", "~1")
	return "/paths/" + pathToken + "/" + method + "/requestBody/content/application~1json/schema"
}
