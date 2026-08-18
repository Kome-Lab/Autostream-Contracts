package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerCaptionRuntimeSettingsSchema(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		wantValid bool
	}{
		{name: "selected caption profile", body: `{"caption_profile_id":"caption-profile-01"}`, wantValid: true},
		{name: "missing caption profile", body: `{}`, wantValid: false},
		{name: "empty caption profile", body: `{"caption_profile_id":""}`, wantValid: false},
		{name: "unknown field", body: `{"caption_profile_id":"caption-profile-01","api_key":"must-not-be-sent"}`, wantValid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			validateWorkerSceneVideoContract(t, "worker-caption-runtime-settings-request.schema.json", test.body, test.wantValid)
		})
	}
}

func TestWorkerCaptionRuntimeSettingsGoTypeCarriesOnlyProfileReference(t *testing.T) {
	body, err := json.Marshal(WorkerCaptionRuntimeSettingsRequest{CaptionProfileID: "caption-profile-01"})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"caption_profile_id":"caption-profile-01"}` {
		t.Fatalf("unexpected payload: %s", body)
	}
}

func TestCaptionProfileLiveApplyFailureIsDocumented(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openAPI := strings.ReplaceAll(string(body), "\r\n", "\n")
	captionItemStart := strings.Index(openAPI, "  /profiles/caption/{id}:\n")
	if captionItemStart < 0 {
		t.Fatal("control-api.yaml is missing the caption profile item path")
	}
	captionItemEnd := strings.Index(openAPI[captionItemStart+1:], "\n  /profiles/overlay:")
	if captionItemEnd < 0 {
		t.Fatal("control-api.yaml caption profile item path is not bounded")
	}
	captionItem := openAPI[captionItemStart : captionItemStart+1+captionItemEnd]
	for _, marker := range []string{`"502":`, "caption_profile_saved_runtime_apply_failed", "$ref: \"#/components/schemas/ErrorResponse\""} {
		if !strings.Contains(captionItem, marker) {
			t.Fatalf("caption profile live-apply contract is missing %q", marker)
		}
	}
}
