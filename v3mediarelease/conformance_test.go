package mediarelease

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The checked-in examples are literal wire records. Their seals were computed
// outside this module from synthetic source identifiers and independently fixed
// expectations. A production encoder or reducer never supplies the oracle.
func fixtureWire(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/handoff.json")
	if err != nil {
		t.Fatal(err)
	}
	var records map[string]json.RawMessage
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatal(err)
	}
	wire, ok := records[name]
	if !ok {
		t.Fatalf("missing conformance record %q", name)
	}
	return wire
}

func vector(t *testing.T, name string) Record {
	t.Helper()
	r, err := DecodeRecord(fixtureWire(t, name))
	if err != nil {
		t.Fatalf("valid vector %s: %v", name, err)
	}
	return r
}

func chain(t *testing.T, names ...string) []Record {
	t.Helper()
	result := make([]Record, 0, len(names))
	for _, name := range names {
		result = append(result, vector(t, name))
	}
	return result
}

func sealedProofs(t *testing.T) []Record { t.Helper(); return chain(t, "a1", "a2", "b", "c") }

type object = map[string]any
type wireFault func(identity, snapshot object)

// Separately recompute seals when injecting semantic faults. This ensures
// that rejecting the mutated data cannot succeed solely by detecting a stale
// digest. The corpus uses only ASCII string keys, booleans and lists, so the
// standard encoder over maps produces the independent sorted representation.
func changedWire(t *testing.T, name string, fault wireFault) []byte {
	t.Helper()
	var envelope object
	if err := json.Unmarshal(fixtureWire(t, name), &envelope); err != nil {
		t.Fatal(err)
	}
	i, s := envelope["identity"].(object), envelope["snapshot"].(object)
	fault(i, s)
	seal := func(label string, value object) string {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(append([]byte("MediaRelease1/"+label+"/v1\x00"), encoded...))
		return hex.EncodeToString(sum[:])
	}
	envelope["immutable_digest"] = seal("identity", i)
	envelope["record_digest"] = seal("record", object{"identity": i, "snapshot": s})
	out, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func badField(t *testing.T, name string, fault wireFault) {
	t.Helper()
	if _, err := DecodeRecord(changedWire(t, name, fault)); err == nil {
		t.Fatalf("%s accepted a correctly sealed invalid payload", name)
	}
}

func badHandoff(t *testing.T, names []string, fault wireFault) {
	t.Helper()
	n := len(names) - 1
	records := chain(t, names...)
	if err := ValidateHandoff(records...); err != nil {
		t.Fatalf("positive control: %v", err)
	}
	mutated, err := DecodeRecord(changedWire(t, names[n], fault))
	if err != nil {
		return
	} // A stricter field invariant may reject the fault first.
	records[n] = mutated
	if ValidateHandoff(records...) == nil {
		t.Fatal("accepted mismatched handoff")
	}
}

func TestAdmissionAuthorityVector(t *testing.T) {
	a := vector(t, "a1")
	if a.Identity.CC != "CC10" || a.Identity.Kind != "CP_ADMISSION_PROOF" || a.Identity.Purpose != "ADMISSION_AUTHORITY" || a.Identity.Issuer.ServiceID != "cp" {
		t.Fatal("A1 issuer/kind")
	}
	if *a.Identity.WorkerOriginalRef == *a.Identity.EncoderOriginalRef {
		t.Fatal("distinct original records required")
	}
	for _, key := range []string{"claim_ref", "reservation_ref", "membership_snapshot_ref", "stop_fence_ref"} {
		t.Run(key, func(t *testing.T) { badField(t, "a1", func(i, s object) { delete(i, key) }) })
	}
}

func TestWorkerBindingVector(t *testing.T) {
	proofs := chain(t, "a1", "a2")
	if err := ValidateHandoff(proofs...); err != nil {
		t.Fatal(err)
	}
	if proofs[1].Identity.CC != "CC10" || proofs[1].Identity.Kind != "WORKER_NON_MEDIA_BINDING" || proofs[1].Identity.CPAdmissionRef.Digest != proofs[0].RecordDigest {
		t.Fatal("independent worker binding")
	}
	t.Run("MissingNonMediaFence", func(t *testing.T) { badField(t, "a2", func(i, s object) { delete(i, "non_media_fence_ref") }) })
	t.Run("ReplacedReservation", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2"}, func(i, s object) { i["reservation_ref"].(object)["source_revision"] = "22" })
	})
}

