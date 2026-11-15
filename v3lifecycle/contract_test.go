package lifecycle

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func fixtureReadJSON(t *testing.T, name string, out any) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("testdata", "conformance", name))
	if e != nil {
		t.Fatal(e)
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		t.Fatal(e)
	}
}

func TestContractRegisterClosed(t *testing.T) {
	var oracle struct {
		SpecRef                       string
		TestOnly, ProductionQualified bool
		Records                       []struct {
			Contract            ContractID
			Input               string
			Expected            ResultKind
			ProductionQualified bool
		}
		ForbiddenFields []string
	}
	fixtureReadJSON(t, "record-expectations.json", &oracle)
	if oracle.SpecRef != SpecRef || !oracle.TestOnly || oracle.ProductionQualified || len(oracle.Records) != 9 {
		t.Fatal("independent nine-contract test oracle changed")
	}
	got := ContractRegister()
	if len(got) != 9 {
		t.Fatalf("contract register length=%d", len(got))
	}
	seen := map[ContractID]bool{}
	for i, want := range []ContractID{CC01, CC02, CC03, CC04, CC05, CC06, CC07, CC08, CC09} {
		if got[i].ID != want || got[i].OwnerRepository != "Kome-Lab/Autostream-Contracts" || got[i].Name == "" || seen[got[i].ID] {
			t.Fatalf("contract %d: %+v", i, got[i])
		}
		seen[got[i].ID] = true
		if oracle.Records[i].Contract != want || oracle.Records[i].Expected != StructurallyCoherent || oracle.Records[i].ProductionQualified {
			t.Fatal("oracle is not fixed structural-only input")
		}
		t.Run(string(want)+"/positive", func(t *testing.T) {
			r, p := fixtureRecord(want)
			fixtureWant(t, ValidateRecord(r, p), oracle.Records[i].Expected)
		})
	}
	got[0].Name = "mutated caller copy"
	if ContractRegister()[0].Name == got[0].Name {
		t.Fatal("caller can mutate closed registry")
	}
	for _, tc := range []struct {
		name   string
		change func(*Record)
	}{
		{"unknown-contract", func(r *Record) {
			r.Header.ContractRef.ContractID = "CC10"
			r.Header.RecordKey.ContractRef = r.Header.ContractRef
		}},
		{"future-logical-revision", func(r *Record) {
			r.Header.ContractRef.LogicalRevision = 2
			r.Header.RecordKey.ContractRef = r.Header.ContractRef
		}},
		{"missing-owner", func(r *Record) { r.Header.RecordKey.OwnerID = "" }},
		{"unknown-component", func(r *Record) { r.Header.ProducerRef.ComponentID = "C15" }},
		{"missing-attempt", func(r *Record) { r.Header.Correlation.AttemptID = "" }},
		{"unknown-resource-kind", func(r *Record) { r.Header.DomainRef.Resources[0].Kind = "FILE_PATH" }},
		{"multiple-union-payloads", func(r *Record) { r.Seal = &SealRecord{} }},
		{"absent-union-payload", func(r *Record) { r.Vector = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, p := fixtureRecord(CC02)
			tc.change(&r)
			fixtureWant(t, ValidateRecord(r, p), RefusedUnchanged)
		})
	}
	r, p := fixtureRecord(CC02)
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	if _, got := DecodeRecord(b, p); got.Kind != StructurallyCoherent {
		t.Fatalf("well-formed internal record: %+v", got)
	}
	h, e := json.Marshal(r.Header)
	if e != nil {
		t.Fatal(e)
	}
	dup := append(append(append([]byte{}, b[:len(b)-1]...), []byte(",\"Header\":")...), h...)
	dup = append(dup, '}')
	_, aliasResult := DecodeRecord(bytes.Replace(dup, []byte(",\"Header\":"), []byte(",\"header\":"), 1), p)
	fixtureWant(t, aliasResult, RefusedUnchanged)
	_, dupResult := DecodeRecord(dup, p)
	fixtureWant(t, dupResult, RefusedUnchanged)
	for _, field := range oracle.ForbiddenFields {
		t.Run("forbidden-field/"+field, func(t *testing.T) {
			sentinel := "REDACTION_SENTINEL_TEST_ONLY"
			input := append(append([]byte{}, b[:len(b)-1]...), []byte(",\""+field+"\":\""+sentinel+"\"}")...)
			_, result := DecodeRecord(input, p)
			fixtureWant(t, result, RefusedUnchanged)
			diagnostics, _ := json.Marshal(result)
			if bytes.Contains(diagnostics, []byte(sentinel)) {
				t.Fatal("diagnostic echoed forbidden payload")
			}
		})
	}
	_, trailing := DecodeRecord(append(b, []byte("\n{}")...), p)
	fixtureWant(t, trailing, RefusedUnchanged)
}

