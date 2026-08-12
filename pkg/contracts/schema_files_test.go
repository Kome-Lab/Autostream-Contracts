package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdaterVersionEndpointContractIsUnauthenticatedAndMinimal(t *testing.T) {
	for _, file := range []string{
		"control-api.yaml",
		"observability-api.yaml",
		"encoder-recorder-api.yaml",
		"discord-bot-api.yaml",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "openapi", file))
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		raw := string(body)
		const pathMarker = "  /updater/version:\n"
		start := strings.Index(raw, pathMarker)
		if start < 0 {
			t.Fatalf("%s is missing %s", file, strings.TrimSpace(pathMarker))
		}
		section := raw[start+len(pathMarker):]
		if end := strings.Index(section, "\n  /"); end >= 0 {
			section = section[:end]
		}
		for _, want := range []string{
			"operationId: getUpdaterVersion",
			"security: []",
			"additionalProperties: false",
			"required: [version]",
			"pattern: '^v[0-9]+\\.[0-9]+\\.[0-9]+",
		} {
			if !strings.Contains(section, want) {
				t.Fatalf("%s updater version contract is missing %q", file, want)
			}
		}
	}
}

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
		"author_id",
		"user_id",
		"display_name",
		"content",
		"text",
		"avatar_url",
		"is_bot",
		"text_channel_id",
		"created_at",
		"Discord Chat Channel ID",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("worker-event.schema.json is missing Discord chat overlay marker %q", want)
		}
	}
}

func TestWorkerEventSchemaValidatesDiscordChatPayloadCompatibility(t *testing.T) {
	schema := compileContractJSONSchema(t, "worker-event.schema.json")

	tests := []struct {
		name    string
		payload map[string]any
		valid   bool
	}{
		{
			name: "canonical payload",
			payload: map[string]any{
				"message_id":      "message-01",
				"author_id":       "user-01",
				"display_name":    "Alice",
				"content":         "hello",
				"avatar_url":      "https://cdn.example.test/avatar.png",
				"is_bot":          false,
				"text_channel_id": "channel-01",
				"created_at":      "2026-08-12T07:30:00Z",
			},
			valid: true,
		},
		{
			name: "legacy aliases remain accepted",
			payload: map[string]any{
				"message_id":      "message-legacy",
				"user_id":         "user-legacy",
				"display_name":    "Legacy User",
				"text":            "legacy text",
				"text_channel_id": "channel-01",
				"created_at":      "2026-08-12T07:30:00Z",
				"legacy_extra":    "preserved",
			},
			valid: true,
		},
		{
			name: "canonical and legacy aliases can coexist during migration",
			payload: map[string]any{
				"author_id": "user-01",
				"user_id":   "user-01",
				"content":   "hello",
				"text":      "hello",
			},
			valid: true,
		},
		{
			name:    "previously accepted empty payload remains accepted",
			payload: map[string]any{},
			valid:   true,
		},
		{
			name: "canonical bot marker must be boolean",
			payload: map[string]any{
				"author_id": "bot-01",
				"content":   "hello",
				"is_bot":    "false",
			},
			valid: false,
		},
		{
			name: "canonical author id must be string",
			payload: map[string]any{
				"author_id": 123,
				"content":   "hello",
			},
			valid: false,
		},
		{
			name: "canonical content must be string",
			payload: map[string]any{
				"author_id": "user-01",
				"content":   123,
			},
			valid: false,
		},
		{
			name: "canonical avatar url must be string",
			payload: map[string]any{
				"author_id":  "user-01",
				"content":    "hello",
				"avatar_url": true,
			},
			valid: false,
		},
		{
			name: "legacy user id alias must be string",
			payload: map[string]any{
				"user_id": false,
				"text":    "legacy text",
			},
			valid: false,
		},
		{
			name: "legacy text alias must be string",
			payload: map[string]any{
				"user_id": "user-legacy",
				"text":    []any{"not", "text"},
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := map[string]any{
				"id":        "event-01",
				"stream_id": "stream-01",
				"type":      "overlay.discord_chat",
				"payload":   test.payload,
				"timestamp": "2026-08-12T07:30:00Z",
			}
			err := schema.Validate(event)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, validation error=%v", test.valid, err)
			}
		})
	}

	t.Run("other worker events keep opaque payload compatibility", func(t *testing.T) {
		event := map[string]any{
			"id":        "event-02",
			"stream_id": "stream-01",
			"type":      "overlay.participants",
			"payload": map[string]any{
				"producer_specific": []any{1.0, true, "value"},
			},
			"timestamp": "2026-08-12T07:30:00Z",
		}
		if err := schema.Validate(event); err != nil {
			t.Fatalf("non-chat worker event compatibility changed: %v", err)
		}
	})
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
	for _, want := range []string{"provider_name", "account_label", "display_name", "refresh_token_updated_at", "access_token_refreshed_at", "access_token_refresh_attempted_at", "access_token_refresh_failed_at", "access_token_refresh_failure_code", "access_token_refresh_relink_required", "Bounded non-secret operational failure class", "configured account label", "stable short account reference"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("oauth-account.schema.json is missing display-name marker %q", want)
		}
	}

	openapiBody, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openapiRaw := string(openapiBody)
	for _, want := range []string{"provider_name:", "display_name:", "refresh_token_updated_at:", "access_token_refreshed_at:", "access_token_refresh_attempted_at:", "access_token_refresh_failed_at:", "access_token_refresh_failure_code:", "access_token_refresh_relink_required:", "Bounded non-secret operational failure class", "configured account label", "stable short account reference"} {
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
	for _, file := range []string{"youtube-output.schema.json", "youtube-output-write.schema.json", "youtube-runtime-config.schema.json", "service-runtime-config.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"live_api_dry_run",
			"live_api",
			"live_api_relay_static",
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
		"relay_binding_id",
		"reusable_live_stream_id",
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
		"Required for live_api, live_api_dry_run, and live_api_relay_static modes.",
	} {
		if !strings.Contains(writeRaw, want) {
			t.Fatalf("youtube-output-write.schema.json is missing Live API write marker %q", want)
		}
	}
}

