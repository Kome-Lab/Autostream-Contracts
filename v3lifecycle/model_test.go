package lifecycle

import (
	"math"
	"reflect"
	"testing"
)

func modelLimits() Limits { return Limits{256, 32, 64, 64, 64, 32, 64, 16} }
func modelNamespace() Namespace {
	return Namespace{"owner", ResourceRef{Job, "owner", "job"}, "JOB", "process-1", "sequence-1"}
}
func modelCorrelation() Correlation {
	return Correlation{"request-1", "operation-1", "attempt-1", "step-1", "execution-1"}
}
func modelAdmission() AdmissionModel {
	return AdmissionModel{
		Key:                RecordKey{ContractRef{CC03, 1, SpecRef, "fixture-commit"}, "owner", "admission-1"},
		MembershipSnapshot: "membership-1", AdmissionEpoch: "admission-epoch-1", Lane: Open, Drain: Waiting, Gate: Held, Revision: 1,
		RequiredGuardRefs: []ID{"guard-1"}, AllowedNamespaces: []Namespace{modelNamespace()},
	}
}
func modelObserved[T any](v T) Fact[T] {
	return Fact[T]{State: Observed, Value: &v, Observation: &ObservationContext{"owner", "frame-1", "scope-1", 1}}
}
func modelUnavailable[T any]() Fact[T] {
	return Fact[T]{State: Unavailable, Unavailable: &UnavailableReason{"MISSING", []ID{"missing"}}}
}
func modelUint(v uint64) *uint64 { return &v }
func modelEmpty() Frontier {
	return Frontier{Kind: "EMPTY", Namespace: modelNamespace(), EmptyFactRef: "empty-fact"}
}
func modelThrough(v uint64) Frontier {
	return Frontier{Kind: "THROUGH", Namespace: modelNamespace(), LastSequence: modelUint(v)}
}
func modelClosure() RegistrationClosure {
	return RegistrationClosure{"membership-1", "admission-epoch-1", modelNamespace(), "registration-epoch-1", "owner", "durable-closure-1"}
}
func modelWork(id, parent ID, sequence uint64) WorkRef {
	return WorkRef{id, parent, modelNamespace(), sequence, "admission-1"}
}
func modelApplyAdmission(t *testing.T, m AdmissionModel, e AdmissionEvent) AdmissionModel {
	t.Helper()
	e.ExpectedRevision = m.Revision
	next, r := ReduceAdmission(m, e, modelLimits())
	if r.Kind != StructurallyCoherent {
		t.Fatalf("admission %s: %v", e.Kind, r)
	}
	return next
}
func modelClosedAdmission(t *testing.T) AdmissionModel {
	t.Helper()
	m := modelAdmission()
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: BeginAdmissionCut, Cut: []Frontier{modelEmpty()}})
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: InstallAdmissionGuard, GuardRef: "guard-1", DurableFact: modelObserved(ID("guard-durable"))})
	return modelApplyAdmission(t, m, AdmissionEvent{Kind: FinishAdmissionCut})
}
func modelRequire(t *testing.T, r Result, want ResultKind) {
	t.Helper()
	if r.Kind != want {
		t.Fatalf("result=%s issues=%v want=%s", r.Kind, r.Issues, want)
	}
	if r.ProductionQualified() || r.AuthorizesMutation() {
		t.Fatal("pure model unexpectedly qualified production or authorized mutation")
	}
}
func modelVector() RevisionVector {
	return RevisionVector{{AxisKey{Writer, "owner", ResourceRef{Job, "owner", "job"}}, "process-1", VersionedValue{Kind: PresentUint64, Uint64: modelUint(7)}, "source-frame-1"}}
}
func modelObservation(kind BoundaryKind, sub SubBoundary, resource ResourceRef, outcome Outcome) BoundaryObservation {
	o := BoundaryObservation{Address: BoundaryAddress{kind, resource.OwnerID, resource, sub}, Knowledge: Known, Presence: Present, Outcome: modelObserved(outcome)}
	if sub == RawWrite {
		o.Details = &BoundaryDetails{RawExtent: &WriteExtent{ExtentRef: "extent-1", Offset: modelUint(0), Length: modelUint(4), RequestedLength: modelUint(4)}}
	}
	return o
}
func modelUnknownObservation(kind BoundaryKind, sub SubBoundary, resource ResourceRef) BoundaryObservation {
	return BoundaryObservation{Address: BoundaryAddress{kind, resource.OwnerID, resource, sub}, Knowledge: KnowledgeUnknown, Presence: NotObserved, Outcome: modelUnavailable[Outcome]()}
}
func modelExpectedBoundaries() []BoundaryAddress {
	return []BoundaryAddress{
		{Provider, "owner", ResourceRef{Sink, "owner", "sink-1"}, TransportResponse},
		{Provider, "owner", ResourceRef{Sink, "owner", "sink-1"}, RemoteAccept},
		{Provider, "owner", ResourceRef{Sink, "owner", "sink-1"}, RemoteResult},
		{Provider, "owner", ResourceRef{Sink, "owner", "sink-1"}, RemoteTerminal},
		{LocalPersistence, "owner", ResourceRef{Sink, "owner", "sink-1"}, RawWrite},
		{CPLifecycle, "owner", ResourceRef{Job, "owner", "job"}, CPApply},
	}
}
func modelEffect() EffectModel {
	return EffectModel{OwnerID: "owner", ExpectedBoundaries: modelExpectedBoundaries(), EffectID: "effect-1", Correlation: modelCorrelation(), Target: ResourceRef{Sink, "owner", "sink-1"}, Purpose: PurposeEffect, ContentRef: ContentRef{"NONSECRET_METADATA_REF", "content-ref-1"}, Revision: 1, DispatchState: NoIntent, Observations: ObservationSet{Gate: VerifiedForProfile}}
}
func modelDispatched(t *testing.T) EffectModel {
	t.Helper()
	m := modelEffect()
	next, r := ReduceEffect(m, EffectEvent{Kind: RecordIntentCommit, ExpectedRevision: 1, DurableFact: modelObserved(ID("intent-durable"))}, modelLimits())
	modelRequire(t, r, StructurallyCoherent)
	next, r = ReduceEffect(next, EffectEvent{Kind: RecordDispatchCommit, ExpectedRevision: 2, DurableFact: modelObserved(ID("dispatch-durable"))}, modelLimits())
	modelRequire(t, r, StructurallyCoherent)
	return next
}
func modelCompletion() CompletionRecord {
	closure := modelClosure()
	return CompletionRecord{
		ClosureID: "closure-1", ReservationRefs: ReservationRefs{DomainClosureRef: "domain-closure-1", CPReservation: OptionalRef{State: Absent, AbsenceFactRef: "cp-reservation-not-applicable", Observation: &ObservationContext{"owner", "frame-1", "scope-1", 1}}},
		SourceVector:        modelVector(),
		Partitions:          []PartitionProof{{Namespace: modelNamespace(), CutFrontier: modelThrough(1), FinalAcceptedFrontier: modelThrough(2), TerminalFrontier: modelThrough(2), RegistrationClosed: closure}},
		LineageClosureFacts: []RegistrationClosure{closure}, TerminalPolicyRef: "terminal-policy-1", ProofRefs: []ID{"proof-1"},
	}
}