func TestMembershipFreezeAndAliasCoverage(t *testing.T) {
	a, p := fixtureRecord(CC01)
	b := fixtureClone(t, a)
	for i, j := 0, len(b.Membership.Members)-1; i < j; i, j = i+1, j-1 {
		b.Membership.Members[i], b.Membership.Members[j] = b.Membership.Members[j], b.Membership.Members[i]
	}
	for i, j := 0, len(b.Header.DomainRef.Resources)-1; i < j; i, j = i+1, j-1 {
		b.Header.DomainRef.Resources[i], b.Header.DomainRef.Resources[j] = b.Header.DomainRef.Resources[j], b.Header.DomainRef.Resources[i]
	}
	fixtureWant(t, CompareImmutable(a, b, p), StructurallyCoherent)
	b = fixtureClone(t, a)
	b.Membership.Members[0].Producer.WriterID = "other-writer"
	fixtureWant(t, CompareImmutable(a, b, p), Conflict)
	b = fixtureClone(t, a)
	b.Membership.Members = append(b.Membership.Members, b.Membership.Members[0])
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b = fixtureClone(t, a)
	b.Membership.Aliases = nil
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b = fixtureClone(t, a)
	b.Membership.Aliases[0].ResolutionFactRef = ""
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b = fixtureClone(t, a)
	b.Membership.SourceFrames = nil
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b = fixtureClone(t, a)
	b.Membership.Coverage.Observation.Scope = "other-domain"
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b = fixtureClone(t, a)
	b.Header.DomainRef.Resources = nil
	b.Membership.Members = nil
	b.Membership.Aliases = nil
	b.Membership.RequiredAliasIDs = nil
	b.Membership.AffectedRuns = nil
	fixtureWant(t, ValidateRecord(b, p), RefusedUnchanged)
	b.Membership.CoverageKind = "EXPLICIT_EMPTY"
	fixtureWant(t, ValidateRecord(b, p), StructurallyCoherent)
	b.Membership.Coverage = Fact[ID]{State: Unavailable, Unavailable: &UnavailableReason{Code: "empty-roster-not-observed"}}
	fixtureWant(t, ValidateRecord(b, p), NeedsEvidence)
}

