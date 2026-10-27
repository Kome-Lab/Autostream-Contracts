package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPanelSettingsRevisionAcceptsOnlyCanonicalSettingsAuthority(t *testing.T) {
	schema := compileContractJSONSchema(t, "stream-settings-revision.schema.json")
	for _, test := range []struct {
		value string
		valid bool
	}{
		{"unconfigured", true}, {"2030-01-01T01:02:03Z", true}, {"2030-01-01T01:02:03.123456Z", true},
		{"", false}, {"2030-01-01T01:02:03+00:00", false}, {"2030-01-01T01:02:03.100000Z", false}, {"2030-01-01T01:02:03.1234567Z", false}, {"0001-01-01T00:00:00Z", false},
	} {
		if valid := schema.Validate(test.value) == nil; valid != test.valid {
			t.Fatalf("revision %q valid=%t want=%t", test.value, valid, test.valid)
		}
	}
	expected := "unconfigured"
	request := PanelEncoderRuntimeSettingsWriteRequest{ExpectedSettingsRevision: &expected, EncoderAudioGainDB: 3.5, OverlayProfileID: ""}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err = json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	runtime := compileContractJSONSchema(t, "panel-encoder-runtime-settings-write.schema.json", "stream-settings-revision.schema.json")
	if err = runtime.Validate(document); err != nil {
		t.Fatal(err)
	}
	downstream, err := json.Marshal(EncoderRuntimeSettingsRequest{EncoderAudioGainDB: 3.5, OverlayProfileID: ""})
	if err != nil || strings.Contains(string(downstream), "expected_settings_revision") {
		t.Fatal("Panel revision leaked into the downstream Encoder wire")
	}
}

func TestPanelCreateAndCandidateFilenameBoundsPreserveFullName(t *testing.T) {
	schema := compileContractJSONSchema(t, "stream-create-v2.schema.json", "stream-visual-write.schema.json", "stream-cover-candidate.schema.json")
	candidate := map[string]any{"name": strings.Repeat("長", 251) + ".png", "source": "upload", "asset_id": "asset-one", "variant_id": "variant-one"}
	visual := map[string]any{"expected_revision": float64(0), "background_mode": "default", "header_title_mode": "default", "discord_target": map[string]any{"mode": "manual", "guild_id": "1001", "text_channel_id": "1002", "voice_channel_id": "1003"}, "cover_source": "upload", "cover_asset_id": "asset-one", "cover_variant_id": "variant-one", "cover_candidates": []any{candidate}}
	request := map[string]any{"name": strings.Repeat("配", 255), "encoder_service_id": "encoder-one", "worker_service_id": "worker-one", "discord_config_id": "selected-auth", "auto_start_trigger": "discord_voice_join", "visual_settings": visual}
	if err := schema.Validate(request); err != nil {
		t.Fatal(err)
	}
	candidate["name"] = strings.Repeat("長", 252) + ".png"
	if schema.Validate(request) == nil {
		t.Fatal("256-character candidate name accepted")
	}
	candidate["name"] = "full.png"
	request["name"] = strings.Repeat("配", 256)
	if schema.Validate(request) == nil {
		t.Fatal("256-character stream name accepted")
	}
	request["name"] = "slot"
	request["expected_settings_revision"] = "unconfigured"
	if schema.Validate(request) == nil {
		t.Fatal("edit-only revision accepted by create")
	}
	delete(request, "expected_settings_revision")
	delete(request, "encoder_service_id")
	if schema.Validate(request) == nil {
		t.Fatal("create without required primary Encoder accepted")
	}
}

