package lifecycle

import (
	"encoding/json"
	"testing"
)

// These small bounds and synthetic facts are test inputs, never deployment defaults.
func fixtureLimits() Limits {
	return Limits{MaxIDBytes: 192, MaxMembers: 64, MaxAxes: 64, MaxRefs: 128, MaxObligations: 32, MaxPartitions: 16, MaxRanges: 32, MaxDepth: 32}
}
func fixtureResource(k ResourceKind, id ID) ResourceRef {
	return ResourceRef{Kind: k, OwnerID: "owner-1", ResourceID: id}
}
func fixtureResources() []ResourceRef {
	return []ResourceRef{fixtureResource(Run, "run-1"), fixtureResource(Partition, "partition-1"), fixtureResource(Sink, "sink-1"), fixtureResource(Host, "host-1"), fixtureResource(Projection, "projection-1")}
}
func fixtureObservation() *ObservationContext {
	return &ObservationContext{OwnerID: "owner-1", FrameRef: "frame-1", Scope: "domain-1", SourceRevision: 1}
}
func fixtureFact(v ID) Fact[ID] {
	return Fact[ID]{State: Observed, Value: &v, Observation: fixtureObservation()}
}
func fixtureOutcome(v Outcome) Fact[Outcome] {
	return Fact[Outcome]{State: Observed, Value: &v, Observation: fixtureObservation()}
}
func fixtureAbsent() OptionalRef {
	return OptionalRef{State: Absent, AbsenceFactRef: "absence-fact-1", Observation: fixtureObservation()}
}
func fixturePresent(v ID) OptionalRef {
	return OptionalRef{State: Present, Ref: &v, Observation: fixtureObservation()}
}
func fixtureProducer() ProducerRef {
	return ProducerRef{ComponentID: "C01", ServiceID: "owner-1", HostID: "host-1", BootID: "boot-1", ProcessIncarnation: "process-1", WriterID: "writer-1"}
}
func fixtureCorrelation() Correlation {
	return Correlation{RequestID: "request-1", OperationID: "operation-1", AttemptID: "attempt-1", StepID: "step-1", ExecutionKey: "execution-1"}
}
func fixtureFrame() SourceFrame {
	return SourceFrame{FrameRef: "frame-1", OwnerID: "owner-1", Scope: "domain-1", SourceRevision: 1}
}
func fixtureBoundary() BoundaryAddress {
	return BoundaryAddress{Kind: CPLifecycle, OwnerID: "owner-1", Resource: fixtureResource(Run, "run-1"), SubBoundary: CPApply}
}
func fixtureAxes() []Axis {
	return []Axis{ARevision, CEpoch, Identity, Membership, Admission, Ownership, Fence, RunAxis, JobAxis, ConnectionAxis, Writer, Effect, Cover, Policy, Trust, Boot, Process, ReleaseAxis, ConfigAxis}
}
func fixtureIsUint(a Axis) bool {
	switch a {
	case ARevision, CEpoch, Identity, Membership, JobAxis, ConnectionAxis, Writer, Effect, Cover:
		return true
	}
	return false
}
func fixtureAxisResource(a Axis) ResourceRef {
	switch a {
	case Ownership, Fence, Boot, Process:
		return fixtureResource(Host, "host-1")
	}
	return fixtureResource(Run, "run-1")
}
func fixtureAxisKey(a Axis) AxisKey {
	return AxisKey{Axis: a, OwnerID: "owner-1", Resource: fixtureAxisResource(a)}
}
func fixtureLiteral(a Axis) ID {
	// Independent input values follow the explicit semantic field represented by each axis.
	switch a {
	case Admission:
		return "admission-1"
	case Ownership:
		return "ownership-1"
	case Fence:
		return "fence-1"
	case RunAxis:
		return "run-epoch-1"
	case Policy:
		return "policy-1"
	case Trust:
		return "trust-1"
	case Boot:
		return "boot-1"
	case Process:
		return "process-1"
	case ReleaseAxis:
		return "release-1"
	case ConfigAxis:
		return "config-1"
	}
	return ""
}
func fixtureProfile(id ProfileID) ComparisonProfile {
	purpose := map[ProfileID]Purpose{ApprovalLocal: PurposeApproval, MediaIngress: PurposeIngress, ClosureAssess: PurposeClosure, StopLookup: PurposeLookup, EffectOutcome: PurposeEffect, RecordRepair: PurposeRepair, ExplicitResume: PurposeResume}[id]
	p := ComparisonProfile{Ref: ProfileRef{ProfileID: id, Revision: 1}, ResourceKinds: []ResourceKind{Run, Partition, Sink, Host, Projection}, Purposes: []Purpose{purpose}, RequiredBoundaries: []BoundaryAddress{fixtureBoundary()}, TerminalPolicyRef: "test-terminal-policy-1", LimitsRef: "test-limits-1", Limits: fixtureLimits()}
	for _, a := range fixtureAxes() {
		kind := PresentLiteral
		if fixtureIsUint(a) {
			kind = PresentUint64
		}
		p.AxisRules = append(p.AxisRules, AxisRule{Key: fixtureAxisKey(a), OwnerRole: "C01", ValueKind: kind, Required: true, Comparison: "EXACT"})
	}
	return p
}
func fixtureVector() RevisionVector {
	v := RevisionVector{}
	for _, a := range fixtureAxes() {
		value := VersionedValue{Kind: PresentLiteral}
		if fixtureIsUint(a) {
			n := uint64(1)
			value = VersionedValue{Kind: PresentUint64, Uint64: &n}
		} else {
			s := fixtureLiteral(a)
			value.Literal = &s
		}
		v = append(v, RevisionEntry{Key: fixtureAxisKey(a), Incarnation: "incarnation-1", Value: value, SourceFrameRef: "frame-1"})
	}
	return v
}
func fixtureHeader(id ContractID, p ComparisonProfile) Header {
	ref := ContractRef{ContractID: id, LogicalRevision: 1, SpecRef: SpecRef, ModuleCommitRef: "1111111111111111111111111111111111111111"}
	return Header{ContractRef: ref, RecordKey: RecordKey{ContractRef: ref, OwnerID: "owner-1", RecordID: ID("record-" + string(id))}, Correlation: fixtureCorrelation(), DomainRef: DomainRef{DeploymentID: "test-deployment-1", Realm: "test-realm-1", DomainID: "domain-1", MembershipRevision: 1, SnapshotRef: "membership-1", Resources: fixtureResources()}, ProducerRef: fixtureProducer(), RecordRevision: 1, ProfileRef: p.Ref}
}
func fixtureNamespace() Namespace {
	return Namespace{OwnerID: "owner-1", Resource: fixtureResource(Partition, "partition-1"), WorkKind: "MEDIA", ProducerIncarnation: "process-1", SequenceEpoch: "sequence-epoch-1"}
}
func fixtureFrontier(last uint64) Frontier {
	return Frontier{Kind: "THROUGH", Namespace: fixtureNamespace(), LastSequence: &last}
}
func fixtureRegistration() RegistrationClosure {
	return RegistrationClosure{MembershipSnapshot: "membership-1", AdmissionEpoch: "admission-1", Namespace: fixtureNamespace(), RegistrationEpoch: "registration-1", OwnerID: "owner-1", DurableFactRef: "registration-closed-1"}
}
func fixtureReservations() ReservationRefs {
	return ReservationRefs{DomainClosureRef: "domain-closure-1", CPReservation: fixturePresent("cp-reservation-1")}
}
func fixturePartition() PartitionProof {
	return PartitionProof{Namespace: fixtureNamespace(), CutFrontier: fixtureFrontier(1), FinalAcceptedFrontier: fixtureFrontier(2), TerminalFrontier: fixtureFrontier(2), RegistrationClosed: fixtureRegistration()}
}
func fixtureRecord(id ContractID) (Record, ComparisonProfile) {
	pid := ClosureAssess
	switch id {
	case CC02, CC03:
		pid = MediaIngress
	case CC07:
		pid = EffectOutcome
	case CC08:
		pid = StopLookup
	case CC09:
		pid = RecordRepair
	}
	p := fixtureProfile(pid)
	r := Record{Header: fixtureHeader(id, p)}
	switch id {
	case CC01:
		m := MembershipRecord{DomainKind: "MEDIA_RUN", OwnerID: "owner-1", Revision: 1, PreviousSnapshot: fixtureAbsent(), CoverageKind: "COMPLETE_ROSTER", Coverage: fixtureFact("coverage-1"), SourceFrames: []SourceFrame{fixtureFrame()}, RequiredAliasIDs: []ID{"shared-host-alias-1"}, Aliases: []Alias{{AliasID: "shared-host-alias-1", Resource: fixtureResource(Host, "host-1"), ResolutionFactRef: "alias-resolution-1"}}, AffectedRuns: []ResourceRef{fixtureResource(Run, "run-1")}}
		for i, res := range fixtureResources() {
			prod := fixtureProducer()
			prod.WriterID = ID("writer-member-" + string(rune('a'+i)))
			m.Members = append(m.Members, Member{Producer: prod, Resource: res, SourceFrameRef: "frame-1"})
		}
		r.Membership = &m
	case CC02:
		r.Vector = &VectorRecord{Vector: fixtureVector()}
	case CC03:
		r.Admission = &AdmissionRecord{ReservationRefs: fixtureReservations(), MembershipSnapshot: "membership-1", AdmissionEpoch: "admission-1", ActorGuardRefs: []ID{"guard-1"}, Cut: []Frontier{fixtureFrontier(1)}, TerminalPolicyRef: "test-terminal-policy-1"}
	case CC04:
		r.Completion = &CompletionRecord{ClosureID: "closure-1", ReservationRefs: fixtureReservations(), SourceVector: fixtureVector(), Partitions: []PartitionProof{fixturePartition()}, LineageClosureFacts: []RegistrationClosure{fixtureRegistration()}, TerminalPolicyRef: "test-terminal-policy-1", ProofRefs: []ID{"proof-1"}}
	case CC05:
		r.Seal = &SealRecord{SealID: "seal-1", ClosureID: "closure-1", ReservationRefs: fixtureReservations(), SourceVector: fixtureVector(), ProofSet: []ProofRef{{Ref: "proof-1", ClosureID: "closure-1", MembershipSnapshot: "membership-1", WorkPhase: "PRIOR"}}, ReleaseRef: "release-1", ConfigRef: "config-1", PolicyRef: "policy-1", TrustRef: "trust-1", ReleaseAxisKey: fixtureAxisKey(ReleaseAxis), ConfigAxisKey: fixtureAxisKey(ConfigAxis), PolicyAxisKey: fixtureAxisKey(Policy), TrustAxisKey: fixtureAxisKey(Trust)}
	case CC06:
		r.Fence = &FenceRecord{AuthorityConsumptionRef: "consumption-ref-1", CanonicalRef: "canonical-ref-1", SealRef: "seal-1", ExpectedTargetVector: fixtureVector(), Target: fixtureResource(Host, "host-1"), OwnershipEpoch: "ownership-1", FenceEpoch: "fence-1", DecisionTimeRef: "unqualified-time-reference-1", BudgetRef: "unqualified-budget-reference-1", BackendOperation: "REFERENCE_ONLY", DeadlineSemantics: "D1_GUARD_DECISION"}
	case CC07:
		r.Effect = &EffectRecord{EffectID: "effect-1", ParentEffect: fixtureAbsent(), Target: fixtureResource(Sink, "sink-1"), Purpose: PurposeEffect, ContentRef: ContentRef{Kind: "NONSECRET_METADATA_REF", Ref: "nonsecret-metadata-1"}, SourceVector: fixtureVector(), IntentRevision: 1, SinkRefs: []ResourceRef{fixtureResource(Sink, "sink-1")}, ProjectionRefs: []ID{"projection-1"}}
	case CC08:
		r.Inspection = &InspectionRecord{Target: QueryTarget{Correlation: fixtureCorrelation(), BindingRef: "binding-1", ExecutionKey: "execution-1", TargetRecordRef: "original-record-1", Scope: "domain-1", Purpose: PurposeLookup}, Observations: ObservationSet{Observations: []BoundaryObservation{{Address: fixtureBoundary(), Knowledge: Known, Presence: Present, Outcome: fixtureOutcome(SuccessAtBoundary)}}, Gate: VerifiedForProfile}, ObservedScopes: []ID{"domain-1"}, SourceFrames: []SourceFrame{fixtureFrame()}, Coverage: fixtureFact("coverage-1"), HistoricalRefs: []ID{"history-1"}}
	case CC09:
		r.Reconcile = &ReconcileRecord{PlanID: "plan-1", OriginalCorrelation: fixtureCorrelation(), Purpose: PurposeRepair, SourceReceiptRefs: []ID{"receipt-1"}, SourceWitnessRefs: []ID{"witness-1"}, CurrentSourceVector: fixtureVector(), ExpectedProjectionRevision: 1, ExpectedProjection: fixtureResource(Projection, "projection-1"), IntendedMetadataDelta: MetadataDelta{Kind: "FIXED_RECEIPT_REFERENCE", SourceReceiptRef: "receipt-1", MetadataRef: "nonsecret-metadata-1"}, RequiredRecordCapability: "fresh-record-capability-reference-1", PlanRevision: 1, Freshness: fixtureFact("independent-freshness-1")}
	}
	return r, p
}
func fixtureClone[T any](t *testing.T, v T) T {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var out T
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func fixtureWant(t *testing.T, r Result, want ResultKind) {
	t.Helper()
	if r.Kind != want {
		t.Fatalf("got %s issues=%+v; want %s", r.Kind, r.Issues, want)
	}
	if r.ProductionQualified() || r.AuthorizesMutation() {
		t.Fatal("test input produced production qualification or authority")
	}
}
