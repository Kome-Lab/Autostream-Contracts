package evidence

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// This literal is an independent byte oracle for the existing C10 v1 schema.
// Do not derive it by marshaling the implementation's Command or private wire.
const requestTestGoldenCommand = `{"Schema":1,"Operation":"registry.reference_only","Scope":{"Realm":"realm-1","Host":"host-1","Resource":"resource-1","Backend":"backend-1"},"Execution":{"RequestID":"request-1","ExecutionKey":"execution-1","OperationID":"operation-1","AttemptID":"attempt-1","StepID":"step-1"},"RequestingPrincipal":"operator-1","AuthorizationIssuer":"cp-1","CandidateRevision":"candidate-1","CandidateContentPin":"content-1","Expected":{"Revision":"previous-1","StateVersion":7},"RequiredGuard":{"Policy":"policy-1","Trust":"trust-1","Membership":"membership-1","OwnershipEpoch":"owner-1","FenceEpoch":"fence-1","Generation":"generation-1","BootID":"boot-1","ProcessIncarnation":"process-1"},"Timing":{"TimeBase":"clock-1","DeadlineTick":100},"OriginalBudget":"budget-1"}`

func requestTestCommand() Command {
	return Command{
		Schema: CanonicalRequestVersion, Operation: "registry.reference_only",
		Scope: Scope{Realm: "realm-1", Host: "host-1", Resource: "resource-1", Backend: "backend-1"},
		Execution: Execution{RequestID: "request-1", ExecutionKey: "execution-1", OperationID: "operation-1", AttemptID: "attempt-1", StepID: "step-1"},
		RequestingPrincipal: "operator-1", AuthorizationIssuer: "cp-1",
		CandidateRevision: "candidate-1", CandidateContentPin: "content-1",
		Expected: ExpectedReference{Revision: "previous-1", StateVersion: 7},
		RequiredGuard: RequiredGuard{Policy: "policy-1", Trust: "trust-1", Membership: "membership-1", OwnershipEpoch: "owner-1", FenceEpoch: "fence-1", Generation: "generation-1", BootID: "boot-1", ProcessIncarnation: "process-1"},
		Timing: CommandTiming{TimeBase: "clock-1", DeadlineTick: 100}, OriginalBudget: "budget-1",
	}
}

func requestTestRequest(t *testing.T, command Command) Request {
	t.Helper()
	binding, err := ComputeBinding(command)
	if err != nil {
		t.Fatal(err)
	}
	return Request{Command: command, Binding: binding}
}

