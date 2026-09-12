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

func stringSliceContainsForSchemaTest(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
