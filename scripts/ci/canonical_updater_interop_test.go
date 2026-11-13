//go:build canonical_request_interop

// Copied only into the pinned disposable Updater checkout. No Updater product
// source, branch or PR is modified. All target data and authority are synthetic.
package registry

import (
 "bytes"
 "encoding/json"
 "math"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "testing"

 evidence "github.com/Kome-Lab/Autostream-Contracts/v3evidence"
)

type canonicalC10Fixture struct {
 Request json.RawMessage `json:"request"`
 CanonicalBytes string `json:"canonical_bytes"`
 Scope evidence.Scope `json:"scope"`
 Execution evidence.Execution `json:"execution"`
 Frame evidence.BackendReadFrame `json:"backend_read_frame"`
 Issuer string `json:"issuer"`
 Audience string `json:"audience"`
}

func canonicalInteropDirectory(t *testing.T) string {
 t.Helper();d:=os.Getenv("CANONICAL_INTEROP_DIR");if d==""||!filepath.IsAbs(d){t.Fatal("explicit absolute synthetic interop directory required")};return d
}

func canonicalCompareC10(t *testing.T,legacy Request) evidence.Request {
 t.Helper()
 if err:=ValidateRequest(legacy);err!=nil{t.Fatal("legacy validation",err)}
 wire,err:=json.Marshal(legacy);if err!=nil{t.Fatal(err)}
 shared,err:=evidence.DecodeRequest(wire);if err!=nil{t.Fatal("shared rejected actual C10 request",err)}
 previous,err:=CanonicalBinding(legacy.Command);if err!=nil{t.Fatal(err)}
 canonical,err:=evidence.CanonicalBytes(shared.Command);if err!=nil||!bytes.Equal(canonical,[]byte(previous)){t.Fatal("canonical Command bytes differ",err)}
 binding,err:=evidence.ComputeBinding(shared.Command);if err!=nil||binding!=string(previous)||binding!=shared.Binding{t.Fatal("binding differs",err)}
 if err:=evidence.VerifyBinding(shared.Command,string(previous));err!=nil{t.Fatal(err)}
 encoded,err:=evidence.EncodeRequest(shared);if err!=nil{t.Fatal(err)}
 if !bytes.Equal(encoded,wire){t.Fatal("canonical Request wire differs from legacy C10 field order or representation")}
 decoded,err:=DecodeRequest(encoded);if err!=nil||!reflect.DeepEqual(decoded,legacy){t.Fatal("round trip changed C10 semantic fields",err)}
 if err:=evidence.ValidateRequest(shared);err!=nil{t.Fatal(err)}
 return shared
}

func TestCanonicalContractsC10ExactCompatibility(t *testing.T) {
 if evidence.CanonicalRequestVersion!=1||evidence.CanonicalBindingVersion!=1||evidence.ProtocolVersion!=3{t.Fatal("independent versions changed")}
 canonicalCompareC10(t,validationValidRequest(t))
 command:=validationValidCommand();command.Expected.StateVersion=math.MaxUint64-1;command.Timing.DeadlineTick=math.MaxUint64
 binding,err:=CanonicalBinding(command);if err!=nil{t.Fatal(err)};canonicalCompareC10(t,Request{Command:command,Binding:binding})
 t.Log("legacy C10 bytes and every shared encoding/binding round trip agree; no new binding format")
}

func TestCanonicalContractsC10EveryFieldDifferential(t *testing.T) {
 baseline:=validationValidCommand();var paths [][]int;var names []string
 var visit func(reflect.Type,[]int,string)
 visit=func(typ reflect.Type,path []int,prefix string){for n:=0;n<typ.NumField();n++{f:=typ.Field(n);p:=append(append([]int(nil),path...),n);name:=prefix+"."+f.Name;if f.Type.Kind()==reflect.Struct{visit(f.Type,p,name)}else{paths=append(paths,p);names=append(names,name)}}}
 visit(reflect.TypeOf(baseline),nil,"Command")
 original,_:=CanonicalBinding(baseline)
 for n,path:=range paths{t.Run(names[n],func(t *testing.T){command:=baseline;v:=reflect.ValueOf(&command).Elem().FieldByIndex(path);switch v.Kind(){case reflect.String:if names[n]=="Command.Operation"{v.SetString("not.allowed")}else{v.SetString(v.String()+"-changed")};case reflect.Uint32:v.SetUint(2);case reflect.Uint64:v.SetUint(v.Uint()+1);default:t.Fatal("unexamined semantic kind")}
  binding,legacyErr:=CanonicalBinding(command)
  candidate:=Request{Command:command,Binding:original};raw,_:=json.Marshal(candidate)
  _,sharedStaleErr:=evidence.DecodeRequest(raw);legacyStaleErr:=ValidateRequest(candidate)
  if (sharedStaleErr==nil)!=(legacyStaleErr==nil)||sharedStaleErr==nil{t.Fatal("stale binding was not rejected consistently")}
  if legacyErr!=nil{raw,_:=json.Marshal(Request{Command:command,Binding:original});if _,err:=evidence.DecodeRequest(raw);err==nil{t.Fatal("shared accepted legacy-invalid semantics")};return}
  if binding==original{t.Fatal("legacy field absent from binding")};canonicalCompareC10(t,Request{Command:command,Binding:binding})
 })}
}

