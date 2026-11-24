package mediarelease

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
)

// Decision describes only a proposed record or a read-only observation.
type Decision string

const (
	Coherent  Decision = "STRUCTURALLY_COHERENT"
	Held      Decision = "HELD"
	Unknown   Decision = "UNKNOWN"
	Conflict  Decision = "CONFLICT"
	Duplicate Decision = "DUPLICATE"
)

func (Decision) AuthorizesMutation() bool  { return false }
func (Decision) ProductionQualified() bool { return false }

// referenceOf is only used for records already validated by their caller.
func referenceOf(r Record) Ref {
	i := r.Identity
	return Ref{SourceFamily: i.Family, SourceVersion: i.WireVersion, SourceType: i.Kind,
		SourceOwner: i.Issuer.Peer(), SourceKey: i.RecordKey, SourceRevision: r.Snapshot.Revision,
		Representation: "MR1_RECORD_V1", Digest: r.RecordDigest}
}

func exact(r *Ref, record Record) bool {
	return r != nil && *r == referenceOf(record)
}

func addressedTo(i Identity, owner Owner) bool {
	return slices.Contains(i.Audience, owner.Peer())
}

func within(child, parent *Window) bool {
	return child != nil && parent != nil && child.NotBefore >= parent.NotBefore && child.ExpiresAt <= parent.ExpiresAt
}

// ValidateHandoff checks a contiguous prefix A1, A2, B, C, D, E. A reference
// is the complete versioned origin, including its sealed snapshot digest.
// No call resolves an external reference or authenticates its issuer.
func ValidateHandoff(records ...Record) error {
	kinds := [...]string{CPAdmissionProof, WorkerNonMediaBinding, AppliedCoverProof,
		MediaReleasePermit, MediaReleaseConsumption, MediaReleaseReceipt}
	if len(records) == 0 || len(records) > len(kinds) {
		return errors.New("invalid proof chain length")
	}
	for n, r := range records {
		if err := ValidateRecord(r); err != nil {
			return err
		}
		i := r.Identity
		if i.Kind != kinds[n] || i.Domain != records[0].Identity.Domain || i.Original != records[0].Identity.Original {
			return errors.New("proof chain kind, domain or original mismatch")
		}
		var valid bool
		switch n {
		case 0:
			valid = true
		case 1:
			valid = workerMatches(records[0], i)
		case 2:
			valid = coverMatches(records[0], records[1], i)
		case 3:
			valid = permitMatches(records[0], records[1], records[2], i)
		case 4:
			valid = consumptionMatches(records[3], r)
		case 5:
			valid = receiptMatches(records[3], records[4], records[4], r)
		}
		if !valid {
			return errors.New("inconsistent proof handoff")
		}
	}
	return nil
}

func workerMatches(cp Record, i Identity) bool {
	a := cp.Identity
	return exact(i.CPAdmissionRef, cp) && addressedTo(a, i.Issuer) &&
		i.Issuer.Peer() == a.WorkerAssignment.Assignee && i.Issuer.Peer() != a.Issuer.Peer() &&
		reflect.DeepEqual(i.WorkerAssignment, a.WorkerAssignment) &&
		*i.WorkerOriginalRef == *a.WorkerOriginalRef && *i.EncoderOriginalRef == *a.EncoderOriginalRef &&
		*i.ReservationRef == *a.ReservationRef && *i.MembershipSnapshotRef == *a.MembershipSnapshotRef &&
		i.AdmissionEpoch == a.AdmissionEpoch && within(i.Window, a.Window)
}

func coverMatches(cp, worker Record, i Identity) bool {
	return exact(i.CPAdmissionRef, cp) && exact(i.WorkerBindingRef, worker) &&
		i.Issuer.Peer() == cp.Identity.EncoderAssignment.Assignee && i.Issuer.Peer() != worker.Identity.Issuer.Peer() && i.Issuer.Peer() != cp.Identity.Issuer.Peer() &&
		addressedTo(cp.Identity, i.Issuer) && addressedTo(worker.Identity, i.Issuer) &&
		i.WorkerGeneration == worker.Identity.WorkerGeneration && within(i.Window, worker.Identity.Window)
}

