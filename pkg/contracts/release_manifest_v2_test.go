package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dockerManifestV2Fixture(t *testing.T) map[string]any {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "release-manifest.docker.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestDockerManifestV2RequiresExactComponentSourcesAndIndependentUpdater(t *testing.T) {
	schema := compileContractJSONSchema(t, "release-manifest.schema.json")
	valid := dockerManifestV2Fixture(t)
	assertV2SchemaFixture(t, schema, valid, true)
	for _, field := range []string{"schema_version", "release_id", "published_at", "protocol_major"} {
		candidate := cloneV2Fixture(t, valid)
		delete(candidate, field)
		assertV2SchemaFixture(t, schema, candidate, false)
	}
	for field, value := range map[string]any{
		"schema_version": 1, "protocol_major": 1, "minimum_agent_version": "v1.0.0",
		"bundle_version": "v1.3.0", "generated_at": "2026-07-18T07:08:09Z",
	} {
		candidate := cloneV2Fixture(t, valid)
		candidate[field] = value
		assertV2SchemaFixture(t, schema, candidate, false)
	}
	for index := range valid["components"].([]any) {
		missing := cloneV2Fixture(t, valid)
		delete(missing["components"].([]any)[index].(map[string]any), "commit")
		assertV2SchemaFixture(t, schema, missing, false)
		invalid := cloneV2Fixture(t, valid)
		invalid["components"].([]any)[index].(map[string]any)["commit"] = "not-an-exact-source-sha"
		assertV2SchemaFixture(t, schema, invalid, false)
	}
	missingUpdater := cloneV2Fixture(t, valid)
	missingUpdater["components"] = missingUpdater["components"].([]any)[:5]
	assertV2SchemaFixture(t, schema, missingUpdater, false)
	duplicate := cloneV2Fixture(t, valid)
	components := duplicate["components"].([]any)
	components[4] = components[5]
	assertV2SchemaFixture(t, schema, duplicate, false)
	for _, field := range []string{"source_version", "image", "artifacts", "rollback_compatible", "database_schema", "manifest_digest", "platform_digests"} {
		candidate := cloneV2Fixture(t, valid)
		candidate["components"].([]any)[5].(map[string]any)[field] = nil
		assertV2SchemaFixture(t, schema, candidate, false)
	}
	for _, value := range []any{nil, 0, 1, 3} {
		candidate := cloneV2Fixture(t, valid)
		candidate["components"].([]any)[5].(map[string]any)["protocol_major"] = value
		assertV2SchemaFixture(t, schema, candidate, false)
	}
	missingProtocol := cloneV2Fixture(t, valid)
	delete(missingProtocol["components"].([]any)[5].(map[string]any), "protocol_major")
	assertV2SchemaFixture(t, schema, missingProtocol, false)
}

func TestReleaseManifestUpdaterMetadataGoEncodingIsExact(t *testing.T) {
	component := ReleaseManifestComponent{Service: "updater", Commit: strings.Repeat("a", 40), ProtocolMajor: 2}
	body, err := json.Marshal(component)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"service":"updater","commit":"` + strings.Repeat("a", 40) + `","protocol_major":2}`
	if string(body) != want {
		t.Fatalf("Updater metadata encoding=%s, want %s", body, want)
	}
	var parsed ReleaseManifestComponent
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Service != component.Service || parsed.Commit != component.Commit || parsed.ProtocolMajor != component.ProtocolMajor {
		t.Fatalf("Updater metadata round trip changed source binding: %v", err)
	}
	for name, mutate := range map[string]func(*ReleaseManifestComponent){
		"wrong major":        func(value *ReleaseManifestComponent) { value.ProtocolMajor = 1 },
		"short source":       func(value *ReleaseManifestComponent) { value.Commit = "abc" },
		"non-hex source":     func(value *ReleaseManifestComponent) { value.Commit = strings.Repeat("z", 40) },
		"co-release version": func(value *ReleaseManifestComponent) { value.SourceVersion = "v2.0.0" },
		"fictitious image":   func(value *ReleaseManifestComponent) { value.Image = "synthetic-image" },
		"host artifacts":     func(value *ReleaseManifestComponent) { value.Artifacts = []ReleaseArtifact{} },
		"platform digests":   func(value *ReleaseManifestComponent) { value.PlatformDigests = map[string]string{} },
		"rollback policy":    func(value *ReleaseManifestComponent) { value.RollbackCompatible = true },
		"database policy":    func(value *ReleaseManifestComponent) { value.DatabaseSchema = "none" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := component
			mutate(&candidate)
			if _, err := json.Marshal(candidate); err == nil {
				t.Fatal("invalid Updater metadata serialized")
			}
		})
	}
}

func TestReleaseManifestV2PreservesHostArtifactContracts(t *testing.T) {
	schema := compileContractJSONSchema(t, "release-manifest.schema.json")
	for service, database := range map[string]string{
		"control-panel": "backward_compatible", "discord-bot": "none", "encoder-recorder": "none",
		"observability": "backward_compatible", "worker": "none",
	} {
		artifacts := []ReleaseArtifact{}
		for _, arch := range []string{"amd64", "arm64"} {
			artifacts = append(artifacts, ReleaseArtifact{OS: "linux", Arch: arch, Name: fmt.Sprintf("autostream-%s_v2.0.0_linux_%s.tar.gz", service, arch), Size: 1024, SHA256: strings.Repeat("b", 64)})
		}
		manifest := ReleaseManifest{
			SchemaVersion: 2, ReleaseID: "v2.0.0", Channel: ReleaseChannelHost,
			PublishedAt: time.Date(2026, time.September, 3, 1, 2, 3, 0, time.UTC), MinimumAgentVersion: "v1.0.0",
			Components: []ReleaseManifestComponent{{Service: service, SourceVersion: "v2.0.0", Commit: strings.Repeat("a", 40), Artifacts: artifacts, RollbackCompatible: true, DatabaseSchema: database}},
		}
		valid := cloneV2Fixture(t, manifest)
		assertV2SchemaFixture(t, schema, valid, true)
		if _, exists := valid["protocol_major"]; exists {
			t.Fatal("host encoding gained Docker-only protocol metadata")
		}
		withoutVersion := cloneV2Fixture(t, valid)
		delete(withoutVersion, "minimum_agent_version")
		assertV2SchemaFixture(t, schema, withoutVersion, false)
		mixedChannel := cloneV2Fixture(t, valid)
		mixedChannel["protocol_major"] = 2
		assertV2SchemaFixture(t, schema, mixedChannel, false)
		withoutArtifacts := cloneV2Fixture(t, valid)
		delete(withoutArtifacts["components"].([]any)[0].(map[string]any), "artifacts")
		assertV2SchemaFixture(t, schema, withoutArtifacts, false)
		wrongArchitecture := cloneV2Fixture(t, valid)
		wrongArchitecture["components"].([]any)[0].(map[string]any)["artifacts"].([]any)[0].(map[string]any)["arch"] = "arm64"
		assertV2SchemaFixture(t, schema, wrongArchitecture, false)
	}
}