func TestRevisionVectorOwnerResourceAndIncarnation(t *testing.T) {
	p := fixtureProfile(MediaIngress)
	v := fixtureVector()
	fixtureWant(t, CompareVectors(v, fixtureClone(t, v), p), StructurallyCoherent)
	for _, tc := range []struct {
		name   string
		change func(RevisionVector)
		want   ResultKind
	}{
		{"incarnation", func(v RevisionVector) { v[0].Incarnation = "incarnation-2" }, Stale},
		{"owner", func(v RevisionVector) { v[0].Key.OwnerID = "other-owner" }, RefusedUnchanged},
		{"resource", func(v RevisionVector) { v[0].Key.Resource.ResourceID = "other-run" }, RefusedUnchanged},
		{"boot-literal", func(v RevisionVector) {
			for i := range v {
				if v[i].Key.Axis == Boot {
					*v[i].Value.Literal = "other-boot"
				}
			}
		}, Stale},
		{"job-counter", func(v RevisionVector) {
			for i := range v {
				if v[i].Key.Axis == JobAxis {
					*v[i].Value.Uint64 = 2
				}
			}
		}, Stale},
		{"connection-counter", func(v RevisionVector) {
			for i := range v {
				if v[i].Key.Axis == ConnectionAxis {
					*v[i].Value.Uint64 = 2
				}
			}
		}, Stale},
		{"missing-uint", func(v RevisionVector) { v[0].Value.Uint64 = nil }, RefusedUnchanged},
		{"string-is-not-number", func(v RevisionVector) { n := ID("1"); v[0].Value = VersionedValue{Kind: PresentLiteral, Literal: &n} }, RefusedUnchanged},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := fixtureClone(t, v)
			tc.change(current)
			fixtureWant(t, CompareVectors(v, current, p), tc.want)
		})
	}
	zero := fixtureClone(t, v)
	*zero[0].Value.Uint64 = 0
	fixtureWant(t, ValidateVector(zero, p), StructurallyCoherent)
	max := fixtureClone(t, v)
	*max[0].Value.Uint64 = math.MaxUint64
	fixtureWant(t, ValidateVector(max, p), StructurallyCoherent)
	fixtureWant(t, CompareVectors(max, zero, p), Stale)
	dup := fixtureClone(t, v)
	extra := fixtureClone(t, v[0])
	extra.Incarnation = "another-incarnation"
	dup = append(dup, extra)
	fixtureWant(t, ValidateVector(dup, p), RefusedUnchanged)
	fixtureWant(t, ValidateVector(v[1:], p), NeedsEvidence)
	unknown := fixtureClone(t, v)
	unknown[0].Value = VersionedValue{Kind: ValueUnavailable, Unavailable: &UnavailableReason{Code: "source-unavailable"}}
	unknown[0].SourceFrameRef = ""
	fixtureWant(t, ValidateVector(unknown, p), NeedsEvidence)
	unknown[0].SourceFrameRef = "invented-current-frame"
	fixtureWant(t, ValidateVector(unknown, p), RefusedUnchanged)
	// Typed JSON counters must reject overflow, signed, fractional and string values.
	record, recordProfile := fixtureRecord(CC02)
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, wire := range []string{"18446744073709551616", "-1", "1.5", "\"1\""} {
		invalid := bytes.Replace(encoded, []byte("\"Uint64\":1"), []byte("\"Uint64\":"+wire), 1)
		_, got := DecodeRecord(invalid, recordProfile)
		fixtureWant(t, got, RefusedUnchanged)
	}

}

func TestReadySealPriorWorkOnly(t *testing.T) {
	seal, p := fixtureRecord(CC05)
	completion, _ := fixtureRecord(CC04)
	fixtureWant(t, ValidateSealAgainstCompletion(seal, completion, p), StructurallyCoherent)
	for _, tc := range []struct {
		name   string
		change func(*SealRecord)
	}{
		{"release-ref", func(s *SealRecord) { s.ReleaseRef = "other-release" }},
		{"config-ref", func(s *SealRecord) { s.ConfigRef = "other-config" }},
		{"policy-ref", func(s *SealRecord) { s.PolicyRef = "other-policy" }},
		{"trust-ref", func(s *SealRecord) { s.TrustRef = "other-trust" }},
		{"missing-release-axis-key", func(s *SealRecord) { s.ReleaseAxisKey = AxisKey{} }},
		{"release-key-wrong-resource", func(s *SealRecord) { s.ReleaseAxisKey.Resource = fixtureResource(Host, "host-1") }},
	} {
		t.Run("snapshot-binding/"+tc.name, func(t *testing.T) {
			bad := fixtureClone(t, seal)
			tc.change(bad.Seal)
			fixtureWant(t, ValidateRecord(bad, p), RefusedUnchanged)
		})
	}

	for _, future := range []string{"CONSUME", "FENCE", "START", "RECEIPT", "FUTURE"} {
		t.Run(future, func(t *testing.T) {
			r := fixtureClone(t, seal)
			r.Seal.ProofSet[0].WorkPhase = future
			fixtureWant(t, ValidateRecord(r, p), RefusedUnchanged)
		})
	}
	for _, tc := range []struct {
		name   string
		change func(*Record)
	}{
		{"closure", func(r *Record) { r.Seal.ClosureID = "other-closure"; r.Seal.ProofSet[0].ClosureID = "other-closure" }},
		{"proof-set", func(r *Record) { r.Seal.ProofSet[0].Ref = "not-in-completion" }},
		{"source-vector", func(r *Record) { *r.Seal.SourceVector[0].Value.Uint64 = 2 }},
		{"membership", func(r *Record) { r.Seal.ProofSet[0].MembershipSnapshot = "other-snapshot" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := fixtureClone(t, seal)
			tc.change(&r)
			got := ValidateSealAgainstCompletion(r, completion, p)
			if got.Kind == StructurallyCoherent || got.ProductionQualified() || got.AuthorizesMutation() {
				t.Fatalf("incompatible seal accepted: %+v", got)
			}
		})
	}
}