func TestAdmissionCutAndLineageModel(t *testing.T) {
	l := modelLimits()
	m := modelAdmission()
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: RegisterIndependentWork, Work: modelWork("parent", "", 1), DurableFact: modelObserved(ID("registered-parent"))})
	old := modelCloneAdmission(m)
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: BeginAdmissionCut, Cut: []Frontier{modelThrough(1)}})
	if m.Lane != Closing || old.Lane != Open {
		t.Fatal("cut did not separate new admission from accepted lineage")
	}
	next, r := ReduceAdmission(m, AdmissionEvent{Kind: FinishAdmissionCut, ExpectedRevision: m.Revision}, l)
	modelRequire(t, r, PendingObligation)
	if !reflect.DeepEqual(next, m) {
		t.Fatal("missing guard ACK changed state")
	}
	next, r = ReduceAdmission(m, AdmissionEvent{Kind: RegisterIndependentWork, ExpectedRevision: m.Revision, Work: modelWork("late-independent", "", 2), DurableFact: modelObserved(ID("registered-late"))}, l)
	modelRequire(t, r, RefusedUnchanged)
	if !reflect.DeepEqual(next, m) {
		t.Fatal("refused new admission changed state")
	}
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: InstallAdmissionGuard, GuardRef: "guard-1", DurableFact: modelObserved(ID("durable-guard"))})
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: FinishAdmissionCut})
	m = modelApplyAdmission(t, m, AdmissionEvent{Kind: RegisterChildWork, Work: modelWork("child", "parent", 2), DurableFact: modelObserved(ID("durable-child"))})
	if m.Lane != Closed || len(m.Works) != 2 || *m.Cut[0].LastSequence != 1 {
		t.Fatal("accepted child after cut was dropped or independent admission reopened")
	}
	t.Run("register-before-dispatch", func(t *testing.T) {
		original := modelCloneAdmission(m)
		next, r := ReduceAdmission(m, AdmissionEvent{Kind: RecordWorkDispatch, ExpectedRevision: m.Revision, WorkID: "unregistered"}, l)
		modelRequire(t, r, RefusedUnchanged)
		if !reflect.DeepEqual(next, original) {
			t.Fatal("unregistered dispatch changed state")
		}
		m = modelApplyAdmission(t, m, AdmissionEvent{Kind: RecordWorkDispatch, WorkID: "child"})
		_, r = ReduceAdmission(m, AdmissionEvent{Kind: RecordWorkDispatch, ExpectedRevision: m.Revision, WorkID: "child"}, l)
		modelRequire(t, r, Conflict)
	})
	t.Run("durable-handoff", func(t *testing.T) {
		before := modelCloneAdmission(m)
		next, r := ReduceAdmission(m, AdmissionEvent{Kind: RecordChildHandoff, ExpectedRevision: m.Revision, WorkID: "parent", ChildWorkID: "child", DurableFact: modelUnavailable[ID]()}, l)
		modelRequire(t, r, NeedsEvidence)
		if !reflect.DeepEqual(next, before) || next.Works[0].HandoffReceipt.State == Observed {
			t.Fatal("unknown handoff released parent duty")
		}
		updated := modelApplyAdmission(t, m, AdmissionEvent{Kind: RecordChildHandoff, WorkID: "parent", ChildWorkID: "child", DurableFact: modelObserved(ID("handoff-durable"))})
		_, r = ReduceAdmission(updated, AdmissionEvent{Kind: RegisterChildWork, ExpectedRevision: updated.Revision, Work: modelWork("second-child", "parent", 3), DurableFact: modelObserved(ID("registered-second"))}, l)
		modelRequire(t, r, Conflict)
	})
	t.Run("membership-owner-resource-incarnation", func(t *testing.T) {
		for _, mutate := range []func(*WorkRef){
			func(w *WorkRef) { w.Namespace.OwnerID = "other" },
			func(w *WorkRef) { w.Namespace.Resource.ResourceID = "other-job" },
			func(w *WorkRef) { w.Namespace.ProducerIncarnation = "other-process" },
			func(w *WorkRef) { w.AdmissionRecordRef = "other-admission" },
		} {
			w := modelWork("mismatched-child", "parent", 3)
			mutate(&w)
			_, r := ReduceAdmission(m, AdmissionEvent{Kind: RegisterChildWork, ExpectedRevision: m.Revision, Work: w, DurableFact: modelObserved(ID("durable"))}, l)
			modelRequire(t, r, Conflict)
		}
	})
	t.Run("registration-closure", func(t *testing.T) {
		closed := modelApplyAdmission(t, m, AdmissionEvent{Kind: CloseLineageRegistration, RegistrationClosure: modelClosure()})
		_, r := ReduceAdmission(closed, AdmissionEvent{Kind: RegisterChildWork, ExpectedRevision: closed.Revision, Work: modelWork("late-child", "parent", 3), DurableFact: modelObserved(ID("durable-late-child"))}, l)
		modelRequire(t, r, RefusedUnchanged)
	})
	t.Run("preseeded-invalid-guard-state", func(t *testing.T) {
		forged := modelAdmission()
		forged.Lane = Closed
		forged.Cut = []Frontier{modelEmpty()}
		_, r := ReduceAdmission(forged, AdmissionEvent{Kind: AdmissionTimeout, ExpectedRevision: 1}, l)
		modelRequire(t, r, Conflict)
		forged = modelAdmission()
		forged.Works = []RegisteredWork{{Work: modelWork("one", "", 1), DurableRegistration: modelObserved(ID("one"))}, {Work: modelWork("two", "", 1), DurableRegistration: modelObserved(ID("two"))}}
		_, r = ReduceAdmission(forged, AdmissionEvent{Kind: AdmissionTimeout, ExpectedRevision: 1}, l)
		modelRequire(t, r, Conflict)
	})
	t.Run("CAS-and-overflow", func(t *testing.T) {
		_, r := ReduceAdmission(m, AdmissionEvent{Kind: AdmissionTimeout, ExpectedRevision: m.Revision - 1}, l)
		modelRequire(t, r, Stale)
		exhausted := modelAdmission()
		exhausted.Revision = math.MaxUint64
		_, r = ReduceAdmission(exhausted, AdmissionEvent{Kind: BeginAdmissionCut, ExpectedRevision: math.MaxUint64, Cut: []Frontier{modelEmpty()}}, l)
		modelRequire(t, r, Conflict)
	})
}

