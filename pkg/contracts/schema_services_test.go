package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControlOpenAPIIncludesServiceRegistrationContracts(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"/services/register:",
		"#/components/schemas/ServiceRegistrationRequest",
		"#/components/schemas/RegisteredService",
		"invalid service registration payload",
		"pattern: \"^https?://\"",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing %q", want)
		}
	}
}

func TestServiceAssignmentRoleSchemas(t *testing.T) {
	for _, file := range []string{
		"service-assignment.schema.json",
		"service-assignment-write.schema.json",
		"registered-service.schema.json",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"assignment_role",
			"primary",
			"standby",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing service assignment role marker %q", file, want)
			}
		}
	}
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"assignment_role:",
		"standby is registered as a failover candidate but is not started automatically",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing service assignment role marker %q", want)
		}
	}
	requireControlOpenAPIServiceAssignmentRole(t)
}

func TestRegisteredServiceSchemaDoesNotExposeTokenBindingID(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "registered-service.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(body)), "token_id") {
		t.Fatal("registered-service.schema.json must not expose service token binding ids")
	}
}

func TestServiceRuntimeConfigSchemaDocumentsSecretBoundary(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"ServiceRuntimeConfig",
		"service-assignment.schema.json",
		"authenticated service only",
		"standby services are failover candidates",
		"cannot resolve stream-scoped secrets until promoted",
		"non-secret runtime profiles",
		"runtime secret reference names",
		"stream_archive_configs",
		"stream_youtube_configs",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("service-runtime-config.schema.json is missing runtime config boundary marker %q", want)
		}
	}
	for _, want := range []string{
		"Runtime config for the authenticated service",
		"Non-secret runtime profiles",
		"runtime secret reference names",
		"stream_archive_configs:",
		"stream_youtube_configs:",
		"standby services are failover candidates and cannot resolve stream-scoped secrets until promoted",
		"Always no-store because runtime config includes assignments and runtime secret reference names.",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing runtime config boundary marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"\"token_id\"",
		"raw_secret",
		"webhook_url\"",
		"stream_key\":",
		"refresh_token\"",
		"folder_id\"",
		"smtp_password\"",
	} {
		if strings.Contains(strings.ToLower(raw), forbidden) {
			t.Fatalf("service-runtime-config.schema.json exposes forbidden raw runtime field marker %q", forbidden)
		}
	}
}

func TestControlOpenAPIDocumentsRuntimeSecretLease(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"/services/runtime-secrets/resolve:",
		"runtime_secret_lease_active",
		"409",
		"server-side lease",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing runtime secret lease marker %q", want)
		}
	}
}

func TestStartReadinessContractsDocumentIntegrationIssueCodes(t *testing.T) {
	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	schemaBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "start-readiness-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"youtube_stream_key_unavailable",
		"youtube_oauth_account_unavailable",
		"youtube_relay_static_unavailable",
		"youtube_relay_static_binding_unavailable",
		"youtube_relay_binding_store_unavailable",
		"youtube_relay_binding_in_use",
		"youtube_relay_static_recovery_required",
		"archive_profile_invalid_config",
		"drive_destination_unavailable",
		"drive_oauth_account_unavailable",
		"discord_config_service_mismatch",
		"discord_caption_audio_forward_unavailable",
		"worker_deepgram_transcription_unavailable",
		"side-effect-free",
		"primary_service_count",
		"Number of primary assignments",
		"standby",
		"Service token binding IDs and raw tokens are never returned",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing readiness issue marker %q", want)
		}
		if !strings.Contains(string(schemaBody), want) {
			t.Fatalf("start-readiness-response.schema.json is missing readiness issue marker %q", want)
		}
	}
	if strings.Contains(strings.ToLower(string(schemaBody)), "raw_secret") {
		t.Fatal("start-readiness schema must not expose raw secret fields")
	}
	if strings.Contains(strings.ToLower(string(schemaBody)), "token_id") {
		t.Fatal("start-readiness schema must not expose service token binding ids")
	}
}

func TestExternalE2EConfigContractsDocumentSecretBoundary(t *testing.T) {
	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	schemaBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "external-e2e-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/streams/{id}/external-e2e-config:",
		"#/components/schemas/ExternalE2EConfigResponse",
		"streams.read",
		"Cache-Control",
		"no-store",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing external verification config marker %q", want)
		}
	}
	for _, want := range []string{
		"ExternalE2EConfigResponse",
		"streams.read",
		"Cache-Control",
		"no-store",
		"schema_version",
		"runtime_config",
		"service_assignments",
		"confirmations",
		"readiness",
		"youtube_output_id",
		"drive_destination_id",
		"discord_config_id",
		"encoder_profile_id",
		"archive_profile_id",
		"discord_bot_service_id",
		"encoder_recorder_primary_service_id",
		"worker_primary_service_id",
		"runtime_config_distribution_enabled",
		"missing_runtime_ids",
		"missing_primary_services",
		"missing_runtime_config_capabilities",
	} {
		if !strings.Contains(string(schemaBody), want) {
			t.Fatalf("external-e2e-config.schema.json is missing marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"discord_guild_id",
		"discord_voice_channel_id",
		"drive_folder_id",
		"folder_id\"",
		"refresh_token",
		"client_secret",
		"stream_key",
		"rtmp_url",
		"session_cookie",
		"token_id",
		"raw_secret",
	} {
		if strings.Contains(strings.ToLower(string(schemaBody)), forbidden) {
			t.Fatalf("external-e2e-config.schema.json exposes forbidden raw field marker %q", forbidden)
		}
	}
}

func TestDiagnosticRerunContractsRemainReportOnly(t *testing.T) {
	schema, err := os.ReadFile(filepath.Join("..", "..", "schemas", "diagnostic-rerun-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		AdditionalProperties bool           `json:"additionalProperties"`
		Required             []string       `json:"required"`
		Properties           map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(schema, &document); err != nil {
		t.Fatal(err)
	}
	if document.AdditionalProperties || !stringSliceContainsForSchemaTest(document.Required, "incident") || !stringSliceContainsForSchemaTest(document.Required, "outcome") {
		t.Fatalf("diagnostic rerun response schema must be a bounded incident/outcome result: %#v", document)
	}
	if _, ok := document.Properties["reason"]; !ok {
		t.Fatalf("diagnostic rerun response schema must expose a safe inconclusive reason: %#v", document.Properties)
	}

	openAPIs := map[string][]string{
		"observability-api.yaml": {
			"/incidents/{id}/diagnostics/rerun:",
			"Requires diagnostics.run",
			"never changes incident lifecycle, executes remediation, or sends notifications",
			"#/components/schemas/DiagnosticRerunResponse",
		},
		"control-api.yaml": {
			"/observability/incidents/{id}/diagnostics/rerun:",
			"Requires diagnostics.run",
			"never resolves the incident or executes remediation",
			"#/components/schemas/DiagnosticRerunResponse",
		},
	}
	for file, wants := range openAPIs {
		body, err := os.ReadFile(filepath.Join("..", "..", "openapi", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing diagnostic rerun contract marker %q", file, want)
			}
		}
	}
}
