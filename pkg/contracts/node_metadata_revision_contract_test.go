package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNodeMetadataRevisionWirePreservesOpaqueBaseline(t *testing.T) {
	revision := strings.Repeat("a", 64)
	schema := compileContractJSONSchema(t, "node-metadata-write.schema.json")
	body := map[string]any{"name": "Encoder", "description": "Operator note"}
	if err := schema.Validate(body); err != nil {
		t.Fatalf("legacy omission rejected: %v", err)
	}
	body["expected_metadata_revision"] = revision
	if err := schema.Validate(body); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []any{"", strings.Repeat("A", 64), strings.Repeat("a", 63), "2030-01-01T00:00:00Z", nil} {
		body["expected_metadata_revision"] = invalid
		if schema.Validate(body) == nil {
			t.Fatal("malformed metadata baseline accepted")
		}
	}
	name := "Encoder"
	encoded, err := json.Marshal(NodeMetadataWriteRequest{Name: &name, ExpectedMetadataRevision: &revision})
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(encoded, &document); err != nil || document["expected_metadata_revision"] != revision {
		t.Fatal("Node edit DTO changed the original opaque revision")
	}
	view := compileContractJSONSchema(t, "node-metadata-view.schema.json")
	projection := map[string]any{"id": "encoder-one", "service_id": "encoder-one", "service_type": "encoder_recorder", "service_name": "Encoder", "metadata_revision": revision, "transport_mode": "direct_v1", "metrics": map[string]any{"cpu_percent": float64(4)}}
	if err := view.Validate(projection); err != nil {
		t.Fatal(err)
	}
	delete(projection, "metadata_revision")
	if err := view.Validate(projection); err != nil {
		t.Fatalf("legacy read projection rejected: %v", err)
	}
}