func TestCompletionFrontierAndPhysicalObligations(t *testing.T) {
	l := modelLimits()
	modelRequire(t, AssessCompletion(modelCompletion(), l), StructurallyCoherent)
	t.Run("accepted-child-after-cut", func(t *testing.T) {
		c := modelCompletion()
		if *c.Partitions[0].CutFrontier.LastSequence != 1 || *c.Partitions[0].FinalAcceptedFrontier.LastSequence != 2 {
			t.Fatal("fixture lost child after cut")
		}
		modelRequire(t, AssessCompletion(c, l), StructurallyCoherent)
		c.Partitions[0].TerminalFrontier = modelThrough(1)
		modelRequire(t, AssessCompletion(c, l), PendingObligation)
	})
	t.Run("holes-and-frontier-order", func(t *testing.T) {
		c := modelCompletion()
		c.Partitions[0].TerminalFrontier = modelThrough(1)
		c.Partitions[0].Holes = []SequenceRange{{2, 2}}
		modelRequire(t, AssessCompletion(c, l), PendingObligation)
		c.Partitions[0].Holes = append(c.Partitions[0].Holes, SequenceRange{2, 2})
		modelRequire(t, AssessCompletion(c, l), Conflict)
		c = modelCompletion()
		c.Partitions[0].TerminalFrontier = modelThrough(3)
		modelRequire(t, AssessCompletion(c, l), Conflict)
		c = modelCompletion()
		c.Partitions[0].Holes = []SequenceRange{{0, 1}}
		modelRequire(t, AssessCompletion(c, l), Conflict)
	})
	t.Run("empty-needs-durable-fact", func(t *testing.T) {
		c := modelCompletion()
		c.Partitions[0].TerminalFrontier = Frontier{Kind: "EMPTY", Namespace: modelNamespace()}
		modelRequire(t, AssessCompletion(c, l), Conflict)
	})
	t.Run("registration-closure-scope", func(t *testing.T) {
		c := modelCompletion()
		c.Partitions[0].RegistrationClosed.Namespace.ProducerIncarnation = "old-process"
		modelRequire(t, AssessCompletion(c, l), Conflict)
		c = modelCompletion()
		c.LineageClosureFacts = nil
		modelRequire(t, AssessCompletion(c, l), NeedsEvidence)
	})
	t.Run("known-commit-unknown-connection-release", func(t *testing.T) {
		c := modelCompletion()
		c.PhysicalObligations = []PhysicalObligation{{"connection-1", DBConnection, "owner", ResourceRef{Job, "owner", "job"}, "work-1", KnowledgeUnknown, modelUnavailable[ID]()}}
		modelRequire(t, AssessCompletion(c, l), Unknown)
		if c.PhysicalObligations[0].Knowledge != KnowledgeUnknown {
			t.Fatal("release unknown erased")
		}
	})
	t.Run("effect-reference-is-not-terminal-proof", func(t *testing.T) {
		c := modelCompletion()
		c.Effects = []ID{"effect-1"}
		modelRequire(t, AssessCompletion(c, l), NeedsEvidence)
		sink := ResourceRef{Sink, "owner", "sink-1"}
		c.RequiredEffects = []EffectRequirement{{EffectID: "effect-1", Target: sink, Correlation: modelCorrelation()}}
		modelRequire(t, AssessCompletion(c, l), PendingObligation)
		modelRequire(t, AssessCompletionWithEffects(c, []EffectTerminal{{EffectID: "effect-1", Correlation: modelCorrelation(), Observation: modelObservation(Provider, RemoteTerminal, sink, SuccessAtBoundary)}}, l), StructurallyCoherent)
		modelRequire(t, AssessCompletionWithEffects(c, []EffectTerminal{{EffectID: "effect-1", Correlation: modelCorrelation(), Observation: modelUnknownObservation(Provider, RemoteTerminal, sink)}}, l), Unknown)
		terminal := EffectTerminal{EffectID: "effect-1", Correlation: modelCorrelation(), Observation: modelObservation(Provider, RemoteTerminal, sink, SuccessAtBoundary)}
		wrongTarget := terminal
		wrongTarget.Observation.Address.Resource.ResourceID = "wrong-sink"
		modelRequire(t, AssessCompletionWithEffects(c, []EffectTerminal{wrongTarget}, l), Conflict)
		wrongCorrelation := terminal
		wrongCorrelation.Correlation.AttemptID = "different-attempt"
		modelRequire(t, AssessCompletionWithEffects(c, []EffectTerminal{wrongCorrelation}, l), Conflict)
		wrongBoundary := terminal
		wrongBoundary.Observation.Address.SubBoundary = TransportResponse
		modelRequire(t, AssessCompletionWithEffects(c, []EffectTerminal{wrongBoundary}, l), Conflict)
	})
}

