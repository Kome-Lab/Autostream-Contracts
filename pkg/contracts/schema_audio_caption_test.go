package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerEventSchemaDocumentsDiscordChatOverlay(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "worker-event.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"overlay.discord_chat",
		"message_id",
		"author_id",
		"display_name",
		"content",
		"avatar_url",
		"is_bot",
		"text_channel_id",
		"created_at",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("worker-event.schema.json is missing Discord chat overlay marker %q", want)
		}
	}
}

func TestWorkerEventSchemaValidatesDiscordChatPayloadCompatibility(t *testing.T) {
	schema := compileContractJSONSchema(t, "worker-event.schema.json")

	tests := []struct {
		name    string
		payload map[string]any
		valid   bool
	}{
		{
			name: "canonical payload",
			payload: map[string]any{
				"message_id":      "message-01",
				"author_id":       "user-01",
				"display_name":    "Alice",
				"content":         "hello",
				"avatar_url":      "https://cdn.example.test/avatar.png",
				"is_bot":          false,
				"text_channel_id": "channel-01",
				"created_at":      "2026-08-12T07:30:00Z",
			},
			valid: true,
		},
		{
			name: "removed aliases are rejected",
			payload: map[string]any{
				"author_id": "user-01",
				"user_id":   "user-01",
				"content":   "hello",
				"text":      "hello",
			},
			valid: false,
		},
		{
			name:    "chat payload requires canonical fields",
			payload: map[string]any{},
			valid:   false,
		},
		{
			name: "canonical bot marker must be boolean",
			payload: map[string]any{
				"author_id": "bot-01",
				"content":   "hello",
				"is_bot":    "false",
			},
			valid: false,
		},
		{
			name: "canonical author id must be string",
			payload: map[string]any{
				"author_id": 123,
				"content":   "hello",
			},
			valid: false,
		},
		{
			name: "canonical content must be string",
			payload: map[string]any{
				"author_id": "user-01",
				"content":   123,
			},
			valid: false,
		},
		{
			name: "canonical avatar url must be string",
			payload: map[string]any{
				"author_id":  "user-01",
				"content":    "hello",
				"avatar_url": true,
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := map[string]any{
				"id":        "event-01",
				"stream_id": "stream-01",
				"type":      "overlay.discord_chat",
				"payload":   test.payload,
				"timestamp": "2026-08-12T07:30:00Z",
			}
			err := schema.Validate(event)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, validation error=%v", test.valid, err)
			}
		})
	}

	t.Run("other worker events keep opaque payload compatibility", func(t *testing.T) {
		event := map[string]any{
			"id":        "event-02",
			"stream_id": "stream-01",
			"type":      "overlay.participants",
			"payload": map[string]any{
				"producer_specific": []any{1.0, true, "value"},
			},
			"timestamp": "2026-08-12T07:30:00Z",
		}
		if err := schema.Validate(event); err != nil {
			t.Fatalf("non-chat worker event compatibility changed: %v", err)
		}
	})
}

func TestDeepgramCaptionAndSessionRefreshContracts(t *testing.T) {
	files := map[string][]string{
		"caption-profile-config.schema.json": {
			"CaptionProfileConfig", "deepgram", "nova-3", "deepgram_api_key", "manual caption input is not a runtime provider",
		},
		"discord-bot-start-job-request.schema.json": {
			"caption_audio_url", "caption_audio_token", "Short-lived stream-scoped token",
		},
		"discord-opus-ingest-v2.schema.json": {
			"job_generation", "connection_generation", `"minimum": 1`,
		},
		"session-refresh-response.schema.json": {
			"SessionRefreshResponse", "idle_expires_at", "absolute_expires_at", "Activity refresh never moves this timestamp",
		},
	}
	for file, wants := range files {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing %q", file, want)
			}
		}
	}

	runtimeConfig, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(runtimeConfig), "caption_audio_url") {
		t.Fatal("service runtime config must not accept a profile supplied caption audio URL")
	}

	openapi, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/auth/session/refresh:",
		"#/components/schemas/SessionRefreshResponse",
		"without extending its absolute lifetime",
		"discord_caption_audio_forward_unavailable",
		"worker_deepgram_transcription_unavailable",
		"minimum: 8",
	} {
		if !strings.Contains(string(openapi), want) {
			t.Fatalf("control-api.yaml is missing %q", want)
		}
	}
}

