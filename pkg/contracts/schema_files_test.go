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
		"\"format\": \"email\"",
		"\"smtp_from\"",
		"\"smtp_password\"",
		"\"writeOnly\": true",
		"Write-only Discord, Slack, or generic webhook URL",
	} {
		if !strings.Contains(string(writeBody), want) {
			t.Fatalf("notification-channel-write.schema.json is missing email marker %q", want)
		}
	}
	for _, want := range []string{
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
		"proxied notification channels without raw webhook URLs or raw SMTP settings",
		"masked_email_target:",
		"smtp_password_configured:",
		"smtp_password:",
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
		"writeOnly: true",
		"Write-only secret. Only http/https absolute URLs are accepted.",
	} {
		if !strings.Contains(string(observabilityOpenAPIBody), want) {
			t.Fatalf("observability-api.yaml is missing webhook write-only marker %q", want)
		}
	}
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
