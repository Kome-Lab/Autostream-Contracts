package mediarelease

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"unicode/utf8"
)

const (
	maxWireBytes   = 64 * 1024
	maxWireDepth   = 16
	maxWireMembers = 64
)

type wireError string

func (e wireError) Error() string { return string(e) }

// These errors do not echo rejected input, which can include unknown fields.
const (
	ErrInvalidWire    wireError = "media release: invalid canonical wire"
	ErrDigestMismatch wireError = "media release: record digest mismatch"
)

// wireRecord avoids recursive calls to Record's standard JSON methods.
type wireRecord Record

// CanonicalBytes encodes the JSON value subset used by MediaRelease1. Object
// members sort by ASCII key, while array order is preserved. Kind-specific
// field and set rules are checked separately by ValidateFields.
func CanonicalBytes(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrInvalidWire
	}
	return canonicalJSON(raw)
}

func canonicalJSON(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > maxWireBytes || !utf8.Valid(raw) {
		return nil, ErrInvalidWire
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := readValue(decoder, 0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, ErrInvalidWire
	}
	// All keys and strings are in the restricted ASCII alphabet. The JSON
	// encoder sorts map keys and emits these strings without escape sequences.
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, ErrInvalidWire
	}
	return canonical, nil
}

func readValue(d *json.Decoder, depth int) (any, error) {
	token, err := d.Token()
	if err != nil {
		return nil, ErrInvalidWire
	}
	switch value := token.(type) {
	case bool:
		return value, nil
	case string:
		if !wireString(value) {
			return nil, ErrInvalidWire
		}
		return value, nil
	case json.Delim:
		if depth >= maxWireDepth {
			return nil, ErrInvalidWire
		}
		switch value {
		case '{':
			return readObject(d, depth+1)
		case '[':
			return readArray(d, depth+1)
		}
	}
	return nil, ErrInvalidWire
}

func readObject(d *json.Decoder, depth int) (map[string]any, error) {
	object := make(map[string]any)
	for d.More() {
		if len(object) == maxWireMembers {
			return nil, ErrInvalidWire
		}
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok || !wireKey(key) {
			return nil, ErrInvalidWire
		}
		if _, exists := object[key]; exists {
			return nil, ErrInvalidWire
		}
		object[key], err = readValue(d, depth)
		if err != nil {
			return nil, err
		}
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') {
		return nil, ErrInvalidWire
	}
	return object, nil
}

func readArray(d *json.Decoder, depth int) ([]any, error) {
	array := make([]any, 0)
	for d.More() {
		if len(array) == maxWireMembers {
			return nil, ErrInvalidWire
		}
		value, err := readValue(d, depth)
		if err != nil {
			return nil, err
		}
		array = append(array, value)
	}
	if end, err := d.Token(); err != nil || end != json.Delim(']') {
		return nil, ErrInvalidWire
	}
	return array, nil
}

func wireString(s string) bool {
	if len(s) == 0 || len(s) > 192 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '.', '_', ':', '/', '-':
		default:
			return false
		}
	}
	return true
}

func wireKey(s string) bool {
	if len(s) == 0 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func digest(domain string, canonical []byte) string {
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write([]byte{0})
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil))
}

// OpenRequestDigest computes the request binding for a consumption record.
// It does not authorize or perform an open.
func OpenRequestDigest(i Identity) (string, error) {
	body, err := CanonicalBytes(struct {
		ConnectionOperationID string `json:"connection_operation_id"`
		LogicalEffectKey      string `json:"logical_effect_key"`
		PermitRef             *Ref   `json:"permit_ref"`
		ReceiptKey            *Key   `json:"receipt_key"`
	}{i.ConnectionOperationID, i.LogicalEffectKey, i.PermitRef, i.ReceiptKey})
	if err != nil {
		return "", err
	}
	return digest("MediaRelease1/open/v1", body), nil
}

// Digests computes both domain-separated digests of a structurally valid
// record. Neither digest covers its own field or transport credentials.
func Digests(r Record) (immutable, record string, err error) {
	if err := ValidateFields(r.Identity, r.Snapshot); err != nil {
		return "", "", err
	}
	identity, err := CanonicalBytes(r.Identity)
	if err != nil {
		return "", "", err
	}
	body, err := CanonicalBytes(struct {
		Identity Identity `json:"identity"`
		Snapshot Snapshot `json:"snapshot"`
	}{r.Identity, r.Snapshot})
	if err != nil {
		return "", "", err
	}
	// The additional two envelope members always carry 64 lowercase hex
	// characters each. Reject an oversized frame before returning digests.
	const digestMembers = len(`,"immutable_digest":"","record_digest":""`) + 4*sha256.Size
	if len(body)+digestMembers > maxWireBytes {
		return "", "", ErrInvalidWire
	}
	return digest("MediaRelease1/identity/v1", identity), digest("MediaRelease1/record/v1", body), nil
}

// EncodeRecord verifies both supplied digests and returns exact canonical
// envelope bytes. It never repairs or silently replaces a supplied digest.
func EncodeRecord(r Record) ([]byte, error) {
	immutable, record, err := Digests(r)
	if err != nil {
		return nil, err
	}
	if immutable != r.ImmutableDigest || record != r.RecordDigest {
		return nil, ErrDigestMismatch
	}
	return CanonicalBytes(wireRecord(r))
}

// ValidateRecord checks the closed schema, canonical size and both digests.
// Authentication, durable readback and owner observations are independent.
func ValidateRecord(r Record) error {
	_, err := EncodeRecord(r)
	return err
}

// DecodeRecord accepts one bounded, byte-canonical record. It never applies
// defaults or converts legacy or future records into this family.
func DecodeRecord(data []byte) (Record, error) {
	canonical, err := canonicalJSON(data)
	if err != nil || !bytes.Equal(data, canonical) {
		return Record{}, ErrInvalidWire
	}
	var wire wireRecord
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return Record{}, ErrInvalidWire
	}
	r := Record(wire)
	encoded, err := EncodeRecord(r)
	if err != nil {
		return Record{}, err
	}
	// Also catches differently spelled keys matched by encoding/json and
	// present-but-empty optional fields discarded by the typed encoding.
	if !bytes.Equal(data, encoded) {
		return Record{}, ErrInvalidWire
	}
	return r, nil
}

// CompareImmutable compares exact validated identity bytes. Snapshot revisions
// remain a separate CAS operand even when identities match.
func CompareImmutable(a, b Record) (bool, error) {
	if err := ValidateRecord(a); err != nil {
		return false, err
	}
	if err := ValidateRecord(b); err != nil {
		return false, err
	}
	left, err := CanonicalBytes(a.Identity)
	if err != nil {
		return false, err
	}
	right, err := CanonicalBytes(b.Identity)
	return bytes.Equal(left, right), err
}

// MarshalJSON preserves canonical wire bytes for standard JSON callers.
func (r Record) MarshalJSON() ([]byte, error) { return EncodeRecord(r) }

// UnmarshalJSON applies the same closed decoder and preserves the receiver on
// failure. Transports must pass the original untrimmed body to DecodeRecord
// when frame-level whitespace and trailing bytes must also be checked.
func (r *Record) UnmarshalJSON(data []byte) error {
	if r == nil {
		return ErrInvalidWire
	}
	value, err := DecodeRecord(data)
	if err != nil {
		return err
	}
	*r = value
	return nil
}