func requestTestWire(t *testing.T, request Request) []byte {
	t.Helper()
	wire, err := EncodeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestCanonicalRequestGoldenBytesAndBinding(t *testing.T) {
	command := requestTestCommand()
	canonical, err := CanonicalBytes(command)
	if err != nil || string(canonical) != requestTestGoldenCommand {
		t.Fatalf("canonical bytes differ from the frozen C10 v1 oracle: %v", err)
	}
	binding, err := ComputeBinding(command)
	if err != nil || binding != requestTestGoldenCommand || binding != string(canonical) {
		t.Fatal("binding is not the exact canonical command bytes", err)
	}
	if err := VerifyBinding(command, binding); err != nil {
		t.Fatal(err)
	}
	request := Request{Command: command, Binding: binding}
	wire := requestTestWire(t, request)
	want := `{"Command":` + requestTestGoldenCommand + `,"Binding":` + strconv.Quote(requestTestGoldenCommand) + `}`
	if string(wire) != want {
		t.Fatal("request encoding changed its legacy order, spelling or binding representation")
	}
	decoded, err := DecodeRequest(wire)
	if err != nil || !reflect.DeepEqual(decoded, request) {
		t.Fatal("canonical request round trip lost semantics", err)
	}
	if err := ValidateRequest(decoded); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 10; n++ {
		repeated, err := EncodeRequest(request)
		if err != nil || !bytes.Equal(repeated, wire) {
			t.Fatal("identical request produced different bytes", err)
		}
	}
}

func TestCanonicalRequestUsesSharedScopeExecutionWithoutChangingEnvelopeJSON(t *testing.T) {
	command := requestTestCommand()
	scope, err := json.Marshal(command.Scope)
	if err != nil || string(scope) != `{"realm":"realm-1","host":"host-1","resource":"resource-1","backend":"backend-1"}` {
		t.Fatal("existing evidence Scope representation changed", err)
	}
	execution, err := json.Marshal(command.Execution)
	if err != nil || !bytes.Contains(execution, []byte(`"request_id":"request-1"`)) {
		t.Fatal("existing evidence Execution representation changed", err)
	}
	canonical, err := CanonicalBytes(command)
	if err != nil || !bytes.Contains(canonical, []byte(`"Scope":{"Realm":`)) || !bytes.Contains(canonical, []byte(`"Execution":{"RequestID":`)) {
		t.Fatal("canonical v1 lost PascalCase despite reusing evidence types", err)
	}
	if ProtocolVersion != 3 || CanonicalRequestVersion != 1 || CanonicalBindingVersion != 1 {
		t.Fatal("independent protocol versions changed silently")
	}
}

func TestCanonicalRequestJSONMethodsUseQualifiedCodec(t *testing.T) {
	request := requestTestRequest(t, requestTestCommand())
	commandWire, err := json.Marshal(request.Command)
	if err != nil || string(commandWire) != requestTestGoldenCommand {
		t.Fatal("Command.MarshalJSON bypassed the canonical codec", err)
	}
	requestWire, err := json.Marshal(request)
	if err != nil || !bytes.Equal(requestWire, requestTestWire(t, request)) {
		t.Fatal("Request.MarshalJSON bypassed canonical encoding", err)
	}
	var command Command
	if err := json.Unmarshal(commandWire, &command); err != nil || !reflect.DeepEqual(command, request.Command) {
		t.Fatal("Command.UnmarshalJSON lost exact typed fields", err)
	}
	var decoded Request
	if err := json.Unmarshal(requestWire, &decoded); err != nil || !reflect.DeepEqual(decoded, request) {
		t.Fatal("Request.UnmarshalJSON lost exact typed fields", err)
	}
	bad := bytes.Replace(requestWire, []byte(`"Schema":1`), []byte(`"Schema":1,"Schema":1`), 1)
	if json.Unmarshal(bad, &decoded) == nil {
		t.Fatal("ordinary Unmarshal bypassed duplicate rejection")
	}
	command.Schema = 2
	if _, err := json.Marshal(command); err == nil {
		t.Fatal("ordinary Marshal accepted unknown command schema")
	}
}

func TestCanonicalBindingEveryMeaningFieldIsBound(t *testing.T) {
	mutations := map[string]func(*Command){
		"scope.realm": func(c *Command) { c.Scope.Realm = "realm-2" },
		"scope.host": func(c *Command) { c.Scope.Host = "host-2" },
		"scope.resource": func(c *Command) { c.Scope.Resource = "resource-2" },
		"scope.backend": func(c *Command) { c.Scope.Backend = "backend-2" },
		"execution.request": func(c *Command) { c.Execution.RequestID = "request-2" },
		"execution.key": func(c *Command) { c.Execution.ExecutionKey = "execution-2" },
		"execution.operation": func(c *Command) { c.Execution.OperationID = "operation-2" },
		"execution.attempt": func(c *Command) { c.Execution.AttemptID = "attempt-2" },
		"execution.step": func(c *Command) { c.Execution.StepID = "step-2" },
		"requesting-principal": func(c *Command) { c.RequestingPrincipal = "operator-2" },
		"authorization-issuer": func(c *Command) { c.AuthorizationIssuer = "cp-2" },
		"candidate-revision": func(c *Command) { c.CandidateRevision = "candidate-2" },
		"candidate-content-pin": func(c *Command) { c.CandidateContentPin = "content-2" },
		"expected-revision": func(c *Command) { c.Expected.Revision = "previous-2" },
		"expected-state-version": func(c *Command) { c.Expected.StateVersion++ },
		"guard-policy": func(c *Command) { c.RequiredGuard.Policy = "policy-2" },
		"guard-trust": func(c *Command) { c.RequiredGuard.Trust = "trust-2" },
		"guard-membership": func(c *Command) { c.RequiredGuard.Membership = "membership-2" },
		"guard-owner": func(c *Command) { c.RequiredGuard.OwnershipEpoch = "owner-2" },
		"guard-fence": func(c *Command) { c.RequiredGuard.FenceEpoch = "fence-2" },
		"guard-generation": func(c *Command) { c.RequiredGuard.Generation = "generation-2" },
		"guard-boot": func(c *Command) { c.RequiredGuard.BootID = "boot-2" },
		"guard-process": func(c *Command) { c.RequiredGuard.ProcessIncarnation = "process-2" },
		"time-base": func(c *Command) { c.Timing.TimeBase = "clock-2" },
		"deadline": func(c *Command) { c.Timing.DeadlineTick++ },
		"original-budget": func(c *Command) { c.OriginalBudget = "budget-2" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			command := requestTestCommand()
			mutate(&command)
			binding, err := ComputeBinding(command)
			if err != nil || binding == requestTestGoldenCommand {
				t.Fatal("changed meaning was omitted from binding", err)
			}
			if VerifyBinding(command, requestTestGoldenCommand) == nil || ValidateRequest(Request{Command: command, Binding: requestTestGoldenCommand}) == nil {
				t.Fatal("original binding authorized changed meaning")
			}
		})
	}
}

