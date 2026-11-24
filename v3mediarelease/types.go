// Package mediarelease defines the MediaRelease1 wire contract and pure comparisons.
// Its records describe attributed evidence. Structural validity does not authenticate
// an issuer, persist a consumption, or authorize any external operation.
package mediarelease

const (
	Family                  = "MediaRelease1"
	WireVersion             = "1"
	EncodingID              = "MR1_JSON_ASCII_V1"
	ProfileID               = "MEDIA_RELEASE_HANDOFF_V1"
	ProfileRevision         = "1"
	CPAdmissionProof        = "CP_ADMISSION_PROOF"
	WorkerNonMediaBinding   = "WORKER_NON_MEDIA_BINDING"
	AppliedCoverProof       = "APPLIED_COVER_PROOF"
	MediaReleasePermit      = "MEDIA_RELEASE_PERMIT"
	MediaReleaseConsumption = "MEDIA_RELEASE_CONSUMPTION"
	MediaReleaseReceipt     = "MEDIA_RELEASE_RECEIPT"
)

type Domain struct {
	DeploymentID string `json:"deployment_id"`
	RealmID      string `json:"realm_id"`
	DomainID     string `json:"domain_id"`
}

type Peer struct {
	ComponentID string `json:"component_id"`
	ServiceID   string `json:"service_id"`
	ResourceID  string `json:"resource_id"`
}

type Owner struct {
	ComponentID        string `json:"component_id"`
	ServiceID          string `json:"service_id"`
	ResourceID         string `json:"resource_id"`
	HostID             string `json:"host_id"`
	BootID             string `json:"boot_id"`
	ProcessIncarnation string `json:"process_incarnation"`
	WriterID           string `json:"writer_id"`
	PrincipalID        string `json:"principal_id"`
}

type Key struct {
	DeploymentID string `json:"deployment_id"`
	OwnerID      string `json:"owner_id"`
	Kind         string `json:"kind"`
	RecordID     string `json:"record_id"`
}

type Original struct {
	RequestID    string `json:"request_id"`
	OperationID  string `json:"operation_id"`
	AttemptID    string `json:"attempt_id"`
	StepID       string `json:"step_id"`
	ExecutionKey string `json:"execution_key"`
	StreamID     string `json:"stream_id"`
	JobID        string `json:"job_id"`
	RunID        string `json:"run_id"`
}

type Ref struct {
	SourceFamily   string `json:"source_family"`
	SourceVersion  string `json:"source_version"`
	SourceType     string `json:"source_type"`
	SourceOwner    Peer   `json:"source_owner"`
	SourceKey      Key    `json:"source_key"`
	SourceRevision string `json:"source_revision" wire:"positive"`
	Representation string `json:"representation"`
	Digest         string `json:"digest" wire:"hash"`
}

type Assignment struct {
	AssignmentID   string `json:"assignment_id"`
	OwnershipEpoch string `json:"ownership_epoch" wire:"positive"`
	Assignee       Peer   `json:"assignee"`
	EndpointID     string `json:"endpoint_id"`
	ConfigRevision string `json:"config_revision" wire:"positive"`
	AssignmentRef  Ref    `json:"assignment_ref"`
}

type Window struct {
	NotBefore         string `json:"not_before" wire:"time"`
	ExpiresAt         string `json:"expires_at" wire:"time"`
	MaxClockErrorNS   string `json:"max_clock_error_ns" wire:"uint"`
	ClockAuthorityRef Ref    `json:"clock_authority_ref"`
}

type Scope struct {
	Route             string   `json:"route"`
	GateID            string   `json:"gate_id"`
	TargetOwner       Owner    `json:"target_owner"`
	DestinationID     string   `json:"destination_id"`
	EndpointRevision  string   `json:"endpoint_revision" wire:"positive"`
	Media             []string `json:"media"`
	EffectKind        string   `json:"effect_kind"`
	BoundaryID        string   `json:"boundary_id"`
	SuccessProfileRef Ref      `json:"success_profile_ref"`
}

