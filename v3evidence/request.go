package evidence

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// These are separate version domains. Schema1 explicitly selects binding1;
// Evidence ProtocolVersion3 does not select a Request or Binding version.
// No new wire field/prefix/hash is added to C10's existing v1 representation.
const (
	CanonicalRequestVersion uint32 = 1
	CanonicalBindingVersion uint32 = 1
	MaxRequestBytes                = 64 << 10
	MaxRequestDepth                = maxEvidenceJSONDepth // shared strict token walker; fixed v1 shape is shallower
)

// ExpectedReference's state version is not a candidate/configuration revision.
// Zero is valid in the existing C10 schema; MaxUint64 cannot be incremented.
type ExpectedReference struct {
	Revision     string `json:"Revision"`
	StateVersion uint64 `json:"StateVersion"`
}
type RequiredGuard struct {
	Policy             string `json:"Policy"`
	Trust              string `json:"Trust"`
	Membership         string `json:"Membership"`
	OwnershipEpoch     string `json:"OwnershipEpoch"`
	FenceEpoch         string `json:"FenceEpoch"`
	Generation         string `json:"Generation"`
	BootID             string `json:"BootID"`
	ProcessIncarnation string `json:"ProcessIncarnation"`
}
type CommandTiming struct {
	TimeBase     string `json:"TimeBase"`
	DeadlineTick uint64 `json:"DeadlineTick"`
}

// Command reuses the Evidence Scope/Execution semantic types. Its canonical
// encoding deliberately differs from their snake_case Evidence serialization.
// Do not regroup Authorization/Candidate fields or add later evidence here.
type Command struct {
	Schema              uint32
	Operation           string
	Scope               Scope
	Execution           Execution
	RequestingPrincipal string
	AuthorizationIssuer string
	CandidateRevision   string
	CandidateContentPin string
	Expected            ExpectedReference
	RequiredGuard       RequiredGuard
	Timing              CommandTiming
	OriginalBudget      string
}

// Binding is an exact equality representation, NOT a digest, authenticated
// principal, signature, consumed authorization or permission to mutate.
type Binding = string
type Request struct {
	Command Command
	Binding Binding
}

// CanonicalForm is an optional explicit-version storage/comparison companion.
// It is NOT part of the legacy Request wire or an Evidence payload. Opaque
// Envelope.Binding alone cannot prove the caller selected a supported profile.
type CanonicalForm struct {
	RequestVersion uint32
	BindingVersion uint32
	Bytes          []byte
	Binding        Binding
}

func ValidateCommand(command Command) error {
	if command.Schema != CanonicalRequestVersion {
		return problem("request_version")
	}
	if command.Operation != AuthorizationAction {
		return problem("request_operation")
	}
	s, e, g := command.Scope, command.Execution, command.RequiredGuard
	if !identifiers(s.Realm, s.Host, s.Resource, s.Backend, e.RequestID, e.ExecutionKey, e.OperationID, e.AttemptID, e.StepID,
		command.RequestingPrincipal, command.AuthorizationIssuer, command.CandidateRevision, command.CandidateContentPin,
		command.Expected.Revision, g.Policy, g.Trust, g.Membership, g.OwnershipEpoch, g.FenceEpoch, g.Generation, g.BootID, g.ProcessIncarnation,
		command.Timing.TimeBase, command.OriginalBudget) {
		return problem("request_identity")
	}
	if command.Expected.StateVersion == math.MaxUint64 || command.Timing.DeadlineTick == 0 {
		return problem("request_number")
	}
	return nil
}