func TestCanonicalBindingExcludesItselfAndLaterEvidence(t *testing.T) {
	command := requestTestCommand()
	request := requestTestRequest(t, command)
	before, err := CanonicalBytes(request.Command)
	if err != nil {
		t.Fatal(err)
	}
	request.Binding = "tampered-binding"
	after, err := CanonicalBytes(request.Command)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("Binding recursively participates in its own preimage", err)
	}
	if _, err := EncodeRequest(request); err == nil {
		t.Fatal("changed Binding was silently recomputed")
	}
	for _, forbidden := range []string{"Binding", "Delivery", "Evidence", "Receipt", "BackendReadFrame", "Consume", "FenceProof"} {
		if bytes.Contains(before, []byte(`"`+forbidden+`":`)) {
			t.Fatalf("later metadata %s entered immutable command", forbidden)
		}
	}
}

func TestCanonicalBindingVersionAndOriginalBytes(t *testing.T) {
	command := requestTestCommand()
	form, err := Canonicalize(command)
	if err != nil || form.RequestVersion != CanonicalRequestVersion || form.BindingVersion != CanonicalBindingVersion || string(form.Bytes) != requestTestGoldenCommand || form.Binding != requestTestGoldenCommand {
		t.Fatal("canonical form does not retain explicit versions and original bytes", err)
	}
	if err := VerifyCanonicalForm(command, form); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*CanonicalForm){
		"request-version": func(f *CanonicalForm) { f.RequestVersion++ },
		"binding-version": func(f *CanonicalForm) { f.BindingVersion++ },
		"zero-request-version": func(f *CanonicalForm) { f.RequestVersion = 0 },
		"zero-binding-version": func(f *CanonicalForm) { f.BindingVersion = 0 },
		"canonical-bytes": func(f *CanonicalForm) { f.Bytes = []byte(`{"Schema":1}`) },
		"binding": func(f *CanonicalForm) { f.Binding = "different" },
		"absent-canonical": func(f *CanonicalForm) { f.Bytes = nil },
	} {
		t.Run(name, func(t *testing.T) {
			changed := form
			changed.Bytes = bytes.Clone(form.Bytes)
			mutate(&changed)
			if VerifyCanonicalForm(command, changed) == nil {
				t.Fatal("incompatible or mismatched canonical form accepted")
			}
		})
	}
	form.Bytes[0] = 'x'
	if form.Binding != requestTestGoldenCommand {
		t.Fatal("caller-owned bytes changed the retained binding string")
	}
	fresh, err := CanonicalBytes(command)
	if err != nil || string(fresh) != requestTestGoldenCommand {
		t.Fatal("caller mutation corrupted future encoding", err)
	}
}

