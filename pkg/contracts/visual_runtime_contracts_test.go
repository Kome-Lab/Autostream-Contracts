package contracts

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const visualCatalogSchema = "discord-bot-start-job-request.schema.json"

func TestWorkerStartSceneAppearanceUsesCanonicalReadySnapshot(t *testing.T) {
	schema := compileContractJSONSchema(t, "worker-start-job-request.schema.json", visualCatalogSchema)
	legacy := map[string]any{"stream_id": "stream-1", "stream_name": "Legacy"}
	assertV2SchemaFixture(t, schema, legacy, true)

	withDefault := cloneV2Fixture(t, legacy)
	withDefault["scene_appearance"] = validSceneAppearanceFixture(false)
	assertV2SchemaFixture(t, schema, withDefault, true)

	withImage := cloneV2Fixture(t, legacy)
	withImage["scene_appearance"] = validSceneAppearanceFixture(true)
	assertV2SchemaFixture(t, schema, withImage, true)

	notReady := cloneV2Fixture(t, withImage)
	notReadyAppearance := notReady["scene_appearance"].(map[string]any)
	notReadyAppearance["readiness"] = "not_ready"
	notReadyAppearance["error"] = map[string]any{"code": "media_asset_timeout"}
	assertV2SchemaFixture(t, schema, notReady, false)

	unknown := cloneV2Fixture(t, withImage)
	unknown["scene_appearance"].(map[string]any)["asset_url"] = "https://forbidden.example.test/background.png"
	assertV2SchemaFixture(t, schema, unknown, false)

	source := readContractSource(t, "schemas", "worker-start-job-request.schema.json")
	if !strings.Contains(source, visualCatalogSchema+"#/$defs/sceneAppearance") {
		t.Fatal("Worker start must reference the canonical visual catalog sceneAppearance definition")
	}
}

func TestVisualRuntimeAuthorityMetadataAndSafeErrors(t *testing.T) {
	document := readContractJSONMap(t, "schemas", visualCatalogSchema)
	defs := requireTestMap(t, document, "$defs")
	scene := requireTestMap(t, defs, "sceneAppearance")
	sceneProperties := requireTestMap(t, scene, "properties")
	assertAuthorityMetadata(t, requireTestMap(t, sceneProperties, "generation"), map[string]any{
		"x-autostream-owner":   "control_panel",
		"x-autostream-epoch":   "video_cover_job_generation",
		"x-autostream-advance": "every_stream_start",
	})
	assertAuthorityMetadata(t, requireTestMap(t, sceneProperties, "revision"), map[string]any{
		"x-autostream-source": "stream_visual_settings.revision",
	})
	asset := requireTestMap(t, defs, "mediaAssetDescriptor")
	assetProperties := requireTestMap(t, asset, "properties")
	assertAuthorityMetadata(t, requireTestMap(t, assetProperties, "revision"), map[string]any{
		"x-autostream-source":       "media_asset_variants.processor_revision",
		"x-autostream-immutability": "processed_variant",
	})
	runtimeState := requireTestMap(t, defs, "videoCoverRuntimeState")
	runtimeProperties := requireTestMap(t, runtimeState, "properties")
	assertAuthorityMetadata(t, requireTestMap(t, runtimeProperties, "job_generation"), map[string]any{
		"x-autostream-owner": "control_panel",
		"x-autostream-epoch": "video_cover_job_generation",
	})
	assertAuthorityMetadata(t, requireTestMap(t, runtimeProperties, "generation"), map[string]any{
		"x-autostream-owner":     "encoder_recorder",
		"x-autostream-scope":     "job_generation",
		"x-autostream-monotonic": true,
	})

	safeError := compileV2SchemaFragment(t, visualCatalogSchema, "safeError")
	for _, code := range []string{
		"media_asset_unauthorized",
		"media_asset_not_found",
		"media_asset_hash_mismatch",
		"media_asset_dimension_mismatch",
		"media_asset_timeout",
		"stale_cover_generation",
		"idempotency_conflict",
		"cover_apply_ambiguous",
	} {
		t.Run(code, func(t *testing.T) {
			assertV2SchemaFixture(t, safeError, map[string]any{"code": code}, true)
		})
	}
}

