package contracts

import (
	"time"
)

type SystemUpdatePullOwnershipActivateRequest struct {
	ProtocolVersion                     int               `json:"protocol_version"`
	IdempotencyKey                      string            `json:"idempotency_key"`
	DesiredRevision                     int64             `json:"desired_revision"`
	Fence                               int64             `json:"fence"`
	RequiredCapability                  UpdaterCapability `json:"required_capability"`
	ExpectedExecutionHostID             string            `json:"expected_execution_host_id"`
	ExpectedOwnershipEpoch              int64             `json:"expected_ownership_epoch"`
	ExpectedSourcePolicyRevision        int64             `json:"expected_source_policy_revision"`
	ExpectedProjectionRevision          int64             `json:"expected_projection_revision"`
	ExpectedLocalExecutorPolicyRevision int64             `json:"expected_local_executor_policy_revision"`
	ExpectedLocalExecutorPolicySHA256   string            `json:"expected_local_executor_policy_sha256"`
}

type SystemUpdatePullOwnershipActivateResponse struct {
	ProtocolVersion             int                 `json:"protocol_version"`
	IdempotencyKey              string              `json:"idempotency_key"`
	DesiredRevision             int64               `json:"desired_revision"`
	AppliedRevision             int64               `json:"applied_revision"`
	Fence                       int64               `json:"fence"`
	Capability                  UpdaterCapability   `json:"capability"`
	UpdaterID                   string              `json:"updater_id"`
	ExecutionHostID             string              `json:"execution_host_id"`
	TransportMode               UpdateTransportMode `json:"transport_mode"`
	AgentServiceID              string              `json:"agent_service_id"`
	OwnershipEpoch              int64               `json:"ownership_epoch"`
	SourcePolicyRevision        int64               `json:"source_policy_revision"`
	ProjectionRevision          int64               `json:"projection_revision"`
	LocalExecutorPolicyRevision int64               `json:"local_executor_policy_revision"`
	LocalExecutorPolicySHA256   string              `json:"local_executor_policy_sha256"`
}

// SystemUpdatePullOwnershipDeactivateRequest is an administrative
// compare-and-swap request returning a pull_v2 updater to observer mode.
// The client cannot supply a replacement owner, transport or epoch.
type SystemUpdatePullOwnershipDeactivateRequest struct {
	ProtocolVersion                     int               `json:"protocol_version"`
	IdempotencyKey                      string            `json:"idempotency_key"`
	DesiredRevision                     int64             `json:"desired_revision"`
	Fence                               int64             `json:"fence"`
	RequiredCapability                  UpdaterCapability `json:"required_capability"`
	ExpectedExecutionHostID             string            `json:"expected_execution_host_id"`
	ExpectedOwnershipEpoch              int64             `json:"expected_ownership_epoch"`
	ExpectedSourcePolicyRevision        int64             `json:"expected_source_policy_revision"`
	ExpectedProjectionRevision          int64             `json:"expected_projection_revision"`
	ExpectedLocalExecutorPolicyRevision int64             `json:"expected_local_executor_policy_revision"`
	ExpectedLocalExecutorPolicySHA256   string            `json:"expected_local_executor_policy_sha256"`
}

// SystemUpdatePullOwnershipDeactivateResponse reports the pull_v2
// execution-host identity and the updater's observer epoch. It contains no
// credential and does not expose a client-selectable replacement owner.
type SystemUpdatePullOwnershipDeactivateResponse struct {
	ProtocolVersion             int                 `json:"protocol_version"`
	IdempotencyKey              string              `json:"idempotency_key"`
	DesiredRevision             int64               `json:"desired_revision"`
	AppliedRevision             int64               `json:"applied_revision"`
	Fence                       int64               `json:"fence"`
	Capability                  UpdaterCapability   `json:"capability"`
	UpdaterID                   string              `json:"updater_id"`
	ExecutionHostID             string              `json:"execution_host_id"`
	TransportMode               UpdateTransportMode `json:"transport_mode"`
	AgentServiceID              string              `json:"agent_service_id"`
	OwnershipEpoch              int64               `json:"ownership_epoch"`
	AgentOwnershipEpoch         int64               `json:"agent_ownership_epoch"`
	SourcePolicyRevision        int64               `json:"source_policy_revision"`
	ProjectionRevision          int64               `json:"projection_revision"`
	LocalExecutorPolicyRevision int64               `json:"local_executor_policy_revision"`
	LocalExecutorPolicySHA256   string              `json:"local_executor_policy_sha256"`
}

