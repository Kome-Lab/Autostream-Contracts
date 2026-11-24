package mediarelease

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidFields reports a structural failure. Diagnostics name schema fields
// only; an invalid value can contain sensitive input.
var ErrInvalidFields = errors.New("invalid media release fields")

func fieldError(name string) error { return fmt.Errorf("%w: %s", ErrInvalidFields, name) }

func validID(s string) bool {
	if len(s) < 1 || len(s) > 192 || strings.Contains(s, "://") {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._:/-", rune(c))) {
			return false
		}
	}
	return true
}

func validUint(s string, positive bool) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && strconv.FormatUint(n, 10) == s && (!positive || n != 0)
}

func validTimestamp(s string) bool {
	const layout = "2006-01-02T15:04:05.000000000Z"
	if len(s) != len(layout) || strings.HasPrefix(s, "0000-") {
		return false
	}
	t, err := time.Parse(layout, s)
	return err == nil && t.Format(layout) == s
}

func validHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'f') {
			return false
		}
	}
	return true
}

// recordClass is the complete registry. Short markers address the applicability
// columns on Identity and Snapshot, and are never wire values.
func recordClass(kind string) (class, cc, purpose string) {
	switch kind {
	case CPAdmissionProof:
		return "a", "CC10", "ADMISSION_AUTHORITY"
	case WorkerNonMediaBinding:
		return "w", "CC10", "NON_MEDIA_BINDING"
	case AppliedCoverProof:
		return "b", "CC11", "COVER_ATTESTATION"
	case MediaReleasePermit:
		return "c", "CC12", "AUTHORIZE_OPEN"
	case MediaReleaseConsumption:
		return "d", "CC13", "CONSUME_OPEN"
	case MediaReleaseReceipt:
		return "e", "CC14", "REPORT_RESULT"
	}
	return "", "", ""
}

func scalarIsValid(s, kind string) bool {
	switch kind {
	case "positive", "uint":
		return validUint(s, kind == "positive")
	case "time":
		return validTimestamp(s)
	case "hash":
		return validHash(s)
	case "":
		return validID(s)
	default:
		return false
	}
}

// visit applies requiredness, the closed kind union, scalar encodings and
// canonical set ordering before checking relationships between fields.
func visit(v reflect.Value, name, scalar, class string) error {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return fieldError(name)
		}
		return visit(v.Elem(), name, scalar, class)
	case reflect.Bool:
		return nil
	case reflect.String:
		if !scalarIsValid(v.String(), scalar) {
			return fieldError(name)
		}
	case reflect.Slice:
		if v.Len() == 0 || v.Len() > 64 {
			return fieldError(name)
		}
		var previous string
		for n := 0; n < v.Len(); n++ {
			item := v.Index(n)
			if err := visit(item, name, scalar, class); err != nil {
				return err
			}
			encoded, err := CanonicalBytes(item.Interface())
			if err != nil || n != 0 && string(encoded) <= previous {
				return fieldError(name)
			}
			previous = string(encoded)
		}
	case reflect.Struct:
		typ := v.Type()
		for j := 0; j < v.NumField(); j++ {
			f := typ.Field(j)
			field := v.Field(j)
			jsonName, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			allow := f.Tag.Get("kinds")
			if allow != "" && (class == "" || !strings.Contains(allow, class)) {
				if !field.IsZero() {
					return fieldError(jsonName)
				}
				continue
			}
			if f.Tag.Get("optional") == "true" && field.IsZero() {
				continue
			}
			if err := visit(field, jsonName, f.Tag.Get("wire"), class); err != nil {
				return err
			}
		}
		return checkNested(v.Interface())
	default:
		return fieldError(name)
	}
	return nil
}

func validateValue(v any) error { return visit(reflect.ValueOf(v), "value", "", "") }

