package lifecycle

import (
	"math"
	"slices"
)

// AdmissionModel describes one owner's guard and accepted lineage. It is not a
// durable guard and cannot authorize a writer. Every durable fact is still an
// untrusted claim until the product owner verifies it in its local frame.
type AdmissionModel struct {
	Key                                   RecordKey
	MembershipSnapshot, AdmissionEpoch    ID
	Lane                                  AdmissionLane
	Drain                                 DrainState
	Gate                                  Gate
	Revision                              uint64
	RequiredGuardRefs, InstalledGuardRefs []ID
	AllowedNamespaces                     []Namespace
	Cut                                   []Frontier
	Works                                 []RegisteredWork
	ClosedRegistrations                   []RegistrationClosure
}
type RegisteredWork struct {
	Work                WorkRef
	DurableRegistration Fact[ID]
	Dispatched          bool
	HandoffReceipt      Fact[ID]
}
type AdmissionEventKind string

const (
	BeginAdmissionCut        AdmissionEventKind = "BEGIN_CUT"
	InstallAdmissionGuard    AdmissionEventKind = "INSTALL_GUARD"
	FinishAdmissionCut       AdmissionEventKind = "FINISH_CUT"
	RegisterIndependentWork  AdmissionEventKind = "REGISTER_INDEPENDENT"
	RegisterChildWork        AdmissionEventKind = "REGISTER_CHILD"
	RecordWorkDispatch       AdmissionEventKind = "RECORD_DISPATCH"
	RecordChildHandoff       AdmissionEventKind = "RECORD_HANDOFF"
	CloseLineageRegistration AdmissionEventKind = "CLOSE_LINEAGE_REGISTRATION"
	AdmissionTimeout         AdmissionEventKind = "TIMEOUT"
	AdmissionCancellation    AdmissionEventKind = "CANCELLATION"
	AdmissionRestart         AdmissionEventKind = "RESTART"
)

type AdmissionEvent struct {
	Kind                AdmissionEventKind
	ExpectedRevision    uint64
	GuardRef            ID
	Work                WorkRef
	WorkID, ChildWorkID ID
	DurableFact         Fact[ID]
	Cut                 []Frontier
	RegistrationClosure RegistrationClosure
}