type HostAgentPolicyRequest struct {
	ServiceID       string `json:"service_id"`
	CurrentRevision int64  `json:"current_revision"`
}

type HostAgentPolicyTarget struct {
	ServiceID             string           `json:"service_id"`
	ServiceType           string           `json:"service_type"`
	DeploymentMode        string           `json:"deployment_mode"`
	DesiredEndpoint       *ServiceEndpoint `json:"desired_endpoint,omitempty"`
	AppliedEndpoint       *ServiceEndpoint `json:"applied_endpoint,omitempty"`
	LocalListenEndpoint   *ServiceEndpoint `json:"local_listen_endpoint,omitempty"`
	LocalHealthEndpoint   *ServiceEndpoint `json:"local_health_endpoint,omitempty"`
	AppliedConfigRevision int64            `json:"applied_config_revision,omitempty"`
	AppliedConfigSHA256   string           `json:"applied_config_sha256,omitempty"`
}

// HostSelfUpdateReleaseBinding is the credential-free immutable release
// identity carried across the Host Agent policy and root-executor grant
// boundaries. Download URLs, tokens, and the Control Panel's
// attestation_verified_at audit timestamp are intentionally excluded.
type HostSelfUpdateReleaseBinding struct {
	Tag                     string    `json:"tag"`
	Commit                  string    `json:"commit"`
	PublishedAt             time.Time `json:"published_at"`
	ManifestAssetID         int64     `json:"manifest_asset_id"`
	ManifestAssetName       string    `json:"manifest_asset_name"`
	ManifestSHA256          string    `json:"manifest_sha256"`
	ManifestChecksumAssetID int64     `json:"manifest_checksum_asset_id"`
	ManifestChecksumSHA256  string    `json:"manifest_checksum_sha256"`
	ArchiveAssetID          int64     `json:"archive_asset_id"`
	ArchiveAssetName        string    `json:"archive_asset_name"`
	ArchiveSize             int64     `json:"archive_size"`
	ArchiveSHA256           string    `json:"archive_sha256"`
	ArchiveChecksumAssetID  int64     `json:"archive_checksum_asset_id"`
	ArchiveChecksumSHA256   string    `json:"archive_checksum_sha256"`
	Arch                    string    `json:"arch"`
	AgentProtocolVersion    int       `json:"agent_protocol_version"`
	ExecutorProtocolVersion int       `json:"executor_protocol_version"`
	MutationProtocolVersion int       `json:"mutation_protocol_version"`
	RecoveryProtocolVersion int       `json:"recovery_protocol_version"`
	MinimumPanelVersion     string    `json:"minimum_panel_version"`
}

type HostAgentRuntimeRequirement struct {
	MinimumAgentVersion     string `json:"minimum_agent_version"`
	MinimumExecutorVersion  string `json:"minimum_executor_version"`
	AgentProtocolVersion    int    `json:"agent_protocol_version"`
	ExecutorProtocolVersion int    `json:"executor_protocol_version"`
	MutationProtocolVersion int    `json:"mutation_protocol_version"`
	RecoveryProtocolVersion int    `json:"recovery_protocol_version"`
}

type HostAgentSelfUpdateDirective struct {
	Generation              string                       `json:"generation"`
	AgentVersion            string                       `json:"agent_version"`
	ExecutorVersion         string                       `json:"executor_version"`
	Commit                  string                       `json:"commit"`
	ArtifactSHA256          string                       `json:"artifact_sha256"`
	AgentProtocolVersion    int                          `json:"agent_protocol_version"`
	ExecutorProtocolVersion int                          `json:"executor_protocol_version"`
	MutationProtocolVersion int                          `json:"mutation_protocol_version"`
	RecoveryProtocolVersion int                          `json:"recovery_protocol_version"`
	Release                 HostSelfUpdateReleaseBinding `json:"release"`
	StagedAt                time.Time                    `json:"staged_at"`
}