func checkNested(value any) error {
	switch v := value.(type) {
	case Ref:
		if v.SourceFamily == Family || v.Representation == "MR1_RECORD_V1" {
			class, _, _ := recordClass(v.SourceType)
			if v.SourceFamily != Family || v.SourceVersion != WireVersion || v.Representation != "MR1_RECORD_V1" || class == "" || v.SourceKey.Kind != v.SourceType {
				return fieldError("representation")
			}
			if class != "d" && class != "e" && v.SourceRevision != "1" {
				return fieldError("source_revision")
			}
		} else if v.Representation != "REQUEST1_COMMAND_BYTES" && v.Representation != "PINNED_RECORD_BYTES" && v.Representation != "OWNER_SEALED_BYTES" {
			return fieldError("representation")
		}
	case Window:
		if v.NotBefore >= v.ExpiresAt {
			return fieldError("expires_at")
		}
	case Scope:
		switch v.Route {
		case "ENCODER_DIRECT", "RELAY_UPSTREAM", "WORKER_SRT", "WORKER_DEEPGRAM", "HTTP_CALLBACK":
		default:
			return fieldError("route")
		}
	case Closure:
		if v.IntervalStart > v.ObservedThrough {
			return fieldError("observed_through")
		}
		if v.FenceRef.SourceOwner != v.Scope.TargetOwner.Peer() || v.FrameRef.SourceOwner != v.Scope.TargetOwner.Peer() {
			return fieldError("fence_ref")
		}
	case Obs:
		switch v.Knowledge {
		case "KNOWN":
			if v.FrameRef == nil || v.FrameRef.SourceOwner != v.Observer.Peer() || v.ReasonCode != "" {
				return fieldError("frame_ref")
			}
		case "UNKNOWN":
			if v.FrameRef != nil || v.ReasonCode == "" {
				return fieldError("reason_code")
			}
		default:
			return fieldError("knowledge")
		}
	}
	return nil
}

func sourceIs(r *Ref, kind, deployment string) bool {
	return r != nil && r.SourceFamily == Family && r.SourceVersion == WireVersion && r.Representation == "MR1_RECORD_V1" && r.SourceType == kind && r.SourceKey.Kind == kind && r.SourceKey.DeploymentID == deployment
}

func keyIs(k Key, kind, deployment string) bool {
	return k.Kind == kind && k.DeploymentID == deployment
}

// observationInScope permits a subset of the authorized media on the exact
// same gate, owner incarnation, endpoint and success profile.
func observationInScope(observed, authorized Scope) bool {
	for _, medium := range observed.Media {
		if !slices.Contains(authorized.Media, medium) {
			return false
		}
	}
	observed.Media, authorized.Media = nil, nil
	return reflect.DeepEqual(observed, authorized)
}

// coversAllMedia is a necessary condition for a full-scope success. The
// referenced success profile and qualified evidence are independently resolved.
func coversAllMedia(observations []Obs, full Scope) bool {
	for _, medium := range full.Media {
		seen := slices.ContainsFunc(observations, func(o Obs) bool {
			return o.Knowledge == "KNOWN" && observationInScope(o.Scope, full) && slices.Contains(o.Scope.Media, medium)
		})
		if !seen {
			return false
		}
	}
	return true
}

// ValidateFields checks a single record body. Verifying its seals, resolving
// originals, authenticating issuers and evaluating freshness are separate steps.
func ValidateFields(i Identity, s Snapshot) error {
	class, cc, purpose := recordClass(i.Kind)
	if class == "" || i.CC != cc || i.Purpose != purpose {
		return fieldError("kind")
	}
	if i.Family != Family || i.WireVersion != WireVersion || i.EncodingID != EncodingID || i.ProfileID != ProfileID || i.ProfileRevision != ProfileRevision {
		return fieldError("family")
	}
	if err := visit(reflect.ValueOf(i), "identity", "", class); err != nil {
		return err
	}
	if err := visit(reflect.ValueOf(s), "snapshot", "", class); err != nil {
		return err
	}
	if !keyIs(i.RecordKey, i.Kind, i.Domain.DeploymentID) {
		return fieldError("record_key")
	}
	if s.RecordedAt < i.IssuedAt || (s.Revision == "1") != (s.PreviousDigest == "") {
		return fieldError("snapshot")
	}
	if i.Window != nil && (s.Revision != "1" || i.IssuedAt >= i.Window.ExpiresAt || s.RecordedAt >= i.Window.ExpiresAt) {
		return fieldError("window")
	}
	if err := checkIdentity(i); err != nil {
		return err
	}
	if i.Kind == MediaReleaseConsumption || i.Kind == MediaReleaseReceipt {
		return checkResult(i, s)
	}
	return nil
}

