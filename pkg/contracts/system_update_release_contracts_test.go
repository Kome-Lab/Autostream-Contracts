package contracts

import (
	"bytes"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseManifestSchemasSeparateHostAndDockerChannels(t *testing.T) {
	rawManifest, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"release-manifest.json.sha256", "64 lowercase hexadecimal characters", "two spaces", "trailing newline"} {
		if !strings.Contains(string(rawManifest), marker) {
			t.Fatalf("release manifest sidecar contract is missing %q", marker)
		}
	}
	manifest := readContractSchema(t, "release-manifest.schema.json")
	requireContractFields(t, manifest.Required, "schema_version", "release_id", "channel", "published_at", "components")
	if contractSliceHas(manifest.Required, "minimum_agent_version") || contractSliceHas(manifest.Required, "protocol_major") {
		t.Fatal("release compatibility fields must be required by channel, not across both channels")
	}
	if !contractSliceHas(manifest.Properties["channel"].Enum, "host") ||
		!contractSliceHas(manifest.Properties["channel"].Enum, "docker") {
		t.Fatal("release manifest must support host and docker channels")
	}
	if manifest.Properties["schema_version"].Const != float64(2) {
		t.Fatalf("release manifest schema version must be 2, got %#v", manifest.Properties["schema_version"].Const)
	}
	for _, removed := range []string{"bundle_version", "generated_at"} {
		if _, ok := manifest.Properties[removed]; ok {
			t.Fatalf("release manifest retained removed alias %q", removed)
		}
	}
	if len(manifest.AllOf) != 2 || manifest.AllOf[0].Then == nil || manifest.AllOf[1].Then == nil {
		t.Fatalf("release manifest channel conditions changed: %#v", manifest.AllOf)
	}

	docker := manifest.AllOf[0].Then
	requireContractFields(t, docker.Required, "protocol_major")
	if manifest.Properties["protocol_major"].Const != float64(2) || docker.Not == nil || !contractSliceHas(docker.Not.Required, "minimum_agent_version") {
		t.Fatal("Docker releases must use protocol-major 2 and forbid version-based compatibility")
	}
	requireContractFields(t, manifest.AllOf[1].Then.Required, "minimum_agent_version")
	if manifest.Properties["minimum_agent_version"].Pattern != "^v[0-9]+\\.[0-9]+\\.[0-9]+$" {
		t.Fatal("minimum_agent_version must use the canonical release version format")
	}
	dockerComponents := docker.Properties["components"]
	if dockerComponents.MinItems != 6 || dockerComponents.MaxItems != 6 || dockerComponents.Items == nil || len(dockerComponents.Items.OneOf) != 2 {
		t.Fatalf("docker manifest must contain five images and one independent Updater metadata component: %#v", dockerComponents)
	}
	imageComponent := dockerComponents.Items.OneOf[0]
	updaterComponent := dockerComponents.Items.OneOf[1]
	requireContractFields(t, imageComponent.Required,
		"service", "source_version", "commit", "image", "manifest_digest", "platform_digests",
		"rollback_compatible", "database_schema",
	)
	requireContractFields(t, updaterComponent.Required, "service", "commit", "protocol_major")
	if len(updaterComponent.Required) != 3 || updaterComponent.Properties["service"].Const != "updater" || updaterComponent.Properties["protocol_major"].Const != float64(2) {
		t.Fatal("independent Updater component must contain only source identity and protocol-major metadata")
	}
	if len(dockerComponents.AllOf) != 6 {
		t.Fatalf("docker manifest must contain each known service exactly once: %#v", dockerComponents.AllOf)
	}
	if !strings.Contains(imageComponent.Properties["image"].Pattern, "ghcr\\.io/kome-lab/autostream-docker") {
		t.Fatal("docker images must use the fixed lowercase GHCR namespace")
	}
	if imageComponent.Properties["rollback_compatible"].Const != true {
		t.Fatal("Docker releases must explicitly allow rollback")
	}
	if imageComponent.Not == nil {
		t.Fatal("Docker components must reject host-only fields")
	}
	for _, forbidden := range imageComponent.Not.AnyOf {
		for _, policyField := range []string{"rollback_compatible", "database_schema"} {
			if contractSliceHas(forbidden.Required, policyField) {
				t.Fatalf("Docker schema both requires and forbids %q", policyField)
			}
		}
	}
	if schemas := imageComponent.Properties["database_schema"].Enum; !contractSliceHas(schemas, "none") || !contractSliceHas(schemas, "backward_compatible") || contractSliceHas(schemas, "irreversible") {
		t.Fatalf("Docker database_schema policy is unsafe: %#v", schemas)
	}
	expectedSchemas := []string{"backward_compatible", "none", "none", "backward_compatible", "none"}
	for i, expected := range expectedSchemas {
		if imageComponent.AllOf[i].Then == nil || imageComponent.AllOf[i].Then.Properties["database_schema"].Const != expected {
			t.Fatalf("Docker service policy %d must require database_schema %q", i, expected)
		}
	}
	componentJSON, err := json.Marshal(ReleaseManifestComponent{})
	if err != nil {
		t.Fatal(err)
	}
	var componentWire map[string]any
	if err := json.Unmarshal(componentJSON, &componentWire); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"rollback_compatible", "database_schema"} {
		if _, ok := componentWire[required]; !ok {
			t.Fatalf("ReleaseManifestComponent omits required field %q", required)
		}
	}

	platforms := manifest.Properties["components"].Items.Properties["platform_digests"]
	requireContractFields(t, platforms.Required, "linux/amd64", "linux/arm64")
	if platforms.AdditionalProperties != false {
		t.Fatal("platform_digests must contain exactly amd64 and arm64")
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		if platforms.Properties[platform].Pattern != "^sha256:[a-f0-9]{64}$" {
			t.Fatalf("%s digest is not canonical", platform)
		}
	}

	hostComponents := manifest.AllOf[1].Then.Properties["components"]
	if hostComponents.MinItems != 1 || hostComponents.MaxItems != 1 || hostComponents.Items == nil {
		t.Fatal("a host release manifest must describe exactly its repository component")
	}
	requireContractFields(t, hostComponents.Items.Required,
		"service", "source_version", "commit", "artifacts",
		"rollback_compatible", "database_schema",
	)
	if hostComponents.Items.Properties["rollback_compatible"].Const != true {
		t.Fatal("host releases must explicitly allow rollback")
	}
	hostArtifacts := hostComponents.Items.Properties["artifacts"]
	if hostArtifacts.MinItems != 2 || hostArtifacts.MaxItems != 2 {
		t.Fatal("host releases must contain exactly amd64 and arm64 artifacts")
	}
	artifacts := manifest.Properties["components"].Items.Properties["artifacts"]
	if artifacts.Items == nil || artifacts.Items.Properties["sha256"].Pattern != "^[a-f0-9]{64}$" ||
		artifacts.Items.Properties["size"].Maximum != 268435456 {
		t.Fatal("host artifact checksum or size limit differs from the updater")
	}
}