func TestFenceObservationIsNotAuthority(t *testing.T) {
	r, p := fixtureRecord(CC06)
	fixtureWant(t, ValidateRecord(r, p), StructurallyCoherent)
	for _, axis := range []Axis{Ownership, Fence} {
		t.Run("matching-profile-still-wrong-target/"+string(axis), func(t *testing.T) {
			bad := fixtureClone(t, r)
			badProfile := fixtureClone(t, p)
			for i := range bad.Fence.ExpectedTargetVector {
				if bad.Fence.ExpectedTargetVector[i].Key.Axis == axis {
					bad.Fence.ExpectedTargetVector[i].Key.Resource = fixtureResource(Run, "run-1")
				}
			}
			for i := range badProfile.AxisRules {
				if badProfile.AxisRules[i].Key.Axis == axis {
					badProfile.AxisRules[i].Key.Resource = fixtureResource(Run, "run-1")
				}
			}
			fixtureWant(t, ValidateProfile(badProfile), StructurallyCoherent)
			// A generic vector/profile match must not replace exact target/resource binding at Final Fence.
			fixtureWant(t, ValidateVector(bad.Fence.ExpectedTargetVector, badProfile), StructurallyCoherent)
			fixtureWant(t, ValidateRecord(bad, badProfile), RefusedUnchanged)
		})
	}

	for _, tc := range []struct {
		name   string
		change func(*FenceRecord)
	}{
		{"ownership-body", func(f *FenceRecord) { f.OwnershipEpoch = "other-ownership" }},
		{"fence-body", func(f *FenceRecord) { f.FenceEpoch = "other-fence" }},
		{"ownership-vector-value", func(f *FenceRecord) {
			for i := range f.ExpectedTargetVector {
				if f.ExpectedTargetVector[i].Key.Axis == Ownership {
					*f.ExpectedTargetVector[i].Value.Literal = "other-ownership"
				}
			}
		}},
		{"fence-vector-value", func(f *FenceRecord) {
			for i := range f.ExpectedTargetVector {
				if f.ExpectedTargetVector[i].Key.Axis == Fence {
					*f.ExpectedTargetVector[i].Value.Literal = "other-fence"
				}
			}
		}},
		{"ownership-vector-resource", func(f *FenceRecord) {
			for i := range f.ExpectedTargetVector {
				if f.ExpectedTargetVector[i].Key.Axis == Ownership {
					f.ExpectedTargetVector[i].Key.Resource = fixtureResource(Run, "run-1")
				}
			}
		}},
		{"fence-vector-resource", func(f *FenceRecord) {
			for i := range f.ExpectedTargetVector {
				if f.ExpectedTargetVector[i].Key.Axis == Fence {
					f.ExpectedTargetVector[i].Key.Resource = fixtureResource(Run, "run-1")
				}
			}
		}},
	} {
		t.Run("target-vector-binding/"+tc.name, func(t *testing.T) {
			bad := fixtureClone(t, r)
			tc.change(bad.Fence)
			fixtureWant(t, ValidateRecord(bad, p), RefusedUnchanged)
		})
	}

	for _, tc := range []struct {
		name   string
		change func(*FenceRecord)
	}{
		{"no-consumption", func(f *FenceRecord) { f.AuthorityConsumptionRef = "" }},
		{"wrong-deadline-meaning", func(f *FenceRecord) { f.DeadlineSemantics = "D2_COMMIT_COMPLETION" }},
		{"implicit-service-apply", func(f *FenceRecord) { f.BackendOperation = "APPLY_AND_RESTART" }},
		{"no-fence-epoch", func(f *FenceRecord) { f.FenceEpoch = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := fixtureClone(t, r)
			tc.change(bad.Fence)
			got := ValidateRecord(bad, p)
			if got.Kind == StructurallyCoherent {
				t.Fatal("invalid fence became coherent")
			}
			if got.AuthorizesMutation() || got.ProductionQualified() {
				t.Fatal("fence DTO issued authority")
			}
		})
	}
}

