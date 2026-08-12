package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validateWorkerSceneVideoContract(t *testing.T, schemaFile, body string, wantValid bool, dependencies ...string) {
	t.Helper()

	schema := compileContractJSONSchema(t, schemaFile, dependencies...)
	var document any
	if err := json.Unmarshal([]byte(body), &document); err != nil {
		t.Fatalf("decode %s fixture: %v", schemaFile, err)
	}
	err := schema.Validate(document)
	if wantValid && err != nil {
		t.Fatalf("%s rejected a valid payload: %v\n%s", schemaFile, err, body)
	}
	if !wantValid && err == nil {
		t.Fatalf("%s accepted an invalid payload:\n%s", schemaFile, body)
	}
}

func TestWorkerSceneVideoStartSchemaCompatibility(t *testing.T) {
	const passphrase = "0123456789abcdef0123456789abcdef"

	t.Run("legacy encoder start remains valid", func(t *testing.T) {
		validateWorkerSceneVideoContract(t, "encoder-start-stream-request.schema.json", `{
  "stream_id":"stream-1",
  "name":"Legacy",
  "input_url":"srt://source.example.test:9000",
  "input_mode":"external",
  "rtmp_url":"rtmps://youtube.example.test/live2"
}`, true, "youtube-runtime-config.schema.json")
	})

	t.Run("encoder opts in to a job scoped worker video listener", func(t *testing.T) {
		validateWorkerSceneVideoContract(t, "encoder-start-stream-request.schema.json", `{
  "stream_id":"stream-1",
  "name":"Worker scene",
  "rtmp_url":"rtmps://youtube.example.test/live2",
  "encoder_profile_id":"profile-1",
  "worker_video_ingest":true,
  "worker_video_ingest_token":"job-scoped-token"
}`, true, "youtube-runtime-config.schema.json")
	})

	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "opt in requires token",
			body: `{"stream_id":"stream-1","name":"Scene","rtmp_url":"","encoder_profile_id":"profile-1","worker_video_ingest":true}`,
		},
		{
			name: "opt in requires encoder profile",
			body: `{"stream_id":"stream-1","name":"Scene","rtmp_url":"","worker_video_ingest":true,"worker_video_ingest_token":"job-scoped-token"}`,
		},
		{
			name: "token requires opt in",
			body: `{"stream_id":"stream-1","name":"Scene","rtmp_url":"","worker_video_ingest_token":"job-scoped-token"}`,
		},
		{
			name: "opt in forbids caller input mode",
			body: `{"stream_id":"stream-1","name":"Scene","input_mode":"discord_opus_rtp","rtmp_url":"","worker_video_ingest":true,"worker_video_ingest_token":"job-scoped-token"}`,
		},
		{
			name: "opt in forbids caller input url",
			body: `{"stream_id":"stream-1","name":"Scene","input_url":"srt://untrusted.example.test:9000","rtmp_url":"","worker_video_ingest":true,"worker_video_ingest_token":"job-scoped-token"}`,
		},
		{
			name: "internal input mode is not caller supplied",
			body: `{"stream_id":"stream-1","name":"Scene","input_mode":"worker_scene_srt","rtmp_url":"","worker_video_ingest":true,"worker_video_ingest_token":"job-scoped-token"}`,
		},
	} {
		t.Run("encoder rejects "+test.name, func(t *testing.T) {
			validateWorkerSceneVideoContract(t, "encoder-start-stream-request.schema.json", test.body, false, "youtube-runtime-config.schema.json")
		})
	}

	t.Run("legacy worker start remains valid", func(t *testing.T) {
		validateWorkerSceneVideoContract(t, "worker-start-job-request.schema.json", `{
  "stream_id":"stream-1",
  "stream_name":"Legacy worker"
}`, true)
	})

	validWorkerStart := `{
  "stream_id":"stream-1",
  "stream_name":"Worker scene",
  "encoder_profile_id":"profile-1",
  "video_width":1920,
  "video_height":1080,
  "video_fps":60,
  "video_ingest_url":"srt://encoder.example.test:9000",
  "video_ingest_passphrase":"` + passphrase + `",
  "video_ingest_pbkeylen":32
}`
	validateWorkerSceneVideoContract(t, "worker-start-job-request.schema.json", validWorkerStart, true)
	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "720p",
			body: strings.Replace(strings.Replace(validWorkerStart, `"video_width":1920`, `"video_width":1280`, 1), `"video_height":1080`, `"video_height":720`, 1),
		},
		{
			name: "480p at minimum frame rate",
			body: strings.Replace(strings.Replace(strings.Replace(validWorkerStart, `"video_width":1920`, `"video_width":854`, 1), `"video_height":1080`, `"video_height":480`, 1), `"video_fps":60`, `"video_fps":1`, 1),
		},
	} {
		t.Run("worker accepts "+test.name, func(t *testing.T) {
			validateWorkerSceneVideoContract(t, "worker-start-job-request.schema.json", test.body, true)
		})
	}

	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "userinfo in ingest URL",
			body: strings.Replace(validWorkerStart, "srt://encoder.example.test:9000", "srt://user:secret@encoder.example.test:9000", 1),
		},
		{
			name: "query secret in ingest URL",
			body: strings.Replace(validWorkerStart, "srt://encoder.example.test:9000", "srt://encoder.example.test:9000?passphrase=secret", 1),
		},
		{
			name: "missing ingest port",
			body: strings.Replace(validWorkerStart, "srt://encoder.example.test:9000", "srt://encoder.example.test", 1),
		},
		{
			name: "trailing slash in ingest URL",
			body: strings.Replace(validWorkerStart, "srt://encoder.example.test:9000", "srt://encoder.example.test:9000/", 1),
		},
		{
			name: "out of range ingest port",
			body: strings.Replace(validWorkerStart, "srt://encoder.example.test:9000", "srt://encoder.example.test:65536", 1),
		},
		{
			name: "non ASCII passphrase",
			body: strings.Replace(validWorkerStart, passphrase, strings.Repeat("あ", 32), 1),
		},
		{
			name: "short passphrase",
			body: strings.Replace(validWorkerStart, passphrase, passphrase[:31], 1),
		},
		{
			name: "long passphrase",
			body: strings.Replace(validWorkerStart, passphrase, strings.Repeat("a", 80), 1),
		},
		{
			name: "unsupported pbkeylen",
			body: strings.Replace(validWorkerStart, `"video_ingest_pbkeylen":32`, `"video_ingest_pbkeylen":16`, 1),
		},
		{
			name: "incomplete video configuration",
			body: strings.Replace(validWorkerStart, `  "video_width":1920,`+"\n", "", 1),
		},
		{
			name: "unsupported resolution",
			body: strings.Replace(strings.Replace(validWorkerStart, `"video_width":1920`, `"video_width":640`, 1), `"video_height":1080`, `"video_height":360`, 1),
		},
		{
			name: "mismatched resolution pair",
			body: strings.Replace(validWorkerStart, `"video_height":1080`, `"video_height":720`, 1),
		},
		{
			name: "unsupported frame rate",
			body: strings.Replace(validWorkerStart, `"video_fps":60`, `"video_fps":61`, 1),
		},
	} {
		t.Run("worker rejects "+test.name, func(t *testing.T) {
			validateWorkerSceneVideoContract(t, "worker-start-job-request.schema.json", test.body, false)
		})
	}

	legacyResponse := `{
  "stream_id":"stream-1",
  "name":"Legacy",
  "status":"running",
  "started_at_jst":"2026-08-12T12:00:00+09:00",
  "archive":{}
}`
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", legacyResponse, true)

	validResponse := `{
  "stream_id":"stream-1",
  "name":"Worker scene",
  "status":"starting",
  "started_at_jst":"2026-08-12T12:00:00+09:00",
  "archive":{},
  "video_ingest":{
    "url":"srt://encoder.example.test:9000",
    "passphrase":"` + passphrase + `",
    "pbkeylen":32
  }
}`
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", validResponse, true)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, `"pbkeylen":32`, `"pbkeylen":16`, 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, "srt://encoder.example.test:9000", "srt://encoder.example.test:9000?passphrase=secret", 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, "srt://encoder.example.test:9000", "srt://encoder.example.test", 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, "srt://encoder.example.test:9000", "srt://encoder.example.test:9000/", 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, "srt://encoder.example.test:9000", "srt://encoder.example.test:65536", 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, passphrase, strings.Repeat("a", 80), 1), false)
	validateWorkerSceneVideoContract(t, "encoder-start-stream-response.schema.json", strings.Replace(validResponse, passphrase, strings.Repeat("あ", 32), 1), false)
}

