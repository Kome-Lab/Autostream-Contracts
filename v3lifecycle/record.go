package lifecycle

import "reflect"

func ValidateOptionalRef(v OptionalRef, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	switch v.State {
	case Present:
		c.require(v.Ref != nil && v.AbsenceFactRef == "" && v.Observation != nil && v.Unavailable == nil, "INVALID_PRESENT_REFERENCE", "Reference")
		if v.Ref != nil {
			c.id(*v.Ref, "Reference.Ref")
		}
	case Absent:
		c.require(v.Ref == nil && v.Observation != nil && v.Unavailable == nil, "INVALID_ABSENT_REFERENCE", "Reference")
		c.id(v.AbsenceFactRef, "Reference.AbsenceFact")
	case NotObserved:
		c.require(v.Ref == nil && v.AbsenceFactRef == "" && v.Observation == nil && v.Unavailable != nil, "INVALID_UNKNOWN_REFERENCE", "Reference")
		if v.Unavailable != nil {
			c.id(v.Unavailable.Code, "Reference.Reason")
			c.ids(v.Unavailable.MissingRefs, l.MaxRefs, "Reference.MissingRefs", false)
		}
		c.add(NeedsEvidence, "REFERENCE_NOT_OBSERVED", "Reference")
	default:
		c.add(RefusedUnchanged, "UNKNOWN_REFERENCE_PRESENCE", "Reference")
	}
	if v.Observation != nil {
		o := v.Observation
		c.id(o.OwnerID, "Reference.Owner")
		c.id(o.FrameRef, "Reference.Frame")
		c.id(o.Scope, "Reference.Scope")
		c.require(o.SourceRevision > 0, "MISSING_SOURCE_REVISION", "Reference")
	}
	return c.finish()
}
func (c *checker) domain(d DomainRef) {
	c.id(d.DeploymentID, "Domain.Deployment")
	c.id(d.Realm, "Domain.Realm")
	c.id(d.DomainID, "Domain.ID")
	c.id(d.SnapshotRef, "Domain.Snapshot")
	c.require(d.MembershipRevision > 0, "MISSING_MEMBERSHIP_REVISION", "Domain")
	c.resources(d.Resources, "Domain.Resources", false)
}
func (c *checker) header(h Header, p ComparisonProfile) {
	cr := h.ContractRef
	c.require(validContract(cr.ContractID) && cr.LogicalRevision == 1 && cr.SpecRef == SpecRef, "UNKNOWN_CONTRACT_REVISION", "Header.Contract")
	c.id(cr.ModuleCommitRef, "Header.ModuleCommitRef")
	validSHA := len(cr.ModuleCommitRef) == 40
	for _, r := range string(cr.ModuleCommitRef) {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			validSHA = false
		}
	}
	c.require(validSHA, "EXPLICIT_MODULE_COMMIT_REQUIRED", "Header.ModuleCommitRef")
	c.require(cr == h.RecordKey.ContractRef, "RECORD_KEY_CONTRACT_MISMATCH", "Header.RecordKey")
	c.id(h.RecordKey.OwnerID, "Header.Owner")
	c.id(h.RecordKey.RecordID, "Header.RecordID")
	c.correlation(h.Correlation, "Header.Correlation")
	c.domain(h.DomainRef)
	c.producer(h.ProducerRef, "Header.Producer")
	c.require(h.RecordRevision > 0, "MISSING_RECORD_REVISION", "Header")
	c.require(h.ProfileRef == p.Ref, "PROFILE_REFERENCE_MISMATCH", "Header.Profile")
	if len(h.DomainRef.Resources) > c.limits.MaxMembers {
		return
	}
	for _, res := range h.DomainRef.Resources {
		found := false
		for _, kind := range p.ResourceKinds {
			if res.Kind == kind {
				found = true
			}
		}
		c.require(found, "DOMAIN_RESOURCE_OUTSIDE_PROFILE", "Header.Domain")
	}
}
func (c *checker) reservations(r ReservationRefs, p ComparisonProfile) {
	c.id(r.DomainClosureRef, "Reservation.DomainClosure")
	c.take(ValidateOptionalRef(r.CPReservation, c.limits), "Reservation.CP")
	if p.Ref.ProfileID == ApprovalLocal {
		c.require(r.CPReservation.State == Present, "CP_RESERVATION_REQUIRED_FOR_PROFILE", "Reservation.CP")
	}
}
func (c *checker) namespace(n Namespace, path string) {
	c.id(n.OwnerID, path+".Owner")
	c.resource(n.Resource, path+".Resource")
	c.require(n.OwnerID == n.Resource.OwnerID, "NAMESPACE_OWNER_MISMATCH", path)
	c.require(validWorkKind(n.WorkKind), "UNKNOWN_WORK_KIND", path)
	c.id(n.ProducerIncarnation, path+".Incarnation")
	c.id(n.SequenceEpoch, path+".Epoch")
}
func (c *checker) frontier(f Frontier, path string) {
	c.namespace(f.Namespace, path+".Namespace")
	switch f.Kind {
	case "EMPTY":
		c.id(f.EmptyFactRef, path+".EmptyFact")
		c.require(f.LastSequence == nil, "EMPTY_HAS_SEQUENCE", path)
	case "THROUGH":
		c.require(f.LastSequence != nil && f.EmptyFactRef == "", "INVALID_THROUGH_UNION", path)
		if f.LastSequence != nil {
			c.require(*f.LastSequence > 0, "ZERO_WORK_SEQUENCE", path)
		}
	default:
		c.add(RefusedUnchanged, "UNKNOWN_FRONTIER_KIND", path)
	}
}
func (c *checker) registration(r RegistrationClosure, path string) {
	for _, v := range []ID{r.MembershipSnapshot, r.AdmissionEpoch, r.RegistrationEpoch, r.OwnerID, r.DurableFactRef} {
		c.id(v, path)
	}
	c.namespace(r.Namespace, path+".Namespace")
	c.require(r.OwnerID == r.Namespace.OwnerID, "REGISTRATION_OWNER_MISMATCH", path)
}
func (c *checker) obligation(o PhysicalObligation, path string) {
	c.id(o.ObligationID, path+".ID")
	c.id(o.OwnerID, path+".Owner")
	c.resource(o.Resource, path+".Resource")
	c.id(o.WorkRef, path+".Work")
	c.require(o.OwnerID == o.Resource.OwnerID, "OBLIGATION_OWNER_MISMATCH", path)
	c.require(member(o.Kind, FD, DBConnection, Child, Callback, WriterBorrow, QueueObligation, Lineage), "UNKNOWN_OBLIGATION_KIND", path)
	c.require(validKnowledge(o.Knowledge), "UNKNOWN_KNOWLEDGE", path)
	c.take(ValidateFact(o.Terminal, c.limits), path+".Terminal")
	if o.Knowledge == KnowledgeUnknown {
		c.add(Unknown, "PHYSICAL_RELEASE_UNKNOWN", path)
	}
	if o.Knowledge == KnowledgeConflict {
		c.add(Conflict, "PHYSICAL_RELEASE_CONFLICT", path)
	}
	if o.Knowledge == Known {
		c.require(o.Terminal.State == Observed, "KNOWN_OBLIGATION_WITHOUT_FACT", path)
	}
	if o.Terminal.Observation != nil {
		c.require(o.Terminal.Observation.OwnerID == o.OwnerID, "TERMINAL_FACT_OWNER_MISMATCH", path)
	}
}
func (c *checker) membership(b MembershipRecord, h Header) {
	c.require(word(b.DomainKind, "CP_IDENTITY MEDIA_RUN HOST_MUTATION RELEASE_RETURN"), "UNKNOWN_DOMAIN_KIND", "Membership")
	coordinator := ComponentID("C01")
	if b.DomainKind == "HOST_MUTATION" {
		coordinator = "C10"
	}
	if b.DomainKind == "RELEASE_RETURN" {
		coordinator = "C11"
	}
	c.require(h.ProducerRef.ComponentID == coordinator, "DOMAIN_COORDINATOR_COMPONENT_MISMATCH", "Membership")
	c.id(b.OwnerID, "Membership.Owner")
	c.require(b.OwnerID == h.RecordKey.OwnerID, "MEMBERSHIP_COORDINATOR_MISMATCH", "Membership")
	c.require(b.Revision == h.DomainRef.MembershipRevision, "MEMBERSHIP_REVISION_MISMATCH", "Membership")
	c.take(ValidateOptionalRef(b.PreviousSnapshot, c.limits), "Membership.Previous")
	if o := b.PreviousSnapshot.Observation; o != nil {
		c.require(o.OwnerID == b.OwnerID && o.Scope == h.DomainRef.DomainID, "PREVIOUS_SNAPSHOT_OWNER_SCOPE_MISMATCH", "Membership.Previous")
	}
	if b.PreviousSnapshot.State == Absent {
		c.require(b.Revision == 1, "PREVIOUS_SNAPSHOT_MISSING_AFTER_GENESIS", "Membership.Previous")
	}
	c.take(ValidateFact(b.Coverage, c.limits), "Membership.Coverage")
	if b.Coverage.Observation != nil {
		c.require(b.Coverage.Observation.OwnerID == b.OwnerID && b.Coverage.Observation.Scope == h.DomainRef.DomainID, "COVERAGE_OWNER_SCOPE_MISMATCH", "Membership.Coverage")
	}
	c.require(len(b.Members) <= c.limits.MaxMembers, "TOO_MANY_MEMBERS", "Membership")
	c.require(len(b.SourceFrames) <= c.limits.MaxRefs, "TOO_MANY_FRAMES", "Membership")
	if len(b.Members) > c.limits.MaxMembers || len(b.SourceFrames) > c.limits.MaxRefs {
		return
	}
	c.require(b.CoverageKind == "COMPLETE_ROSTER" && len(b.Members) > 0 || b.CoverageKind == "EXPLICIT_EMPTY" && len(b.Members) == 0 && len(h.DomainRef.Resources) == 0, "ROSTER_EMPTY_REQUIRES_EXPLICIT_FACT", "Membership")
	frames := map[ID]SourceFrame{}
	for _, f := range b.SourceFrames {
		c.frame(f, "Membership.Frame")
		_, dup := frames[f.FrameRef]
		c.require(!dup, "DUPLICATE_FRAME", "Membership")
		frames[f.FrameRef] = f
	}
	seen := map[Member]bool{}
	covered := map[ResourceRef]bool{}
	for _, m := range b.Members {
		c.producer(m.Producer, "Membership.Member")
		c.resource(m.Resource, "Membership.Member.Resource")
		c.id(m.SourceFrameRef, "Membership.Member.Frame")
		c.require(!seen[m], "DUPLICATE_MEMBER", "Membership")
		seen[m] = true
		covered[m.Resource] = true
		f, ok := frames[m.SourceFrameRef]
		c.require(ok && f.OwnerID == m.Resource.OwnerID, "MEMBER_FRAME_OWNER_MISMATCH", "Membership")
		c.require(memberResource(h.DomainRef.Resources, m.Resource), "MEMBER_OUTSIDE_DOMAIN", "Membership")
	}
	for _, r := range h.DomainRef.Resources {
		c.require(covered[r], "UNCOVERED_RESOURCE", "Membership")
	}
	c.ids(b.RequiredAliasIDs, c.limits.MaxRefs, "Membership.RequiredAliases", false)
	c.require(len(b.Aliases) <= c.limits.MaxRefs, "TOO_MANY_ALIASES", "Membership")
	if len(b.Aliases) > c.limits.MaxRefs {
		return
	}
	aliases := map[ID]bool{}
	for _, a := range b.Aliases {
		c.id(a.AliasID, "Membership.Alias")
		c.resource(a.Resource, "Membership.Alias.Resource")
		c.id(a.ResolutionFactRef, "Membership.Alias.Fact")
		c.require(!aliases[a.AliasID], "DUPLICATE_ALIAS", "Membership")
		aliases[a.AliasID] = true
		c.require(memberResource(h.DomainRef.Resources, a.Resource), "ALIAS_OUTSIDE_DOMAIN", "Membership")
	}
	for _, id := range b.RequiredAliasIDs {
		c.require(aliases[id], "MISSING_ALIAS_COVERAGE", "Membership")
	}
	c.resources(b.AffectedRuns, "Membership.AffectedRuns", false)
	if len(b.AffectedRuns) <= c.limits.MaxMembers {
		for _, r := range b.AffectedRuns {
			c.require(r.Kind == Run, "AFFECTED_RUN_KIND", "Membership")
		}
	}
}
func memberResource(rs []ResourceRef, r ResourceRef) bool {
	for _, x := range rs {
		if x == r {
			return true
		}
	}
	return false
}

