package contracts

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDiscordBotStartV2AcceptsBoundedCaptionTuningWithResolvedTarget(t *testing.T) {
	schema := compileContractJSONSchema(t, visualCatalogSchema)
	validV2 := map[string]any{
		"schema_version": 2,
		"stream_id":      "stream-1",
		"job_generation": 7,
		"discord_target": map[string]any{
			"revision": 9,
			"resolved": map[string]any{
				"guild_id":         "123456789012345678",
				"text_channel_id":  "223456789012345678",
				"voice_channel_id": "323456789012345678",
			},
		},
		"caption_audio_flush_ms":          10,
		"caption_audio_max_batch_packets": 100,
		"unresolved_ssrc_buffer_ms":       5000,
	}
	assertV2SchemaFixture(t, schema, validV2, true)

	omitted := cloneV2Fixture(t, validV2)
	delete(omitted, "caption_audio_flush_ms")
	delete(omitted, "caption_audio_max_batch_packets")
	delete(omitted, "unresolved_ssrc_buffer_ms")
	assertV2SchemaFixture(t, schema, omitted, true)

	legacy := map[string]any{
		"stream_id": "stream-legacy", "job_generation": 6,
		"guild_id": "123456789012345678", "voice_channel_id": "323456789012345678",
		"caption_audio_flush_ms": 100, "caption_audio_max_batch_packets": 5,
		"unresolved_ssrc_buffer_ms": 1000,
	}
	assertV2SchemaFixture(t, schema, legacy, true)

	flatTargetLeak := cloneV2Fixture(t, validV2)
	flatTargetLeak["guild_id"] = "123456789012345678"
	assertV2SchemaFixture(t, schema, flatTargetLeak, false)

	for _, test := range []struct {
		name  string
		field string
		value int
	}{
		{name: "flush below minimum", field: "caption_audio_flush_ms", value: 9},
		{name: "flush above maximum", field: "caption_audio_flush_ms", value: 1001},
		{name: "batch below minimum", field: "caption_audio_max_batch_packets", value: 0},
		{name: "batch above maximum", field: "caption_audio_max_batch_packets", value: 101},
		{name: "SSRC buffer below minimum", field: "unresolved_ssrc_buffer_ms", value: -1},
		{name: "SSRC buffer above maximum", field: "unresolved_ssrc_buffer_ms", value: 5001},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneV2Fixture(t, validV2)
			invalid[test.field] = test.value
			assertV2SchemaFixture(t, schema, invalid, false)
		})
	}
}

func TestDiscordVoiceJobLegacySourceAndWireCompatibility(t *testing.T) {
	schema := compileContractJSONSchema(t, visualCatalogSchema)
	legacy := DiscordVoiceJob{
		StreamID: "stream-legacy", JobGeneration: 6,
		GuildID: "123456789012345678", VoiceChannelID: "323456789012345678", TextChannelID: "223456789012345678",
		EncoderAudioURL: "https://encoder.example.test/audio", CaptionAudioURL: "https://worker.example.test/captions",
		CaptionAudioToken: "caption-token", CaptionAudioFlushMS: 100, CaptionAudioMaxBatchPackets: 5,
		UnresolvedSSRCBufferMS: 1000, StreamIngestToken: "ingest-token",
		WorkerEventsURL: "https://worker.example.test/events", WorkerEventsToken: "events-token",
	}
	var legacyAlias DiscordBotStartJobRequest = legacy
	if legacyAlias.UnresolvedSSRCBufferMS != 1000 {
		t.Fatalf("legacy alias changed field semantics: %#v", legacyAlias)
	}
	field, ok := reflect.TypeOf(DiscordVoiceJob{}).FieldByName("UnresolvedSSRCBufferMS")
	if !ok || field.Type.Kind() != reflect.Int || field.Tag.Get("json") != "unresolved_ssrc_buffer_ms,omitempty" {
		t.Fatalf("legacy UnresolvedSSRCBufferMS source contract drifted: %#v", field)
	}
	for name, wantTag := range map[string]string{"GuildID": "guild_id", "VoiceChannelID": "voice_channel_id"} {
		field, ok := reflect.TypeOf(DiscordVoiceJob{}).FieldByName(name)
		if !ok || field.Tag.Get("json") != wantTag {
			t.Fatalf("legacy %s tag=%q, want %q", name, field.Tag.Get("json"), wantTag)
		}
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"stream_id":"stream-legacy","job_generation":6,"guild_id":"123456789012345678","voice_channel_id":"323456789012345678","text_channel_id":"223456789012345678","encoder_audio_url":"https://encoder.example.test/audio","caption_audio_url":"https://worker.example.test/captions","caption_audio_token":"caption-token","caption_audio_flush_ms":100,"caption_audio_max_batch_packets":5,"unresolved_ssrc_buffer_ms":1000,"stream_ingest_token":"ingest-token","worker_events_url":"https://worker.example.test/events","worker_events_token":"events-token"}`
	if string(payload) != want {
		t.Fatalf("legacy wire changed:\n got %s\nwant %s", payload, want)
	}
	assertJSONPayloadAgainstSchema(t, schema, payload, true)
}

func TestDiscordBotStartJobV2DTORepresentsResolvedSnapshot(t *testing.T) {
	schema := compileContractJSONSchema(t, visualCatalogSchema)
	v2 := DiscordBotStartJobV2Request{
		SchemaVersion: 2,
		StreamID:      "stream-1",
		JobGeneration: 7,
		DiscordTarget: DiscordTargetSnapshot{
			Revision: 9,
			Resolved: ResolvedDiscordTarget{
				GuildID: "123456789012345678", TextChannelID: "223456789012345678", VoiceChannelID: "323456789012345678",
			},
		},
		CaptionAudioFlushMS: 10, CaptionAudioMaxBatchPackets: 100, UnresolvedSSRCBufferMS: discordBufferMS(5000),
	}
	payload, err := json.Marshal(v2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"guild_id":""`) || strings.Contains(string(payload), `"voice_channel_id":""`) {
		t.Fatalf("v2 DTO leaked empty legacy flat target fields: %s", payload)
	}
	assertJSONPayloadAgainstSchema(t, schema, payload, true)
}

