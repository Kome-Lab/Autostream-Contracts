package lifecycle

type ContractID string

const (
	CC01 ContractID = "CC01"
	CC02 ContractID = "CC02"
	CC03 ContractID = "CC03"
	CC04 ContractID = "CC04"
	CC05 ContractID = "CC05"
	CC06 ContractID = "CC06"
	CC07 ContractID = "CC07"
	CC08 ContractID = "CC08"
	CC09 ContractID = "CC09"
)

type ResultKind string

const (
	StructurallyCoherent ResultKind = "STRUCTURALLY_COHERENT"
	NeedsEvidence        ResultKind = "NEEDS_EVIDENCE"
	NeedsDecision        ResultKind = "NEEDS_DECISION"
	RefusedUnchanged     ResultKind = "REFUSED_UNCHANGED"
	PendingObligation    ResultKind = "PENDING_OBLIGATION"
	Unknown              ResultKind = "UNKNOWN"
	Conflict             ResultKind = "CONFLICT"
	Stale                ResultKind = "STALE"
)

type Issue struct {
	Code string
	Path string
}
type Result struct {
	Kind   ResultKind
	Issues []Issue
}

// ProductionQualified is deliberately false even for structurally valid claims.
func (Result) ProductionQualified() bool { return false }
func (Result) AuthorizesMutation() bool  { return false }
func result(k ResultKind, code, path string) Result {
	r := Result{Kind: k}
	if code != "" {
		r.Issues = []Issue{{Code: code, Path: path}}
	}
	return r
}
func coherent() Result { return result(StructurallyCoherent, "", "") }

type Knowledge string

const (
	Known             Knowledge = "KNOWN"
	KnowledgeUnknown  Knowledge = "UNKNOWN"
	KnowledgeConflict Knowledge = "CONFLICT"
)

type Presence string

const (
	Present     Presence = "PRESENT"
	Absent      Presence = "ABSENT"
	NotObserved Presence = "NOT_OBSERVED"
)

type Gate string

const (
	Held               Gate = "HELD"
	NotQualified       Gate = "NOT_QUALIFIED"
	VerifiedForProfile Gate = "VERIFIED_FOR_PROFILE"
)

type FactState string

const (
	Observed    FactState = "OBSERVED"
	Unavailable FactState = "UNAVAILABLE"
)

type ValueKind string

const (
	PresentUint64    ValueKind = "PRESENT_UINT64"
	PresentLiteral   ValueKind = "PRESENT_LITERAL"
	ValueUnavailable ValueKind = "UNAVAILABLE"
)

type ResourceKind string

const (
	User                ResourceKind = "USER"
	Role                ResourceKind = "ROLE"
	RoleAssignment      ResourceKind = "ROLE_ASSIGNMENT"
	Session             ResourceKind = "SESSION"
	IdentityLink        ResourceKind = "IDENTITY_LINK"
	AuthProvider        ResourceKind = "AUTH_PROVIDER"
	AuthPolicy          ResourceKind = "AUTH_POLICY"
	Factor              ResourceKind = "FACTOR"
	Passkey             ResourceKind = "PASSKEY"
	RecoveryMetadata    ResourceKind = "RECOVERY_METADATA"
	WorkloadSubject     ResourceKind = "WORKLOAD_SUBJECT"
	Approval            ResourceKind = "APPROVAL"
	TrustRootRef        ResourceKind = "TRUST_ROOT_REF"
	LedgerRef           ResourceKind = "LEDGER_REF"
	ClockRef            ResourceKind = "CLOCK_REF"
	GrantRef            ResourceKind = "GRANT_REF"
	KeyMetadata         ResourceKind = "KEY_METADATA"
	SecretMetadata      ResourceKind = "SECRET_METADATA"
	Stream              ResourceKind = "STREAM"
	Run                 ResourceKind = "RUN"
	Node                ResourceKind = "NODE"
	Job                 ResourceKind = "JOB"
	Connection          ResourceKind = "CONNECTION"
	Host                ResourceKind = "HOST"
	Partition           ResourceKind = "PARTITION"
	Sink                ResourceKind = "SINK"
	Artifact            ResourceKind = "ARTIFACT"
	MediaAsset          ResourceKind = "MEDIA_ASSET"
	Archive             ResourceKind = "ARCHIVE"
	Queue               ResourceKind = "QUEUE"
	Incident            ResourceKind = "INCIDENT"
	NotificationChannel ResourceKind = "NOTIFICATION_CHANNEL"
	Config              ResourceKind = "CONFIG"
	Release             ResourceKind = "RELEASE"
	BackupPoint         ResourceKind = "BACKUP_POINT"
	ReturnPoint         ResourceKind = "RETURN_POINT"
	Projection          ResourceKind = "PROJECTION"
)

type Axis string

const (
	ARevision      Axis = "A_REVISION"
	CEpoch         Axis = "C_EPOCH"
	Identity       Axis = "IDENTITY"
	Membership     Axis = "MEMBERSHIP"
	Admission      Axis = "ADMISSION"
	Ownership      Axis = "OWNERSHIP"
	Fence          Axis = "FENCE"
	RunAxis        Axis = "RUN"
	JobAxis        Axis = "JOB"
	ConnectionAxis Axis = "CONNECTION"
	Writer         Axis = "WRITER"
	Effect         Axis = "EFFECT"
	Cover          Axis = "COVER"
	Policy         Axis = "POLICY"
	Trust          Axis = "TRUST"
	Boot           Axis = "BOOT"
	Process        Axis = "PROCESS"
	ReleaseAxis    Axis = "RELEASE"
	ConfigAxis     Axis = "CONFIG"
)