func TestYouTubeRelayStaticWriteRejectsIngestFields(t *testing.T) {
	validator := compileContractJSONSchema(t, "youtube-output-write.schema.json")
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "static output without ingest fields",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}`,
			valid: true,
		},
		{
			name:  "static output with rtmp url",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","rtmp_url":"rtmps://a.rtmps.youtube.com/live2"}`,
			valid: false,
		},
		{
			name:  "static output with stream key",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","stream_key":"not-a-real-key"}`,
			valid: false,
		},
		{
			name:  "static output with watch url",
			body:  `{"name":"fixed relay","mode":"live_api_relay_static","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","watch_url":"https://www.youtube.com/watch?v=abc12345"}`,
			valid: false,
		},
	}
	for _, test := range tests {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := validator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	outputValidator := compileContractJSONSchema(t, "youtube-output.schema.json")
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "incomplete static output read remains repairable",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: true,
		},
		{
			name:  "static output read with rtmp url",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","rtmp_url":"rtmps://a.rtmps.youtube.com/live2","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: false,
		},
		{
			name:  "static output read with manually supplied watch url",
			body:  `{"id":"output-1","name":"fixed relay","mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","watch_url":"https://www.youtube.com/watch?v=abc12345","created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z"}`,
			valid: false,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := outputValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}
}

