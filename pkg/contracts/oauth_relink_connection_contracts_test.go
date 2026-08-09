package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestOAuthAccountRelinkConnectionWireMatchesCanonicalSchemas(t *testing.T) {
	request := OAuthAccountConnectionStartRequest{
		OAuthAccountID: "oauth-account-01",
		AccountPurpose: string(OAuthAccountPurposeYouTube),
		RedirectAfter:  "/admin/integrations",
	}
	validateOAuthRelinkContractJSON(t, "oauth-account-connection-start.schema.json", request)

	if err := validateOAuthRelinkContractJSONError(t, "oauth-account-connection-start.schema.json", OAuthAccountConnectionStartRequest{}); err == nil {
		t.Fatal("OAuth relink start schema accepted a request without provider_id or oauth_account_id")
	}

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	startResponse := OAuthAccountConnectionStartResponse{
		Provider: OAuthLoginProvider{
			ID:           "google-01",
			ProviderType: "google",
			Name:         "Google",
			Scopes:       []string{"openid", "email", "profile"},
			RedirectURI:  "https://control.example.com/integrations/oauth-accounts/callback",
		},
		AuthorizationURL: "https://accounts.example.com/authorize?state=one-time-state",
		State:            "one-time-state",
		Nonce:            "nonce-01",
		ExpiresAt:        now.Add(10 * time.Minute),
		AccountPurpose:   OAuthAccountPurposeYouTube,
		Relink:           true,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/youtube.force-ssl",
		},
	}
	validateOAuthRelinkContractJSON(t, "oauth-account-connection-start-response.schema.json", startResponse, "oauth-login-provider.schema.json")

	callbackResponse := OAuthAccount{
		ID:                     "oauth-account-01",
		ProviderID:             "google-01",
		ProviderType:           "google",
		AccountLabel:           "Existing operational account",
		AccountPurpose:         OAuthAccountPurposeYouTube,
		Scopes:                 []string{"openid", "email", "https://www.googleapis.com/auth/youtube.force-ssl"},
		RefreshTokenConfigured: true,
		CreatedAt:              now.Add(-time.Hour),
		UpdatedAt:              now,
	}
	validateOAuthRelinkContractJSON(t, "oauth-account.schema.json", callbackResponse)
}

func TestOAuthAccountRelinkOpenAPITracksStartAndCallbackWireShape(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"/integrations/oauth-accounts/start:",
		"oauth_account_id:",
		"account_purpose:",
		"OAuthAccountConnectionStartResponse",
		"OAuthAccountConnectionCallbackRequest",
		"Connected account relinked while retaining its account ID",
		"account_purpose:",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("control-api.yaml is missing OAuth relink wire marker %q", want)
		}
	}
	startRequest := openAPIComponentSectionForOAuthRelinkContractTest(t, raw, "OAuthAccountConnectionStartRequest")
	for _, want := range []string{"anyOf:", "oauth_account_id:", "account_purpose:", "minLength: 1"} {
		if !strings.Contains(startRequest, want) {
			t.Fatalf("OAuth start request component is missing %q: %s", want, startRequest)
		}
	}
	startResponse := openAPIComponentSectionForOAuthRelinkContractTest(t, raw, "OAuthAccountConnectionStartResponse")
	const canonicalStartResponseSchema = `$ref: "../schemas/oauth-account-connection-start-response.schema.json"`
	if strings.Count(startResponse, canonicalStartResponseSchema) != 1 || strings.Contains(startResponse, "properties:") {
		t.Fatalf("control-api.yaml OAuth start response must use only the strict canonical schema %q: %s", canonicalStartResponseSchema, startResponse)
	}

	callback := openAPIPathSectionForOAuthRelinkContractTest(t, raw, "/integrations/oauth-accounts/callback:")
	for _, want := range []string{
		`"200":`,
		"Connected account relinked while retaining its account ID",
		`"201":`,
		"Connected account created",
		"#/components/schemas/OAuthAccount",
	} {
		if !strings.Contains(callback, want) {
			t.Fatalf("callback contract is missing %q: %s", want, callback)
		}
	}

	for schemaName, wants := range map[string][]string{
		"oauth-account.schema.json":                           {"account_purpose", "scopes"},
		"oauth-account-connection-start.schema.json":          {"oauth_account_id", "account_purpose"},
		"oauth-account-connection-start-response.schema.json": {"account_purpose", "relink", "scopes"},
	} {
		schema, err := os.ReadFile(filepath.Join("..", "..", "schemas", schemaName))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(schema), want) {
				t.Fatalf("%s is missing %q", schemaName, want)
			}
		}
	}
}

func validateOAuthRelinkContractJSON(t *testing.T, schemaName string, value any, dependencies ...string) {
	t.Helper()
	if err := validateOAuthRelinkContractJSONError(t, schemaName, value, dependencies...); err != nil {
		t.Fatalf("%s rejected Go wire shape: %v", schemaName, err)
	}
}

func validateOAuthRelinkContractJSONError(t *testing.T, schemaName string, value any, dependencies ...string) error {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		return err
	}
	compiler := compileOAuthRelinkContractJSONSchema(t, schemaName, dependencies...)
	return compiler.Validate(document)
}

// compileOAuthRelinkContractJSONSchema registers each schema using both its
// repository-relative name and its canonical $id. OAuth response schemas use
// canonical IDs for reference resolution, while their OpenAPI components use
// repository-relative external references.
func compileOAuthRelinkContractJSONSchema(t *testing.T, name string, dependencies ...string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	resources := append([]string{name}, dependencies...)
	for _, resourceName := range resources {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", resourceName))
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", resourceName, err)
		}
		if err := compiler.AddResource(resourceName, document); err != nil {
			t.Fatalf("%s: %v", resourceName, err)
		}

		var metadata struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(body, &metadata); err != nil {
			t.Fatalf("%s metadata: %v", resourceName, err)
		}
		if metadata.ID != "" && metadata.ID != resourceName {
			if err := compiler.AddResource(metadata.ID, document); err != nil {
				t.Fatalf("%s canonical ID: %v", resourceName, err)
			}
		}
	}
	compiled, err := compiler.Compile(name)
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func openAPIPathSectionForOAuthRelinkContractTest(t *testing.T, raw, path string) string {
	t.Helper()
	start := strings.Index(raw, "  "+path+"\n")
	if start < 0 {
		t.Fatalf("OpenAPI is missing %s", path)
	}
	section := raw[start:]
	if end := strings.Index(section[len("  "+path+"\n"):], "\n  /"); end >= 0 {
		section = section[:len("  "+path+"\n")+end]
	}
	return section
}

func openAPIComponentSectionForOAuthRelinkContractTest(t *testing.T, raw, component string) string {
	t.Helper()
	start := strings.Index(raw, "    "+component+":\n")
	if start < 0 {
		t.Fatalf("OpenAPI is missing %s component", component)
	}
	lines := strings.Split(raw[start:], "\n")
	for index, line := range lines[1:] {
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
			return strings.Join(lines[:index+1], "\n")
		}
	}
	return strings.Join(lines, "\n")
}