func TestLegacyProjectionCannotMintEvidence(t *testing.T) {
	for _, lane := range []AdmissionLane{Open, Closing, Closed} {
		for _, drain := range []DrainState{Waiting, Proved} {
			for _, gate := range []Gate{Held, NotQualified, VerifiedForProfile} {
				got := LegacyProjection(lane, drain, gate, fixtureProfile(ClosureAssess).Ref)
				fixtureWant(t, got, NeedsDecision)
				found := false
				for _, issue := range got.Issues {
					if issue.Code == "NO_PROJECTION" {
						found = true
					}
				}
				if !found {
					t.Fatal("missing explicit NO_PROJECTION")
				}
			}
		}
	}
	fixtureWant(t, LegacyProjection(AdmissionLane("UNKNOWN"), Proved, VerifiedForProfile, fixtureProfile(ClosureAssess).Ref), RefusedUnchanged)
}

func TestIdentityClassAndPrincipalKindRemainSeparate(t *testing.T) {
	limits := fixtureLimits()
	human := IdentityContext{Class: Human, PrincipalKind: PrincipalHuman, ActorID: "human-1", Purpose: PurposeClosure}
	fixtureWant(t, ValidateIdentity(human, limits), StructurallyCoherent)
	pending := human
	pending.Class = PendingIdentity
	pending.FlowRef = "onboarding-flow-1"
	fixtureWant(t, ValidateIdentity(pending, limits), StructurallyCoherent)
	pending.PrincipalKind = PrincipalKind("PENDING")
	fixtureWant(t, ValidateIdentity(pending, limits), RefusedUnchanged)
	pending.PrincipalKind = PrincipalHuman
	pending.FlowRef = ""
	fixtureWant(t, ValidateIdentity(pending, limits), RefusedUnchanged)
	wrong := human
	wrong.PrincipalKind = PrincipalSystem
	fixtureWant(t, ValidateIdentity(wrong, limits), RefusedUnchanged)
	wrong = human
	wrong.WorkloadSubject = "workload-subject-1"
	fixtureWant(t, ValidateIdentity(wrong, limits), RefusedUnchanged)
	preauth := IdentityContext{Class: Preauth, PrincipalKind: PrincipalPreauth, ActorID: "preauth-1", Purpose: PurposeClosure, FlowRef: "preauth-flow-1"}
	fixtureWant(t, ValidateIdentity(preauth, limits), StructurallyCoherent)
	system := IdentityContext{Class: System, PrincipalKind: PrincipalSystem, ActorID: "system-1", Purpose: PurposeClosure, WorkloadSubject: "workload-subject-1"}
	fixtureWant(t, ValidateIdentity(system, limits), StructurallyCoherent)
}

