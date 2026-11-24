package mediarelease

import (
	"reflect"
	"slices"
	"strconv"
	"time"
)

// ConsumptionView is supplied by a trusted product adapter, never by a request.
// Both uniqueness indexes, the prior record and slot revision must come from
// one consistent ledger snapshot. A missing row alone cannot establish this.
// Unit is the independently registered, indivisible physical opening unit.
type ConsumptionView struct {
	AuthenticatedOwner                                                           Owner
	Predecessor, Freshness, Authority, Revocation, StopFence, Lease, Reservation Ref
	WorkRegistration, Dispatch                                                   Ref
	WorkerAssignment, EncoderAssignment                                          Assignment
	WorkerGeneration                                                             string
	Unit                                                                         Scope
	SlotRevision, Now, ClockErrorNS                                              string
	ExistingClaims                                                               []Record
	LiveContinuation, LedgerComplete, Current, ClosuresVerified, UnitRegistered  bool
	Revoked, Stopped, Restarted                                                  bool
	Outcome                                                                      OutcomeProfile
	// ClockAuthorities attest the comparison bounds for each clock used by
	// the chain. Now and ClockErrorNS cover all of these conversions.
	ClockAuthorities []Ref
	GraphOutlets     GraphOutlets
}

// GraphOutlets carries the independently resolved configuration inventory.
// A Worker input closure alone cannot account for the public graph outputs.
type GraphOutlets struct {
	ConfigRef       Ref
	RosterRef       Ref
	Egress          []Scope
	IsolatedSinkRef *Ref
	TargetClosure   *Closure
}

func verifiedOutlets(cover, permit Identity, v GraphOutlets) bool {
	if v.ConfigRef != *cover.GraphConfigRef || validateValue(v.RosterRef) != nil ||
		v.RosterRef.SourceOwner != cover.Issuer.Peer() || v.RosterRef == v.ConfigRef || len(v.Egress) > 64 {
		return false
	}
	if len(v.Egress) == 0 {
		if v.IsolatedSinkRef == nil || *v.IsolatedSinkRef != *cover.QualifiedSinkRef || *v.IsolatedSinkRef == v.RosterRef {
			return false
		}
	} else {
		if v.IsolatedSinkRef != nil {
			return false
		}
		for _, out := range v.Egress {
			if out.Route != "ENCODER_DIRECT" && out.Route != "RELAY_UPSTREAM" {
				return false
			}
			if !slices.ContainsFunc(cover.Closures, func(c Closure) bool { return reflect.DeepEqual(out, c.Scope) }) {
				return false
			}
		}
	}
	// Reject an incomplete roster even if it still names one valid exit.
	for _, c := range cover.Closures {
		if c.Scope.Route == "ENCODER_DIRECT" || c.Scope.Route == "RELAY_UPSTREAM" {
			if !slices.ContainsFunc(v.Egress, func(out Scope) bool { return reflect.DeepEqual(out, c.Scope) }) {
				return false
			}
		}
	}
	if slices.ContainsFunc(cover.Closures, func(c Closure) bool { return reflect.DeepEqual(c.Scope, *permit.Scope) }) {
		return true
	}
	cl := v.TargetClosure
	return cl != nil && validateValue(*cl) == nil && cl.GraphID == cover.GraphID && cl.GraphGeneration == cover.GraphGeneration &&
		reflect.DeepEqual(cl.Scope, *permit.Scope) && slices.Contains(permit.ClosureRefs, cl.FenceRef) &&
		cl.FenceRef.SourceOwner == permit.Scope.TargetOwner.Peer() && cl.IntervalStart <= cover.ObservedAt && cl.ObservedThrough >= cover.ObservedAt
}

// TerminalCriterion describes one independently verified frame and attributed
// observation under the exact success profile. Frame payloads stay at the
// source. In particular a caller must verify byte coverage, not just counts,
// when resolving a complete-frame SRT receipt criterion.
type TerminalCriterion struct {
	Media, Fact string
	Observer    Owner
	Frame       Ref
}

type OutcomeProfile struct {
	Reference   Ref
	Scope       Scope
	EffectState string
	Criteria    []TerminalCriterion
	Verified    bool
}

