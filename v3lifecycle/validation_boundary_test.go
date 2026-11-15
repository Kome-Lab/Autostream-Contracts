package lifecycle

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidationBoundaryBindings(t *testing.T) {
	seal, p := fixtureRecord(CC05)
	completion, _ := fixtureRecord(CC04)
	for _, field := range []string{"attempt", "execution", "request", "owner", "module"} {
		t.Run("seal/"+field, func(t *testing.T) {
			c := fixtureClone(t, completion)
			switch field {
			case "attempt":
				c.Header.Correlation.AttemptID = "different-attempt"
			case "execution":
				c.Header.Correlation.ExecutionKey = "different-execution"
			case "request":
				c.Header.Correlation.RequestID = "different-request"
			case "owner":
				c.Header.RecordKey.OwnerID = "different-owner"
			case "module":
				c.Header.ContractRef.ModuleCommitRef = "2222222222222222222222222222222222222222"
				c.Header.RecordKey.ContractRef = c.Header.ContractRef
			}
			fixtureWant(t, ValidateRecord(c, p), StructurallyCoherent)
			fixtureWant(t, ValidateSealAgainstCompletion(seal, c, p), Conflict)
		})
	}
	for _, field := range []string{"query-scope", "coverage-scope", "frame-scope", "outcome-owner", "outcome-revision", "missing-frame"} {
		t.Run("inspection/"+field, func(t *testing.T) {
			r, p := fixtureRecord(CC08)
			switch field {
			case "query-scope":
				r.Inspection.Target.Scope = "other-scope"
			case "coverage-scope":
				r.Inspection.Coverage.Observation.Scope = "other-scope"
			case "frame-scope":
				r.Inspection.SourceFrames[0].Scope = "other-scope"
			case "outcome-owner":
				r.Inspection.Observations.Observations[0].Outcome.Observation.OwnerID = "other-owner"
			case "outcome-revision":
				r.Inspection.Observations.Observations[0].Outcome.Observation.SourceRevision = 2
			case "missing-frame":
				r.Inspection.Observations.Observations[0].Outcome.Observation.FrameRef = "missing-frame"
			}
			got := ValidateRecord(r, p)
			if got.Kind == StructurallyCoherent || got.AuthorizesMutation() {
				t.Fatalf("scoped frame mismatch accepted: %+v", got)
			}
		})
	}
	limits := fixtureLimits()
	badIDs := []ID{ID(strings.Repeat("x", limits.MaxIDBytes+1)), ID("https:" + "/" + "/" + "fixture.invalid/private"), ID("control\x00value")}
	for _, bad := range badIDs {
		t.Run("conditional-IDs", func(t *testing.T) {
			o := BoundaryObservation{Address: fixtureBoundary(), Knowledge: Known, Presence: Present, Outcome: fixtureOutcome(SuccessAtBoundary), DetailRef: bad}
			fixtureWant(t, ValidateBoundaryObservation(o, limits), RefusedUnchanged)
			identity := IdentityContext{Class: Human, PrincipalKind: PrincipalHuman, ActorID: "human", Purpose: PurposeClosure, FlowRef: bad}
			fixtureWant(t, ValidateIdentity(identity, limits), RefusedUnchanged)
			q := BoundaryObservation{Address: BoundaryAddress{Kind: LocalPersistence, OwnerID: "owner-1", Resource: fixtureResource(Partition, "partition-1"), SubBoundary: QueueAdmission}, Knowledge: Known, Presence: Present, Outcome: fixtureOutcome(SuccessAtBoundary), Details: &BoundaryDetails{Queue: &QueueDetail{Work: WorkRef{WorkID: "work-1", ParentWorkID: bad, Namespace: fixtureNamespace(), Sequence: 1, AdmissionRecordRef: "admission-record-1"}, ReceiptRef: "queue-receipt-1"}}}
			fixtureWant(t, ValidateBoundaryObservation(q, limits), RefusedUnchanged)
		})
	}
	r, p := fixtureRecord(CC09)
	r.Reconcile.ExpectedProjection = fixtureResource(Host, "host-1")
	fixtureWant(t, ValidateRecord(r, p), RefusedUnchanged)
}

func TestStrictLiteralJSON(t *testing.T) {
	r, p := fixtureRecord(CC02)
	good, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	original := []byte("\"RequestID\":\"request-1\"")
	for _, replacement := range [][]byte{
		append([]byte("\"RequestID\":\""), append([]byte{0xff}, []byte("\"")...)...),
		[]byte("\"RequestID\":\"\\uD800\""),
		[]byte("\"RequestID\":\"\\uDC00\""),
		[]byte("\"RequestID\":\"\\uD800x\\uDC00\""),
		[]byte("\"RequestID\":\"\\uD800\\u0041\""),
	} {
		bad := bytes.Replace(good, original, replacement, 1)
		if bytes.Equal(good, bad) {
			t.Fatal("mutation missed original input")
		}
		_, got := DecodeRecord(bad, p)
		fixtureWant(t, got, RefusedUnchanged)
	}
	validPair := bytes.Replace(good, original, []byte("\"RequestID\":\"\\uD83D\\uDE00\""), 1)
	decoded, got := DecodeRecord(validPair, p)
	fixtureWant(t, got, StructurallyCoherent)
	if decoded.Header.Correlation.RequestID != "😀" {
		t.Fatal("valid surrogate pair not preserved")
	}
	longAlias := append(append([]byte{}, good[:len(good)-1]...), []byte(",\"ſeal\":null}")...)
	_, longResult := DecodeRecord(longAlias, p)
	fixtureWant(t, longResult, RefusedUnchanged)
	// Unicode simple-fold aliases accepted by encoding/json cannot replace a payload.
	head, e := json.Marshal(r.Header)
	if e != nil {
		t.Fatal(e)
	}
	alias := append(append(append([]byte{}, good[:len(good)-1]...), []byte(",\"Header\":")...), head...)
	alias = append(alias, '}')
	_, dup := DecodeRecord(alias, p)
	fixtureWant(t, dup, RefusedUnchanged)
}