func TestCanonicalRequestVersionsAndIdentifierValidation(t *testing.T) {
	for _, schema := range []uint32{0, 2, math.MaxUint32} {
		command := requestTestCommand()
		command.Schema = schema
		if _, err := CanonicalBytes(command); err == nil {
			t.Fatal("unknown Schema accepted")
		}
	}
	for _, invalid := range []string{"", "Operator", " operator", "operator ", "/path", ".first", "_first", ":first", "-first", "has/slash", "has\\slash", "has\x00nul", "é", "e\u0301", "元音声", "\ufffd", strings.Repeat("a", 129)} {
		t.Run(strconv.Quote(invalid), func(t *testing.T) {
			command := requestTestCommand()
			command.RequestingPrincipal = invalid
			if _, err := ComputeBinding(command); err == nil {
				t.Fatal("identifier was normalized or accepted instead of rejected")
			}
		})
	}
	for _, valid := range []string{"a", "0", "a:._-", strings.Repeat("a", 128)} {
		command := requestTestCommand()
		command.RequestingPrincipal = valid
		request := requestTestRequest(t, command)
		if _, err := DecodeRequest(requestTestWire(t, request)); err != nil {
			t.Fatal("canonical identifier boundary rejected", err)
		}
	}
	command := requestTestCommand()
	command.Operation = "other.operation"
	if _, err := CanonicalBytes(command); err == nil {
		t.Fatal("other action accepted")
	}
}

func TestCanonicalRequestNumericBoundaries(t *testing.T) {
	for _, state := range []uint64{0, 1, uint64(math.MaxInt64), uint64(math.MaxInt64) + 1, math.MaxUint64 - 1} {
		command := requestTestCommand()
		command.Expected.StateVersion = state
		command.Timing.DeadlineTick = math.MaxUint64
		request := requestTestRequest(t, command)
		wire := requestTestWire(t, request)
		decoded, err := DecodeRequest(wire)
		if err != nil || decoded.Command.Expected.StateVersion != state || decoded.Command.Timing.DeadlineTick != math.MaxUint64 {
			t.Fatal("uint64 value was rounded, truncated or rejected", err)
		}
	}
	command := requestTestCommand()
	command.Expected.StateVersion = math.MaxUint64
	if _, err := ComputeBinding(command); err == nil {
		t.Fatal("exhausted reference state version accepted")
	}
	command = requestTestCommand()
	command.Timing.DeadlineTick = 0
	if _, err := ComputeBinding(command); err == nil {
		t.Fatal("zero deadline accepted")
	}
	wire := requestTestWire(t, requestTestRequest(t, requestTestCommand()))
	for _, number := range []string{"7.0", "7e0", "7E+0", "-7", "+7", "007", "18446744073709551616", "null", `"7"`, "true"} {
		bad := bytes.Replace(wire, []byte(`"StateVersion":7`), []byte(`"StateVersion":`+number), 1)
		if _, err := DecodeRequest(bad); err == nil {
			t.Fatalf("noncanonical or invalid number %s accepted", number)
		}
	}
	for _, number := range []string{"1.0", "1e0", "-1", "4294967296", "0", "2"} {
		bad := bytes.Replace(wire, []byte(`"Schema":1`), []byte(`"Schema":`+number), 1)
		if _, err := DecodeRequest(bad); err == nil {
			t.Fatal("invalid schema numeric representation accepted")
		}
	}
}