func checkIdentity(i Identity) error {
	deployment := i.Domain.DeploymentID
	for _, expected := range []struct {
		r    *Ref
		kind string
	}{
		{i.CPAdmissionRef, CPAdmissionProof}, {i.WorkerBindingRef, WorkerNonMediaBinding},
		{i.AppliedCoverRef, AppliedCoverProof}, {i.PermitRef, MediaReleasePermit},
		{i.ConsumptionRef, MediaReleaseConsumption},
	} {
		if expected.r != nil && !sourceIs(expected.r, expected.kind, deployment) {
			return fieldError("source_type")
		}
	}
	if i.WorkerOriginalRef != nil && i.WorkerOriginalRef.SourceKey == i.EncoderOriginalRef.SourceKey {
		return fieldError("encoder_original_ref")
	}
	switch i.Kind {
	case CPAdmissionProof:
		if i.WorkerAssignment.Assignee == i.EncoderAssignment.Assignee {
			return fieldError("encoder_assignment")
		}
	case WorkerNonMediaBinding:
		if i.WorkerAssignment.Assignee != i.Issuer.Peer() {
			return fieldError("worker_assignment")
		}
	case AppliedCoverProof:
		if *i.AppliedActive != (i.AssetRef != nil) || *i.AppliedActive != (i.VariantID != "") || *i.AppliedActive == (i.HiddenObservationRef != nil) {
			return fieldError("applied_active")
		}
		if i.ObservedAt > i.IssuedAt {
			return fieldError("observed_at")
		}
		for _, closure := range i.Closures {
			if closure.GraphID != i.GraphID || closure.GraphGeneration != i.GraphGeneration || closure.IntervalStart > i.ObservedAt || closure.ObservedThrough < i.ObservedAt {
				return fieldError("closures")
			}
		}
	case MediaReleasePermit:
		if len(i.Audience) != 1 || i.Audience[0] != i.Scope.TargetOwner.Peer() {
			return fieldError("audience")
		}
		if !keyIs(*i.ConsumptionKey, MediaReleaseConsumption, deployment) || !keyIs(*i.ReceiptKey, MediaReleaseReceipt, deployment) || i.ConsumptionKey.OwnerID != i.ReceiptKey.OwnerID {
			return fieldError("consumption_key")
		}
	case MediaReleaseConsumption:
		if !keyIs(*i.ReceiptKey, MediaReleaseReceipt, deployment) || i.RecordKey.OwnerID != i.ReceiptKey.OwnerID {
			return fieldError("receipt_key")
		}
		sum, err := OpenRequestDigest(i)
		if err != nil || sum != i.OpenRequestDigest {
			return fieldError("open_request_digest")
		}
	case MediaReleaseReceipt:
		if i.Issuer != i.Scope.TargetOwner || i.ConsumptionRef.SourceOwner != i.Issuer.Peer() || i.ConsumptionRef.SourceKey.OwnerID != i.RecordKey.OwnerID {
			return fieldError("consumption_ref")
		}
	}
	return nil
}

func checkResult(i Identity, s Snapshot) error {
	switch s.ConsumptionState {
	case "NOT_CONSUMED", "CONSUMPTION_PENDING", "CONSUMED":
	default:
		return fieldError("consumption_state")
	}
	switch s.EffectState {
	case "RELEASE_CONFIRMED":
		if s.ConsumptionState != "CONSUMED" {
			return fieldError("consumption_state")
		}
	case "RELEASE_FAILED", "EFFECT_UNKNOWN":
	default:
		return fieldError("effect_state")
	}
	anyKnown := false
	for _, obs := range s.Observations {
		if i.Scope != nil && !observationInScope(obs.Scope, *i.Scope) {
			return fieldError("observations")
		}
		anyKnown = anyKnown || obs.Knowledge == "KNOWN"
	}
	if s.EffectState != "EFFECT_UNKNOWN" && !anyKnown {
		return fieldError("observations")
	}
	if i.Kind == MediaReleaseReceipt {
		if s.EffectState == "RELEASE_CONFIRMED" && !coversAllMedia(s.Observations, *i.Scope) {
			return fieldError("observations")
		}
		ref := s.ObservedConsumptionRef
		origin := i.ConsumptionRef
		if !sourceIs(ref, MediaReleaseConsumption, i.Domain.DeploymentID) || ref.SourceKey != origin.SourceKey || ref.SourceOwner != origin.SourceOwner ||
			(s.Revision == "1" && *ref != *origin) {
			return fieldError("observed_consumption_ref")
		}
		actual, _ := strconv.ParseUint(ref.SourceRevision, 10, 64)
		first, _ := strconv.ParseUint(origin.SourceRevision, 10, 64)
		if actual < first || (actual == first && *ref != *origin) {
			return fieldError("observed_consumption_ref")
		}
		if s.ObservationTime > s.RecordedAt || (s.EffectState != "EFFECT_UNKNOWN") != (s.ResultEvidenceRefs != nil) {
			return fieldError("result_evidence_refs")
		}
		return nil
	}
	admitted := s.ConsumptionState != "NOT_CONSUMED"
	if admitted != (s.AuthorizationRefs != nil) || admitted != (s.DispatchRef != nil) || admitted != (s.WorkRegistrationRef != nil) || admitted != (s.CheckedAt != "") || s.CheckedAt > s.RecordedAt {
		return fieldError("authorization_refs")
	}
	if (s.ConsumptionState == "CONSUMED") != (s.ConsumeCommitID != "") {
		return fieldError("consume_commit_id")
	}
	if s.ReceiptRef != nil && (s.Revision == "1" || !sourceIs(s.ReceiptRef, MediaReleaseReceipt, i.Domain.DeploymentID) || s.ReceiptRef.SourceKey != *i.ReceiptKey || s.ReceiptRef.SourceOwner != i.Issuer.Peer()) {
		return fieldError("receipt_ref")
	}
	return nil
}