func TestOwnerBoundUploadInspectionSchemaKeepsUnknownDistinctFromEmpty(t *testing.T) {
	schema := compileContractJSONSchema(t, "media-upload-operation-response.schema.json", "media-upload-session.schema.json", "panel-media-asset.schema.json", "panel-media-variant.schema.json")
	operation := "11111111-1111-4111-8111-111111111111"
	response := map[string]any{"operation_id": operation, "session": map[string]any{"id": "session-one", "owner_type": "upload_draft", "expires_at": "2030-01-01T00:00:00Z", "created_at": "2029-01-01T00:00:00Z", "updated_at": "2029-01-01T00:00:00Z"}, "assets": []any{}}
	if err := schema.Validate(response); err != nil {
		t.Fatal(err)
	}
	asset := map[string]any{"id": "asset-one", "owner_type": "upload_draft", "owner_id": "session-one", "upload_session_id": "session-one", "client_operation_id": operation, "usage_type": "video_cover", "sha256": strings.Repeat("0", 64), "media_type": "image/png", "byte_size": float64(100), "width": float64(16), "height": float64(9), "opaque": true, "processor_revision": float64(1), "created_at": "2029-01-01T00:00:00Z", "updated_at": "2029-01-01T00:00:00Z"}
	response["assets"] = []any{map[string]any{"asset": asset, "variants": []any{}}}
	if err := schema.Validate(response); err != nil {
		t.Fatal(err)
	}
	asset["client_operation_id"] = "NOT-A-UUID"
	if schema.Validate(response) == nil {
		t.Fatal("invalid operation UUID accepted")
	}
	asset["client_operation_id"] = operation
	asset["owner_user_id"] = "private-user"
	if schema.Validate(response) == nil {
		t.Fatal("private owner identity accepted in public metadata")
	}
}

func TestResourceWriteRevisionKeepsOriginalBaselineAndLegacyOmission(t *testing.T) {
	const stamp = "2030-01-01T01:02:03.123456Z"
	for _, test := range []struct {
		name string
		body map[string]any
	}{
		{"profile-write.schema.json", map[string]any{"name": "Encoder", "config": map[string]any{"width": float64(1920)}}},
		{"discord-config-write.schema.json", map[string]any{"name": "Selected Bot", "service_id": "selected-bot"}},
		{"youtube-output-write.schema.json", map[string]any{"name": "YouTube", "mode": "stream_key"}},
		{"drive-destination-write.schema.json", map[string]any{"name": "Drive", "auth_mode": "oauth2", "oauth_account_id": "account-one"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := compileContractJSONSchema(t, test.name, "resource-updated-at.schema.json")
			if err := schema.Validate(test.body); err != nil {
				t.Fatalf("legacy omission rejected: %v", err)
			}
			test.body["expected_updated_at"] = stamp
			if err := schema.Validate(test.body); err != nil {
				t.Fatalf("original canonical baseline rejected: %v", err)
			}
			for _, invalid := range []any{"", "unconfigured", "2030-01-01T01:02:03+00:00", "2030-01-01T01:02:03.100000Z", "2030-01-01T01:02:03.1234567Z", "0001-01-01T00:00:00Z", nil} {
				test.body["expected_updated_at"] = invalid
				if schema.Validate(test.body) == nil {
					t.Fatalf("invalid resource baseline accepted: %v", invalid)
				}
			}
		})
	}
	for _, value := range []any{
		ProfileWriteRequest{Name: "Encoder", Config: map[string]any{"width": 1920}, ExpectedUpdatedAt: pointerResourceStamp(stamp)},
		DiscordConfigWriteRequest{Name: "Selected Bot", ServiceID: "selected-bot", ExpectedUpdatedAt: pointerResourceStamp(stamp)},
		YouTubeOutputWriteRequest{Name: "YouTube", Mode: "stream_key", ExpectedUpdatedAt: pointerResourceStamp(stamp)},
		DriveDestinationWriteRequest{Name: "Drive", AuthMode: "oauth2", OAuthAccountID: "account-one", ExpectedUpdatedAt: pointerResourceStamp(stamp)},
	} {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(body, &document); err != nil || document["expected_updated_at"] != stamp {
			t.Fatal("wire DTO did not preserve the original baseline string")
		}
	}
}

func pointerResourceStamp(value string) *string { return &value }