func TestDiscordBotStartJobV2PreservesExplicitZeroUnresolvedSSRCBuffer(t *testing.T) {
	explicitZero := DiscordBotStartJobV2Request{
		SchemaVersion: 2, StreamID: "stream-1", JobGeneration: 7,
		DiscordTarget: DiscordTargetSnapshot{
			Revision: 9,
			Resolved: ResolvedDiscordTarget{
				GuildID: "123456789012345678", TextChannelID: "223456789012345678", VoiceChannelID: "323456789012345678",
			},
		},
		UnresolvedSSRCBufferMS: discordBufferMS(0),
	}
	payload, err := json.Marshal(explicitZero)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"unresolved_ssrc_buffer_ms":0`) {
		t.Fatalf("explicit zero unresolved SSRC buffer was omitted: %s", payload)
	}
	explicitZero.UnresolvedSSRCBufferMS = nil
	payload, err = json.Marshal(explicitZero)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"unresolved_ssrc_buffer_ms"`) {
		t.Fatalf("nil unresolved SSRC buffer was serialized: %s", payload)
	}
}

func TestDiscordResolvedTargetV2CapabilityNegotiatesMixedFleetFailClosed(t *testing.T) {
	if CapabilityDiscordResolvedTargetV2 != "discord_resolved_target_v2" {
		t.Fatalf("Discord resolved-target capability drifted: %q", CapabilityDiscordResolvedTargetV2)
	}

	capabilitySchema := compileContractJSONSchema(t, "encoder-output-relay-capabilities.schema.json")
	for _, test := range []struct {
		name         string
		capabilities map[string]any
		schemaValid  bool
		strictV2     bool
	}{
		{name: "legacy Bot has no advertisement", capabilities: map[string]any{}, schemaValid: true, strictV2: false},
		{name: "new Bot advertises actual support", capabilities: map[string]any{"discord_resolved_target_v2": true}, schemaValid: true, strictV2: true},
		{name: "false is not an advertisement", capabilities: map[string]any{"discord_resolved_target_v2": false}, schemaValid: false, strictV2: false},
		{name: "string is not an advertisement", capabilities: map[string]any{"discord_resolved_target_v2": "true"}, schemaValid: false, strictV2: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertV2SchemaFixture(t, capabilitySchema, test.capabilities, test.schemaValid)
			if got := SupportsDiscordResolvedTargetV2(test.capabilities); got != test.strictV2 {
				t.Fatalf("SupportsDiscordResolvedTargetV2()=%v, want %v", got, test.strictV2)
			}
		})
	}

	capabilityPayload, err := json.Marshal(EncoderOutputRelayCapabilities{DiscordResolvedTargetV2: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(capabilityPayload) != `{"discord_resolved_target_v2":true}` {
		t.Fatalf("typed Discord capability wire drifted: %s", capabilityPayload)
	}

	legacyPayload, err := json.Marshal(DiscordBotStartJobRequest{
		StreamID: "stream-legacy", JobGeneration: 6,
		GuildID: "123456789012345678", VoiceChannelID: "323456789012345678",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(legacyPayload), "schema_version") || strings.Contains(string(legacyPayload), "discord_target") {
		t.Fatalf("legacy mixed-fleet payload contains v2-only fields: %s", legacyPayload)
	}

	for _, source := range []struct {
		path    []string
		markers []string
	}{
		{
			path: []string{"schemas", "discord-bot-start-job-request.schema.json"},
			markers: []string{
				"discord_resolved_target_v2",
				"version string alone",
				"legacy flat target",
			},
		},
		{
			path: []string{"openapi", "discord-bot-api.yaml"},
			markers: []string{
				"discord_resolved_target_v2: true",
				"must omit every v2-only field",
			},
		},
		{
			path: []string{"openapi", "control-api.yaml"},
			markers: []string{
				"discord_resolved_target_v2:",
				"actual Discord Bot support",
			},
		},
	} {
		document := readContractSource(t, source.path...)
		for _, marker := range source.markers {
			if !strings.Contains(document, marker) {
				t.Fatalf("%s is missing mixed-fleet authority %q", strings.Join(source.path, "/"), marker)
			}
		}
	}
}

func discordBufferMS(value int) *int {
	return &value
}

func TestDiscordBotSchemaDescribesBundle8LegacyCompatibility(t *testing.T) {
	document := readContractJSONMap(t, "schemas", visualCatalogSchema)
	description, ok := document["description"].(string)
	if !ok {
		t.Fatalf("description=%T, want string", document["description"])
	}
	for _, marker := range []string{"legacy flat target fields remain accepted", "Execution Bundle 8"} {
		if !strings.Contains(description, marker) {
			t.Fatalf("schema description does not disclose Bundle 8 compatibility branch (%q): %s", marker, description)
		}
	}
}

func assertJSONPayloadAgainstSchema(t *testing.T, schema interface{ Validate(any) error }, payload []byte, wantValid bool) {
	t.Helper()
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	err := schema.Validate(value)
	if wantValid && err != nil {
		t.Fatalf("payload rejected: %v\n%s", err, payload)
	}
	if !wantValid && err == nil {
		t.Fatalf("invalid payload accepted: %s", payload)
	}
}
