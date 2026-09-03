package contracts

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNodeListenerConfigCanonicalBytes(t *testing.T) {
	const canonical = "{\"schema_version\":2,\"service_type\":\"worker\",\"bind_address\":\"127.0.0.1:3010\",\"config_revision\":7}\n"
	config, err := ParseNodeListenerConfig([]byte(` {"config_revision":7,"bind_address":"127.0.0.1:3010","service_type":"worker","schema_version":2} `))
	if err != nil {
		t.Fatal(err)
	}
	if config != (NodeListenerConfig{SchemaVersion: 2, ServiceType: "worker", BindAddress: "127.0.0.1:3010", ConfigRevision: 7}) {
		t.Fatalf("unexpected listener projection: %#v", config)
	}
	body, err := MarshalNodeListenerConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != canonical {
		t.Fatalf("canonical listener bytes = %q, want %q", body, canonical)
	}
	roundTrip, err := ParseNodeListenerConfig(body)
	if err != nil || roundTrip != config {
		t.Fatalf("canonical round trip changed projection: %#v, %v", roundTrip, err)
	}
}

func TestNodeListenerConfigLiteralAddressAndRevisionBoundaries(t *testing.T) {
	for _, service := range []string{"worker", "encoder_recorder", "discord_bot", "observability"} {
		for _, address := range []string{"127.0.0.1:1024", "0.0.0.0:65535", "192.0.2.7:3010", "[::1]:3010", "[::]:3010", "[2001:db8::1]:3010"} {
			config := NodeListenerConfig{SchemaVersion: 2, ServiceType: service, BindAddress: address, ConfigRevision: 1}
			body, err := MarshalNodeListenerConfig(config)
			if err != nil {
				t.Fatalf("valid %s listener %s rejected: %v", service, address, err)
			}
			parsed, err := ParseNodeListenerConfig(body)
			if err != nil || parsed != config {
				t.Fatalf("valid listener round trip failed: %#v, %v", parsed, err)
			}
		}
	}
	valid := NodeListenerConfig{SchemaVersion: 2, ServiceType: "worker", BindAddress: "127.0.0.1:3010", ConfigRevision: 1}
	invalid := []NodeListenerConfig{}
	for _, address := range []string{"", "localhost:3010", ":3010", "127.0.0.1", "127.0.0.1:0", "127.0.0.1:1023", "127.0.0.1:65536", "127.0.0.1:03010", " 127.0.0.1:3010", "127.0.0.1:3010\n", "127.000.0.1:3010", "[fe80::1%eth0]:3010", "[2001:DB8::1]:3010", "[0:0:0:0:0:0:0:1]:3010", "::1:3010"} {
		candidate := valid
		candidate.BindAddress = address
		invalid = append(invalid, candidate)
	}
	for _, service := range []string{"", "control_panel", "update_agent", "Worker", "worker "} {
		candidate := valid
		candidate.ServiceType = service
		invalid = append(invalid, candidate)
	}
	for _, version := range []int{0, 1, 3} {
		candidate := valid
		candidate.SchemaVersion = version
		invalid = append(invalid, candidate)
	}
	for _, revision := range []int64{-1, 0} {
		candidate := valid
		candidate.ConfigRevision = revision
		invalid = append(invalid, candidate)
	}
	for _, config := range invalid {
		if err := config.Validate(); err == nil {
			t.Fatalf("invalid listener accepted: %#v", config)
		}
		if body, err := MarshalNodeListenerConfig(config); err == nil || body != nil {
			t.Fatalf("invalid listener was serialized: %#v", config)
		}
		body, err := json.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		if parsed, err := ParseNodeListenerConfig(body); err == nil || parsed != (NodeListenerConfig{}) {
			t.Fatalf("invalid listener bytes accepted: %#v", config)
		}
	}
}