func TestYouTubeRelayStaticRecoveryContractsRequireExplicitSafeResolution(t *testing.T) {
	if YouTubeOutputModeLiveAPIRelayStatic != "live_api_relay_static" {
		t.Fatalf("YouTube relay static mode wire value = %q", YouTubeOutputModeLiveAPIRelayStatic)
	}
	if ErrorCodeYouTubeRelayStaticConfigChangedReload != "youtube_relay_static_config_changed_reload" {
		t.Fatalf("YouTube relay static config-change error wire value = %q", ErrorCodeYouTubeRelayStaticConfigChangedReload)
	}
	if ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay != "live_api_requires_managed_output_relay" {
		t.Fatalf("YouTube dynamic relay error wire value = %q", ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay)
	}
	if ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream != "youtube_relay_static_completion_requires_completed_stream" {
		t.Fatalf("YouTube relay static completion guard wire value = %q", ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable != "youtube_relay_static_recovery_encoder_stop_unavailable" {
		t.Fatalf("YouTube relay static recovery unavailable wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed != "youtube_relay_static_recovery_encoder_stop_unconfirmed" {
		t.Fatalf("YouTube relay static recovery unconfirmed wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown != "youtube_relay_static_recovery_broadcast_unknown" {
		t.Fatalf("YouTube relay static recovery unknown broadcast wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid != "youtube_relay_static_recovery_dispatch_state_invalid" {
		t.Fatalf("YouTube relay static recovery invalid dispatch state wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid)
	}
	if ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed != "youtube_relay_static_recovery_complete_failed" {
		t.Fatalf("YouTube relay static recovery complete failure wire value = %q", ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed)
	}

	requestBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-relay-static-recovery-resolve-request.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var requestSchema struct {
		AdditionalProperties bool                       `json:"additionalProperties"`
		Required             []string                   `json:"required"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(requestBody, &requestSchema); err != nil {
		t.Fatal(err)
	}
	if requestSchema.AdditionalProperties || len(requestSchema.Required) != 1 || requestSchema.Required[0] != "confirm_external_cleanup" || len(requestSchema.Properties) != 1 {
		t.Fatal("relay static recovery request must accept only required confirm_external_cleanup")
	}
	var confirmation struct {
		Type  string `json:"type"`
		Const bool   `json:"const"`
	}
	if err := json.Unmarshal(requestSchema.Properties["confirm_external_cleanup"], &confirmation); err != nil {
		t.Fatal(err)
	}
	if confirmation.Type != "boolean" || !confirmation.Const {
		t.Fatal("relay static recovery must require confirm_external_cleanup=true")
	}
	requestValidator := compileContractJSONSchema(t, "youtube-relay-static-recovery-resolve-request.schema.json")
	for _, test := range []struct {
		body  string
		valid bool
	}{
		{body: `{"confirm_external_cleanup":true}`, valid: true},
		{body: `{"confirm_external_cleanup":false}`, valid: false},
		{body: `{"confirm_external_cleanup":true,"broadcast_id":"must-not-be-accepted"}`, valid: false},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := requestValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("relay static recovery request valid=%t for %s: %v", test.valid, test.body, err)
		}
	}
	if got, err := json.Marshal(YouTubeRelayStaticRecoveryResolveRequest{ConfirmExternalCleanup: true}); err != nil || string(got) != `{"confirm_external_cleanup":true}` {
		t.Fatalf("relay static recovery request wire shape = %s, err=%v", got, err)
	}

	responseBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "youtube-relay-static-recovery-resolve-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var responseSchema struct {
		AdditionalProperties bool                       `json:"additionalProperties"`
		Required             []string                   `json:"required"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(responseBody, &responseSchema); err != nil {
		t.Fatal(err)
	}
	if responseSchema.AdditionalProperties || len(responseSchema.Required) != 3 || len(responseSchema.Properties) != 3 {
		t.Fatal("relay static recovery response must expose only resolved, cleanup, and relay_binding_id")
	}
	var cleanup struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(responseSchema.Properties["cleanup"], &cleanup); err != nil {
		t.Fatal(err)
	}
	if strings.Join(cleanup.Enum, ",") != "provider_delete,provider_complete,operator_confirmed_unknown_broadcast,operator_confirmed_provider_cleanup" {
		t.Fatalf("relay static cleanup values = %v", cleanup.Enum)
	}
	responseValidator := compileContractJSONSchema(t, "youtube-relay-static-recovery-resolve-response.schema.json")
	for _, test := range []struct {
		body  string
		valid bool
	}{
		{body: `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"provider_complete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"operator_confirmed_unknown_broadcast","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":true,"cleanup":"operator_confirmed_provider_cleanup","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: true},
		{body: `{"resolved":false,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}`, valid: false},
		{body: `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","broadcast_id":"must-not-be-returned"}`, valid: false},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := responseValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("relay static recovery response valid=%t for %s: %v", test.valid, test.body, err)
		}
	}
	if got, err := json.Marshal(YouTubeRelayStaticRecoveryResolveResponse{
		Resolved:       true,
		Cleanup:        "provider_delete",
		RelayBindingID: "relay-01234567-89ab-4def-8123-456789abcdef",
	}); err != nil || string(got) != `{"resolved":true,"cleanup":"provider_delete","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}` {
		t.Fatalf("relay static recovery response wire shape = %s, err=%v", got, err)
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"youtube_relay_static_config_changed_reload",
		"live_api_requires_managed_output_relay",
		"youtube_relay_static_completion_requires_completed_stream",
	} {
		if !strings.Contains(string(openAPI), want) {
			t.Fatalf("control-api.yaml is missing relay start error code %q", want)
		}
	}
	pathMarker := "  /streams/{id}/youtube/relay-static/recovery/resolve:\n"
	start := strings.Index(string(openAPI), pathMarker)
	if start < 0 {
		t.Fatal("control-api.yaml is missing relay static recovery route")
	}
	pathSection := string(openAPI)[start+len(pathMarker):]
	if end := strings.Index(pathSection, "\n  /"); end >= 0 {
		pathSection = pathSection[:end]
	}
	for _, want := range []string{
		"operationId: resolveYouTubeRelayStaticRecovery",
		"Requires streams.stop.",
		"YouTubeRelayStaticRecoveryResolveRequest",
		"YouTubeRelayStaticRecoveryResolveResponse",
		"youtube_relay_static_external_cleanup_confirmation_required",
		"youtube_relay_static_recovery_not_found",
		"stream_relay_recovery_not_safe_while_active",
		"youtube_relay_static_recovery_not_required",
		"youtube_relay_static_recovery_encoder_stop_unavailable",
		"youtube_relay_static_recovery_encoder_stop_unconfirmed",
		"youtube_relay_static_recovery_broadcast_unknown",
		"youtube_relay_static_recovery_dispatch_state_invalid",
		"youtube_relay_static_recovery_complete_failed",
		"youtube_relay_static_recovery_cleanup_failed",
		"youtube_relay_static_recovery_cleanup_unavailable",
		"youtube_relay_static_recovery_credentials_failed",
		"resolve_youtube_relay_static_recovery_failed",
	} {
		if !strings.Contains(pathSection, want) {
			t.Fatalf("relay static recovery path is missing %q", want)
		}
	}
}

func TestYouTubeRelayStaticRuntimeConfigExcludesIngestFields(t *testing.T) {
	validator := compileContractJSONSchema(t, "youtube-runtime-config.schema.json")
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "fixed relay carries all prepared non-secret lifecycle fields",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}`,
			valid: true,
		},
		{
			name:  "fixed relay requires prepared identity and lifecycle fields",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1"}`,
			valid: false,
		},
		{
			name:  "fixed relay must complete on stop",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":false}`,
			valid: false,
		},
		{
			name:  "fixed relay cannot carry rtmp endpoint",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"rtmp_url":"rtmps://a.rtmps.youtube.com/live2"}`,
			valid: false,
		},
		{
			name:  "fixed relay cannot carry key secret reference",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"stream_key_secret_name":"youtube_stream_key_runtime_1"}`,
			valid: false,
		},
		{
			name:  "fixed relay runtime can carry canonical public watch url",
			body:  `{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}`,
			valid: true,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := validator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	serviceValidator := compileContractJSONSchema(t,
		"service-runtime-config.schema.json",
		"registered-service.schema.json",
		"encoder-output-relay-capabilities.schema.json",
		"service-assignment.schema.json",
		"profile.schema.json",
	)
	serviceRuntimePayload := func(streamYouTubeConfig string) string {
		return `{
"service":{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","ssl_enabled":false,"version":"v1","status":"online","capabilities":{},"created_at":"2026-08-09T00:00:00Z","updated_at":"2026-08-09T00:00:00Z","public_url":"http://encoder.example.com"},
"assignments":[],"profiles":{},
"stream_youtube_configs":[` + streamYouTubeConfig + `]
}`
	}
	for _, test := range []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "ready fixed relay service config requires identity and lifecycle fields",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"},"active_runtime":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}}`),
			valid: false,
		},
		{
			name:  "not ready fixed relay config may expose incomplete profile for repair",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":false,"readiness_code":"youtube_output_invalid_config","readiness_message":"selected YouTube output is incomplete","youtube_config":{"mode":"live_api_relay_static","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef"}}`),
			valid: true,
		},
		{
			name:  "not ready fixed relay config cannot expose a raw stream key",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":false,"readiness_code":"youtube_output_invalid_config","readiness_message":"selected YouTube output is incomplete","youtube_config":{"mode":"live_api_relay_static","stream_key":"not-a-real-key"}}`),
			valid: false,
		},
		{
			name:  "ready fixed relay config does not require an active runtime before preparation",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime requires broadcast and live stream ids",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","complete_on_stop":true}}`),
			valid: false,
		},
		{
			name:  "prepared fixed relay active runtime carries provider identifiers",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime validates relay binding format",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","relay_binding_id":"relay-binding-1","complete_on_stop":true}}`),
			valid: false,
		},
		{
			name:  "prepared fixed relay active runtime can carry canonical public watch url",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}}`),
			valid: true,
		},
		{
			name:  "prepared fixed relay active runtime cannot expose a raw stream key",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true},"active_runtime":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","broadcast_id":"broadcast-1","live_stream_id":"live-stream-1","complete_on_stop":true,"stream_key":"not-a-real-key"}}`),
			valid: false,
		},
		{
			name:  "fixed relay service config can carry canonical public watch url",
			body:  serviceRuntimePayload(`{"stream_id":"stream-1","assignment_role":"primary","youtube_output_id":"output-1","ready":true,"youtube_config":{"mode":"live_api_relay_static","output_id":"output-1","oauth_account_id":"oauth-1","relay_binding_id":"relay-01234567-89ab-4def-8123-456789abcdef","reusable_live_stream_id":"live-stream-1","complete_on_stop":true,"watch_url":"https://www.youtube.com/watch?v=abc12345"}}`),
			valid: true,
		},
	} {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := serviceValidator.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}

	for _, file := range []struct {
		name    string
		path    string
		markers []string
	}{
		{
			name: "service-runtime-config.schema.json",
			path: filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"),
			markers: []string{
				"live_api_relay_static", `"required": ["rtmp_url"]`, `"required": ["stream_key_secret_name"]`,
			},
		},
		{
			name: "control-api.yaml",
			path: filepath.Join("..", "..", "openapi", "control-api.yaml"),
			markers: []string{
				"live_api_relay_static", "required: [rtmp_url]", "required: [stream_key_secret_name]",
			},
		},
	} {
		body, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range file.markers {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing fixed-relay runtime exclusion marker %q", file.name, want)
			}
		}
	}
}

