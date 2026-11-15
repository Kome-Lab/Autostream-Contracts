package lifecycle

// ValidateBoundaryObservation checks one boundary claim without selecting a
// comparison profile. Required-boundary coverage is checked by ValidateObservationSet.
func ValidateBoundaryObservation(o BoundaryObservation, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	if o.DetailRef != "" {
		c.id(o.DetailRef, "Observation.DetailRef")
	}
	c.take(ValidateBoundary(o.Address, l), "Observation.Address")
	c.require(validKnowledge(o.Knowledge) && member(o.Presence, Present, Absent, NotObserved), "UNKNOWN_OBSERVATION_ENUM", "Observation")
	c.take(ValidateFact(o.Outcome, l), "Observation.Outcome")
	if o.Outcome.Observation != nil {
		c.require(o.Outcome.Observation.OwnerID == o.Address.OwnerID, "OUTCOME_OWNER_MISMATCH", "Observation")
	}
	switch o.Knowledge {
	case Known:
		c.require(o.Outcome.State == Observed && o.Outcome.Value != nil && o.Presence != NotObserved, "KNOWN_WITHOUT_OBSERVATION", "Observation")
		if o.Outcome.Value != nil {
			c.require(*o.Outcome.Value != OutcomeUnknown, "KNOWN_WITH_UNKNOWN_OUTCOME", "Observation")
			if *o.Outcome.Value == Pending {
				c.add(PendingObligation, "BOUNDARY_PENDING", "Observation")
			}
			if *o.Outcome.Value == NoEffectProven {
				c.id(o.DetailRef, "Observation.NoEffectWitness")
			}
		}
	case KnowledgeUnknown:
		c.require(o.Outcome.State == Unavailable || o.Outcome.Value != nil && *o.Outcome.Value == OutcomeUnknown, "UNKNOWN_WITH_KNOWN_OUTCOME", "Observation")
		c.add(Unknown, "BOUNDARY_UNKNOWN", "Observation")
	case KnowledgeConflict:
		c.add(Conflict, "BOUNDARY_CONFLICT", "Observation")
	}
	if o.Outcome.State == Observed {
		c.observationDetails(o, nil)
	}
	return c.finish()
}
func ValidateObservationSet(s ObservationSet, p ComparisonProfile) Result {
	if r := ValidateProfile(p); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(p.Limits)
	c.require(validGate(s.Gate), "UNKNOWN_GATE", "Observations")
	c.require(len(s.Observations) <= p.Limits.MaxRefs && len(s.PhysicalObligations) <= p.Limits.MaxObligations && len(s.PhaseHints) <= p.Limits.MaxRefs, "EXCESS_OBSERVATIONS", "Observations")
	if len(s.Observations) > p.Limits.MaxRefs || len(s.PhysicalObligations) > p.Limits.MaxObligations || len(s.PhaseHints) > p.Limits.MaxRefs {
		return c.finish()
	}
	seen := map[BoundaryAddress]bool{}
	for _, o := range s.Observations {
		c.take(ValidateBoundaryObservation(o, p.Limits), "Observation")
		c.require(!seen[o.Address], "DUPLICATE_BOUNDARY_OBSERVATION", "Observations")
		seen[o.Address] = true
		if o.Outcome.State == Observed {
			c.observationDetails(o, &p.Ref)
		}
	}
	for _, b := range p.RequiredBoundaries {
		if !seen[b] {
			c.add(NeedsEvidence, "REQUIRED_BOUNDARY_NOT_OBSERVED", "Observations")
		}
	}
	obligations := map[ID]bool{}
	for _, o := range s.PhysicalObligations {
		c.obligation(o, "Observations.Physical")
		c.require(!obligations[o.ObligationID], "DUPLICATE_OBLIGATION", "Observations")
		obligations[o.ObligationID] = true
	}
	c.ids(s.RemainingObligations, p.Limits.MaxObligations, "Observations.Remaining", false)
	c.ids(s.Contradictions, p.Limits.MaxRefs, "Observations.Contradictions", false)
	if len(s.RemainingObligations) > 0 {
		c.add(PendingObligation, "REMAINING_OBLIGATIONS", "Observations")
	}
	if len(s.Contradictions) > 0 {
		c.add(Conflict, "CONTRADICTORY_OBSERVATIONS", "Observations")
	}
	for _, b := range s.PhaseHints {
		c.take(ValidateBoundary(b, c.limits), "Observations.PhaseHint")
	}
	if s.Gate == Held {
		c.add(PendingObligation, "HELD_REMAINS_HELD", "Observations")
	}
	if s.Gate == NotQualified {
		c.add(NeedsEvidence, "PROFILE_NOT_QUALIFIED", "Observations")
	}
	return c.finish()
}
func (c *checker) observationDetails(o BoundaryObservation, expected *ProfileRef) {
	sub := o.Address.SubBoundary
	needRaw := sub == RawWrite
	needDurability := sub == Durability
	needQueue := member(sub, QueueAdmission, QueueTerminal)
	if !needRaw && !needDurability && !needQueue {
		c.require(o.Details == nil, "DETAILS_OUTSIDE_BOUNDARY", "Observation")
		return
	}
	if o.Details == nil {
		c.add(NeedsEvidence, "TYPED_BOUNDARY_DETAIL_REQUIRED", "Observation")
		return
	}
	d := o.Details
	c.require((d.RawExtent != nil) == needRaw && (d.Durability != nil) == needDurability && (d.Queue != nil) == needQueue, "BOUNDARY_DETAIL_UNION_MISMATCH", "Observation")
	if v := d.RawExtent; v != nil {
		c.id(v.ExtentRef, "Observation.RawExtent")
		c.require(v.Offset != nil && v.Length != nil && v.RequestedLength != nil, "MISSING_WRITE_EXTENT", "Observation")
		if v.Offset != nil && v.Length != nil && v.RequestedLength != nil {
			c.require(*v.Length <= *v.RequestedLength && *v.Offset <= ^uint64(0)-*v.Length, "INVALID_WRITE_EXTENT", "Observation")
		}
	}
	if v := d.Durability; v != nil {
		c.require(validProfile(v.ProfileRef.ProfileID) && v.ProfileRef.Revision == 1 && word(v.Level, "PROCESS_BUFFERED SYNC_OBSERVED OWNER_TERMINAL"), "DURABILITY_PROFILE_REQUIRED", "Observation")
		if expected != nil {
			c.require(v.ProfileRef == *expected, "DURABILITY_PROFILE_MISMATCH", "Observation")
		}
		c.id(v.WitnessRef, "Observation.DurabilityWitness")
	}
	if v := d.Queue; v != nil {
		c.namespace(v.Work.Namespace, "Observation.Queue.Namespace")
		c.id(v.Work.WorkID, "Observation.Queue.Work")
		c.id(v.Work.AdmissionRecordRef, "Observation.Queue.Admission")
		if v.Work.ParentWorkID != "" {
			c.id(v.Work.ParentWorkID, "Observation.Queue.Parent")
		}
		c.require(v.Work.WorkID != v.Work.ParentWorkID, "SELF_PARENT_WORK", "Observation.Queue")
		c.require(v.Work.Sequence > 0, "ZERO_WORK_SEQUENCE", "Observation.Queue")
		c.id(v.ReceiptRef, "Observation.Queue.Receipt")
	}
}