func TestDiscordBotStartJobRequiresPositiveJobGeneration(t *testing.T) {
	schema := compileContractJSONSchema(t, "discord-bot-start-job-request.schema.json")
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "positive generation",
			body:  `{"stream_id":"stream-01","job_generation":17,"guild_id":"guild-01","voice_channel_id":"voice-01"}`,
			valid: true,
		},
		{
			name: "missing generation",
			body: `{"stream_id":"stream-01","guild_id":"guild-01","voice_channel_id":"voice-01"}`,
		},
		{
			name: "zero generation",
			body: `{"stream_id":"stream-01","job_generation":0,"guild_id":"guild-01","voice_channel_id":"voice-01"}`,
		},
		{
			name: "negative generation",
			body: `{"stream_id":"stream-01","job_generation":-1,"guild_id":"guild-01","voice_channel_id":"voice-01"}`,
		},
		{
			name: "fractional generation",
			body: `{"stream_id":"stream-01","job_generation":1.5,"guild_id":"guild-01","voice_channel_id":"voice-01"}`,
		},
		{
			name: "string generation",
			body: `{"stream_id":"stream-01","job_generation":"17","guild_id":"guild-01","voice_channel_id":"voice-01"}`,
		},
		{
			name: "unknown property",
			body: `{"stream_id":"stream-01","job_generation":17,"guild_id":"guild-01","voice_channel_id":"voice-01","unexpected":true}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var payload any
			decoder := json.NewDecoder(strings.NewReader(test.body))
			decoder.UseNumber()
			if err := decoder.Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(payload); (err == nil) != test.valid {
				t.Fatalf("valid=%t: %v", test.valid, err)
			}
		})
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "discord-bot-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"  /jobs/start:",
		"operationId: startDiscordBotJob",
		`$ref: "#/components/schemas/DiscordBotStartJobRequest"`,
		`$ref: "../schemas/discord-bot-start-job-request.schema.json"`,
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("discord-bot-api.yaml is missing start-job contract marker %q", marker)
		}
	}
}

func TestDiscordBotStartJobGoJSONUsesUint64Generation(t *testing.T) {
	var maximum DiscordBotStartJobRequest
	if err := json.Unmarshal([]byte(`{"stream_id":"stream-01","job_generation":18446744073709551615,"guild_id":"guild-01","voice_channel_id":"voice-01"}`), &maximum); err != nil {
		t.Fatalf("decode maximum uint64 generation: %v", err)
	}
	if maximum.JobGeneration != 18446744073709551615 {
		t.Fatalf("decoded job_generation=%d, want maximum uint64", maximum.JobGeneration)
	}

	var overflow DiscordBotStartJobRequest
	if err := json.Unmarshal([]byte(`{"stream_id":"stream-01","job_generation":18446744073709551616,"guild_id":"guild-01","voice_channel_id":"voice-01"}`), &overflow); err == nil {
		t.Fatal("Go JSON decode accepted job_generation outside the uint64 range")
	}

	encoded, err := json.Marshal(DiscordBotStartJobRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"job_generation":0`) {
		t.Fatalf("DiscordBotStartJobRequest omitted required job_generation: %s", encoded)
	}
}

func TestDiscordOpusIngestV2RequiresGenerationFences(t *testing.T) {
	schema := compileContractJSONSchema(t, "discord-opus-ingest-v2.schema.json")
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "generation fenced packet",
			body:  `{"stream_id":"stream-01","source":"discord","packets":[{"ssrc":42,"user_id":"user-42","job_generation":7,"connection_generation":3,"sequence":1,"timestamp":960,"received_at":"2026-08-16T08:08:01Z","opus_base64":"AQ=="}]}`,
			valid: true,
		},
		{
			name:  "missing job generation",
			body:  `{"stream_id":"stream-01","source":"discord","packets":[{"ssrc":42,"connection_generation":3,"sequence":1,"timestamp":960,"received_at":"2026-08-16T08:08:01Z","opus_base64":"AQ=="}]}`,
			valid: false,
		},
		{
			name:  "missing connection generation",
			body:  `{"stream_id":"stream-01","source":"discord","packets":[{"ssrc":42,"job_generation":7,"sequence":1,"timestamp":960,"received_at":"2026-08-16T08:08:01Z","opus_base64":"AQ=="}]}`,
			valid: false,
		},
		{
			name:  "zero connection generation",
			body:  `{"stream_id":"stream-01","source":"discord","packets":[{"ssrc":42,"job_generation":7,"connection_generation":0,"sequence":1,"timestamp":960,"received_at":"2026-08-16T08:08:01Z","opus_base64":"AQ=="}]}`,
			valid: false,
		},
	}
	for _, test := range tests {
		var payload any
		if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(payload); (err == nil) != test.valid {
			t.Fatalf("%s valid=%t: %v", test.name, test.valid, err)
		}
	}
}
