package lifecycle

// Profiles are explicit inputs. Logical profile IDs do not imply production qualification.
func ValidateProfile(p ComparisonProfile) Result {
	if r := ValidateLimits(p.Limits); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(p.Limits)
	c.require(validProfile(p.Ref.ProfileID) && p.Ref.Revision == LogicalRevision, "UNKNOWN_PROFILE_REVISION", "Profile")
	c.id(p.TerminalPolicyRef, "Profile.TerminalPolicy")
	c.id(p.LimitsRef, "Profile.LimitsRef")
	c.require(len(p.AxisRules) > 0 && len(p.AxisRules) <= p.Limits.MaxAxes, "MISSING_OR_EXCESS_AXES", "Profile.AxisRules")
	c.require(len(p.ResourceKinds) > 0 && len(p.ResourceKinds) <= p.Limits.MaxMembers, "MISSING_RESOURCE_KINDS", "Profile.ResourceKinds")
	if len(p.AxisRules) > p.Limits.MaxAxes || len(p.ResourceKinds) > p.Limits.MaxMembers {
		return c.finish()
	}
	resources := map[ResourceKind]bool{}
	for _, k := range p.ResourceKinds {
		c.require(validResource(k) && !resources[k], "INVALID_RESOURCE_KIND_SET", "Profile.ResourceKinds")
		resources[k] = true
	}
	c.require(len(p.Purposes) > 0 && len(p.Purposes) <= p.Limits.MaxRefs, "MISSING_PURPOSES", "Profile.Purposes")
	if len(p.Purposes) > p.Limits.MaxRefs {
		return c.finish()
	}
	purposes := map[Purpose]bool{}
	for _, v := range p.Purposes {
		c.require(validPurpose(v) && !purposes[v], "INVALID_PURPOSE_SET", "Profile.Purposes")
		purposes[v] = true
	}
	expectedPurpose := map[ProfileID]Purpose{ApprovalLocal: PurposeApproval, MediaIngress: PurposeIngress, ClosureAssess: PurposeClosure, StopLookup: PurposeLookup, EffectOutcome: PurposeEffect, RecordRepair: PurposeRepair, ExplicitResume: PurposeResume}
	c.require(purposes[expectedPurpose[p.Ref.ProfileID]], "PROFILE_PURPOSE_MISMATCH", "Profile.Purposes")
	seen := map[AxisKey]bool{}
	for _, a := range p.AxisRules {
		c.axisKey(a.Key, "Profile.AxisRules")
		c.require(!seen[a.Key], "DUPLICATE_AXIS_RULE", "Profile.AxisRules")
		seen[a.Key] = true
		c.require(validComponent(a.OwnerRole) && resources[a.Key.Resource.Kind], "INVALID_AXIS_ROLE_RESOURCE", "Profile.AxisRules")
		c.require(member(a.ValueKind, PresentUint64, PresentLiteral) && a.Comparison == "EXACT", "UNKNOWN_AXIS_COMPARISON", "Profile.AxisRules")
		c.require(a.ValueKind == axisValueKind(a.Key.Axis), "AXIS_VALUE_KIND_MISMATCH", "Profile.AxisRules")
	}
	for _, axis := range requiredProfileAxes(p.Ref.ProfileID) {
		found := false
		for _, a := range p.AxisRules {
			if a.Required && a.Key.Axis == axis {
				found = true
			}
		}
		c.require(found, "MISSING_LOGICAL_PROFILE_AXIS", "Profile.AxisRules")
	}
	c.require(len(p.RequiredBoundaries) > 0 && len(p.RequiredBoundaries) <= p.Limits.MaxRefs, "MISSING_BOUNDARIES", "Profile.RequiredBoundaries")
	if len(p.RequiredBoundaries) > p.Limits.MaxRefs {
		return c.finish()
	}
	bounds := map[BoundaryAddress]bool{}
	for _, b := range p.RequiredBoundaries {
		c.take(ValidateBoundary(b, p.Limits), "Profile.Boundaries")
		c.require(!bounds[b], "DUPLICATE_BOUNDARY", "Profile.Boundaries")
		bounds[b] = true
	}
	if c.kind != StructurallyCoherent {
		if c.kind == RefusedUnchanged {
			c.kind = NeedsDecision
		}
		return c.finish()
	}
	return coherent()
}
func axisValueKind(a Axis) ValueKind {
	if member(a, ARevision, CEpoch, Identity, Membership, JobAxis, ConnectionAxis, Writer, Cover, Effect) {
		return PresentUint64
	}
	return PresentLiteral
}
func ValidateVector(v RevisionVector, p ComparisonProfile) Result {
	if r := ValidateProfile(p); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(p.Limits)
	c.require(len(v) > 0 && len(v) <= p.Limits.MaxAxes, "INVALID_VECTOR_SIZE", "Vector")
	if len(v) > p.Limits.MaxAxes {
		return c.finish()
	}
	rules := map[AxisKey]AxisRule{}
	for _, a := range p.AxisRules {
		rules[a.Key] = a
	}
	seen := map[AxisKey]bool{}
	for _, e := range v {
		c.axisKey(e.Key, "Vector.Entry")
		c.id(e.Incarnation, "Vector.Incarnation")
		c.require(!seen[e.Key], "DUPLICATE_AXIS_KEY", "Vector")
		seen[e.Key] = true
		rule, ok := rules[e.Key]
		c.require(ok, "AXIS_OUTSIDE_PROFILE", "Vector")
		c.value(e, "Vector.Value")
		if ok && e.Value.Kind != ValueUnavailable {
			c.require(rule.ValueKind == e.Value.Kind, "VALUE_KIND_PROFILE_MISMATCH", "Vector")
		}
	}
	for _, a := range p.AxisRules {
		if a.Required && !seen[a.Key] {
			c.add(NeedsEvidence, "MISSING_REQUIRED_AXIS", "Vector")
		}
	}
	return c.finish()
}
func sameValue(a, b VersionedValue) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case PresentUint64:
		return a.Uint64 != nil && b.Uint64 != nil && *a.Uint64 == *b.Uint64
	case PresentLiteral:
		return a.Literal != nil && b.Literal != nil && *a.Literal == *b.Literal
	}
	return false
}
func CompareVectors(expected, current RevisionVector, p ComparisonProfile) Result {
	if r := ValidateVector(expected, p); r.Kind != StructurallyCoherent {
		return r
	}
	if r := ValidateVector(current, p); r.Kind != StructurallyCoherent {
		return r
	}
	if len(expected) != len(current) {
		return result(Stale, "VECTOR_KEY_SET_CHANGED", "Vector")
	}
	for _, e := range expected {
		found := false
		for _, v := range current {
			if e.Key == v.Key {
				found = true
				if e.Incarnation != v.Incarnation || !sameValue(e.Value, v.Value) {
					return result(Stale, "VECTOR_VALUE_OR_INCARNATION_CHANGED", "Vector")
				}
			}
		}
		if !found {
			return result(Stale, "VECTOR_KEY_CHANGED", "Vector")
		}
	}
	return coherent()
}

func requiredProfileAxes(id ProfileID) []Axis {
	switch id {
	case ApprovalLocal:
		return []Axis{ARevision, CEpoch, Identity, Policy, Trust}
	case MediaIngress:
		return []Axis{Membership, Admission, Boot, Process, RunAxis, JobAxis, ConnectionAxis, Writer, Cover}
	case ClosureAssess:
		return []Axis{Membership, Admission, Boot, Process, Writer, Policy, Trust, ReleaseAxis, ConfigAxis}
	case StopLookup:
		return []Axis{RunAxis}
	case EffectOutcome:
		return []Axis{Membership, Effect, Policy, Trust}
	case RecordRepair:
		return []Axis{Membership}
	case ExplicitResume:
		return []Axis{Membership, Admission, Ownership, Fence, Boot}
	}
	return nil
}