// ValidateRecord validates structure and explicit evidence gaps; it never verifies an authority.
func ValidateRecord(r Record, p ComparisonProfile) Result {
	if v := ValidateProfile(p); v.Kind != StructurallyCoherent {
		return v
	}
	c := newChecker(p.Limits)
	c.header(r.Header, p)
	count := 0
	for _, yes := range []bool{r.Membership != nil, r.Vector != nil, r.Admission != nil, r.Completion != nil, r.Seal != nil, r.Fence != nil, r.Effect != nil, r.Inspection != nil, r.Reconcile != nil} {
		if yes {
			count++
		}
	}
	c.require(count == 1, "EXACTLY_ONE_CONTRACT_PAYLOAD_REQUIRED", "Record")
	id := r.Header.ContractRef.ContractID
	c.require((id == CC01 && r.Membership != nil) || (id == CC02 && r.Vector != nil) || (id == CC03 && r.Admission != nil) || (id == CC04 && r.Completion != nil) || (id == CC05 && r.Seal != nil) || (id == CC06 && r.Fence != nil) || (id == CC07 && r.Effect != nil) || (id == CC08 && r.Inspection != nil) || (id == CC09 && r.Reconcile != nil), "PAYLOAD_TAG_MISMATCH", "Record")
	if c.kind != StructurallyCoherent {
		return c.finish()
	}
	if b := r.Membership; b != nil {
		c.membership(*b, r.Header)
	}
	if b := r.Vector; b != nil {
		c.recordVector(b.Vector, p, r.Header.DomainRef, "Vector")
	}
	if b := r.Admission; b != nil {
		c.reservations(b.ReservationRefs, p)
		c.id(b.MembershipSnapshot, "Admission.Membership")
		c.require(b.MembershipSnapshot == r.Header.DomainRef.SnapshotRef, "ADMISSION_MEMBERSHIP_MISMATCH", "Admission")
		c.id(b.AdmissionEpoch, "Admission.Epoch")
		c.ids(b.ActorGuardRefs, c.limits.MaxMembers, "Admission.ActorGuards", true)
		c.id(b.TerminalPolicyRef, "Admission.TerminalPolicy")
		c.require(b.TerminalPolicyRef == p.TerminalPolicyRef, "TERMINAL_POLICY_MISMATCH", "Admission")
		c.require(len(b.Cut) > 0 && len(b.Cut) <= c.limits.MaxPartitions, "INVALID_CUT_SIZE", "Admission")
		if len(b.Cut) <= c.limits.MaxPartitions {
			seen := map[Namespace]bool{}
			for _, f := range b.Cut {
				c.frontier(f, "Admission.Cut")
				c.require(memberResource(r.Header.DomainRef.Resources, f.Namespace.Resource), "CUT_RESOURCE_OUTSIDE_DOMAIN", "Admission.Cut")
				c.require(!seen[f.Namespace], "DUPLICATE_PARTITION", "Admission.Cut")
				seen[f.Namespace] = true
			}
		}
	}
	if b := r.Completion; b != nil {
		c.id(b.ClosureID, "Completion.Closure")
		c.reservations(b.ReservationRefs, p)
		c.recordVector(b.SourceVector, p, r.Header.DomainRef, "Completion.Vector")
		c.id(b.TerminalPolicyRef, "Completion.TerminalPolicy")
		c.require(b.TerminalPolicyRef == p.TerminalPolicyRef, "TERMINAL_POLICY_MISMATCH", "Completion")
		c.ids(b.ProofRefs, c.limits.MaxRefs, "Completion.Proofs", true)
		c.ids(b.Effects, c.limits.MaxRefs, "Completion.Effects", false)
		c.require(len(b.Partitions) > 0 && len(b.Partitions) <= c.limits.MaxPartitions, "INVALID_PARTITION_SET", "Completion")
		c.require(len(b.PhysicalObligations) <= c.limits.MaxObligations && len(b.LineageClosureFacts) <= c.limits.MaxRefs, "EXCESS_OBLIGATIONS", "Completion")
		if len(b.Partitions) <= c.limits.MaxPartitions {
			seen := map[Namespace]bool{}
			for _, part := range b.Partitions {
				c.namespace(part.Namespace, "Completion.Partition")
				c.require(memberResource(r.Header.DomainRef.Resources, part.Namespace.Resource), "PARTITION_RESOURCE_OUTSIDE_DOMAIN", "Completion")
				c.require(!seen[part.Namespace], "DUPLICATE_PARTITION", "Completion")
				seen[part.Namespace] = true
				c.frontier(part.CutFrontier, "Completion.Cut")
				c.frontier(part.FinalAcceptedFrontier, "Completion.Final")
				c.frontier(part.TerminalFrontier, "Completion.Terminal")
				c.require(part.Namespace == part.CutFrontier.Namespace && part.Namespace == part.FinalAcceptedFrontier.Namespace && part.Namespace == part.TerminalFrontier.Namespace, "FRONTIER_NAMESPACE_MISMATCH", "Completion")
				c.registration(part.RegistrationClosed, "Completion.Registration")
				c.require(part.RegistrationClosed.Namespace == part.Namespace && part.RegistrationClosed.MembershipSnapshot == r.Header.DomainRef.SnapshotRef, "REGISTRATION_SNAPSHOT_MISMATCH", "Completion")
				c.ids(part.PendingRegistrationRefs, c.limits.MaxRefs, "Completion.Pending", false)
				c.require(len(part.Holes) <= c.limits.MaxRanges, "EXCESS_HOLES", "Completion")
			}
		}
		if len(b.PhysicalObligations) <= c.limits.MaxObligations {
			seen := map[ID]bool{}
			for _, o := range b.PhysicalObligations {
				c.obligation(o, "Completion.Obligation")
				c.require(memberResource(r.Header.DomainRef.Resources, o.Resource), "OBLIGATION_RESOURCE_OUTSIDE_DOMAIN", "Completion")
				c.require(!seen[o.ObligationID], "DUPLICATE_OBLIGATION", "Completion")
				seen[o.ObligationID] = true
			}
		}
		if len(b.LineageClosureFacts) <= c.limits.MaxRefs {
			for _, reg := range b.LineageClosureFacts {
				c.registration(reg, "Completion.Lineage")
				c.require(reg.MembershipSnapshot == r.Header.DomainRef.SnapshotRef, "LINEAGE_MEMBERSHIP_MISMATCH", "Completion")
			}
		}
		if c.kind == StructurallyCoherent {
			c.take(AssessCompletion(*b, c.limits), "Completion.Assessment")
		}
	}
	if b := r.Seal; b != nil {
		c.id(b.SealID, "Seal.ID")
		c.id(b.ClosureID, "Seal.Closure")
		c.reservations(b.ReservationRefs, p)
		c.recordVector(b.SourceVector, p, r.Header.DomainRef, "Seal.Vector")
		for _, v := range []ID{b.ReleaseRef, b.ConfigRef, b.PolicyRef, b.TrustRef} {
			c.id(v, "Seal.Reference")
		}
		c.literalBinding(b.SourceVector, b.ReleaseAxisKey, ReleaseAxis, b.ReleaseRef, "Seal.Release", p)
		c.literalBinding(b.SourceVector, b.ConfigAxisKey, ConfigAxis, b.ConfigRef, "Seal.Config", p)
		c.literalBinding(b.SourceVector, b.PolicyAxisKey, Policy, b.PolicyRef, "Seal.Policy", p)
		c.literalBinding(b.SourceVector, b.TrustAxisKey, Trust, b.TrustRef, "Seal.Trust", p)
		c.require(len(b.ProofSet) > 0 && len(b.ProofSet) <= c.limits.MaxRefs, "INVALID_PROOF_SET", "Seal")
		if len(b.ProofSet) <= c.limits.MaxRefs {
			seen := map[ID]bool{}
			for _, proof := range b.ProofSet {
				c.id(proof.Ref, "Seal.Proof")
				c.require(!seen[proof.Ref], "DUPLICATE_PROOF", "Seal")
				seen[proof.Ref] = true
				c.require(proof.WorkPhase == "PRIOR", "FUTURE_WORK_IN_SEAL", "Seal")
				c.require(proof.ClosureID == b.ClosureID && proof.MembershipSnapshot == r.Header.DomainRef.SnapshotRef, "PROOF_CLOSURE_MEMBERSHIP_MISMATCH", "Seal")
			}
		}
	}
	if b := r.Fence; b != nil {
		for _, v := range []ID{b.AuthorityConsumptionRef, b.CanonicalRef, b.SealRef, b.OwnershipEpoch, b.FenceEpoch, b.DecisionTimeRef, b.BudgetRef} {
			c.id(v, "Fence.Reference")
		}
		c.resource(b.Target, "Fence.Target")
		c.require(memberResource(r.Header.DomainRef.Resources, b.Target), "FENCE_TARGET_OUTSIDE_DOMAIN", "Fence")
		c.recordVector(b.ExpectedTargetVector, p, r.Header.DomainRef, "Fence.Vector")
		c.literalBinding(b.ExpectedTargetVector, AxisKey{Axis: Ownership, OwnerID: b.Target.OwnerID, Resource: b.Target}, Ownership, b.OwnershipEpoch, "Fence.Ownership", p)
		c.literalBinding(b.ExpectedTargetVector, AxisKey{Axis: Fence, OwnerID: b.Target.OwnerID, Resource: b.Target}, Fence, b.FenceEpoch, "Fence.Epoch", p)
		c.require(b.BackendOperation == "REFERENCE_ONLY", "UNREGISTERED_BACKEND_OPERATION", "Fence")
		c.require(b.DeadlineSemantics == "D1_GUARD_DECISION", "D1_NOT_COMMIT_COMPLETION_DEADLINE", "Fence")
	}
	if b := r.Effect; b != nil {
		c.id(b.EffectID, "Effect.ID")
		c.take(ValidateOptionalRef(b.ParentEffect, c.limits), "Effect.Parent")
		c.resource(b.Target, "Effect.Target")
		c.require(validPurpose(b.Purpose) && member(b.Purpose, p.Purposes...), "EFFECT_PURPOSE_PROFILE_MISMATCH", "Effect")
		c.require(b.ContentRef.Kind == "NONSECRET_METADATA_REF", "RAW_OR_SECRET_CONTENT_FORBIDDEN", "Effect.Content")
		c.id(b.ContentRef.Ref, "Effect.Content.Ref")
		c.recordVector(b.SourceVector, p, r.Header.DomainRef, "Effect.Vector")
		c.require(b.IntentRevision > 0, "MISSING_INTENT_REVISION", "Effect")
		c.resources(b.SinkRefs, "Effect.Sinks", true)
		if len(b.SinkRefs) <= c.limits.MaxMembers {
			for _, sink := range b.SinkRefs {
				c.require(memberResource(r.Header.DomainRef.Resources, sink), "SINK_RESOURCE_OUTSIDE_DOMAIN", "Effect.Sinks")
			}
		}
		c.ids(b.ProjectionRefs, c.limits.MaxRefs, "Effect.ProjectionRefs", false)
		c.require(memberResource(r.Header.DomainRef.Resources, b.Target), "EFFECT_TARGET_OUTSIDE_DOMAIN", "Effect")
	}
	if b := r.Inspection; b != nil {
		q := b.Target
		c.correlation(q.Correlation, "Inspect.Correlation")
		c.require(q.Correlation == r.Header.Correlation && q.ExecutionKey == q.Correlation.ExecutionKey, "INSPECT_IMMUTABLE_CORRELATION_MISMATCH", "Inspect")
		for _, v := range []ID{q.BindingRef, q.ExecutionKey, q.TargetRecordRef, q.Scope} {
			c.id(v, "Inspect.Target")
		}
		c.require(q.Purpose == PurposeLookup, "INSPECT_PURPOSE_MISMATCH", "Inspect")
		c.take(ValidateObservationSet(b.Observations, p), "Inspect.Observations")
		c.ids(b.ObservedScopes, c.limits.MaxRefs, "Inspect.Scopes", true)
		c.take(ValidateFact(b.Coverage, c.limits), "Inspect.Coverage")
		c.ids(b.HistoricalRefs, c.limits.MaxRefs, "Inspect.History", false)
		c.require(len(b.SourceFrames) > 0 && len(b.SourceFrames) <= c.limits.MaxRefs, "INVALID_SOURCE_FRAMES", "Inspect")
		if len(b.SourceFrames) <= c.limits.MaxRefs {
			c.inspectionFrames(*b)
		}
	}
	if b := r.Reconcile; b != nil {
		c.id(b.PlanID, "Reconcile.ID")
		c.require(b.OriginalCorrelation == r.Header.Correlation && b.Purpose == PurposeRepair, "RECORD_REPAIR_PURPOSE_OR_TUPLE_MISMATCH", "Reconcile")
		c.ids(b.SourceReceiptRefs, c.limits.MaxRefs, "Reconcile.Receipts", true)
		c.ids(b.SourceWitnessRefs, c.limits.MaxRefs, "Reconcile.Witnesses", true)
		c.recordVector(b.CurrentSourceVector, p, r.Header.DomainRef, "Reconcile.Vector")
		c.resource(b.ExpectedProjection, "Reconcile.Target")
		c.require(b.ExpectedProjection.Kind == Projection && memberResource(r.Header.DomainRef.Resources, b.ExpectedProjection), "RECONCILE_PROJECTION_TARGET_MISMATCH", "Reconcile")
		c.require(b.PlanRevision > 0 && b.ExpectedProjectionRevision > 0, "EXPLICIT_CAS_REVISION_REQUIRED", "Reconcile")
		c.id(b.RequiredRecordCapability, "Reconcile.RecordCapability")
		c.require(b.IntendedMetadataDelta.Kind == "FIXED_RECEIPT_REFERENCE", "BUSINESS_ACTION_IN_RECORD_REPAIR", "Reconcile")
		c.id(b.IntendedMetadataDelta.SourceReceiptRef, "Reconcile.Delta.Receipt")
		c.id(b.IntendedMetadataDelta.MetadataRef, "Reconcile.Delta.Metadata")
		c.require(member(b.IntendedMetadataDelta.SourceReceiptRef, b.SourceReceiptRefs...), "DELTA_RECEIPT_MISMATCH", "Reconcile")
		c.take(ValidateFact(b.Freshness, c.limits), "Reconcile.Freshness")
	}
	return c.finish()
}