// CanonicalBytes is the CLOSED v1 encoder, not an assertion that any normal
// json.Marshal result is canonical. Field names/order and decimal representation
// below are fixed to C10 6f71ee3a0906665f0a639abf69c27749ba211dc7. Validation makes
// every string ASCII with no character needing JSON escaping. Thus unescaped
// values below are safe and exactly match the legacy Go1.26.5 marshal bytes.
// There is no whitespace, BOM, trailing newline, normalization or float coercion.
func CanonicalBytes(c Command) ([]byte, error) {
	if err := ValidateCommand(c); err != nil {
		return nil, err
	}
	b := make([]byte, 0, 1024)
	b = append(b, `{"Schema":1,"Operation":`...)
	b = canonicalString(b, c.Operation)
	b = append(b, `,"Scope":{"Realm":`...)
	b = canonicalString(b, c.Scope.Realm)
	b = append(b, `,"Host":`...)
	b = canonicalString(b, c.Scope.Host)
	b = append(b, `,"Resource":`...)
	b = canonicalString(b, c.Scope.Resource)
	b = append(b, `,"Backend":`...)
	b = canonicalString(b, c.Scope.Backend)
	b = append(b, `},"Execution":{"RequestID":`...)
	b = canonicalString(b, c.Execution.RequestID)
	b = append(b, `,"ExecutionKey":`...)
	b = canonicalString(b, c.Execution.ExecutionKey)
	b = append(b, `,"OperationID":`...)
	b = canonicalString(b, c.Execution.OperationID)
	b = append(b, `,"AttemptID":`...)
	b = canonicalString(b, c.Execution.AttemptID)
	b = append(b, `,"StepID":`...)
	b = canonicalString(b, c.Execution.StepID)
	b = append(b, `},"RequestingPrincipal":`...)
	b = canonicalString(b, c.RequestingPrincipal)
	b = append(b, `,"AuthorizationIssuer":`...)
	b = canonicalString(b, c.AuthorizationIssuer)
	b = append(b, `,"CandidateRevision":`...)
	b = canonicalString(b, c.CandidateRevision)
	b = append(b, `,"CandidateContentPin":`...)
	b = canonicalString(b, c.CandidateContentPin)
	b = append(b, `,"Expected":{"Revision":`...)
	b = canonicalString(b, c.Expected.Revision)
	b = append(b, `,"StateVersion":`...)
	b = strconv.AppendUint(b, c.Expected.StateVersion, 10)
	b = append(b, `},"RequiredGuard":{"Policy":`...)
	b = canonicalString(b, c.RequiredGuard.Policy)
	b = append(b, `,"Trust":`...)
	b = canonicalString(b, c.RequiredGuard.Trust)
	b = append(b, `,"Membership":`...)
	b = canonicalString(b, c.RequiredGuard.Membership)
	b = append(b, `,"OwnershipEpoch":`...)
	b = canonicalString(b, c.RequiredGuard.OwnershipEpoch)
	b = append(b, `,"FenceEpoch":`...)
	b = canonicalString(b, c.RequiredGuard.FenceEpoch)
	b = append(b, `,"Generation":`...)
	b = canonicalString(b, c.RequiredGuard.Generation)
	b = append(b, `,"BootID":`...)
	b = canonicalString(b, c.RequiredGuard.BootID)
	b = append(b, `,"ProcessIncarnation":`...)
	b = canonicalString(b, c.RequiredGuard.ProcessIncarnation)
	b = append(b, `},"Timing":{"TimeBase":`...)
	b = canonicalString(b, c.Timing.TimeBase)
	b = append(b, `,"DeadlineTick":`...)
	b = strconv.AppendUint(b, c.Timing.DeadlineTick, 10)
	b = append(b, `},"OriginalBudget":`...)
	b = canonicalString(b, c.OriginalBudget)
	b = append(b, '}')
	if len(b) > MaxRequestBytes {
		return nil, problem("request_bounds")
	}
	return b, nil
}
func canonicalString(b []byte, s string) []byte {
	b = append(b, '"')
	b = append(b, s...)
	return append(b, '"')
}