func TestVisualSafeErrorCodeSchemaGoParity(t *testing.T) {
	document := readContractJSONMap(t, "schemas", visualCatalogSchema)
	defs := requireTestMap(t, document, "$defs")
	safeError := requireTestMap(t, defs, "safeError")
	properties := requireTestMap(t, safeError, "properties")
	code := requireTestMap(t, properties, "code")
	schemaValues, ok := code["enum"].([]any)
	if !ok {
		t.Fatalf("safeError code enum=%T, want array", code["enum"])
	}
	schemaCodes := make(map[string]struct{}, len(schemaValues))
	for _, value := range schemaValues {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("safeError code=%T, want string", value)
		}
		schemaCodes[text] = struct{}{}
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), "types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	goCodes := make(map[string]struct{})
	for _, declaration := range parsed.Decls {
		constants, ok := declaration.(*ast.GenDecl)
		if !ok || constants.Tok != token.CONST {
			continue
		}
		for _, specification := range constants.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeName, ok := value.Type.(*ast.Ident)
			if !ok || typeName.Name != "VisualSafeErrorCode" {
				continue
			}
			for _, expression := range value.Values {
				literal, ok := expression.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Fatalf("VisualSafeErrorCode constant must be a string literal: %T", expression)
				}
				decoded, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				goCodes[decoded] = struct{}{}
			}
		}
	}
	if len(goCodes) != len(schemaCodes) {
		t.Fatalf("VisualSafeErrorCode count=%d, schema safeError enum count=%d; go=%v schema=%v", len(goCodes), len(schemaCodes), goCodes, schemaCodes)
	}
	for code := range schemaCodes {
		if _, ok := goCodes[code]; !ok {
			t.Errorf("schema safeError code %q has no typed VisualSafeErrorCode constant", code)
		}
	}
}

func TestVisualPipelineRequiresAudioEncoderAndMuxContinuityWitnesses(t *testing.T) {
	schema := compileV2SchemaFragment(t, visualCatalogSchema, "visualPipelineInvariant")
	pipeline := v2VisualPipelineFixture()
	assertV2SchemaFixture(t, schema, pipeline, true)

	for _, field := range []string{"audio_encoder_restart", "audio_mux_restart"} {
		t.Run("missing_"+field, func(t *testing.T) {
			missing := cloneV2Fixture(t, pipeline)
			delete(missing["audio_continuity"].(map[string]any), field)
			assertV2SchemaFixture(t, schema, missing, false)
		})
		t.Run("nonzero_"+field, func(t *testing.T) {
			nonzero := cloneV2Fixture(t, pipeline)
			nonzero["audio_continuity"].(map[string]any)[field] = 1
			assertV2SchemaFixture(t, schema, nonzero, false)
		})
	}
}