func TestDockerReleaseManifestGeneratorShapeValidatesAgainstSchema(t *testing.T) {
	// This fixture is generated with autostream-docker's release manifest
	// generator. Its producer-side tests separately pin the exact field set.
	instanceJSON, err := os.ReadFile(filepath.Join("..", "..", "testdata", "release-manifest.docker.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		t.Fatal(err)
	}

	schemaJSON, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	schemaDocument, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err := compiler.AddResource("release-manifest.schema.json", schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("release-manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(instance); err != nil {
		t.Fatalf("generator-shaped Docker manifest violates shared schema: %v", err)
	}

	instanceMap := instance.(map[string]any)
	componentMaps := instanceMap["components"].([]any)
	first := componentMaps[0].(map[string]any)
	first["rollback_compatible"] = false
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted rollback_compatible=false")
	}
	delete(first, "rollback_compatible")
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted Docker component without rollback_compatible")
	}
	first["rollback_compatible"] = true
	delete(first, "database_schema")
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted Docker component without database_schema")
	}
	first["database_schema"] = "none"
	if err := compiled.Validate(instance); err == nil {
		t.Fatal("schema accepted unsafe control-panel database_schema")
	}
}

func readReleaseDescription(t *testing.T, property string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "release-manifest.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Properties[property].Description
}