// AssessOutcome compares positive evidence for the declared boundary and each
// required track. A failure does not establish that no bytes escaped.
func AssessOutcome(scope Scope, observations []Obs, state string, profile OutcomeProfile) Decision {
	if state == "EFFECT_UNKNOWN" {
		return Unknown
	}
	if state != "RELEASE_CONFIRMED" && state != "RELEASE_FAILED" {
		return Conflict
	}
	if !profile.Verified || profile.Reference != scope.SuccessProfileRef || profile.EffectState != state ||
		!reflect.DeepEqual(profile.Scope, scope) || validateValue(scope) != nil ||
		len(profile.Criteria) == 0 || len(profile.Criteria) > 64 || len(observations) == 0 || len(observations) > 64 {
		return Unknown
	}
	for _, o := range observations {
		if validateValue(o) != nil || !physicalScopeSubset(o.Scope, scope) {
			return Conflict
		}
	}
	covered := make(map[string]bool)
	for _, criterion := range profile.Criteria {
		if !slices.Contains(scope.Media, criterion.Media) || !validID(criterion.Fact) || validateValue(criterion.Observer) != nil ||
			validateValue(criterion.Frame) != nil || criterion.Frame.SourceOwner != criterion.Observer.Peer() {
			return Conflict
		}
		found := false
		for _, o := range observations {
			if o.Knowledge == "KNOWN" && o.Fact == criterion.Fact && o.Observer == criterion.Observer &&
				o.FrameRef != nil && *o.FrameRef == criterion.Frame && slices.Contains(o.Scope.Media, criterion.Media) {
				found = true
				break
			}
		}
		if !found {
			return Unknown
		}
		covered[criterion.Media] = true
	}
	if len(covered) != len(scope.Media) {
		return Unknown
	}
	return Coherent
}

func timeInWindow(window Window, now, errorNS string) bool {
	if !validTimestamp(now) || !validUint(errorNS, false) {
		return false
	}
	errNS, err := strconv.ParseInt(errorNS, 10, 64)
	if err != nil {
		return false
	}
	maxNS, err := strconv.ParseUint(window.MaxClockErrorNS, 10, 64)
	if err != nil || uint64(errNS) > maxNS {
		return false
	}
	const format = "2006-01-02T15:04:05.000000000Z"
	t, err := time.Parse(format, now)
	if err != nil {
		return false
	}
	start, e1 := time.Parse(format, window.NotBefore)
	end, e2 := time.Parse(format, window.ExpiresAt)
	return e1 == nil && e2 == nil && !t.Add(-time.Duration(errNS)).Before(start) && t.Add(time.Duration(errNS)).Before(end)
}

// sameSlot excludes replaceable generation, owner, operation and permit IDs.
// Media aliases and overlapping track subsets cannot split one physical unit.
func sameSlot(a, b Identity) bool {
	return a.Domain == b.Domain && a.Original.AttemptID == b.Original.AttemptID &&
		a.Original.StreamID == b.Original.StreamID && a.Original.RunID == b.Original.RunID &&
		a.Scope.Route == b.Scope.Route && a.Scope.DestinationID == b.Scope.DestinationID &&
		a.Scope.EffectKind == b.Scope.EffectKind &&
		slices.ContainsFunc(a.Scope.Media, func(track string) bool { return slices.Contains(b.Scope.Media, track) })
}

func compareClaims(c Record, state string, claims []Record) Decision {
	var matches int
	for _, claimed := range claims {
		if ValidateRecord(claimed) != nil || claimed.Identity.Kind != MediaReleasePermit {
			return Conflict
		}
		a, b := c.Identity, claimed.Identity
		if a.Domain != b.Domain {
			continue
		}
		if a.PermitID != b.PermitID && a.RecordKey != b.RecordKey && a.LogicalEffectKey != b.LogicalEffectKey && !sameSlot(a, b) {
			continue
		}
		if c.RecordDigest != claimed.RecordDigest {
			return Conflict
		}
		matches++
	}
	switch {
	case matches > 1:
		return Conflict
	case state == "NOT_CONSUMED" && matches == 1:
		return Duplicate
	case state == "CONSUMPTION_PENDING" && matches != 1:
		return Unknown
	default:
		return Coherent
	}
}