func TestEncoderVideoCoverSchemasFenceAndWitnessGraphApplication(t *testing.T) {
	startSchema := compileContractJSONSchema(t, "encoder-video-cover-start-snapshot.schema.json", visualCatalogSchema)
	applySchema := compileContractJSONSchema(t, "encoder-video-cover-apply-request.schema.json", visualCatalogSchema)
	responseSchema := compileContractJSONSchema(t, "encoder-video-cover-apply-response.schema.json", visualCatalogSchema)
	runtimeSchema := compileContractJSONSchema(t, "encoder-video-cover-runtime-state.schema.json", visualCatalogSchema)

	start := map[string]any{
		"job_generation": 9, "revision": 4, "active": true,
		"idempotency_key": "start-cover-9", "cover_asset": validCoverAssetFixture(),
	}
	assertV2SchemaFixture(t, startSchema, start, true)
	inactiveStart := cloneV2Fixture(t, start)
	inactiveStart["active"] = false
	delete(inactiveStart, "cover_asset")
	assertV2SchemaFixture(t, startSchema, inactiveStart, true)
	inactiveWithAsset := cloneV2Fixture(t, start)
	inactiveWithAsset["active"] = false
	assertV2SchemaFixture(t, startSchema, inactiveWithAsset, false)
	startWithoutAsset := cloneV2Fixture(t, start)
	delete(startWithoutAsset, "cover_asset")
	assertV2SchemaFixture(t, startSchema, startWithoutAsset, false)

	show := map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "expected_generation": 3,
		"revision": 4, "active": true, "idempotency_key": "show-4",
		"cover_asset": validCoverAssetFixture(),
	}
	assertV2SchemaFixture(t, applySchema, show, true)
	showWithoutAsset := cloneV2Fixture(t, show)
	delete(showWithoutAsset, "cover_asset")
	assertV2SchemaFixture(t, applySchema, showWithoutAsset, false)
	showMutatesWatermark := cloneV2Fixture(t, show)
	showMutatesWatermark["watermark"] = map[string]any{"enabled": false}
	assertV2SchemaFixture(t, applySchema, showMutatesWatermark, false)

	hide := map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "expected_generation": 3,
		"revision": 5, "active": false, "idempotency_key": "hide-5", "hide_confirmed": true,
	}
	assertV2SchemaFixture(t, applySchema, hide, true)
	hideWithoutConfirmation := cloneV2Fixture(t, hide)
	delete(hideWithoutConfirmation, "hide_confirmed")
	assertV2SchemaFixture(t, applySchema, hideWithoutConfirmation, false)
	hideWithAsset := cloneV2Fixture(t, hide)
	hideWithAsset["cover_asset"] = validCoverAssetFixture()
	assertV2SchemaFixture(t, applySchema, hideWithAsset, false)

	runtime := validVideoCoverRuntimeFixture()
	assertV2SchemaFixture(t, runtimeSchema, runtime, true)
	withoutWitness := cloneV2Fixture(t, runtime)
	delete(withoutWitness, "applied_witness")
	assertV2SchemaFixture(t, runtimeSchema, withoutWitness, false)
	beforeGraphApply := cloneV2Fixture(t, runtime)
	beforeGraphApply["applied_witness"].(map[string]any)["graph_applied"] = false
	assertV2SchemaFixture(t, runtimeSchema, beforeGraphApply, false)
	wrongPipeline := cloneV2Fixture(t, runtime)
	wrongPipeline["pipeline"].(map[string]any)["layers"] = []any{
		"base_or_worker_scene", "watermark", "video_cover", "video_encode", "tee_live_archive_preview",
	}
	assertV2SchemaFixture(t, runtimeSchema, wrongPipeline, false)
	linkedLayerMutation := cloneV2Fixture(t, runtime)
	linkedLayerMutation["pipeline"].(map[string]any)["cover_watermark_independent"] = false
	assertV2SchemaFixture(t, runtimeSchema, linkedLayerMutation, false)

	applied := map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 4,
		"actual_generation": 3,
		"accepted":          true, "rejected": false, "applied": true, "outcome": "applied",
		"actual": runtime,
	}
	assertV2SchemaFixture(t, responseSchema, applied, true)
	appliedWithoutGraphWitness := cloneV2Fixture(t, applied)
	delete(appliedWithoutGraphWitness["actual"].(map[string]any), "applied_witness")
	assertV2SchemaFixture(t, responseSchema, appliedWithoutGraphWitness, false)
	inconsistentApplied := cloneV2Fixture(t, applied)
	inconsistentApplied["applied"] = false
	assertV2SchemaFixture(t, responseSchema, inconsistentApplied, false)

	rejected := map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 4,
		"actual_generation": 3,
		"accepted":          false, "rejected": true, "applied": false, "outcome": "rejected",
		"actual": runtime, "error": map[string]any{"code": "stale_cover_generation"},
	}
	assertV2SchemaFixture(t, responseSchema, rejected, true)
	rejectedWithoutError := cloneV2Fixture(t, rejected)
	delete(rejectedWithoutError, "error")
	assertV2SchemaFixture(t, responseSchema, rejectedWithoutError, false)

	ambiguousActual := validAmbiguousVideoCoverRuntimeFixture()
	ambiguous := map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 5,
		"actual_generation": 4,
		"accepted":          true, "rejected": false, "applied": false, "outcome": "ambiguous",
		"actual": ambiguousActual, "error": map[string]any{"code": "cover_apply_ambiguous"},
	}
	assertV2SchemaFixture(t, responseSchema, ambiguous, true)
	ambiguousClaimsApplied := cloneV2Fixture(t, ambiguous)
	ambiguousClaimsApplied["applied"] = true
	assertV2SchemaFixture(t, responseSchema, ambiguousClaimsApplied, false)
}

