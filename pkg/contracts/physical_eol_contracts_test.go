package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func compilePhysicalEOLSchemaFragment(t *testing.T, name, fragment string) *jsonschema.Schema {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return compilePhysicalEOLDocumentFragment(t, name, document, fragment)
}

func compilePhysicalEOLDocumentFragment(t *testing.T, name string, document any, fragment string) *jsonschema.Schema {
	t.Helper()
	stripPhysicalEOLSchemaIDs(document)
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.UseRegexpEngine(compileECMAContractSchemaRegexp)
	if err := compiler.AddResource(name, document); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join("..", "..", "schemas"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		dependency, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		stripPhysicalEOLSchemaIDs(dependency)
		if entry.Name() != name {
			if err := compiler.AddResource(entry.Name(), dependency); err != nil {
				t.Fatalf("register %s: %v", entry.Name(), err)
			}
		}
	}
	schema, err := compiler.Compile(name + "#" + fragment)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func stripPhysicalEOLSchemaIDs(value any) {
	switch typed := value.(type) {
	case map[string]any:
		delete(typed, "$id")
		for _, child := range typed {
			stripPhysicalEOLSchemaIDs(child)
		}
	case []any:
		for _, child := range typed {
			stripPhysicalEOLSchemaIDs(child)
		}
	}
}

func TestPhysicalEOLEncoderRequestsRejectInlineRuntimeFields(t *testing.T) {
	started := time.Date(2026, time.September, 3, 1, 2, 3, 0, time.UTC)
	encoderOpenAPI := readNormalizedOpenAPICharacterization(t, "encoder-recorder-api.json")
	start := EncoderStartStreamRequest{
		StreamID: "stream-1", ArchiveRunID: "run-1", Name: "Stream", StartedAt: started,
		RTMPURL: "rtmps://example.invalid/live", StreamKeySecretName: "assignment-stream-key",
		ArchiveProfileID: "archive-profile-1", YouTubeRuntime: YouTubeRuntimeConfig{Mode: "stream_key"},
	}
	pack := EncoderPackageStreamRequest{StreamID: "stream-1", ArchiveRunID: "run-1", Name: "Stream", StartedAt: started}
	for _, test := range []struct {
		name    string
		value   any
		schema  *jsonschema.Schema
		removed []string
	}{
		{"start", start, compileContractJSONSchema(t, "encoder-start-stream-request.schema.json", "youtube-runtime-config.schema.json", visualCatalogSchema), []string{"stream_key", "archive_config", "base_path"}},
		{"package", pack, compileContractJSONSchema(t, "encoder-package-stream-request.schema.json"), []string{"archive_config", "base_path"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			var valid map[string]any
			if err := json.Unmarshal(body, &valid); err != nil {
				t.Fatal(err)
			}
			assertV2SchemaFixture(t, test.schema, valid, true)
			for _, removed := range test.removed {
				for _, value := range []any{nil, "", map[string]any{"base_path": "synthetic"}} {
					candidate := cloneV2Fixture(t, valid)
					candidate[removed] = value
					assertV2SchemaFixture(t, test.schema, candidate, false)
				}
				for _, field := range reflect.VisibleFields(reflect.TypeOf(test.value)) {
					if strings.Split(field.Tag.Get("json"), ",")[0] == removed {
						t.Fatalf("Go %s DTO still declares removed field %s", test.name, removed)
					}
				}
			}
		})
	}
	startSchema := compilePhysicalEOLDocumentFragment(t, "encoder-start.json", encoderOpenAPI, "/components/schemas/EncoderStartStreamRequest")
	valid := cloneV2Fixture(t, start)
	assertV2SchemaFixture(t, startSchema, valid, true)
	for _, removed := range []string{"stream_key", "archive_config", "base_path"} {
		candidate := cloneV2Fixture(t, valid)
		candidate[removed] = "synthetic-forbidden"
		assertV2SchemaFixture(t, startSchema, candidate, false)
	}
}

func TestPhysicalEOLEncoderCapabilityAliasesFailAcrossOpenAPIComponents(t *testing.T) {
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	components := map[string]func(string) string{
		"ServiceRegistrationRequest": func(capabilities string) string {
			return `{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","public_url":"http://encoder.example.invalid","version":"v2.0.0","capabilities":` + capabilities + `}`
		},
		"Heartbeat": func(capabilities string) string {
			return `{"service_id":"encoder-1","status":"online","capabilities":` + capabilities + `}`
		},
		"ServiceTokenCreateRequest": func(capabilities string) string {
			return `{"service_type":"encoder_recorder","scopes":["service.register"],"capabilities":` + capabilities + `}`
		},
	}
	for component, fixture := range components {
		schema := compilePhysicalEOLDocumentFragment(t, "control-capability-"+component+".json", control, "/components/schemas/"+component)
		for _, capabilities := range []string{
			`{"scene_frames_mjpeg_srt":true,"worker_frame_ingest_mjpeg_srt":true,"scene_appearance_v1":true,"live_video_cover_v1":true,"live_encoder_runtime_settings":true}`,
			`{"future_non_secret_capability":true}`,
		} {
			var value any
			if err := json.Unmarshal([]byte(fixture(capabilities)), &value); err != nil {
				t.Fatal(err)
			}
			assertV2SchemaFixture(t, schema, value, true)
		}
		for _, capabilities := range []string{`{"scene_video_srt":true}`, `{"scene_video_srt":false}`, `{"worker_video_ingest_srt":true}`, `{"worker_video_ingest_srt":false}`} {
			var value any
			if err := json.Unmarshal([]byte(fixture(capabilities)), &value); err != nil {
				t.Fatal(err)
			}
			assertV2SchemaFixture(t, schema, value, false)
		}
	}
}

func TestPhysicalEOLRunScopedArtifactReportMatchesGoAndOpenAPI(t *testing.T) {
	started := time.Date(2026, time.September, 3, 1, 2, 3, 0, time.UTC)
	report := ServiceArtifactReport{
		ServiceID: "encoder-1", StreamID: "stream-1", ArchiveRunID: "run-1", ArchiveStartedAt: &started,
		Artifacts: []StreamArtifact{{Kind: "archive", Name: "final.mp4", RelativePath: "final/stream-1/run-1/final.mp4", SizeBytes: 0}},
	}
	valid := cloneV2Fixture(t, report)
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	schemas := []*jsonschema.Schema{
		compileContractJSONSchema(t, "service-artifact-report.schema.json"),
		compilePhysicalEOLDocumentFragment(t, "control-report.json", control, "/components/schemas/ServiceArtifactReport"),
	}
	for _, schema := range schemas {
		assertV2SchemaFixture(t, schema, valid, true)
		for _, field := range []string{"archive_run_id", "archive_started_at"} {
			candidate := cloneV2Fixture(t, valid)
			delete(candidate, field)
			assertV2SchemaFixture(t, schema, candidate, false)
		}
		runless := cloneV2Fixture(t, valid)
		runless["artifacts"].([]any)[0].(map[string]any)["relative_path"] = "final/stream-1/final.mp4"
		assertV2SchemaFixture(t, schema, runless, false)
	}
}

func TestPhysicalEOLRuntimeArchiveRejectsRemovedPathWithoutClosingExtensions(t *testing.T) {
	valid := map[string]any{
		"drive_destination_id": "drive-1", "archive_profile_id": "profile-1", "auth_mode": "oauth2",
		"folder_id_secret_name": "folder-reference", "refresh_token_secret_name": "refresh-reference",
		"shared_drive": true, "future_non_secret_hint": true,
	}
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	schemas := []*jsonschema.Schema{
		compilePhysicalEOLSchemaFragment(t, "service-runtime-config.schema.json", "/properties/stream_archive_configs/items/properties/archive_config"),
		compilePhysicalEOLDocumentFragment(t, "control-runtime.json", control, "/components/schemas/ServiceRuntimeConfig/properties/stream_archive_configs/items/properties/archive_config"),
	}
	for _, schema := range schemas {
		assertV2SchemaFixture(t, schema, valid, true)
		for _, path := range []any{"", "legacy-retained-path", nil} {
			candidate := cloneV2Fixture(t, valid)
			candidate["base_path"] = path
			assertV2SchemaFixture(t, schema, candidate, false)
		}
	}
}

func TestPhysicalEOLPreservesFiveFourFieldApplicationProbes(t *testing.T) {
	rawSchema := compileV2SchemaFragment(t, "system-update-target.schema.json", "systemUpdateTargetApplicationProbe")
	hostSchema := compileV2SchemaFragment(t, "system-update-host-status.schema.json", "applicationProbe")
	for _, service := range []SystemUpdateTargetType{SystemUpdateTargetControlPanel, SystemUpdateTargetWorker, SystemUpdateTargetEncoderRecorder, SystemUpdateTargetDiscordBot, SystemUpdateTargetObservability} {
		probe := ApplicationRuntimeIdentityProbe{Version: "v2.0.0", ServiceID: "application-1", ServiceType: service, ConfigRevision: 1}
		raw := cloneV2Fixture(t, probe)
		if len(raw) != 4 {
			t.Fatalf("raw application probe gained fields: %v", raw)
		}
		assertV2SchemaFixture(t, rawSchema, raw, true)
		host := cloneV2Fixture(t, SystemUpdateHostApplicationProbe{Status: "ready", ApplicationRuntimeIdentityProbe: probe})
		assertV2SchemaFixture(t, hostSchema, host, true)
		assertV2SchemaFixture(t, rawSchema, host, false)
		assertV2SchemaFixture(t, hostSchema, raw, false)
	}
}

func TestPhysicalEOLSystemUpdateProtocolOmissionFailsClosed(t *testing.T) {
	target := map[string]any{
		"protocol_version": 2, "target_id": "worker-1", "target_type": "worker", "name": "Worker",
		"host_id": "host-1", "updater_id": "updater-1", "capabilities": []any{"host.update"},
		"desired_revision": 2, "applied_revision": 0, "fence": 1,
		"updater_health":    map[string]any{"status": "ready", "revision": 2},
		"application_probe": map[string]any{"version": "v2.0.0", "service_id": "worker-1", "service_type": "worker", "config_revision": 1},
		"update_available":  false, "updater_online": true, "eligible": false, "busy": false,
	}
	for _, test := range []struct {
		name  string
		value any
	}{
		{"system-update-create-request.schema.json", SystemUpdateCreateRequest{ProtocolVersion: 2, TargetID: "worker-1", Strategy: SystemUpdateWhenIdle, IdempotencyKey: "update-1", DesiredRevision: 2, Fence: 1, RequiredCapability: UpdaterCapabilityUpdate}},
		{"system-update-job.schema.json", systemUpdateV2JobFixture(time.Date(2026, time.September, 3, 1, 2, 3, 0, time.UTC))},
		{"system-update-target.schema.json", target},
		{"system-update-pull-ownership-activate-request.schema.json", pullOwnershipActivationRequestFixture()},
		{"system-update-pull-ownership-activate-response.schema.json", pullOwnershipActivationResponseFixture()},
		{"system-update-pull-ownership-deactivate-request.schema.json", pullOwnershipDeactivationRequestFixture()},
		{"system-update-pull-ownership-deactivate-response.schema.json", pullOwnershipDeactivationResponseFixture()},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := compileContractJSONSchema(t, test.name)
			valid := cloneV2Fixture(t, test.value)
			assertV2SchemaFixture(t, schema, valid, true)
			for _, version := range []any{nil, 0, 1, 3, "2"} {
				candidate := cloneV2Fixture(t, valid)
				candidate["protocol_version"] = version
				assertV2SchemaFixture(t, schema, candidate, false)
			}
			delete(valid, "protocol_version")
			assertV2SchemaFixture(t, schema, valid, false)
		})
	}
}

