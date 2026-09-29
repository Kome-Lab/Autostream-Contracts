package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	contractschemas "github.com/example/autostream-contracts/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"sync"
	"time"
)

const CapabilityStreamStartPrepareCommitV2 = "stream_start_prepare_commit_v2"

// The visual epoch is CP-owned; it is not the Worker's job generation.
type EncoderStartPreparationIdentity struct {
	StreamID         string `json:"stream_id"`
	StartID          string `json:"start_id"`
	EncoderServiceID string `json:"encoder_service_id"`
	JobGeneration    uint64 `json:"job_generation"`
	ArchiveRunID     string `json:"archive_run_id"`
}
type EncoderStartPreparationRequest struct {
	SchemaVersion    int                             `json:"schema_version"`
	StartID          string                          `json:"start_id"`
	EncoderServiceID string                          `json:"encoder_service_id"`
	StartRequest     EncoderPreparationStreamRequest `json:"start_request"`
}
type EncoderStartPreparationAction struct {
	SchemaVersion    int    `json:"schema_version"`
	StreamID         string `json:"stream_id"`
	StartID          string `json:"start_id"`
	EncoderServiceID string `json:"encoder_service_id"`
	JobGeneration    uint64 `json:"job_generation"`
}
type EncoderStartPreparationPrepared struct {
	SchemaVersion int                             `json:"schema_version"`
	Identity      EncoderStartPreparationIdentity `json:"identity"`
	Phase         string                          `json:"phase"`
	ExpiresAt     time.Time                       `json:"expires_at"`
	VideoIngest   EncoderVideoIngest              `json:"video_ingest"`
}

// Route credentials and arbitrary config are deliberately unrepresentable.
type EncoderStartPreparationStatus struct {
	SchemaVersion int                             `json:"schema_version"`
	Identity      EncoderStartPreparationIdentity `json:"identity"`
	Phase         string                          `json:"phase"`
	ExpiresAt     *time.Time                      `json:"expires_at,omitempty"`
	Process       *EncoderStartStreamResponse     `json:"process,omitempty"`
	CoverState    *VideoCoverRuntimeState         `json:"cover_state,omitempty"`
	Code          string                          `json:"code,omitempty"`
}

// The nested wire preserves the existing runtime metadata schema's rtmp_url.
type EncoderPreparationStreamRequest struct {
	EncoderStartStreamRequest
	YouTubeRuntime *EncoderPreparationYouTubeRuntime `json:"youtube_runtime,omitempty"`
}
type EncoderPreparationYouTubeRuntime struct {
	YouTubeRuntimeConfig
	RTMPURL string `json:"rtmp_url,omitempty"`
}

var errStartPreparation = errors.New("invalid start preparation")
var preparationSchemas struct {
	once   sync.Once
	values map[string]*jsonschema.Schema
	err    error
}

func preparationSchema(name string) (*jsonschema.Schema, error) {
	preparationSchemas.once.Do(func() {
		c := jsonschema.NewCompiler()
		c.AssertFormat()
		c.UseLoader(denyEncoderVideoCoverExternalSchemaLoader{})
		entries, err := contractschemas.RuntimeValidationFS.ReadDir(".")
		if err != nil {
			preparationSchemas.err = err
			return
		}
		for _, entry := range entries {
			b, e := contractschemas.RuntimeValidationFS.ReadFile(entry.Name())
			if e != nil {
				preparationSchemas.err = e
				return
			}
			doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(b))
			if e != nil {
				preparationSchemas.err = e
				return
			}
			if e = c.AddResource(entry.Name(), doc); e != nil {
				preparationSchemas.err = e
				return
			}
			if entry.Name() == encoderVideoCoverCatalogSchemaName {
				if e = c.AddResource(encoderVideoCoverCatalogSchemaID, doc); e != nil {
					preparationSchemas.err = e
					return
				}
			}
			var id struct {
				ID string `json:"$id"`
			}
			_ = json.Unmarshal(b, &id)
			if id.ID != "" {
				if e = c.AddResource(id.ID, doc); e != nil {
					preparationSchemas.err = e
					return
				}
			}
		}
		preparationSchemas.values = map[string]*jsonschema.Schema{}
		for _, kind := range []string{"request", "action", "prepared", "status"} {
			schema, e := c.Compile("encoder-start-preparation-" + kind + ".schema.json")
			if e != nil {
				preparationSchemas.err = e
				return
			}
			preparationSchemas.values[kind] = schema
		}
	})
	return preparationSchemas.values[name], preparationSchemas.err
}
func decodePreparation(data []byte, kind string, value any) error {
	schema, err := preparationSchema(kind)
	if err != nil {
		return errStartPreparation
	}
	doc, err := decodeEncoderVideoCoverStrictJSON(data, value)
	if err != nil || schema.Validate(doc) != nil {
		return errStartPreparation
	}
	return nil
}
func DecodeEncoderStartPreparation(data []byte) (EncoderStartPreparationRequest, error) {
	var r EncoderStartPreparationRequest
	err := decodePreparation(data, "request", &r)
	return r, err
}
func DecodeEncoderStartPreparationAction(data []byte, streamID, startID string) (EncoderStartPreparationAction, error) {
	var r EncoderStartPreparationAction
	if err := decodePreparation(data, "action", &r); err != nil {
		return r, err
	}
	if r.StreamID != streamID || r.StartID != startID {
		return r, errStartPreparation
	}
	return r, nil
}
func DecodeEncoderStartPreparationPrepared(data []byte, expected EncoderStartPreparationIdentity) (EncoderStartPreparationPrepared, error) {
	var r EncoderStartPreparationPrepared
	if err := decodePreparation(data, "prepared", &r); err != nil {
		return r, err
	}
	if r.Identity != expected || r.ExpiresAt.IsZero() {
		return r, errStartPreparation
	}
	return r, nil
}
func DecodeEncoderStartPreparationStatus(data []byte, expected EncoderStartPreparationIdentity) (EncoderStartPreparationStatus, error) {
	var r EncoderStartPreparationStatus
	if err := decodePreparation(data, "status", &r); err != nil {
		return r, err
	}
	if r.Identity != expected {
		return r, errStartPreparation
	}
	if r.Phase == "running" {
		if r.Process == nil || r.Process.StreamID != expected.StreamID || r.Process.VideoIngest != nil || r.CoverState == nil || r.CoverState.JobGeneration != expected.JobGeneration {
			return r, errStartPreparation
		}
		encoded, _ := json.Marshal(r.CoverState)
		if ValidateEncoderVideoCoverRuntimeState(expected.StreamID, encoded) != nil {
			return r, errStartPreparation
		}
	}
	return r, nil
}