func TestAppliedCoverVector(t *testing.T) {
	t.Run("WrongClosureIssuer", func(t *testing.T) {
		b := vector(t, "b")
		b.Identity.Closures[0].FenceRef.SourceOwner = b.Identity.Issuer.Peer()
		if ValidateFields(b.Identity, b.Snapshot) == nil {
			t.Fatal("encoder attested a worker-owned closure")
		}
	})
	proofs := chain(t, "a1", "a2", "b")
	if err := ValidateHandoff(proofs...); err != nil {
		t.Fatal(err)
	}
	if proofs[2].Identity.CC != "CC11" || proofs[2].Identity.AppliedActive == nil || !*proofs[2].Identity.AppliedActive {
		t.Fatal("expected applied asset")
	}
	for _, key := range []string{"closure", "qualified_sink_ref", "output_progress_ref", "feed_acceptance_ref", "asset_ref"} {
		t.Run(key, func(t *testing.T) {
			badField(t, "b", func(i, s object) {
				if key == "closure" {
					i["closures"] = []any{}
				} else {
					delete(i, key)
				}
			})
		})
	}
	t.Run("HiddenWithActiveAsset", func(t *testing.T) { badField(t, "b", func(i, s object) { i["applied_active"] = false }) })
}

func TestTargetPermitVector(t *testing.T) {
	proofs := sealedProofs(t)
	if err := ValidateHandoff(proofs...); err != nil {
		t.Fatal(err)
	}
	p := proofs[3].Identity
	if p.CC != "CC12" || p.Scope.Route != "WORKER_SRT" || len(p.Audience) != 1 || p.Audience[0] != p.Scope.TargetOwner.Peer() {
		t.Fatal("one physical target")
	}
	t.Run("WrongConsumer", func(t *testing.T) {
		badField(t, "c", func(i, s object) { i["audience"].([]any)[0].(object)["service_id"] = "encoder" })
	})
	t.Run("WrongAppliedProof", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c"}, func(i, s object) { i["applied_cover_ref"].(object)["digest"] = strings.Repeat("b", 64) })
	})
	t.Run("WrongClosureDestination", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c"}, func(i, s object) { i["scope"].(object)["destination_id"] = "other-receiver" })
	})
}

