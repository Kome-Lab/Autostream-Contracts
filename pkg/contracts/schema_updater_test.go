package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestUpdaterVersionEndpointContractIsUnauthenticatedAndStrict(t *testing.T) {
	for _, test := range []struct {
		file        string
		serviceType string
	}{
		{file: "control-api.yaml", serviceType: "control_panel"},
		{file: "observability-api.yaml", serviceType: "observability"},
		{file: "encoder-recorder-api.yaml", serviceType: "encoder_recorder"},
		{file: "discord-bot-api.yaml", serviceType: "discord_bot"},
	} {
		file := test.file
		bundleName := strings.TrimSuffix(file, ".yaml") + ".json"
		bundle := readNormalizedOpenAPICharacterization(t, bundleName)
		paths := requireCharacterizationMap(t, bundle, "paths")
		pathItem := requireCharacterizationMap(t, paths, "/updater/version")
		operation := requireCharacterizationMap(t, pathItem, "get")
		if operation["operationId"] != "getUpdaterVersion" {
			t.Fatalf("%s updater version operationId=%v", file, operation["operationId"])
		}
		security, ok := operation["security"].([]any)
		if !ok || len(security) != 0 {
			t.Fatalf("%s updater version security=%v, want an explicit empty requirement", file, operation["security"])
		}
		responses := requireCharacterizationMap(t, operation, "responses")
		success := resolveCharacterizationSchema(t, bundle, responses["200"])
		content := requireCharacterizationMap(t, success, "content")
		jsonContent := requireCharacterizationMap(t, content, "application/json")
		responseSchema := resolveCharacterizationSchema(t, bundle, jsonContent["schema"])
		if responseSchema["additionalProperties"] != false {
			t.Fatalf("%s updater version response permits unknown fields", file)
		}
		required := requireCharacterizationStringSet(t, responseSchema, "required")
		for _, field := range []string{"version", "service_id", "service_type", "config_revision"} {
			if _, ok := required[field]; !ok {
				t.Fatalf("%s updater version required=%v, missing %s", file, required, field)
			}
		}
		properties := requireCharacterizationMap(t, responseSchema, "properties")
		if len(required) != 4 || len(properties) != 4 {
			t.Fatalf("%s updater version required=%v properties=%v, want the exact application probe", file, required, properties)
		}
		version := requireCharacterizationMap(t, properties, "version")
		pattern, ok := version["pattern"].(string)
		if !ok {
			t.Fatalf("%s updater version pattern=%v", file, version["pattern"])
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("%s updater version pattern does not compile: %v", file, err)
		}
		if !compiled.MatchString("v1.2.3") || compiled.MatchString("1.2.3") || compiled.MatchString("v1.2") {
			t.Fatalf("%s updater version pattern does not enforce a v-prefixed semantic version: %q", file, pattern)
		}
		serviceType := requireCharacterizationMap(t, properties, "service_type")
		if serviceType["const"] != test.serviceType {
			t.Fatalf("%s updater version service_type=%v, want %s", file, serviceType["const"], test.serviceType)
		}
	}
}

func TestSystemUpdateContractsKeepExecutionDetailsServerSide(t *testing.T) {
	for _, file := range []string{
		"system-update-create-request.schema.json",
		"update-agent-claim-request.schema.json",
		"update-agent-report-request.schema.json",
		"update-agent-mutation-grant-issue-request.schema.json",
		"update-agent-mutation-grant-consume-request.schema.json",
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", file))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			AdditionalProperties bool                      `json:"additionalProperties"`
			Properties           map[string]map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatal(err)
		}
		if doc.AdditionalProperties {
			t.Fatalf("%s must reject unknown execution fields", file)
		}
		for _, forbidden := range []string{
			"url", "path", "command", "unit", "image", "version", "digest",
			"ssh_address", "ssh_user", "ssh_path", "identity_file", "remote_command",
		} {
			if _, ok := doc.Properties[forbidden]; ok {
				t.Fatalf("%s must not accept caller-supplied %s", file, forbidden)
			}
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/system-updates:",
		"/system-updates/{id}/cancel:",
		"/services/update-jobs/claim:",
		"/services/update-jobs/{id}/report:",
		"/services/update-jobs/{id}/mutation-grants:",
		"/services/update-jobs/{id}/mutation-grants/consume:",
		"The request cannot supply a URL, path, image, command, digest, version, or systemd unit.",
		"Atomically claims the next eligible job",
		"Idempotently reports monotonic progress",
		"one-time mutation grant",
		"The grant token is never accepted in the JSON body.",
	} {
		if !strings.Contains(string(openAPI), want) {
			t.Fatalf("control-api.yaml is missing system update marker %q", want)
		}
	}
}

func TestHeartbeatContractCarriesVerifiedBuildIdentity(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schemas", "heartbeat.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"version", "commit", "build_date", "hostname", "os", "arch", "capabilities", "api"} {
		if !strings.Contains(string(body), `"`+want+`"`) {
			t.Fatalf("heartbeat schema is missing %q", want)
		}
	}
}
