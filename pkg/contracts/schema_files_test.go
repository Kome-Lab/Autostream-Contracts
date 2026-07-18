package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaFilesAreValidJSONAndDoNotExposeRawSecrets(t *testing.T) {
	root := filepath.Join("..", "..", "schemas")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected schema files")
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var doc map[string]any
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatalf("%s is not valid JSON: %v", entry.Name(), err)
		}
		raw := strings.ToLower(string(body))
		for _, forbidden := range []string{"secrets.read_raw", "raw_secret"} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("%s exposes forbidden raw secret contract marker %q", entry.Name(), forbidden)
			}
		}
	}
}

func TestSystemUpdateContractsKeepExecutionDetailsServerSide(t *testing.T) {
	for _, file := range []string{
		"system-update-create-request.schema.json",
		"update-agent-claim-request.schema.json",
		"update-agent-report-request.schema.json",
		"update-agent-authorize-request.schema.json",
		"update-agent-mutation-grant-issue-request.schema.json",
		"update-agent-mutation-grant-consume-request.schema.json",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			AdditionalProperties bool                      `json:"additionalProperties"`
			Properties           map[string]map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatal(err)
		}
		if doc.AdditionalProperties {
			t.Fatalf("%s must reject unknown execution fields", file)
		}
		for _, forbidden := range []string{
			"url", "path", "command", "unit", "image", "version", "digest",
			"ssh_address", "ssh_user", "ssh_path", "identity_file", "remote_command",
		} {
			if _, ok := doc.Properties[forbidden]; ok {
				t.Fatalf("%s must not accept caller-supplied %s", file, forbidden)
			}
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/system-updates:",
		"/system-updates/{id}/cancel:",
		"/services/update-jobs/claim:",
		"/services/update-jobs/{id}/report:",
		"/services/update-jobs/{id}/authorize:",
		"/services/update-jobs/{id}/mutation-grants:",
		"/services/update-jobs/{id}/mutation-grants/consume:",
		"The request cannot supply a URL, path, image, command, digest, version, or systemd unit.",
		"Atomically claims the next eligible job",
		"Idempotently reports monotonic progress",
		"Permanently disabled legacy per-host mutation authorization endpoint",
		"one-time mutation grant",
		"The grant token is never accepted in the JSON body.",
	} {
		if !strings.Contains(string(openAPI), want) {
			t.Fatalf("control-api.yaml is missing system update marker %q", want)
		}
	}
}

func TestHeartbeatContractCarriesVerifiedBuildIdentity(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "heartbeat.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"version", "commit", "build_date", "hostname", "os", "arch", "capabilities", "api"} {
		if !strings.Contains(string(body), `"`+want+`"`) {
			t.Fatalf("heartbeat schema is missing %q", want)
		}
	}
}

func TestURLSchemasRestrictHTTPOnly(t *testing.T) {
	tests := []struct {
		file  string
		field string
	}{
		{file: "service-registration.schema.json", field: "public_url"},
		{file: "registered-service.schema.json", field: "public_url"},
		{file: "notification-channel-write.schema.json", field: "webhook_url"},
	}
	for _, tt := range tests {
		t.Run(tt.file+"."+tt.field, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("..", "..", "schemas", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(body, &doc); err != nil {
				t.Fatal(err)
			}
			properties, ok := doc["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s has no properties", tt.file)
			}
			field, ok := properties[tt.field].(map[string]any)
			if !ok {
				t.Fatalf("%s has no %s field", tt.file, tt.field)
			}
			if field["format"] != "uri" || field["pattern"] != "^https?://" {
				t.Fatalf("%s %s must require http/https URI, got %#v", tt.file, tt.field, field)
			}
		})
	}
}

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
		"enum: [primary, standby]",
		"standby is registered as a failover candidate but is not started automatically",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing service assignment role marker %q", want)
		}
	}
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

func TestWorkerEventSchemaDocumentsDiscordChatOverlay(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "worker-event.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"overlay.discord_chat",
		"message_id",
		"user_id",
		"display_name",
		"text_channel_id",
		"created_at",
		"Discord Chat Channel ID",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("worker-event.schema.json is missing Discord chat overlay marker %q", want)
		}
	}
}