func permitMatches(cp, worker, cover Record, i Identity) bool {
	if !exact(i.CPAdmissionRef, cp) || !exact(i.WorkerBindingRef, worker) || !exact(i.AppliedCoverRef, cover) ||
		i.Issuer != cp.Identity.Issuer || !addressedTo(cover.Identity, i.Issuer) ||
		!within(i.Window, cover.Identity.Window) || !within(i.Window, worker.Identity.Window) || !within(i.Window, cp.Identity.Window) {
		return false
	}
	switch i.Scope.Route {
	case "WORKER_SRT", "WORKER_DEEPGRAM":
		if i.Scope.TargetOwner != worker.Identity.Issuer {
			return false
		}
	case "ENCODER_DIRECT":
		if i.Scope.TargetOwner != cover.Identity.Issuer {
			return false
		}
	}
	// C must bind the exact closure evidence present in the attested graph.
	// A Worker input gate and an Encoder output gate can have different scopes.
	sameRoute, exactTarget := false, false
	for _, c := range cover.Identity.Closures {
		if !slices.Contains(i.ClosureRefs, c.FenceRef) {
			return false
		}
		if c.Scope.Route == i.Scope.Route {
			sameRoute = true
			if reflect.DeepEqual(c.Scope, *i.Scope) {
				exactTarget = true
			}
		}
	}
	if sameRoute || i.Scope.Route == "ENCODER_DIRECT" || i.Scope.Route == "RELAY_UPSTREAM" {
		return exactTarget
	}
	// For a separate upstream/input gate, C additionally identifies that gate's
	// own closure. An authenticated, scope-bound resolution is checked at CAS.
	return slices.ContainsFunc(i.ClosureRefs, func(ref Ref) bool {
		return ref.SourceOwner == i.Scope.TargetOwner.Peer() &&
			!slices.ContainsFunc(cover.Identity.Closures, func(cl Closure) bool { return cl.FenceRef == ref })
	})
}

func physicalScopeSubset(part, full Scope) bool {
	media := part.Media
	part.Media = full.Media
	if !reflect.DeepEqual(part, full) || len(media) == 0 {
		return false
	}
	for _, m := range media {
		if !slices.Contains(full.Media, m) {
			return false
		}
	}
	return true
}

func factsInScope(observations []Obs, scope Scope) bool {
	for _, o := range observations {
		if !physicalScopeSubset(o.Scope, scope) {
			return false
		}
	}
	return true
}

func consumptionMatches(p, d Record) bool {
	c, i := p.Identity, d.Identity
	return exact(i.PermitRef, p) && *c.ConsumptionKey == i.RecordKey && *c.ReceiptKey == *i.ReceiptKey &&
		c.Scope.TargetOwner == i.Issuer && addressedTo(c, i.Issuer) &&
		i.ExpectedSlotRevision == c.ExpectedSlotRevision && i.ConnectionOperationID == c.ConnectionOperationID &&
		i.LogicalEffectKey == c.LogicalEffectKey && factsInScope(d.Snapshot.Observations, *c.Scope) &&
		(d.Snapshot.EffectState != "RELEASE_CONFIRMED" || coversAllMedia(d.Snapshot.Observations, *c.Scope))
}

func receiptMatches(p, origin, observed, e Record) bool {
	c, i, s := p.Identity, e.Identity, e.Snapshot
	if !exact(i.PermitRef, p) || !exact(i.ConsumptionRef, origin) || !exact(s.ObservedConsumptionRef, observed) ||
		*c.ReceiptKey != i.RecordKey || i.Issuer != c.Scope.TargetOwner || i.LogicalEffectKey != c.LogicalEffectKey ||
		!reflect.DeepEqual(i.Scope, c.Scope) || s.ConsumptionState != observed.Snapshot.ConsumptionState ||
		!factsInScope(s.Observations, *c.Scope) {
		return false
	}
	// The most recent D may have been recorded before E obtained more evidence.
	// Only opposing known terminal observations are contradictory.
	return observed.Snapshot.EffectState == "EFFECT_UNKNOWN" || s.EffectState == observed.Snapshot.EffectState
}

func successor(old, new string) bool {
	a, err := strconv.ParseUint(old, 10, 64)
	return err == nil && a != ^uint64(0) && new == strconv.FormatUint(a+1, 10)
}