type ProfileID string

const (
	ApprovalLocal  ProfileID = "G0-P-APPROVAL-LOCAL"
	MediaIngress   ProfileID = "G0-P-MEDIA-INGRESS"
	ClosureAssess  ProfileID = "G0-P-CLOSURE-ASSESS"
	StopLookup     ProfileID = "G0-P-STOP-LOOKUP"
	EffectOutcome  ProfileID = "G0-P-EFFECT-OUTCOME"
	RecordRepair   ProfileID = "G0-P-RECORD-REPAIR"
	ExplicitResume ProfileID = "G0-P-EXPLICIT-RESUME"
)

type BoundaryKind string

const (
	CPLifecycle        BoundaryKind = "CP_LIFECYCLE"
	Ledger             BoundaryKind = "LEDGER"
	BackendMutation    BoundaryKind = "BACKEND_MUTATION"
	Provider           BoundaryKind = "PROVIDER"
	LocalPersistence   BoundaryKind = "LOCAL_PERSISTENCE"
	ProjectionBoundary BoundaryKind = "PROJECTION"
)

type SubBoundary string

const (
	CPApply             SubBoundary = "CP_APPLY"
	ReceiptSave         SubBoundary = "RECEIPT_SAVE"
	CompleteACK         SubBoundary = "COMPLETE_ACK"
	LocalParticipantACK SubBoundary = "LOCAL_PARTICIPANT_ACK"
	Reserve             SubBoundary = "RESERVE"
	Finalize            SubBoundary = "FINALIZE"
	LedgerReceipt       SubBoundary = "RECEIPT"
	Complete            SubBoundary = "COMPLETE"
	ReferenceCommit     SubBoundary = "REFERENCE_COMMIT"
	ServiceApply        SubBoundary = "SERVICE_APPLY"
	MutationReceipt     SubBoundary = "MUTATION_RECEIPT"
	TransportResponse   SubBoundary = "TRANSPORT_RESPONSE"
	RemoteAccept        SubBoundary = "REMOTE_ACCEPT"
	RemoteResult        SubBoundary = "REMOTE_RESULT"
	RemoteTerminal      SubBoundary = "REMOTE_TERMINAL"
	Intent              SubBoundary = "INTENT"
	DispatchPossible    SubBoundary = "DISPATCH_POSSIBLE"
	RawWrite            SubBoundary = "RAW_WRITE"
	Durability          SubBoundary = "DURABILITY"
	QueueAdmission      SubBoundary = "QUEUE_ADMISSION"
	QueueTerminal       SubBoundary = "QUEUE_TERMINAL"
	LocalDBCommit       SubBoundary = "LOCAL_DB_COMMIT"
	LocalResultRecord   SubBoundary = "LOCAL_RESULT_RECORD"
	HistoryRecord       SubBoundary = "HISTORY_RECORD"
	CPOutcome           SubBoundary = "CP_OUTCOME"
	ExecutorOutcome     SubBoundary = "EXECUTOR_OUTCOME"
	HistoryView         SubBoundary = "HISTORY_VIEW"
	UIView              SubBoundary = "UI_VIEW"
)

type Outcome string

const (
	SuccessAtBoundary Outcome = "SUCCESS_AT_BOUNDARY"
	FailureAtBoundary Outcome = "FAILURE_AT_BOUNDARY"
	NoEffectProven    Outcome = "NO_EFFECT_PROVEN"
	Pending           Outcome = "PENDING"
	OutcomeUnknown    Outcome = "UNKNOWN"
)

type ObligationKind string

const (
	FD              ObligationKind = "FD"
	DBConnection    ObligationKind = "DB_CONNECTION"
	Child           ObligationKind = "CHILD"
	Callback        ObligationKind = "CALLBACK"
	WriterBorrow    ObligationKind = "WRITER_BORROW"
	QueueObligation ObligationKind = "QUEUE"
	Lineage         ObligationKind = "LINEAGE"
)

type AdmissionLane string

const (
	Open    AdmissionLane = "OPEN"
	Closing AdmissionLane = "CLOSING"
	Closed  AdmissionLane = "CLOSED"
)

type DrainState string

const (
	Waiting DrainState = "WAITING"
	Proved  DrainState = "PROVED"
)

type DispatchState string

const (
	NoIntent       DispatchState = "NO_INTENT"
	IntentRecorded DispatchState = "INTENT_RECORDED"
	MayDispatch    DispatchState = "DISPATCH_POSSIBLE"
)

type Purpose string

const (
	PurposeApproval       Purpose = "APPROVAL_LOCAL"
	PurposeIngress        Purpose = "MEDIA_INGRESS"
	PurposeClosure        Purpose = "CLOSURE_ASSESS"
	PurposeLookup         Purpose = "STOP_LOOKUP"
	PurposeEffect         Purpose = "EFFECT_OUTCOME"
	PurposeRepair         Purpose = "RECORD_PROJECTION_ONLY"
	PurposeResume         Purpose = "EXPLICIT_RESUME"
	PurposeRecovery       Purpose = "RECOVERY"
	PurposeRecordTransfer Purpose = "RECORD_TRANSFER"
)

type IdentityClass string

const (
	Human           IdentityClass = "HUMAN"
	PendingIdentity IdentityClass = "PENDING"
	Preauth         IdentityClass = "PREAUTH"
	System          IdentityClass = "SYSTEM"
)

type PrincipalKind string

const (
	PrincipalHuman   PrincipalKind = "HUMAN"
	PrincipalPreauth PrincipalKind = "PREAUTH"
	PrincipalSystem  PrincipalKind = "SYSTEM"
)