func TestEffectBoundaryOutcomeModel(t *testing.T) {
	l := modelLimits()
	m := modelEffect()
	next, r := ReduceEffect(m, EffectEvent{Kind: RecordDispatchCommit, ExpectedRevision: 1, DurableFact: modelObserved(ID("durable"))}, l)
	modelRequire(t, r, Conflict)
	if !reflect.DeepEqual(next, m) {
		t.Fatal("dispatch occurred before durable intent")
	}
	next, r = ReduceEffect(m, EffectEvent{Kind: RecordIntentCommit, ExpectedRevision: 1, DurableFact: modelUnavailable[ID]()}, l)
	modelRequire(t, r, Unknown)
	if next.DispatchState != NoIntent {
		t.Fatal("unknown intent commit enabled dispatch")
	}
	m, r = ReduceEffect(m, EffectEvent{Kind: RecordIntentCommit, ExpectedRevision: 1, DurableFact: modelObserved(ID("intent-durable"))}, l)
	modelRequire(t, r, StructurallyCoherent)
	next, r = ReduceEffect(m, EffectEvent{Kind: RecordDispatchCommit, ExpectedRevision: 2, DurableFact: modelUnavailable[ID]()}, l)
	modelRequire(t, r, Unknown)
	if next.DispatchState != IntentRecorded {
		t.Fatal("unknown dispatch persistence became permission to send")
	}
	m = modelDispatched(t)
	_, r = ReduceEffect(m, EffectEvent{Kind: RecordDispatchCommit, ExpectedRevision: m.Revision, DurableFact: modelObserved(ID("second-dispatch"))}, l)
	modelRequire(t, r, Conflict)
	t.Run("pending-to-fresh-terminal", func(t *testing.T) {
		base := modelDispatched(t)
		pending := modelObservation(Provider, RemoteResult, base.Target, Pending)
		base, r := ReduceEffect(base, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: base.Revision, Observation: pending}, l)
		modelRequire(t, r, StructurallyCoherent)
		pendingUnknown, r := ReduceEffect(base, EffectEvent{Kind: EffectACKLost, ExpectedRevision: base.Revision, Observation: modelUnknownObservation(Provider, RemoteResult, base.Target)}, l)
		modelRequire(t, r, StructurallyCoherent)
		modelRequire(t, AssessObservations(pendingUnknown.Observations, l), Unknown)
		modelRequire(t, AssessObservationHistory(ObservationSet{Gate: VerifiedForProfile, Observations: pendingUnknown.ObservationHistory}, l), Unknown)
		terminal := modelObservation(Provider, RemoteResult, base.Target, SuccessAtBoundary)
		terminal.Outcome.Observation.FrameRef = "frame-2"
		terminal.Outcome.Observation.SourceRevision = 2
		final, r := ReduceEffect(base, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: base.Revision, Observation: terminal}, l)
		modelRequire(t, r, StructurallyCoherent)
		modelRequire(t, AssessObservations(final.Observations, l), StructurallyCoherent)
		modelRequire(t, AssessObservationHistory(ObservationSet{Gate: VerifiedForProfile, Observations: final.ObservationHistory}, l), StructurallyCoherent)
		stale := terminal
		context := *terminal.Outcome.Observation
		context.SourceRevision = 1
		stale.Outcome.Observation = &context
		unchanged, r := ReduceEffect(base, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: base.Revision, Observation: stale}, l)
		modelRequire(t, r, Stale)
		if !reflect.DeepEqual(unchanged, base) {
			t.Fatal("stale terminal changed pending state")
		}
		wrongScope := terminal
		context = *terminal.Outcome.Observation
		context.Scope = "other-scope"
		wrongScope.Outcome.Observation = &context
		_, r = ReduceEffect(base, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: base.Revision, Observation: wrongScope}, l)
		modelRequire(t, r, Conflict)
		latePending := pending
		context = *pending.Outcome.Observation
		context.FrameRef = "frame-3"
		context.SourceRevision = 3
		latePending.Outcome.Observation = &context
		preserved, r := ReduceEffect(final, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: final.Revision, Observation: latePending}, l)
		modelRequire(t, r, StructurallyCoherent)
		if *preserved.Observations.Observations[0].Outcome.Value != SuccessAtBoundary || len(preserved.ObservationHistory) != 3 {
			t.Fatal("late pending downgraded known terminal or erased history")
		}
		modelRequire(t, AssessObservationHistory(ObservationSet{Gate: VerifiedForProfile, Observations: preserved.ObservationHistory}, l), StructurallyCoherent)
	})
	t.Run("exact-provider-and-local-owner-binding", func(t *testing.T) {
		base := modelDispatched(t)
		for _, resource := range []ResourceRef{{Sink, "owner", "wrong-sink"}, {Sink, "foreign-owner", "sink-1"}} {
			foreign := modelUnknownObservation(Provider, RemoteResult, resource)
			_, r := ReduceEffect(base, EffectEvent{Kind: EffectACKLost, ExpectedRevision: base.Revision, Observation: foreign}, l)
			modelRequire(t, r, Conflict)
		}
		foreignSet := base
		foreignSet.ExpectedBoundaries = append(modelExpectedBoundaries(), BoundaryAddress{Provider, "owner", ResourceRef{Sink, "owner", "wrong-sink"}, RemoteResult})
		_, r := ReduceEffect(foreignSet, EffectEvent{Kind: EffectTimeout, ExpectedRevision: base.Revision}, l)
		modelRequire(t, r, Conflict)
		local := modelEffect()
		local.OwnerID = "local-intent-owner"
		localFact := modelObserved(ID("intent-durable"))
		localFact.Observation.OwnerID = "local-intent-owner"
		_, r = ReduceEffect(local, EffectEvent{Kind: RecordIntentCommit, ExpectedRevision: local.Revision, DurableFact: localFact}, l)
		modelRequire(t, r, StructurallyCoherent)
	})
	t.Run("preseeded-history", func(t *testing.T) {
		base := modelDispatched(t)
		known := modelObservation(CPLifecycle, CPApply, ResourceRef{Job, "owner", "job"}, SuccessAtBoundary)
		base.ObservationHistory = []BoundaryObservation{known}
		failed := known
		failed.Outcome = modelObserved(FailureAtBoundary)
		base.Observations.Observations = []BoundaryObservation{failed}
		_, r := ReduceEffect(base, EffectEvent{Kind: EffectTimeout, ExpectedRevision: base.Revision}, l)
		modelRequire(t, r, Conflict)
		base.Observations.Observations = nil
		_, r = ReduceEffect(base, EffectEvent{Kind: EffectTimeout, ExpectedRevision: base.Revision}, l)
		modelRequire(t, r, Conflict)
		base.Observations.Observations = []BoundaryObservation{known}
		base.ObservationHistory[0].Knowledge = Knowledge("UNREGISTERED")
		_, r = ReduceEffect(base, EffectEvent{Kind: EffectTimeout, ExpectedRevision: base.Revision}, l)
		modelRequire(t, r, RefusedUnchanged)
		base.ObservationHistory[0] = known
		base.ObservationHistory[0].Outcome.Value = nil
		_, r = ReduceEffect(base, EffectEvent{Kind: EffectTimeout, ExpectedRevision: base.Revision}, l)
		modelRequire(t, r, RefusedUnchanged)
	})
	t.Run("append-only-history-current-view-and-aliasing", func(t *testing.T) {
		base := modelDispatched(t)
		raw := modelObservation(LocalPersistence, RawWrite, base.Target, SuccessAtBoundary)
		withRaw, r := ReduceEffect(base, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: base.Revision, Observation: raw}, l)
		modelRequire(t, r, StructurallyCoherent)
		*withRaw.ObservationHistory[0].Details.RawExtent.Length = 3
		if *raw.Details.RawExtent.Length != 4 || *withRaw.Observations.Observations[0].Details.RawExtent.Length != 4 {
			t.Fatal("output history aliases event or current view typed details")
		}
		known := modelObservation(Provider, RemoteResult, base.Target, SuccessAtBoundary)
		withKnown, r := ReduceEffect(withRaw, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: withRaw.Revision, Observation: known}, l)
		modelRequire(t, r, StructurallyCoherent)
		laterUnknown, r := ReduceEffect(withKnown, EffectEvent{Kind: EffectACKLost, ExpectedRevision: withKnown.Revision, Observation: modelUnknownObservation(Provider, RemoteResult, base.Target)}, l)
		modelRequire(t, r, StructurallyCoherent)
		if len(laterUnknown.Observations.Observations) != 2 || len(laterUnknown.ObservationHistory) != 3 || laterUnknown.Observations.Observations[1].Knowledge != Known || laterUnknown.ObservationHistory[2].Knowledge != KnowledgeUnknown {
			t.Fatal("later UNKNOWN erased known result or append history")
		}
		modelRequire(t, AssessObservations(laterUnknown.Observations, l), StructurallyCoherent)
		conflict := modelObservation(Provider, RemoteResult, base.Target, FailureAtBoundary)
		_, r = ReduceEffect(laterUnknown, EffectEvent{Kind: AppendEffectObservation, ExpectedRevision: laterUnknown.Revision, Observation: conflict}, l)
		modelRequire(t, r, Conflict)
	})
	providerUnknown := modelUnknownObservation(Provider, RemoteResult, m.Target)
	m, r = ReduceEffect(m, EffectEvent{Kind: EffectACKLost, ExpectedRevision: m.Revision, Observation: providerUnknown}, l)
	modelRequire(t, r, StructurallyCoherent)
	modelRequire(t, AssessObservations(m.Observations, l), Unknown)
	if m.DispatchState != MayDispatch {
		t.Fatal("ACK loss reverted dispatch possibility")
	}
	// A transport failure after a local write is still a fact at that boundary.
	write := modelObservation(LocalPersistence, RawWrite, m.Target, SuccessAtBoundary)
	failedResponse := modelObservation(Provider, TransportResponse, m.Target, FailureAtBoundary)
	set := ObservationSet{Gate: VerifiedForProfile, Observations: []BoundaryObservation{write, failedResponse}}
	modelRequire(t, AssessObservations(set, l), StructurallyCoherent)
	for _, o := range set.Observations {
		if *o.Outcome.Value == NoEffectProven {
			t.Fatal("failure or accepted-count zero inferred no effect")
		}
	}
}

