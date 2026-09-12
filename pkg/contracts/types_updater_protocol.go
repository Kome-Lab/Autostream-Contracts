package contracts

import (
	"time"
)

type UpdaterCapability string

const (
	UpdaterCapabilitySystemd    UpdaterCapability = "host.systemd"
	UpdaterCapabilityDocker     UpdaterCapability = "host.docker"
	UpdaterCapabilityUpdate     UpdaterCapability = "host.update"
	UpdaterCapabilityBootstrap  UpdaterCapability = "host.bootstrap"
	UpdaterCapabilityPort       UpdaterCapability = "host.port"
	UpdaterCapabilitySelfUpdate UpdaterCapability = "host.self_update"
)

// UpdaterDesiredOperationType is the closed operation vocabulary accepted by
// the independent Updater command protocol. It deliberately has no generic
// command, shell, argv, environment, path, URL, or credential escape hatch.
type UpdaterDesiredOperationType string

const (
	UpdaterDesiredSoftwareUpdate  UpdaterDesiredOperationType = "software_update"
	UpdaterDesiredBootstrap       UpdaterDesiredOperationType = "bootstrap"
	UpdaterDesiredPortReconfigure UpdaterDesiredOperationType = "port_reconfigure"
	UpdaterDesiredHostSelfUpdate  UpdaterDesiredOperationType = "host_self_update"
)

type UpdaterSoftwareUpdateDesiredOperation struct {
	ExpectedCurrentVersion string               `json:"expected_current_version"`
	TargetVersion          string               `json:"target_version"`
	Strategy               SystemUpdateStrategy `json:"strategy"`
}

type UpdaterBootstrapDesiredOperation struct {
	ExpectedState string `json:"expected_state"`
	TargetVersion string `json:"target_version"`
}

// UpdaterDesiredOperation is a discriminated union. Exactly one pointer must
// be non-nil and it must match Operation; the raw validator enforces this
// before the payload can lose presence information during Go decoding.
type UpdaterDesiredOperation struct {
	Operation       UpdaterDesiredOperationType            `json:"operation"`
	SoftwareUpdate  *UpdaterSoftwareUpdateDesiredOperation `json:"software_update,omitempty"`
	Bootstrap       *UpdaterBootstrapDesiredOperation      `json:"bootstrap,omitempty"`
	PortReconfigure *SystemUpdatePortReconfiguration       `json:"port_reconfigure,omitempty"`
	HostSelfUpdate  *HostAgentSelfUpdateDirective          `json:"host_self_update,omitempty"`
}

// UpdaterMutationOperation is the closed root Local Executor action performed
// under one opaque grant. It separates initial execution from reconciliation
// and separates self-update staging from activation.
type UpdaterMutationOperation string

const (
	UpdaterMutationApply                    UpdaterMutationOperation = "apply"
	UpdaterMutationReconcile                UpdaterMutationOperation = "reconcile"
	UpdaterMutationPortReconfigure          UpdaterMutationOperation = "port_reconfigure"
	UpdaterMutationPortReconfigureReconcile UpdaterMutationOperation = "port_reconfigure_reconcile"
	UpdaterMutationBootstrap                UpdaterMutationOperation = "bootstrap"
	UpdaterMutationBootstrapReconcile       UpdaterMutationOperation = "bootstrap_reconcile"
	UpdaterMutationHostSelfUpdateStage      UpdaterMutationOperation = "host_self_update_stage"
	UpdaterMutationHostSelfUpdateActivate   UpdaterMutationOperation = "host_self_update_activate"
	UpdaterMutationHostSelfUpdateReconcile  UpdaterMutationOperation = "host_self_update_reconcile"
)

type UpdaterOutcome string

const (
	UpdaterOutcomeSucceeded  UpdaterOutcome = "succeeded"
	UpdaterOutcomeFailed     UpdaterOutcome = "failed"
	UpdaterOutcomeRolledBack UpdaterOutcome = "rolled_back"
	UpdaterOutcomeAmbiguous  UpdaterOutcome = "ambiguous"
	UpdaterOutcomeCanceled   UpdaterOutcome = "canceled"
)