func TestCanonicalRequestStrictShape(t *testing.T) {
	wire := requestTestWire(t, requestTestRequest(t, requestTestCommand()))
	mutations := map[string][]byte{
		"empty": nil,
		"unknown-root": append(bytes.Clone(wire[:len(wire)-1]), []byte(`,"Delivery":{}}`)...),
		"unknown-command": bytes.Replace(wire, []byte(`"Schema":1`), []byte(`"Schema":1,"Evidence":{}`), 1),
		"unknown-scope": bytes.Replace(wire, []byte(`"Realm":"realm-1"`), []byte(`"Realm":"realm-1","Unknown":"x"`), 1),
		"wrong-case": bytes.Replace(wire, []byte(`"Command":`), []byte(`"command":`), 1),
		"snake-case-scope": bytes.Replace(wire, []byte(`"Realm":`), []byte(`"realm":`), 1),
		"duplicate-root": append(bytes.Clone(wire[:len(wire)-1]), []byte(`,"Binding":"other"}`)...),
		"duplicate-command": bytes.Replace(wire, []byte(`"Schema":1`), []byte(`"Schema":1,"Schema":1`), 1),
		"duplicate-escaped-key": bytes.Replace(wire, []byte(`"Realm":"realm-1"`), []byte(`"Realm":"realm-1","\u0052ealm":"realm-1"`), 1),
		"null-command": []byte(`{"Command":null,"Binding":"x"}`),
		"null-scope": bytes.Replace(wire, []byte(`"Scope":{"Realm":"realm-1","Host":"host-1","Resource":"resource-1","Backend":"backend-1"}`), []byte(`"Scope":null`), 1),
		"null-binding": []byte(`{"Command":` + requestTestGoldenCommand + `,"Binding":null}`),
		"missing-command": []byte(`{"Binding":"x"}`),
		"missing-binding": []byte(`{"Command":` + requestTestGoldenCommand + `}`),
		"missing-scalar": bytes.Replace(wire, []byte(`"OriginalBudget":"budget-1"`), nil, 1),
		"trailing": append(bytes.Clone(wire), []byte(` {}`)...),
		"binding-empty": []byte(`{"Command":` + requestTestGoldenCommand + `,"Binding":""}`),
		"binding-reordered": []byte(`{"Command":` + requestTestGoldenCommand + `,"Binding":"different-order-is-not-equality"}`),
		"invalid-utf8": bytes.Replace(wire, []byte(`"cp-1"`), []byte{'"', 0xff, '"'}, 1),
		"unicode-id": bytes.Replace(wire, []byte(`"operator-1"`), []byte(`"日本語"`), 1),
		"replacement-rune": bytes.Replace(wire, []byte(`"operator-1"`), []byte(`"\ufffd"`), 1),
		"unpaired-high-surrogate": bytes.Replace(wire, []byte(`"operator-1"`), []byte(`"\ud800"`), 1),
		"unpaired-low-surrogate": bytes.Replace(wire, []byte(`"operator-1"`), []byte(`"\udc00"`), 1),
	}
	for name, raw := range mutations {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeRequest(raw); err == nil {
				t.Fatal("invalid or ambiguous request was accepted")
			}
		})
	}
	// Zero is valid but remains REQUIRED. A decoder which only validates the
	// default Go value would incorrectly accept a missing StateVersion here.
	zero := requestTestCommand()
	zero.Expected.StateVersion = 0
	zeroWire := requestTestWire(t, requestTestRequest(t, zero))
	missingZero := bytes.Replace(zeroWire, []byte(`,"StateVersion":0`), nil, 1)
	if _, err := DecodeRequest(missingZero); err == nil {
		t.Fatal("missing required zero-valued scalar accepted")
	}
}