func TestUnknownPreservesKnownBoundaries(t *testing.T) {
	resource := ResourceRef{Job, "owner", "job"}
	sink := ResourceRef{Sink, "owner", "sink"}
	knownCommit := modelObservation(LocalPersistence, LocalDBCommit, resource, SuccessAtBoundary)
	providerUnknown := modelUnknownObservation(Provider, RemoteResult, sink)
	projectionUnknown := modelUnknownObservation(ProjectionBoundary, CPOutcome, resource)
	release := PhysicalObligation{"connection", DBConnection, "owner", resource, "work", KnowledgeUnknown, modelUnavailable[ID]()}
	set := ObservationSet{Gate: Held, Observations: []BoundaryObservation{knownCommit, providerUnknown, projectionUnknown}, PhysicalObligations: []PhysicalObligation{release}}
	modelRequire(t, AssessObservations(set, modelLimits()), Unknown)
	if set.Observations[0].Knowledge != Known || *set.Observations[0].Outcome.Value != SuccessAtBoundary {
		t.Fatal("unknown elsewhere erased known commit")
	}
	if set.Observations[1].Address.Kind == set.Observations[2].Address.Kind {
		t.Fatal("provider and CP projection unknown collapsed")
	}
	// Historical unknown may be narrowed by a same-boundary owner observation.
	set = ObservationSet{Gate: VerifiedForProfile, Observations: []BoundaryObservation{modelUnknownObservation(LocalPersistence, LocalDBCommit, resource), knownCommit}}
	modelRequire(t, AssessObservationHistory(set, modelLimits()), StructurallyCoherent)
	bad := knownCommit
	bad.Outcome = modelObserved(FailureAtBoundary)
	set.Observations = append(set.Observations, bad)
	modelRequire(t, AssessObservations(set, modelLimits()), Conflict)
}

