package lifecycle

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func member[T comparable](v T, values ...T) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}
func word(v string, list string) bool {
	for _, x := range strings.Fields(list) {
		if v == x {
			return true
		}
	}
	return false
}
func validComponent(c ComponentID) bool {
	return word(string(c), "C01 C02 C03 C04 C05 C06 C07 C08 C09 C10 C11 C12 C13 C14")
}
func validContract(c ContractID) bool {
	return member(c, CC01, CC02, CC03, CC04, CC05, CC06, CC07, CC08, CC09)
}
func validResource(k ResourceKind) bool {
	return word(string(k), "USER ROLE ROLE_ASSIGNMENT SESSION IDENTITY_LINK AUTH_PROVIDER AUTH_POLICY FACTOR PASSKEY RECOVERY_METADATA WORKLOAD_SUBJECT APPROVAL TRUST_ROOT_REF LEDGER_REF CLOCK_REF GRANT_REF KEY_METADATA SECRET_METADATA STREAM RUN NODE JOB CONNECTION HOST PARTITION SINK ARTIFACT MEDIA_ASSET ARCHIVE QUEUE INCIDENT NOTIFICATION_CHANNEL CONFIG RELEASE BACKUP_POINT RETURN_POINT PROJECTION")
}
func validAxis(a Axis) bool {
	return word(string(a), "A_REVISION C_EPOCH IDENTITY MEMBERSHIP ADMISSION OWNERSHIP FENCE RUN JOB CONNECTION WRITER EFFECT COVER POLICY TRUST BOOT PROCESS RELEASE CONFIG")
}
func validPurpose(p Purpose) bool {
	return member(p, PurposeApproval, PurposeIngress, PurposeClosure, PurposeLookup, PurposeEffect, PurposeRepair, PurposeResume, PurposeRecovery, PurposeRecordTransfer)
}
func validProfile(p ProfileID) bool {
	return member(p, ApprovalLocal, MediaIngress, ClosureAssess, StopLookup, EffectOutcome, RecordRepair, ExplicitResume)
}
func validKnowledge(k Knowledge) bool { return member(k, Known, KnowledgeUnknown, KnowledgeConflict) }
func validGate(g Gate) bool           { return member(g, Held, NotQualified, VerifiedForProfile) }

// ValidateID preserves literal bytes. Diagnostics never echo input values.
// Semantic secrecy of an opaque reference remains the trusted owner's duty.
func ValidateID(id ID, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	if id == "" || len(id) > l.MaxIDBytes || !utf8.ValidString(string(id)) {
		return result(RefusedUnchanged, "INVALID_LITERAL", "ID")
	}
	for _, r := range string(id) {
		if unicode.IsControl(r) {
			return result(RefusedUnchanged, "CONTROL_IN_LITERAL", "ID")
		}
	}
	text := string(id)
	for _, s := range []string{"-----BEGIN ", "ghp_", "github_pat_", "xoxb-", "xoxp-", "AIza", "AKIA", "https://", "http://", "postgres://", "mysql://", "C:\\"} {
		if strings.Contains(text, s) {
			return result(RefusedUnchanged, "SENSITIVE_OR_RAW_LITERAL", "ID")
		}
	}
	return coherent()
}

type checker struct {
	limits Limits
	issues []Issue
	kind   ResultKind
}