func TestControlOpenAPIDocumentsPasskeyCSRF(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"/auth/passkeys/register/start:",
		"/auth/passkeys/register/finish:",
		"/auth/passkeys/login/start:",
		"/auth/passkeys/login/finish:",
		"/auth/passkeys/{id}:",
		"$ref: \"#/components/parameters/csrfToken\"",
		"#/components/schemas/PasskeyRegistrationStartResponse",
		"#/components/schemas/PasskeyRegistrationFinishRequest",
		"#/components/schemas/PasskeyLoginStartResponse",
		"#/components/schemas/PasskeyLoginFinishRequest",
		"Cache-Control",
		"name: X-CSRF-Token",
		"Required for unsafe cookie-authenticated requests.",
		"must not enumerate users",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing passkey CSRF contract marker %q", want)
		}
	}
}

func TestControlOpenAPIDocumentsMFARolePolicy(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"#/components/schemas/SecuritySettings",
		"mfa_required_roles:",
		"Empty means an enabled mfa_mode applies to all users.",
		"enum: [disabled, totp, passkey]",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing MFA role policy marker %q", want)
		}
	}
}

func TestDeepgramCaptionAndSessionRefreshContracts(t *testing.T) {
	files := map[string][]string{
		"caption-profile-config.schema.json": {
			"CaptionProfileConfig", "deepgram", "nova-3", "deepgram_api_key", "manual caption input is not a runtime provider",
		},
		"discord-bot-start-job-request.schema.json": {
			"caption_audio_url", "caption_audio_token", "Short-lived stream-scoped token",
		},
		"session-refresh-response.schema.json": {
			"SessionRefreshResponse", "idle_expires_at", "absolute_expires_at", "Activity refresh never moves this timestamp",
		},
	}
	for file, wants := range files {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing %q", file, want)
			}
		}
	}

	runtimeConfig, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(runtimeConfig), "caption_audio_url") {
		t.Fatal("service runtime config must not accept a profile supplied caption audio URL")
	}

	openapi, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/auth/session/refresh:",
		"#/components/schemas/SessionRefreshResponse",
		"without extending its absolute lifetime",
		"discord_caption_audio_forward_unavailable",
		"worker_deepgram_transcription_unavailable",
		"minimum: 8",
	} {
		if !strings.Contains(string(openapi), want) {
			t.Fatalf("control-api.yaml is missing %q", want)
		}
	}
}

