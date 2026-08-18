package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControlOpenAPIStreamArtifactReportFailureResponses(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.ReplaceAll(string(body), "\r\n", "\n")

	pathMarker := "  /services/stream-artifacts:\n"
	pathStart := strings.Index(raw, pathMarker)
	if pathStart < 0 {
		t.Fatal("control-api.yaml is missing POST /services/stream-artifacts")
	}
	pathSection := raw[pathStart+len(pathMarker):]
	if end := strings.Index(pathSection, "\n  /"); end >= 0 {
		pathSection = pathSection[:end]
	}
	const postMarker = "    post:\n"
	postStart := strings.Index(pathSection, postMarker)
	if postStart < 0 {
		t.Fatal("control-api.yaml is missing POST /services/stream-artifacts")
	}
	operation := pathSection[postStart+len(postMarker):]

	tests := []struct {
		status string
		codes  []string
	}{
		{
			status: "404",
			codes:  []string{"service_or_stream_not_found", "service_not_registered", "stream_not_found"},
		},
		{status: "500", codes: []string{"stream_artifact_store_failed"}},
		{status: "503", codes: []string{"stream_artifact_store_unavailable"}},
	}
	for _, test := range tests {
		responseMarker := "        \"" + test.status + "\":\n"
		responseStart := strings.Index(operation, responseMarker)
		if responseStart < 0 {
			t.Fatalf("POST /services/stream-artifacts is missing HTTP %s", test.status)
		}
		response := operation[responseStart+len(responseMarker):]
		if end := strings.Index(response, "\n        \""); end >= 0 {
			response = response[:end]
		}
		for _, code := range test.codes {
			if !strings.Contains(response, code) {
				t.Fatalf("POST /services/stream-artifacts HTTP %s is missing %q", test.status, code)
			}
		}
		if !strings.Contains(response, `$ref: "#/components/schemas/ErrorResponse"`) {
			t.Fatalf("POST /services/stream-artifacts HTTP %s must return ErrorResponse", test.status)
		}
		if test.status == "503" {
			for _, marker := range []string{"Retry-After:", "type: integer", "minimum: 1", "example: 1"} {
				if !strings.Contains(response, marker) {
					t.Fatalf("POST /services/stream-artifacts HTTP 503 is missing retry header marker %q", marker)
				}
			}
		} else if strings.Contains(response, "Retry-After:") {
			t.Fatalf("POST /services/stream-artifacts HTTP %s must not advertise a retry delay", test.status)
		}
	}

	errorResponseMarker := "    ErrorResponse:\n"
	errorResponseStart := strings.Index(raw, errorResponseMarker)
	if errorResponseStart < 0 {
		t.Fatal("control-api.yaml is missing ErrorResponse")
	}
	errorResponse := raw[errorResponseStart+len(errorResponseMarker):]
	if end := strings.Index(errorResponse, "\n    User:\n"); end >= 0 {
		errorResponse = errorResponse[:end]
	}
	for _, code := range []string{
		"service_or_stream_not_found",
		"service_not_registered",
		"stream_not_found",
		"stream_artifact_store_failed",
		"stream_artifact_store_unavailable",
	} {
		if !strings.Contains(errorResponse, "\n            - "+code+"\n") {
			t.Fatalf("ErrorResponse enum is missing %q", code)
		}
	}
}

func TestServiceArtifactReportSchemaSupportsLegacyAndRunScopedArchives(t *testing.T) {
	schema := compileContractJSONSchema(t, "service-artifact-report.schema.json")
	validate := func(name, body string, wantValid bool) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			var value any
			if err := json.Unmarshal([]byte(body), &value); err != nil {
				t.Fatal(err)
			}
			err := schema.Validate(value)
			if wantValid && err != nil {
				t.Fatalf("expected valid report, got %v", err)
			}
			if !wantValid && err == nil {
				t.Fatal("expected invalid report")
			}
		})
	}

	validate("legacy path remains accepted", `{
		"service_id":"enc-01","stream_id":"stream-01",
		"artifacts":[{"kind":"archive","name":"final.mp4","relative_path":"final/stream-01/final.mp4","size_bytes":123}]
	}`, true)
	validate("run scoped path and metadata are accepted", `{
		"service_id":"enc-01","stream_id":"stream-01",
		"archive_run_id":"20260818_140629_123456789_JST",
		"archive_started_at":"2026-08-18T05:06:29.123456789Z",
		"artifacts":[{"kind":"archive","name":"final.mp4","relative_path":"final/stream-01/20260818_140629_123456789_JST/final.mp4","size_bytes":456}]
	}`, true)
	validate("run id requires start time", `{
		"service_id":"enc-01","stream_id":"stream-01","archive_run_id":"run-01",
		"artifacts":[{"kind":"archive","name":"final.mp4","relative_path":"final/stream-01/run-01/final.mp4","size_bytes":1}]
	}`, false)
	validate("start time requires run id", `{
		"service_id":"enc-01","stream_id":"stream-01","archive_started_at":"2026-08-18T05:06:29Z",
		"artifacts":[{"kind":"archive","name":"final.mp4","relative_path":"final/stream-01/final.mp4","size_bytes":1}]
	}`, false)
	validate("nested traversal is rejected", `{
		"service_id":"enc-01","stream_id":"stream-01","archive_run_id":"run-01","archive_started_at":"2026-08-18T05:06:29Z",
		"artifacts":[{"kind":"archive","name":"final.mp4","relative_path":"final/stream-01/run-01/extra/final.mp4","size_bytes":1}]
	}`, false)
}
