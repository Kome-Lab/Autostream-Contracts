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
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
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