func TestConsumptionVectors(t *testing.T) {
	for _, tc := range []struct{ name, state, slot string }{
		{"d1", "NOT_CONSUMED", "5"}, {"d2", "CONSUMPTION_PENDING", "6"}, {"d3", "CONSUMED", "7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := vector(t, tc.name)
			if r.Identity.CC != "CC13" || r.Snapshot.ConsumptionState != tc.state || r.Snapshot.SlotRevision != tc.slot || r.Snapshot.EffectState != "EFFECT_UNKNOWN" {
				t.Fatal("consumption is separate from the effect")
			}
			if err := ValidateHandoff(append(sealedProofs(t), r)...); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("PendingWithoutWork", func(t *testing.T) { badField(t, "d2", func(i, s object) { delete(s, "work_registration_ref") }) })
	t.Run("ConsumedWithoutCommit", func(t *testing.T) { badField(t, "d3", func(i, s object) { delete(s, "consume_commit_id") }) })
	t.Run("PrivateState", func(t *testing.T) { badField(t, "d3", func(i, s object) { s["consumption_state"] = "DIALING" }) })
}

func TestReleaseReceiptVector(t *testing.T) {
	t.Run("WrongFrameObserver", func(t *testing.T) {
		r := vector(t, "e")
		r.Snapshot.Observations[0].FrameRef.SourceOwner.ServiceID = "other"
		if ValidateFields(r.Identity, r.Snapshot) == nil {
			t.Fatal("result evidence was attributed to the wrong observer")
		}
	})
	t.Run("MissingTrack", func(t *testing.T) {
		r := vector(t, "e")
		r.Identity.Scope.Media = []string{"audio", "video"}
		if ValidateFields(r.Identity, r.Snapshot) == nil {
			t.Fatal("one video frame satisfied both audio and video")
		}
	})
	records := chain(t, "a1", "a2", "b", "c", "d4", "e")
	if err := ValidateHandoff(records...); err != nil {
		t.Fatal(err)
	}
	receipt := records[5]
	if receipt.Identity.CC != "CC14" || receipt.Snapshot.EffectState != "RELEASE_CONFIRMED" || receipt.Identity.Scope.BoundaryID != "local-writer" || receipt.Identity.ConsumptionRef.Digest != records[4].RecordDigest {
		t.Fatal("scope or receipt anchor changed")
	}
	t.Run("NoResultEvidence", func(t *testing.T) { badField(t, "e", func(i, s object) { delete(s, "result_evidence_refs") }) })
	t.Run("UnqualifiedProfile", func(t *testing.T) {
		requireDecision(t, AssessReceipt(records[:4], records[4:5], records[5:], OutcomeProfile{}), Unknown)
	})
	t.Run("OtherConsumption", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c", "d4", "e"}, func(i, s object) { i["consumption_ref"].(object)["source_revision"] = "3" })
	})
}

func TestCanonicalWireVectors(t *testing.T) {
	for _, name := range []string{"a1", "a2", "b", "c", "d1", "d2", "d3", "d4", "e"} {
		t.Run(name, func(t *testing.T) {
			rec := vector(t, name)
			got, err := EncodeRecord(rec)
			if err != nil || !bytes.Equal(got, fixtureWire(t, name)) {
				t.Fatalf("canonical roundtrip: %v", err)
			}
		})
	}
	a := vector(t, "a1")
	if a.ImmutableDigest != "900840af9e424c19bd3975bfe631ebfd884b298f2886659b3641fe189cc83a2a" || a.RecordDigest != "2cd3bc5c7576cf40a55a8eaac954c96e0c5aab356c8bfb6381ae3412a5139def" {
		t.Fatal("fixed A1 digest")
	}
	if vector(t, "d1").Identity.OpenRequestDigest != "b84866b0832b984b1bf5ff529411eecbb77a1d28afd48df3cffa52624ffb71e0" {
		t.Fatal("fixed open digest")
	}
}

func TestImmutableEqualityVectors(t *testing.T) {
	first, later := vector(t, "d1"), vector(t, "d3")
	if eq, err := CompareImmutable(first, later); err != nil || !eq || first.RecordDigest == later.RecordDigest {
		t.Fatalf("same identity, new revision: %v", err)
	}
	a, b := vector(t, "a1"), vector(t, "a2")
	if eq, err := CompareImmutable(a, b); err != nil || eq {
		t.Fatalf("two distinct authorities: %v", err)
	}
	tampered := later
	tampered.Snapshot.SlotRevision = "9"
	if _, err := CompareImmutable(first, tampered); err == nil {
		t.Fatal("unverified payload compared equal")
	}
}

func TestWireFormatRejections(t *testing.T) {
	raw := fixtureWire(t, "a1")
	faults := []struct {
		name string
		data []byte
	}{
		{"LeadingSpace", append([]byte(" "), raw...)},
		{"TrailingNewline", append(append([]byte{}, raw...), '\n')},
		{"DuplicateMember", bytes.Replace(raw, []byte(`"cc":"CC10"`), []byte(`"cc":"CC10","cc":"CC10"`), 1)},
		{"EscapedMember", bytes.Replace(raw, []byte(`"cc"`), []byte(`"\u0063c"`), 1)},
		{"WrongSeal", bytes.Replace(raw, []byte("2cd3bc5c"), []byte("3cd3bc5c"), 1)},
		{"UnknownNested", changedWire(t, "a1", func(i, s object) { i["domain"].(object)["new_axis"] = "x" })},
		{"WrongCase", changedWire(t, "a1", func(i, s object) { i["Claim_ref"] = i["claim_ref"]; delete(i, "claim_ref") })},
		{"NonCanonicalUint", changedWire(t, "a1", func(i, s object) { i["admission_epoch"] = "07" })},
		{"NonCanonicalTime", changedWire(t, "a1", func(i, s object) { i["issued_at"] = "2030-01-01T00:00:01Z" })},
		{"NonString", changedWire(t, "a1", func(i, s object) { i["member_revision"] = 1 })},
		{"UnsortedSet", changedWire(t, "a1", func(i, s object) { a := i["audience"].([]any); a[0], a[1] = a[1], a[0] })},
		{"DuplicateSet", changedWire(t, "a1", func(i, s object) { a := i["audience"].([]any); a[1] = a[0] })},
	}
	for _, tc := range faults {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeRecord(tc.data); err == nil {
				t.Fatal("invalid wire accepted")
			}
		})
	}
}