func TestFixtureCannotQualifyProduction(t *testing.T) {
	for _, id := range []ContractID{CC01, CC02, CC03, CC04, CC05, CC06, CC07, CC08, CC09} {
		r, p := fixtureRecord(id)
		fixtureWant(t, ValidateRecord(r, p), StructurallyCoherent)
	}
	for _, kind := range []ResultKind{StructurallyCoherent, NeedsEvidence, NeedsDecision, RefusedUnchanged, PendingObligation, Unknown, Conflict, Stale} {
		r := Result{Kind: kind}
		if r.ProductionQualified() || r.AuthorizesMutation() {
			t.Fatalf("%s granted production capability", kind)
		}
	}
	v := ID("synthetic-only-fact")
	unavailable := Fact[ID]{State: Unavailable, Unavailable: &UnavailableReason{Code: "D4-not-decided"}}
	fixtureWant(t, ValidateFact(unavailable, fixtureLimits()), NeedsEvidence)
	unavailable.Value = &v
	fixtureWant(t, ValidateFact(unavailable, fixtureLimits()), RefusedUnchanged)
}

func TestProducerConsumerProfileConformance(t *testing.T) {
	var expected []ConsumerExpectation
	fixtureReadJSON(t, "consumer-expectations.json", &expected)
	actual := ConsumerExpectations()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("consumer register differs from independent expectation: actual=%+v expected=%+v", actual, expected)
	}
	var vectors struct {
		SpecRef                       string
		TestOnly, ProductionQualified bool
		Profiles                      []ComparisonProfile
		ExpectedVector                RevisionVector
		Cases                         []struct {
			Name, Mutation string
			Expected       ResultKind
		}
	}
	fixtureReadJSON(t, "profiles-vectors.json", &vectors)
	if vectors.SpecRef != SpecRef || !vectors.TestOnly || vectors.ProductionQualified || len(vectors.Profiles) != 7 {
		t.Fatal("test profile qualification or revision changed")
	}
	profiles := map[ProfileID]ComparisonProfile{}
	for _, p := range vectors.Profiles {
		fixtureWant(t, ValidateProfile(p), StructurallyCoherent)
		profiles[p.Ref.ProfileID] = p
	}
	repos := map[string]bool{}
	components := map[ComponentID]bool{}
	workstreams := map[string]bool{}
	for _, consumer := range expected {
		key := consumer.Repository + "/" + string(consumer.Component) + "/" + consumer.Workstream
		t.Run(key, func(t *testing.T) {
			if consumer.ContractRevision != 1 || consumer.Expected != StructurallyCoherent || consumer.ProductionQualified {
				t.Fatal("consumer fixture can qualify production")
			}
			p, ok := profiles[consumer.Profile]
			if !ok {
				t.Fatal("missing frozen shared profile")
			}
			for _, tc := range vectors.Cases {
				t.Run(tc.Name, func(t *testing.T) {
					current := fixtureClone(t, vectors.ExpectedVector)
					switch tc.Mutation {
					case "NONE":
					case "INCARNATION":
						current[0].Incarnation = "restarted-incarnation"
					case "ZERO":
						*current[0].Value.Uint64 = 0
					case "UNAVAILABLE":
						current[0].Value = VersionedValue{Kind: ValueUnavailable, Unavailable: &UnavailableReason{Code: "owner-unavailable"}}
						current[0].SourceFrameRef = ""
					case "DUPLICATE":
						extra := fixtureClone(t, current[0])
						extra.Incarnation = "different-incarnation"
						current = append(current, extra)
					case "TYPE":
						n := ID("1")
						current[0].Value = VersionedValue{Kind: PresentLiteral, Literal: &n}
					default:
						t.Fatalf("unknown independently specified mutation %q", tc.Mutation)
					}
					fixtureWant(t, CompareVectors(vectors.ExpectedVector, current, p), tc.Expected)
				})
			}
		})
		repos[consumer.Repository] = true
		components[consumer.Component] = true
		workstreams[consumer.Workstream] = true
	}
	if len(repos) != 9 || len(components) != 14 || len(workstreams) != 7 {
		t.Fatalf("incomplete consumer coverage: repos=%d components=%d WS=%d", len(repos), len(components), len(workstreams))
	}
	actual[0].ProductionQualified = true
	if ConsumerExpectations()[0].ProductionQualified {
		t.Fatal("consumer registry was mutated by caller")
	}
}