func TestOAuthLoginSchemasDocumentProviderAndSecretBoundaries(t *testing.T) {
	files := []string{
		"oauth-login-provider.schema.json",
		"oauth-login-start-request.schema.json",
		"oauth-login-start-response.schema.json",
		"oauth-login-callback-request.schema.json",
		"oauth-user-link.schema.json",
		"oauth-user-link-write.schema.json",
	}
	for _, file := range files {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := strings.ToLower(string(body))
		for _, forbidden := range []string{
			"\"client_secret\":",
			"\"access_token\":",
			"\"refresh_token\":",
			"\"smtp_password\":",
			"\"webhook_url\":",
		} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("%s must not expose operational secret field %q", file, forbidden)
			}
		}
	}

	providerBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "oauth-login-provider.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	providerRaw := string(providerBody)
	for _, want := range []string{
		"OAuthLoginProvider",
		"google",
		"github",
		"discord",
		"Operational Google connected accounts are represented by oauth-account.schema.json instead.",
	} {
		if !strings.Contains(providerRaw, want) {
			t.Fatalf("oauth-login-provider schema is missing %q", want)
		}
	}

	startBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "oauth-login-start-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	startRaw := string(startBody)
	for _, want := range []string{
		"OAuthLoginStartResponse",
		"authorization_url",
		"state",
		"nonce",
		"HttpOnly state cookie",
	} {
		if !strings.Contains(startRaw, want) {
			t.Fatalf("oauth-login-start-response schema is missing %q", want)
		}
	}

	callbackBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "oauth-login-callback-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	callbackRaw := string(callbackBody)
	for _, want := range []string{
		"OAuthLoginCallbackRequest",
		"writeOnly",
		"code",
		"HttpOnly state cookie",
		"must never be logged",
	} {
		if !strings.Contains(callbackRaw, want) {
			t.Fatalf("oauth-login-callback-request schema is missing %q", want)
		}
	}

	userLinkBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "oauth-user-link.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	userLinkRaw := string(userLinkBody)
	for _, want := range []string{
		"OAuthUserLink",
		"provider_type",
		"subject",
		"manual link creation should not be exposed in normal UI",
	} {
		if !strings.Contains(userLinkRaw, want) {
			t.Fatalf("oauth-user-link schema is missing %q", want)
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openapiRaw := string(openapiBody)
	for _, want := range []string{
		"Manual OAuth login links are disabled",
		"manual_oauth_link_disabled",
		"manual_oauth_account_create_disabled",
		"manual OAuth refresh token update is disabled",
		"OAuth user links are created only",
		"OAuth callback ceremony",
	} {
		if !strings.Contains(openapiRaw, want) {
			t.Fatalf("control-api.yaml is missing OAuth user link safety marker %q", want)
		}
	}
}

func TestOAuthAccountSchemaIncludesOperatorDisplayNames(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "oauth-account.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{"provider_name", "account_label", "display_name", "configured account label", "stable short account reference"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("oauth-account.schema.json is missing display-name marker %q", want)
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openapiRaw := string(openapiBody)
	for _, want := range []string{"provider_name:", "display_name:", "configured account label", "stable short account reference"} {
		if !strings.Contains(openapiRaw, want) {
			t.Fatalf("control-api.yaml is missing OAuth account display-name marker %q", want)
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

func TestNotificationChannelSchemasDocumentEmailSecretBoundary(t *testing.T) {
	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	publicBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	observabilityOpenAPIBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "observability-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\"email\"",
		"\"webhook_url\"",
		"\"email_recipients\"",
		"\"uses_global_smtp\"",
		"\"format\": \"email\"",
		"\"smtp_from\"",
		"\"smtp_password\"",
		"\"deprecated\": true",
		"\"writeOnly\": true",
		"Write-only Discord, Slack, or generic webhook URL",
	} {
		if !strings.Contains(string(writeBody), want) {
			t.Fatalf("notification-channel-write.schema.json is missing email marker %q", want)
		}
	}
	for _, want := range []string{
		"\"uses_global_smtp\"",
		"\"smtp_password_configured\"",
		"\"masked_email_target\"",
	} {
		if !strings.Contains(string(publicBody), want) {
			t.Fatalf("notification-channel.schema.json is missing public email marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"\"smtp_host\"",
		"\"smtp_from\"",
		"\"smtp_username\"",
		"\"smtp_password\"",
		"\"email_recipients\"",
	} {
		if strings.Contains(string(publicBody), forbidden) {
			t.Fatalf("notification-channel.schema.json exposes raw email notification field %q", forbidden)
		}
	}
	for _, want := range []string{
		"enum: [discord, slack, generic, email]",
		"NotificationChannel:",
		"uses_global_smtp:",
		"Deprecated direct-Observability SMTP compatibility status.",
		"proxied notification channels without raw webhook URLs or raw SMTP settings",
		"masked_email_target:",
		"smtp_password_configured:",
		"writeOnly: true",
		"format: email",
		"Raw email recipients and SMTP settings are never returned as unmasked response fields.",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing email notification marker %q", want)
		}
	}
	for _, want := range []string{
		"webhook_url:",
		"uses_global_smtp:",
		"writeOnly: true",
		"Write-only secret. Only http/https absolute URLs are accepted.",
	} {
		if !strings.Contains(string(observabilityOpenAPIBody), want) {
			t.Fatalf("observability-api.yaml is missing webhook write-only marker %q", want)
		}
	}
}

func TestControlNotificationChannelWriteContractsExcludeLegacySMTP(t *testing.T) {
	type property struct {
		Type        string `json:"type"`
		MinItems    int    `json:"minItems"`
		WriteOnly   bool   `json:"writeOnly"`
		Description string `json:"description"`
	}
	read := func(name string) ([]byte, map[string]property) {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]property `json:"properties"`
		}
		if err := json.Unmarshal(body, &schema); err != nil {
			t.Fatal(err)
		}
		return body, schema.Properties
	}

	updateBody, updateProperties := read("control-notification-channel-update.schema.json")
	for _, forbidden := range []string{"uses_global_smtp", "smtp_host", "smtp_port", "smtp_tls", "smtp_from", "smtp_username", "smtp_password"} {
		if _, exists := updateProperties[forbidden]; exists {
			t.Fatalf("Control notification update schema exposes forbidden field %q", forbidden)
		}
	}
	if updateProperties["email_recipients"].MinItems != 1 || !strings.Contains(string(updateBody), "Omission preserves existing recipients") {
		t.Fatal("Control notification update must reject explicit empty recipients while preserving omission")
	}
	migration, exists := updateProperties["migrate_to_global_smtp"]
	if !exists || migration.Type != "boolean" || !migration.WriteOnly || !strings.Contains(migration.Description, "Omission or false preserves") {
		t.Fatalf("Control notification update must expose only the explicit write-only global SMTP migration flag: %#v", migration)
	}

	createBody, _ := read("control-notification-channel-create.schema.json")
	var create struct {
		AllOf []struct {
			Ref string `json:"$ref"`
			Not struct {
				Required []string `json:"required"`
			} `json:"not"`
			Then struct {
				Required []string `json:"required"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(createBody, &create); err != nil {
		t.Fatal(err)
	}
	if len(create.AllOf) != 3 || create.AllOf[0].Ref != "control-notification-channel-update.schema.json" || !stringSliceContainsForSchemaTest(create.AllOf[1].Not.Required, "migrate_to_global_smtp") || !stringSliceContainsForSchemaTest(create.AllOf[2].Then.Required, "email_recipients") {
		t.Fatalf("Control email create must require recipients on top of its update shape: %#v", create.AllOf)
	}

	controlOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(controlOpenAPI)
	for _, want := range []string{
		"#/components/schemas/ControlNotificationChannelCreateRequest",
		"#/components/schemas/ControlNotificationChannelUpdateRequest",
		"migrate_to_global_smtp is the only supported delivery-mode transition",
		"an explicitly empty array is invalid",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing Control notification write marker %q", want)
		}
	}
	for _, forbidden := range []string{"    NotificationChannelWriteRequest:", "    NotificationChannelCreateRequest:"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("control-api.yaml still exposes legacy request field/component %q", forbidden)
		}
	}
	start := strings.Index(raw, "    ControlNotificationChannelUpdateRequest:")
	if start < 0 {
		t.Fatal("could not isolate Control notification request components")
	}
	endOffset := strings.Index(raw[start:], "\n    NotificationDeliveryResult:")
	if endOffset < 0 {
		t.Fatal("could not isolate Control notification request components")
	}
	requestComponents := raw[start : start+endOffset]
	if !strings.Contains(requestComponents, "        migrate_to_global_smtp:\n          type: boolean\n          writeOnly: true") {
		t.Fatal("Control notification update OpenAPI component must expose migrate_to_global_smtp as a write-only boolean")
	}
	for _, forbidden := range []string{"        uses_global_smtp:", "        smtp_host:", "        smtp_port:", "        smtp_tls:", "        smtp_from:", "        smtp_username:", "        smtp_password:"} {
		if strings.Contains(requestComponents, forbidden) {
			t.Fatalf("Control notification request components expose forbidden field %q", forbidden)
		}
	}
}

func TestServiceNotificationEmailRelayContracts(t *testing.T) {
	requestBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-notification-email-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	responseBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-notification-email-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	serviceTokenBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-token-create-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serviceTokenBody), `"notifications.email.send"`) {
		t.Fatal("service token create schema is missing the dedicated notification email scope")
	}

	type property struct {
		Type        string `json:"type"`
		MinItems    int    `json:"minItems"`
		MaxItems    int    `json:"maxItems"`
		MinLength   int    `json:"minLength"`
		MaxLength   int    `json:"maxLength"`
		Pattern     string `json:"pattern"`
		UniqueItems bool   `json:"uniqueItems"`
		Const       string `json:"const"`
	}
	var request struct {
		AdditionalProperties bool                `json:"additionalProperties"`
		Required             []string            `json:"required"`
		Properties           map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(requestBody, &request); err != nil {
		t.Fatal(err)
	}
	if request.AdditionalProperties {
		t.Fatal("service notification email request must reject unknown fields")
	}
	for _, field := range []string{"recipients", "subject", "text"} {
		if !stringSliceContainsForSchemaTest(request.Required, field) {
			t.Fatalf("service notification email request must require %q", field)
		}
	}
	if stringSliceContainsForSchemaTest(request.Required, "html") {
		t.Fatal("service notification email HTML alternative must remain optional")
	}
	recipients := request.Properties["recipients"]
	if recipients.MinItems != 1 || recipients.MaxItems != 20 || !recipients.UniqueItems {
		t.Fatalf("recipient bounds must be 1..20 unique, got %#v", recipients)
	}
	subject := request.Properties["subject"]
	if subject.MinLength != 1 || subject.MaxLength != 200 || !strings.Contains(subject.Pattern, `\r`) || !strings.Contains(subject.Pattern, `\n`) {
		t.Fatalf("subject must be 1..200 code points with CR/LF excluded, got %#v", subject)
	}
	text := request.Properties["text"]
	if text.MinLength != 1 || text.MaxLength != 16384 || !strings.Contains(text.Pattern, `\u0000`) {
		t.Fatalf("text must be 1..16384 with NUL excluded, got %#v", text)
	}
	html := request.Properties["html"]
	if html.MaxLength != 65536 || !strings.Contains(html.Pattern, `\u0000`) {
		t.Fatalf("optional HTML must be at most 65536 with NUL excluded, got %#v", html)
	}

	var response struct {
		Required   []string            `json:"required"`
		Properties map[string]property `json:"properties"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		t.Fatal(err)
	}
	if response.Properties["status"].Const != "sent" || !stringSliceContainsForSchemaTest(response.Required, "recipient_count") {
		t.Fatalf("service notification email response must expose only sent status and recipient count: %#v", response)
	}

	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var write struct {
		AllOf []struct {
			Then struct {
				Required   []string            `json:"required"`
				Properties map[string]property `json:"properties"`
				Not        struct {
					AnyOf []struct {
						Required []string `json:"required"`
					} `json:"anyOf"`
				} `json:"not"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(writeBody, &write); err != nil {
		t.Fatal(err)
	}
	if len(write.AllOf) != 1 {
		t.Fatalf("expected one global SMTP compatibility condition: %#v", write.AllOf)
	}
	if stringSliceContainsForSchemaTest(write.AllOf[0].Then.Required, "email_recipients") {
		t.Fatalf("update schema must allow omitted recipients to preserve the existing masked set: %#v", write.AllOf)
	}
	if write.AllOf[0].Then.Properties["type"].Const != "email" {
		t.Fatalf("uses_global_smtp=true must be limited to email channels: %#v", write.AllOf[0].Then.Properties)
	}
	forbidden := map[string]bool{}
	for _, condition := range write.AllOf[0].Then.Not.AnyOf {
		for _, field := range condition.Required {
			forbidden[field] = true
		}
	}
	for _, field := range []string{"smtp_host", "smtp_port", "smtp_tls", "smtp_from", "smtp_username", "smtp_password"} {
		if !forbidden[field] {
			t.Fatalf("uses_global_smtp=true must reject legacy field %q", field)
		}
	}

	createBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-channel-create.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var create struct {
		AllOf []struct {
			Ref  string `json:"$ref"`
			Then struct {
				Required []string `json:"required"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(createBody, &create); err != nil {
		t.Fatal(err)
	}
	if len(create.AllOf) != 2 || create.AllOf[0].Ref != "notification-channel-write.schema.json" || !stringSliceContainsForSchemaTest(create.AllOf[1].Then.Required, "email_recipients") {
		t.Fatalf("email create schema must require recipients while reusing the update-compatible write schema: %#v", create.AllOf)
	}

	controlOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/services/notifications/email:",
		"requires the dedicated notifications.email.send scope",
		"#/components/schemas/ServiceNotificationEmailRequest",
		"#/components/schemas/ServiceNotificationEmailResponse",
		"#/components/schemas/ControlNotificationChannelCreateRequest",
		"#/components/schemas/NotificationDeliveryResult",
		"status:",
		"const: sent",
		"recipient_count:",
		"smtp_not_configured",
		"smtp_requires_tls",
		"invalid_email_notification",
		"smtp_auth_failed",
		"smtp_recipient_rejected",
		"send_failed",
		"service_type_not_allowed",
		"service_token_not_registered",
		"rate_limited",
		"list_services_failed",
		"app_settings_failed",
		"secret_encryption_key_required",
	} {
		if !strings.Contains(string(controlOpenAPI), want) {
			t.Fatalf("control-api.yaml is missing service email relay marker %q", want)
		}
	}
}

func TestAdministrativeNotificationEventContracts(t *testing.T) {
	if NotificationAdminAudit != NotificationEventType("admin.audit") {
		t.Fatalf("unexpected admin audit event constant %q", NotificationAdminAudit)
	}

	eventBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-event-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	resultBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "notification-delivery-result.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	type eventProperty struct {
		Const     string   `json:"const"`
		Enum      []string `json:"enum"`
		MinLength int      `json:"minLength"`
		MaxLength int      `json:"maxLength"`
		Pattern   string   `json:"pattern"`
		Format    string   `json:"format"`
	}
	var event struct {
		AdditionalProperties bool                     `json:"additionalProperties"`
		Required             []string                 `json:"required"`
		Properties           map[string]eventProperty `json:"properties"`
	}
	if err := json.Unmarshal(eventBody, &event); err != nil {
		t.Fatal(err)
	}
	if event.AdditionalProperties {
		t.Fatal("notification event request must reject metadata and other unknown fields")
	}
	for _, field := range []string{"event_type", "action"} {
		if !stringSliceContainsForSchemaTest(event.Required, field) {
			t.Fatalf("notification event request must require %q", field)
		}
	}
	if _, exists := event.Properties["metadata"]; exists {
		t.Fatal("notification event request must not define metadata")
	}
	for _, field := range []string{"event_type", "severity", "status", "action", "resource_type", "resource_id", "actor_username", "summary", "timestamp"} {
		if _, exists := event.Properties[field]; !exists {
			t.Fatalf("notification event request is missing %q", field)
		}
	}
	if event.Properties["event_type"].Const != "admin.audit" {
		t.Fatalf("notification event type must be admin.audit: %#v", event.Properties["event_type"])
	}
	for field, maxLength := range map[string]int{"status": 64, "action": 128, "resource_type": 80, "resource_id": 160, "actor_username": 80, "summary": 240} {
		property := event.Properties[field]
		if property.MaxLength != maxLength || property.Pattern == "" {
			t.Fatalf("notification event %s must document its safe length and pattern: %#v", field, property)
		}
	}
	const actionPattern = `^[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)*$`
	if event.Properties["action"].Pattern != actionPattern {
		t.Fatalf("notification event action pattern = %q, want exact implementation pattern %q", event.Properties["action"].Pattern, actionPattern)
	}
	if event.Properties["timestamp"].Format != "date-time" {
		t.Fatalf("notification event timestamp must use date-time format: %#v", event.Properties["timestamp"])
	}
	for _, marker := range []string{"Raw tokens", "credentials", "webhook URLs", "passwords", "authorization values"} {
		if !strings.Contains(string(eventBody), marker) {
			t.Fatalf("notification event schema is missing secret-boundary marker %q", marker)
		}
	}

	var result struct {
		AdditionalProperties bool                     `json:"additionalProperties"`
		Required             []string                 `json:"required"`
		Properties           map[string]eventProperty `json:"properties"`
	}
	if err := json.Unmarshal(resultBody, &result); err != nil {
		t.Fatal(err)
	}
	if result.AdditionalProperties {
		t.Fatal("notification delivery result must reject undeclared response fields")
	}
	for _, field := range []string{"event_type", "channel", "target", "status"} {
		if !stringSliceContainsForSchemaTest(result.Required, field) {
			t.Fatalf("notification delivery result must require %q", field)
		}
	}
	if !stringSliceContainsForSchemaTest(result.Properties["event_type"].Enum, "admin.audit") {
		t.Fatalf("notification delivery result event enum is missing admin.audit: %#v", result.Properties["event_type"])
	}

	observabilityOpenAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "observability-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/notification-events:",
		"operationId: createNotificationEvent",
		"#/components/schemas/NotificationEventWriteRequest",
		"#/components/schemas/NotificationDeliveryResult",
		"required: [event_type, action]",
		"const: admin.audit",
		"metadata and all other unknown properties are rejected",
		"event_id and outbox/idempotency semantics are not part of this endpoint",
		"invalid_notification_event",
		"missing_admin_scope",
		"rate_limited",
		"rate_limit_unavailable",
	} {
		if !strings.Contains(string(observabilityOpenAPI), want) {
			t.Fatalf("observability-api.yaml is missing administrative notification marker %q", want)
		}
	}
}

func stringSliceContainsForSchemaTest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestEncoderPackageSchemaDocumentsRuntimeArchiveConfig(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "encoder-package-stream-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"EncoderPackageStreamRequest",
		"archive_config",
		"folder_id_secret_name",
		"refresh_token_secret_name",
		"Raw Drive folder IDs and OAuth refresh tokens must not be sent",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("encoder package schema is missing %q", want)
		}
	}
}

func TestArchiveMetadataSchemaRedactsDriveIDs(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "archive-metadata.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"folder_id_fingerprint",
		"file_fingerprints",
		"Raw Drive file IDs must not be stored in metadata",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("archive metadata schema is missing redaction marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"\"folder_id\"",
		"\"file_ids\"",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("archive metadata schema still exposes raw Drive ID field %q", forbidden)
		}
	}
}

func TestStreamSchemasDocumentDiscordChannelOverrides(t *testing.T) {
	for _, file := range []string{"stream-write.schema.json", "stream-settings-write.schema.json", "stream-job.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"discord_config_id",
			"discord_guild_id",
			"discord_voice_channel_id",
			"discord_text_channel_id",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing stream-specific Discord routing field %q", file, want)
			}
		}
	}

	for _, file := range []string{"stream-write.schema.json", "stream-settings-write.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"auto_start_trigger",
			"discord_voice_join",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing Discord voice join auto-start field %q", file, want)
			}
		}
	}

	runtimeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"auto_start_trigger",
		"discord_voice_join",
	} {
		if !strings.Contains(string(runtimeBody), want) {
			t.Fatalf("service-runtime-config.schema.json is missing Discord voice join auto-start marker %q", want)
		}
	}
}

