//go:build request_api_compat

package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type legacyAPIResult struct {
	Name    string
	Code    string
	Message string
	Value   []byte
}

// Identical fixture for reviewed 790be and candidate; ONLY pre-existing APIs.
func TestRequestLegacyAPICompatibilitySnapshot(t *testing.T) {
	target := os.Getenv("REQUEST_API_SNAPSHOT")
	if !filepath.IsAbs(target) {
		t.Fatal("explicit synthetic snapshot path required")
	}
	var rows []legacyAPIResult
	add := func(name string, value []byte, err error) {
		row := legacyAPIResult{Name: name, Value: value}
		if err != nil {
			row.Message = err.Error()
			var e *Error
			if errors.As(err, &e) && e != nil {
				row.Code = e.Code
			} else {
				row.Code = "untyped"
			}
		}
		rows = append(rows, row)
	}
	c := requestTestCommand()
	r := requestTestRequest(t, c)
	normal := requestTestWire(t, r)
	command := requestTestGoldenCommand
	wire := func(cmd, binding string) []byte { return []byte(`{"Binding":` + binding + `,"Command":` + cmd + `}`) }
	cases := []struct {
		name string
		raw  []byte
	}{
		{"normal", normal},
		{"empty", wire(command, `""`)}, {"mismatch", wire(command, `"other"`)},
		{"literal-replacement", wire(command, string([]byte{'"', 0xef, 0xbf, 0xbd, '"'}))},
		{"escaped-replacement", wire(command, `"\ufffd"`)},
		{"high-surrogate", wire(command, `"\ud800"`)}, {"low-surrogate", wire(command, `"\udfff"`)},
		{"null", wire(command, `null`)}, {"type", wire(command, `{}`)},
		{"unknown-after-binding", []byte(`{"Binding":"\ufffd","Command":` + command + `,"unknown":"secret"}`)},
		{"duplicate", bytes.Replace(normal, []byte(`"Schema":1`), []byte(`"Schema":1,"Schema":1`), 1)},
		{"alias", bytes.Replace(normal, []byte(`"Schema":1`), []byte(`"Schema":1,"\u0053chema":1`), 1)},
		{"missing", bytes.Replace(normal, []byte(`,"StateVersion":7`), nil, 1)},
		{"schema", bytes.Replace(normal, []byte(`"Schema":1`), []byte(`"Schema":2`), 1)},
		{"operation", bytes.Replace(normal, []byte(`"Operation":"registry.reference_only"`), []byte(`"Operation":"other"`), 1)},
		{"identity", bytes.Replace(normal, []byte(`"Realm":"realm-1"`), []byte(`"Realm":"UPPER"`), 1)},
		{"number", bytes.Replace(normal, []byte(`"StateVersion":7`), []byte(`"StateVersion":18446744073709551615`), 1)},
		{"fraction", bytes.Replace(normal, []byte(`"Schema":1`), []byte(`"Schema":1.0`), 1)},
		{"trailing", append(append([]byte(nil), normal...), []byte("{}")...)},
		{"utf8", []byte{0xff}}, {"size", bytes.Repeat([]byte(" "), MaxRequestBytes+1)},
	}
	for _, tc := range cases {
		got, err := DecodeRequest(tc.raw)
		var encoded []byte
		if err == nil {
			encoded, _ = EncodeRequest(got)
		}
		add("DecodeRequest/"+tc.name, encoded, err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Command)
	}{
		{"normal", func(*Command) {}}, {"schema", func(c *Command) { c.Schema = 2 }},
		{"operation", func(c *Command) { c.Operation = "other" }},
		{"identity", func(c *Command) { c.Scope.Host = "UPPER" }},
		{"number", func(c *Command) { c.Expected.StateVersion = ^uint64(0) }},
		{"deadline", func(c *Command) { c.Timing.DeadlineTick = 0 }},
	} {
		got := c
		tc.change(&got)
		add("ValidateCommand/"+tc.name, nil, ValidateCommand(got))
		value, err := CanonicalBytes(got)
		add("CanonicalBytes/"+tc.name, value, err)
		binding, err := ComputeBinding(got)
		add("ComputeBinding/"+tc.name, []byte(binding), err)
		typed := Request{Command: got, Binding: r.Binding}
		add("ValidateRequest/"+tc.name, nil, ValidateRequest(typed))
		add("VerifyBinding/"+tc.name, nil, VerifyBinding(got, r.Binding))
		value, err = EncodeRequest(typed)
		add("EncodeRequest/"+tc.name, value, err)
	}
	add("ValidateRequestFor/normal", nil, ValidateRequestFor(r, c.Scope, c.Execution))
	other := c.Scope
	other.Host = "other"
	add("ValidateRequestFor/mismatch", nil, ValidateRequestFor(r, other, c.Execution))
	form, err := Canonicalize(c)
	add("Canonicalize", form.Bytes, err)
	add("VerifyCanonicalForm", nil, VerifyCanonicalForm(c, form))
	form.Binding = "other"
	add("VerifyCanonicalForm/mismatch", nil, VerifyCanonicalForm(c, form))
	for _, kind := range evidenceTestKinds() {
		got, err := Decode(evidenceTestRaw(kind))
		var value []byte
		if err == nil {
			value, _ = Encode(got)
		}
		add("Envelope/"+string(kind), value, err)
		add("EnvelopeValidate/"+string(kind), nil, Validate(got))
		cloned, err := Encode(Clone(got))
		add("EnvelopeClone/"+string(kind), cloned, err)
		bad := bytes.Replace(evidenceTestRaw(kind), []byte(`"binding":"synthetic-immutable-binding"`), []byte(`"binding":"\ufffd"`), 1)
		_, err = Decode(bad)
		add("EnvelopeUnicode/"+string(kind), nil, err)
	}
	value, err := json.Marshal(r)
	add("MarshalRequest", value, err)
	var decoded Request
	err = json.Unmarshal(normal, &decoded)
	add("UnmarshalRequest", nil, err)
	value, err = json.Marshal(c)
	add("MarshalCommand", value, err)
	var decodedCommand Command
	err = json.Unmarshal([]byte(command), &decodedCommand)
	add("UnmarshalCommand", nil, err)
	value, err = json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, value, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("original API snapshot: %d deterministic observations", len(rows))
}