func fixtureSourceBoundaryIssues(name string, source any) []string {
	allowed := map[string]bool{"bytes": true, "encoding/json": true, "errors": true, "fmt": true, "io": true, "math": true, "reflect": true, "slices": true, "sort": true, "strconv": true, "strings": true, "unicode": true, "unicode/utf8": true}
	forbiddenFields := map[string]bool{"password": true, "passwordhash": true, "mfasecret": true, "recoverycode": true, "sessiontoken": true, "csrftoken": true, "oauthsecret": true, "privatekey": true, "encryptionkey": true, "rawpayload": true, "rawaudio": true, "notificationbody": true, "connectionurl": true}
	issues := []string{}
	f, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
	if err != nil {
		return []string{"unparseable production source"}
	}
	for _, imp := range f.Imports {
		imported, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !allowed[imported] {
			issues = append(issues, "non-pure dependency")
		}
	}
	// An ast.Field may contain multiple names: type X struct { Label, SessionToken string }.
	ast.Inspect(f, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok {
			return true
		}
		for _, name := range field.Names {
			normalized := strings.ReplaceAll(strings.ToLower(name.Name), "_", "")
			if forbiddenFields[normalized] {
				issues = append(issues, "secret or raw field")
			}
		}
		return true
	})
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "init" || fn.Name.Name == "main" {
			issues = append(issues, "runtime entrypoint")
		}
		lower := strings.ToLower(fn.Name.Name)
		constructor := strings.HasPrefix(lower, "new") || strings.HasPrefix(lower, "mint") || strings.HasPrefix(lower, "issue") || strings.HasPrefix(lower, "consume")
		authority := strings.Contains(lower, "permit") || strings.Contains(lower, "authority") || strings.Contains(lower, "recovery") || strings.Contains(lower, "qualification") || strings.Contains(lower, "approval")
		if constructor && authority {
			issues = append(issues, "authority issuer")
		}
		if fn.Name.Name == "AuthorizesMutation" || fn.Name.Name == "ProductionQualified" {
			if fn.Body == nil || len(fn.Body.List) != 1 {
				issues = append(issues, "nonconstant qualification")
				continue
			}
			ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				issues = append(issues, "nonconstant qualification")
				continue
			}
			ident, ok := ret.Results[0].(*ast.Ident)
			if !ok || ident.Name != "false" {
				issues = append(issues, "production capability can be true")
			}
		}
	}
	return issues
}
func TestSourceBoundaryNoAuthorityConstructors(t *testing.T) {
	files := 0
	err := filepath.WalkDir(".", func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		files++
		if issues := fixtureSourceBoundaryIssues(name, nil); len(issues) > 0 {
			t.Fatalf("%s violates SourceBoundary: %v", name, issues)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 5 {
		t.Fatal("production source boundary unexpectedly empty")
	}
	// Exercise the scanner independently so comma fields or raw-quoted imports cannot evade it.
	rawQuote := string(rune(96))
	for _, tc := range []struct{ name, source string }{
		{"raw-quoted-import", "package nested\nimport " + rawQuote + "net/http" + rawQuote + "\n"},
		{"second-comma-field", "package nested\ntype Record struct { Label, SessionToken string }\n"},
		{"underscore-field", "package nested\ntype Record struct { OAuth_Secret string }\n"},
		{"initializer", "package nested\nfunc init() {}\n"},
		{"entrypoint", "package nested\nfunc main() {}\n"},
		{"authority-issuer", "package nested\nfunc IssuePermit() {}\n"},
		{"approval-consumption", "package nested\nfunc ConsumeApproval() {}\n"},
		{"qualification", "package nested\nfunc ProductionQualified() bool { return true }\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(fixtureSourceBoundaryIssues("nested/guard.go", tc.source)) == 0 {
				t.Fatal("invalid source escaped AST boundary")
			}
		})
	}
	safe := "package nested\nimport \"strings\"\ntype Record struct { Reference string }\nfunc ValidateReference(v string) bool { return strings.TrimSpace(v)!=\"\" }\n"
	if issues := fixtureSourceBoundaryIssues("nested/valid.go", safe); len(issues) > 0 {
		t.Fatalf("pure source rejected: %v", issues)
	}
}

