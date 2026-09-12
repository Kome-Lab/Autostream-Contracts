package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncoderPackageSchemaUsesOnlyRunScopedRuntimeAssignment(t *testing.T) {
	schema := readContractSchema(t, "encoder-package-stream-request.schema.json")
	if schema.AdditionalProperties != false {
		t.Fatal("encoder package requests must reject unknown fields")
	}
	requireContractFields(t, schema.Required, "stream_id", "archive_run_id", "name", "started_at")
	for _, removed := range []string{"archive_config", "base_path", "folder_id_secret_name", "refresh_token_secret_name"} {
		if _, exists := schema.Properties[removed]; exists {
			t.Fatalf("encoder package schema retained inline runtime field %q", removed)
		}
	}
}

func TestArchiveMetadataSchemaRedactsDriveIDs(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "archive-metadata.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, want := range []string{
		"folder_id_fingerprint",
		"file_fingerprints",
		"Raw Drive file IDs must not be stored in metadata",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("archive metadata schema is missing redaction marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"\"folder_id\"",
		"\"file_ids\"",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("archive metadata schema still exposes raw Drive ID field %q", forbidden)
		}
	}
}

func TestStreamSchemasDocumentDiscordChannelOverrides(t *testing.T) {
	for _, file := range []string{"stream-write.schema.json", "stream-settings-write.schema.json", "stream-job.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"discord_config_id",
			"discord_guild_id",
			"discord_voice_channel_id",
			"discord_text_channel_id",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing stream-specific Discord routing field %q", file, want)
			}
		}
	}

	for _, file := range []string{"stream-write.schema.json", "stream-settings-write.schema.json"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(body)
		for _, want := range []string{
			"auto_start_trigger",
			"discord_voice_join",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s is missing Discord voice join auto-start field %q", file, want)
			}
		}
	}

	runtimeBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "service-runtime-config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"auto_start_trigger",
		"discord_voice_join",
	} {
		if !strings.Contains(string(runtimeBody), want) {
			t.Fatalf("service-runtime-config.schema.json is missing Discord voice join auto-start marker %q", want)
		}
	}
}

func TestPreviewAndDiscordNotificationContracts(t *testing.T) {
	files := map[string][]string{
		"stream-preview-link.schema.json": {
			"StreamPreviewLink", "expires_at", "bearer capability", "uri-reference", "12 hours", "stream is no longer",
		},
		"youtube-live-notification-request.schema.json": {
			"YouTubeLiveNotificationRequest", "event_id", "watch_url", "Idempotency key", "runtime config",
		},
		"youtube-live-notification-response.schema.json": {
			"YouTubeLiveNotificationResponse", "message_id", "already_sent",
		},
		"service-notification-error.schema.json": {
			"ServiceNotificationError", "retryable",
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

	openAPIs := map[string][]string{
		"control-api.yaml": {
			"/streams/{id}/preview/{name}:",
			"/streams/{id}/preview-links:",
			"/stream-previews/{token}/{name}:",
			"segment-[0-9]{6}",
			"12 hours",
			"never rolls the live stream",
			"watch URL fingerprint",
		},
		"encoder-recorder-api.yaml": {
			"/streams/{id}/preview/{name}:",
			"serviceToken",
			"full active-stream playlist",
			"must not stop either primary output",
		},
		"discord-bot-api.yaml": {
			"/streams/{id}/notifications/youtube-live:",
			"event_id",
			"watch_url",
			"mentions disabled",
			"runtime config",
			"Retry-After",
		},
	}
	for file, wants := range openAPIs {
		body, err := os.ReadFile(filepath.Join("..", "..", "openapi", file))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s is missing %q", file, want)
			}
		}
	}
}