func ComputeBinding(command Command) (Binding, error) {
	b, err := CanonicalBytes(command)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func VerifyBinding(command Command, binding Binding) error {
	expected, err := ComputeBinding(command)
	if err != nil {
		return err
	}
	if len(binding) == 0 || len(binding) > MaxRequestBytes || !utf8.ValidString(binding) || binding != expected {
		return problem("request_binding")
	}
	return nil
}
func ValidateRequest(request Request) error { return VerifyBinding(request.Command, request.Binding) }

// ValidateRequestFor correlates the entire request to a trusted caller target.
// It checks equality, never authority, global uniqueness or resource ownership.
func ValidateRequestFor(request Request, scope Scope, execution Execution) error {
	if err := ValidateRequest(request); err != nil {
		return err
	}
	if request.Command.Scope != scope || request.Command.Execution != execution {
		return problem("request_correlation")
	}
	return nil
}
func Canonicalize(command Command) (CanonicalForm, error) {
	b, err := CanonicalBytes(command)
	if err != nil {
		return CanonicalForm{}, err
	}
	return CanonicalForm{RequestVersion: CanonicalRequestVersion, BindingVersion: CanonicalBindingVersion, Bytes: b, Binding: string(b)}, nil
}
func VerifyCanonicalForm(command Command, form CanonicalForm) error {
	if form.RequestVersion != CanonicalRequestVersion || form.BindingVersion != CanonicalBindingVersion {
		return problem("request_version")
	}
	b, err := CanonicalBytes(command)
	if err != nil {
		return err
	}
	if !bytes.Equal(b, form.Bytes) || form.Binding != string(b) {
		return problem("request_binding")
	}
	return nil
}

func EncodeRequest(request Request) ([]byte, error) {
	if err := ValidateRequest(request); err != nil {
		return nil, err
	}
	command, err := CanonicalBytes(request.Command)
	if err != nil {
		return nil, err
	}
	b := append([]byte(`{"Command":`), command...)
	b = append(b, `,"Binding":`...)
	// A valid binding is the ASCII JSON preimage above. AppendQuote has the same
	// quote/backslash escapes as legacy json.Marshal for that restricted alphabet.
	b = strconv.AppendQuote(b, request.Binding)
	b = append(b, '}')
	if len(b) > MaxRequestBytes {
		return nil, problem("request_bounds")
	}
	return b, nil
}

// Private fixed wire shapes override Scope/Execution's Evidence tags without
// duplicating public semantic types. The strict token walker rejects decoded-key
// duplicates (including escaped aliases), case aliases, null, omitted fields,
// unknown fields, fractional/exponent/signed integers and excessive depth.
type requestScopeWire struct {
	Realm    string `json:"Realm"`
	Host     string `json:"Host"`
	Resource string `json:"Resource"`
	Backend  string `json:"Backend"`
}
type requestExecutionWire struct {
	RequestID    string `json:"RequestID"`
	ExecutionKey string `json:"ExecutionKey"`
	OperationID  string `json:"OperationID"`
	AttemptID    string `json:"AttemptID"`
	StepID       string `json:"StepID"`
}
type commandWire struct {
	Schema              uint32               `json:"Schema"`
	Operation           string               `json:"Operation"`
	Scope               requestScopeWire     `json:"Scope"`
	Execution           requestExecutionWire `json:"Execution"`
	RequestingPrincipal string               `json:"RequestingPrincipal"`
	AuthorizationIssuer string               `json:"AuthorizationIssuer"`
	CandidateRevision   string               `json:"CandidateRevision"`
	CandidateContentPin string               `json:"CandidateContentPin"`
	Expected            ExpectedReference    `json:"Expected"`
	RequiredGuard       RequiredGuard        `json:"RequiredGuard"`
	Timing              CommandTiming        `json:"Timing"`
	OriginalBudget      string               `json:"OriginalBudget"`
}
type requestWire struct {
	Command commandWire `json:"Command"`
	Binding string      `json:"Binding"`
}

func commandFromWire(w commandWire) Command {
	return Command{Schema: w.Schema, Operation: w.Operation, Scope: Scope{Realm: w.Scope.Realm, Host: w.Scope.Host, Resource: w.Scope.Resource, Backend: w.Scope.Backend},
		Execution:           Execution{RequestID: w.Execution.RequestID, ExecutionKey: w.Execution.ExecutionKey, OperationID: w.Execution.OperationID, AttemptID: w.Execution.AttemptID, StepID: w.Execution.StepID},
		RequestingPrincipal: w.RequestingPrincipal, AuthorizationIssuer: w.AuthorizationIssuer, CandidateRevision: w.CandidateRevision, CandidateContentPin: w.CandidateContentPin,
		Expected: w.Expected, RequiredGuard: w.RequiredGuard, Timing: w.Timing, OriginalBudget: w.OriginalBudget}
}
func decodeRequestShape(data []byte, target any) error {
	if len(data) == 0 || len(data) > MaxRequestBytes || !utf8.Valid(data) {
		return problem("request_encoding")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := checkValue(decoder, reflect.TypeOf(target).Elem(), 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return problem("request_trailing")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return problem("request_encoding")
	}
	return nil
}
func DecodeRequest(data []byte) (Request, error) {
	var wire requestWire
	if err := decodeRequestShape(data, &wire); err != nil {
		return Request{}, err
	}
	r := Request{Command: commandFromWire(wire.Command), Binding: wire.Binding}
	if err := ValidateRequest(r); err != nil {
		return Request{}, err
	}
	return r, nil
}

// Marshal/Unmarshal keep accidental generic JSON use from silently selecting the
// Evidence snake_case representation or bypassing strict request validation.
// Use CanonicalBytes/EncodeRequest explicitly at contract boundaries.
func (c Command) MarshalJSON() ([]byte, error) { return CanonicalBytes(c) }
func (c *Command) UnmarshalJSON(data []byte) error {
	if c == nil {
		return problem("request_encoding")
	}
	var wire commandWire
	if err := decodeRequestShape(data, &wire); err != nil {
		return err
	}
	decoded := commandFromWire(wire)
	if err := ValidateCommand(decoded); err != nil {
		return err
	}
	*c = decoded
	return nil
}
func (r Request) MarshalJSON() ([]byte, error) { return EncodeRequest(r) }
func (r *Request) UnmarshalJSON(data []byte) error {
	if r == nil {
		return problem("request_encoding")
	}
	decoded, err := DecodeRequest(data)
	if err != nil {
		return err
	}
	*r = decoded
	return nil
}
