package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing MFA role policy marker %q", want)
		}
	}
	requireControlOpenAPIMFAPolicy(t)
}

func TestOAuthLoginSchemasDocumentProviderAndSecretBoundaries(t *testing.T) {
	files := []string{
		"oauth-login-provider.schema.json",
		"oauth-login-start-request.schema.json",
		"oauth-login-start-response.schema.json",
		"oauth-login-callback-request.schema.json",
		"oauth-user-link.schema.json",
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
		"/auth/oauth/callback:",
		"/integrations/oauth-accounts/start:",
		"/integrations/oauth-accounts/callback:",
	} {
		if !strings.Contains(openapiRaw, want) {
			t.Fatalf("control-api.yaml is missing OAuth user link safety marker %q", want)
		}
	}
	for _, removed := range []string{"OAuthUserLinkWriteRequest", "OAuthAccountWriteRequest", "manual_oauth_link_disabled", "manual_oauth_account_create_disabled"} {
		if strings.Contains(openapiRaw, removed) {
			t.Fatalf("control-api.yaml retained removed manual OAuth surface %q", removed)
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

func TestIntegrationWriteSchemasDocumentSecretBoundaries(t *testing.T) {
	tests := []struct {
		file  string
		wants []string
	}{
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
	for _, removed := range []string{"oauth-account-write.schema.json", "oauth-user-link-write.schema.json"} {
		if _, err := os.Stat(filepath.Join("..", "..", "schemas", removed)); !os.IsNotExist(err) {
			t.Fatalf("removed manual OAuth schema %s still exists: %v", removed, err)
		}
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