type V2UpdaterSafeError struct {
	Code               string `json:"code"`
	Message            string `json:"message"`
	Retryable          bool   `json:"retryable"`
	AuditCorrelationID string `json:"audit_correlation_id,omitempty"`
}

type ApplicationRuntimeIdentityProbe struct {
	Version        string                 `json:"version"`
	ServiceID      string                 `json:"service_id"`
	ServiceType    SystemUpdateTargetType `json:"service_type"`
	ConfigRevision int64                  `json:"config_revision"`
}

// SystemUpdateHostApplicationProbe is the host readiness projection. Status is
// not part of the application-owned four-field runtime identity probe.
type SystemUpdateHostApplicationProbe struct {
	Status string `json:"status"`
	ApplicationRuntimeIdentityProbe
}

type UpdaterHealth struct {
	Status        string `json:"status"`
	Revision      int64  `json:"revision"`
	SafeErrorCode string `json:"safe_error_code,omitempty"`
}

type UpdaterTargetKind string

const (
	UpdaterTargetApplication UpdaterTargetKind = "application"
	UpdaterTargetUpdateAgent UpdaterTargetKind = "update_agent"
	UpdaterTargetHostRuntime UpdaterTargetKind = "host_runtime"
)

type UpdaterTargetIdentity struct {
	TargetKind             UpdaterTargetKind          `json:"target_kind"`
	ServiceID              string                     `json:"service_id"`
	ServiceType            SystemUpdateTargetType     `json:"service_type"`
	DeploymentMode         SystemUpdateDeploymentMode `json:"deployment_mode"`
	ExpectedConfigRevision int64                      `json:"expected_config_revision,omitempty"`
	ExecutionHostID        string                     `json:"execution_host_id,omitempty"`
}

type UpdaterCommandIssuer struct {
	ServiceID      string `json:"service_id"`
	ServiceType    string `json:"service_type"`
	Authentication string `json:"authentication"`
	Permission     string `json:"permission"`
}

type UpdaterMutationAuthorization struct {
	AuthorizationID         string                `json:"authorization_id"`
	NonceID                 string                `json:"nonce_id"`
	JobID                   string                `json:"job_id"`
	UpdaterID               string                `json:"updater_id"`
	HostID                  string                `json:"host_id"`
	ActionType              UpdaterCapability     `json:"action_type"`
	Target                  UpdaterTargetIdentity `json:"target"`
	CanonicalArgumentDigest string                `json:"canonical_argument_digest"`
	DesiredRevision         int64                 `json:"desired_revision"`
	Fence                   int64                 `json:"fence"`
	ExpiresAt               time.Time             `json:"expires_at"`
	RequiredCapability      UpdaterCapability     `json:"required_capability"`
	OneTime                 bool                  `json:"one_time"`
}

type UpdaterCommandEnvelope struct {
	ProtocolVersion        int                          `json:"protocol_version"`
	CommandID              string                       `json:"command_id"`
	Issuer                 UpdaterCommandIssuer         `json:"issuer"`
	IdempotencyKey         string                       `json:"idempotency_key"`
	CanonicalPayloadDigest string                       `json:"canonical_payload_digest"`
	MutationAuthorization  UpdaterMutationAuthorization `json:"mutation_authorization"`
	DesiredOperation       UpdaterDesiredOperation      `json:"desired_operation"`
	AuditCorrelationID     string                       `json:"audit_correlation_id"`
}

type UpdaterLeaseEnvelope struct {
	ProtocolVersion int                    `json:"protocol_version"`
	LeaseID         string                 `json:"lease_id"`
	LeaseGeneration int64                  `json:"lease_generation"`
	LeaseExpiresAt  time.Time              `json:"lease_expires_at"`
	Command         UpdaterCommandEnvelope `json:"command"`
}