func TestPhaseHintDoesNotProveCommit(t *testing.T) {
	resource := ResourceRef{Job, "owner", "job"}
	set := ObservationSet{Gate: Held, PhaseHints: []BoundaryAddress{{CPLifecycle, "owner", resource, CPApply}}}
	r := AssessObservations(set, modelLimits())
	if r.Kind == StructurallyCoherent || len(set.Observations) != 0 {
		t.Fatal("CP_COMMITTED phase hint supplied missing SQL commit fact")
	}
	if r.AuthorizesMutation() || r.ProductionQualified() {
		t.Fatal("phase hint created authority")
	}
}

func TestInspectionIsObservationOnly(t *testing.T) {
	q := QueryTarget{modelCorrelation(), "binding-1", "execution-1", "effect-1", "scope-1", PurposeLookup}
	resource := ResourceRef{Job, "owner", "job"}
	set := ObservationSet{Gate: Held, Observations: []BoundaryObservation{modelUnknownObservation(LocalPersistence, LocalDBCommit, resource)}}
	observed, r := InspectSameAttempt(q, q, set, modelLimits())
	modelRequire(t, r, Unknown)
	if observed.Target != q || !reflect.DeepEqual(set, observed.Observations) {
		t.Fatal("inspection changed immutable tuple or outcomes")
	}
	observed.Observations.Observations[0].Outcome.Unavailable.MissingRefs[0] = "changed"
	if set.Observations[0].Outcome.Unavailable.MissingRefs[0] == "changed" {
		t.Fatal("inspection output aliases caller-owned observations")
	}
	for _, change := range []func(*QueryTarget){
		func(v *QueryTarget) { v.Correlation.AttemptID = "new-attempt" },
		func(v *QueryTarget) { v.BindingRef = "other-binding" },
		func(v *QueryTarget) { v.Scope = "other-scope" },
		func(v *QueryTarget) { v.ExecutionKey = "other-execution" },
		func(v *QueryTarget) { v.TargetRecordRef = "other-record" },
		func(v *QueryTarget) { v.Purpose = PurposeRepair },
	} {
		changed := q
		change(&changed)
		_, r := InspectSameAttempt(q, changed, set, modelLimits())
		modelRequire(t, r, Conflict)
	}
	absent := modelObservation(LocalPersistence, LocalDBCommit, resource, Pending)
	absent.Presence = Absent
	_, r = InspectSameAttempt(q, q, ObservationSet{Gate: Held, Observations: []BoundaryObservation{absent}}, modelLimits())
	if r.Kind == StructurallyCoherent || *absent.Outcome.Value == NoEffectProven {
		t.Fatal("ABSENT turned into unused or no-effect proof")
	}
}