func TestWorkerSceneVideoSecretsAreWriteOnlyAndInternal(t *testing.T) {
	for schemaFile, fields := range map[string][]string{
		"encoder-start-stream-request.schema.json": {"worker_video_ingest_token"},
		"worker-start-job-request.schema.json":     {"stream_ingest_token", "video_ingest_passphrase"},
	} {
		body, err := os.ReadFile(filepath.Join("..", "..", "schemas", schemaFile))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(body, &document); err != nil {
			t.Fatal(err)
		}
		for _, field := range fields {
			properties := document["properties"].(map[string]any)
			parts := strings.Split(field, ".")
			property := properties[parts[0]].(map[string]any)
			if len(parts) == 2 {
				properties = property["properties"].(map[string]any)
				property = properties[parts[1]].(map[string]any)
			}
			if property["writeOnly"] != true {
				t.Fatalf("%s %s must be writeOnly", schemaFile, field)
			}
			for _, forbidden := range []string{"example", "examples", "default"} {
				if _, ok := property[forbidden]; ok {
					t.Fatalf("%s %s must not publish a secret %s", schemaFile, field, forbidden)
				}
			}
		}
	}

	responseBody, err := os.ReadFile(filepath.Join("..", "..", "schemas", "encoder-start-stream-response.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var responseDocument map[string]any
	if err := json.Unmarshal(responseBody, &responseDocument); err != nil {
		t.Fatal(err)
	}
	videoIngest := responseDocument["properties"].(map[string]any)["video_ingest"].(map[string]any)
	passphrase := videoIngest["properties"].(map[string]any)["passphrase"].(map[string]any)
	if passphrase["readOnly"] != true {
		t.Fatal("encoder start response video_ingest.passphrase must be readOnly")
	}
	for _, forbidden := range []string{"example", "examples", "default"} {
		if _, ok := passphrase[forbidden]; ok {
			t.Fatalf("encoder start response passphrase must not publish a secret %s", forbidden)
		}
	}

	publicAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"worker_video_ingest_token", "video_ingest_passphrase"} {
		if strings.Contains(string(publicAPI), forbidden) {
			t.Fatalf("public Control API must not expose internal secret field %q", forbidden)
		}
	}

	internalAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "encoder-recorder-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/streams/start:",
		"internal service-to-service response",
		"#/components/schemas/EncoderStartStreamRequest",
		"#/components/schemas/EncoderStartStreamResponse",
		"../schemas/encoder-start-stream-request.schema.json",
		"../schemas/encoder-start-stream-response.schema.json",
	} {
		if !strings.Contains(string(internalAPI), want) {
			t.Fatalf("Encoder/Recorder internal OpenAPI is missing %q", want)
		}
	}
}