func TestBoundarySubkindRegistryClosed(t *testing.T) {
	registry := map[BoundaryKind][]SubBoundary{
		CPLifecycle:        {CPApply, ReceiptSave, CompleteACK, LocalParticipantACK},
		Ledger:             {Reserve, Finalize, LedgerReceipt, Complete},
		BackendMutation:    {ReferenceCommit, ServiceApply, MutationReceipt},
		Provider:           {TransportResponse, RemoteAccept, RemoteResult, RemoteTerminal},
		LocalPersistence:   {Intent, DispatchPossible, RawWrite, Durability, QueueAdmission, QueueTerminal, LocalDBCommit, LocalResultRecord, HistoryRecord},
		ProjectionBoundary: {CPOutcome, ExecutorOutcome, HistoryView, UIView},
	}
	for kind, subs := range registry {
		for _, sub := range subs {
			t.Run(string(kind)+"/"+string(sub), func(t *testing.T) {
				a := fixtureBoundary()
				a.Kind = kind
				a.SubBoundary = sub
				fixtureWant(t, ValidateBoundary(a, fixtureLimits()), StructurallyCoherent)
			})
		}
	}
	for _, a := range []BoundaryAddress{
		{Kind: BoundaryKind("NETWORK_SUCCESS"), OwnerID: "owner-1", Resource: fixtureResource(Run, "run-1"), SubBoundary: RemoteResult},
		{Kind: Provider, OwnerID: "owner-1", Resource: fixtureResource(Run, "run-1"), SubBoundary: SubBoundary("SUCCESS")},
		{Kind: CPLifecycle, OwnerID: "owner-1", Resource: fixtureResource(Run, "run-1"), SubBoundary: RemoteResult},
		{Kind: CPLifecycle, OwnerID: "different-owner", Resource: fixtureResource(Run, "run-1"), SubBoundary: CPApply},
	} {
		fixtureWant(t, ValidateBoundary(a, fixtureLimits()), RefusedUnchanged)
	}
}

func TestImmutableAttemptCorrelation(t *testing.T) {
	a, p := fixtureRecord(CC02)
	for _, tc := range []struct {
		name   string
		change func(*Correlation)
	}{
		{"request", func(c *Correlation) { c.RequestID = "other-request" }},
		{"operation", func(c *Correlation) { c.OperationID = "other-operation" }},
		{"attempt", func(c *Correlation) { c.AttemptID = "other-attempt" }},
		{"step", func(c *Correlation) { c.StepID = "other-step" }},
		{"execution-key", func(c *Correlation) { c.ExecutionKey = "other-execution" }},
		{"literal-no-trim", func(c *Correlation) { c.AttemptID = " attempt-1 " }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := fixtureClone(t, a)
			tc.change(&b.Header.Correlation)
			fixtureWant(t, CompareImmutable(a, b, p), Conflict)
		})
	}
	b := fixtureClone(t, a)
	b.Header.RecordRevision = 2
	fixtureWant(t, CompareImmutable(a, b, p), StructurallyCoherent)
	inspection, q := fixtureRecord(CC08)
	bad := fixtureClone(t, inspection)
	bad.Inspection.Target.ExecutionKey = "other-execution"
	fixtureWant(t, ValidateRecord(bad, q), RefusedUnchanged)
	bad = fixtureClone(t, inspection)
	bad.Inspection.Target.Correlation.AttemptID = "other-attempt"
	fixtureWant(t, ValidateRecord(bad, q), RefusedUnchanged)
	bad = fixtureClone(t, inspection)
	bad.Inspection.Target.BindingRef = "other-binding"
	fixtureWant(t, CompareImmutable(inspection, bad, q), Conflict)
	literal := fixtureClone(t, a)
	for i := range literal.Vector.Vector {
		if literal.Vector.Vector[i].Key.Axis == RunAxis {
			*literal.Vector.Vector[i].Value.Literal = "2026-11-15T00:00:00.000000001Z"
		}
	}
	fixtureWant(t, CompareImmutable(a, literal, p), Conflict)
}