// CompareRevision checks the immutable key, predecessor seal, and monotone
// observations. Identical committed bytes are a read-only duplicate.
func CompareRevision(previous, next Record) Decision {
	equal, err := CompareImmutable(previous, next)
	if err != nil || !equal {
		return Conflict
	}
	if previous.RecordDigest == next.RecordDigest {
		return Duplicate
	}
	kind := next.Identity.Kind
	if kind != MediaReleaseConsumption && kind != MediaReleaseReceipt {
		return Conflict
	}
	a, b := previous.Snapshot, next.Snapshot
	if b.PreviousDigest != previous.RecordDigest || !successor(a.Revision, b.Revision) || b.RecordedAt < a.RecordedAt {
		return Conflict
	}
	if a.EffectState != "EFFECT_UNKNOWN" && b.EffectState != a.EffectState {
		return Conflict
	}
	for _, fact := range a.Observations {
		if !slices.ContainsFunc(b.Observations, func(f Obs) bool { return reflect.DeepEqual(f, fact) }) {
			return Conflict
		}
	}
	if kind == MediaReleaseReceipt {
		// E may observe several intervening D revisions at once; progress is
		// monotone even when those revisions are not adjacent in E's history.
		from, to := *a.ObservedConsumptionRef, *b.ObservedConsumptionRef
		older, _ := strconv.ParseUint(from.SourceRevision, 10, 64)
		newer, _ := strconv.ParseUint(to.SourceRevision, 10, 64)
		if from.SourceKey != to.SourceKey || from.SourceOwner != to.SourceOwner || newer < older ||
			newer == older && from != to || b.ObservationTime < a.ObservationTime ||
			!(validConsumptionStep(a.ConsumptionState, b.ConsumptionState) || a.ConsumptionState == "NOT_CONSUMED" && b.ConsumptionState == "CONSUMED") {
			return Conflict
		}
	}
	if kind == MediaReleaseConsumption {
		if !validConsumptionStep(a.ConsumptionState, b.ConsumptionState) {
			return Conflict
		}
		if a.ConsumptionState != b.ConsumptionState {
			if !successor(a.SlotRevision, b.SlotRevision) {
				return Conflict
			}
		} else if a.SlotRevision != b.SlotRevision || !reflect.DeepEqual(a.AuthorizationRefs, b.AuthorizationRefs) ||
			!reflect.DeepEqual(a.WorkRegistrationRef, b.WorkRegistrationRef) || !reflect.DeepEqual(a.DispatchRef, b.DispatchRef) ||
			a.CheckedAt != b.CheckedAt {
			return Conflict
		}
		if a.ConsumeCommitID != "" && a.ConsumeCommitID != b.ConsumeCommitID {
			return Conflict
		}
		for _, pair := range [][2]*Ref{{a.StopObservationRef, b.StopObservationRef}, {a.RevokeObservationRef, b.RevokeObservationRef},
			{a.FirstEffectRef, b.FirstEffectRef}, {a.ReceiptRef, b.ReceiptRef}} {
			if pair[0] != nil && (pair[1] == nil || *pair[1] != *pair[0]) {
				return Conflict
			}
		}
	}
	return Coherent
}

func validConsumptionStep(a, b string) bool {
	return a == b || (a == "NOT_CONSUMED" && b == "CONSUMPTION_PENDING") || (a == "CONSUMPTION_PENDING" && b == "CONSUMED")
}

// ValidateReceiptHistory permits a later Receipt to cite a verified descendant
// of its original consumption revision. Complete, consecutive D and E histories
// begin respectively at the D origin and first E; E cannot cite a future E.
func ValidateReceiptHistory(proofs []Record, consumption, receipts []Record) Decision {
	if len(proofs) != 4 || ValidateHandoff(proofs...) != nil || len(consumption) == 0 || len(receipts) == 0 {
		return Conflict
	}
	seenD := make(map[Ref]int, len(consumption))
	p := proofs[3]
	for n, d := range consumption {
		if d.Identity.Kind != MediaReleaseConsumption || d.Identity.Original != p.Identity.Original || d.Identity.Domain != p.Identity.Domain ||
			ValidateRecord(d) != nil || !consumptionMatches(p, d) || n > 0 && CompareRevision(consumption[n-1], d) != Coherent {
			return Conflict
		}
		seenD[referenceOf(d)] = n
	}
	seenE := make(map[Ref]bool, len(receipts))
	lastD := -1
	for n, e := range receipts {
		if ValidateRecord(e) != nil || e.Identity.Kind != MediaReleaseReceipt || e.Identity.Original != p.Identity.Original ||
			e.Identity.Domain != p.Identity.Domain || n > 0 && CompareRevision(receipts[n-1], e) != Coherent {
			return Conflict
		}
		pos, ok := seenD[*e.Snapshot.ObservedConsumptionRef]
		if !ok || pos < lastD || !receiptMatches(p, consumption[0], consumption[pos], e) {
			return Conflict
		}
		if n == 0 && (e.Snapshot.Revision != "1" || pos != 0) {
			return Conflict
		}
		for _, d := range consumption[lastD+1 : pos+1] {
			if d.Snapshot.ReceiptRef != nil && !seenE[*d.Snapshot.ReceiptRef] {
				return Conflict
			}
		}
		seenE[referenceOf(e)] = true
		lastD = pos
	}
	return Coherent
}

// AssessReceipt verifies both the historical chain and the scoped criteria for
// the claimed effect. A structurally coherent envelope or an HTTP ACK alone is
// insufficient. Receipt persistence must be witnessed separately by Inspect.
func AssessReceipt(proofs []Record, consumption, receipts []Record, profile OutcomeProfile) Decision {
	if history := ValidateReceiptHistory(proofs, consumption, receipts); history != Coherent {
		return history
	}
	e := receipts[len(receipts)-1]
	if result := AssessOutcome(*e.Identity.Scope, e.Snapshot.Observations, e.Snapshot.EffectState, profile); result != Coherent {
		return result
	}
	for _, rule := range profile.Criteria {
		if !slices.Contains(e.Snapshot.ResultEvidenceRefs, rule.Frame) {
			return Conflict
		}
	}
	return Coherent
}
