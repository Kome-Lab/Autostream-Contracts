package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNodeActionPermissionProjectionSchemaIsStrictAndSecretNegative(t *testing.T) {
	schema := compileContractJSONSchema(t, "node-action-permission-projection.schema.json")
	valid := map[string]any{
		"contract_version":     1,
		"projection_revision": "projection-7",
		"evaluated_at":        "2026-09-02T00:00:00Z",
		"action":              "configure_token_regenerate",
		"availability":        "denied",
		"reason_code":         "additional_permission_required",
		"required_permissions": []any{
			"api_tokens.create",
			"api_tokens.revoke",
		},
		"missing_permissions": []any{"api_tokens.revoke"},
	}
	if err := schema.Validate(valid); err != nil {
		t.Fatalf("valid node action projection rejected: %v", err)
	}

	mutants := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "unknown property", mutate: func(value map[string]any) { value["future"] = true }},
		{name: "unknown availability", mutate: func(value map[string]any) { value["availability"] = "future" }},
		{name: "unknown action", mutate: func(value map[string]any) { value["action"] = "future" }},
		{name: "unknown reason", mutate: func(value map[string]any) { value["reason_code"] = "future" }},
		{name: "unknown permission", mutate: func(value map[string]any) { value["required_permissions"] = []any{"api_tokens.create", "future.permission"} }},
		{name: "raw token", mutate: func(value map[string]any) { value["runtime_token"] = "must-not-appear" }},
	}
	for _, mutant := range mutants {
		t.Run(mutant.name, func(t *testing.T) {
			candidate := cloneNodeProjectionPayload(t, valid)
			mutant.mutate(candidate)
			if err := schema.Validate(candidate); err == nil {
				t.Fatalf("schema accepted %s mutant: %#v", mutant.name, candidate)
			}
		})
	}
}

func TestNodeActionPermissionProjectionOpenAPIIsRouteStrict(t *testing.T) {
	bundle := readNormalizedOpenAPICharacterization(t, "control-api.json")
	paths := requireCharacterizationMap(t, bundle, "paths")
	pathItem := requireCharacterizationMap(t, paths, "/nodes/action-permissions")
	operation := requireCharacterizationMap(t, pathItem, "get")
	if operation["operationId"] != "getNodeActionPermissionProjection" {
		t.Fatalf("operationId=%v", operation["operationId"])
	}
	if operation["x-autostream-required-permission"] != "api_tokens.create" {
		t.Fatalf("base permission=%v, want api_tokens.create", operation["x-autostream-required-permission"])
	}

	parameters, ok := operation["parameters"].([]any)
	if !ok {
		t.Fatalf("node projection parameters=%T", operation["parameters"])
	}
	wantParameters := map[string]bool{
		"action":                false,
		"node_type":             false,
		"node_id":               false,
		"allow_runtime_secrets": false,
		"allow_remediation":     false,
	}
	for _, raw := range parameters {
		parameter := resolveCharacterizationSchema(t, bundle, raw)
		if name, ok := parameter["name"].(string); ok {
			if _, wanted := wantParameters[name]; wanted {
				wantParameters[name] = true
			}
		}
	}
	for name, found := range wantParameters {
		if !found {
			t.Fatalf("node projection is missing query parameter %q", name)
		}
	}

	responses := requireCharacterizationMap(t, operation, "responses")
	wantCodes := map[string][]string{
		"400": {"invalid_node_action_projection_request"},
		"401": {"unauthorized"},
		"403": {"password_change_required", "permission_denied"},
		"404": {"node_not_found", "runtime_token_not_found"},
		"500": {"get_node_failed", "list_service_tokens_failed", "check_system_update_emergency_recovery_failed"},
		"503": {"system_update_identity_fence_unavailable"},
	}
	for _, status := range []string{"200", "400", "401", "403", "404", "500", "503"} {
		response := resolveCharacterizationSchema(t, bundle, responses[status])
		assertNodeProjectionPrivacyHeaders(t, bundle, status, response)
		content := requireCharacterizationMap(t, response, "content")
		jsonContent := requireCharacterizationMap(t, content, "application/json")
		responseSchema := resolveCharacterizationSchema(t, bundle, jsonContent["schema"])
		if status == "200" {
			if responseSchema["additionalProperties"] != false {
				t.Fatalf("success response permits unknown fields: %v", responseSchema)
			}
			continue
		}
		if responseSchema["additionalProperties"] != false {
			t.Fatalf("%s error response permits unknown fields: %v", status, responseSchema)
		}
		properties := requireCharacterizationMap(t, responseSchema, "properties")
		if len(properties) != 1 {
			t.Fatalf("%s error properties=%v, want code only", status, properties)
		}
		code := requireCharacterizationMap(t, properties, "code")
		actual := schemaStringValues(code)
		if fmt.Sprint(actual) != fmt.Sprint(wantCodes[status]) {
			t.Fatalf("%s error codes=%v, want %v", status, actual, wantCodes[status])
		}
	}

	success := resolveCharacterizationSchema(t, bundle, requireCharacterizationMap(t,
		requireCharacterizationMap(t, resolveCharacterizationSchema(t, bundle, responses["200"]), "content"),
		"application/json",
	)["schema"])
	raw, err := json.Marshal(success)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"raw_token", "runtime_token", "token_id", "raw_scopes", "service_id", "node_id",
		"token_hash", "ciphertext", "nonce", "activation_token", "configure_token",
		"private_key", "raw_error", "stack", "url",
	} {
		if strings.Contains(lower, `"`+forbidden+`"`) {
			t.Fatalf("success response exposes forbidden field %q", forbidden)
		}
	}
}

func cloneNodeProjectionPayload(t *testing.T, source map[string]any) map[string]any {
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

func assertNodeProjectionPrivacyHeaders(t *testing.T, bundle map[string]any, status string, response map[string]any) {
	t.Helper()
	headers := requireCharacterizationMap(t, response, "headers")
	want := map[string]string{
		"Content-Type":    "application/json",
		"Cache-Control":   "no-store, no-cache",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	}
	for name, value := range want {
		header := resolveCharacterizationSchema(t, bundle, headers[name])
		schema := requireCharacterizationMap(t, header, "schema")
		if header["required"] != true || schema["const"] != value {
			t.Fatalf("%s %s header=%v, want required const %q", status, name, header, value)
		}
	}
}

func schemaStringValues(schema map[string]any) []string {
	if value, ok := schema["const"].(string); ok {
		return []string{value}
	}
	values, _ := schema["enum"].([]any)
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprint(value))
	}
	return out
}
