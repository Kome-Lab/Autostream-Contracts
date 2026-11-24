package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func diagnosticTestWire(command, binding string, first bool) []byte {
	if first {
		return []byte(`{"Binding":` + binding + `,"Command":` + command + `}`)
	}
	return []byte(`{"Command":` + command + `,"Binding":` + binding + `}`)
}

func diagnosticTestCompare(t *testing.T, raw []byte, want RequestDiagnostic) {
	t.Helper()
	before := append([]byte(nil), raw...)
	direct, oldErr := DecodeRequest(raw)
	got, diagnostic, err := DecodeRequestWithDiagnostic(raw)
	if diagnostic != want || !bytes.Equal(before, raw) {
		t.Fatal("diagnostic mismatch or input mutation", diagnostic, want)
	}
	if oldErr == nil {
		if err != nil || diagnostic != RequestDiagnosticNone || !reflect.DeepEqual(got, direct) {
			t.Fatal("success did not come from existing decoder", err)
		}
		return
	}
	var a, b *Error
	if err == nil || got != (Request{}) || !errors.As(oldErr, &a) || !errors.As(err, &b) ||
		a.Code != b.Code || oldErr.Error() != err.Error() {
		t.Fatal("original refusal/error changed", oldErr, err)
	}
	if diagnostic == RequestDiagnosticNone {
		t.Fatal("refusal became a success diagnostic")
	}
}

func TestRequestDiagnosticSuccessAndOriginalErrors(t *testing.T) {
	r := requestTestRequest(t, requestTestCommand())
	raw := requestTestWire(t, r)
	var wire requestWire
	if json.Unmarshal(raw, &wire) != nil {
		t.Fatal("fixture")
	}
	binding, _ := json.Marshal(wire.Binding)
	for _, data := range [][]byte{raw, diagnosticTestWire(requestTestGoldenCommand, string(binding), true),
		bytes.Replace(raw, []byte(`"Command":`), []byte(`"Comman\u0064":`), 1)} {
		diagnosticTestCompare(t, data, RequestDiagnosticNone)
	}
	for _, tc := range []struct {
		value string
		want  RequestDiagnostic
	}{
		{`""`, RequestDiagnosticBindingInvalid}, {`"other"`, RequestDiagnosticBindingConflict},
		{`null`, RequestDiagnosticInvalid}, {`1`, RequestDiagnosticInvalid},
		{`{}`, RequestDiagnosticInvalid}, {`[]`, RequestDiagnosticInvalid}, {`false`, RequestDiagnosticInvalid},
	} {
		diagnosticTestCompare(t, diagnosticTestWire(requestTestGoldenCommand, tc.value, false), tc.want)
	}
	if reflect.TypeOf(Error{}).NumField() != 1 || reflect.TypeOf(Error{}).Field(0).Name != "Code" {
		t.Fatal("existing Error shape changed")
	}
}

func TestRequestDiagnosticCompleteShapeThenCommandThenBinding(t *testing.T) {
	for _, binding := range []struct{ name, value string }{
		{"literal", string([]byte{'"', 0xef, 0xbf, 0xbd, '"'})},
		{"escaped", `"\ufffd"`}, {"high", `"\ud800"`}, {"low", `"\udfff"`},
	} {
		for _, first := range []bool{false, true} {
			order := "command-first"
			if first {
				order = "binding-first"
			}
			prefix := binding.name + "/" + order
			t.Run(prefix+"/only", func(t *testing.T) {
				diagnosticTestCompare(t, diagnosticTestWire(requestTestGoldenCommand, binding.value, first), RequestDiagnosticBindingConflict)
			})
			for _, change := range []struct{ name, old, new string }{
				{"unknown-command", `"Schema":1`, `"private-key":"secret","Schema":1`},
				{"unknown-scope", `"Realm":"realm-1"`, `"private-key":0,"Realm":"realm-1"`},
				{"duplicate", `"Schema":1`, `"Schema":1,"Schema":1`},
				{"escaped-duplicate", `"Schema":1`, `"Schema":1,"\u0053chema":1`},
				{"missing", `,"StateVersion":7`, ""},
				{"identity", `"Realm":"realm-1"`, `"Realm":"UPPER"`},
				{"unicode-command", `"Realm":"realm-1"`, `"Realm":"\ufffd"`},
				{"schema", `"Schema":1`, `"Schema":2`},
				{"operation", `"Operation":"registry.reference_only"`, `"Operation":"shell.run"`},
				{"schema-overflow", `"Schema":1`, `"Schema":4294967296`},
				{"max-state", `"StateVersion":7`, `"StateVersion":18446744073709551615`},
				{"state-overflow", `"StateVersion":7`, `"StateVersion":18446744073709551616`},
				{"deadline-zero", `"DeadlineTick":100`, `"DeadlineTick":0`},
				{"fraction", `"DeadlineTick":100`, `"DeadlineTick":1.0`},
				{"exponent", `"DeadlineTick":100`, `"DeadlineTick":1e2`},
				{"signed", `"StateVersion":7`, `"StateVersion":-0`},
				{"null", `"Schema":1`, `"Schema":null`},
				{"type", `"Schema":1`, `"Schema":"1"`},
			} {
				t.Run(prefix+"/"+change.name, func(t *testing.T) {
					command := strings.Replace(requestTestGoldenCommand, change.old, change.new, 1)
					if command == requestTestGoldenCommand {
						t.Fatal("fixture mutation missed field")
					}
					diagnosticTestCompare(t, diagnosticTestWire(command, binding.value, first), RequestDiagnosticInvalid)
				})
			}
			raw := string(diagnosticTestWire(requestTestGoldenCommand, binding.value, first))
			for _, change := range []struct{ name, wire string }{
				{"unknown-root", strings.TrimSuffix(raw, "}") + `,"private-key":"secret"}`},
				{"duplicate-root", strings.TrimSuffix(raw, "}") + `,"Command":` + requestTestGoldenCommand + `}`},
				{"binding-alias", strings.TrimSuffix(raw, "}") + `,"\u0042inding":"other"}`},
				{"missing-command", `{"Binding":` + binding.value + `}`},
				{"trailing-object", raw + "{}"}, {"trailing-scalar", raw + "0"},
			} {
				t.Run(prefix+"/"+change.name, func(t *testing.T) { diagnosticTestCompare(t, []byte(change.wire), RequestDiagnosticInvalid) })
			}
		}
	}
}

