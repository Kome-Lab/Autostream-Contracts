package contracts

import (
	"time"
)

type RemediationMode string

const (
	RemediationDisabled       RemediationMode = "disabled"
	RemediationSuggestOnly    RemediationMode = "suggest_only"
	RemediationSafeAuto       RemediationMode = "safe_auto"
	RemediationManualApproval RemediationMode = "manual_approval"
)

type RemediationAction struct {
	ID               string          `json:"id"`
	IncidentID       string          `json:"incident_id"`
	Action           string          `json:"action"`
	Mode             RemediationMode `json:"mode"`
	Status           string          `json:"status"`
	SafeAuto         bool            `json:"safe_auto"`
	RequiresApproval bool            `json:"requires_approval"`
	Result           string          `json:"result,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ExecutedAt       *time.Time      `json:"executed_at,omitempty"`
}

// RemediationExecutionAuthority identifies the only two principals that may
// execute an authorized remediation. Observability deliberately has no value.
type RemediationExecutionAuthority string

const (
	RemediationAuthorityUpdater           RemediationExecutionAuthority = "updater"
	RemediationAuthorityTargetApplication RemediationExecutionAuthority = "target_application"
)

type RemediationExecutionScope string

const (
	RemediationScopeHostSystem       RemediationExecutionScope = "host_system"
	RemediationScopeApplicationLocal RemediationExecutionScope = "application_local"
)

type RemediationActionType string

const (
	RemediationActionHostSystemd          RemediationActionType = "host.systemd"
	RemediationActionHostDocker           RemediationActionType = "host.docker"
	RemediationActionHostUpdate           RemediationActionType = "host.update"
	RemediationActionHostBootstrap        RemediationActionType = "host.bootstrap"
	RemediationActionHostPort             RemediationActionType = "host.port"
	RemediationActionHostSelfUpdate       RemediationActionType = "host.self_update"
	RemediationActionRetryGDriveUpload    RemediationActionType = "application.retry_gdrive_upload"
	RemediationActionRetryPackageRemux    RemediationActionType = "application.retry_package_remux"
	RemediationActionRefreshServiceStatus RemediationActionType = "refresh_service_status"
	RemediationActionRerunDiagnostics     RemediationActionType = "rerun_diagnostics"
	RemediationActionClearStaleWarning    RemediationActionType = "clear_stale_warning"
)

type RemediationResult string

const (
	RemediationResultSucceeded  RemediationResult = "succeeded"
	RemediationResultFailed     RemediationResult = "failed"
	RemediationResultRolledBack RemediationResult = "rolled_back"
	RemediationResultAmbiguous  RemediationResult = "ambiguous"
)

type RemediationExecutorIdentity struct {
	ServiceID      string                        `json:"service_id,omitempty"`
	Authority      RemediationExecutionAuthority `json:"authority"`
	ExecutionScope RemediationExecutionScope     `json:"execution_scope"`
}

type RemediationTargetIdentity struct {
	ServiceID   string                 `json:"service_id"`
	ServiceType SystemUpdateTargetType `json:"service_type"`
	HostID      string                 `json:"host_id,omitempty"`
	IncidentID  string                 `json:"incident_id,omitempty"`
}

type RemediationDetectorIdentity struct {
	ServiceID   string      `json:"service_id"`
	ServiceType ServiceType `json:"service_type"`
}

type RemediationRequestOrigin struct {
	OriginType  string                 `json:"origin_type"`
	PrincipalID string                 `json:"principal_id"`
	ServiceID   string                 `json:"service_id,omitempty"`
	ServiceType SystemUpdateTargetType `json:"service_type,omitempty"`
	Permission  string                 `json:"permission"`
}

// RemediationEvidence is deliberately bounded and secret-safe. Callers must
// not place raw logs, payloads, credentials, configuration, or command output
// into any field.
type RemediationEvidence struct {
	EvidenceCode     string    `json:"evidence_code"`
	ObservedAt       time.Time `json:"observed_at"`
	ObservedRevision int64     `json:"observed_revision"`
	EvidenceDigest   string    `json:"evidence_digest,omitempty"`
}

// RemediationProposal is Observability's non-executable proposal. Control
// Panel authorization is always required before any cross-service execution.
type RemediationProposal struct {
	ProposalID                        string                      `json:"proposal_id"`
	IncidentID                        string                      `json:"incident_id"`
	Detector                          RemediationDetectorIdentity `json:"detector"`
	Target                            RemediationTargetIdentity   `json:"target"`
	ActionType                        RemediationActionType       `json:"action_type"`
	ProposalRevision                  int64                       `json:"proposal_revision"`
	RequiredCapability                RemediationActionType       `json:"required_capability"`
	Evidence                          []RemediationEvidence       `json:"evidence"`
	AuditCorrelationID                string                      `json:"audit_correlation_id"`
	ObservedAt                        time.Time                   `json:"observed_at"`
	ControlPanelAuthorizationRequired bool                        `json:"control_panel_authorization_required"`
}

// RemediationGrant is the bounded authority minted and audited by Control
// Panel. It cannot represent arbitrary shell, credentials, or an Observability
// executor.
type RemediationGrant struct {
	AuthorizationID         string                      `json:"authorization_id"`
	AuthorizationNonceID    string                      `json:"authorization_nonce_id"`
	ProposalID              string                      `json:"proposal_id"`
	RequestOrigin           RemediationRequestOrigin    `json:"request_origin"`
	Executor                RemediationExecutorIdentity `json:"executor"`
	Target                  RemediationTargetIdentity   `json:"target"`
	ActionType              RemediationActionType       `json:"action_type"`
	IdempotencyKey          string                      `json:"idempotency_key"`
	CanonicalArgumentDigest string                      `json:"canonical_argument_digest"`
	DesiredRevision         int64                       `json:"desired_revision"`
	Fence                   int64                       `json:"fence"`
	ExpiresAt               time.Time                   `json:"expires_at"`
	Capability              RemediationActionType       `json:"capability"`
	OneTime                 bool                        `json:"one_time"`
	AuditCorrelationID      string                      `json:"audit_correlation_id"`
}

type RemediationSafeError struct {
	Code               string `json:"code"`
	Message            string `json:"message"`
	Retryable          bool   `json:"retryable"`
	AuditCorrelationID string `json:"audit_correlation_id,omitempty"`
}

type RemediationResultEvidence struct {
	AuthorizationID         string                      `json:"authorization_id"`
	ProposalID              string                      `json:"proposal_id"`
	Executor                RemediationExecutorIdentity `json:"executor"`
	Target                  RemediationTargetIdentity   `json:"target"`
	ActionType              RemediationActionType       `json:"action_type"`
	IdempotencyKey          string                      `json:"idempotency_key"`
	CanonicalArgumentDigest string                      `json:"canonical_argument_digest"`
	DesiredRevision         int64                       `json:"desired_revision"`
	AppliedRevision         int64                       `json:"applied_revision,omitempty"`
	Fence                   int64                       `json:"fence"`
	Result                  RemediationResult           `json:"result"`
	ReconciliationRequired  bool                        `json:"reconciliation_required"`
	AutomaticResendAllowed  bool                        `json:"automatic_resend_allowed"`
	Evidence                []RemediationEvidence       `json:"evidence"`
	SafeError               *RemediationSafeError       `json:"safe_error,omitempty"`
	AuditCorrelationID      string                      `json:"audit_correlation_id"`
	CompletedAt             time.Time                   `json:"completed_at"`
}

type ServiceRemediationExecuteRequest struct {
	ActionID   string `json:"action_id"`
	Action     string `json:"action"`
	IncidentID string `json:"incident_id"`
	StreamID   string `json:"stream_id"`
}
