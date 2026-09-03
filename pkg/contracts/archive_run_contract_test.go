package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestEncoderArchiveRunRequestSchemasAreV2Only(t *testing.T) {
	tests := []struct {
		name       string
		schemaFile string
		body       string
		wantValid  bool
	}{
		{
			name:       "start without archive run",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","name":"Live","rtmp_url":"rtmps://example.invalid/live"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped start",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","rtmp_url":"rtmps://example.invalid/live","started_at":"2026-08-18T05:06:29.123456789Z"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped start requires time",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","rtmp_url":"rtmps://example.invalid/live"}`,
			wantValid:  false,
		},
		{
			name:       "unsafe run id is rejected",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run..01","name":"Live","rtmp_url":"rtmps://example.invalid/live","started_at":"2026-08-18T05:06:29Z"}`,
			wantValid:  false,
		},
		{
			name:       "runless package is rejected",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","name":"Live"}`,
			wantValid:  false,
		},
		{
			name:       "run scoped package",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","started_at":"2026-08-18T05:06:29Z"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped package requires time",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live"}`,
			wantValid:  false,
		},
		{
			name:       "stream response carries current archive run",
			schemaFile: "stream-job.schema.json",
			body:       `{"id":"stream-01","name":"Live","status":"completed","archive_run_id":"run-01","archive_started_at":"2026-08-18T05:06:29Z","archive_reported_at":"2026-08-18T05:07:29Z"}`,
			wantValid:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dependencies := []string(nil)
			if test.schemaFile == "encoder-start-stream-request.schema.json" {
				dependencies = append(dependencies, "youtube-runtime-config.schema.json", visualCatalogSchema)
			}
			schema := compileContractJSONSchema(t, test.schemaFile, dependencies...)
			var value any
			if err := json.Unmarshal([]byte(test.body), &value); err != nil {
				t.Fatal(err)
			}
			err := schema.Validate(value)
			if test.wantValid && err != nil {
				t.Fatalf("expected valid request: %v", err)
			}
			if !test.wantValid && err == nil {
				t.Fatal("expected invalid request")
			}
		})
	}
}

func TestV2MigrationProofContractFixturesAndNegativeMutants(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "v2-migration")
	schemaBody, err := os.ReadFile(filepath.Join(root, "migration-proof.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBody))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("migration-proof.schema.json", document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("migration-proof.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	fixtureBody, err := os.ReadFile(filepath.Join(root, "contracts-migration-fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture any
	if err := json.Unmarshal(fixtureBody, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(fixture); err != nil {
		t.Fatalf("migration proof fixture does not satisfy its versioned contract: %v", err)
	}
	if err := validateV2MigrationFixtureSemantics(fixture); err != nil {
		t.Fatal(err)
	}

	documentMap := fixture.(map[string]any)
	rows := documentMap["rows"].([]any)
	gotIDs := make([]string, 0, len(rows))
	for _, raw := range rows {
		row := raw.(map[string]any)
		gotIDs = append(gotIDs, row["inventory_id"].(string))
	}
	wantIDs := []string{"DEP-CON-0001", "DEP-CON-0003", "DEP-CON-0005", "DEP-CON-0010", "DEP-CON-0012"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("persistent contract migration denominator = %v, want %v", gotIDs, wantIDs)
	}

	mutants := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing backup", mutate: func(doc map[string]any) { delete(firstV2MigrationRow(doc), "backup") }},
		{name: "restore did not pass", mutate: func(doc map[string]any) { firstV2MigrationRow(doc)["restore"].(map[string]any)["status"] = "FAILED" }},
		{name: "orphan remains", mutate: func(doc map[string]any) { firstV2MigrationRow(doc)["orphan_count"] = float64(1) }},
		{name: "unexpected empty denominator", mutate: func(doc map[string]any) {
			firstV2MigrationRow(doc)["pre_counts"].(map[string]any)["source_count"] = float64(0)
		}},
		{name: "physical deletion started", mutate: func(doc map[string]any) { doc["physical_eol_deletion_started"] = true }},
		{name: "production mutation", mutate: func(doc map[string]any) { doc["production_mutation"] = true }},
		{name: "unsupported strategy", mutate: func(doc map[string]any) { firstV2MigrationRow(doc)["strategy"] = "copy maybe" }},
	}
	for _, mutant := range mutants {
		t.Run(mutant.name, func(t *testing.T) {
			copy := cloneV2MigrationFixture(t, fixture)
			mutant.mutate(copy)
			if err := schema.Validate(copy); err == nil {
				t.Fatal("negative migration proof mutant unexpectedly validated")
			}
		})
	}

	t.Run("post count self-reference or mismatch", func(t *testing.T) {
		copy := cloneV2MigrationFixture(t, fixture)
		firstV2MigrationRow(copy)["post_counts"].(map[string]any)["represented_count"] = float64(2)
		if err := schema.Validate(copy); err != nil {
			t.Fatalf("cross-field mutant must reach the independent semantic checker: %v", err)
		}
		if err := validateV2MigrationFixtureSemantics(copy); err == nil {
			t.Fatal("count mismatch unexpectedly passed the independent semantic checker")
		}
	})
}

func firstV2MigrationRow(document map[string]any) map[string]any {
	return document["rows"].([]any)[0].(map[string]any)
}

func cloneV2MigrationFixture(t *testing.T, value any) map[string]any {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(body, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func validateV2MigrationFixtureSemantics(value any) error {
	document, ok := value.(map[string]any)
	if !ok {
		return errors.New("migration fixture root must be an object")
	}
	rows, ok := document["rows"].([]any)
	if !ok || len(rows) == 0 {
		return errors.New("migration fixture rows must be non-empty")
	}
	seen := map[string]struct{}{}
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			return errors.New("migration fixture row must be an object")
		}
		id, _ := row["inventory_id"].(string)
		if _, duplicate := seen[id]; duplicate {
			return errors.New("migration fixture inventory IDs must be unique")
		}
		seen[id] = struct{}{}
		pre, preOK := row["pre_counts"].(map[string]any)
		post, postOK := row["post_counts"].(map[string]any)
		if !preOK || !postOK || pre["source_count"] != pre["represented_count"] ||
			pre["source_count"] != post["source_count"] ||
			pre["source_count"] != post["represented_count"] {
			return errors.New("migration fixture counts must derive from and preserve pre-state")
		}
	}
	lower := strings.ToLower(string(mustMarshalV2MigrationFixture(value)))
	for _, forbidden := range []string{"raw_password", "raw_token", "ciphertext", "nonce"} {
		if strings.Contains(lower, forbidden) {
			return errors.New("migration fixture exposes secret-bearing material")
		}
	}
	return nil
}

func mustMarshalV2MigrationFixture(value any) []byte {
	body, _ := json.Marshal(value)
	return body
}