// ValidateInspectRequest recognizes an exact, read-only query, including when
// the caller lost the response and cannot supply the result digest.
func ValidateInspectRequest(q InspectRequest) error {
	if err := validateValue(q); err != nil {
		return err
	}
	if class, _, _ := recordClass(q.LookupKey.Kind); class == "" {
		return fieldError("lookup_key")
	}
	return nil
}

func ValidateInspectResult(result InspectResult) error {
	if err := validateValue(result); err != nil {
		return err
	}
	if class, _, _ := recordClass(result.LookupKey.Kind); class == "" {
		return fieldError("lookup_key")
	}
	found := result.Status == "FOUND"
	if !found && result.Status != "UNKNOWN" && result.Status != "CONFLICT" {
		return fieldError("status")
	}
	if found != (result.FoundRef != nil) || found != (result.StorageWitnessRef != nil) {
		return fieldError("found_ref")
	}
	if found && (!sourceIs(result.FoundRef, result.LookupKey.Kind, result.LookupKey.DeploymentID) || result.FoundRef.SourceKey != result.LookupKey || result.FoundRef.SourceOwner != result.Responder.Peer()) {
		return fieldError("found_ref")
	}
	switch result.Currentness {
	case "CURRENT":
		if !found || result.FreshnessRef == nil {
			return fieldError("freshness_ref")
		}
	case "EXPIRED", "REVOKED":
		if !found || result.FreshnessRef != nil || result.ReasonRefs == nil {
			return fieldError("reason_refs")
		}
	case "UNKNOWN":
		if result.FreshnessRef != nil {
			return fieldError("freshness_ref")
		}
	default:
		return fieldError("currentness")
	}
	if result.Status == "CONFLICT" && result.ReasonRefs == nil {
		return fieldError("reason_refs")
	}
	return nil
}

func ValidateFenceObservation(f FenceObservation) error {
	if err := validateValue(f); err != nil {
		return err
	}
	if f.PermitRef != nil {
		if f.AuthorityRef != nil || f.SlotKey != nil || !sourceIs(f.PermitRef, MediaReleasePermit, f.PermitRef.SourceKey.DeploymentID) {
			return fieldError("permit_ref")
		}
	} else if f.AuthorityRef == nil || f.SlotKey == nil || f.SlotKey.Kind != MediaReleaseConsumption {
		return fieldError("slot_key")
	}
	for _, o := range f.Observations {
		if !observationInScope(o.Scope, f.Scope) {
			return fieldError("observations")
		}
		switch o.Fact {
		case "GATE_FENCE_DURABLE", "INFLIGHT_JOINED", "LOCAL_FD_CLOSED":
			if o.Knowledge == "KNOWN" && o.Observer != f.Scope.TargetOwner {
				return fieldError("observer")
			}
		case "CP_REVOCATION_RECORDED":
			source := f.PermitRef
			if source == nil {
				source = f.AuthorityRef
			}
			if o.Knowledge == "KNOWN" && o.Observer.Peer() != source.SourceOwner {
				return fieldError("observer")
			}
		case "REMOTE_TERMINAL":
			// The remote role and exact target require independent authentication.
		default:
			return fieldError("fact")
		}
		if o.Knowledge == "KNOWN" && f.DecisionRef == nil {
			return fieldError("decision_ref")
		}
	}
	return nil
}