type UpdaterEvidence struct {
	EvidenceCode     string    `json:"evidence_code"`
	ObservedAt       time.Time `json:"observed_at"`
	ObservedRevision int64     `json:"observed_revision"`
	ArtifactDigest   string    `json:"artifact_digest,omitempty"`
}

type UpdaterProgressEnvelope struct {
	ProtocolVersion    int       `json:"protocol_version"`
	CommandID          string    `json:"command_id"`
	JobID              string    `json:"job_id"`
	UpdaterID          string    `json:"updater_id"`
	HostID             string    `json:"host_id"`
	LeaseID            string    `json:"lease_id"`
	LeaseGeneration    int64     `json:"lease_generation"`
	Sequence           int64     `json:"sequence"`
	Phase              string    `json:"phase"`
	Progress           int       `json:"progress"`
	DesiredRevision    int64     `json:"desired_revision"`
	Fence              int64     `json:"fence"`
	AuditCorrelationID string    `json:"audit_correlation_id"`
	ObservedAt         time.Time `json:"observed_at"`
}

type UpdaterResultEnvelope struct {
	ProtocolVersion        int                       `json:"protocol_version"`
	CommandID              string                    `json:"command_id"`
	JobID                  string                    `json:"job_id"`
	UpdaterID              string                    `json:"updater_id"`
	HostID                 string                    `json:"host_id"`
	LeaseID                string                    `json:"lease_id"`
	LeaseGeneration        int64                     `json:"lease_generation"`
	IdempotencyKey         string                    `json:"idempotency_key"`
	CanonicalPayloadDigest string                    `json:"canonical_payload_digest"`
	AuthorizationID        string                    `json:"authorization_id"`
	DesiredRevision        int64                     `json:"desired_revision"`
	AppliedRevision        int64                     `json:"applied_revision,omitempty"`
	Fence                  int64                     `json:"fence"`
	Outcome                UpdaterOutcome            `json:"outcome"`
	Status                 SystemUpdateStatus        `json:"status"`
	AutomaticResendAllowed bool                      `json:"automatic_resend_allowed"`
	AuditCorrelationID     string                    `json:"audit_correlation_id"`
	Evidence               []UpdaterEvidence         `json:"evidence"`
	SafeError              *V2UpdaterSafeError       `json:"safe_error,omitempty"`
	PortReconfigure        *SystemUpdatePortResultV2 `json:"port_reconfigure,omitempty"`
}

// UpdaterRuntimeTokenRotationCredentialClaimRequest is the only shared wire
// shape needed to claim a staged runtime credential. The raw replacement token
// is intentionally absent: it is returned only by the one-time no-store claim
// response and is forbidden in Updater commands, progress, results and local
// journal records.
type UpdaterRuntimeTokenRotationCredentialClaimRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	ClaimID          string `json:"claim_id"`
}

// UpdaterMutationGrantBinding is the complete credential-free request and
// consume binding for one root Local Executor mutation. Reusing the lease
// preserves the exact command, authorization, fence, JCS digest and desired
// plan without a second consumer-local representation.
type UpdaterMutationGrantBinding struct {
	Lease     UpdaterLeaseEnvelope     `json:"lease"`
	Operation UpdaterMutationOperation `json:"operation"`
	SessionID string                   `json:"session_id"`
}

type UpdaterMutationGrantIssueRequest struct {
	Binding UpdaterMutationGrantBinding `json:"binding"`
}

// UpdaterMutationGrantIssueResponse reuses the existing response-only opaque
// credential shape. The grant token must be emitted with Cache-Control:
// no-store and supplied to Local Executor only as its authorization proof.
type UpdaterMutationGrantIssueResponse = UpdateAgentMutationGrantIssueResponse

type UpdaterMutationGrantConsumeRequest struct {
	Binding UpdaterMutationGrantBinding `json:"binding"`
}