func modelReconcile() (ReconcileRecord, ProjectionSnapshot, ReconcileSnapshot) {
	plan := ReconcileRecord{
		PlanID: "plan-1", OriginalCorrelation: modelCorrelation(), Purpose: PurposeRepair,
		SourceReceiptRefs: []ID{"receipt-1", "receipt-2"}, SourceWitnessRefs: []ID{"witness-1"}, CurrentSourceVector: modelVector(),
		ExpectedProjection: ResourceRef{Projection, "owner", "projection-1"}, ExpectedProjectionRevision: 5, IntendedMetadataDelta: MetadataDelta{"FIXED_RECEIPT_REFERENCE", "receipt-1", "metadata-ref-1"},
		RequiredRecordCapability: "fresh-record-capability", PlanRevision: 1, Freshness: modelObserved(ID("independent-freshness")),
	}
	projection := ProjectionSnapshot{"owner", "projection-1", 5, "old-receipt", "old-metadata"}
	current := ReconcileSnapshot{modelCorrelation(), modelVector(), []ID{"receipt-2", "receipt-1"}, []ID{"witness-1"}, "fresh-record-capability", "owner", modelObserved(ID("independent-freshness")), VerifiedForProfile}
	return plan, projection, current
}
func TestRecordOnlyReconciliationCASModel(t *testing.T) {
	l := modelLimits()
	plan, projection, current := modelReconcile()
	modelRequire(t, CompareProjectionCAS(plan, projection, current, l), StructurallyCoherent)
	t.Run("exact-projection-target", func(t *testing.T) {
		p, target, frame := modelReconcile()
		target.ProjectionID = "other-projection"
		modelRequire(t, CompareProjectionCAS(p, target, frame, l), Conflict)
		p, target, frame = modelReconcile()
		target.OwnerID = "other-owner"
		modelRequire(t, CompareProjectionCAS(p, target, frame, l), Conflict)
		p, target, frame = modelReconcile()
		p.ExpectedProjection.Kind = ResourceKind("UNREGISTERED")
		modelRequire(t, CompareProjectionCAS(p, target, frame, l), Conflict)
	})
	t.Run("exact-CAS-no-rebase", func(t *testing.T) {
		projection.Revision = 6
		modelRequire(t, CompareProjectionCAS(plan, projection, current, l), Stale)
		if plan.ExpectedProjectionRevision != 5 || plan.PlanRevision != 1 {
			t.Fatal("CAS conflict rebased the plan")
		}
	})
	t.Run("immutable-source-and-purpose", func(t *testing.T) {
		p, projection, current := modelReconcile()
		p.Purpose = PurposeEffect
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), Conflict)
		p, projection, current = modelReconcile()
		current.OriginalCorrelation.AttemptID = "replacement-attempt"
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), Conflict)
		p, projection, current = modelReconcile()
		current.SourceVector[0].Incarnation = "old-process"
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), Conflict)
		p, projection, current = modelReconcile()
		current.SourceWitnessRefs = []ID{"other-witness"}
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), Conflict)
	})
	t.Run("freshness-capability-hold", func(t *testing.T) {
		p, projection, current := modelReconcile()
		current.Freshness = modelUnavailable[ID]()
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), NeedsEvidence)
		p, projection, current = modelReconcile()
		current.FreshRecordCapability = ""
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), NeedsEvidence)
		p, projection, current = modelReconcile()
		current.Gate = Held
		modelRequire(t, CompareProjectionCAS(p, projection, current, l), PendingObligation)
	})
}