func TestUnknownVersionAndFamily(t *testing.T) {
	for key, value := range map[string]string{"wire_version": "2", "family": "Lifecycle1", "profile_id": "MEDIA_RELEASE_HANDOFF_V2", "profile_revision": "2", "encoding_id": "FUTURE_ENCODING", "cc": "CC03"} {
		t.Run(key, func(t *testing.T) { badField(t, "a1", func(i, s object) { i[key] = value }) })
	}
}
func TestSameAttemptHandoff(t *testing.T) {
	for _, key := range []string{"attempt_id", "run_id", "stream_id", "request_id", "execution_key"} {
		t.Run(key, func(t *testing.T) {
			badHandoff(t, []string{"a1", "a2"}, func(i, s object) { i["original"].(object)[key] = "other" })
		})
	}
}

func TestPhysicalOwnerCorrelation(t *testing.T) {
	t.Run("Boot", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c"}, func(i, s object) { i["scope"].(object)["target_owner"].(object)["boot_id"] = "successor" })
	})
	t.Run("OtherWriter", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c", "d3"}, func(i, s object) { i["issuer"].(object)["writer_id"] = "unrelated" })
	})
	t.Run("OtherDestination", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b", "c", "d4", "e"}, func(i, s object) { i["scope"].(object)["destination_id"] = "other" })
	})
}

func TestWorkerGenerationCorrelation(t *testing.T) {
	t.Run("PreparationIsNotWorkerGeneration", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2", "b"}, func(i, s object) { i["worker_generation"] = i["preparation_generation"] })
	})
	t.Run("AssignmentABA", func(t *testing.T) {
		badHandoff(t, []string{"a1", "a2"}, func(i, s object) { i["worker_assignment"].(object)["ownership_epoch"] = "5" })
	})
}