type Closure struct {
	Scope           Scope  `json:"scope"`
	GraphID         string `json:"graph_id"`
	GraphGeneration string `json:"graph_generation" wire:"positive"`
	FenceRef        Ref    `json:"fence_ref"`
	IntervalStart   string `json:"interval_start" wire:"time"`
	ObservedThrough string `json:"observed_through" wire:"time"`
	FrameRef        Ref    `json:"frame_ref"`
}

type Obs struct {
	Scope      Scope  `json:"scope"`
	Fact       string `json:"fact"`
	Knowledge  string `json:"knowledge"`
	ObservedAt string `json:"observed_at" wire:"time"`
	Observer   Owner  `json:"observer"`
	FrameRef   *Ref   `json:"frame_ref,omitempty" optional:"true"`
	ReasonCode string `json:"reason_code,omitempty" optional:"true"`
}

func (o Owner) Peer() Peer {
	return Peer{ComponentID: o.ComponentID, ServiceID: o.ServiceID, ResourceID: o.ResourceID}
}

// Identity contains the immutable claims of exactly one record kind.
// Kind-specific fields are flattened on the wire. The kinds tag is the closed
// applicability table used by ValidateFields (a/w/b/c/d/e in contract order).
type Identity struct {
	Family          string   `json:"family"`
	WireVersion     string   `json:"wire_version"`
	EncodingID      string   `json:"encoding_id"`
	ProfileID       string   `json:"profile_id"`
	ProfileRevision string   `json:"profile_revision"`
	CC              string   `json:"cc"`
	Kind            string   `json:"kind"`
	RecordKey       Key      `json:"record_key"`
	Issuer          Owner    `json:"issuer"`
	Domain          Domain   `json:"domain"`
	Original        Original `json:"original"`
	Audience        []Peer   `json:"audience"`
	Purpose         string   `json:"purpose"`
	IssuedAt        string   `json:"issued_at" wire:"time"`
	ClockRef        Ref      `json:"clock_ref"`

	ClaimRef              *Ref        `json:"claim_ref,omitempty" kinds:"a"`
	EncoderAssignment     *Assignment `json:"encoder_assignment,omitempty" kinds:"a"`
	MembershipRef         *Ref        `json:"membership_ref,omitempty" kinds:"a"`
	MemberRevision        string      `json:"member_revision,omitempty" wire:"positive" kinds:"a"`
	CurrentFrameRefs      []Ref       `json:"current_frame_refs,omitempty" kinds:"a"`
	DomainClosureRef      *Ref        `json:"domain_closure_ref,omitempty" kinds:"a"`
	HoldRef               *Ref        `json:"hold_ref,omitempty" kinds:"a"`
	WorkerOriginalRef     *Ref        `json:"worker_original_ref,omitempty" kinds:"aw"`
	EncoderOriginalRef    *Ref        `json:"encoder_original_ref,omitempty" kinds:"aw"`
	WorkerAssignment      *Assignment `json:"worker_assignment,omitempty" kinds:"aw"`
	MembershipSnapshotRef *Ref        `json:"membership_snapshot_ref,omitempty" kinds:"aw"`
	ReservationRef        *Ref        `json:"reservation_ref,omitempty" kinds:"aw"`
	AdmissionEpoch        string      `json:"admission_epoch,omitempty" wire:"positive" kinds:"aw"`
	StopFenceRef          *Ref        `json:"stop_fence_ref,omitempty" kinds:"ac"`
	Window                *Window     `json:"window,omitempty" kinds:"awbc"`
	CPAdmissionRef        *Ref        `json:"cp_admission_ref,omitempty" kinds:"wbc"`
	ManagerProcessID      string      `json:"manager_process_id,omitempty" kinds:"w"`
	LocalBindingRef       *Ref        `json:"local_binding_ref,omitempty" kinds:"w"`
	NonMediaFenceRef      *Ref        `json:"non_media_fence_ref,omitempty" kinds:"w"`
	WorkerGeneration      string      `json:"worker_generation,omitempty" wire:"positive" kinds:"wb"`
	WorkerBindingRef      *Ref        `json:"worker_binding_ref,omitempty" kinds:"bc"`
	PreparationID         string      `json:"preparation_id,omitempty" kinds:"b"`
	PreparationGeneration string      `json:"preparation_generation,omitempty" wire:"positive" kinds:"b"`
	GraphID               string      `json:"graph_id,omitempty" kinds:"b"`
	GraphGeneration       string      `json:"graph_generation,omitempty" wire:"positive" kinds:"b"`
	GraphConfigRef        *Ref        `json:"graph_config_ref,omitempty" kinds:"b"`
	DesiredCoverRef       *Ref        `json:"desired_cover_ref,omitempty" kinds:"b"`
	RequestedRevision     string      `json:"requested_revision,omitempty" wire:"positive" kinds:"b"`
	AppliedRevision       string      `json:"applied_revision,omitempty" wire:"positive" kinds:"b"`
	AppliedActive         *bool       `json:"applied_active,omitempty" kinds:"b"`
	AssetRef              *Ref        `json:"asset_ref,omitempty" kinds:"b" optional:"true"`
	VariantID             string      `json:"variant_id,omitempty" kinds:"b" optional:"true"`
	HiddenObservationRef  *Ref        `json:"hidden_observation_ref,omitempty" kinds:"b" optional:"true"`
	FeedAcceptanceRef     *Ref        `json:"feed_acceptance_ref,omitempty" kinds:"b"`
	OutputProgressRef     *Ref        `json:"output_progress_ref,omitempty" kinds:"b"`
	QualifiedSinkRef      *Ref        `json:"qualified_sink_ref,omitempty" kinds:"b"`
	SourceFrameRef        *Ref        `json:"source_frame_ref,omitempty" kinds:"b"`
	Closures              []Closure   `json:"closures,omitempty" kinds:"b"`
	ObservedAt            string      `json:"observed_at,omitempty" wire:"time" kinds:"b"`
	PermitID              string      `json:"permit_id,omitempty" kinds:"c"`
	ReleaseAttemptID      string      `json:"release_attempt_id,omitempty" kinds:"c"`
	AppliedCoverRef       *Ref        `json:"applied_cover_ref,omitempty" kinds:"c"`
	ClosureRefs           []Ref       `json:"closure_refs,omitempty" kinds:"c"`
	AuthorityRef          *Ref        `json:"authority_ref,omitempty" kinds:"c"`
	RevocationRef         *Ref        `json:"revocation_ref,omitempty" kinds:"c"`
	LeaseRef              *Ref        `json:"lease_ref,omitempty" kinds:"c"`
	ConsumptionKey        *Key        `json:"consumption_key,omitempty" kinds:"c"`
	SlotReservationRef    *Ref        `json:"slot_reservation_ref,omitempty" kinds:"c"`
	LogicalEffectKey      string      `json:"logical_effect_key,omitempty" kinds:"cde"`
	ConnectionOperationID string      `json:"connection_operation_id,omitempty" kinds:"cd"`
	ReceiptKey            *Key        `json:"receipt_key,omitempty" kinds:"cd"`
	ExpectedSlotRevision  string      `json:"expected_slot_revision,omitempty" wire:"positive" kinds:"cd"`
	Scope                 *Scope      `json:"scope,omitempty" kinds:"ce"`
	PermitRef             *Ref        `json:"permit_ref,omitempty" kinds:"de"`
	OpenRequestDigest     string      `json:"open_request_digest,omitempty" wire:"hash" kinds:"d"`
	ConsumptionRef        *Ref        `json:"consumption_ref,omitempty" kinds:"e"`
	StoreRecordID         string      `json:"store_record_id,omitempty" kinds:"e"`
}

