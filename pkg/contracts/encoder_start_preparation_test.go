package contracts

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func preparationFixture(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile("../../tests/start_preparation_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if json.Unmarshal(b, &document) != nil {
		t.Fatal("fixture JSON")
	}
	return document
}
func TestStartPreparationCanonicalDecoder(t *testing.T) {
	makeRequest := func() map[string]any { return preparationFixture(t)["prepare"].(map[string]any) }
	cases := []struct {
		name  string
		edit  func(map[string]any)
		valid bool
	}{
		{"managed", func(map[string]any) {}, true},
		{"schema missing", func(m map[string]any) { delete(m, "schema_version") }, false},
		{"schema type", func(m map[string]any) { m["schema_version"] = "2" }, false},
		{"schema future", func(m map[string]any) { m["schema_version"] = 3 }, false},
		{"uuid invalid", func(m map[string]any) { m["start_id"] = "arbitrary" }, false},
		{"unknown", func(m map[string]any) { m["unbound"] = true }, false},
		{"nested unknown", func(m map[string]any) { m["start_request"].(map[string]any)["stream_key"] = "fixture" }, false},
		{"managed false", func(m map[string]any) { m["start_request"].(map[string]any)["worker_video_ingest"] = false }, false},
		{"managed missing", func(m map[string]any) { delete(m["start_request"].(map[string]any), "worker_video_ingest") }, false},
		{"input even empty", func(m map[string]any) { m["start_request"].(map[string]any)["input_url"] = "" }, false},
		{"epoch zero", func(m map[string]any) {
			m["start_request"].(map[string]any)["video_cover_start"].(map[string]any)["job_generation"] = 0
		}, false},
		{"active required false", func(m map[string]any) {
			delete(m["start_request"].(map[string]any)["video_cover_start"].(map[string]any), "active")
		}, false},
		{"archive missing", func(m map[string]any) { delete(m["start_request"].(map[string]any), "archive_run_id") }, false},
		{"null nested", func(m map[string]any) { m["start_request"] = nil }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeRequest()
			tc.edit(m)
			b, _ := json.Marshal(m)
			_, err := DecodeEncoderStartPreparation(b)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
	b, _ := json.Marshal(makeRequest())
	for _, bad := range []string{string(b) + " {}", strings.Replace(string(b), `"schema_version":2`, `"schema_version":2,"schema_version":2`, 1)} {
		if _, err := DecodeEncoderStartPreparation([]byte(bad)); err == nil {
			t.Fatal("accepted duplicate or trailing JSON")
		}
	}
}

func TestStartPreparationSharedResponseFixtures(t *testing.T) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(readContractSource(t, "tests", "start_preparation_fixtures.json")), &values); err != nil {
		t.Fatal(err)
	}
	var action EncoderStartPreparationAction
	_ = json.Unmarshal(values["action"], &action)
	id := EncoderStartPreparationIdentity{StreamID: action.StreamID, StartID: action.StartID, EncoderServiceID: action.EncoderServiceID, JobGeneration: action.JobGeneration, ArchiveRunID: "archive-01"}
	if _, err := DecodeEncoderStartPreparationPrepared(values["prepared"], id); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeEncoderStartPreparationStatus(values["running"], id); err != nil {
		t.Fatal(err)
	}
}

func TestStartPreparationSchemasCompileOffline(t *testing.T) {
	for _, name := range []string{"request", "action", "prepared", "status"} {
		if _, err := preparationSchema(name); err != nil {
			t.Fatal(err)
		}
	}
}
func TestStartPreparationActionsExactPathAndPresence(t *testing.T) {
	m := preparationFixture(t)["action"].(map[string]any)
	b, _ := json.Marshal(m)
	if _, err := DecodeEncoderStartPreparationAction(b, "stream-01", m["start_id"].(string)); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeEncoderStartPreparationAction(b, "other", m["start_id"].(string)); err == nil {
		t.Fatal("wrong stream")
	}
	if _, err := DecodeEncoderStartPreparationAction(b, "stream-01", "other"); err == nil {
		t.Fatal("wrong attempt")
	}
	for _, key := range []string{"schema_version", "stream_id", "start_id", "encoder_service_id", "job_generation"} {
		copy := preparationFixture(t)["action"].(map[string]any)
		delete(copy, key)
		b, _ := json.Marshal(copy)
		if _, err := DecodeEncoderStartPreparationAction(b, "stream-01", m["start_id"].(string)); err == nil {
			t.Fatalf("missing %s", key)
		}
	}
}
func TestStartPreparationStatusRequiresRealWitness(t *testing.T) {
	id := EncoderStartPreparationIdentity{StreamID: "stream-1", StartID: "11111111-1111-4111-8111-111111111111", EncoderServiceID: "encoder-01", JobGeneration: 9, ArchiveRunID: "archive-01"}
	makeStatus := func() map[string]any {
		return map[string]any{"schema_version": 2, "identity": id, "phase": "running", "process": map[string]any{"stream_id": "stream-1", "name": "Synthetic", "status": "running", "started_at_jst": "2026-09-29T09:00:00+09:00", "archive": map[string]string{}}, "cover_state": validVideoCoverRuntimeFixture()}
	}
	valid := makeStatus()
	b, _ := json.Marshal(valid)
	if _, err := DecodeEncoderStartPreparationStatus(b, id); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(map[string]any){func(m map[string]any) { m["phase"] = "prepared" }, func(m map[string]any) { delete(m["cover_state"].(map[string]any), "applied_witness") }, func(m map[string]any) { m["video_ingest"] = map[string]any{} }, func(m map[string]any) { m["phase"] = "succeeded" }, func(m map[string]any) { m["cover_state"].(map[string]any)["job_generation"] = 8 }} {
		m := makeStatus()
		mutate(m)
		b, _ := json.Marshal(m)
		if _, err := DecodeEncoderStartPreparationStatus(b, id); err == nil {
			t.Fatal("accepted invalid running witness")
		}
	}
}