type HostSelfUpdateGrant struct {
	ID                                  string                       `json:"id"`
	SelfUpdateID                        string                       `json:"self_update_id"`
	AttemptGeneration                   string                       `json:"attempt_generation"`
	Operation                           string                       `json:"operation"`
	ExecutionHostID                     string                       `json:"execution_host_id"`
	AgentServiceID                      string                       `json:"agent_service_id"`
	ExpectedSelfUpdateRevision          int64                        `json:"expected_self_update_revision"`
	ExpectedOwnershipEpoch              int64                        `json:"expected_ownership_epoch"`
	ExpectedSourcePolicyRevision        int64                        `json:"expected_source_policy_revision"`
	ExpectedProjectionRevision          int64                        `json:"expected_projection_revision"`
	ExpectedLocalExecutorPolicyRevision int64                        `json:"expected_local_executor_policy_revision"`
	ExpectedLocalExecutorPolicySHA256   string                       `json:"expected_local_executor_policy_sha256"`
	AgentVersion                        string                       `json:"agent_version"`
	ExecutorVersion                     string                       `json:"executor_version"`
	ReleaseCommit                       string                       `json:"release_commit"`
	ArtifactSHA256                      string                       `json:"artifact_sha256"`
	AgentProtocolVersion                int                          `json:"agent_protocol_version"`
	ExecutorProtocolVersion             int                          `json:"executor_protocol_version"`
	MutationProtocolVersion             int                          `json:"mutation_protocol_version"`
	RecoveryProtocolVersion             int                          `json:"recovery_protocol_version"`
	Release                             HostSelfUpdateReleaseBinding `json:"release"`
	DirectiveIssuedAt                   time.Time                    `json:"directive_issued_at"`
	PlanSHA256                          string                       `json:"plan_sha256"`
	SessionID                           string                       `json:"session_id"`
	Revision                            int64                        `json:"revision"`
	IssuedAt                            time.Time                    `json:"issued_at"`
	ExpiresAt                           time.Time                    `json:"expires_at"`
	ConsumedAt                          *time.Time                   `json:"consumed_at,omitempty"`
	StageClaimRevision                  int64                        `json:"stage_claim_revision,omitempty"`
	StageClaimedAt                      *time.Time                   `json:"stage_claimed_at,omitempty"`
	CreatedAt                           time.Time                    `json:"created_at"`
	UpdatedAt                           time.Time                    `json:"updated_at"`
}

type HostAgentPolicyResponse struct {
	ServiceID                   string                        `json:"service_id"`
	TransportMode               UpdateTransportMode           `json:"transport_mode"`
	ExecutionHostID             string                        `json:"execution_host_id"`
	OwnershipEpoch              int64                         `json:"ownership_epoch"`
	Revision                    int64                         `json:"revision"`
	SourcePolicyRevision        int64                         `json:"source_policy_revision"`
	LocalExecutorPolicyRevision int64                         `json:"local_executor_policy_revision"`
	LocalExecutorPolicySHA256   string                        `json:"local_executor_policy_sha256,omitempty"`
	ObserveOnly                 bool                          `json:"observe_only"`
	RuntimeRequirement          *HostAgentRuntimeRequirement  `json:"runtime_requirement,omitempty"`
	SelfUpdate                  *HostAgentSelfUpdateDirective `json:"self_update,omitempty"`
	SelfUpdateID                string                        `json:"self_update_id,omitempty"`
	SelfUpdateRevision          int64                         `json:"self_update_revision,omitempty"`
	SelfUpdateStatus            string                        `json:"self_update_status,omitempty"`
	Targets                     []HostAgentPolicyTarget       `json:"targets"`
}
