package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticRerunResponseWireMatchesCanonicalSchema(t *testing.T) {
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	response := DiagnosticRerunResponse{
		Incident: Incident{
			ID:        "incident-01",
			Rule:      IncidentRule("worker_event_send_failed"),
			Severity:  SeverityWarning,
			Status:    IncidentAcknowledged,
			SummaryJA: "Worker event dispatch failed.",
			ServiceID: "worker-01",
			SignalID:  "signal-01",
			Report: DiagnosticReport{
				Summary:            "Worker event dispatch failed",
				LikelyCause:        "Encoder endpoint unavailable",
				Confidence:         0.9,
				Evidence:           []string{"signal_id=signal-01"},
				Impact:             "archive event delivery is delayed",
				RecommendedActions: []string{"Refresh Service Status"},
				SafeAutoCandidates: []string{},
				ApprovalRequired:   []string{},
			},
			OpenedAt:  now.Add(-time.Minute),
			UpdatedAt: now,
		},
		Outcome: DiagnosticRerunOutcomeInconclusive,
		Reason:  DiagnosticRerunReasonSavedSignalMissing,
	}

	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	canonical := compileContractJSONSchema(t, "diagnostic-rerun-response.schema.json", "incident.schema.json")
	if err := canonical.Validate(document); err != nil {
		t.Fatalf("DiagnosticRerunResponse wire JSON violates its canonical schema: %v\n%s", err, body)
	}

	response.Reason = ""
	body, err = json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	if err := canonical.Validate(document); err == nil {
		t.Fatal("diagnostic rerun schema accepted inconclusive output without a reason")
	}

	response.Outcome = DiagnosticRerunOutcomeEvaluated
	response.Reason = DiagnosticRerunReasonSavedSignalMissing
	body, err = json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	if err := canonical.Validate(document); err == nil {
		t.Fatal("diagnostic rerun schema accepted an evaluated output with an inconclusive reason")
	}

	response.Outcome = DiagnosticRerunOutcomeInconclusive
	response.Reason = DiagnosticRerunReason("unexpected_reason")
	body, err = json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	if err := canonical.Validate(document); err == nil {
		t.Fatal("diagnostic rerun schema accepted an unknown inconclusive reason")
	}
}

func TestDiagnosticRerunOpenAPIUsesCanonicalResponseSchemaAndDocumentsStatuses(t *testing.T) {
	tests := []struct {
		file     string
		path     string
		statuses map[string]string
	}{
		{
			file: "observability-api.yaml",
			path: "/incidents/{id}/diagnostics/rerun:",
			statuses: map[string]string{
				"200": "diagnostic report re-evaluated or safely left inconclusive",
				"401": "missing or invalid bearer token",
				"403": "missing diagnostics.run scope",
				"404": "incident not found",
				"500": "persisted incident, signal, or diagnostic report update failed",
			},
		},
		{
			file: "control-api.yaml",
			path: "/observability/incidents/{id}/diagnostics/rerun:",
			statuses: map[string]string{
				"200": "proxied diagnostic re-evaluation result",
				"400": "invalid incident identifier",
				"401": "missing or expired authenticated session",
				"403": "missing diagnostics.run permission or failed CSRF check",
				"404": "incident not found",
				"429": "Observability rate limited the re-evaluation request",
				"502": "Observability request or authentication failed",
				"503": "Observability is not configured or unavailable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("..", "..", "openapi", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			raw := string(body)
			pathSection := openAPIPathSectionForDiagnosticRerunTest(t, raw, tt.path)
			if !strings.Contains(pathSection, `#/components/schemas/DiagnosticRerunResponse`) {
				t.Fatalf("%s 200 response does not reference DiagnosticRerunResponse: %s", tt.file, pathSection)
			}
			for status, description := range tt.statuses {
				want := `"` + status + `":` + "\n          description: " + description
				if !strings.Contains(pathSection, want) {
					t.Fatalf("%s diagnostic rerun response lacks %s", tt.file, want)
				}
			}

			component := openAPIComponentSectionForDiagnosticRerunTest(t, raw, "DiagnosticRerunResponse")
			const canonicalReference = `$ref: "../schemas/diagnostic-rerun-response.schema.json"`
			if strings.Count(component, canonicalReference) != 1 || strings.Contains(component, "properties:") || strings.Contains(component, "additionalProperties:") {
				t.Fatalf("%s DiagnosticRerunResponse must use only canonical strict schema: %s", tt.file, component)
			}
		})
	}
}

func openAPIPathSectionForDiagnosticRerunTest(t *testing.T, raw, path string) string {
	t.Helper()
	start := strings.Index(raw, "  "+path+"\n")
	if start < 0 {
		t.Fatalf("OpenAPI is missing %s", path)
	}
	section := raw[start:]
	if end := strings.Index(section[len("  "+path+"\n"):], "\n  /"); end >= 0 {
		section = section[:len("  "+path+"\n")+end]
	}
	return section
}

func openAPIComponentSectionForDiagnosticRerunTest(t *testing.T, raw, component string) string {
	t.Helper()
	start := strings.Index(raw, "    "+component+":\n")
	if start < 0 {
		t.Fatalf("OpenAPI is missing %s component", component)
	}
	section := raw[start:]
	if end := strings.Index(section[len("    "+component+":\n"):], "\n    "); end >= 0 {
		section = section[:len("    "+component+":\n")+end]
	}
	return section
}