func TestCanonicalRequestInputOrderingAndASCIIEscapes(t *testing.T) {
	request := requestTestRequest(t, requestTestCommand())
	wire := requestTestWire(t, request)
	for name, changed := range map[string][]byte{
		"command-order": bytes.Replace(wire, []byte(`"Schema":1,"Operation":"registry.reference_only"`), []byte(`"Operation":"registry.reference_only","Schema":1`), 1),
		"escaped-ascii-value": bytes.Replace(wire, []byte(`"Realm":"realm-1"`), []byte(`"Realm":"realm-\u0031"`), 1),
		"escaped-ascii-key": bytes.Replace(wire, []byte(`"Scope":`), []byte(`"Scop\u0065":`), 1),
		"whitespace": append([]byte("\n \t"), wire...),
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := DecodeRequest(changed)
			if err != nil || !reflect.DeepEqual(decoded, request) {
				t.Fatal("legacy-compatible input representation rejected or changed meaning", err)
			}
			canonical, err := CanonicalBytes(decoded.Command)
			if err != nil || string(canonical) != requestTestGoldenCommand {
				t.Fatal("input representation changed canonical bytes", err)
			}
		})
	}
}

func TestCanonicalRequestBoundsAndDepth(t *testing.T) {
	wire := requestTestWire(t, requestTestRequest(t, requestTestCommand()))
	atLimit := append([]byte(strings.Repeat(" ", MaxRequestBytes-len(wire))), wire...)
	if len(atLimit) != MaxRequestBytes {
		t.Fatal("size fixture does not reach the exact bound")
	}
	if _, err := DecodeRequest(atLimit); err != nil {
		t.Fatal("existing C10 exact size boundary was narrowed", err)
	}
	if _, err := DecodeRequest(append(atLimit, ' ')); err == nil {
		t.Fatal("over-limit request accepted")
	}
	depth := strings.Repeat(`{"Scope":`, MaxRequestDepth+1) + `{}` + strings.Repeat(`}`, MaxRequestDepth+1)
	if _, err := DecodeRequest([]byte(`{"Command":` + depth + `,"Binding":"x"}`)); err == nil {
		t.Fatal("deeply nested input accepted")
	}
	request := requestTestRequest(t, requestTestCommand())
	request.Binding = strings.Repeat("a", MaxRequestBytes+1)
	if ValidateRequest(request) == nil || VerifyBinding(request.Command, request.Binding) == nil {
		t.Fatal("oversized opaque Binding accepted")
	}
}

func TestCanonicalRequestParallelDeterminism(t *testing.T) {
	command := requestTestCommand()
	const workers = 16
	errors := make(chan string, workers)
	var wg sync.WaitGroup
	for n := 0; n < workers; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repeat := 0; repeat < 20; repeat++ {
				canonical, err := CanonicalBytes(command)
				if err != nil || string(canonical) != requestTestGoldenCommand {
					errors <- "parallel canonicalization changed bytes"
					return
				}
				request := Request{Command: command, Binding: string(canonical)}
				wire, err := EncodeRequest(request)
				if err != nil {
					errors <- "parallel encoding failed"
					return
				}
				if decoded, err := DecodeRequest(wire); err != nil || !reflect.DeepEqual(decoded, request) {
					errors <- "parallel decode changed meaning"
					return
				}
				canonical[0] = 'x'
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestCanonicalRequestTrustedTargetCorrelation(t *testing.T) {
 request:=requestTestRequest(t,requestTestCommand())
 if err:=ValidateRequestFor(request,request.Command.Scope,request.Command.Execution);err!=nil{t.Fatal(err)}
 scope:=request.Command.Scope;scope.Host="another-host"
 if ValidateRequestFor(request,scope,request.Command.Execution)==nil{t.Fatal("scope mismatch accepted")}
 execution:=request.Command.Execution;execution.AttemptID="another-attempt"
 if ValidateRequestFor(request,request.Command.Scope,execution)==nil{t.Fatal("execution mismatch accepted")}
}