func TestNodeListenerConfigRejectsAmbiguousJSON(t *testing.T) {
	const valid = `{"schema_version":2,"service_type":"worker","bind_address":"127.0.0.1:3010","config_revision":1}`
	invalid := []string{"", "null", "[]", `"value"`, "{}", valid + "{}", valid + "null", valid + " trailing"}
	for _, field := range []string{"schema_version", "service_type", "bind_address", "config_revision"} {
		var document map[string]any
		if err := json.Unmarshal([]byte(valid), &document); err != nil {
			t.Fatal(err)
		}
		delete(document, field)
		body, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		invalid = append(invalid, string(body))
		document[field] = nil
		body, err = json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		invalid = append(invalid, string(body))
	}
	invalid = append(invalid,
		strings.Replace(valid, `"schema_version":2`, `"schema_version":2,"schema_version":2`, 1),
		strings.Replace(valid, `"schema_version":2`, `"Schema_Version":2`, 1),
		strings.Replace(valid, `"schema_version":2`, `"schema_version":"2"`, 1),
		strings.Replace(valid, `"config_revision":1`, `"config_revision":1.5`, 1),
		strings.Replace(valid, `"config_revision":1`, `"config_revision":9223372036854775808`, 1),
		strings.Replace(valid, `"config_revision":1`, `"config_revision":1,"runtime_token":"must-not-be-reflected"`, 1),
		strings.Replace(valid, `"service_type":"worker"`, `"service_type":[]`, 1),
	)
	for index, input := range invalid {
		parsed, err := ParseNodeListenerConfig([]byte(input))
		if err == nil || parsed != (NodeListenerConfig{}) {
			t.Fatalf("ambiguous JSON case %d accepted", index)
		}
		if strings.Contains(err.Error(), "must-not-be-reflected") {
			t.Fatal("parser error reflected input")
		}
	}
	for _, input := range [][]byte{append([]byte(valid), 0xff), bytes.Repeat([]byte(" "), 4097)} {
		if _, err := ParseNodeListenerConfig(input); err == nil {
			t.Fatal("invalid encoding or oversized document accepted")
		}
	}
}

func TestNodeListenerConfigSchemaHasExactNonSecretShape(t *testing.T) {
	schema := compileContractJSONSchema(t, "node-listener-config.schema.json")
	valid := map[string]any{
		"schema_version": 2, "service_type": "worker",
		"bind_address": "127.0.0.1:3010", "config_revision": 1,
	}
	assertV2SchemaFixture(t, schema, valid, true)
	for field := range valid {
		missing := cloneV2Fixture(t, valid)
		delete(missing, field)
		assertV2SchemaFixture(t, schema, missing, false)
		null := cloneV2Fixture(t, valid)
		null[field] = nil
		assertV2SchemaFixture(t, schema, null, false)
	}
	for field, value := range map[string]any{
		"schema_version": 1, "service_type": "update_agent", "config_revision": 0,
		"runtime_token": "synthetic-forbidden", "command": "synthetic-forbidden",
	} {
		candidate := cloneV2Fixture(t, valid)
		candidate[field] = value
		assertV2SchemaFixture(t, schema, candidate, false)
	}
	for _, address := range []string{"0.0.0.0:1024", "127.0.0.1:65535", "[::1]:3010", "[::]:3010", "[2001:db8::1]:3010"} {
		candidate := cloneV2Fixture(t, valid)
		candidate["bind_address"] = address
		assertV2SchemaFixture(t, schema, candidate, true)
	}
	for _, address := range []string{"localhost:3010", ":3010", "127.0.0.1:0", "127.0.0.1:1023", "127.0.0.1:65536", "127.0.0.1:03010", "127.000.0.1:3010", "127.0.0.1:3010\n", "[fe80::1%eth0]:3010", "[2001:DB8::1]:3010"} {
		candidate := cloneV2Fixture(t, valid)
		candidate["bind_address"] = address
		assertV2SchemaFixture(t, schema, candidate, false)
	}
}
