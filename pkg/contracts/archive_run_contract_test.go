package contracts

import (
	"encoding/json"
	"testing"
)

func TestEncoderArchiveRunRequestSchemasRemainBackwardCompatible(t *testing.T) {
	tests := []struct {
		name       string
		schemaFile string
		body       string
		wantValid  bool
	}{
		{
			name:       "legacy start",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","name":"Live","rtmp_url":"rtmps://example.invalid/live"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped start",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","rtmp_url":"rtmps://example.invalid/live","started_at":"2026-08-18T05:06:29.123456789Z"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped start requires time",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","rtmp_url":"rtmps://example.invalid/live"}`,
			wantValid:  false,
		},
		{
			name:       "unsafe run id is rejected",
			schemaFile: "encoder-start-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run..01","name":"Live","rtmp_url":"rtmps://example.invalid/live","started_at":"2026-08-18T05:06:29Z"}`,
			wantValid:  false,
		},
		{
			name:       "legacy package",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","name":"Live"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped package",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live","started_at":"2026-08-18T05:06:29Z"}`,
			wantValid:  true,
		},
		{
			name:       "run scoped package requires time",
			schemaFile: "encoder-package-stream-request.schema.json",
			body:       `{"stream_id":"stream-01","archive_run_id":"run-01","name":"Live"}`,
			wantValid:  false,
		},
		{
			name:       "stream response carries current archive run",
			schemaFile: "stream-job.schema.json",
			body:       `{"id":"stream-01","name":"Live","status":"completed","archive_run_id":"run-01","archive_started_at":"2026-08-18T05:06:29Z","archive_reported_at":"2026-08-18T05:07:29Z"}`,
			wantValid:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dependencies := []string(nil)
			if test.schemaFile == "encoder-start-stream-request.schema.json" {
				dependencies = append(dependencies, "youtube-runtime-config.schema.json")
			}
			schema := compileContractJSONSchema(t, test.schemaFile, dependencies...)
			var value any
			if err := json.Unmarshal([]byte(test.body), &value); err != nil {
				t.Fatal(err)
			}
			err := schema.Validate(value)
			if test.wantValid && err != nil {
				t.Fatalf("expected valid request: %v", err)
			}
			if !test.wantValid && err == nil {
				t.Fatal("expected invalid request")
			}
		})
	}
}