func TestPhysicalEOLNotificationCreateRequiresGlobalEmailRelayInSchemaAndOpenAPI(t *testing.T) {
	observability := readNormalizedOpenAPICharacterization(t, "observability-api.json")
	schemas := []*jsonschema.Schema{
		compileContractJSONSchema(t, "notification-channel-create.schema.json", "notification-channel-write.schema.json"),
		compilePhysicalEOLDocumentFragment(t, "observability-notification.json", observability, "/components/schemas/NotificationChannelCreateRequest"),
	}
	for _, schema := range schemas {
		valid := map[string]any{"name": "Email", "type": "email", "enabled": true, "uses_global_smtp": true, "email_recipients": []any{"ops@example.invalid"}}
		assertV2SchemaFixture(t, schema, valid, true)
		for _, field := range []string{"uses_global_smtp", "email_recipients"} {
			candidate := cloneV2Fixture(t, valid)
			delete(candidate, field)
			assertV2SchemaFixture(t, schema, candidate, false)
		}
		for _, field := range []string{"smtp_host", "smtp_port", "smtp_tls", "smtp_from", "smtp_username", "smtp_password"} {
			candidate := cloneV2Fixture(t, valid)
			candidate[field] = "synthetic-forbidden"
			assertV2SchemaFixture(t, schema, candidate, false)
		}
		falseSelector := cloneV2Fixture(t, valid)
		falseSelector["uses_global_smtp"] = false
		assertV2SchemaFixture(t, schema, falseSelector, false)
		webhook := map[string]any{"name": "Webhook", "type": "generic", "enabled": true, "webhook_url": "https://example.invalid/hook"}
		assertV2SchemaFixture(t, schema, webhook, true)
		webhook["uses_global_smtp"] = true
		assertV2SchemaFixture(t, schema, webhook, false)
	}
}