type UpdaterHeartbeat struct {
	ProtocolVersion         int                        `json:"protocol_version"`
	UpdaterID               string                     `json:"updater_id"`
	HostID                  string                     `json:"host_id"`
	ServiceID               string                     `json:"service_id"`
	Authentication          string                     `json:"authentication"`
	Sequence                int64                      `json:"sequence"`
	Capabilities            []UpdaterCapability        `json:"capabilities"`
	DesiredRevision         int64                      `json:"desired_revision"`
	AppliedRevision         int64                      `json:"applied_revision"`
	Fence                   int64                      `json:"fence"`
	Status                  string                     `json:"status"`
	ObservedAt              time.Time                  `json:"observed_at"`
	SafeError               *V2UpdaterSafeError        `json:"safe_error,omitempty"`
	PortContractVersion     int                        `json:"port_contract_version,omitempty"`
	PolicyTransitionVersion int                        `json:"policy_transition_version,omitempty"`
	PortPolicyBaseline      *UpdaterPortPolicyBaseline `json:"port_policy_baseline,omitempty"`
}

type UpdaterLocalJournalBoundary struct {
	JournalID              string    `json:"journal_id"`
	UpdaterID              string    `json:"updater_id"`
	HostID                 string    `json:"host_id"`
	CommandID              string    `json:"command_id"`
	AuthorizationID        string    `json:"authorization_id"`
	IdempotencyKey         string    `json:"idempotency_key"`
	CanonicalPayloadDigest string    `json:"canonical_payload_digest"`
	DesiredRevision        int64     `json:"desired_revision"`
	Fence                  int64     `json:"fence"`
	Phase                  string    `json:"phase"`
	RecoveryState          string    `json:"recovery_state"`
	RecordedAt             time.Time `json:"recorded_at"`
}

type SystemUpdateAgentStatus struct {
	ProtocolVersion                   int                 `json:"protocol_version"`
	UpdaterID                         string              `json:"updater_id"`
	HostID                            string              `json:"host_id"`
	ServiceID                         string              `json:"service_id"`
	Authentication                    string              `json:"authentication"`
	Name                              string              `json:"name"`
	TransportMode                     UpdateTransportMode `json:"transport_mode"`
	ExecutionHostID                   string              `json:"execution_host_id,omitempty"`
	OwnershipEpoch                    int64               `json:"ownership_epoch,omitempty"`
	Status                            string              `json:"status"`
	Online                            bool                `json:"online"`
	Version                           string              `json:"version"`
	LastHeartbeatAt                   *time.Time          `json:"last_heartbeat_at,omitempty"`
	HeartbeatSequence                 int64               `json:"heartbeat_sequence"`
	Capabilities                      []UpdaterCapability `json:"capabilities"`
	DesiredRevision                   int64               `json:"desired_revision"`
	AppliedRevision                   int64               `json:"applied_revision"`
	Fence                             int64               `json:"fence"`
	PolicyStatus                      string              `json:"policy_status,omitempty"`
	PolicyErrorCode                   string              `json:"policy_error_code,omitempty"`
	BootstrapEncryptionPublicKey      string              `json:"bootstrap_encryption_public_key,omitempty"`
	BootstrapEncryptionKeyFingerprint string              `json:"bootstrap_encryption_key_fingerprint,omitempty"`
	PortContractVersion               int                 `json:"port_contract_version,omitempty"`
	PolicyTransitionVersion           int                 `json:"policy_transition_version,omitempty"`
}

type SystemUpdateHostStatus struct {
	ProtocolVersion       int                               `json:"protocol_version"`
	HostID                string                            `json:"host_id"`
	Name                  string                            `json:"name"`
	UpdaterID             string                            `json:"updater_id"`
	Reachability          SystemUpdateReachability          `json:"reachability"`
	ReachabilityCheckedAt *time.Time                        `json:"reachability_checked_at,omitempty"`
	ReachabilityCode      string                            `json:"reachability_code,omitempty"`
	UpdaterHealth         *UpdaterHealth                    `json:"updater_health"`
	ApplicationProbe      *SystemUpdateHostApplicationProbe `json:"application_probe"`
}

type UpdateAgentClaimRequest struct {
	UpdaterID       string `json:"updater_id"`
	HostID          string `json:"host_id"`
	LeaseGeneration int64  `json:"lease_generation"`
	Fence           int64  `json:"fence"`
	ActiveJobID     string `json:"active_job_id,omitempty"`
}