func TestRevisionCASVectors(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want Decision
	}{
		{"d1", "d2", Coherent}, {"d2", "d3", Coherent}, {"d3", "d4", Coherent}, {"d1", "d3", Conflict}, {"d3", "d2", Conflict}, {"d3", "d3", Duplicate},
	} {
		t.Run(tc.a+"-"+tc.b, func(t *testing.T) {
			if got := CompareRevision(vector(t, tc.a), vector(t, tc.b)); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// modelRef mirrors the public reference fields for a literal stored record. It
// copies the independently sealed digest from the fixture.
func modelRef(r Record) Ref {
	return Ref{SourceFamily: r.Identity.Family, SourceVersion: r.Identity.WireVersion, SourceType: r.Identity.Kind,
		SourceOwner: r.Identity.Issuer.Peer(), SourceKey: r.Identity.RecordKey, SourceRevision: r.Snapshot.Revision,
		Representation: "MR1_RECORD_V1", Digest: r.RecordDigest}
}

// liveView is synthetic evidence from the four distinct owners. Every test
// mutates one fact, keeping the original two-index snapshot and proofs stable.
func liveView(proofs []Record, previous, next Record) ConsumptionView {
	a1, a2, b, c := proofs[0].Identity, proofs[1].Identity, proofs[2].Identity, proofs[3].Identity
	roster := *b.GraphConfigRef
	roster.SourceKey.RecordID = "closed-outlet-roster"
	rosterSum := sha256.Sum256([]byte("fixture/closed-outlet-roster"))
	roster.Digest = hex.EncodeToString(rosterSum[:])
	v := ConsumptionView{AuthenticatedOwner: c.Scope.TargetOwner, Predecessor: modelRef(previous),
		Freshness: *previous.Snapshot.FreshnessRef, Authority: *c.AuthorityRef, Revocation: *c.RevocationRef,
		StopFence: *c.StopFenceRef, Lease: *c.LeaseRef, Reservation: *a1.ReservationRef,
		WorkRegistration: *next.Snapshot.WorkRegistrationRef, Dispatch: *next.Snapshot.DispatchRef,
		WorkerAssignment: *a1.WorkerAssignment, EncoderAssignment: *a1.EncoderAssignment,
		WorkerGeneration: a2.WorkerGeneration, Unit: *c.Scope, SlotRevision: previous.Snapshot.SlotRevision,
		Now: next.Snapshot.CheckedAt, ClockErrorNS: "0", LiveContinuation: true, LedgerComplete: true, Current: true,
		ClosuresVerified: true, UnitRegistered: true}
	v.ClockAuthorities = []Ref{a1.Window.ClockAuthorityRef, a2.Window.ClockAuthorityRef, b.Window.ClockAuthorityRef}
	v.GraphOutlets = GraphOutlets{ConfigRef: *b.GraphConfigRef, RosterRef: roster, IsolatedSinkRef: b.QualifiedSinkRef}
	if previous.Snapshot.ConsumptionState == "CONSUMPTION_PENDING" {
		v.ExistingClaims = []Record{proofs[3]}
	}
	return v
}

func requireDecision(t *testing.T, got, want Decision) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReleaseWindowBoundaries(t *testing.T) {
	proofs := sealedProofs(t)
	old, pending := vector(t, "d1"), vector(t, "d2")
	v := liveView(proofs, old, pending)
	requireDecision(t, ReduceConsumption(proofs, old, pending, v), Coherent)
	for _, tc := range []struct {
		name, at string
		want     Decision
	}{
		{"JustBeforeExpiry", "2030-01-01T00:00:54.999999999Z", Coherent},
		{"AtExpiry", "2030-01-01T00:00:55.000000000Z", Held},
		{"AfterExpiry", "2030-01-01T00:00:55.000000001Z", Held},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate, err := DecodeRecord(changedWire(t, "d2", func(i, s object) { s["checked_at"] = tc.at; s["recorded_at"] = tc.at }))
			if err != nil {
				t.Fatal(err)
			}
			ctx := liveView(proofs, old, candidate)
			requireDecision(t, ReduceConsumption(proofs, old, candidate, ctx), tc.want)
		})
	}
	t.Run("UntrustedClock", func(t *testing.T) {
		v.ClockErrorNS = "1"
		requireDecision(t, ReduceConsumption(proofs, old, pending, v), Held)
	})
}

func TestRevocationAndStopFence(t *testing.T) {
	t.Run("PrePermitRevocationIssuer", func(t *testing.T) {
		c := vector(t, "c")
		cp := c.Identity.Issuer
		ref := *c.Identity.AuthorityRef
		scope := *c.Identity.Scope
		obs := Obs{Scope: scope, Fact: "CP_REVOCATION_RECORDED", Knowledge: "KNOWN", ObservedAt: c.Snapshot.RecordedAt, Observer: cp, FrameRef: &ref}
		f := FenceObservation{Original: c.Identity.Original, Scope: scope, Responder: cp,
			ObservedAt: c.Snapshot.RecordedAt, Observations: []Obs{obs}, AuthorityRef: &ref, SlotKey: c.Identity.ConsumptionKey, DecisionRef: &ref}
		if err := ValidateFenceObservation(f); err != nil {
			t.Fatalf("valid issuer: %v", err)
		}
		wrongAuthority := ref
		wrongAuthority.SourceOwner = scope.TargetOwner.Peer()
		f.AuthorityRef = &wrongAuthority
		if ValidateFenceObservation(f) == nil {
			t.Fatal("accepted a revocation attributed to the gate as CP")
		}
	})
	proofs := sealedProofs(t)
	old, next := vector(t, "d2"), vector(t, "d3")
	requireDecision(t, ReduceConsumption(proofs, old, next, liveView(proofs, old, next)), Coherent)
	for _, tc := range []struct {
		name   string
		change func(*ConsumptionView)
	}{
		{"CPRevoked", func(v *ConsumptionView) { v.Revoked = true }},
		{"OwnerStopped", func(v *ConsumptionView) { v.Stopped = true }},
		{"ReplacedOwner", func(v *ConsumptionView) { v.AuthenticatedOwner.BootID = "later" }},
		{"AssignmentABA", func(v *ConsumptionView) { v.WorkerAssignment.OwnershipEpoch = "4" }},
		{"StaleGeneration", func(v *ConsumptionView) { v.WorkerGeneration = "12" }},
		{"FenceUnknown", func(v *ConsumptionView) { v.Current = false }},
		{"NoOutletRoster", func(v *ConsumptionView) { v.GraphOutlets.RosterRef = Ref{} }},
		{"UnboundClockAuthority", func(v *ConsumptionView) { v.ClockAuthorities = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := liveView(proofs, old, next)
			tc.change(&v)
			requireDecision(t, ReduceConsumption(proofs, old, next, v), Held)
		})
	}
}

func TestDuplicateReleaseRefused(t *testing.T) {
	proofs := sealedProofs(t)
	old, next := vector(t, "d1"), vector(t, "d2")
	v := liveView(proofs, old, next)
	t.Run("BothIndexesEmpty", func(t *testing.T) { requireDecision(t, ReduceConsumption(proofs, old, next, v), Coherent) })
	t.Run("Replay", func(t *testing.T) {
		x := v
		x.ExistingClaims = []Record{proofs[3]}
		requireDecision(t, ReduceConsumption(proofs, old, next, x), Duplicate)
	})
	for _, tc := range []struct {
		name  string
		fault wireFault
	}{
		{"SamePermitChangedAttempt", func(i, s object) { i["original"].(object)["attempt_id"] = "later" }},
		{"NewIDsSamePhysicalSlot", func(i, s object) {
			i["record_key"].(object)["record_id"] = "second"
			i["logical_effect_key"] = "second-slot-id"
			i["permit_id"] = "second-permit"
			i["original"].(object)["operation_id"] = "second-op"
		}},
		{"OverlappingTracks", func(i, s object) {
			i["record_key"].(object)["record_id"] = "extra"
			i["permit_id"] = "new"
			i["logical_effect_key"] = "new-slot"
			i["scope"].(object)["media"] = []string{"audio", "video"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other, err := DecodeRecord(changedWire(t, "c", tc.fault))
			if err != nil {
				t.Fatal(err)
			}
			x := v
			x.ExistingClaims = []Record{other}
			requireDecision(t, ReduceConsumption(proofs, old, next, x), Conflict)
		})
	}
}

func TestRestartAfterConsumption(t *testing.T) {
	proofs := sealedProofs(t)
	d1, d2, d3 := vector(t, "d1"), vector(t, "d2"), vector(t, "d3")
	for _, tc := range []struct {
		name  string
		fault func(*ConsumptionView)
		want  Decision
	}{
		{"LostContinuation", func(v *ConsumptionView) { v.LiveContinuation = false }, Held},
		{"Restart", func(v *ConsumptionView) { v.Restarted = true }, Held},
		{"NoFreshSnapshot", func(v *ConsumptionView) { v.LedgerComplete = false }, Unknown},
		{"RolledBackClaim", func(v *ConsumptionView) { v.ExistingClaims = nil }, Unknown},
		{"WrongPredecessor", func(v *ConsumptionView) { v.Predecessor = modelRef(d1) }, Conflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := liveView(proofs, d2, d3)
			tc.fault(&v)
			requireDecision(t, ReduceConsumption(proofs, d2, d3, v), tc.want)
		})
	}
	t.Run("LostCommitACK", func(t *testing.T) {
		requireDecision(t, CompareRevision(d3, d3), Duplicate)
		if d3.Snapshot.EffectState != "EFFECT_UNKNOWN" {
			t.Fatal("consumption claimed effect success")
		}
	})
}

func TestUnknownEffectAndPartialWrite(t *testing.T) {
	proofs := sealedProofs(t)
	r := vector(t, "e")
	scope := *proofs[3].Identity.Scope
	obs := r.Snapshot.Observations[0]
	full := OutcomeProfile{Scope: scope, Reference: scope.SuccessProfileRef, EffectState: "RELEASE_CONFIRMED", Verified: true,
		Criteria: []TerminalCriterion{{Media: "video", Fact: obs.Fact, Observer: obs.Observer, Frame: *obs.FrameRef}}}
	t.Run("ScopedLocalObservation", func(t *testing.T) {
		requireDecision(t, AssessOutcome(scope, []Obs{obs}, "RELEASE_CONFIRMED", full), Coherent)
	})
	t.Run("Unknown", func(t *testing.T) {
		requireDecision(t, AssessOutcome(scope, []Obs{obs}, "EFFECT_UNKNOWN", full), Unknown)
	})
	t.Run("PartialWrite", func(t *testing.T) {
		partial := obs
		partial.Fact = "PARTIAL_BYTES"
		requireDecision(t, AssessOutcome(scope, []Obs{partial}, "RELEASE_CONFIRMED", full), Unknown)
	})
	t.Run("OtherObserver", func(t *testing.T) {
		other := obs
		other.Observer.BootID = "later"
		requireDecision(t, AssessOutcome(scope, []Obs{other}, "RELEASE_CONFIRMED", full), Unknown)
	})
	t.Run("RemoteTerminal", func(t *testing.T) {
		remote := full
		remote.Criteria = []TerminalCriterion{{Media: "video", Fact: "REMOTE_TERMINAL", Observer: obs.Observer, Frame: *obs.FrameRef}}
		requireDecision(t, AssessOutcome(scope, []Obs{obs}, "RELEASE_CONFIRMED", remote), Unknown)
	})
	t.Run("Unconsumed", func(t *testing.T) { badField(t, "d1", func(i, s object) { s["effect_state"] = "RELEASE_CONFIRMED" }) })
}

func TestOriginalReceiptInspection(t *testing.T) {
	e := vector(t, "e")
	q := InspectQuery{LookupKey: e.Identity.RecordKey, Original: e.Identity.Original, Owner: e.Identity.Issuer}
	ref := modelRef(e)
	found := InspectResult{LookupKey: q.LookupKey, Original: q.Original, Responder: q.Owner, Status: "FOUND", Currentness: "CURRENT",
		ObservedAt: e.Snapshot.RecordedAt, FoundRef: &ref, StorageWitnessRef: &e.Identity.ClockRef, FreshnessRef: &e.Identity.ClockRef}
	t.Run("OriginalKeyAfterACKLoss", func(t *testing.T) { requireDecision(t, InspectOriginal(q, found, &e), Coherent) })
	t.Run("MissingRecord", func(t *testing.T) { requireDecision(t, InspectOriginal(q, found, nil), Unknown) })
	t.Run("UnknownStore", func(t *testing.T) {
		x := InspectResult{LookupKey: q.LookupKey, Original: q.Original, Responder: q.Owner, Status: "UNKNOWN", Currentness: "UNKNOWN", ObservedAt: e.Snapshot.RecordedAt}
		requireDecision(t, InspectOriginal(q, x, nil), Unknown)
	})
	t.Run("OtherAttempt", func(t *testing.T) {
		other := q
		other.Original.AttemptID = "successor"
		requireDecision(t, InspectOriginal(other, found, &e), Conflict)
	})
	t.Run("DifferentExpectation", func(t *testing.T) {
		other := q
		other.ExpectedDigest = strings.Repeat("b", 64)
		requireDecision(t, InspectOriginal(other, found, &e), Conflict)
	})
	t.Run("RetainedAfterRevocation", func(t *testing.T) {
		x := found
		x.Currentness = "REVOKED"
		x.ReasonRefs = []Ref{e.Identity.ClockRef}
		x.FreshnessRef = nil
		requireDecision(t, InspectOriginal(q, x, &e), Coherent)
	})
}

// Offers spell out the required operations independently of the production
// registry, including both pure Inspect calls on the physical sender.
func negotiatedVectors(t *testing.T) ([]Participant, []Capability) {
	t.Helper()
	a1, a2, b, c := vector(t, "a1"), vector(t, "a2"), vector(t, "b"), vector(t, "c")
	tuple := VersionTuple{Family: "MediaRelease1", WireVersion: "1", ProfileID: "MEDIA_RELEASE_HANDOFF_V1", ProfileRevision: "1", EncodingID: "MR1_JSON_ASCII_V1"}
	var required []Participant
	var offered []Capability
	for _, p := range []struct {
		role       string
		record     Record
		operations []string
	}{
		{"CP", a1, []string{"getMediaReleaseCapabilities", "issueStartAdmissionProof", "issueMediaReleasePermit", "revokeMediaReleasePermit", "inspectMediaReleaseRecord"}},
		{"WORKER", a2, []string{"getMediaReleaseCapabilities", "bindNonMediaWorker", "inspectMediaReleaseRecord"}},
		{"ENCODER", b, []string{"getMediaReleaseCapabilities", "attestAppliedCover", "inspectMediaReleaseRecord"}},
		{"GATE", a2, []string{"getMediaReleaseCapabilities", "openMediaRelease", "fenceMediaRelease", "inspectMediaReleaseRecord", "inspectMediaReleaseReceipt"}},
	} {
		who := Participant{Role: p.role, Owner: p.record.Identity.Issuer, Original: c.Identity.Original, Scope: *c.Identity.Scope}
		body, err := json.Marshal(struct {
			Participant Participant
			Version     VersionTuple
			Operations  []string
		}{who, tuple, p.operations})
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		ref := Ref{SourceFamily: "FixtureSource", SourceVersion: "1", SourceType: "CAPABILITY_SNAPSHOT", SourceOwner: who.Owner.Peer(),
			SourceKey:      Key{DeploymentID: c.Identity.Domain.DeploymentID, OwnerID: who.Owner.ServiceID, Kind: "CAPABILITY_SNAPSHOT", RecordID: "capability-" + p.role},
			SourceRevision: "1", Representation: "OWNER_SEALED_BYTES", Digest: hex.EncodeToString(sum[:])}
		required = append(required, who)
		offered = append(offered, Capability{Participant: who, Versions: []VersionTuple{tuple}, Operations: p.operations, SnapshotRef: ref,
			Authenticated: true, ObservedAt: "2030-01-01T00:00:01.000000000Z", ExpiresAt: "2030-01-01T00:00:55.000000000Z"})
	}
	return required, offered
}

func TestNegotiatedFamily(t *testing.T) {
	req, offers := negotiatedVectors(t)
	const now = "2030-01-01T00:00:08.000000000Z"
	t.Run("AllRoles", func(t *testing.T) { requireDecision(t, Negotiate(req, offers, now), Coherent) })
	t.Run("ReusedRoleSnapshot", func(t *testing.T) {
		x := append([]Capability(nil), offers...)
		x[3].SnapshotRef = x[1].SnapshotRef
		requireDecision(t, Negotiate(req, x, now), Conflict)
	})
	t.Run("ClockIsNotRoleSnapshot", func(t *testing.T) {
		x := append([]Capability(nil), offers...)
		x[3].SnapshotRef = vector(t, "a2").Identity.ClockRef
		requireDecision(t, Negotiate(req, x, now), Held)
	})
	t.Run("MissingIndependentOwner", func(t *testing.T) { requireDecision(t, Negotiate(req, offers[:3], now), Held) })
	t.Run("Expired", func(t *testing.T) { requireDecision(t, Negotiate(req, offers, "2030-01-01T00:00:55.000000000Z"), Held) })
}
func TestMixedAndOldConsumers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fault func(*Capability)
	}{
		{"OldFamily", func(c *Capability) { c.Versions[0].Family = "Lifecycle1" }},
		{"FutureOnly", func(c *Capability) { c.Versions[0].WireVersion = "2" }},
		{"UntrustedPeer", func(c *Capability) { c.Authenticated = false }},
		{"StaleBoot", func(c *Capability) { c.Participant.Owner.BootID = "previous" }},
		{"MissingInspect", func(c *Capability) { c.Operations = c.Operations[:len(c.Operations)-1] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, offers := negotiatedVectors(t)
			tc.fault(&offers[3])
			requireDecision(t, Negotiate(req, offers, "2030-01-01T00:00:08.000000000Z"), Held)
		})
	}
}

func TestSyntheticFactsNeverAuthorize(t *testing.T) {
	for _, outcome := range []Decision{Coherent, Held, Unknown, Duplicate, Conflict, Decision("STARTED")} {
		t.Run(string(outcome), func(t *testing.T) {
			if outcome.ProductionQualified() || outcome.AuthorizesMutation() {
				t.Fatal("structural comparison qualified an external action")
			}
		})
	}
}