func newChecker(l Limits) *checker { return &checker{limits: l, kind: StructurallyCoherent} }
func (c *checker) add(k ResultKind, code, path string) {
	c.issues = append(c.issues, Issue{Code: code, Path: path})
	if c.kind == StructurallyCoherent || priority(k) > priority(c.kind) {
		c.kind = k
	}
}
func priority(k ResultKind) int {
	switch k {
	case Conflict:
		return 8
	case RefusedUnchanged:
		return 7
	case NeedsDecision:
		return 6
	case Unknown:
		return 5
	case NeedsEvidence:
		return 4
	case Stale:
		return 3
	case PendingObligation:
		return 2
	}
	return 0
}
func (c *checker) take(r Result, path string) {
	if r.Kind != StructurallyCoherent {
		for _, i := range r.Issues {
			c.add(r.Kind, i.Code, path+"."+i.Path)
		}
		if len(r.Issues) == 0 {
			c.add(r.Kind, "INCOMPLETE", path)
		}
	}
}
func (c *checker) finish() Result { return Result{Kind: c.kind, Issues: c.issues} }
func (c *checker) require(ok bool, code, path string) {
	if !ok {
		c.add(RefusedUnchanged, code, path)
	}
}
func (c *checker) id(v ID, path string) { c.take(ValidateID(v, c.limits), path) }
func (c *checker) ids(ids []ID, max int, path string, nonempty bool) {
	c.require(len(ids) <= max && (!nonempty || len(ids) > 0), "INVALID_SET_SIZE", path)
	if len(ids) > max {
		return
	}
	seen := map[ID]bool{}
	for _, v := range ids {
		c.id(v, path)
		c.require(!seen[v], "DUPLICATE_SET_MEMBER", path)
		seen[v] = true
	}
}
func (c *checker) resource(r ResourceRef, path string) {
	c.require(validResource(r.Kind), "UNKNOWN_RESOURCE_KIND", path)
	c.id(r.OwnerID, path+".OwnerID")
	c.id(r.ResourceID, path+".ResourceID")
}
func (c *checker) resources(rs []ResourceRef, path string, nonempty bool) {
	c.require(len(rs) <= c.limits.MaxMembers && (!nonempty || len(rs) > 0), "INVALID_RESOURCE_SET", path)
	if len(rs) > c.limits.MaxMembers {
		return
	}
	seen := map[ResourceRef]bool{}
	for _, r := range rs {
		c.resource(r, path)
		c.require(!seen[r], "DUPLICATE_RESOURCE", path)
		seen[r] = true
	}
}
func (c *checker) producer(p ProducerRef, path string) {
	c.require(validComponent(p.ComponentID), "UNKNOWN_COMPONENT", path)
	for _, id := range []ID{p.ServiceID, p.HostID, p.BootID, p.ProcessIncarnation, p.WriterID} {
		c.id(id, path)
	}
}
func (c *checker) frame(f SourceFrame, path string) {
	for _, id := range []ID{f.FrameRef, f.OwnerID, f.Scope} {
		c.id(id, path)
	}
	c.require(f.SourceRevision > 0, "MISSING_SOURCE_REVISION", path)
}
func (c *checker) correlation(t Correlation, path string) {
	for _, id := range []ID{t.RequestID, t.OperationID, t.AttemptID, t.StepID, t.ExecutionKey} {
		c.id(id, path)
	}
}
func ValidateCorrelation(t Correlation, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	c.correlation(t, "Correlation")
	return c.finish()
}
func ValidateIdentity(i IdentityContext, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	c.id(i.ActorID, "Identity.Actor")
	c.require(validPurpose(i.Purpose), "UNKNOWN_PURPOSE", "Identity.Purpose")
	if i.FlowRef != "" {
		c.id(i.FlowRef, "Identity.Flow")
	}
	switch i.Class {
	case Human:
		c.require(i.PrincipalKind == PrincipalHuman && i.WorkloadSubject == "", "HUMAN_WORKLOAD_SEPARATION", "Identity")
	case PendingIdentity:
		c.require(i.PrincipalKind == PrincipalHuman && i.WorkloadSubject == "", "PENDING_CLASS_NOT_PRINCIPAL_ENUM", "Identity")
		c.id(i.FlowRef, "Identity.Flow")
	case Preauth:
		c.require(i.PrincipalKind == PrincipalPreauth && i.WorkloadSubject == "", "PREAUTH_KIND_MISMATCH", "Identity")
		c.id(i.FlowRef, "Identity.Flow")
	case System:
		c.require(i.PrincipalKind == PrincipalSystem, "SYSTEM_KIND_MISMATCH", "Identity")
		c.id(i.WorkloadSubject, "Identity.Workload")
	default:
		c.add(RefusedUnchanged, "UNKNOWN_IDENTITY_CLASS", "Identity")
	}
	return c.finish()
}
func ValidateFact[T any](f Fact[T], l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	switch f.State {
	case Observed:
		c.require(f.Value != nil && f.Observation != nil && f.Unavailable == nil, "INVALID_OBSERVED_UNION", "Fact")
		if f.Observation != nil {
			o := f.Observation
			c.id(o.OwnerID, "Fact.Owner")
			c.id(o.FrameRef, "Fact.Frame")
			c.id(o.Scope, "Fact.Scope")
			c.require(o.SourceRevision > 0, "MISSING_SOURCE_REVISION", "Fact")
		}
		if f.Value != nil {
			switch v := any(*f.Value).(type) {
			case ID:
				c.id(v, "Fact.Value")
			case Outcome:
				c.require(member(v, SuccessAtBoundary, FailureAtBoundary, NoEffectProven, Pending, OutcomeUnknown), "UNKNOWN_OUTCOME", "Fact.Value")
			default:
				c.add(RefusedUnchanged, "UNSUPPORTED_FACT_TYPE", "Fact")
			}
		}
	case Unavailable:
		c.require(f.Value == nil && f.Observation == nil && f.Unavailable != nil, "INVALID_UNAVAILABLE_UNION", "Fact")
		if f.Unavailable != nil {
			c.id(f.Unavailable.Code, "Fact.Reason")
			c.ids(f.Unavailable.MissingRefs, l.MaxRefs, "Fact.MissingRefs", false)
		}
		c.add(NeedsEvidence, "UNAVAILABLE_FACT", "Fact")
	default:
		c.add(RefusedUnchanged, "UNKNOWN_FACT_STATE", "Fact")
	}
	return c.finish()
}
func (c *checker) value(e RevisionEntry, path string) {
	v := e.Value
	switch v.Kind {
	case PresentUint64:
		c.require(v.Uint64 != nil && v.Literal == nil && v.Unavailable == nil, "INVALID_UINT_UNION", path)
		c.id(e.SourceFrameRef, path+".Frame")
	case PresentLiteral:
		c.require(v.Literal != nil && v.Uint64 == nil && v.Unavailable == nil, "INVALID_LITERAL_UNION", path)
		if v.Literal != nil {
			c.id(*v.Literal, path)
		}
		c.id(e.SourceFrameRef, path+".Frame")
	case ValueUnavailable:
		c.require(v.Uint64 == nil && v.Literal == nil && v.Unavailable != nil && e.SourceFrameRef == "", "INVALID_UNAVAILABLE_UNION", path)
		if v.Unavailable != nil {
			c.id(v.Unavailable.Code, path+".Reason")
			c.ids(v.Unavailable.MissingRefs, c.limits.MaxRefs, path+".Missing", false)
		}
		c.add(NeedsEvidence, "UNAVAILABLE_AXIS", path)
	default:
		c.add(RefusedUnchanged, "UNKNOWN_VALUE_KIND", path)
	}
}
func (c *checker) axisKey(k AxisKey, path string) {
	c.require(validAxis(k.Axis), "UNKNOWN_AXIS", path)
	c.id(k.OwnerID, path+".Owner")
	c.resource(k.Resource, path+".Resource")
	c.require(k.OwnerID == k.Resource.OwnerID, "AXIS_OWNER_RESOURCE_MISMATCH", path)
}
func ValidateBoundary(a BoundaryAddress, l Limits) Result {
	if r := ValidateLimits(l); r.Kind != StructurallyCoherent {
		return r
	}
	c := newChecker(l)
	c.id(a.OwnerID, "Boundary.Owner")
	c.resource(a.Resource, "Boundary.Resource")
	c.require(a.OwnerID == a.Resource.OwnerID, "BOUNDARY_OWNER_MISMATCH", "Boundary")
	valid := false
	switch a.Kind {
	case CPLifecycle:
		valid = member(a.SubBoundary, CPApply, ReceiptSave, CompleteACK, LocalParticipantACK)
	case Ledger:
		valid = member(a.SubBoundary, Reserve, Finalize, LedgerReceipt, Complete)
	case BackendMutation:
		valid = member(a.SubBoundary, ReferenceCommit, ServiceApply, MutationReceipt)
	case Provider:
		valid = member(a.SubBoundary, TransportResponse, RemoteAccept, RemoteResult, RemoteTerminal)
	case LocalPersistence:
		valid = member(a.SubBoundary, Intent, DispatchPossible, RawWrite, Durability, QueueAdmission, QueueTerminal, LocalDBCommit, LocalResultRecord, HistoryRecord)
	case ProjectionBoundary:
		valid = member(a.SubBoundary, CPOutcome, ExecutorOutcome, HistoryView, UIView)
	}
	c.require(valid, "UNKNOWN_BOUNDARY_PAIR", "Boundary")
	return c.finish()
}

func validWorkKind(k string) bool {
	return word(k, "BUSINESS MEDIA JOB PACKET CALLBACK WRITER CHILD QUEUE EFFECT RECORD")
}