// Snapshot holds a single immutable revision of the attributed observations.
// Kind-specific fields are flattened on the wire. The kinds tag is the closed
// applicability table used by ValidateFields (a/w/b/c/d/e in contract order).
type Snapshot struct {
	Revision       string `json:"revision" wire:"positive"`
	RecordedAt     string `json:"recorded_at" wire:"time"`
	PreviousDigest string `json:"previous_digest,omitempty" wire:"hash" optional:"true"`
	DiagnosticRef  *Ref   `json:"diagnostic_ref,omitempty" optional:"true"`

	ConsumptionState       string `json:"consumption_state,omitempty" kinds:"de"`
	EffectState            string `json:"effect_state,omitempty" kinds:"de"`
	Observations           []Obs  `json:"observations,omitempty" kinds:"de"`
	SlotRevision           string `json:"slot_revision,omitempty" wire:"positive" kinds:"d"`
	FreshnessRef           *Ref   `json:"freshness_ref,omitempty" kinds:"d"`
	AuthorizationRefs      []Ref  `json:"authorization_refs,omitempty" kinds:"d" optional:"true"`
	DispatchRef            *Ref   `json:"dispatch_ref,omitempty" kinds:"d" optional:"true"`
	WorkRegistrationRef    *Ref   `json:"work_registration_ref,omitempty" kinds:"d" optional:"true"`
	CheckedAt              string `json:"checked_at,omitempty" wire:"time" kinds:"d" optional:"true"`
	ConsumeCommitID        string `json:"consume_commit_id,omitempty" kinds:"d" optional:"true"`
	StopObservationRef     *Ref   `json:"stop_observation_ref,omitempty" kinds:"d" optional:"true"`
	RevokeObservationRef   *Ref   `json:"revoke_observation_ref,omitempty" kinds:"d" optional:"true"`
	FirstEffectRef         *Ref   `json:"first_effect_ref,omitempty" kinds:"d" optional:"true"`
	ReceiptRef             *Ref   `json:"receipt_ref,omitempty" kinds:"d" optional:"true"`
	ObservationTime        string `json:"observation_time,omitempty" wire:"time" kinds:"e"`
	ObservedConsumptionRef *Ref   `json:"observed_consumption_ref,omitempty" kinds:"e"`
	DecisionRefs           []Ref  `json:"decision_refs,omitempty" kinds:"e"`
	ResultEvidenceRefs     []Ref  `json:"result_evidence_refs,omitempty" kinds:"e" optional:"true"`
}