func TestRequestDiagnosticUnknownAndContradictoryErrors(t *testing.T) {
	valid := requestTestWire(t, requestTestRequest(t, requestTestCommand()))
	unicode := diagnosticTestWire(requestTestGoldenCommand, `"\ufffd"`, true)
	var typedNil *Error
	for _, err := range []error{nil, typedNil, errors.New("request_binding"), &Error{Code: "future-private-code"},
		&Error{Code: "request_version"}, &Error{Code: "request_binding"}, &Error{Code: "encoding"}} {
		if diagnoseRequestRejection(valid, err) != RequestDiagnosticInvalid {
			t.Fatal("unknown/contradictory refusal became Binding conflict")
		}
	}
	if diagnoseRequestRejection(unicode, &Error{Code: "request_binding"}) != RequestDiagnosticInvalid {
		t.Fatal("contradictory original Code accepted")
	}
	if diagnoseRequestRejection(unicode, &Error{Code: "encoding"}) != RequestDiagnosticBindingConflict {
		t.Fatal("qualified conflict was lost")
	}
}

func TestRequestDiagnosticBoundsAndRedaction(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("{"), []byte("{}"), []byte{0xff},
		bytes.Repeat([]byte(" "), MaxRequestBytes+1),
		[]byte(`{"Binding":` + strings.Repeat("[", MaxRequestDepth+1) + `"\ufffd"` + strings.Repeat("]", MaxRequestDepth+1) + `,"Command":` + requestTestGoldenCommand + `}`)} {
		diagnosticTestCompare(t, raw, RequestDiagnosticInvalid)
	}
	secret := "private-request-binding-secret"
	raw := diagnosticTestWire(requestTestGoldenCommand, `"`+secret+`\ufffd"`, true)
	request, diagnostic, err := DecodeRequestWithDiagnostic(raw)
	if request != (Request{}) || err == nil || diagnostic != RequestDiagnosticBindingConflict {
		t.Fatal("redaction fixture")
	}
	if reflect.TypeOf(diagnostic).Kind() != reflect.Uint8 || strings.Contains(err.Error(), secret) {
		t.Fatal("diagnostic leaked input")
	}
	if got, _ := json.Marshal(diagnostic); string(got) != "3" {
		t.Fatal("diagnostic is not a fixed classification")
	}
}

func TestRequestDiagnosticEnvelopeAndTypedAPIsRemainStrict(t *testing.T) {
	for _, kind := range evidenceTestKinds() {
		raw := evidenceTestRaw(kind)
		value := evidenceTestDecode(t, raw)
		if Validate(value) != nil {
			t.Fatal("existing Envelope changed")
		}
		encoded, err := Encode(value)
		if err != nil {
			t.Fatal(err)
		}
		bad := bytes.Replace(encoded, []byte(`"binding":"synthetic-immutable-binding"`), []byte(`"binding":"\ufffd"`), 1)
		if bytes.Equal(bad, encoded) {
			t.Fatal("fixture")
		}
		if _, err = Decode(bad); err == nil {
			t.Fatal("Request diagnostics relaxed Envelope decoder")
		}
	}
	c := requestTestCommand()
	binding, err := ComputeBinding(c)
	if err != nil || binding != requestTestGoldenCommand {
		t.Fatal("canonical bytes changed")
	}
	c.Expected.StateVersion = ^uint64(0)
	if ValidateCommand(c) == nil || ValidateRequest(Request{Command: c, Binding: binding}) == nil {
		t.Fatal("typed validation relaxed")
	}
}

func TestRequestDiagnosticConcurrentDeterminism(t *testing.T) {
	raw := diagnosticTestWire(requestTestGoldenCommand, `"\ud800"`, true)
	var wg sync.WaitGroup
	failures := make(chan bool, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				r, d, e := DecodeRequestWithDiagnostic(raw)
				if r != (Request{}) || d != RequestDiagnosticBindingConflict || e == nil {
					failures <- true
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	if len(failures) != 0 {
		t.Fatal("shared diagnostic state changed")
	}
}