func TestYouTubeSchemasDocumentLiveAPILifecycle(t *testing.T) {
	for _, file := range []string{"youtube-output.schema.json", "youtube-output-write.schema.json", "youtube-runtime-config.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"live_api_dry_run",
			"live_api",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing YouTube Live API mode marker %q", file, want)
			}
		}
	}

	runtimeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeRaw := string(runtimeBody)
	for _, want := range []string{
		"broadcast_id",
		"live_stream_id",
		"watch_url",
		"dry_run",
		"complete_on_stop",
	} {
		if !strings.Contains(runtimeRaw, want) {
			t.Fatalf("youtube-runtime-config.schema.json is missing runtime lifecycle field %q", want)
		}
	}

	writeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-output-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeRaw := string(writeBody)
	for _, want := range []string{
		"writeOnly",
		"enable_auto_start",
		"enable_auto_stop",
		"complete_on_stop",
		"Required for live_api and live_api_dry_run modes.",
	} {
		if !strings.Contains(writeRaw, want) {
			t.Fatalf("youtube-output-write.schema.json is missing Live API write marker %q", want)
		}
	}
}

func TestIntegrationWriteSchemasDocumentSecretBoundaries(t *testing.T) {
	tests := []struct {
		file  string
		wants []string
	}{
		{
			file: "oauth-account-write.schema.json",
			wants: []string{
				"OAuthAccountWriteRequest",
				"manual refresh token entry is disabled",
				"refresh_token",
				"writeOnly",
				"Drive destinations require a Google Drive scope",
				"YouTube Live API outputs require a YouTube scope",
			},
		},
		{
			file: "drive-destination-write.schema.json",
			wants: []string{
				"DriveDestinationWriteRequest",
				"folder_id",
				"writeOnly",
				"supportsAllDrives=true",
				"shared drive folder IDs",
			},
		},
	}
	for _, tt := range tests {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", tt.file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range tt.wants {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing integration write-schema marker %q", tt.file, want)
			}
		}
	}
}