func TestEncoderStartCarriesDesiredCoverSnapshotAndOpenAPIExposesReconcile(t *testing.T) {
	startSchema := compileContractJSONSchema(t, "encoder-start-stream-request.schema.json", "youtube-runtime-config.schema.json", visualCatalogSchema)
	legacy := map[string]any{"stream_id": "stream-1", "name": "Legacy", "rtmp_url": ""}
	assertV2SchemaFixture(t, startSchema, legacy, true)
	withCover := cloneV2Fixture(t, legacy)
	withCover["video_cover_start"] = map[string]any{
		"job_generation": 9, "revision": 4, "active": true,
		"idempotency_key": "start-cover-9", "cover_asset": validCoverAssetFixture(),
	}
	assertV2SchemaFixture(t, startSchema, withCover, true)
	withoutCover := cloneV2Fixture(t, legacy)
	withoutCover["video_cover_start"] = map[string]any{
		"job_generation": 9, "revision": 5, "active": false,
		"idempotency_key": "start-no-cover-9",
	}
	assertV2SchemaFixture(t, startSchema, withoutCover, true)
	inactiveWithAsset := cloneV2Fixture(t, withoutCover)
	inactiveWithAsset["video_cover_start"].(map[string]any)["cover_asset"] = validCoverAssetFixture()
	assertV2SchemaFixture(t, startSchema, inactiveWithAsset, false)

	startSource := readContractSource(t, "schemas", "encoder-start-stream-request.schema.json")
	for _, marker := range []string{"New Control Panel producers send this snapshot for both active and inactive desired state", "legacy callers"} {
		if !strings.Contains(startSource, marker) {
			t.Fatalf("Encoder start schema does not document desired snapshot compatibility (%q)", marker)
		}
	}

	capabilities := compileContractJSONSchema(t, "encoder-output-relay-capabilities.schema.json")
	assertV2SchemaFixture(t, capabilities, map[string]any{
		"live_video_cover_v1": true,
		"scene_appearance_v1": true,
	}, true)
	assertV2SchemaFixture(t, capabilities, map[string]any{"live_video_cover_v1": false}, false)
	assertV2SchemaFixture(t, capabilities, map[string]any{"scene_appearance_v1": "yes"}, false)

	openAPI := readContractSource(t, "openapi", "encoder-recorder-api.yaml")
	for _, marker := range []string{
		"/streams/{id}/video-cover-state:",
		"operationId: getVideoCoverRuntimeState",
		"operationId: applyVideoCoverState",
		"../schemas/encoder-video-cover-apply-request.schema.json",
		"../schemas/encoder-video-cover-apply-response.schema.json",
		"../schemas/encoder-video-cover-runtime-state.schema.json",
	} {
		if !strings.Contains(openAPI, marker) {
			t.Fatalf("Encoder OpenAPI is missing %q", marker)
		}
	}
	controlAPI := readContractSource(t, "openapi", "control-api.yaml")
	for _, marker := range []string{"live_video_cover_v1:", "scene_appearance_v1:"} {
		if !strings.Contains(controlAPI, marker) {
			t.Fatalf("Control API capability vocabulary is missing %q", marker)
		}
	}
}