func TestRecordTransferIsNotEffectReplay(t *testing.T) {
	transfer := RecordTransfer{modelCorrelation(), PurposeRecordTransfer, "owner", "owner", "fixed-receipt", "projection-1", 9, "fresh-record-capability", modelObserved(ID("freshness"))}
	modelRequire(t, ValidateRecordTransfer(transfer, modelLimits()), StructurallyCoherent)
	if transfer.OriginalCorrelation != modelCorrelation() || transfer.ReceiptRef != "fixed-receipt" {
		t.Fatal("record transfer replaced original execution")
	}
	t.Run("projection-revision", func(t *testing.T) {
		missing := transfer
		missing.ExpectedProjectionRevision = 0
		modelRequire(t, ValidateRecordTransfer(missing, modelLimits()), NeedsEvidence)
	})
	transfer.Purpose = PurposeEffect
	modelRequire(t, ValidateRecordTransfer(transfer, modelLimits()), Conflict)
	transfer.Purpose = PurposeRecordTransfer
	transfer.RequiredRecordCapability = ""
	modelRequire(t, ValidateRecordTransfer(transfer, modelLimits()), NeedsEvidence)
}

func TestRestoreFreshnessIsNotEquality(t *testing.T) {
	restore := RestoreFreshness{modelCorrelation(), "backup-point-1", "independent-witness", modelUnavailable[ID](), VerifiedForProfile}
	// Matching restored values are not an independent observation.
	modelRequire(t, AssessRestoreFreshness(restore, modelLimits()), NeedsEvidence)
	restore.IndependentFreshness = modelObserved(ID("independent-witness"))
	modelRequire(t, AssessRestoreFreshness(restore, modelLimits()), StructurallyCoherent)
	restore.IndependentFreshness = modelObserved(ID("other-witness"))
	modelRequire(t, AssessRestoreFreshness(restore, modelLimits()), Conflict)
	restore.IndependentFreshness = modelObserved(ID("independent-witness"))
	restore.Gate = Held
	modelRequire(t, AssessRestoreFreshness(restore, modelLimits()), PendingObligation)
	plan, projection, current := modelReconcile()
	current.Freshness = modelUnavailable[ID]()
	modelRequire(t, CompareProjectionCAS(plan, projection, current, modelLimits()), NeedsEvidence)
	if plan.OriginalCorrelation != modelCorrelation() {
		t.Fatal("partial/common restore invented a new attempt")
	}
}

func TestTimeoutCancellationDoNotReopen(t *testing.T) {
	m := modelClosedAdmission(t)
	for _, kind := range []AdmissionEventKind{AdmissionTimeout, AdmissionCancellation, AdmissionRestart} {
		next, r := ReduceAdmission(m, AdmissionEvent{Kind: kind, ExpectedRevision: m.Revision}, modelLimits())
		modelRequire(t, r, PendingObligation)
		if !reflect.DeepEqual(next, m) || next.Lane != Closed || next.Gate != Held || next.Drain != Waiting {
			t.Fatal("timeout/cancel/restart reopened or completed drain")
		}
	}
	effect := modelDispatched(t)
	for _, kind := range []EffectEventKind{EffectTimeout, EffectCancellation, EffectRestart} {
		next, r := ReduceEffect(effect, EffectEvent{Kind: kind, ExpectedRevision: effect.Revision}, modelLimits())
		modelRequire(t, r, Unknown)
		if !reflect.DeepEqual(next, effect) || next.EffectID != effect.EffectID || next.DispatchState != MayDispatch {
			t.Fatal("uncertainty reset dispatch state or changed identity")
		}
	}
}

func TestFanoutOutcomeObligations(t *testing.T) {
	sink1, sink2 := ResourceRef{Sink, "owner", "sink-1"}, ResourceRef{Sink, "owner", "sink-2"}
	fanout := FanoutModel{"parent-effect", modelCorrelation(), []FanoutChild{
		{"child-effect-1", sink1, ObservationSet{Gate: VerifiedForProfile, Observations: []BoundaryObservation{modelObservation(Provider, RemoteTerminal, sink1, SuccessAtBoundary)}}},
		{"child-effect-2", sink2, ObservationSet{Gate: VerifiedForProfile, Observations: []BoundaryObservation{modelObservation(Provider, RemoteTerminal, sink2, SuccessAtBoundary)}}},
	}}
	modelRequire(t, AssessFanout(fanout, modelLimits()), StructurallyCoherent)
	fanout.Children[1].Observations.Observations[0] = modelUnknownObservation(Provider, RemoteTerminal, sink2)
	modelRequire(t, AssessFanout(fanout, modelLimits()), Unknown)
	// Parent cancellation and process-local dedupe cannot terminate this child.
	if fanout.Children[1].EffectID != "child-effect-2" || fanout.Children[1].Observations.Observations[0].Knowledge != KnowledgeUnknown {
		t.Fatal("unknown detached child lost")
	}
	fanout.Children[1].Observations.Observations[0] = modelObservation(Provider, TransportResponse, sink2, SuccessAtBoundary)
	modelRequire(t, AssessFanout(fanout, modelLimits()), PendingObligation)
	fanout.Children[1].Observations.Observations[0] = modelObservation(Provider, RemoteTerminal, sink2, SuccessAtBoundary)
	fanout.Children[1].EffectID = fanout.Children[0].EffectID
	modelRequire(t, AssessFanout(fanout, modelLimits()), Conflict)
}
