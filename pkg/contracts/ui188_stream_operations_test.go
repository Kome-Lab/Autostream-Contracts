package contracts

import (
	"strings"
	"testing"
)

func TestUI188CoverActionWireAndCandidateBoundaries(t *testing.T) {
	action := compileV2SchemaFragment(t, "stream-operations.schema.json", "CoverAction")
	valid := map[string]any{"active": true, "expected_job_generation": 1, "expected_revision": 2, "idempotency_key": "フタ切替-01", "candidate_variant_id": "variant-B", "expected_visual_revision": 3}
	assertV2SchemaFixture(t, action, valid, true)
	for name, value := range map[string]any{"hide_confirmed": false, "idempotency_key": strings.Repeat("a", 129), "expected_visual_revision": 0} {
		copy := cloneV2Fixture(t, valid)
		copy[name] = value
		assertV2SchemaFixture(t, action, copy, false)
	}
	copy := cloneV2Fixture(t, valid)
	delete(copy, "expected_visual_revision")
	assertV2SchemaFixture(t, action, copy, false)
	hide := map[string]any{"active": false, "expected_job_generation": 1, "expected_revision": 2, "idempotency_key": "hide-1", "hide_confirmed": true}
	assertV2SchemaFixture(t, action, hide, true)
	hide["hide_confirmed"] = false
	assertV2SchemaFixture(t, action, hide, false)
	candidate := compileV2SchemaFragment(t, "stream-operations.schema.json", "CoverCandidate")
	snapshot := map[string]any{"name": "保存済みフタ", "source": "preset", "preset_id": "preset-1", "preset_revision": 1, "asset_id": "asset-1", "variant_id": "variant-1"}
	assertV2SchemaFixture(t, candidate, snapshot, true)
	delete(snapshot, "preset_revision")
	assertV2SchemaFixture(t, candidate, snapshot, false)
}

func TestUI188PresetWriteDoesNotRequireServerOwnedMetadata(t *testing.T) {
	write := compileV2SchemaFragment(t, "stream-operations.schema.json", "PresetWrite")
	value := map[string]any{"name": "フタA", "asset_id": "asset-A", "asset_variant_id": "variant-A", "enabled": true}
	assertV2SchemaFixture(t, write, value, true)
	for _, field := range []string{"name", "asset_id", "asset_variant_id", "enabled"} {
		copy := cloneV2Fixture(t, value)
		delete(copy, field)
		assertV2SchemaFixture(t, write, copy, false)
	}
}
