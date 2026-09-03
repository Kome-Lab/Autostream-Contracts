package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"unicode/utf8"
)

var errNodeListenerConfigInvalid = errors.New("invalid node listener config")

// NodeListenerConfig is the non-secret, revision-bound listener projection for
// an application Node. It contains no identity credential or runtime secret.
type NodeListenerConfig struct {
	SchemaVersion  int    `json:"schema_version"`
	ServiceType    string `json:"service_type"`
	BindAddress    string `json:"bind_address"`
	ConfigRevision int64  `json:"config_revision"`
}

// Validate requires an explicit v2 application listener. BindAddress is the
// canonical netip.AddrPort spelling of a literal IP and an unprivileged port.
// Host names, interface zones, whitespace and implicit/default ports are invalid.
func (config NodeListenerConfig) Validate() error {
	if config.SchemaVersion != 2 || config.ConfigRevision < 1 {
		return errNodeListenerConfigInvalid
	}
	switch config.ServiceType {
	case "worker", "encoder_recorder", "discord_bot", "observability":
	default:
		return errNodeListenerConfigInvalid
	}
	address, err := netip.ParseAddrPort(config.BindAddress)
	if err != nil || address.Port() < 1024 || address.Addr().Zone() != "" || address.String() != config.BindAddress {
		return errNodeListenerConfigInvalid
	}
	return nil
}

// ParseNodeListenerConfig accepts exactly the four declared fields in one JSON
// object. Unknown, duplicate, missing, null, differently cased and trailing data
// fail closed; errors never include the input document or field values.
func ParseNodeListenerConfig(payload []byte) (NodeListenerConfig, error) {
	var config NodeListenerConfig
	if len(payload) > 4096 || !utf8.Valid(payload) {
		return NodeListenerConfig{}, errNodeListenerConfigInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return NodeListenerConfig{}, errNodeListenerConfigInvalid
	}
	seen := make(map[string]bool, 4)
	for decoder.More() {
		token, err := decoder.Token()
		field, ok := token.(string)
		if err != nil || !ok || seen[field] {
			return NodeListenerConfig{}, errNodeListenerConfigInvalid
		}
		seen[field] = true
		var destination any
		switch field {
		case "schema_version":
			destination = &config.SchemaVersion
		case "service_type":
			destination = &config.ServiceType
		case "bind_address":
			destination = &config.BindAddress
		case "config_revision":
			destination = &config.ConfigRevision
		default:
			return NodeListenerConfig{}, errNodeListenerConfigInvalid
		}
		if err := decoder.Decode(destination); err != nil {
			return NodeListenerConfig{}, errNodeListenerConfigInvalid
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || len(seen) != 4 {
		return NodeListenerConfig{}, errNodeListenerConfigInvalid
	}
	if _, err := decoder.Token(); err != io.EOF {
		return NodeListenerConfig{}, errNodeListenerConfigInvalid
	}
	if err := config.Validate(); err != nil {
		return NodeListenerConfig{}, err
	}
	return config, nil
}

// MarshalNodeListenerConfig returns the shared canonical digest input: compact
// JSON in schema_version, service_type, bind_address, config_revision order,
// followed by exactly one LF. Consumers hash these bytes, not an independent
// serialization. This fixed-field encoding is not the Updater command JCS format.
func MarshalNodeListenerConfig(config NodeListenerConfig) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	body, err := json.Marshal(config)
	if err != nil {
		return nil, errNodeListenerConfigInvalid
	}
	return append(body, '\n'), nil
}
