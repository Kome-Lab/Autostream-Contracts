package contracts

import (
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