type UpdateAgentClaimResponse struct {
	Job              SystemUpdateJob    `json:"job"`
	LeaseToken       string             `json:"lease_token"`
	LeaseExpiresAt   time.Time          `json:"lease_expires_at"`
	LeaseGeneration  int64              `json:"lease_generation"`
	ReportSequence   int64              `json:"report_sequence"`
	RecoveryRequired bool               `json:"recovery_required"`
	LastStatus       SystemUpdateStatus `json:"last_status"`
}

// UpdateAgentClearActiveJobResponse tells an updater that the active job it
// reported is terminal, missing, or no longer owned by that updater. The
// response is intentionally disjoint from UpdateAgentClaimResponse so a
// client never mistakes a clear instruction for a newly claimed job.
type UpdateAgentClearActiveJobResponse struct {
	ClearActiveJobID bool `json:"clear_active_job_id"`
}

type UpdateAgentReportRequest struct {
	UpdaterID       string                           `json:"updater_id"`
	HostID          string                           `json:"host_id"`
	LeaseToken      string                           `json:"lease_token"`
	LeaseGeneration int64                            `json:"lease_generation"`
	Fence           int64                            `json:"fence"`
	Sequence        int64                            `json:"sequence"`
	Status          SystemUpdateStatus               `json:"status"`
	Progress        int                              `json:"progress,omitempty"`
	Code            string                           `json:"code,omitempty"`
	Message         string                           `json:"message,omitempty"`
	ArtifactDigest  string                           `json:"artifact_digest,omitempty"`
	PreviousDigest  string                           `json:"previous_digest,omitempty"`
	PortReconfigure *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
}

type UpdateAgentMutationGrantIssueRequest struct {
	ServiceID       string                           `json:"service_id"`
	LeaseToken      string                           `json:"lease_token"`
	LeaseGeneration int64                            `json:"lease_generation"`
	ExecutionHostID string                           `json:"host_id"`
	TransportMode   UpdateTransportMode              `json:"transport_mode,omitempty"`
	OwnershipEpoch  int64                            `json:"ownership_epoch,omitempty"`
	PolicyRevision  int64                            `json:"policy_revision,omitempty"`
	TargetID        string                           `json:"target_id"`
	ServiceType     SystemUpdateTargetType           `json:"service_type,omitempty"`
	TargetVersion   string                           `json:"target_version"`
	DeploymentMode  SystemUpdateDeploymentMode       `json:"deployment_mode"`
	JobOperation    SystemUpdateOperation            `json:"job_operation,omitempty"`
	Operation       SystemUpdateMutationOperation    `json:"operation"`
	PlanSHA256      string                           `json:"plan_sha256"`
	SessionID       string                           `json:"session_id"`
	PortReconfigure *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
}

type UpdateAgentMutationGrantIssueResponse struct {
	GrantToken string    `json:"grant_token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type UpdateAgentMutationGrantConsumeRequest struct {
	LeaseGeneration int64                            `json:"lease_generation"`
	ExecutionHostID string                           `json:"host_id"`
	TransportMode   UpdateTransportMode              `json:"transport_mode,omitempty"`
	OwnershipEpoch  int64                            `json:"ownership_epoch,omitempty"`
	PolicyRevision  int64                            `json:"policy_revision,omitempty"`
	TargetID        string                           `json:"target_id"`
	ServiceType     SystemUpdateTargetType           `json:"service_type,omitempty"`
	TargetVersion   string                           `json:"target_version"`
	DeploymentMode  SystemUpdateDeploymentMode       `json:"deployment_mode"`
	JobOperation    SystemUpdateOperation            `json:"job_operation,omitempty"`
	Operation       SystemUpdateMutationOperation    `json:"operation"`
	PlanSHA256      string                           `json:"plan_sha256"`
	SessionID       string                           `json:"session_id"`
	PortReconfigure *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
}