func currentInputs(proofs []Record, previous, next Record, v ConsumptionView) bool {
	cp, worker, c := proofs[0].Identity, proofs[1].Identity, proofs[3].Identity
	s := next.Snapshot
	if !v.Current || !v.ClosuresVerified || !v.UnitRegistered || !v.LiveContinuation || v.Restarted || v.Stopped || v.Revoked ||
		previous.Snapshot.StopObservationRef != nil || previous.Snapshot.RevokeObservationRef != nil ||
		s.StopObservationRef != nil || s.RevokeObservationRef != nil ||
		v.AuthenticatedOwner != c.Scope.TargetOwner || !reflect.DeepEqual(v.Unit, *c.Scope) ||
		v.WorkerAssignment != *cp.WorkerAssignment || v.EncoderAssignment != *cp.EncoderAssignment ||
		v.WorkerGeneration != worker.WorkerGeneration || v.Reservation != *cp.ReservationRef ||
		v.Authority != *c.AuthorityRef || v.Revocation != *c.RevocationRef || v.StopFence != *c.StopFenceRef || v.Lease != *c.LeaseRef ||
		s.DispatchRef == nil || *s.DispatchRef != v.Dispatch || s.WorkRegistrationRef == nil || *s.WorkRegistrationRef != v.WorkRegistration ||
		s.CheckedAt != v.Now || !verifiedOutlets(proofs[2].Identity, c, v.GraphOutlets) {
		return false
	}
	for _, required := range []Ref{v.Authority, v.Revocation, v.StopFence, v.Lease, v.Reservation,
		v.WorkerAssignment.AssignmentRef, v.EncoderAssignment.AssignmentRef} {
		if !slices.Contains(s.AuthorizationRefs, required) {
			return false
		}
	}
	for _, r := range proofs {
		if !slices.Contains(v.ClockAuthorities, r.Identity.Window.ClockAuthorityRef) || !timeInWindow(*r.Identity.Window, v.Now, v.ClockErrorNS) {
			return false
		}
	}
	return true
}

// ReduceConsumption compares an observed transition and its two exclusive
// keys. The return value is never a transferable continuation or an instruction
// to perform I/O. Durable reservation/commit/readback must be enforced together
// at the specified product owner.
func ReduceConsumption(proofs []Record, previous, next Record, v ConsumptionView) Decision {
	if len(proofs) != 4 || ValidateHandoff(proofs...) != nil {
		return Conflict
	}
	c := proofs[3]
	for _, d := range []Record{previous, next} {
		if ValidateRecord(d) != nil || d.Identity.Kind != MediaReleaseConsumption ||
			d.Identity.Domain != c.Identity.Domain || d.Identity.Original != c.Identity.Original || !consumptionMatches(c, d) {
			return Conflict
		}
	}
	step := CompareRevision(previous, next)
	if step == Conflict {
		return step
	}
	if !v.LedgerComplete || validateValue(v.Freshness) != nil || v.Freshness != *previous.Snapshot.FreshnessRef {
		return Unknown
	}
	if v.Predecessor != referenceOf(previous) || v.SlotRevision != previous.Snapshot.SlotRevision {
		return Conflict
	}
	if step == Duplicate {
		return Duplicate
	}
	a, b := previous.Snapshot, next.Snapshot
	if a.EffectState != b.EffectState && b.EffectState != "EFFECT_UNKNOWN" {
		if result := AssessOutcome(*c.Identity.Scope, b.Observations, b.EffectState, v.Outcome); result != Coherent {
			return result
		}
	}
	if a.ConsumptionState == b.ConsumptionState {
		// Appending observations never needs a renewed permit. A process restart
		// cannot clear a stop or use a historical append to re-open the slot.
		if v.AuthenticatedOwner != next.Identity.Issuer {
			return Held
		}
		return Coherent
	}
	if b.EffectState != "EFFECT_UNKNOWN" || b.FirstEffectRef != nil || b.ReceiptRef != nil {
		return Conflict
	}
	if a.ConsumptionState == "NOT_CONSUMED" && v.SlotRevision != c.Identity.ExpectedSlotRevision ||
		a.ConsumptionState == "CONSUMPTION_PENDING" && !successor(c.Identity.ExpectedSlotRevision, v.SlotRevision) {
		return Conflict
	}
	if !currentInputs(proofs, previous, next, v) {
		return Held
	}
	return compareClaims(c, a.ConsumptionState, v.ExistingClaims)
}

type InspectQuery = InspectRequest

// InspectOriginal is pure. A missing or rolled-back record cannot be inferred
// to be unused. A found historical receipt is readable after expiry/revocation.
func InspectOriginal(q InspectQuery, reply InspectResult, record *Record) Decision {
	if ValidateInspectResult(reply) != nil || ValidateInspectRequest(q) != nil ||
		q.LookupKey != reply.LookupKey || q.Original != reply.Original || q.Owner != reply.Responder {
		return Conflict
	}
	switch reply.Status {
	case "CONFLICT":
		return Conflict
	case "UNKNOWN":
		return Unknown
	case "FOUND":
		if record == nil {
			return Unknown
		}
		if ValidateRecord(*record) != nil || !exact(reply.FoundRef, *record) || record.Identity.Original != q.Original ||
			record.Identity.RecordKey != q.LookupKey || record.Identity.Issuer != q.Owner ||
			q.ExpectedRevision != "" && record.Snapshot.Revision != q.ExpectedRevision ||
			q.ExpectedDigest != "" && record.RecordDigest != q.ExpectedDigest {
			return Conflict
		}
		if reply.Currentness == "UNKNOWN" {
			return Unknown
		}
		return Coherent
	default:
		return Conflict
	}
}
