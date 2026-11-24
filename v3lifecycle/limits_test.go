package lifecycle

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestExplicitLimitsAndMissingValues(t *testing.T) {
	fixtureWant(t, ValidateLimits(Limits{}), NeedsDecision)
	limits := fixtureLimits()
	fixtureWant(t, ValidateLimits(limits), StructurallyCoherent)
	for i := 0; i < 8; i++ {
		for _, value := range []int{0, -1} {
			bad := limits
			fields := []*int{&bad.MaxIDBytes, &bad.MaxMembers, &bad.MaxAxes, &bad.MaxRefs, &bad.MaxObligations, &bad.MaxPartitions, &bad.MaxRanges, &bad.MaxDepth}
			*fields[i] = value
			fixtureWant(t, ValidateLimits(bad), NeedsDecision)
		}
	}
	overflow := limits
	overflow.MaxRefs = math.MaxInt
	fixtureWant(t, ValidateLimits(overflow), NeedsDecision)
	for _, id := range []ID{"", ID(string([]byte{0xff})), ID("control\x00value"), ID(strings.Repeat("x", limits.MaxIDBytes+1))} {
		fixtureWant(t, ValidateID(id, limits), RefusedUnchanged)
	}
	for _, id := range []ID{"  literal preserved  ", "é", "e\u0301", "CaseSensitive", "caseSensitive", "0"} {
		fixtureWant(t, ValidateID(id, limits), StructurallyCoherent)
	}
	p := fixtureProfile(MediaIngress)
	fixtureWant(t, ValidateProfile(ComparisonProfile{}), NeedsDecision)
	missing := fixtureClone(t, p)
	missing.AxisRules = nil
	fixtureWant(t, ValidateProfile(missing), NeedsDecision)
	allOptional := fixtureClone(t, p)
	for i := range allOptional.AxisRules {
		allOptional.AxisRules[i].Required = false
	}
	fixtureWant(t, ValidateProfile(allOptional), NeedsDecision)
	typed := fixtureClone(t, p)
	typed.AxisRules[0].ValueKind = PresentLiteral
	fixtureWant(t, ValidateProfile(typed), NeedsDecision)
	v := fixtureVector()
	v = append(v, fixtureClone(t, v[0]))
	v[len(v)-1].Incarnation = "new-incarnation-still-same-key"
	fixtureWant(t, ValidateVector(v, p), RefusedUnchanged)
	r, p := fixtureRecord(CC02)
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	_, good := DecodeRecord(b, p)
	fixtureWant(t, good, StructurallyCoherent)
	// The decoder also rejects case-aliased duplicate object keys accepted by encoding/json.
	h, e := json.Marshal(r.Header)
	if e != nil {
		t.Fatal(e)
	}
	alias := append(append(append([]byte{}, b[:len(b)-1]...), []byte(",\"header\":")...), h...)
	alias = append(alias, '}')
	_, duplicate := DecodeRecord(alias, p)
	fixtureWant(t, duplicate, RefusedUnchanged)
	shallow := p
	shallow.Limits.MaxDepth = 1
	_, deep := DecodeRecord(b, shallow)
	fixtureWant(t, deep, RefusedUnchanged)
	_, oversize := DecodeRecord(bytes.Repeat([]byte(" "), p.Limits.MaxIDBytes*p.Limits.MaxRefs*16+1), p)
	fixtureWant(t, oversize, RefusedUnchanged)
	for _, f := range []Fact[ID]{
		{},
		{State: Observed},
		{State: Observed, Value: new(ID("value")), Observation: nil},
		{State: Unavailable, Unavailable: &UnavailableReason{Code: "unavailable"}, Observation: fixtureObservation()},
	} {
		fixtureWant(t, ValidateFact(f, limits), RefusedUnchanged)
	}
	absent := OptionalRef{State: Absent, AbsenceFactRef: "not-applicable-owner-fact", Observation: fixtureObservation()}
	fixtureWant(t, ValidateOptionalRef(absent, limits), StructurallyCoherent)
	absent.Ref = new(ID("invented-present-reference"))
	fixtureWant(t, ValidateOptionalRef(absent, limits), RefusedUnchanged)
	unknown := OptionalRef{State: NotObserved, Unavailable: &UnavailableReason{Code: "not-observed"}}
	fixtureWant(t, ValidateOptionalRef(unknown, limits), NeedsEvidence)
}