// CompareImmutable compares validated typed contents as bounded sets, preserving all
// literal values. A record revision change is an observation/CAS change, not a
// license to replace immutable content.
func CompareImmutable(a, b Record, p ComparisonProfile) Result {
	if r := ValidateRecord(a, p); r.Kind != StructurallyCoherent {
		return r
	}
	if r := ValidateRecord(b, p); r.Kind != StructurallyCoherent {
		return r
	}
	if a.Header.RecordKey != b.Header.RecordKey {
		return result(Conflict, "RECORD_KEY_CHANGED", "Record")
	}
	a.Header.RecordRevision = 0
	b.Header.RecordRevision = 0
	if !structuralEqual(reflect.ValueOf(a), reflect.ValueOf(b)) {
		return result(Conflict, "IMMUTABLE_CONTENT_CHANGED", "Record")
	}
	return coherent()
}
func structuralEqual(a, b reflect.Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return structuralEqual(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if !structuralEqual(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if a.Len() != b.Len() {
			return false
		}
		used := make([]bool, b.Len())
		for i := 0; i < a.Len(); i++ {
			found := false
			for j := 0; j < b.Len(); j++ {
				if !used[j] && structuralEqual(a.Index(i), b.Index(j)) {
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	case reflect.String:
		return a.String() == b.String()
	case reflect.Uint64:
		return a.Uint() == b.Uint()
	case reflect.Int:
		return a.Int() == b.Int()
	case reflect.Bool:
		return a.Bool() == b.Bool()
	}
	return false
}
func ValidateSealAgainstCompletion(seal, completion Record, p ComparisonProfile) Result {
	if r := ValidateRecord(seal, p); r.Kind != StructurallyCoherent {
		return r
	}
	if r := ValidateRecord(completion, p); r.Kind != StructurallyCoherent {
		return r
	}
	if seal.Seal == nil || completion.Completion == nil {
		return result(RefusedUnchanged, "SEAL_COMPLETION_PAYLOAD_REQUIRED", "Seal")
	}
	s, c := seal.Seal, completion.Completion
	if seal.Header.Correlation != completion.Header.Correlation || seal.Header.RecordKey.OwnerID != completion.Header.RecordKey.OwnerID || seal.Header.ContractRef.ModuleCommitRef != completion.Header.ContractRef.ModuleCommitRef || s.ClosureID != c.ClosureID || !structuralEqual(reflect.ValueOf(s.ReservationRefs), reflect.ValueOf(c.ReservationRefs)) || !structuralEqual(reflect.ValueOf(seal.Header.DomainRef), reflect.ValueOf(completion.Header.DomainRef)) {
		return result(Conflict, "SEAL_CLOSURE_MISMATCH", "Seal")
	}
	if r := CompareVectors(s.SourceVector, c.SourceVector, p); r.Kind != StructurallyCoherent {
		return r
	}
	for _, proof := range s.ProofSet {
		if !member(proof.Ref, c.ProofRefs...) {
			return result(Conflict, "SEAL_PROOF_NOT_IN_COMPLETION", "Seal")
		}
	}
	return coherent()
}

func (c *checker) literalBinding(v RevisionVector, key AxisKey, axis Axis, want ID, path string, p ComparisonProfile) {
	c.axisKey(key, path+".Key")
	c.require(key.Axis == axis, "LITERAL_BINDING_AXIS_MISMATCH", path)
	allowed := false
	for _, rule := range p.AxisRules {
		if rule.Key == key {
			allowed = true
		}
	}
	if !allowed {
		c.add(RefusedUnchanged, "LITERAL_BINDING_OUTSIDE_PROFILE", path)
		return
	}
	if len(v) > c.limits.MaxAxes {
		return
	}
	found := false
	for _, e := range v {
		if e.Key == key {
			found = true
			if e.Value.Kind == ValueUnavailable {
				c.add(NeedsEvidence, "BOUND_LITERAL_UNAVAILABLE", path)
			} else {
				c.require(e.Value.Kind == PresentLiteral && e.Value.Literal != nil && *e.Value.Literal == want, "DUPLICATE_LITERAL_BINDING_MISMATCH", path)
			}
		}
	}
	if !found {
		c.add(NeedsEvidence, "BOUND_LITERAL_AXIS_MISSING", path)
	}
}
func (c *checker) inspectionFrames(b InspectionRecord) {
	scope := b.Target.Scope
	if len(b.ObservedScopes) > c.limits.MaxRefs || len(b.Observations.Observations) > c.limits.MaxRefs || len(b.Observations.PhysicalObligations) > c.limits.MaxObligations {
		return
	}
	c.require(member(scope, b.ObservedScopes...), "QUERY_SCOPE_NOT_OBSERVED", "Inspect.Scope")
	frames := map[ID]SourceFrame{}
	for _, f := range b.SourceFrames {
		c.frame(f, "Inspect.Frame")
		_, dup := frames[f.FrameRef]
		c.require(!dup, "DUPLICATE_FRAME", "Inspect")
		frames[f.FrameRef] = f
		c.require(member(f.Scope, b.ObservedScopes...), "FRAME_SCOPE_NOT_OBSERVED", "Inspect.Frame")
	}
	check := func(o *ObservationContext, path string) {
		if o == nil {
			return
		}
		f, ok := frames[o.FrameRef]
		if !ok {
			c.add(NeedsEvidence, "OBSERVED_FRAME_MISSING", path)
			return
		}
		c.require(f.OwnerID == o.OwnerID && f.Scope == o.Scope && f.SourceRevision == o.SourceRevision && o.Scope == scope, "OBSERVED_FRAME_SCOPE_OWNER_REVISION_MISMATCH", path)
	}
	check(b.Coverage.Observation, "Inspect.Coverage")
	for _, o := range b.Observations.Observations {
		check(o.Outcome.Observation, "Inspect.Outcome")
	}
	for _, o := range b.Observations.PhysicalObligations {
		check(o.Terminal.Observation, "Inspect.Physical")
	}
}

func (c *checker) recordVector(v RevisionVector, p ComparisonProfile, d DomainRef, path string) {
	c.take(ValidateVector(v, p), path)
	if len(v) > c.limits.MaxAxes || len(d.Resources) > c.limits.MaxMembers {
		return
	}
	for _, entry := range v {
		c.require(memberResource(d.Resources, entry.Key.Resource), "VECTOR_RESOURCE_OUTSIDE_DOMAIN", path)
	}
}