func TestYouTubeRelayBindingMutationErrorsArePublicContracts(t *testing.T) {
	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{name: "claim check", got: ErrorCodeYouTubeRelayBindingClaimCheckFailed, want: "youtube_relay_binding_claim_check_failed"},
		{name: "release pending", got: ErrorCodeYouTubeRelayBindingReleasePending, want: "youtube_relay_binding_release_pending"},
	} {
		if test.got != test.want {
			t.Fatalf("%s relay binding mutation error = %q, want %q", test.name, test.got, test.want)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(openAPI)
	errorResponseStart := strings.Index(raw, "    ErrorResponse:\n")
	if errorResponseStart < 0 {
		t.Fatal("control-api.yaml is missing ErrorResponse")
	}
	for _, code := range []string{
		"youtube_relay_binding_claim_check_failed",
		"youtube_relay_binding_release_pending",
	} {
		if !strings.Contains(raw, "\n            - "+code+"\n") {
			t.Fatalf("ErrorResponse enum is missing %q", code)
		}
	}

	operationSection := func(path, method string) string {
		t.Helper()
		pathMarker := "  " + path + ":\n"
		pathStart := strings.Index(raw, pathMarker)
		if pathStart < 0 {
			t.Fatalf("control-api.yaml is missing %s", path)
		}
		pathSection := raw[pathStart+len(pathMarker):]
		if end := strings.Index(pathSection, "\n  /"); end >= 0 {
			pathSection = pathSection[:end]
		}
		methodMarker := "    " + method + ":\n"
		methodStart := strings.Index(pathSection, methodMarker)
		if methodStart < 0 {
			t.Fatalf("%s is missing %s", path, method)
		}
		section := pathSection[methodStart+len(methodMarker):]
		for _, nextMethod := range []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"} {
			if end := strings.Index(section, "\n    "+nextMethod+":\n"); end >= 0 {
				section = section[:end]
			}
		}
		return section
	}
	responseSection := func(operation, status string) string {
		t.Helper()
		statusMarker := "        \"" + status + "\":\n"
		statusStart := strings.Index(operation, statusMarker)
		if statusStart < 0 {
			t.Fatalf("operation is missing HTTP %s response", status)
		}
		section := operation[statusStart+len(statusMarker):]
		if end := strings.Index(section, "\n        \""); end >= 0 {
			section = section[:end]
		}
		return section
	}

	for _, test := range []struct {
		path   string
		method string
		status string
		code   string
	}{
		{path: "/youtube/outputs/{id}", method: "put", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/youtube/outputs/{id}", method: "put", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/youtube/outputs/{id}", method: "delete", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/youtube/outputs/{id}", method: "delete", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/streams/{id}", method: "delete", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/streams/{id}", method: "delete", status: "500", code: "youtube_relay_binding_claim_check_failed"},
		{path: "/streams/{id}/settings", method: "put", status: "409", code: "youtube_relay_binding_release_pending"},
		{path: "/streams/{id}/settings", method: "put", status: "500", code: "youtube_relay_binding_claim_check_failed"},
	} {
		response := responseSection(operationSection(test.path, test.method), test.status)
		if !strings.Contains(response, test.code) {
			t.Fatalf("%s %s HTTP %s is missing %q", test.method, test.path, test.status, test.code)
		}
		if !strings.Contains(response, `#/components/schemas/ErrorResponse`) {
			t.Fatalf("%s %s HTTP %s must return ErrorResponse", test.method, test.path, test.status)
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
			"full active-stream playlist",
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