func TestWorkerSceneVideoCapabilityVocabulary(t *testing.T) {
	schema := compileContractJSONSchema(t, "service-registration.schema.json", "encoder-output-relay-capabilities.schema.json")
	for _, body := range []string{
		`{"service_id":"worker-1","service_type":"worker","service_name":"Worker","public_url":"https://worker.example.test","version":"v1","capabilities":{"scene_video_srt":true}}`,
		`{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","public_url":"https://encoder.example.test","version":"v1","capabilities":{"worker_video_ingest_srt":true}}`,
	} {
		var document any
		if err := json.Unmarshal([]byte(body), &document); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(document); err != nil {
			t.Fatalf("scene video capability rejected: %v", err)
		}
	}

	for _, invalid := range []string{
		`{"service_id":"worker-1","service_type":"worker","service_name":"Worker","public_url":"https://worker.example.test","version":"v1","capabilities":{"scene_video_srt":"yes"}}`,
		`{"service_id":"worker-1","service_type":"worker","service_name":"Worker","public_url":"https://worker.example.test","version":"v1","capabilities":{"scene_video_srt":false}}`,
		`{"service_id":"encoder-1","service_type":"encoder_recorder","service_name":"Encoder","public_url":"https://encoder.example.test","version":"v1","capabilities":{"worker_video_ingest_srt":false}}`,
	} {
		var document any
		if err := json.Unmarshal([]byte(invalid), &document); err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(document); err == nil {
			t.Fatalf("scene video capability must be advertised only as true: %s", invalid)
		}
	}

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"scene_video_srt:", "worker_video_ingest_srt:"} {
		if !strings.Contains(string(openAPI), want) {
			t.Fatalf("Control API service capability vocabulary is missing %q", want)
		}
	}
}

func TestWorkerSceneVideoGoTypes(t *testing.T) {
	if got := string(EncoderInputModeWorkerSceneSRT); got != "worker_scene_srt" {
		t.Fatalf("worker scene input mode = %q", got)
	}
	if CapabilitySceneVideoSRT != "scene_video_srt" {
		t.Fatalf("worker capability = %q", CapabilitySceneVideoSRT)
	}
	if CapabilityWorkerVideoIngestSRT != "worker_video_ingest_srt" {
		t.Fatalf("encoder capability = %q", CapabilityWorkerVideoIngestSRT)
	}

	request := WorkerStartJobRequest{
		StreamID:              "stream-1",
		EncoderProfileID:      "profile-1",
		VideoWidth:            1920,
		VideoHeight:           1080,
		VideoFPS:              60,
		VideoIngestURL:        "srt://encoder.example.test:9000",
		VideoIngestPassphrase: "0123456789abcdef0123456789abcdef",
		VideoIngestPBKeyLen:   32,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"encoder_profile_id", "video_width", "video_height", "video_fps", "video_ingest_url", "video_ingest_passphrase", "video_ingest_pbkeylen"} {
		if !strings.Contains(string(payload), `"`+want+`"`) {
			t.Fatalf("worker start payload is missing %q: %s", want, payload)
		}
	}

	response := EncoderStartStreamResponse{VideoIngest: &EncoderVideoIngest{
		URL:        "srt://encoder.example.test:9000",
		Passphrase: "0123456789abcdef0123456789abcdef",
		PBKeyLen:   32,
	}}
	payload, err = json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"video_ingest":{"url":"srt://encoder.example.test:9000","passphrase":"0123456789abcdef0123456789abcdef","pbkeylen":32}`) {
		t.Fatalf("encoder start response does not preserve the internal ingest object: %s", payload)
	}
}
