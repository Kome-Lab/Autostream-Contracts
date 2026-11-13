package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

// RequestDiagnostic is a closed, payload-free rejection classification. It is
// not a Request, execution qualification, authority, repair or retry instruction.
type RequestDiagnostic uint8

const (
	RequestDiagnosticNone            RequestDiagnostic = iota // existing DecodeRequest succeeded
	RequestDiagnosticInvalid                                  // structural/Command/unknown refusal
	RequestDiagnosticBindingInvalid                           // valid Command, invalid Binding representation
	RequestDiagnosticBindingConflict                          // valid Command, nonempty mismatching Binding
)

// DecodeRequestWithDiagnostic preserves DecodeRequest's success values and
// original error unchanged. A refusal ALWAYS returns a zero Request, even when
// diagnostics identify a Binding conflict. Diagnostics cannot accept or repair.
func DecodeRequestWithDiagnostic(data []byte) (Request, RequestDiagnostic, error) {
	request, err := DecodeRequest(data)
	if err == nil {
		return request, RequestDiagnosticNone, nil
	}
	return Request{}, diagnoseRequestRejection(data, err), err
}

func diagnoseRequestRejection(data []byte, original error) RequestDiagnostic {
	var failure *Error
	if !errors.As(original, &failure) || failure == nil ||
		(failure.Code != "encoding" && failure.Code != "request_binding") {
		return RequestDiagnosticInvalid
	}
	if len(data) == 0 || len(data) > MaxRequestBytes || !utf8.Valid(data) {
		return RequestDiagnosticInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	// Complete the closed shape AND trailing check before any semantic/Binding
	// classification. Binding may precede a later invalid Command or field.
	if !checkRequestDiagnosticShape(decoder) {
		return RequestDiagnosticInvalid
	}
	if _, err := decoder.Token(); err != io.EOF {
		return RequestDiagnosticInvalid
	}
	var wire requestWire
	if json.Unmarshal(data, &wire) != nil {
		return RequestDiagnosticInvalid
	}
	command := commandFromWire(wire.Command)
	if ValidateCommand(command) != nil {
		return RequestDiagnosticInvalid
	}
	// The only intentional diagnostic tolerance is a decoded RuneError in the
	// Binding string. Existing DecodeRequest still rejects it as encoding.
	// A contradictory/future source error must not become a Binding conflict.
	if (failure.Code == "encoding") != strings.ContainsRune(wire.Binding, utf8.RuneError) {
		return RequestDiagnosticInvalid
	}
	if len(wire.Binding) == 0 || len(wire.Binding) > MaxRequestBytes || !utf8.ValidString(wire.Binding) {
		return RequestDiagnosticBindingInvalid
	}
	var bindingFailure *Error
	if err := VerifyBinding(command, wire.Binding); !errors.As(err, &bindingFailure) ||
		bindingFailure == nil || bindingFailure.Code != "request_binding" {
		return RequestDiagnosticInvalid
	}
	return RequestDiagnosticBindingConflict
}

// This is Request-only rejection analysis, NOT an alternate normal decoder.
// It delegates the complete Command shape to the unchanged shared walker at
// exactly the existing Request depth (1). Only Binding's string token tolerates
// RuneError; all other fields/null/types/duplicates/required/depth rules remain.
func checkRequestDiagnosticShape(decoder *json.Decoder) bool {
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return false
	}
	commandSeen, bindingSeen := false, false
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		switch key {
		case "Command":
			if commandSeen {
				return false
			}
			commandSeen = true
			if checkValue(decoder, reflect.TypeOf(commandWire{}), 1) != nil {
				return false
			}
		case "Binding":
			if bindingSeen {
				return false
			}
			bindingSeen = true
			value, err := decoder.Token()
			if _, ok := value.(string); err != nil || !ok {
				return false
			}
		default:
			return false
		}
	}
	end, err := decoder.Token()
	return err == nil && end == json.Delim('}') && commandSeen && bindingSeen
}