// Record is the sealed wire envelope. A digest identifies bytes, not a principal.
type Record struct {
	Identity        Identity `json:"identity"`
	Snapshot        Snapshot `json:"snapshot"`
	ImmutableDigest string   `json:"immutable_digest" wire:"hash"`
	RecordDigest    string   `json:"record_digest" wire:"hash"`
}

type InspectRequest struct {
	LookupKey        Key      `json:"lookup_key"`
	Original         Original `json:"original"`
	Owner            Owner    `json:"owner"`
	ExpectedRevision string   `json:"expected_revision,omitempty" wire:"positive" optional:"true"`
	ExpectedDigest   string   `json:"expected_digest,omitempty" wire:"hash" optional:"true"`
}

type InspectResult struct {
	LookupKey         Key      `json:"lookup_key"`
	Original          Original `json:"original"`
	Responder         Owner    `json:"responder"`
	ObservedAt        string   `json:"observed_at" wire:"time"`
	Status            string   `json:"status"`
	Currentness       string   `json:"currentness"`
	FoundRef          *Ref     `json:"found_ref,omitempty" optional:"true"`
	StorageWitnessRef *Ref     `json:"storage_witness_ref,omitempty" optional:"true"`
	FreshnessRef      *Ref     `json:"freshness_ref,omitempty" optional:"true"`
	ReasonRefs        []Ref    `json:"reason_refs,omitempty" optional:"true"`
}

type FenceObservation struct {
	Original     Original `json:"original"`
	Scope        Scope    `json:"scope"`
	Responder    Owner    `json:"responder"`
	ObservedAt   string   `json:"observed_at" wire:"time"`
	Observations []Obs    `json:"observations"`
	PermitRef    *Ref     `json:"permit_ref,omitempty" optional:"true"`
	AuthorityRef *Ref     `json:"authority_ref,omitempty" optional:"true"`
	SlotKey      *Key     `json:"slot_key,omitempty" optional:"true"`
	DecisionRef  *Ref     `json:"decision_ref,omitempty" optional:"true"`
}