func TestPhysicalEOLAuthorizeUpdaterIdentityMatchesSchemaAndOpenAPI(t *testing.T) {
	control := readNormalizedOpenAPICharacterization(t, "control-api.json")
	schemas := []*jsonschema.Schema{
		compileContractJSONSchema(t, "update-agent-authorize-request.schema.json"),
		compilePhysicalEOLDocumentFragment(t, "control-authorize.json", control, "/components/schemas/UpdateAgentAuthorizeRequest"),
	}
	valid := map[string]any{
		"updater_id": "updater-1", "host_id": "host-1", "lease_token": strings.Repeat("a", 43),
		"lease_generation": 1, "fence": 2, "target_id": "worker-1", "target_version": "v2.0.0", "deployment_mode": "systemd",
	}
	for _, schema := range schemas {
		assertV2SchemaFixture(t, schema, valid, true)
		maximum := cloneV2Fixture(t, valid)
		maximum["updater_id"] = strings.Repeat("a", 128)
		assertV2SchemaFixture(t, schema, maximum, true)
		for _, id := range []string{"", strings.Repeat("a", 129), "updater 1", " updater-1", "updater-1\n"} {
			candidate := cloneV2Fixture(t, valid)
			candidate["updater_id"] = id
			assertV2SchemaFixture(t, schema, candidate, false)
		}
		legacy := cloneV2Fixture(t, valid)
		delete(legacy, "updater_id")
		legacy["service_id"] = "updater-1"
		assertV2SchemaFixture(t, schema, legacy, false)
	}
}
