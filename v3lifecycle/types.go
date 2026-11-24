package lifecycle

const LogicalRevision uint64 = 1
const SpecRef = "G0-LOGICAL-20261115-R1"

type ID string
type ComponentID string
type Limits struct{ MaxIDBytes, MaxMembers, MaxAxes, MaxRefs, MaxObligations, MaxPartitions, MaxRanges, MaxDepth int }
type ContractRef struct {
	ContractID      ContractID
	LogicalRevision uint64
	SpecRef         ID
	ModuleCommitRef ID
}
type RecordKey struct {
	ContractRef ContractRef
	OwnerID     ID
	RecordID    ID
}
type Correlation struct{ RequestID, OperationID, AttemptID, StepID, ExecutionKey ID }
type ResourceRef struct {
	Kind                ResourceKind
	OwnerID, ResourceID ID
}
type DomainRef struct {
	DeploymentID, Realm, DomainID ID
	MembershipRevision            uint64
	SnapshotRef                   ID
	Resources                     []ResourceRef
}
type ProducerRef struct {
	ComponentID                                             ComponentID
	ServiceID, HostID, BootID, ProcessIncarnation, WriterID ID
}
type ProfileRef struct {
	ProfileID ProfileID
	Revision  uint64
}
type Header struct {
	ContractRef    ContractRef
	RecordKey      RecordKey
	Correlation    Correlation
	DomainRef      DomainRef
	ProducerRef    ProducerRef
	RecordRevision uint64
	ProfileRef     ProfileRef
}
type IdentityContext struct {
	Class           IdentityClass
	PrincipalKind   PrincipalKind
	ActorID         ID
	Purpose         Purpose
	FlowRef         ID
	WorkloadSubject ID
}
type SourceFrame struct {
	FrameRef, OwnerID, Scope ID
	SourceRevision           uint64
}
type ObservationContext struct {
	OwnerID, FrameRef, Scope ID
	SourceRevision           uint64
}
type UnavailableReason struct {
	Code        ID
	MissingRefs []ID
}
type Fact[T any] struct {
	State       FactState
	Value       *T
	Observation *ObservationContext
	Unavailable *UnavailableReason
}
type VersionedValue struct {
	Kind        ValueKind
	Uint64      *uint64
	Literal     *ID
	Unavailable *UnavailableReason
}
type AxisKey struct {
	Axis     Axis
	OwnerID  ID
	Resource ResourceRef
}
type RevisionEntry struct {
	Key            AxisKey
	Incarnation    ID
	Value          VersionedValue
	SourceFrameRef ID
}
type RevisionVector []RevisionEntry
type AxisRule struct {
	Key        AxisKey
	OwnerRole  ComponentID
	ValueKind  ValueKind
	Required   bool
	Comparison string
}
type ComparisonProfile struct {
	Ref                          ProfileRef
	AxisRules                    []AxisRule
	ResourceKinds                []ResourceKind
	Purposes                     []Purpose
	RequiredBoundaries           []BoundaryAddress
	TerminalPolicyRef, LimitsRef ID
	Limits                       Limits
}
type BoundaryAddress struct {
	Kind        BoundaryKind
	OwnerID     ID
	Resource    ResourceRef
	SubBoundary SubBoundary
}
type BoundaryObservation struct {
	Address   BoundaryAddress
	Knowledge Knowledge
	Presence  Presence
	Outcome   Fact[Outcome]
	DetailRef ID
	Details   *BoundaryDetails
}
type PhysicalObligation struct {
	ObligationID ID
	Kind         ObligationKind
	OwnerID      ID
	Resource     ResourceRef
	WorkRef      ID
	Knowledge    Knowledge
	Terminal     Fact[ID]
}
type ObservationSet struct {
	Observations         []BoundaryObservation
	PhysicalObligations  []PhysicalObligation
	RemainingObligations []ID
	Contradictions       []ID
	PhaseHints           []BoundaryAddress
	Gate                 Gate
}
type ContentRef struct {
	Kind string
	Ref  ID
} // NONSECRET_METADATA_REF only; never payload or secret-derived commitment.
type OptionalRef struct {
	State          Presence
	Ref            *ID
	AbsenceFactRef ID
	Observation    *ObservationContext
	Unavailable    *UnavailableReason
}
type ReservationRefs struct {
	DomainClosureRef ID
	CPReservation    OptionalRef
}
type Member struct {
	Producer       ProducerRef
	Resource       ResourceRef
	SourceFrameRef ID
}
type Alias struct {
	AliasID           ID
	Resource          ResourceRef
	ResolutionFactRef ID
}
type MembershipRecord struct {
	DomainKind       string
	OwnerID          ID
	Revision         uint64
	PreviousSnapshot OptionalRef
	RequiredAliasIDs []ID
	CoverageKind     string
	Members          []Member
	Aliases          []Alias
	AffectedRuns     []ResourceRef
	Coverage         Fact[ID]
	SourceFrames     []SourceFrame
}
type VectorRecord struct{ Vector RevisionVector }
type AdmissionRecord struct {
	ReservationRefs    ReservationRefs
	MembershipSnapshot ID
	AdmissionEpoch     ID
	ActorGuardRefs     []ID
	Cut                []Frontier
	TerminalPolicyRef  ID
}
type Namespace struct {
	OwnerID                            ID
	Resource                           ResourceRef
	WorkKind                           string
	ProducerIncarnation, SequenceEpoch ID
}
type Frontier struct {
	Kind         string
	Namespace    Namespace
	EmptyFactRef ID
	LastSequence *uint64
} // EMPTY or THROUGH, sequence >=1.
type SequenceRange struct{ First, Last uint64 }
type RegistrationClosure struct {
	MembershipSnapshot, AdmissionEpoch         ID
	Namespace                                  Namespace
	RegistrationEpoch, OwnerID, DurableFactRef ID
}
type PartitionProof struct {
	Namespace                                            Namespace
	CutFrontier, FinalAcceptedFrontier, TerminalFrontier Frontier
	Holes                                                []SequenceRange
	PendingRegistrationRefs                              []ID
	RegistrationClosed                                   RegistrationClosure
}
type WorkRef struct {
	WorkID, ParentWorkID ID
	Namespace            Namespace
	Sequence             uint64
	AdmissionRecordRef   ID
}
type CompletionRecord struct {
	ClosureID           ID
	ReservationRefs     ReservationRefs
	SourceVector        RevisionVector
	Partitions          []PartitionProof
	PhysicalObligations []PhysicalObligation
	Effects             []ID
	RequiredEffects     []EffectRequirement
	LineageClosureFacts []RegistrationClosure
	TerminalPolicyRef   ID
	ProofRefs           []ID
}
type ProofRef struct {
	Ref                ID
	ClosureID          ID
	MembershipSnapshot ID
	WorkPhase          string
} // PRIOR only.
type SealRecord struct {
	SealID, ClosureID                                          ID
	ReservationRefs                                            ReservationRefs
	SourceVector                                               RevisionVector
	ProofSet                                                   []ProofRef
	ReleaseRef, ConfigRef, PolicyRef, TrustRef                 ID
	ReleaseAxisKey, ConfigAxisKey, PolicyAxisKey, TrustAxisKey AxisKey
}
type FenceRecord struct {
	AuthorityConsumptionRef, CanonicalRef, SealRef         ID
	ExpectedTargetVector                                   RevisionVector
	Target                                                 ResourceRef
	OwnershipEpoch, FenceEpoch, DecisionTimeRef, BudgetRef ID
	BackendOperation, DeadlineSemantics                    string
}
type EffectRecord struct {
	EffectID       ID
	ParentEffect   OptionalRef
	Target         ResourceRef
	Purpose        Purpose
	ContentRef     ContentRef
	SourceVector   RevisionVector
	IntentRevision uint64
	SinkRefs       []ResourceRef
	ProjectionRefs []ID
}
type QueryTarget struct {
	Correlation                                      Correlation
	BindingRef, ExecutionKey, TargetRecordRef, Scope ID
	Purpose                                          Purpose
}
type InspectionRecord struct {
	Target         QueryTarget
	Observations   ObservationSet
	ObservedScopes []ID
	SourceFrames   []SourceFrame
	Coverage       Fact[ID]
	HistoricalRefs []ID
}
type MetadataDelta struct {
	Kind                          string
	SourceReceiptRef, MetadataRef ID
} // FIXED_RECEIPT_REFERENCE only.
type ReconcileRecord struct {
	PlanID                               ID
	OriginalCorrelation                  Correlation
	Purpose                              Purpose
	SourceReceiptRefs, SourceWitnessRefs []ID
	CurrentSourceVector                  RevisionVector
	ExpectedProjection                   ResourceRef
	ExpectedProjectionRevision           uint64
	IntendedMetadataDelta                MetadataDelta
	RequiredRecordCapability             ID
	PlanRevision                         uint64
	Freshness                            Fact[ID]
}

// Record is a closed tagged union. Exactly one payload must match Header.ContractRef.ContractID.
// No private handle, credential, signed authority, or raw content can be represented.
type Record struct {
	Header     Header
	Membership *MembershipRecord
	Vector     *VectorRecord
	Admission  *AdmissionRecord
	Completion *CompletionRecord
	Seal       *SealRecord
	Fence      *FenceRecord
	Effect     *EffectRecord
	Inspection *InspectionRecord
	Reconcile  *ReconcileRecord
}

type WriteExtent struct {
	ExtentRef                       ID
	Offset, Length, RequestedLength *uint64
}
type DurabilityDetail struct {
	ProfileRef ProfileRef
	Level      string
	WitnessRef ID
}
type QueueDetail struct {
	Work       WorkRef
	ReceiptRef ID
}
type BoundaryDetails struct {
	RawExtent  *WriteExtent
	Durability *DurabilityDetail
	Queue      *QueueDetail
}

type EffectRequirement struct {
	EffectID    ID
	Target      ResourceRef
	Correlation Correlation
}