func TestDriveDestinationWriteSchemaAllowsUpdateWithoutRawFolderID(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "drive-destination-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Required []string `json:"required"`
		Props    map[string]struct {
			Description string `json:"description"`
			WriteOnly   bool   `json:"writeOnly"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	for _, field := range doc.Required {
		if field == "folder_id" {
			t.Fatal("drive destination write schema must allow update without resending the raw folder_id")
		}
	}
	folderID := doc.Props["folder_id"]
	if !folderID.WriteOnly {
		t.Fatal("drive destination folder_id must remain writeOnly")
	}
	for _, want := range []string{"required when creating", "optional when updating", "never returned raw"} {
		if !strings.Contains(folderID.Description, want) {
			t.Fatalf("drive destination folder_id description is missing %q: %q", want, folderID.Description)
		}
	}
}

func TestSecretStatusSchemaDocumentsManagedRuntimeSecretPrefixes(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "secret-status.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"encoder_runtime_secret_main",
		"webhook_url_main",
		"smtp_password_main",
		"Raw values are never returned",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("secret-status.schema.json is missing managed secret prefix marker %q", want)
		}
	}
}

func TestAppSettingsContractsSeparatePublicAndManagedViews(t *testing.T) {
	type schemaDocument struct {
		Properties map[string]struct {
			WriteOnly bool `json:"writeOnly"`
		} `json:"properties"`
	}

	readSchema := func(name string) schemaDocument {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
		if err != nil {
			t.Fatal(err)
		}
		var doc schemaDocument
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}

	public := readSchema("app-settings-public.schema.json")
	for _, want := range []string{"app_name", "timezone", "turnstile_site_key", "google_analytics_enabled", "google_analytics_measurement_id"} {
		if _, ok := public.Properties[want]; !ok {
			t.Fatalf("public app settings schema is missing %q", want)
		}
	}
	for name := range public.Properties {
		if strings.HasPrefix(name, "smtp_") || name == "turnstile_secret" {
			t.Fatalf("public app settings schema exposes administrator-only field %q", name)
		}
	}

	managed := readSchema("app-settings-manage.schema.json")
	for _, want := range []string{"smtp_enabled", "smtp_password_configured", "google_analytics_measurement_id"} {
		if _, ok := managed.Properties[want]; !ok {
			t.Fatalf("managed app settings schema is missing %q", want)
		}
	}
	for _, forbidden := range []string{"smtp_password", "turnstile_secret"} {
		if _, ok := managed.Properties[forbidden]; ok {
			t.Fatalf("managed app settings response exposes raw secret field %q", forbidden)
		}
	}

	write := readSchema("app-settings-write.schema.json")
	for _, secret := range []string{"smtp_password", "turnstile_secret"} {
		field, ok := write.Properties[secret]
		if !ok || !field.WriteOnly {
			t.Fatalf("app settings write field %q must be present and writeOnly", secret)
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/settings/app:",
		"/settings/app/manage:",
		"#/components/schemas/PublicAppSettings",
		"#/components/schemas/ManagedAppSettings",
		"google_analytics_measurement_id:",
		"SMTP settings and",
	} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing app settings marker %q", want)
		}
	}
}

func TestPreviewAndDiscordNotificationContracts(t *testing.T) {
	files := map[string][]string{
		"stream-preview-link.schema.json": {
			"StreamPreviewLink", "expires_at", "bearer capability", "uri-reference", "12 hours", "stream is no longer",
		},
		"youtube-live-notification-request.schema.json": {
			"YouTubeLiveNotificationRequest", "event_id", "watch_url", "Idempotency key", "runtime config",
		},
		"youtube-live-notification-response.schema.json": {
			"YouTubeLiveNotificationResponse", "message_id", "already_sent",
		},
		"service-notification-error.schema.json": {
			"ServiceNotificationError", "retryable",
		},
	}
	for file, wants := range files {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing %q", file, want)
			}
		}
	}

	openAPIs := map[string][]string{
		"control-api.yaml": {
			"/streams/{id}/preview/{name}:",
			"/streams/{id}/preview-links:",
			"/stream-previews/{token}/{name}:",
			"segment-[0-9]{6}",
			"12 hours",
			"never rolls the live stream",
			"watch URL fingerprint",
		},
		"encoder-recorder-api.yaml": {
			"/streams/{id}/preview/{name}:",
			"serviceToken",
			"six-segment rolling playlist",
			"must not stop either primary output",
		},
		"discord-bot-api.yaml": {
			"/streams/{id}/notifications/youtube-live:",
			"event_id",
			"watch_url",
			"mentions disabled",
			"runtime config",
			"Retry-After",
		},
	}
	for file, wants := range openAPIs {
		body, err := os.ReadFile(filepath.Join("..", "..", "openapi", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing %q", file, want)
			}
		}
	}
}

func TestYouTubeOutputWatchURLContractIsBackwardCompatible(t *testing.T) {
	for _, file := range []string{"youtube-output.schema.json", "youtube-output-write.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "watch_url") {
			t.Fatalf("%s is missing watch_url", file)
		}
	}

	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-output-write.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	for _, required := range doc.Required {
		if required == "watch_url" {
			t.Fatal("watch_url must remain optional in the API for existing stream_key profiles")
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"watch_url:", "New UI-created profiles require it", "existing API profiles without this field remain compatible"} {
		if !strings.Contains(string(openapiBody), want) {
			t.Fatalf("control-api.yaml is missing YouTube watch URL compatibility marker %q", want)
		}
	}
}
