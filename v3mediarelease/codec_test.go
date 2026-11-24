package mediarelease

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestRecordJSONBoundary(t *testing.T) {
	data, err := os.ReadFile("testdata/handoff.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	raw := fixtures["a1"]
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("standard JSON decode failed: %v", err)
	}
	encoded, err := json.Marshal(record)
	if err != nil || !bytes.Equal(raw, encoded) {
		t.Fatalf("standard JSON roundtrip changed the wire: %v", err)
	}
	for _, framed := range [][]byte{
		append([]byte{' '}, raw...),
		append(bytes.Clone(raw), '\n'),
		bytes.Replace(raw, []byte(`"identity"`), []byte(`"\u0069dentity"`), 1),
	} {
		if _, err := DecodeRecord(framed); err != ErrInvalidWire {
			t.Fatalf("noncanonical transport frame accepted or wrong error: %v", err)
		}
	}
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &record); err == nil {
		t.Fatal("accepted malformed replacement")
	}
	saved, err := EncodeRecord(record)
	if err != nil || !bytes.Equal(raw, saved) {
		t.Fatal("failed decode changed the receiver")
	}
}

// This encoding-only vector has a literal expected digest computed outside
// the implementation. It is not a record or an authority fixture.
func TestCanonicalDomainVector(t *testing.T) {
	preimage := []byte(`{"a":"1","z":["left","right"]}`)
	const want = "0460b03eaaa802158e417554035e6f54849a53f68b3cb03f06d15d6a3c4db290"
	out, err := CanonicalBytes(map[string]any{"z": []string{"left", "right"}, "a": "1"})
	if err != nil || !bytes.Equal(out, preimage) {
		t.Fatalf("canonical bytes differ: %v", err)
	}
	if got := digest("MediaRelease1/identity/v1", preimage); got != want {
		t.Fatalf("identity digest %s, want %s", got, want)
	}
	if digest("MediaRelease1/record/v1", preimage) == want || digest("MediaRelease1/open/v1", preimage) == want {
		t.Fatal("digest domains overlap")
	}
}

func TestCanonicalSubset(t *testing.T) {
	cases := map[string]string{
		"duplicate":         `{"a":"b","a":"b"}`,
		"nested_duplicate":  `{"a":[{"b":"1","b":"2"}]}`,
		"escaped_duplicate": `{"a":"1","\u0061":"1"}`,
		"number":            `{"a":1}`,
		"exponent":          `{"a":1e1}`,
		"null":              `{"a":null}`,
		"truncated":         `{"a":["1"}`,
		"trailing_value":    `{"a":"b"}true`,
		"unknown_unicode":   `{"a":"\u2603"}`,
		"lone_surrogate":    `{"a":"\udc00"}`,
		"non_ascii":         `{"a":"雪"}`,
		"wrong_key_case":    `{"Kind":"x"}`,
		"invalid_id_text":   `{"a":"space in identifier"}`,
		"bad_utf8":          "{\"a\":\"" + string([]byte{0xc0, 0x80}) + "\"}",
		"bom":               "\xef\xbb\xbf" + `{"a":"1"}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := canonicalJSON([]byte(input)); err != ErrInvalidWire {
				t.Fatalf("want fixed wire rejection, got %v", err)
			}
		})
	}

	input := `{"z":false,"a":[{"z":"b","a":"c"},"b","a"],"x":true}`
	const want = `{"a":[{"a":"c","z":"b"},"b","a"],"x":true,"z":false}`
	if got, err := canonicalJSON([]byte(input)); err != nil || string(got) != want {
		t.Fatalf("nested ordering or input array order changed: %v", err)
	}
}

func TestCanonicalLimits(t *testing.T) {
	keys := func(n int) string {
		members := make([]string, n)
		for i := range members {
			members[i] = fmt.Sprintf(`"a%02d":"x"`, i)
		}
		return "{" + strings.Join(members, ",") + "}"
	}
	array := func(n int) string {
		return "[" + strings.TrimSuffix(strings.Repeat(`"a",`, n), ",") + "]"
	}
	nested := func(n int) string {
		return strings.Repeat("[", n) + `"x"` + strings.Repeat("]", n)
	}
	for _, tc := range []struct {
		name  string
		input string
		valid bool
	}{
		{"members_64", keys(64), true},
		{"members_65", keys(65), false},
		{"array_64", array(64), true},
		{"array_65", array(65), false},
		{"depth_16", nested(16), true},
		{"depth_17", nested(17), false},
		{"id_192", `"` + strings.Repeat("a", 192) + `"`, true},
		{"id_193", `"` + strings.Repeat("a", 193) + `"`, false},
		{"oversize", strings.Repeat(" ", 64*1024) + `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := canonicalJSON([]byte(tc.input))
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected limit result: %v", err)
			}
		})
	}
}