func TestCanonicalContractsC10StrictDecodeDifferential(t *testing.T) {
 raw:=validationJSON(t)
 for name,wire:=range map[string]string{
  "unknown-version":strings.Replace(raw,`"Schema":1`,`"Schema":2`,1),
  "extra":strings.Replace(raw,`"Command":{`,`"Command":{"Extra":"not-allowed",`,1),
  "duplicate":strings.Replace(raw,`"Schema":1`,`"Schema":1,"Schema":1`,1),
  "duplicate-escape":strings.Replace(raw,`"Schema":1`,`"Schema":1,"\u0053chema":1`,1),
  "wrong-case":strings.Replace(raw,`"Schema":1`,`"schema":1`,1),
  "null":strings.Replace(raw,`"Schema":1`,`"Schema":null`,1),
  "missing":strings.Replace(raw,`"Schema":1,`,``,1),
  "exponent":strings.Replace(raw,`"Schema":1`,`"Schema":1e0`,1),
  "fraction":strings.Replace(raw,`"StateVersion":0`,`"StateVersion":0.0`,1),
  "overflow":strings.Replace(raw,`"StateVersion":0`,`"StateVersion":18446744073709551616`,1),
  "unicode-id":strings.Replace(raw,`"Realm":"realm"`,`"Realm":"réalm"`,1),
  "binding-tamper":strings.Replace(raw,`"Binding":"`,`"Binding":"tampered`,1),
  "trailing":raw+` {}`,
  "invalid-utf8":raw[:len(raw)-1]+string([]byte{0xff})+`}`,
 }{t.Run(name,func(t *testing.T){_,a:=DecodeRequest([]byte(wire));_,b:=evidence.DecodeRequest([]byte(wire));if a==nil||b==nil{t.Fatalf("invalid proposal accepted: C10=%v shared=%v",a,b)}})}
 escaped:=strings.Replace(raw,`"Realm":"realm"`,`"Realm":"\u0072ealm"`,1)
 a,e1:=DecodeRequest([]byte(escaped));b,e2:=evidence.DecodeRequest([]byte(escaped));if e1!=nil||e2!=nil{t.Fatal("valid escaped ASCII differed",e1,e2)};if string(a.Binding)!=b.Binding{t.Fatal("escaped representation changed binding")}
}

func TestCanonicalContractsExportC10Fixture(t *testing.T) {
 f:=contractsTestNew(t);request:=canonicalCompareC10(t,f.request);var cp evidence.Envelope
 for _,raw:=range f.delivery.encoded{e,err:=evidence.Decode(raw);if err!=nil{t.Fatal(err)};if e.Kind==evidence.CPAuthorizationKind{cp=e;break}}
 raw,err:=json.Marshal(f.request);if err!=nil{t.Fatal(err)}
 fixture:=canonicalC10Fixture{Request:raw,CanonicalBytes:request.Binding,Scope:cp.Scope,Execution:cp.Execution,Frame:cp.Frame,Issuer:cp.Issuer,Audience:cp.Audience}
 if cp.Kind!=evidence.CPAuthorizationKind||fixture.Scope!=request.Command.Scope||fixture.Execution!=request.Command.Execution{t.Fatal("export lost scope/execution")}
 out,err:=json.Marshal(fixture);if err!=nil{t.Fatal(err)};if err=os.WriteFile(filepath.Join(canonicalInteropDirectory(t),"c10-canonical-fixture.json"),out,0600);err!=nil{t.Fatal(err)}
}

func TestCanonicalContractsImportCPEnvelope(t *testing.T) {
 raw,err:=os.ReadFile(filepath.Join(canonicalInteropDirectory(t),"cp-canonical-envelope.json"));if err!=nil{t.Fatal(err)}
 cp,err:=evidence.Decode(raw);if err!=nil||cp.Kind!=evidence.CPAuthorizationKind{t.Fatal("CP output malformed",err)}
 f:=contractsTestNew(t);shared:=canonicalCompareC10(t,f.request)
 if cp.Binding!=shared.Binding||cp.Scope!=shared.Command.Scope||cp.Execution!=shared.Command.Execution{t.Fatal("CP and C10 binding/scope/execution differ")}
 replaced:=false;for n,old:=range f.delivery.encoded{e,eerr:=evidence.Decode(old);if eerr!=nil{t.Fatal(eerr)};if e.Kind==evidence.CPAuthorizationKind{if e.Issuer!=cp.Issuer||e.Audience!=cp.Audience||e.Frame!=cp.Frame{t.Fatal("CP changed issuer/audience/frame")};f.delivery.encoded[n]=append([]byte(nil),raw...);replaced=true}}
 if !replaced{t.Fatal("missing CP envelope")}
 f.change(t,evidence.PolicyKind,func(e *evidence.Envelope){e.Policy.SubjectEvidenceID=cp.EvidenceID})
 f.change(t,evidence.TrustKind,func(e *evidence.Envelope){e.Trust.SubjectEvidenceID=cp.EvidenceID})
 f.issue(t,true);f.open(t);if got:=f.check(t);got.Status!=PreflightEligible{t.Fatal("CP with shared resolver failed C10 boundary",got.Status,got.Reason)}
 f.change(t,evidence.CPAuthorizationKind,func(e *evidence.Envelope){e.Binding="tampered-binding"})
 if got:=f.check(t);got.Status!=PreflightUnknown&&got.Status!=PreflightConflict{t.Fatal("tampered binding was accepted",got.Status,got.Reason)}
 t.Log("CP actual Service with disposable shared Resolver connected to C10; remaining Producers/security/time/Continuity synthetic, no production connection")
}