// ReduceAdmission computes an owner-local CAS proposal. No CAS, persistence,
// dispatch, hold release, or authority issuance is performed.
func ReduceAdmission(current AdmissionModel, event AdmissionEvent, limits Limits) (AdmissionModel, Result) {
	if r := modelAdmissionValid(current, limits); r.Kind != StructurallyCoherent {
		return current, r
	}
	if event.ExpectedRevision != current.Revision {
		return current, result(Stale, "RECORD_CAS_MISMATCH", "ExpectedRevision")
	}
	if event.Kind == AdmissionTimeout || event.Kind == AdmissionCancellation || event.Kind == AdmissionRestart {
		return current, result(PendingObligation, "ADMISSION_AND_HOLD_PRESERVED", "Admission")
	}
	if current.Revision == math.MaxUint64 {
		return current, result(Conflict, "REVISION_EXHAUSTED", "Revision")
	}
	next := modelCloneAdmission(current)
	switch event.Kind {
	case BeginAdmissionCut:
		if current.Lane != Open {
			return current, result(Conflict, "CUT_ALREADY_STARTED", "Lane")
		}
		if len(event.Cut) != len(current.AllowedNamespaces) {
			return current, result(NeedsEvidence, "CUT_COVERAGE_MISSING", "Cut")
		}
		for i, f := range event.Cut {
			if r := modelFrontierValid(f, limits); r.Kind != StructurallyCoherent {
				return current, r
			}
			if !slices.Contains(current.AllowedNamespaces, f.Namespace) {
				return current, result(Conflict, "CUT_NAMESPACE_MISMATCH", "Cut")
			}
			for _, earlier := range event.Cut[:i] {
				if earlier.Namespace == f.Namespace {
					return current, result(Conflict, "DUPLICATE_NAMESPACE", "Cut")
				}
			}
			last := modelLastAccepted(current.Works, f.Namespace)
			if modelFrontierLast(f) != last {
				return current, result(Conflict, "CUT_FRONTIER_MISMATCH", "Cut")
			}
		}
		next.Lane = Closing
		next.Cut = modelCloneFrontiers(event.Cut)
	case InstallAdmissionGuard:
		if current.Lane != Closing {
			return current, result(Conflict, "GUARD_OUTSIDE_CUT", "Lane")
		}
		if !slices.Contains(current.RequiredGuardRefs, event.GuardRef) {
			return current, result(Conflict, "UNREGISTERED_GUARD", "GuardRef")
		}
		if slices.Contains(current.InstalledGuardRefs, event.GuardRef) {
			return current, result(Conflict, "GUARD_ALREADY_RECORDED", "GuardRef")
		}
		if r := modelOwnerFact(event.DurableFact, current.Key.OwnerID, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		next.InstalledGuardRefs = append(next.InstalledGuardRefs, event.GuardRef)
	case FinishAdmissionCut:
		if current.Lane != Closing {
			return current, result(Conflict, "NOT_CLOSING", "Lane")
		}
		for _, guard := range current.RequiredGuardRefs {
			if !slices.Contains(current.InstalledGuardRefs, guard) {
				return current, result(PendingObligation, "GUARD_ACK_MISSING", "InstalledGuardRefs")
			}
		}
		next.Lane = Closed
	case RegisterIndependentWork, RegisterChildWork:
		if len(current.Works) >= limits.MaxRefs {
			return current, result(Conflict, "LIMIT_EXCEEDED", "Works")
		}
		w := event.Work
		if r := modelWorkValid(w, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		if w.AdmissionRecordRef != current.Key.RecordID || !slices.Contains(current.AllowedNamespaces, w.Namespace) {
			return current, result(Conflict, "ADMISSION_SCOPE_MISMATCH", "Work")
		}
		if modelWorkIndex(current.Works, w.WorkID) >= 0 {
			return current, result(Conflict, "WORK_ID_REUSED", "Work.WorkID")
		}
		if event.Kind == RegisterIndependentWork {
			if current.Lane != Open || w.ParentWorkID != "" {
				return current, result(RefusedUnchanged, "INDEPENDENT_ADMISSION_CLOSED", "Lane")
			}
		} else {
			parent := modelWorkIndex(current.Works, w.ParentWorkID)
			if parent < 0 {
				return current, result(Conflict, "PARENT_NOT_ACCEPTED", "Work.ParentWorkID")
			}
			pw := current.Works[parent]
			if pw.HandoffReceipt.State == Observed {
				return current, result(Conflict, "PARENT_ALREADY_HANDED_OFF", "Work.ParentWorkID")
			}
			for _, closure := range current.ClosedRegistrations {
				if closure.Namespace == pw.Work.Namespace || closure.Namespace == w.Namespace {
					return current, result(RefusedUnchanged, "LINEAGE_REGISTRATION_CLOSED", "RegistrationClosure")
				}
			}
		}
		last := modelLastAccepted(current.Works, w.Namespace)
		if last == math.MaxUint64 || w.Sequence != last+1 {
			return current, result(Conflict, "WORK_SEQUENCE_NOT_NEXT", "Work.Sequence")
		}
		if r := modelOwnerFact(event.DurableFact, w.Namespace.OwnerID, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		next.Works = append(next.Works, RegisteredWork{Work: w, DurableRegistration: modelCloneFact(event.DurableFact)})
	case RecordWorkDispatch:
		i := modelWorkIndex(current.Works, event.WorkID)
		if i < 0 {
			return current, result(RefusedUnchanged, "REGISTER_BEFORE_DISPATCH", "WorkID")
		}
		if current.Works[i].Dispatched {
			return current, result(Conflict, "NO_AUTOMATIC_REDISPATCH", "WorkID")
		}
		if r := modelOwnerFact(current.Works[i].DurableRegistration, current.Works[i].Work.Namespace.OwnerID, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		next.Works[i].Dispatched = true
	case RecordChildHandoff:
		parent, child := modelWorkIndex(current.Works, event.WorkID), modelWorkIndex(current.Works, event.ChildWorkID)
		if parent < 0 || child < 0 || current.Works[child].Work.ParentWorkID != event.WorkID {
			return current, result(Conflict, "HANDOFF_LINEAGE_MISMATCH", "ChildWorkID")
		}
		if current.Works[parent].HandoffReceipt.State == Observed {
			return current, result(Conflict, "HANDOFF_ALREADY_RECORDED", "WorkID")
		}
		if r := modelOwnerFact(event.DurableFact, current.Works[parent].Work.Namespace.OwnerID, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		if r := modelOwnerFact(current.Works[child].DurableRegistration, current.Works[child].Work.Namespace.OwnerID, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		next.Works[parent].HandoffReceipt = modelCloneFact(event.DurableFact)
	case CloseLineageRegistration:
		closure := event.RegistrationClosure
		if current.Lane != Closed {
			return current, result(PendingObligation, "ADMISSION_NOT_CLOSED", "Lane")
		}
		if r := modelClosureValid(closure, limits); r.Kind != StructurallyCoherent {
			return current, r
		}
		if closure.MembershipSnapshot != current.MembershipSnapshot || closure.AdmissionEpoch != current.AdmissionEpoch || !slices.Contains(current.AllowedNamespaces, closure.Namespace) {
			return current, result(Conflict, "CLOSURE_SCOPE_MISMATCH", "RegistrationClosure")
		}
		for _, old := range current.ClosedRegistrations {
			if old.Namespace == closure.Namespace {
				return current, result(Conflict, "REGISTRATION_ALREADY_CLOSED", "RegistrationClosure")
			}
		}
		next.ClosedRegistrations = append(next.ClosedRegistrations, closure)
	default:
		return current, result(Conflict, "UNKNOWN_ADMISSION_EVENT", "Kind")
	}
	next.Revision++
	return next, coherent()
}

// AssessCompletion checks a structural closure claim. Effect references without
// typed terminal observations remain pending; use AssessCompletionWithEffects
// when external effects are present. Even a coherent result is not a READY seal.
func AssessCompletion(record CompletionRecord, limits Limits) Result {
	if r := modelCompletionValid(record, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if len(record.Effects) > 0 {
		return result(PendingObligation, "EFFECT_TERMINAL_OBSERVATIONS_REQUIRED", "Effects")
	}
	return coherent()
}

// AssessCompletionWithEffects joins external terminal observations by immutable
// EffectID. It does not turn safety termination into business success.
func AssessCompletionWithEffects(record CompletionRecord, effects []EffectTerminal, limits Limits) Result {
	if r := modelCompletionValid(record, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if len(effects) != len(record.Effects) || len(effects) > limits.MaxRefs {
		return result(NeedsEvidence, "EFFECT_COVERAGE_MISMATCH", "Effects")
	}
	for i, e := range effects {
		if !slices.Contains(record.Effects, e.EffectID) {
			return result(Conflict, "UNEXPECTED_EFFECT", "Effects")
		}
		for _, old := range effects[:i] {
			if old.EffectID == e.EffectID {
				return result(Conflict, "DUPLICATE_EFFECT", "Effects")
			}
		}
		var required EffectRequirement
		for _, candidate := range record.RequiredEffects {
			if candidate.EffectID == e.EffectID {
				required = candidate
				break
			}
		}
		if e.Correlation != required.Correlation || e.Observation.Address.Resource != required.Target || e.Observation.Address.OwnerID != required.Target.OwnerID || e.Observation.Address.Kind != Provider || e.Observation.Address.SubBoundary != RemoteTerminal {
			return result(Conflict, "EFFECT_TERMINAL_BINDING_MISMATCH", "Effects")
		}
		if r := modelTerminalObservation(e.Observation, limits); r.Kind != StructurallyCoherent {
			return r
		}
	}
	return coherent()
}

type EffectTerminal struct {
	EffectID    ID
	Correlation Correlation
	Observation BoundaryObservation
}

type EffectModel struct {
	OwnerID            ID
	ExpectedBoundaries []BoundaryAddress
	EffectID           ID
	Correlation        Correlation
	Target             ResourceRef
	Purpose            Purpose
	ContentRef         ContentRef
	Revision           uint64
	DispatchState      DispatchState
	Observations       ObservationSet
	ObservationHistory []BoundaryObservation
}
type EffectEventKind string

const (
	RecordIntentCommit      EffectEventKind = "RECORD_INTENT_COMMIT"
	RecordDispatchCommit    EffectEventKind = "RECORD_DISPATCH_COMMIT"
	AppendEffectObservation EffectEventKind = "APPEND_OBSERVATION"
	EffectACKLost           EffectEventKind = "ACK_LOST"
	EffectTimeout           EffectEventKind = "TIMEOUT"
	EffectCancellation      EffectEventKind = "CANCELLATION"
	EffectRestart           EffectEventKind = "RESTART"
)

type EffectEvent struct {
	Kind             EffectEventKind
	ExpectedRevision uint64
	DurableFact      Fact[ID]
	Observation      BoundaryObservation
}

// ReduceEffect models durable-before-dispatch ordering. DISPATCH_POSSIBLE records
// uncertainty; it is never returned as a token permitting a send or a retry.
func ReduceEffect(current EffectModel, event EffectEvent, limits Limits) (EffectModel, Result) {
	if r := modelEffectValid(current, limits); r.Kind != StructurallyCoherent {
		return current, r
	}
	if event.ExpectedRevision != current.Revision {
		return current, result(Stale, "RECORD_CAS_MISMATCH", "ExpectedRevision")
	}
	if event.Kind == EffectTimeout || event.Kind == EffectCancellation || event.Kind == EffectRestart {
		if current.DispatchState == MayDispatch {
			r := AssessObservations(current.Observations, limits)
			if r.Kind == StructurallyCoherent {
				r = result(PendingObligation, "DISPATCH_HISTORY_PRESERVED", "DispatchState")
			}
			if len(current.Observations.Observations) == 0 {
				r = result(Unknown, "DISPATCH_RESULT_UNKNOWN", "DispatchState")
			}
			return current, r
		}
		return current, result(PendingObligation, "NO_IMPLICIT_DISPATCH", "DispatchState")
	}
	if current.Revision == math.MaxUint64 {
		return current, result(Conflict, "REVISION_EXHAUSTED", "Revision")
	}
	next := current
	next.ExpectedBoundaries = slices.Clone(current.ExpectedBoundaries)
	next.Observations = modelCloneObservations(current.Observations)
	next.ObservationHistory = modelCloneObservationList(current.ObservationHistory)
	switch event.Kind {
	case RecordIntentCommit:
		if current.DispatchState != NoIntent {
			return current, result(Conflict, "INTENT_ALREADY_RECORDED", "DispatchState")
		}
		if r := modelOwnerFact(event.DurableFact, current.OwnerID, limits); r.Kind != StructurallyCoherent {
			if r.Kind == NeedsEvidence {
				r = result(Unknown, "INTENT_COMMIT_UNKNOWN_NO_DISPATCH", "DurableFact")
			}
			return current, r
		}
		next.DispatchState = IntentRecorded
	case RecordDispatchCommit:
		if current.DispatchState != IntentRecorded {
			return current, result(Conflict, "DISPATCH_NOT_NEXT_OR_ALREADY_POSSIBLE", "DispatchState")
		}
		if r := modelOwnerFact(event.DurableFact, current.OwnerID, limits); r.Kind != StructurallyCoherent {
			if r.Kind == NeedsEvidence {
				r = result(Unknown, "DISPATCH_COMMIT_UNKNOWN_NO_SEND", "DurableFact")
			}
			return current, r
		}
		next.DispatchState = MayDispatch
	case AppendEffectObservation, EffectACKLost:
		if current.DispatchState != MayDispatch {
			return current, result(RefusedUnchanged, "NO_DISPATCH_RECORDED", "DispatchState")
		}
		if len(current.Observations.Observations) >= limits.MaxRefs || len(current.ObservationHistory) >= limits.MaxRefs {
			return current, result(Conflict, "LIMIT_EXCEEDED", "Observations")
		}
		if !slices.Contains(current.ExpectedBoundaries, event.Observation.Address) {
			return current, result(Conflict, "UNEXPECTED_EFFECT_BOUNDARY", "Observation.Address")
		}
		if event.Kind == EffectACKLost && event.Observation.Knowledge != KnowledgeUnknown {
			return current, result(Conflict, "ACK_LOSS_REQUIRES_UNKNOWN_BOUNDARY", "Observation")
		}
		if r := modelObservationValid(event.Observation, limits); r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation {
			return current, r
		}
		for _, old := range current.Observations.Observations {
			if old.Address == event.Observation.Address && modelKnownObservationConflict(old, event.Observation) {
				return current, result(Conflict, "CONTRADICTORY_BOUNDARY_FACT", "Observation")
			}
		}
		if modelKnownTerminal(event.Observation) {
			for _, old := range append(modelCloneObservationList(current.ObservationHistory), current.Observations.Observations...) {
				if old.Address == event.Observation.Address && modelKnownPending(old) {
					if r := modelPendingAdvance(old, event.Observation); r.Kind != StructurallyCoherent {
						return current, r
					}
				}
			}
		}
		next.ObservationHistory = append(next.ObservationHistory, modelCloneObservation(event.Observation))
		existing := -1
		for i, old := range next.Observations.Observations {
			if old.Address == event.Observation.Address {
				existing = i
				break
			}
		}
		if existing < 0 {
			next.Observations.Observations = append(next.Observations.Observations, modelCloneObservation(event.Observation))
		} else if !(modelKnownTerminal(next.Observations.Observations[existing]) && (event.Observation.Knowledge == KnowledgeUnknown || modelKnownPending(event.Observation))) {
			next.Observations.Observations[existing] = modelCloneObservation(event.Observation)
		}
	default:
		return current, result(Conflict, "UNKNOWN_EFFECT_EVENT", "Kind")
	}
	next.Revision++
	return next, coherent()
}

// AssessObservations preserves known facts at each boundary while separately
// reporting unknown releases, projections, and provider outcomes. PhaseHints are
// advisory and never supply missing observations.
func AssessObservations(set ObservationSet, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if len(set.Observations) > limits.MaxRefs {
		return result(Conflict, "LIMIT_EXCEEDED", "Observations")
	}
	for i, o := range set.Observations {
		for _, old := range set.Observations[:i] {
			if old.Address == o.Address {
				return result(Conflict, "DUPLICATE_CURRENT_BOUNDARY", "Observations")
			}
		}
	}
	return AssessObservationHistory(set, limits)
}

// AssessObservationHistory reduces append-only facts without discarding history.
// Use AssessObservations for a unique current-view ObservationSet.
func AssessObservationHistory(set ObservationSet, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelGateValid(set.Gate) {
		return result(Conflict, "UNKNOWN_GATE", "Gate")
	}
	if len(set.Observations) > limits.MaxRefs || len(set.PhysicalObligations) > limits.MaxObligations || len(set.RemainingObligations) > limits.MaxObligations || len(set.Contradictions) > limits.MaxRefs || len(set.PhaseHints) > limits.MaxRefs {
		return result(Conflict, "LIMIT_EXCEEDED", "ObservationSet")
	}
	if !modelIDSet(set.RemainingObligations, limits, false) || !modelIDSet(set.Contradictions, limits, false) {
		return result(Conflict, "INVALID_OBLIGATION_OR_CONTRADICTION_REFERENCES", "ObservationSet")
	}
	for _, hint := range set.PhaseHints {
		if r := ValidateBoundary(hint, limits); r.Kind != StructurallyCoherent {
			return r
		}
	}
	if len(set.Contradictions) > 0 {
		return result(Conflict, "CONTRADICTION_RECORDED", "Contradictions")
	}
	unknown, pending := false, false
	for i, o := range set.Observations {
		r := modelObservationValid(o, limits)
		if r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation {
			return r
		}
		resolvedUnknown, resolvedPending := false, false
		for j, other := range set.Observations {
			if i == j || o.Address != other.Address {
				continue
			}
			if modelKnownObservationConflict(o, other) {
				return result(Conflict, "CONTRADICTORY_BOUNDARY_FACT", "Observations")
			}
			if modelKnownTerminal(other) {
				resolvedUnknown = true
				if modelKnownPending(o) {
					if i < j {
						if advance := modelPendingAdvance(o, other); advance.Kind != StructurallyCoherent {
							return advance
						}
					}
					resolvedPending = true
				}
			} else if other.Knowledge == Known && j > i {
				resolvedUnknown = true
			}
		}
		if r.Kind == Unknown && !resolvedUnknown {
			unknown = true
		}
		if r.Kind == PendingObligation && !resolvedPending {
			pending = true
		}
	}
	for _, obligation := range set.PhysicalObligations {
		r := modelObligationValid(obligation, limits)
		if r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation && r.Kind != NeedsEvidence {
			return r
		}
		if r.Kind == Unknown {
			unknown = true
		}
		if r.Kind == PendingObligation || r.Kind == NeedsEvidence {
			pending = true
		}
	}
	if unknown {
		return result(Unknown, "BOUNDARY_OR_RELEASE_UNKNOWN", "ObservationSet")
	}
	if pending || len(set.RemainingObligations) > 0 || set.Gate == Held {
		return result(PendingObligation, "UNRESOLVED_OBLIGATION_OR_HOLD", "ObservationSet")
	}
	if len(set.Observations) == 0 || set.Gate == NotQualified {
		return result(NeedsEvidence, "NO_QUALIFIED_OBSERVATION_COVERAGE", "ObservationSet")
	}
	return coherent()
}

type InspectionModel struct {
	Target       QueryTarget
	Observations ObservationSet
}

// InspectSameAttempt compares every immutable query field and returns a copied
// observation model. It has no mutation, reserve, consume, release, or resend step.
func InspectSameAttempt(original, query QueryTarget, set ObservationSet, limits Limits) (InspectionModel, Result) {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return InspectionModel{}, r
	}
	if r := modelQueryValid(original, limits); r.Kind != StructurallyCoherent {
		return InspectionModel{}, r
	}
	if r := modelQueryValid(query, limits); r.Kind != StructurallyCoherent {
		return InspectionModel{}, r
	}
	if original != query {
		return InspectionModel{}, result(Conflict, "IMMUTABLE_QUERY_MISMATCH", "QueryTarget")
	}
	r := AssessObservations(set, limits)
	if r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation && r.Kind != NeedsEvidence {
		return InspectionModel{}, r
	}
	return InspectionModel{original, modelCloneObservations(set)}, r
}

type ProjectionSnapshot struct {
	OwnerID, ProjectionID   ID
	Revision                uint64
	ReceiptRef, MetadataRef ID
}
type ReconcileSnapshot struct {
	OriginalCorrelation                  Correlation
	SourceVector                         RevisionVector
	SourceReceiptRefs, SourceWitnessRefs []ID
	FreshRecordCapability                ID
	FreshnessOwnerID                     ID
	Freshness                            Fact[ID]
	Gate                                 Gate
}

// CompareProjectionCAS compares a record-only proposal with an explicitly supplied
// current owner frame. It performs no DB write and does not authenticate a supplied
// capability or witness reference.
func CompareProjectionCAS(plan ReconcileRecord, projection ProjectionSnapshot, current ReconcileSnapshot, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelID(plan.PlanID, limits) || !modelID(projection.OwnerID, limits) || !modelID(projection.ProjectionID, limits) || plan.PlanRevision == 0 {
		return result(NeedsEvidence, "REQUIRED_RECORD_FIELD", "Reconcile")
	}
	if plan.Purpose != PurposeRepair || plan.IntendedMetadataDelta.Kind != "FIXED_RECEIPT_REFERENCE" {
		return result(Conflict, "NOT_RECORD_ONLY", "Purpose")
	}
	if !modelResourceValid(plan.ExpectedProjection, limits) || plan.ExpectedProjection.Kind != Projection || plan.ExpectedProjection.OwnerID != projection.OwnerID || plan.ExpectedProjection.ResourceID != projection.ProjectionID {
		return result(Conflict, "PROJECTION_TARGET_MISMATCH", "ExpectedProjection")
	}
	if r := modelCorrelationValid(plan.OriginalCorrelation, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if plan.OriginalCorrelation != current.OriginalCorrelation {
		return result(Conflict, "IMMUTABLE_CORRELATION_MISMATCH", "OriginalCorrelation")
	}
	if !modelGateValid(current.Gate) {
		return result(Conflict, "UNKNOWN_GATE", "Gate")
	}
	if current.Gate == Held {
		return result(PendingObligation, "HOLD_PRESERVED", "Gate")
	}
	if current.Gate != VerifiedForProfile {
		return result(NeedsEvidence, "RECORD_FRAME_NOT_QUALIFIED", "Gate")
	}
	if !modelID(current.FreshnessOwnerID, limits) {
		return result(NeedsEvidence, "EXPLICIT_FRESHNESS_OWNER_REQUIRED", "FreshnessOwnerID")
	}
	if r := modelOwnerFact(plan.Freshness, current.FreshnessOwnerID, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if r := modelOwnerFact(current.Freshness, current.FreshnessOwnerID, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if *plan.Freshness.Value != *current.Freshness.Value {
		return result(Stale, "INDEPENDENT_FRESHNESS_CHANGED", "Freshness")
	}
	if !modelID(plan.RequiredRecordCapability, limits) || plan.RequiredRecordCapability != current.FreshRecordCapability {
		return result(NeedsEvidence, "FRESH_RECORD_CAPABILITY_MISSING", "RequiredRecordCapability")
	}
	if plan.ExpectedProjectionRevision != projection.Revision {
		return result(Stale, "PROJECTION_CAS_MISMATCH", "ExpectedProjectionRevision")
	}
	if !modelIDSet(plan.SourceReceiptRefs, limits, true) || !modelIDSet(plan.SourceWitnessRefs, limits, true) || !modelIDSet(current.SourceReceiptRefs, limits, true) || !modelIDSet(current.SourceWitnessRefs, limits, true) {
		return result(NeedsEvidence, "SOURCE_REFERENCE_COVERAGE", "SourceRefs")
	}
	if !modelSetEqual(plan.SourceReceiptRefs, current.SourceReceiptRefs) || !modelSetEqual(plan.SourceWitnessRefs, current.SourceWitnessRefs) {
		return result(Conflict, "SOURCE_WITNESS_MISMATCH", "SourceRefs")
	}
	if !slices.Contains(plan.SourceReceiptRefs, plan.IntendedMetadataDelta.SourceReceiptRef) || !modelID(plan.IntendedMetadataDelta.MetadataRef, limits) {
		return result(Conflict, "METADATA_SOURCE_MISMATCH", "IntendedMetadataDelta")
	}
	if r := modelVectorEqual(plan.CurrentSourceVector, current.SourceVector, limits); r.Kind != StructurallyCoherent {
		return r
	}
	return coherent()
}

type RecordTransfer struct {
	OriginalCorrelation                                                Correlation
	Purpose                                                            Purpose
	SourceOwnerID, DestinationOwnerID, ReceiptRef, TargetProjectionRef ID
	ExpectedProjectionRevision                                         uint64
	RequiredRecordCapability                                           ID
	Freshness                                                          Fact[ID]
}

// ValidateRecordTransfer describes fixed receipt transfer only. It does not
// resend a business effect and contains no payload, provider URL, or dispatch.
func ValidateRecordTransfer(transfer RecordTransfer, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if transfer.ExpectedProjectionRevision == 0 {
		return result(NeedsEvidence, "EXPLICIT_PROJECTION_REVISION_REQUIRED", "ExpectedProjectionRevision")
	}
	if transfer.Purpose != PurposeRecordTransfer {
		return result(Conflict, "NOT_FIXED_RECORD_TRANSFER", "Purpose")
	}
	if r := modelCorrelationValid(transfer.OriginalCorrelation, limits); r.Kind != StructurallyCoherent {
		return r
	}
	for _, id := range []ID{transfer.SourceOwnerID, transfer.DestinationOwnerID, transfer.ReceiptRef, transfer.TargetProjectionRef, transfer.RequiredRecordCapability} {
		if !modelID(id, limits) {
			return result(NeedsEvidence, "RECORD_TRANSFER_FIELD_MISSING", "RecordTransfer")
		}
	}
	if r := modelOwnerFact(transfer.Freshness, transfer.DestinationOwnerID, limits); r.Kind != StructurallyCoherent {
		return r
	}
	return coherent()
}

type RestoreFreshness struct {
	OriginalCorrelation                    Correlation
	RestorePointRef, IndependentWitnessRef ID
	IndependentFreshness                   Fact[ID]
	Gate                                   Gate
}

// AssessRestoreFreshness requires separate independent freshness claims. Equality
// of restored records, backup hashes, or counters alone is never sufficient.
func AssessRestoreFreshness(restore RestoreFreshness, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if r := modelCorrelationValid(restore.OriginalCorrelation, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelID(restore.RestorePointRef, limits) || !modelID(restore.IndependentWitnessRef, limits) {
		return result(NeedsEvidence, "INDEPENDENT_RESTORE_WITNESS_REQUIRED", "RestoreFreshness")
	}
	if !modelGateValid(restore.Gate) {
		return result(Conflict, "UNKNOWN_GATE", "Gate")
	}
	if restore.Gate == Held {
		return result(PendingObligation, "RESTORE_HOLD_PRESERVED", "Gate")
	}
	if restore.Gate != VerifiedForProfile {
		return result(NeedsEvidence, "RESTORE_PROFILE_UNQUALIFIED", "Gate")
	}
	if r := modelFactID(restore.IndependentFreshness, limits); r.Kind != StructurallyCoherent {
		return r
	}
	if *restore.IndependentFreshness.Value != restore.IndependentWitnessRef {
		return result(Conflict, "INDEPENDENT_WITNESS_MISMATCH", "IndependentFreshness")
	}
	return coherent()
}

type FanoutChild struct {
	EffectID     ID
	Sink         ResourceRef
	Observations ObservationSet
}
type FanoutModel struct {
	ParentEffectID ID
	Correlation    Correlation
	Children       []FanoutChild
}

// AssessFanout requires each channel/sink's remote terminal observation, keeping
// detached children and unknown outcomes outstanding after parent cancellation.
func AssessFanout(fanout FanoutModel, limits Limits) Result {
	if r := modelLimitsValid(limits); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelID(fanout.ParentEffectID, limits) || len(fanout.Children) == 0 {
		return result(NeedsEvidence, "FANOUT_COVERAGE_MISSING", "Fanout")
	}
	if len(fanout.Children) > limits.MaxRefs {
		return result(Conflict, "LIMIT_EXCEEDED", "Children")
	}
	if r := modelCorrelationValid(fanout.Correlation, limits); r.Kind != StructurallyCoherent {
		return r
	}
	aggregate := coherent()
	for i, child := range fanout.Children {
		if !modelID(child.EffectID, limits) || child.EffectID == fanout.ParentEffectID || !modelResourceValid(child.Sink, limits) {
			return result(Conflict, "FANOUT_CHILD_SCOPE_INVALID", "Children")
		}
		for _, earlier := range fanout.Children[:i] {
			if child.EffectID == earlier.EffectID || child.Sink == earlier.Sink {
				return result(Conflict, "DUPLICATE_FANOUT_CHILD_OR_SINK", "Children")
			}
		}
		if r := AssessObservations(child.Observations, limits); r.Kind != StructurallyCoherent {
			if r.Kind == Conflict || r.Kind == NeedsEvidence || r.Kind == NeedsDecision {
				return r
			}
			if r.Kind == Unknown {
				aggregate = r
			} else if aggregate.Kind != Unknown {
				aggregate = r
			}
		}
		terminal := false
		for _, observation := range child.Observations.Observations {
			if observation.Address.Kind == Provider && observation.Address.SubBoundary == RemoteTerminal && observation.Address.Resource == child.Sink {
				r := modelTerminalObservation(observation, limits)
				if r.Kind == StructurallyCoherent {
					terminal = true
				}
				if r.Kind == Unknown {
					aggregate = r
				}
			}
		}
		if !terminal && aggregate.Kind != Unknown {
			aggregate = result(PendingObligation, "CHILD_REMOTE_TERMINAL_MISSING", "Children")
		}
	}
	return aggregate
}

func modelLimitsValid(l Limits) Result { return ValidateLimits(l) }
func modelID(id ID, l Limits) bool     { return ValidateID(id, l).Kind == StructurallyCoherent }
func modelGateValid(g Gate) bool       { return g == Held || g == NotQualified || g == VerifiedForProfile }
func modelIDSet(ids []ID, l Limits, nonempty bool) bool {
	if len(ids) > l.MaxRefs || nonempty && len(ids) == 0 {
		return false
	}
	for i, id := range ids {
		if !modelID(id, l) || slices.Contains(ids[:i], id) {
			return false
		}
	}
	return true
}
func modelSetEqual(a, b []ID) bool {
	if len(a) != len(b) {
		return false
	}
	for _, id := range a {
		if !slices.Contains(b, id) {
			return false
		}
	}
	return true
}
func modelResourceValid(r ResourceRef, l Limits) bool {
	return validResource(r.Kind) && modelID(r.OwnerID, l) && modelID(r.ResourceID, l)
}
func modelNamespaceValid(ns Namespace, l Limits) bool {
	if ns.OwnerID != ns.Resource.OwnerID || !modelResourceValid(ns.Resource, l) || !modelID(ns.OwnerID, l) || !modelID(ns.ProducerIncarnation, l) || !modelID(ns.SequenceEpoch, l) {
		return false
	}
	return validWorkKind(ns.WorkKind)
}
func modelFactID(f Fact[ID], l Limits) Result { return ValidateFact(f, l) }
func modelOwnerFact(f Fact[ID], owner ID, l Limits) Result {
	r := modelFactID(f, l)
	if r.Kind == StructurallyCoherent && f.Observation.OwnerID != owner {
		return result(Conflict, "FACT_OWNER_MISMATCH", "Fact.Observation.OwnerID")
	}
	return r
}
func modelFrontierValid(f Frontier, l Limits) Result {
	if !modelNamespaceValid(f.Namespace, l) {
		return result(Conflict, "FRONTIER_NAMESPACE_INVALID", "Frontier.Namespace")
	}
	switch f.Kind {
	case "EMPTY":
		if !modelID(f.EmptyFactRef, l) || f.LastSequence != nil {
			return result(Conflict, "EMPTY_FRONTIER_REQUIRES_FACT", "Frontier")
		}
	case "THROUGH":
		if f.LastSequence == nil || *f.LastSequence < 1 || f.EmptyFactRef != "" {
			return result(Conflict, "THROUGH_FRONTIER_REQUIRES_POSITIVE_SEQUENCE", "Frontier")
		}
	default:
		return result(Conflict, "UNKNOWN_FRONTIER_KIND", "Frontier.Kind")
	}
	return coherent()
}
func modelFrontierLast(f Frontier) uint64 {
	if f.Kind == "THROUGH" && f.LastSequence != nil {
		return *f.LastSequence
	}
	return 0
}
func modelClosureValid(c RegistrationClosure, l Limits) Result {
	if !modelNamespaceValid(c.Namespace, l) || c.OwnerID != c.Namespace.OwnerID || !modelID(c.MembershipSnapshot, l) || !modelID(c.AdmissionEpoch, l) || !modelID(c.RegistrationEpoch, l) || !modelID(c.DurableFactRef, l) {
		return result(Conflict, "REGISTRATION_CLOSURE_INVALID", "RegistrationClosed")
	}
	return coherent()
}
func modelWorkValid(w WorkRef, l Limits) Result {
	if !modelID(w.WorkID, l) || !modelNamespaceValid(w.Namespace, l) || !modelID(w.AdmissionRecordRef, l) || w.Sequence < 1 || w.WorkID == w.ParentWorkID || w.ParentWorkID != "" && !modelID(w.ParentWorkID, l) {
		return result(Conflict, "WORK_REFERENCE_INVALID", "Work")
	}
	return coherent()
}
func modelWorkIndex(works []RegisteredWork, id ID) int {
	for i, w := range works {
		if w.Work.WorkID == id {
			return i
		}
	}
	return -1
}
func modelLastAccepted(works []RegisteredWork, ns Namespace) uint64 {
	var last uint64
	for _, w := range works {
		if w.Work.Namespace == ns && w.Work.Sequence > last {
			last = w.Work.Sequence
		}
	}
	return last
}
func modelAdmissionValid(m AdmissionModel, l Limits) Result {
	if r := modelLimitsValid(l); r.Kind != StructurallyCoherent {
		return r
	}
	if m.Key.ContractRef.ContractID != CC03 || !modelID(m.Key.OwnerID, l) || !modelID(m.Key.RecordID, l) || !modelID(m.MembershipSnapshot, l) || !modelID(m.AdmissionEpoch, l) || m.Revision == 0 || !modelGateValid(m.Gate) {
		return result(NeedsEvidence, "ADMISSION_MODEL_FIELDS_MISSING", "Admission")
	}
	if m.Lane != Open && m.Lane != Closing && m.Lane != Closed || m.Drain != Waiting && m.Drain != Proved {
		return result(Conflict, "UNKNOWN_ADMISSION_STATE", "Admission")
	}
	if m.Drain == Proved {
		return result(NeedsEvidence, "DRAIN_PROOF_REQUIRED_SEPARATELY", "Drain")
	}
	if !modelIDSet(m.RequiredGuardRefs, l, true) || !modelIDSet(m.InstalledGuardRefs, l, false) || len(m.AllowedNamespaces) == 0 || len(m.AllowedNamespaces) > l.MaxPartitions || len(m.Works) > l.MaxRefs || len(m.Cut) > l.MaxPartitions || len(m.ClosedRegistrations) > l.MaxPartitions {
		return result(Conflict, "ADMISSION_COVERAGE_OR_LIMIT_INVALID", "Admission")
	}
	for _, guard := range m.InstalledGuardRefs {
		if !slices.Contains(m.RequiredGuardRefs, guard) {
			return result(Conflict, "UNREGISTERED_GUARD", "InstalledGuardRefs")
		}
	}
	for i, ns := range m.AllowedNamespaces {
		if !modelNamespaceValid(ns, l) || slices.Contains(m.AllowedNamespaces[:i], ns) {
			return result(Conflict, "NAMESPACE_INVALID_OR_DUPLICATE", "AllowedNamespaces")
		}
	}
	if m.Lane == Closed {
		for _, guard := range m.RequiredGuardRefs {
			if !slices.Contains(m.InstalledGuardRefs, guard) {
				return result(Conflict, "CLOSED_WITHOUT_ALL_GUARD_ACKS", "InstalledGuardRefs")
			}
		}
	}
	if m.Lane == Open && len(m.Cut) != 0 {
		return result(Conflict, "OPEN_WITH_CUT", "Cut")
	}
	if m.Lane != Open {
		if len(m.Cut) != len(m.AllowedNamespaces) {
			return result(Conflict, "CUT_COVERAGE_MISSING", "Cut")
		}
		for i, f := range m.Cut {
			if r := modelFrontierValid(f, l); r.Kind != StructurallyCoherent {
				return r
			}
			if !slices.Contains(m.AllowedNamespaces, f.Namespace) || modelFrontierLast(f) > modelLastAccepted(m.Works, f.Namespace) {
				return result(Conflict, "CUT_SCOPE_OR_FRONTIER_MISMATCH", "Cut")
			}
			for _, old := range m.Cut[:i] {
				if old.Namespace == f.Namespace {
					return result(Conflict, "DUPLICATE_CUT_NAMESPACE", "Cut")
				}
			}
		}
	}
	for i, c := range m.ClosedRegistrations {
		if r := modelClosureValid(c, l); r.Kind != StructurallyCoherent {
			return r
		}
		if m.Lane != Closed || c.MembershipSnapshot != m.MembershipSnapshot || c.AdmissionEpoch != m.AdmissionEpoch || !slices.Contains(m.AllowedNamespaces, c.Namespace) {
			return result(Conflict, "CLOSURE_SCOPE_MISMATCH", "ClosedRegistrations")
		}
		for _, old := range m.ClosedRegistrations[:i] {
			if old.Namespace == c.Namespace {
				return result(Conflict, "DUPLICATE_REGISTRATION_CLOSURE", "ClosedRegistrations")
			}
		}
	}
	for i, w := range m.Works {
		if r := modelWorkValid(w.Work, l); r.Kind != StructurallyCoherent {
			return r
		}
		if !slices.Contains(m.AllowedNamespaces, w.Work.Namespace) || w.Work.AdmissionRecordRef != m.Key.RecordID || modelWorkIndex(m.Works[:i], w.Work.WorkID) >= 0 {
			return result(Conflict, "WORK_SCOPE_OR_ID_CONFLICT", "Works")
		}
		last := modelLastAccepted(m.Works[:i], w.Work.Namespace)
		if last == math.MaxUint64 || w.Work.Sequence != last+1 {
			return result(Conflict, "WORK_SEQUENCE_NOT_CONTIGUOUS", "Works")
		}
		if w.Work.ParentWorkID != "" && modelWorkIndex(m.Works[:i], w.Work.ParentWorkID) < 0 {
			return result(Conflict, "PARENT_NOT_DURABLY_ACCEPTED", "Works")
		}
		if r := modelOwnerFact(w.DurableRegistration, w.Work.Namespace.OwnerID, l); r.Kind != StructurallyCoherent {
			return r
		}
		if w.HandoffReceipt.State != "" {
			if r := modelOwnerFact(w.HandoffReceipt, w.Work.Namespace.OwnerID, l); r.Kind != StructurallyCoherent {
				return r
			}
		}
	}
	return coherent()
}
func modelCompletionValid(record CompletionRecord, l Limits) Result {
	if r := modelLimitsValid(l); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelID(record.ClosureID, l) || !modelID(record.ReservationRefs.DomainClosureRef, l) || !modelID(record.TerminalPolicyRef, l) || !modelIDSet(record.ProofRefs, l, true) {
		return result(NeedsEvidence, "COMPLETION_FIELDS_MISSING", "Completion")
	}
	if r := modelVectorValid(record.SourceVector, l); r.Kind != StructurallyCoherent {
		return r
	}
	if len(record.Partitions) == 0 || len(record.Partitions) > l.MaxPartitions || len(record.PhysicalObligations) > l.MaxObligations || len(record.LineageClosureFacts) > l.MaxPartitions || !modelIDSet(record.Effects, l, false) {
		return result(Conflict, "COMPLETION_COVERAGE_OR_LIMIT_INVALID", "Completion")
	}
	if len(record.Effects) > 0 && len(record.RequiredEffects) == 0 {
		return result(NeedsEvidence, "EFFECT_REQUIREMENTS_MISSING", "RequiredEffects")
	}
	if len(record.RequiredEffects) != len(record.Effects) || len(record.RequiredEffects) > l.MaxRefs {
		return result(Conflict, "EFFECT_REQUIREMENT_COVERAGE_MISMATCH", "RequiredEffects")
	}
	for i, required := range record.RequiredEffects {
		if !modelID(required.EffectID, l) || !slices.Contains(record.Effects, required.EffectID) || !modelResourceValid(required.Target, l) {
			return result(Conflict, "EFFECT_REQUIREMENT_INVALID", "RequiredEffects")
		}
		if r := modelCorrelationValid(required.Correlation, l); r.Kind != StructurallyCoherent {
			return r
		}
		for _, old := range record.RequiredEffects[:i] {
			if old.EffectID == required.EffectID {
				return result(Conflict, "DUPLICATE_EFFECT_REQUIREMENT", "RequiredEffects")
			}
		}
	}
	if r := ValidateOptionalRef(record.ReservationRefs.CPReservation, l); r.Kind != StructurallyCoherent {
		return r
	}
	pending, unknown := false, false
	var snapshot, epoch ID
	for i, p := range record.Partitions {
		if !modelNamespaceValid(p.Namespace, l) {
			return result(Conflict, "PARTITION_NAMESPACE_INVALID", "Partitions")
		}
		for _, old := range record.Partitions[:i] {
			if old.Namespace == p.Namespace {
				return result(Conflict, "DUPLICATE_PARTITION", "Partitions")
			}
		}
		for _, f := range []Frontier{p.CutFrontier, p.FinalAcceptedFrontier, p.TerminalFrontier} {
			if r := modelFrontierValid(f, l); r.Kind != StructurallyCoherent {
				return r
			}
			if f.Namespace != p.Namespace {
				return result(Conflict, "FRONTIER_NAMESPACE_MISMATCH", "Partitions")
			}
		}
		cut, accepted, terminal := modelFrontierLast(p.CutFrontier), modelFrontierLast(p.FinalAcceptedFrontier), modelFrontierLast(p.TerminalFrontier)
		if cut > accepted || terminal > accepted {
			return result(Conflict, "FRONTIER_ORDER_INVALID", "Partitions")
		}
		if len(p.Holes) > l.MaxRanges || !modelIDSet(p.PendingRegistrationRefs, l, false) {
			return result(Conflict, "LIMIT_EXCEEDED_OR_DUPLICATE", "Partitions")
		}
		for j, hole := range p.Holes {
			if hole.First < 1 || hole.First > hole.Last || hole.Last > accepted {
				return result(Conflict, "HOLE_RANGE_INVALID", "Partitions.Holes")
			}
			for _, old := range p.Holes[:j] {
				if hole.First <= old.Last && old.First <= hole.Last {
					return result(Conflict, "HOLE_RANGE_OVERLAP", "Partitions.Holes")
				}
			}
			if hole.First <= terminal {
				return result(Conflict, "CONTIGUOUS_TERMINAL_HAS_HOLE", "Partitions.TerminalFrontier")
			}
		}
		if r := modelClosureValid(p.RegistrationClosed, l); r.Kind != StructurallyCoherent {
			return r
		}
		if p.RegistrationClosed.Namespace != p.Namespace {
			return result(Conflict, "REGISTRATION_NAMESPACE_MISMATCH", "Partitions.RegistrationClosed")
		}
		if i == 0 {
			snapshot, epoch = p.RegistrationClosed.MembershipSnapshot, p.RegistrationClosed.AdmissionEpoch
		}
		if p.RegistrationClosed.MembershipSnapshot != snapshot || p.RegistrationClosed.AdmissionEpoch != epoch {
			return result(Conflict, "REGISTRATION_MEMBERSHIP_OR_EPOCH_MISMATCH", "Partitions")
		}
		if !slices.Contains(record.LineageClosureFacts, p.RegistrationClosed) {
			return result(NeedsEvidence, "LINEAGE_CLOSURE_FACT_MISSING", "LineageClosureFacts")
		}
		if terminal != accepted || len(p.Holes) > 0 || len(p.PendingRegistrationRefs) > 0 {
			pending = true
		}
	}
	for i, o := range record.PhysicalObligations {
		for _, old := range record.PhysicalObligations[:i] {
			if old.ObligationID == o.ObligationID {
				return result(Conflict, "DUPLICATE_PHYSICAL_OBLIGATION", "PhysicalObligations")
			}
		}
		r := modelObligationValid(o, l)
		if r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation && r.Kind != NeedsEvidence {
			return r
		}
		if r.Kind == Unknown {
			unknown = true
		} else if r.Kind != StructurallyCoherent {
			pending = true
		}
	}
	if unknown {
		return result(Unknown, "PHYSICAL_RELEASE_UNKNOWN", "PhysicalObligations")
	}
	if pending {
		return result(PendingObligation, "CLOSURE_OBLIGATION_PENDING", "Completion")
	}
	return coherent()
}
func modelCorrelationValid(c Correlation, l Limits) Result {
	for _, id := range []ID{c.RequestID, c.OperationID, c.AttemptID, c.StepID, c.ExecutionKey} {
		if !modelID(id, l) {
			return result(NeedsEvidence, "CORRELATION_FIELD_MISSING", "Correlation")
		}
	}
	return coherent()
}
func modelEffectValid(m EffectModel, l Limits) Result {
	if r := modelLimitsValid(l); r.Kind != StructurallyCoherent {
		return r
	}
	if !modelID(m.OwnerID, l) || !modelID(m.EffectID, l) || !modelResourceValid(m.Target, l) || m.Revision == 0 || m.Purpose != PurposeEffect || m.ContentRef.Kind != "NONSECRET_METADATA_REF" || !modelID(m.ContentRef.Ref, l) {
		return result(NeedsEvidence, "EFFECT_MODEL_FIELDS_MISSING", "Effect")
	}
	if r := modelCorrelationValid(m.Correlation, l); r.Kind != StructurallyCoherent {
		return r
	}
	if m.DispatchState != NoIntent && m.DispatchState != IntentRecorded && m.DispatchState != MayDispatch {
		return result(Conflict, "UNKNOWN_DISPATCH_STATE", "DispatchState")
	}
	if len(m.ExpectedBoundaries) == 0 {
		return result(NeedsEvidence, "EXPLICIT_EXPECTED_BOUNDARIES_REQUIRED", "ExpectedBoundaries")
	}
	if len(m.ExpectedBoundaries) > l.MaxRefs {
		return result(Conflict, "LIMIT_EXCEEDED", "ExpectedBoundaries")
	}
	for i, address := range m.ExpectedBoundaries {
		if r := ValidateBoundary(address, l); r.Kind != StructurallyCoherent {
			return r
		}
		if address.Kind == Provider && (address.Resource != m.Target || address.OwnerID != m.Target.OwnerID) {
			return result(Conflict, "PROVIDER_TARGET_MISMATCH", "ExpectedBoundaries")
		}
		if slices.Contains(m.ExpectedBoundaries[:i], address) {
			return result(Conflict, "DUPLICATE_EXPECTED_BOUNDARY", "ExpectedBoundaries")
		}
	}
	if len(m.ObservationHistory) > l.MaxRefs {
		return result(Conflict, "LIMIT_EXCEEDED", "ObservationHistory")
	}
	for _, history := range m.ObservationHistory {
		if !slices.Contains(m.ExpectedBoundaries, history.Address) {
			return result(Conflict, "UNEXPECTED_HISTORY_BOUNDARY", "ObservationHistory")
		}
	}
	for _, observation := range m.Observations.Observations {
		if !slices.Contains(m.ExpectedBoundaries, observation.Address) {
			return result(Conflict, "UNEXPECTED_CURRENT_BOUNDARY", "Observations")
		}
	}
	if r := AssessObservationHistory(ObservationSet{Observations: m.ObservationHistory, Gate: m.Observations.Gate}, l); r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation && r.Kind != NeedsEvidence {
		return r
	}
	for _, history := range m.ObservationHistory {
		if !modelKnownTerminal(history) {
			continue
		}
		retained := false
		for _, observation := range m.Observations.Observations {
			if observation.Address != history.Address {
				continue
			}
			if modelKnownObservationConflict(history, observation) {
				return result(Conflict, "HISTORY_CURRENT_KNOWN_CONTRADICTION", "Observations")
			}
			if modelKnownTerminal(observation) {
				retained = true
			}
		}
		if !retained {
			return result(Conflict, "CURRENT_VIEW_LOST_KNOWN_HISTORY", "Observations")
		}
	}
	if r := AssessObservations(m.Observations, l); r.Kind != StructurallyCoherent && r.Kind != Unknown && r.Kind != PendingObligation && r.Kind != NeedsEvidence {
		return r
	}
	if m.DispatchState == NoIntent && (len(m.Observations.Observations) > 0 || len(m.ObservationHistory) > 0) {
		return result(Conflict, "NO_INTENT_WITH_EFFECT_HISTORY", "DispatchState")
	}
	if m.DispatchState == IntentRecorded {
		for _, o := range append(slices.Clone(m.Observations.Observations), m.ObservationHistory...) {
			if o.Address.Kind != LocalPersistence || o.Address.SubBoundary != Intent {
				return result(Conflict, "INTENT_ONLY_WITH_DISPATCH_HISTORY", "DispatchState")
			}
		}
	}
	if !modelGateValid(m.Observations.Gate) || len(m.Observations.Observations) > l.MaxRefs {
		return result(Conflict, "OBSERVATION_MODEL_INVALID", "Observations")
	}
	return coherent()
}
func modelBoundaryValid(a BoundaryAddress, l Limits) bool {
	return ValidateBoundary(a, l).Kind == StructurallyCoherent
}
func modelObservationValid(o BoundaryObservation, l Limits) Result {
	return ValidateBoundaryObservation(o, l)
}
func modelKnownPending(o BoundaryObservation) bool {
	return o.Knowledge == Known && o.Outcome.Value != nil && *o.Outcome.Value == Pending
}
func modelKnownTerminal(o BoundaryObservation) bool {
	return o.Knowledge == Known && o.Outcome.Value != nil && member(*o.Outcome.Value, SuccessAtBoundary, FailureAtBoundary, NoEffectProven)
}
func modelKnownObservationConflict(a, b BoundaryObservation) bool {
	return modelKnownTerminal(a) && modelKnownTerminal(b) && (*a.Outcome.Value != *b.Outcome.Value || a.Presence != b.Presence)
}

// modelPendingAdvance compares owner-local observation revisions, never wall time.
func modelPendingAdvance(pending, terminal BoundaryObservation) Result {
	if !modelKnownPending(pending) || !modelKnownTerminal(terminal) {
		return result(Conflict, "NOT_PENDING_TO_TERMINAL", "Observation")
	}
	a, b := pending.Outcome.Observation, terminal.Outcome.Observation
	if a == nil || b == nil {
		return result(NeedsEvidence, "OWNER_FRAME_REQUIRED", "Observation")
	}
	if pending.Address != terminal.Address || a.OwnerID != b.OwnerID || a.Scope != b.Scope {
		return result(Conflict, "OWNER_FRAME_CHAIN_MISMATCH", "Observation")
	}
	if b.SourceRevision <= a.SourceRevision {
		return result(Stale, "TERMINAL_OBSERVATION_NOT_FRESHER", "Observation")
	}
	if a.FrameRef == b.FrameRef {
		return result(Conflict, "OWNER_FRAME_REUSED", "Observation")
	}
	return coherent()
}
func modelObligationValid(o PhysicalObligation, l Limits) Result {
	if !modelID(o.ObligationID, l) || !modelID(o.OwnerID, l) || !modelResourceValid(o.Resource, l) || o.OwnerID != o.Resource.OwnerID || !modelID(o.WorkRef, l) {
		return result(Conflict, "OBLIGATION_SCOPE_INVALID", "PhysicalObligation")
	}
	switch o.Kind {
	case FD, DBConnection, Child, Callback, WriterBorrow, QueueObligation, Lineage:
	default:
		return result(Conflict, "UNKNOWN_OBLIGATION_KIND", "PhysicalObligation.Kind")
	}
	if o.Knowledge == KnowledgeConflict {
		return result(Conflict, "OBLIGATION_CONFLICT", "PhysicalObligation")
	}
	r := modelOwnerFact(o.Terminal, o.OwnerID, l)
	if o.Knowledge == KnowledgeUnknown {
		if r.Kind != NeedsEvidence {
			return result(Conflict, "UNKNOWN_RELEASE_REQUIRES_UNAVAILABLE_FACT", "PhysicalObligation")
		}
		return result(Unknown, "PHYSICAL_RELEASE_UNKNOWN", "PhysicalObligation")
	}
	if o.Knowledge != Known {
		return result(Conflict, "UNKNOWN_KNOWLEDGE", "PhysicalObligation.Knowledge")
	}
	if r.Kind == NeedsEvidence {
		return result(PendingObligation, "TERMINAL_FACT_PENDING", "PhysicalObligation")
	}
	return r
}
func modelTerminalObservation(o BoundaryObservation, l Limits) Result {
	r := modelObservationValid(o, l)
	if r.Kind != StructurallyCoherent {
		return r
	}
	if o.Address.Kind != Provider || o.Address.SubBoundary != RemoteTerminal {
		return result(PendingObligation, "REMOTE_TERMINAL_REQUIRED", "Observation.Address")
	}
	return coherent()
}
func modelQueryValid(q QueryTarget, l Limits) Result {
	if r := modelCorrelationValid(q.Correlation, l); r.Kind != StructurallyCoherent {
		return r
	}
	for _, id := range []ID{q.BindingRef, q.ExecutionKey, q.TargetRecordRef, q.Scope} {
		if !modelID(id, l) {
			return result(NeedsEvidence, "QUERY_FIELD_MISSING", "QueryTarget")
		}
	}
	if q.ExecutionKey != q.Correlation.ExecutionKey {
		return result(Conflict, "EXECUTION_KEY_MISMATCH", "QueryTarget")
	}
	switch q.Purpose {
	case PurposeLookup, PurposeEffect, PurposeClosure:
		return coherent()
	}
	return result(Conflict, "NOT_OBSERVATION_PURPOSE", "QueryTarget.Purpose")
}
func modelVectorValid(v RevisionVector, l Limits) Result {
	if len(v) == 0 || len(v) > l.MaxAxes {
		return result(NeedsEvidence, "SOURCE_VECTOR_REQUIRED", "CurrentSourceVector")
	}
	c := newChecker(l)
	for i, e := range v {
		c.axisKey(e.Key, "SourceVector")
		c.id(e.Incarnation, "SourceVector.Incarnation")
		c.value(e, "SourceVector.Value")
		for _, old := range v[:i] {
			if old.Key == e.Key {
				return result(Conflict, "DUPLICATE_SOURCE_AXIS", "CurrentSourceVector")
			}
		}
	}
	return c.finish()
}
func modelVectorEqual(a, b RevisionVector, l Limits) Result {
	if r := modelVectorValid(a, l); r.Kind != StructurallyCoherent {
		return r
	}
	if r := modelVectorValid(b, l); r.Kind != StructurallyCoherent {
		return r
	}
	if len(a) == 0 || len(a) > l.MaxAxes || len(b) > l.MaxAxes {
		return result(NeedsEvidence, "SOURCE_VECTOR_REQUIRED", "CurrentSourceVector")
	}
	if len(a) != len(b) {
		return result(Conflict, "SOURCE_VECTOR_MISMATCH", "CurrentSourceVector")
	}
	for i, entry := range a {
		if !modelResourceValid(entry.Key.Resource, l) || entry.Key.OwnerID != entry.Key.Resource.OwnerID || !modelID(entry.Incarnation, l) || !modelID(entry.SourceFrameRef, l) {
			return result(Conflict, "SOURCE_VECTOR_ENTRY_INVALID", "CurrentSourceVector")
		}
		for _, old := range a[:i] {
			if old.Key == entry.Key {
				return result(Conflict, "DUPLICATE_SOURCE_AXIS", "CurrentSourceVector")
			}
		}
		j := -1
		for k, other := range b {
			if entry.Key == other.Key {
				if j >= 0 {
					return result(Conflict, "DUPLICATE_SOURCE_AXIS", "CurrentSourceVector")
				}
				j = k
			}
		}
		if j < 0 || entry.Incarnation != b[j].Incarnation {
			return result(Conflict, "SOURCE_OWNER_RESOURCE_OR_INCARNATION_MISMATCH", "CurrentSourceVector")
		}
		other := b[j]
		if entry.Value.Kind != other.Value.Kind {
			return result(Conflict, "SOURCE_VALUE_TYPE_MISMATCH", "CurrentSourceVector")
		}
		switch entry.Value.Kind {
		case PresentUint64:
			if entry.Value.Uint64 == nil || other.Value.Uint64 == nil || entry.Value.Literal != nil || other.Value.Literal != nil || entry.Value.Unavailable != nil || other.Value.Unavailable != nil || *entry.Value.Uint64 != *other.Value.Uint64 {
				return result(Conflict, "SOURCE_VALUE_MISMATCH", "CurrentSourceVector")
			}
		case PresentLiteral:
			if entry.Value.Literal == nil || other.Value.Literal == nil || entry.Value.Uint64 != nil || other.Value.Uint64 != nil || entry.Value.Unavailable != nil || other.Value.Unavailable != nil || !modelID(*entry.Value.Literal, l) || *entry.Value.Literal != *other.Value.Literal {
				return result(Conflict, "SOURCE_VALUE_MISMATCH", "CurrentSourceVector")
			}
		case ValueUnavailable:
			return result(NeedsEvidence, "SOURCE_VALUE_UNAVAILABLE", "CurrentSourceVector")
		default:
			return result(Conflict, "UNKNOWN_SOURCE_VALUE_KIND", "CurrentSourceVector")
		}
	}
	return coherent()
}

func modelCloneFact[T any](f Fact[T]) Fact[T] {
	out := f
	if f.Value != nil {
		v := *f.Value
		out.Value = &v
	}
	if f.Observation != nil {
		v := *f.Observation
		out.Observation = &v
	}
	if f.Unavailable != nil {
		v := *f.Unavailable
		v.MissingRefs = slices.Clone(v.MissingRefs)
		out.Unavailable = &v
	}
	return out
}
func modelCloneFrontiers(in []Frontier) []Frontier {
	out := slices.Clone(in)
	for i := range out {
		if out[i].LastSequence != nil {
			v := *out[i].LastSequence
			out[i].LastSequence = &v
		}
	}
	return out
}
func modelCloneAdmission(in AdmissionModel) AdmissionModel {
	out := in
	out.RequiredGuardRefs = slices.Clone(in.RequiredGuardRefs)
	out.InstalledGuardRefs = slices.Clone(in.InstalledGuardRefs)
	out.AllowedNamespaces = slices.Clone(in.AllowedNamespaces)
	out.Cut = modelCloneFrontiers(in.Cut)
	out.Works = slices.Clone(in.Works)
	for i := range out.Works {
		out.Works[i].DurableRegistration = modelCloneFact(out.Works[i].DurableRegistration)
		out.Works[i].HandoffReceipt = modelCloneFact(out.Works[i].HandoffReceipt)
	}
	out.ClosedRegistrations = slices.Clone(in.ClosedRegistrations)
	return out
}
func modelCloneObservation(o BoundaryObservation) BoundaryObservation {
	o.Outcome = modelCloneFact(o.Outcome)
	if o.Details != nil {
		d := *o.Details
		if d.RawExtent != nil {
			v := *d.RawExtent
			if v.Offset != nil {
				x := *v.Offset
				v.Offset = &x
			}
			if v.Length != nil {
				x := *v.Length
				v.Length = &x
			}
			if v.RequestedLength != nil {
				x := *v.RequestedLength
				v.RequestedLength = &x
			}
			d.RawExtent = &v
		}
		if d.Durability != nil {
			v := *d.Durability
			d.Durability = &v
		}
		if d.Queue != nil {
			v := *d.Queue
			d.Queue = &v
		}
		o.Details = &d
	}
	return o
}
func modelCloneObservationList(in []BoundaryObservation) []BoundaryObservation {
	out := slices.Clone(in)
	for i := range out {
		out[i] = modelCloneObservation(out[i])
	}
	return out
}
func modelCloneObservations(in ObservationSet) ObservationSet {
	out := in
	out.Observations = slices.Clone(in.Observations)
	for i := range out.Observations {
		out.Observations[i] = modelCloneObservation(out.Observations[i])
	}
	out.PhysicalObligations = slices.Clone(in.PhysicalObligations)
	for i := range out.PhysicalObligations {
		out.PhysicalObligations[i].Terminal = modelCloneFact(out.PhysicalObligations[i].Terminal)
	}
	out.RemainingObligations = slices.Clone(in.RemainingObligations)
	out.Contradictions = slices.Clone(in.Contradictions)
	out.PhaseHints = slices.Clone(in.PhaseHints)
	return out
}