func TestEncoderVideoCoverOpenAPIStatusSchemasAreDisjoint(t *testing.T) {
	fixtures := map[string]map[string]any{
		"applied":   validAppliedVideoCoverResponseFixture(),
		"ambiguous": validAmbiguousVideoCoverResponseFixture(),
		"conflict":  validRejectedVideoCoverResponseFixture("stale_cover_revision"),
		"graph":     validRejectedVideoCoverResponseFixture("cover_graph_unavailable"),
	}
	allowed := map[string]string{
		"200": "applied",
		"202": "ambiguous",
		"409": "conflict",
		"502": "graph",
	}
	for status, allowedFixture := range allowed {
		t.Run(status, func(t *testing.T) {
			fragment := "/paths/~1streams~1{id}~1video-cover-state/put/responses/" + status + "/content/application~1json/schema"
			schema := compileNormalizedOpenAPISchema(t, "encoder-recorder-api.json", fragment)
			for name, fixture := range fixtures {
				assertV2SchemaFixture(t, schema, fixture, name == allowedFixture)
			}
		})
	}
}

func TestVisualRuntimeGoTypesPreserveOptionalCompatibility(t *testing.T) {
	if CapabilitySceneAppearanceV1 != "scene_appearance_v1" || CapabilityLiveVideoCoverV1 != "live_video_cover_v1" {
		t.Fatalf("visual capabilities drifted: scene=%q cover=%q", CapabilitySceneAppearanceV1, CapabilityLiveVideoCoverV1)
	}
	legacy, err := json.Marshal(WorkerStartJobRequest{StreamID: "stream-1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacy), "scene_appearance") {
		t.Fatalf("legacy Worker start unexpectedly emitted scene appearance: %s", legacy)
	}
	worker := WorkerStartJobRequest{
		StreamID: "stream-1",
		SceneAppearance: &SceneAppearance{
			Generation: 9, Revision: 4, Capability: CapabilitySceneAppearanceV1,
			Readiness: VisualReadinessReady, BackgroundMode: "default", HeaderTitleMode: "custom", CustomTitle: "Title",
		},
	}
	payload, err := json.Marshal(worker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"scene_appearance":{"generation":9,"revision":4,"capability":"scene_appearance_v1"`) {
		t.Fatalf("Worker scene appearance wire shape drifted: %s", payload)
	}

	encoder := EncoderStartStreamRequest{StreamID: "stream-1", Name: "Stream", RTMPURL: "", VideoCoverStart: &VideoCoverStartSnapshot{
		JobGeneration: 9, Revision: 4, Active: true, IdempotencyKey: "start-cover-9", CoverAsset: validCoverAssetGoFixture(),
	}}
	payload, err = json.Marshal(encoder)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"video_cover_start":{"job_generation":9,"revision":4,"active":true`) {
		t.Fatalf("Encoder start cover snapshot wire shape drifted: %s", payload)
	}

	encoder.VideoCoverStart = &VideoCoverStartSnapshot{
		JobGeneration: 9, Revision: 5, Active: false, IdempotencyKey: "start-no-cover-9",
	}
	payload, err = json.Marshal(encoder)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"video_cover_start":{"job_generation":9,"revision":5,"active":false`) ||
		strings.Contains(string(payload), `"cover_asset"`) {
		t.Fatalf("inactive desired cover snapshot wire shape drifted: %s", payload)
	}
}

func validSceneAppearanceFixture(withImage bool) map[string]any {
	appearance := map[string]any{
		"generation": 9, "revision": 4, "capability": "scene_appearance_v1", "readiness": "ready",
		"background_mode": "default", "header_title_mode": "custom", "custom_title": "Title",
	}
	if withImage {
		appearance["background_mode"] = "image"
		appearance["background"] = validSceneAssetFixture()
	}
	return appearance
}

func validSceneAssetFixture() map[string]any {
	return map[string]any{
		"asset_id": "asset-1", "variant_id": "variant-1", "usage": "scene_background", "media_type": "image/png",
		"width": 1920, "height": 1080, "byte_size": 1024, "pixel_count": 2073600,
		"animated": false, "sha256": strings.Repeat("a", 64), "revision": 1, "readiness": "ready",
	}
}

func validCoverAssetFixture() map[string]any {
	asset := validSceneAssetFixture()
	asset["usage"] = "video_cover"
	asset["aspect_ratio_error_ppm"] = 0
	asset["opaque"] = true
	return asset
}

func validVideoCoverRuntimeFixture() map[string]any {
	pipeline := v2VisualPipelineFixture()
	cover := map[string]any{"enabled": true, "revision": 4, "variant_id": "variant-1"}
	watermark := map[string]any{"enabled": true, "revision": 2, "variant_id": "watermark-1"}
	return map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "generation": 3,
		"capability": "live_video_cover_v1", "readiness": "ready",
		"desired": map[string]any{"active": true, "revision": 4, "source": "upload", "variant_id": "variant-1"},
		"applied": map[string]any{"state": "known", "active": true, "revision": 4, "variant_id": "variant-1"},
		"cover":   cover, "cover_asset": validCoverAssetFixture(), "watermark": watermark,
		"pipeline": pipeline, "no_automatic_resend": true,
		"applied_witness": map[string]any{
			"graph_applied": true, "generation": 3, "revision": 4, "active": true,
			"cover": cover, "watermark": watermark, "pipeline": pipeline,
		},
	}
}

func validAmbiguousVideoCoverRuntimeFixture() map[string]any {
	state := validVideoCoverRuntimeFixture()
	state["generation"] = 4
	state["readiness"] = "unknown"
	state["desired"] = map[string]any{"active": false, "revision": 5, "source": "none"}
	state["applied"] = map[string]any{"state": "unknown"}
	state["cover"] = map[string]any{"enabled": false, "revision": 5}
	delete(state, "cover_asset")
	delete(state, "applied_witness")
	state["last_good_applied"] = map[string]any{"state": "known", "active": true, "revision": 4, "variant_id": "variant-1"}
	state["error"] = map[string]any{"code": "cover_apply_ambiguous"}
	return state
}

func validAppliedVideoCoverResponseFixture() map[string]any {
	return map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 4,
		"actual_generation": 3,
		"accepted":          true, "rejected": false, "applied": true, "outcome": "applied",
		"actual": validVideoCoverRuntimeFixture(),
	}
}

func validAmbiguousVideoCoverResponseFixture() map[string]any {
	return map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 5,
		"actual_generation": 4,
		"accepted":          true, "rejected": false, "applied": false, "outcome": "ambiguous",
		"actual": validAmbiguousVideoCoverRuntimeFixture(), "error": map[string]any{"code": "cover_apply_ambiguous"},
	}
}

func validRejectedVideoCoverResponseFixture(code string) map[string]any {
	actual := validAmbiguousVideoCoverRuntimeFixture()
	actual["error"] = map[string]any{"code": code}
	return map[string]any{
		"stream_id": "stream-1", "job_generation": 9, "requested_revision": 5,
		"actual_generation": 4,
		"accepted":          false, "rejected": true, "applied": false, "outcome": "rejected",
		"actual": actual, "error": map[string]any{"code": code},
	}
}

func validCoverAssetGoFixture() *MediaAssetDescriptor {
	opaque := true
	aspect := 0
	return &MediaAssetDescriptor{
		AssetID: "asset-1", VariantID: "variant-1", Usage: "video_cover", MediaType: "image/png",
		Width: 1920, Height: 1080, ByteSize: 1024, PixelCount: 2073600, Animated: false,
		SHA256: strings.Repeat("a", 64), Revision: 1, Readiness: VisualReadinessReady,
		AspectRatioErrorPPM: &aspect, Opaque: &opaque,
	}
}

func assertAuthorityMetadata(t *testing.T, actual map[string]any, expected map[string]any) {
	t.Helper()
	for key, want := range expected {
		if got := actual[key]; got != want {
			t.Fatalf("authority metadata %s=%v, want %v", key, got, want)
		}
	}
}

func requireTestMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	result, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s=%T, want object", key, value[key])
	}
	return result
}

func readContractJSONMap(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal([]byte(readContractSource(t, parts...)), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func readContractSource(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", ".."}, parts...)...)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
