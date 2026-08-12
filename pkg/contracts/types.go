package contracts

import "time"

type StreamStatus string

const (
	StreamCreated   StreamStatus = "created"
	StreamStarting  StreamStatus = "starting"
	StreamLive      StreamStatus = "live"
	StreamStopping  StreamStatus = "stopping"
	StreamCompleted StreamStatus = "completed"
	StreamFailed    StreamStatus = "failed"
)

type ServiceType string

const (
	ServiceDiscordBot      ServiceType = "discord_bot"
	ServiceEncoderRecorder ServiceType = "encoder_recorder"
	ServiceWorker          ServiceType = "worker"
	ServiceObservability   ServiceType = "observability"
	ServiceUpdateAgent     ServiceType = "update_agent"
)

const (
	CapabilitySceneFramesMJPEGSRT        = "scene_frames_mjpeg_srt"
	CapabilityWorkerFrameIngestMJPEGSRT  = "worker_frame_ingest_mjpeg_srt"
	CapabilitySceneVideoSRTLegacy        = "scene_video_srt"
	CapabilityWorkerVideoIngestSRTLegacy = "worker_video_ingest_srt"
)

type UpdateTransportMode string

const (
	UpdateTransportSSHV1  UpdateTransportMode = "ssh_v1"
	UpdateTransportPullV2 UpdateTransportMode = "pull_v2"
)

type ServiceStatus string

const (
	ServiceStatusUnknown          ServiceStatus = "unknown"
	ServiceStatusPending          ServiceStatus = "pending"
	ServiceStatusRegistered       ServiceStatus = "registered"
	ServiceStatusAssigned         ServiceStatus = "assigned"
	ServiceStatusRestartRequested ServiceStatus = "restart_requested"
	ServiceStatusOnline           ServiceStatus = "online"
	ServiceStatusDegraded         ServiceStatus = "degraded"
	ServiceStatusOffline          ServiceStatus = "offline"
	ServiceStatusUpdating         ServiceStatus = "updating"
)

const (
	AssignmentRolePrimary = "primary"
	AssignmentRoleStandby = "standby"
)

type ServiceScope string

const (
	ScopeServiceRegister        ServiceScope = "service.register"
	ScopeServiceHeartbeat       ServiceScope = "service.heartbeat"
	ScopeServiceLogsWrite       ServiceScope = "service.logs.write"
	ScopeServiceStatusWrite     ServiceScope = "service.status.write"
	ScopeServiceConfigRead      ServiceScope = "service.config.read"
	ScopeServiceSecretResolve   ServiceScope = "service.secret.resolve"
	ScopeWorkerEventsWrite      ServiceScope = "worker.events.write"
	ScopeEncoderStatusWrite     ServiceScope = "encoder.status.write"
	ScopeDiscordStatusWrite     ServiceScope = "discord.status.write"
	ScopeStreamsStart           ServiceScope = "streams.start"
	ScopeStreamsStop            ServiceScope = "streams.stop"
	ScopeObservabilityIngest    ServiceScope = "observability.ingest"
	ScopeNotificationsEmailSend ServiceScope = "notifications.email.send"
	ScopeRemediationExecute     ServiceScope = "remediation.execute"
	ScopeUpdatesClaim           ServiceScope = "updates.claim"
	ScopeUpdatesReport          ServiceScope = "updates.report"
	ScopeUpdatesAuthorize       ServiceScope = "updates.authorize"
)

type SystemUpdateStrategy string

const (
	SystemUpdateWhenIdle    SystemUpdateStrategy = "when_idle"
	SystemUpdateMaintenance SystemUpdateStrategy = "maintenance"
)

type SystemUpdateTargetType string

const (
	SystemUpdateTargetControlPanel    SystemUpdateTargetType = "control_panel"
	SystemUpdateTargetDiscordBot      SystemUpdateTargetType = "discord_bot"
	SystemUpdateTargetEncoderRecorder SystemUpdateTargetType = "encoder_recorder"
	SystemUpdateTargetObservability   SystemUpdateTargetType = "observability"
	SystemUpdateTargetWorker          SystemUpdateTargetType = "worker"
)

type SystemUpdateDeploymentMode string

const (
	SystemUpdateDeploymentSystemd SystemUpdateDeploymentMode = "systemd"
	SystemUpdateDeploymentDocker  SystemUpdateDeploymentMode = "docker"
)

type SystemUpdateReachability string

const (
	SystemUpdateReachable   SystemUpdateReachability = "reachable"
	SystemUpdateUnreachable SystemUpdateReachability = "unreachable"
	SystemUpdateUnknown     SystemUpdateReachability = "unknown"
)

type SystemUpdateOperation string

const (
	SystemUpdateOperationSoftwareUpdate  SystemUpdateOperation = "software_update"
	SystemUpdateOperationPortReconfigure SystemUpdateOperation = "port_reconfigure"
)

type SystemUpdateMutationOperation string

const (
	SystemUpdateMutationApply                    SystemUpdateMutationOperation = "apply"
	SystemUpdateMutationReconcile                SystemUpdateMutationOperation = "reconcile"
	SystemUpdateMutationPortReconfigure          SystemUpdateMutationOperation = "port_reconfigure"
	SystemUpdateMutationPortReconfigureReconcile SystemUpdateMutationOperation = "port_reconfigure_reconcile"
)

type SystemUpdatePortProtocol string

const (
	SystemUpdatePortProtocolTCP SystemUpdatePortProtocol = "tcp"
)

type SystemUpdatePortReconfigurationResult string

const (
	SystemUpdatePortReconfigurationApplied        SystemUpdatePortReconfigurationResult = "applied"
	SystemUpdatePortReconfigurationRolledBack     SystemUpdatePortReconfigurationResult = "rolled_back"
	SystemUpdatePortReconfigurationUnchanged      SystemUpdatePortReconfigurationResult = "unchanged"
	SystemUpdatePortReconfigurationRollbackFailed SystemUpdatePortReconfigurationResult = "rollback_failed"
)

type SystemUpdatePortMappingState string

const (
	SystemUpdatePortMappingApplied     SystemUpdatePortMappingState = "applied"
	SystemUpdatePortMappingDrifted     SystemUpdatePortMappingState = "drifted"
	SystemUpdatePortMappingUnavailable SystemUpdatePortMappingState = "unavailable"
)

type SystemUpdateDockerPortReconfiguration struct {
	PublishedHostIP             string `json:"published_host_ip,omitempty"`
	OldPublishedPort            int    `json:"old_published_port,omitempty"`
	NewPublishedPort            int    `json:"new_published_port,omitempty"`
	OldContainerPort            int    `json:"old_container_port,omitempty"`
	NewContainerPort            int    `json:"new_container_port,omitempty"`
	OldHealthPort               int    `json:"old_health_port,omitempty"`
	NewHealthPort               int    `json:"new_health_port,omitempty"`
	ApprovedComposeConfigSHA256 string `json:"approved_compose_config_sha256,omitempty"`
	ApprovedComposeRevision     int64  `json:"approved_compose_revision,omitempty"`
	ExpectedVersionEnvSHA256    string `json:"expected_version_env_sha256,omitempty"`
	ExpectedContainerID         string `json:"expected_container_id,omitempty"`
	ExpectedImageID             string `json:"expected_image_id,omitempty"`
	ExpectedRepositoryDigest    string `json:"expected_repository_digest,omitempty"`
}

// SystemUpdatePortReconfiguration is the nested wire shape shared by a
// port-reconfiguration job, claim, mutation grant and terminal report. Jobs
// and grants carry the immutable plan fields, terminal jobs may also expose the
// accepted Result, and reports carry Result only. Software-update payloads omit
// this object entirely.
type SystemUpdatePortReconfiguration struct {
	NetworkNamespace               string                                 `json:"network_namespace,omitempty"`
	Protocol                       SystemUpdatePortProtocol               `json:"protocol,omitempty"`
	OldPort                        int                                    `json:"old_port,omitempty"`
	NewPort                        int                                    `json:"new_port,omitempty"`
	ExpectedEndpointRevision       int64                                  `json:"expected_endpoint_revision,omitempty"`
	TargetEndpointRevision         int64                                  `json:"target_endpoint_revision,omitempty"`
	ExpectedConfigRevision         int64                                  `json:"expected_config_revision,omitempty"`
	TargetConfigRevision           int64                                  `json:"target_config_revision,omitempty"`
	ExpectedConfigSHA256           string                                 `json:"expected_config_sha256,omitempty"`
	TargetConfigSHA256             string                                 `json:"target_config_sha256,omitempty"`
	ExpectedSourcePolicyRevision   int64                                  `json:"expected_source_policy_revision,omitempty"`
	ExpectedUpdaterPolicyRevision  int64                                  `json:"expected_updater_policy_revision,omitempty"`
	ExpectedExecutorPolicyRevision int64                                  `json:"expected_executor_policy_revision,omitempty"`
	ExpectedExecutorPolicySHA256   string                                 `json:"expected_executor_policy_sha256,omitempty"`
	PortPlanSHA256                 string                                 `json:"port_plan_sha256,omitempty"`
	Docker                         *SystemUpdateDockerPortReconfiguration `json:"docker,omitempty"`
	Result                         SystemUpdatePortReconfigurationResult  `json:"result,omitempty"`
}

type SystemUpdateStatus string

const (
	SystemUpdateQueued         SystemUpdateStatus = "queued"
	SystemUpdateClaimed        SystemUpdateStatus = "claimed"
	SystemUpdateDownloading    SystemUpdateStatus = "downloading"
	SystemUpdateVerifying      SystemUpdateStatus = "verifying"
	SystemUpdateStaging        SystemUpdateStatus = "staging"
	SystemUpdateStopping       SystemUpdateStatus = "stopping"
	SystemUpdateInstalling     SystemUpdateStatus = "installing"
	SystemUpdateStarting       SystemUpdateStatus = "starting"
	SystemUpdateHealthChecking SystemUpdateStatus = "health_checking"
	SystemUpdateReconciling    SystemUpdateStatus = "reconciling"
	SystemUpdateRollingBack    SystemUpdateStatus = "rolling_back"
	SystemUpdateSucceeded      SystemUpdateStatus = "succeeded"
	SystemUpdateRolledBack     SystemUpdateStatus = "rolled_back"
	SystemUpdateFailed         SystemUpdateStatus = "failed"
	SystemUpdateCanceled       SystemUpdateStatus = "canceled"
)

type SystemUpdateCreateRequest struct {
	Operation                SystemUpdateOperation `json:"operation,omitempty"`
	TargetID                 string                `json:"target_id"`
	Strategy                 SystemUpdateStrategy  `json:"strategy,omitempty"`
	NewPort                  int                   `json:"new_port,omitempty"`
	NewAdvertisedPort        int                   `json:"new_advertised_port,omitempty"`
	NewPublishedPort         int                   `json:"new_published_port,omitempty"`
	NewContainerPort         int                   `json:"new_container_port,omitempty"`
	ExpectedEndpointRevision int64                 `json:"expected_endpoint_revision,omitempty"`
	IdempotencyKey           string                `json:"idempotency_key"`
}

type SystemUpdatePullOwnershipActivateRequest struct {
	ExpectedExecutionHostID             string `json:"expected_execution_host_id"`
	ExpectedOwnershipEpoch              int64  `json:"expected_ownership_epoch"`
	ExpectedSourcePolicyRevision        int64  `json:"expected_source_policy_revision"`
	ExpectedProjectionRevision          int64  `json:"expected_projection_revision"`
	ExpectedLocalExecutorPolicyRevision int64  `json:"expected_local_executor_policy_revision"`
	ExpectedLocalExecutorPolicySHA256   string `json:"expected_local_executor_policy_sha256"`
}

type SystemUpdatePullOwnershipActivateResponse struct {
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
// compare-and-swap request. The previous ssh_v1 owner is deliberately absent:
// the server restores only the owner it preserved during activation.
type SystemUpdatePullOwnershipDeactivateRequest struct {
	ExpectedExecutionHostID             string `json:"expected_execution_host_id"`
	ExpectedOwnershipEpoch              int64  `json:"expected_ownership_epoch"`
	ExpectedSourcePolicyRevision        int64  `json:"expected_source_policy_revision"`
	ExpectedProjectionRevision          int64  `json:"expected_projection_revision"`
	ExpectedLocalExecutorPolicyRevision int64  `json:"expected_local_executor_policy_revision"`
	ExpectedLocalExecutorPolicySHA256   string `json:"expected_local_executor_policy_sha256"`
}

// SystemUpdatePullOwnershipDeactivateResponse reports the restored ssh_v1
// execution-host owner and the pull agent's observer epoch. It contains no
// credential and does not expose a client-selectable legacy owner field.
type SystemUpdatePullOwnershipDeactivateResponse struct {
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

type SystemUpdatePortMapping struct {
	Mode            SystemUpdateDeploymentMode   `json:"mode"`
	State           SystemUpdatePortMappingState `json:"state"`
	AdvertisedPort  int                          `json:"advertised_port,omitempty"`
	PublishedPort   int                          `json:"published_port,omitempty"`
	ContainerPort   int                          `json:"container_port,omitempty"`
	HealthPort      int                          `json:"health_port,omitempty"`
	ConfigRevision  int64                        `json:"config_revision,omitempty"`
	PublishedHostIP string                       `json:"published_host_ip,omitempty"`
	ReportedAt      *time.Time                   `json:"reported_at,omitempty"`
}

type SystemUpdateTarget struct {
	TargetID                string                     `json:"target_id"`
	TargetType              SystemUpdateTargetType     `json:"target_type"`
	Name                    string                     `json:"name"`
	HostID                  string                     `json:"host_id,omitempty"`
	CurrentVersion          string                     `json:"current_version,omitempty"`
	LatestVersion           string                     `json:"latest_version,omitempty"`
	UpdateAvailable         bool                       `json:"update_available"`
	DeploymentMode          SystemUpdateDeploymentMode `json:"deployment_mode,omitempty"`
	UpdaterID               string                     `json:"updater_id,omitempty"`
	UpdaterOnline           bool                       `json:"updater_online"`
	Eligible                bool                       `json:"eligible"`
	BlockedReason           string                     `json:"blocked_reason,omitempty"`
	EligibleOperations      []SystemUpdateOperation    `json:"eligible_operations,omitempty"`
	OperationBlockedReasons map[string]string          `json:"operation_blocked_reasons,omitempty"`
	Busy                    bool                       `json:"busy"`
	CurrentStreamID         string                     `json:"current_stream_id,omitempty"`
	UpdateCheckSource       string                     `json:"update_check_source,omitempty"`
	UpdateCheckError        string                     `json:"update_check_error,omitempty"`
	PortMapping             *SystemUpdatePortMapping   `json:"port_mapping,omitempty"`
}

type SystemUpdateJob struct {
	ID              string                           `json:"id"`
	TargetID        string                           `json:"target_id"`
	TargetType      SystemUpdateTargetType           `json:"target_type"`
	ExecutionHostID string                           `json:"host_id"`
	TransportMode   UpdateTransportMode              `json:"transport_mode,omitempty"`
	OwnershipEpoch  int64                            `json:"ownership_epoch,omitempty"`
	PolicyRevision  int64                            `json:"policy_revision,omitempty"`
	DeploymentMode  SystemUpdateDeploymentMode       `json:"deployment_mode"`
	CurrentVersion  string                           `json:"current_version"`
	TargetVersion   string                           `json:"target_version"`
	Strategy        SystemUpdateStrategy             `json:"strategy"`
	Status          SystemUpdateStatus               `json:"status"`
	IdempotencyKey  string                           `json:"idempotency_key"`
	UpdaterID       string                           `json:"updater_id,omitempty"`
	RequestedBy     string                           `json:"requested_by,omitempty"`
	LeaseGeneration int64                            `json:"lease_generation"`
	LeaseExpiresAt  *time.Time                       `json:"lease_expires_at,omitempty"`
	Sequence        int64                            `json:"sequence"`
	Progress        int                              `json:"progress"`
	Code            string                           `json:"code,omitempty"`
	Message         string                           `json:"message,omitempty"`
	ArtifactDigest  string                           `json:"artifact_digest,omitempty"`
	PreviousDigest  string                           `json:"previous_digest,omitempty"`
	Operation       SystemUpdateOperation            `json:"operation,omitempty"`
	PortReconfigure *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
	CreatedAt       time.Time                        `json:"created_at"`
	UpdatedAt       time.Time                        `json:"updated_at"`
	ClaimedAt       *time.Time                       `json:"claimed_at,omitempty"`
	CompletedAt     *time.Time                       `json:"completed_at,omitempty"`
	CanceledAt      *time.Time                       `json:"canceled_at,omitempty"`
}

type SystemUpdatesResponse struct {
	Updaters []SystemUpdateAgentStatus `json:"updaters"`
	Hosts    []SystemUpdateHostStatus  `json:"hosts"`
	Targets  []SystemUpdateTarget      `json:"targets"`
	Jobs     []SystemUpdateJob         `json:"jobs"`
}

type SystemUpdateAgentStatus struct {
	UpdaterID                         string              `json:"updater_id"`
	Name                              string              `json:"name"`
	TransportMode                     UpdateTransportMode `json:"transport_mode,omitempty"`
	ExecutionHostID                   string              `json:"execution_host_id,omitempty"`
	OwnershipEpoch                    int64               `json:"ownership_epoch,omitempty"`
	Status                            string              `json:"status"`
	Online                            bool                `json:"online"`
	Version                           string              `json:"version"`
	LastHeartbeatAt                   *time.Time          `json:"last_heartbeat_at,omitempty"`
	DesiredRevision                   int64               `json:"desired_revision,omitempty"`
	AppliedRevision                   int64               `json:"applied_revision,omitempty"`
	PolicyStatus                      string              `json:"policy_status,omitempty"`
	PolicyErrorCode                   string              `json:"policy_error_code,omitempty"`
	SSHClientPublicKeys               map[string]string   `json:"ssh_client_public_keys,omitempty"`
	SSHClientKeyFingerprints          map[string]string   `json:"ssh_client_key_fingerprints,omitempty"`
	BootstrapEncryptionPublicKey      string              `json:"bootstrap_encryption_public_key,omitempty"`
	BootstrapEncryptionKeyFingerprint string              `json:"bootstrap_encryption_key_fingerprint,omitempty"`
}

type SystemUpdateHostStatus struct {
	HostID                  string                   `json:"host_id"`
	Name                    string                   `json:"name"`
	UpdaterID               string                   `json:"updater_id"`
	Reachability            SystemUpdateReachability `json:"reachability"`
	ReachabilityCheckedAt   *time.Time               `json:"reachability_checked_at,omitempty"`
	ReachabilityCode        string                   `json:"reachability_code,omitempty"`
	SSHClientPublicKey      string                   `json:"ssh_client_public_key,omitempty"`
	SSHClientKeyFingerprint string                   `json:"ssh_client_key_fingerprint,omitempty"`
}

type UpdateAgentClaimRequest struct {
	ServiceID   string `json:"service_id"`
	HostID      string `json:"host_id,omitempty"`
	ActiveJobID string `json:"active_job_id,omitempty"`
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
	ServiceID       string                           `json:"service_id"`
	LeaseToken      string                           `json:"lease_token"`
	LeaseGeneration int64                            `json:"lease_generation"`
	Sequence        int64                            `json:"sequence"`
	Status          SystemUpdateStatus               `json:"status"`
	Progress        int                              `json:"progress,omitempty"`
	Code            string                           `json:"code,omitempty"`
	Message         string                           `json:"message,omitempty"`
	ArtifactDigest  string                           `json:"artifact_digest,omitempty"`
	PreviousDigest  string                           `json:"previous_digest,omitempty"`
	PortReconfigure *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
}

type UpdateAgentAuthorizeRequest struct {
	ServiceID       string                     `json:"service_id"`
	LeaseToken      string                     `json:"lease_token"`
	LeaseGeneration int64                      `json:"lease_generation"`
	ExecutionHostID string                     `json:"host_id,omitempty"`
	TargetID        string                     `json:"target_id"`
	TargetVersion   string                     `json:"target_version"`
	DeploymentMode  SystemUpdateDeploymentMode `json:"deployment_mode"`
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

type ReleaseChannel string

const (
	ReleaseChannelHost   ReleaseChannel = "host"
	ReleaseChannelDocker ReleaseChannel = "docker"
)

type ReleaseManifest struct {
	SchemaVersion       int                        `json:"schema_version"`
	ReleaseID           string                     `json:"release_id"`
	Channel             ReleaseChannel             `json:"channel"`
	PublishedAt         time.Time                  `json:"published_at"`
	BundleVersion       string                     `json:"bundle_version,omitempty"`
	GeneratedAt         *time.Time                 `json:"generated_at,omitempty"`
	MinimumAgentVersion string                     `json:"minimum_agent_version"`
	Components          []ReleaseManifestComponent `json:"components"`
}

type ReleaseManifestComponent struct {
	Service            string            `json:"service"`
	SourceVersion      string            `json:"source_version"`
	Commit             string            `json:"commit,omitempty"`
	Image              string            `json:"image,omitempty"`
	ManifestDigest     string            `json:"manifest_digest,omitempty"`
	Artifacts          []ReleaseArtifact `json:"artifacts,omitempty"`
	PlatformDigests    map[string]string `json:"platform_digests,omitempty"`
	RollbackCompatible bool              `json:"rollback_compatible"`
	DatabaseSchema     string            `json:"database_schema"`
}

type ReleaseArtifact struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type ErrorResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type PasskeyCredential struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	CredentialIDHash string   `json:"credential_id_hash"`
	SignCount        uint32   `json:"sign_count"`
	Transports       []string `json:"transports,omitempty"`
	AAGUID           string   `json:"aaguid,omitempty"`
	BackupEligible   bool     `json:"backup_eligible,omitempty"`
	BackedUp         bool     `json:"backed_up,omitempty"`
	CreatedAt        string   `json:"created_at,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	LastUsedAt       string   `json:"last_used_at,omitempty"`
}

type PasskeyRegistrationStartRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}

type PasskeyRegistrationStartResponse struct {
	RegistrationToken string                 `json:"registration_token"`
	ExpiresAt         string                 `json:"expires_at"`
	PublicKey         PasskeyCreationOptions `json:"public_key"`
}

type PasskeyRegistrationFinishRequest struct {
	RegistrationToken string         `json:"registration_token"`
	Name              string         `json:"name,omitempty"`
	Credential        map[string]any `json:"credential"`
}

type PasskeyLoginStartRequest struct {
	Username string `json:"username,omitempty"`
}

type PasskeyLoginStartResponse struct {
	ChallengeToken string         `json:"challenge_token"`
	ExpiresAt      string         `json:"expires_at"`
	PublicKey      map[string]any `json:"public_key"`
}

type PasskeyLoginFinishRequest struct {
	ChallengeToken string         `json:"challenge_token"`
	Credential     map[string]any `json:"credential"`
}

type PasskeyCreationOptions struct {
	Challenge              string                        `json:"challenge"`
	RP                     PasskeyRelyingParty           `json:"rp"`
	User                   PasskeyUser                   `json:"user"`
	PubKeyCredParams       []PasskeyCredentialParameter  `json:"pubKeyCredParams"`
	Timeout                int                           `json:"timeout"`
	Attestation            string                        `json:"attestation"`
	AuthenticatorSelection PasskeyAuthenticatorSelection `json:"authenticatorSelection"`
}

type PasskeyRelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasskeyUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type PasskeyCredentialParameter struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type PasskeyAuthenticatorSelection struct {
	ResidentKey      string `json:"residentKey"`
	UserVerification string `json:"userVerification"`
}

type ServiceRegistration struct {
	ServiceID     string              `json:"service_id"`
	ServiceType   ServiceType         `json:"service_type"`
	ServiceName   string              `json:"service_name"`
	TransportMode UpdateTransportMode `json:"transport_mode,omitempty"`
	Description   string              `json:"description,omitempty"`
	Host          string              `json:"host,omitempty"`
	Port          int                 `json:"port,omitempty"`
	SSLEnabled    bool                `json:"ssl_enabled"`
	PublicURL     string              `json:"public_url,omitempty"`
	Version       string              `json:"version"`
	Commit        string              `json:"commit,omitempty"`
	BuildDate     string              `json:"build_date,omitempty"`
	Capabilities  map[string]any      `json:"capabilities"`
	Hostname      string              `json:"hostname,omitempty"`
	OS            string              `json:"os,omitempty"`
	Arch          string              `json:"arch,omitempty"`
}

type ServiceToken struct {
	ID          string      `json:"id"`
	ServiceType ServiceType `json:"service_type"`
	Scopes      []string    `json:"scopes"`
	RevokedAt   *time.Time  `json:"revoked_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`

	// Token is returned only once on creation or rotation. List/get responses must omit it.
	Token string `json:"token,omitempty"`
}

type ServiceTokenCreateRequest struct {
	ServiceType  ServiceType    `json:"service_type"`
	Scopes       []ServiceScope `json:"scopes"`
	ServiceID    string         `json:"service_id,omitempty"`
	ServiceName  string         `json:"service_name,omitempty"`
	PublicURL    string         `json:"public_url,omitempty"`
	Version      string         `json:"version,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
}

type ServiceEndpoint struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"ssl_enabled"`
	PublicURL  string `json:"public_url"`
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

type RegisteredService struct {
	ServiceID             string              `json:"service_id"`
	ServiceType           ServiceType         `json:"service_type"`
	ServiceName           string              `json:"service_name"`
	TransportMode         UpdateTransportMode `json:"transport_mode,omitempty"`
	ExecutionHostID       string              `json:"execution_host_id,omitempty"`
	OwnershipEpoch        int64               `json:"ownership_epoch,omitempty"`
	PublicURL             string              `json:"public_url,omitempty"`
	DesiredEndpoint       *ServiceEndpoint    `json:"desired_endpoint,omitempty"`
	AppliedEndpoint       *ServiceEndpoint    `json:"applied_endpoint,omitempty"`
	ReportedEndpoint      *ServiceEndpoint    `json:"reported_endpoint,omitempty"`
	EndpointRevision      int64               `json:"endpoint_revision,omitempty"`
	EndpointStatus        string              `json:"endpoint_status,omitempty"`
	AppliedConfigRevision int64               `json:"applied_config_revision,omitempty"`
	AppliedConfigSHA256   string              `json:"applied_config_sha256,omitempty"`
	Version               string              `json:"version"`
	Status                ServiceStatus       `json:"status"`
	AssignmentRole        string              `json:"assignment_role,omitempty"`
	LastHeartbeatAt       *time.Time          `json:"last_heartbeat_at,omitempty"`
	HealthStatus          string              `json:"health_status,omitempty"`
	HeartbeatStale        bool                `json:"heartbeat_stale,omitempty"`
	HeartbeatAgeSec       *int64              `json:"heartbeat_age_sec,omitempty"`
	CurrentStreamID       string              `json:"current_stream_id,omitempty"`
	Capabilities          map[string]any      `json:"capabilities"`
	ReportedCapabilities  map[string]any      `json:"reported_capabilities,omitempty"`
	TokenID               string              `json:"-"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type StreamServiceAssignment struct {
	StreamID       string      `json:"stream_id"`
	ServiceID      string      `json:"service_id"`
	ServiceType    ServiceType `json:"service_type"`
	AssignmentRole string      `json:"assignment_role"`
	AssignedAt     time.Time   `json:"assigned_at"`
}

type ServiceAssignmentWriteRequest struct {
	StreamID       string `json:"stream_id"`
	AssignmentRole string `json:"assignment_role,omitempty"`
}

type ServiceRuntimeConfig struct {
	Service              RegisteredService             `json:"service"`
	Assignments          []StreamServiceAssignment     `json:"assignments"`
	Profiles             map[ProfileKind][]Profile     `json:"profiles"`
	StreamDiscordConfigs []ServiceRuntimeDiscordConfig `json:"stream_discord_configs,omitempty"`
	StreamArchiveConfigs []ServiceRuntimeArchiveConfig `json:"stream_archive_configs,omitempty"`
	StreamYouTubeConfigs []ServiceRuntimeYouTubeConfig `json:"stream_youtube_configs,omitempty"`
}

type ServiceRuntimeDiscordConfig struct {
	StreamID         string `json:"stream_id"`
	AssignmentRole   string `json:"assignment_role"`
	DiscordConfigID  string `json:"discord_config_id"`
	GuildID          string `json:"guild_id"`
	VoiceChannelID   string `json:"voice_channel_id"`
	TextChannelID    string `json:"text_channel_id,omitempty"`
	AutoStartTrigger string `json:"auto_start_trigger,omitempty"`
}

type ServiceRuntimeArchiveConfig struct {
	StreamID         string         `json:"stream_id"`
	AssignmentRole   string         `json:"assignment_role"`
	ArchiveProfileID string         `json:"archive_profile_id"`
	Ready            bool           `json:"ready"`
	ReadinessCode    string         `json:"readiness_code,omitempty"`
	ReadinessMessage string         `json:"readiness_message,omitempty"`
	ArchiveConfig    map[string]any `json:"archive_config,omitempty"`
}

type ServiceRuntimeYouTubeConfig struct {
	StreamID         string         `json:"stream_id"`
	AssignmentRole   string         `json:"assignment_role"`
	YouTubeOutputID  string         `json:"youtube_output_id"`
	Ready            bool           `json:"ready"`
	ReadinessCode    string         `json:"readiness_code,omitempty"`
	ReadinessMessage string         `json:"readiness_message,omitempty"`
	YouTubeConfig    map[string]any `json:"youtube_config,omitempty"`
	ActiveRuntime    map[string]any `json:"active_runtime,omitempty"`
}

type ServiceRuntimeSecretResolveRequest struct {
	ServiceID        string `json:"service_id"`
	StreamID         string `json:"stream_id,omitempty"`
	ArchiveProfileID string `json:"archive_profile_id,omitempty"`
	SecretName       string `json:"secret_name"`
}

type ServiceRuntimeSecretResolveResponse struct {
	SecretName   string `json:"secret_name"`
	Value        string `json:"value"`
	ExpiresInSec int    `json:"expires_in_sec"`
}

// ServiceNotificationEmailRequest asks Control Panel to deliver one text message
// and, optionally, an HTML alternative with its globally managed SMTP settings.
// This service-authenticated
// request is restricted to a registered Observability service token that has
// the dedicated notifications.email.send scope.
type ServiceNotificationEmailRequest struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Text       string   `json:"text"`
	HTML       string   `json:"html,omitempty"`
}

// ServiceNotificationEmailResponse intentionally reports only a count so raw
// recipient addresses and SMTP settings never cross the response boundary.
type ServiceNotificationEmailResponse struct {
	Status         string `json:"status"`
	RecipientCount int    `json:"recipient_count"`
}

type Heartbeat struct {
	ServiceID       string         `json:"service_id"`
	NodeID          string         `json:"nodeId,omitempty"`
	NodeIDSnake     string         `json:"node_id,omitempty"`
	CurrentStreamID string         `json:"current_stream_id,omitempty"`
	Status          string         `json:"status"`
	Version         string         `json:"version,omitempty"`
	Commit          string         `json:"commit,omitempty"`
	BuildDate       string         `json:"build_date,omitempty"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	Hostname        string         `json:"hostname,omitempty"`
	OS              string         `json:"os,omitempty"`
	Arch            string         `json:"arch,omitempty"`
	API             *NodeAgentAPI  `json:"api,omitempty"`
	Metrics         map[string]any `json:"metrics,omitempty"`
	Timestamp       *time.Time     `json:"timestamp,omitempty"`
}

type NodeAgentAPI struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"sslEnabled"`
}

type ServicePreflightCheck struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ServicePreflightResponse struct {
	CheckedAt time.Time               `json:"checked_at"`
	Ready     bool                    `json:"ready"`
	Checks    []ServicePreflightCheck `json:"checks"`
	Summary   map[string]any          `json:"summary,omitempty"`
}

type ServiceStreamEvent struct {
	ServiceID string         `json:"service_id"`
	StreamID  string         `json:"stream_id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
}

type StreamArtifact struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	RelativePath string `json:"relative_path"`
	SizeBytes    int64  `json:"size_bytes"`
}

type ServiceArtifactReport struct {
	ServiceID string           `json:"service_id"`
	StreamID  string           `json:"stream_id"`
	Artifacts []StreamArtifact `json:"artifacts"`
}

type WorkerEventType string

const (
	WorkerEventCurrentTime   WorkerEventType = "overlay.current_time"
	WorkerEventParticipants  WorkerEventType = "overlay.participants"
	WorkerEventActiveSpeaker WorkerEventType = "overlay.active_speaker"
	WorkerEventDiscordChat   WorkerEventType = "overlay.discord_chat"
	WorkerEventCaptionTelop  WorkerEventType = "caption.telop"
	WorkerEventCaptionFinal  WorkerEventType = "caption.final"
)

type WorkerEvent struct {
	ID        string          `json:"id"`
	StreamID  string          `json:"stream_id"`
	ServiceID string          `json:"service_id,omitempty"`
	Type      WorkerEventType `json:"type"`
	Payload   map[string]any  `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type WorkerStreamContext struct {
	StreamID              string `json:"stream_id"`
	StreamName            string `json:"stream_name,omitempty"`
	EncoderRecorderURL    string `json:"encoder_recorder_url,omitempty"`
	StreamIngestToken     string `json:"stream_ingest_token,omitempty"`
	OverlayProfileID      string `json:"overlay_profile_id,omitempty"`
	CaptionProfileID      string `json:"caption_profile_id,omitempty"`
	EncoderProfileID      string `json:"encoder_profile_id,omitempty"`
	VideoWidth            int    `json:"video_width,omitempty"`
	VideoHeight           int    `json:"video_height,omitempty"`
	VideoFPS              int    `json:"video_fps,omitempty"`
	VideoIngestURL        string `json:"video_ingest_url,omitempty"`
	VideoIngestPassphrase string `json:"video_ingest_passphrase,omitempty"`
	VideoIngestPBKeyLen   int    `json:"video_ingest_pbkeylen,omitempty"`
}

type WorkerStartJobRequest = WorkerStreamContext

type DiscordVoiceJob struct {
	StreamID          string `json:"stream_id"`
	GuildID           string `json:"guild_id"`
	VoiceChannelID    string `json:"voice_channel_id"`
	TextChannelID     string `json:"text_channel_id,omitempty"`
	EncoderAudioURL   string `json:"encoder_audio_url,omitempty"`
	CaptionAudioURL   string `json:"caption_audio_url,omitempty"`
	CaptionAudioToken string `json:"caption_audio_token,omitempty"`
	StreamIngestToken string `json:"stream_ingest_token,omitempty"`
	WorkerEventsURL   string `json:"worker_events_url,omitempty"`
	WorkerEventsToken string `json:"worker_events_token,omitempty"`
}

type DiscordBotStartJobRequest = DiscordVoiceJob

type EncoderInputMode string

const (
	EncoderInputModeExternal             EncoderInputMode = "external"
	EncoderInputModeDiscordOpusRTP       EncoderInputMode = "discord_opus_rtp"
	EncoderInputModeWorkerSceneSRTLegacy EncoderInputMode = "worker_scene_srt"
	EncoderInputModeWorkerSceneFramesSRT EncoderInputMode = "worker_scene_frames_srt"
)

type DiscordOpusPacket struct {
	SSRC       uint32    `json:"ssrc"`
	UserID     string    `json:"user_id,omitempty"`
	Sequence   uint16    `json:"sequence"`
	Timestamp  uint32    `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`
	OpusBase64 string    `json:"opus_base64"`
}

type DiscordOpusIngestRequest struct {
	StreamID string              `json:"stream_id"`
	Source   string              `json:"source"`
	Packets  []DiscordOpusPacket `json:"packets"`
}

type DiscordOpusIngestResponse struct {
	Accepted      bool   `json:"accepted"`
	StreamID      string `json:"stream_id"`
	AcceptedCount int    `json:"accepted_count"`
	RTPForwarded  int    `json:"rtp_forwarded"`
	LogsPath      string `json:"logs_path,omitempty"`
}

type YouTubeLiveNotificationRequest struct {
	EventID  string `json:"event_id"`
	WatchURL string `json:"watch_url"`
}

type YouTubeLiveNotificationResponse struct {
	Status      string `json:"status"`
	MessageID   string `json:"message_id"`
	AlreadySent bool   `json:"already_sent"`
}

type ServiceNotificationError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type DiscordAudioBridgeStatus struct {
	StreamID          string    `json:"stream_id"`
	BridgeActive      bool      `json:"bridge_active"`
	StartedAt         time.Time `json:"started_at"`
	LastPacketAt      time.Time `json:"last_packet_at,omitempty"`
	PacketsTotal      int64     `json:"packets_total"`
	RTPForwardedTotal int64     `json:"rtp_forwarded"`
	LastPacketAgeSec  float64   `json:"last_packet_age_sec"`
}

type StartStreamRequest struct {
	DiscordConfigID       string  `json:"discord_config_id,omitempty"`
	DiscordGuildID        string  `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string  `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string  `json:"discord_text_channel_id,omitempty"`
	EncoderInputURL       string  `json:"encoder_input_url,omitempty"`
	EncoderRTMPURL        string  `json:"encoder_rtmp_url,omitempty"`
	EncoderProfileID      string  `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string  `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string  `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB    float64 `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID      string  `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string  `json:"youtube_output_id,omitempty"`
}

type StreamSettingsWriteRequest struct {
	DiscordConfigID       string  `json:"discord_config_id,omitempty"`
	DiscordGuildID        string  `json:"discord_guild_id,omitempty"`
	DiscordVoiceChannelID string  `json:"discord_voice_channel_id,omitempty"`
	DiscordTextChannelID  string  `json:"discord_text_channel_id,omitempty"`
	AutoStartTrigger      string  `json:"auto_start_trigger,omitempty"`
	EncoderProfileID      string  `json:"encoder_profile_id,omitempty"`
	CaptionProfileID      string  `json:"caption_profile_id,omitempty"`
	OverlayProfileID      string  `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB    float64 `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID      string  `json:"archive_profile_id,omitempty"`
	YouTubeOutputID       string  `json:"youtube_output_id,omitempty"`
	EncoderInputURL       string  `json:"encoder_input_url,omitempty"`
}

type StreamWriteRequest struct {
	Name string `json:"name"`
	StreamSettingsWriteRequest
}

type ArchiveRuntimeConfig struct {
	DriveDestinationID     string `json:"drive_destination_id,omitempty"`
	ArchiveProfileID       string `json:"archive_profile_id,omitempty"`
	AuthMode               string `json:"auth_mode,omitempty"`
	OAuthAccountID         string `json:"oauth_account_id,omitempty"`
	OAuthProviderID        string `json:"oauth_provider_id,omitempty"`
	FolderIDSecretName     string `json:"folder_id_secret_name,omitempty"`
	BasePath               string `json:"base_path,omitempty"`
	SharedDrive            bool   `json:"shared_drive,omitempty"`
	ClientID               string `json:"client_id,omitempty"`
	ClientSecretSecretName string `json:"client_secret_secret_name,omitempty"`
	RefreshTokenSecretName string `json:"refresh_token_secret_name,omitempty"`
}

type YouTubeOutputMode string

const (
	YouTubeOutputModeStreamKey          YouTubeOutputMode = "stream_key"
	YouTubeOutputModeLiveAPIDryRun      YouTubeOutputMode = "live_api_dry_run"
	YouTubeOutputModeLiveAPI            YouTubeOutputMode = "live_api"
	YouTubeOutputModeLiveAPIRelayStatic YouTubeOutputMode = "live_api_relay_static"
)

// EncoderOutputRelayMode describes the non-secret output routing capability
// advertised by an Encoder/Recorder. It is intentionally distinct from
// YouTubeOutputMode, which selects the Control Panel output profile behavior.
type EncoderOutputRelayMode string

const (
	EncoderOutputRelayModeDirect          EncoderOutputRelayMode = "direct"
	EncoderOutputRelayModeLegacyStreamKey EncoderOutputRelayMode = "legacy_stream_key"
	EncoderOutputRelayModeLiveAPIStatic   EncoderOutputRelayMode = "live_api_static"
	// EncoderOutputRelayModeStaticLegacyAlias is accepted for existing Encoder
	// capability input and registered-service reads, but new reporters must
	// advertise EncoderOutputRelayModeLegacyStreamKey instead.
	EncoderOutputRelayModeStaticLegacyAlias EncoderOutputRelayMode = "static"
)

// RelayBindingIDPattern is the exact format of a non-secret fixed relay
// binding identity. It is intentionally not an ingest URL or stream key.
const RelayBindingIDPattern = `^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

// EncoderOutputRelayCapabilities is the non-secret subset of an
// Encoder/Recorder capabilities map that describes its output relay. A
// live_api_static relay requires an OutputRelayBindingID matching
// RelayBindingIDPattern.
type EncoderOutputRelayCapabilities struct {
	OutputRelayMode            EncoderOutputRelayMode `json:"output_relay_mode,omitempty"`
	OutputRelayBindingID       string                 `json:"output_relay_binding_id,omitempty"`
	SceneFramesMJPEGSRT        bool                   `json:"scene_frames_mjpeg_srt,omitempty"`
	WorkerFrameIngestMJPEGSRT  bool                   `json:"worker_frame_ingest_mjpeg_srt,omitempty"`
	SceneVideoSRTLegacy        bool                   `json:"scene_video_srt,omitempty"`
	WorkerVideoIngestSRTLegacy bool                   `json:"worker_video_ingest_srt,omitempty"`
}

// ErrorCodeYouTubeRelayStaticConfigChangedReload is returned when a fixed
// relay output or its stream assignment changed after a start request read its
// configuration. Callers must reload before starting again; no provider side
// effect was accepted for that stale configuration.
const (
	// ErrorCodeYouTubeRelayBindingClaimCheckFailed means Control Panel could not
	// determine whether a fixed relay binding is still claimed, so the mutation
	// is rejected rather than risking release while its ingress may be active.
	ErrorCodeYouTubeRelayBindingClaimCheckFailed = "youtube_relay_binding_claim_check_failed"
	// ErrorCodeYouTubeRelayBindingReleasePending means a fixed relay binding is
	// still claimed and must be released through its safe lifecycle before its
	// output or stream association can change.
	ErrorCodeYouTubeRelayBindingReleasePending        = "youtube_relay_binding_release_pending"
	ErrorCodeYouTubeRelayStaticConfigChangedReload    = "youtube_relay_static_config_changed_reload"
	ErrorCodeYouTubeLiveAPIRequiresManagedOutputRelay = "live_api_requires_managed_output_relay"
	// ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream prevents a
	// fixed-relay binding from being released while its stream might still own
	// the relay ingress.
	ErrorCodeYouTubeRelayStaticCompletionRequiresCompletedStream = "youtube_relay_static_completion_requires_completed_stream"
	// ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable is returned
	// when recovery cannot find the primary Encoder required to prove the
	// fixed relay is no longer in use.
	ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnavailable = "youtube_relay_static_recovery_encoder_stop_unavailable"
	// ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed is returned
	// when a possibly-dispatched fixed-relay start cannot be proven stopped.
	ErrorCodeYouTubeRelayStaticRecoveryEncoderStopUnconfirmed = "youtube_relay_static_recovery_encoder_stop_unconfirmed"
	// ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown requires explicit
	// operator investigation because a possibly-dispatched relay claim has no
	// trustworthy YouTube Broadcast identifier to complete.
	ErrorCodeYouTubeRelayStaticRecoveryBroadcastUnknown = "youtube_relay_static_recovery_broadcast_unknown"
	// ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid is returned for a
	// corrupted or unsupported durable recovery phase; the binding remains
	// fenced.
	ErrorCodeYouTubeRelayStaticRecoveryDispatchStateInvalid = "youtube_relay_static_recovery_dispatch_state_invalid"
	// ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed retains the fixed relay
	// claim when YouTube completion after a confirmed Encoder stop fails.
	ErrorCodeYouTubeRelayStaticRecoveryCompleteFailed = "youtube_relay_static_recovery_complete_failed"
)

type DiscordConfig struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	ServiceID            string    `json:"service_id,omitempty"`
	GuildID              string    `json:"guild_id,omitempty"`
	VoiceChannelID       string    `json:"voice_channel_id,omitempty"`
	TextChannelID        string    `json:"text_channel_id,omitempty"`
	BotTokenConfigured   bool      `json:"bot_token_configured,omitempty"`
	BotTokenFingerprint  string    `json:"bot_token_fingerprint,omitempty"`
	CaptionEnabled       bool      `json:"caption_enabled,omitempty"`
	STTProfileID         string    `json:"stt_profile_id,omitempty"`
	ReconnectEnabled     bool      `json:"reconnect_enabled,omitempty"`
	ReconnectMaxAttempts int       `json:"reconnect_max_attempts,omitempty"`
	ReconnectBaseDelay   string    `json:"reconnect_base_delay,omitempty"`
	ReconnectMaxDelay    string    `json:"reconnect_max_delay,omitempty"`
	AudioForwardEnabled  bool      `json:"audio_forward_enabled,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type DiscordConfigWriteRequest struct {
	Name                 string `json:"name"`
	ServiceID            string `json:"service_id,omitempty"`
	GuildID              string `json:"guild_id,omitempty"`
	VoiceChannelID       string `json:"voice_channel_id,omitempty"`
	TextChannelID        string `json:"text_channel_id,omitempty"`
	BotToken             string `json:"bot_token,omitempty"`
	CaptionEnabled       *bool  `json:"caption_enabled,omitempty"`
	STTProfileID         string `json:"stt_profile_id,omitempty"`
	ReconnectEnabled     *bool  `json:"reconnect_enabled,omitempty"`
	ReconnectMaxAttempts int    `json:"reconnect_max_attempts,omitempty"`
	ReconnectBaseDelay   string `json:"reconnect_base_delay,omitempty"`
	ReconnectMaxDelay    string `json:"reconnect_max_delay,omitempty"`
	AudioForwardEnabled  *bool  `json:"audio_forward_enabled,omitempty"`
}

type YouTubeOutput struct {
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Mode                   YouTubeOutputMode `json:"mode"`
	RTMPURL                string            `json:"rtmp_url,omitempty"`
	StreamKeyConfigured    bool              `json:"stream_key_configured,omitempty"`
	StreamKeyFingerprint   string            `json:"stream_key_fingerprint,omitempty"`
	WatchURL               string            `json:"watch_url,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	RelayBindingID         string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID   string            `json:"reusable_live_stream_id,omitempty"`
	BroadcastTitleTemplate string            `json:"broadcast_title_template,omitempty"`
	BroadcastDescription   string            `json:"broadcast_description,omitempty"`
	PrivacyStatus          string            `json:"privacy_status,omitempty"`
	LatencyPreference      string            `json:"latency_preference,omitempty"`
	EnableAutoStart        bool              `json:"enable_auto_start,omitempty"`
	EnableAutoStop         bool              `json:"enable_auto_stop,omitempty"`
	CompleteOnStop         bool              `json:"complete_on_stop"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

type YouTubeOutputWriteRequest struct {
	Name                   string            `json:"name"`
	Mode                   YouTubeOutputMode `json:"mode"`
	RTMPURL                string            `json:"rtmp_url,omitempty"`
	StreamKey              string            `json:"stream_key,omitempty"`
	WatchURL               string            `json:"watch_url,omitempty"`
	OAuthAccountID         string            `json:"oauth_account_id,omitempty"`
	RelayBindingID         string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID   string            `json:"reusable_live_stream_id,omitempty"`
	BroadcastTitleTemplate string            `json:"broadcast_title_template,omitempty"`
	BroadcastDescription   string            `json:"broadcast_description,omitempty"`
	PrivacyStatus          string            `json:"privacy_status,omitempty"`
	LatencyPreference      string            `json:"latency_preference,omitempty"`
	EnableAutoStart        *bool             `json:"enable_auto_start,omitempty"`
	EnableAutoStop         *bool             `json:"enable_auto_stop,omitempty"`
	CompleteOnStop         *bool             `json:"complete_on_stop,omitempty"`
}

type YouTubeRuntimeConfig struct {
	Mode                 YouTubeOutputMode `json:"mode"`
	OutputID             string            `json:"output_id,omitempty"`
	OAuthAccountID       string            `json:"oauth_account_id,omitempty"`
	BroadcastID          string            `json:"broadcast_id,omitempty"`
	LiveStreamID         string            `json:"live_stream_id,omitempty"`
	RelayBindingID       string            `json:"relay_binding_id,omitempty"`
	ReusableLiveStreamID string            `json:"reusable_live_stream_id,omitempty"`
	StreamKeySecretName  string            `json:"stream_key_secret_name,omitempty"`
	WatchURL             string            `json:"watch_url,omitempty"`
	DryRun               bool              `json:"dry_run,omitempty"`
	CompleteOnStop       bool              `json:"complete_on_stop,omitempty"`
	CompleteRetryCount   int               `json:"complete_retry_count,omitempty"`
	CompleteNextRetryAt  string            `json:"complete_next_retry_at,omitempty"`
	CompleteLastError    string            `json:"complete_last_error,omitempty"`
}

type YouTubeRelayStaticRecoveryResolveRequest struct {
	ConfirmExternalCleanup bool `json:"confirm_external_cleanup"`
}

type YouTubeRelayStaticRecoveryResolveResponse struct {
	Resolved       bool   `json:"resolved"`
	Cleanup        string `json:"cleanup"`
	RelayBindingID string `json:"relay_binding_id"`
}

type EncoderStartStreamRequest struct {
	StreamID               string               `json:"stream_id"`
	Name                   string               `json:"name"`
	InputURL               string               `json:"input_url,omitempty"`
	InputMode              string               `json:"input_mode,omitempty"`
	WorkerVideoIngest      bool                 `json:"worker_video_ingest,omitempty"`
	WorkerVideoIngestToken string               `json:"worker_video_ingest_token,omitempty"`
	RTMPURL                string               `json:"rtmp_url"`
	StreamKey              string               `json:"stream_key,omitempty"`
	StreamKeySecretName    string               `json:"stream_key_secret_name,omitempty"`
	EncoderProfileID       string               `json:"encoder_profile_id,omitempty"`
	OverlayProfileID       string               `json:"overlay_profile_id,omitempty"`
	EncoderAudioGainDB     float64              `json:"encoder_audio_gain_db,omitempty"`
	ArchiveProfileID       string               `json:"archive_profile_id,omitempty"`
	YouTubeRuntime         YouTubeRuntimeConfig `json:"youtube_runtime,omitempty"`
	ArchiveConfig          ArchiveRuntimeConfig `json:"archive_config,omitempty"`
	DryRun                 bool                 `json:"dry_run,omitempty"`
}

type EncoderRuntimeSettingsRequest struct {
	EncoderAudioGainDB float64 `json:"encoder_audio_gain_db"`
	OverlayProfileID   string  `json:"overlay_profile_id"`
}

type EncoderVideoIngest struct {
	URL        string `json:"url"`
	Passphrase string `json:"passphrase"`
	PBKeyLen   int    `json:"pbkeylen"`
}

// EncoderStartStreamResponse is an internal service-to-service response. The
// VideoIngest field must be consumed by the dispatcher and removed before any
// audit, status, or public API serialization.
type EncoderStartStreamResponse struct {
	StreamID     string              `json:"stream_id"`
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	StartedAtJST string              `json:"started_at_jst"`
	StoppedAtJST string              `json:"stopped_at_jst,omitempty"`
	Archive      map[string]string   `json:"archive"`
	Error        string              `json:"error,omitempty"`
	VideoIngest  *EncoderVideoIngest `json:"video_ingest,omitempty"`
}

type EncoderPackageStreamRequest struct {
	StreamID      string               `json:"stream_id"`
	Name          string               `json:"name"`
	StartedAt     time.Time            `json:"started_at,omitempty"`
	DryRun        bool                 `json:"dry_run,omitempty"`
	ArchiveConfig ArchiveRuntimeConfig `json:"archive_config,omitempty"`
}

type MissingStreamAssignmentsResponse struct {
	Code                string   `json:"code"`
	MissingServiceTypes []string `json:"missing_service_types"`
}

type ReadinessIssue struct {
	ServiceID   string `json:"service_id,omitempty"`
	ServiceType string `json:"service_type,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

const (
	ReadinessIssueMissingStreamAssignment              = "missing_stream_assignment"
	ReadinessIssueServiceCallTokenMissing              = "service_call_token_missing"
	ReadinessIssueServicePublicURLInvalid              = "service_public_url_invalid"
	ReadinessIssueServicePublicURLBlocked              = "service_public_url_blocked"
	ReadinessIssueEncoderPublicURLMissing              = "encoder_public_url_missing"
	ReadinessIssueEncoderPublicURLInvalid              = "encoder_public_url_invalid"
	ReadinessIssueEncoderPublicURLBlocked              = "encoder_public_url_blocked"
	ReadinessIssueServiceOffline                       = "service_offline"
	ReadinessIssueServiceHeartbeatStale                = "service_heartbeat_stale"
	ReadinessIssueDiscordAudioForwardUnavailable       = "discord_audio_forward_unavailable"
	ReadinessIssueDiscordAudioCaptureUnavailable       = "discord_audio_capture_unavailable"
	ReadinessIssueDiscordConfigRequired                = "discord_config_required"
	ReadinessIssueDiscordConfigNotFound                = "discord_config_not_found"
	ReadinessIssueDiscordConfigInvalid                 = "discord_config_invalid"
	ReadinessIssueDiscordConfigServiceMismatch         = "discord_config_service_mismatch"
	ReadinessIssueYouTubeOutputNotFound                = "youtube_output_not_found"
	ReadinessIssueYouTubeOutputInvalidConfig           = "youtube_output_invalid_config"
	ReadinessIssueYouTubeStreamKeyUnavailable          = "youtube_stream_key_unavailable"
	ReadinessIssueYouTubeLiveAPIUnavailable            = "youtube_live_api_unavailable"
	ReadinessIssueYouTubeOAuthAccountUnavailable       = "youtube_oauth_account_unavailable"
	ReadinessIssueYouTubeRelayStaticUnavailable        = "youtube_relay_static_unavailable"
	ReadinessIssueYouTubeRelayStaticBindingUnavailable = "youtube_relay_static_binding_unavailable"
	ReadinessIssueYouTubeRelayBindingStoreUnavailable  = "youtube_relay_binding_store_unavailable"
	ReadinessIssueYouTubeRelayBindingInUse             = "youtube_relay_binding_in_use"
	ReadinessIssueYouTubeRelayStaticRecoveryRequired   = "youtube_relay_static_recovery_required"
	ReadinessIssueArchiveProfileNotFound               = "archive_profile_not_found"
	ReadinessIssueArchiveProfileInvalidConfig          = "archive_profile_invalid_config"
	ReadinessIssueDriveDestinationNotFound             = "drive_destination_not_found"
	ReadinessIssueDriveDestinationUnavailable          = "drive_destination_unavailable"
	ReadinessIssueDriveOAuthAccountUnavailable         = "drive_oauth_account_unavailable"
)

var KnownStartReadinessIssueCodes = []string{
	ReadinessIssueMissingStreamAssignment,
	ReadinessIssueServiceCallTokenMissing,
	ReadinessIssueServicePublicURLInvalid,
	ReadinessIssueServicePublicURLBlocked,
	ReadinessIssueEncoderPublicURLMissing,
	ReadinessIssueEncoderPublicURLInvalid,
	ReadinessIssueEncoderPublicURLBlocked,
	ReadinessIssueServiceOffline,
	ReadinessIssueServiceHeartbeatStale,
	ReadinessIssueDiscordAudioForwardUnavailable,
	ReadinessIssueDiscordAudioCaptureUnavailable,
	ReadinessIssueDiscordConfigRequired,
	ReadinessIssueDiscordConfigNotFound,
	ReadinessIssueDiscordConfigInvalid,
	ReadinessIssueDiscordConfigServiceMismatch,
	ReadinessIssueYouTubeOutputNotFound,
	ReadinessIssueYouTubeOutputInvalidConfig,
	ReadinessIssueYouTubeStreamKeyUnavailable,
	ReadinessIssueYouTubeLiveAPIUnavailable,
	ReadinessIssueYouTubeOAuthAccountUnavailable,
	ReadinessIssueYouTubeRelayStaticUnavailable,
	ReadinessIssueYouTubeRelayStaticBindingUnavailable,
	ReadinessIssueYouTubeRelayBindingStoreUnavailable,
	ReadinessIssueYouTubeRelayBindingInUse,
	ReadinessIssueYouTubeRelayStaticRecoveryRequired,
	ReadinessIssueArchiveProfileNotFound,
	ReadinessIssueArchiveProfileInvalidConfig,
	ReadinessIssueDriveDestinationNotFound,
	ReadinessIssueDriveDestinationUnavailable,
	ReadinessIssueDriveOAuthAccountUnavailable,
}

type StartReadinessResponse struct {
	StreamID             string              `json:"stream_id"`
	Ready                bool                `json:"ready"`
	MissingServiceTypes  []string            `json:"missing_service_types"`
	Issues               []ReadinessIssue    `json:"issues"`
	AssignedServiceCount int                 `json:"assigned_service_count"`
	PrimaryServiceCount  int                 `json:"primary_service_count"`
	Assignments          []RegisteredService `json:"assignments,omitempty"`
}

type StreamJob struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Status           StreamStatus `json:"status"`
	DiscordConfigID  string       `json:"discord_config_id,omitempty"`
	DiscordGuildID   string       `json:"discord_guild_id,omitempty"`
	DiscordVoiceID   string       `json:"discord_voice_channel_id,omitempty"`
	DiscordTextID    string       `json:"discord_text_channel_id,omitempty"`
	EncoderProfileID string       `json:"encoder_profile_id,omitempty"`
	CaptionProfileID string       `json:"caption_profile_id,omitempty"`
	OverlayProfileID string       `json:"overlay_profile_id,omitempty"`
	ArchiveProfileID string       `json:"archive_profile_id,omitempty"`
	YouTubeOutputID  string       `json:"youtube_output_id,omitempty"`
	EncoderInputMode string       `json:"encoder_input_mode,omitempty"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type EncoderProfile struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Width               int    `json:"width"`
	Height              int    `json:"height"`
	FPS                 int    `json:"fps"`
	VideoCodec          string `json:"video_codec"`
	AudioCodec          string `json:"audio_codec"`
	VideoBitrateKbps    int    `json:"video_bitrate_kbps"`
	AudioBitrateKbps    int    `json:"audio_bitrate_kbps"`
	AudioSampleRateHz   int    `json:"audio_sample_rate_hz"`
	KeyframeIntervalSec int    `json:"keyframe_interval_sec"`
}

type ProfileKind string

const (
	ProfileEncoder       ProfileKind = "encoder"
	ProfileArchive       ProfileKind = "archive"
	ProfileCaption       ProfileKind = "caption"
	ProfileOverlay       ProfileKind = "overlay"
	ProfileDiscordConfig ProfileKind = "discord_config"
	ProfileYouTubeOutput ProfileKind = "youtube_output"
)

type Profile struct {
	ID        string         `json:"id"`
	Kind      ProfileKind    `json:"kind"`
	Name      string         `json:"name"`
	Config    map[string]any `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ProfileWriteRequest struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

type CaptionProfileConfig struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	Language         string `json:"language"`
	APIKeySecretName string `json:"api_key_secret_name"`
	EndpointingMS    int    `json:"endpointing_ms"`
	InterimResults   bool   `json:"interim_results"`
	SmartFormat      bool   `json:"smart_format"`
	DelayMS          int    `json:"delay_ms"`
}

type SessionRefreshResponse struct {
	Status            string    `json:"status"`
	IdleExpiresAt     time.Time `json:"idle_expires_at"`
	AbsoluteExpiresAt time.Time `json:"absolute_expires_at"`
}

type PublicAppSettings struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileConfigured          bool   `json:"turnstile_configured,omitempty"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	UpdatedAt                    string `json:"updated_at,omitempty"`
}

type ManagedAppSettings struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	SMTPEnabled                  bool   `json:"smtp_enabled"`
	SMTPHost                     string `json:"smtp_host,omitempty"`
	SMTPPort                     int    `json:"smtp_port,omitempty"`
	SMTPStartTLS                 bool   `json:"smtp_starttls"`
	SMTPFrom                     string `json:"smtp_from,omitempty"`
	SMTPUsername                 string `json:"smtp_username,omitempty"`
	SMTPPasswordConfigured       bool   `json:"smtp_password_configured,omitempty"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileConfigured          bool   `json:"turnstile_configured,omitempty"`
	UpdatedAt                    string `json:"updated_at,omitempty"`
}

type AppSettingsWriteRequest struct {
	AppName                      string `json:"app_name"`
	Timezone                     string `json:"timezone"`
	GoogleAnalyticsEnabled       bool   `json:"google_analytics_enabled,omitempty"`
	GoogleAnalyticsMeasurementID string `json:"google_analytics_measurement_id,omitempty"`
	SMTPEnabled                  bool   `json:"smtp_enabled"`
	SMTPHost                     string `json:"smtp_host,omitempty"`
	SMTPPort                     int    `json:"smtp_port,omitempty"`
	SMTPStartTLS                 bool   `json:"smtp_starttls"`
	SMTPFrom                     string `json:"smtp_from,omitempty"`
	SMTPUsername                 string `json:"smtp_username,omitempty"`
	SMTPPassword                 string `json:"smtp_password,omitempty"`
	TurnstileEnabled             bool   `json:"turnstile_enabled,omitempty"`
	TurnstileSiteKey             string `json:"turnstile_site_key,omitempty"`
	TurnstileSecret              string `json:"turnstile_secret,omitempty"`
}

type SecuritySettings struct {
	PasswordMinLength        int      `json:"password_min_length"`
	PasswordHash             string   `json:"password_hash"`
	LoginLockoutThreshold    int      `json:"login_lockout_threshold"`
	SessionIdleTimeoutMin    int      `json:"session_idle_timeout_min"`
	SessionAbsoluteLifetimeH int      `json:"session_absolute_lifetime_h"`
	RememberMeEnabled        bool     `json:"remember_me_enabled"`
	MFAMode                  string   `json:"mfa_mode"`
	MFARequiredRoles         []string `json:"mfa_required_roles,omitempty"`
	MFASupportedMethods      []string `json:"mfa_supported_methods,omitempty"`
	PasskeyStatus            string   `json:"passkey_status,omitempty"`
	UpdatedAt                string   `json:"updated_at,omitempty"`
}

type StreamPreviewLink struct {
	StreamID  string    `json:"stream_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SecretStatus struct {
	Name        string `json:"name"`
	Configured  bool   `json:"configured"`
	Fingerprint string `json:"fingerprint,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type SecretUpdateRequest struct {
	Value string `json:"value"`
}

type ArchiveUploadResult struct {
	DryRun              bool              `json:"dry_run"`
	FolderIDConfigured  bool              `json:"folder_id_configured,omitempty"`
	FolderIDFingerprint string            `json:"folder_id_fingerprint,omitempty"`
	FileCount           int               `json:"file_count,omitempty"`
	FileFingerprints    map[string]string `json:"file_fingerprints,omitempty"`
	Attempts            int               `json:"attempts"`
}

type ArchiveMetadata struct {
	StreamID     string              `json:"stream_id"`
	Name         string              `json:"name"`
	StartedAtJST string              `json:"started_at_jst"`
	Archive      map[string]string   `json:"archive"`
	Upload       ArchiveUploadResult `json:"upload"`
	Commands     []map[string]any    `json:"commands,omitempty"`
	Extra        map[string]any      `json:"extra,omitempty"`
}

type AuditEvent struct {
	ID            string         `json:"id"`
	Timestamp     time.Time      `json:"timestamp"`
	ActorUserID   string         `json:"actor_user_id,omitempty"`
	ActorUsername string         `json:"actor_username,omitempty"`
	ActorIP       string         `json:"actor_ip,omitempty"`
	UserAgent     string         `json:"user_agent,omitempty"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id,omitempty"`
	Result        string         `json:"result"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	RequestID     string         `json:"request_id"`
}

type SignalType string

const (
	SignalHeartbeat         SignalType = "heartbeat"
	SignalMetric            SignalType = "metric"
	SignalLog               SignalType = "log"
	SignalEvent             SignalType = "event"
	SignalWarning           SignalType = "warning"
	SignalError             SignalType = "error"
	SignalIncident          SignalType = "incident"
	SignalDiagnosticReport  SignalType = "diagnostic_report"
	SignalRemediationAction SignalType = "remediation_action"
	SignalNotificationEvent SignalType = "notification_event"
)

type ObservabilitySignal struct {
	Type        SignalType         `json:"type"`
	Name        string             `json:"name"`
	ServiceID   string             `json:"service_id"`
	ServiceType ServiceType        `json:"service_type"`
	StreamID    string             `json:"stream_id,omitempty"`
	Status      string             `json:"status,omitempty"`
	Value       *float64           `json:"value,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Attributes  map[string]any     `json:"attributes,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

type MetricName string

const (
	MetricStreamStatus               MetricName = "stream.status"
	MetricStreamStartDurationMS      MetricName = "stream.start_duration_ms"
	MetricStreamLiveDurationSec      MetricName = "stream.live_duration_sec"
	MetricStreamStopDurationMS       MetricName = "stream.stop_duration_ms"
	MetricStreamRestartCount         MetricName = "stream.restart_count"
	MetricEncoderProcessAlive        MetricName = "encoder.process_alive"
	MetricEncoderOutputFPS           MetricName = "encoder.output_fps"
	MetricEncoderOutputBitrateKbps   MetricName = "encoder.output_bitrate_kbps"
	MetricEncoderDroppedFramesTotal  MetricName = "encoder.dropped_frames_total"
	MetricEncoderEncodeLagMS         MetricName = "encoder.encode_lag_ms"
	MetricEncoderAudioLevelDB        MetricName = "encoder.audio_level_db"
	MetricEncoderAudioSilenceSec     MetricName = "encoder.audio_silence_sec"
	MetricEncoderAudioClippingTotal  MetricName = "encoder.audio_clipping_total"
	MetricEncoderRTMPReconnectCount  MetricName = "encoder.rtmp_reconnect_count"
	MetricRecorderFileSizeBytes      MetricName = "recorder.file_size_bytes"
	MetricRecorderWriteBitrateKbps   MetricName = "recorder.write_bitrate_kbps"
	MetricRecorderDiskFreeBytes      MetricName = "recorder.disk_free_bytes"
	MetricRecorderRemuxDurationMS    MetricName = "recorder.remux_duration_ms"
	MetricSRTPacketLossPercent       MetricName = "srt.packet_loss_percent"
	MetricSRTRTTMS                   MetricName = "srt.rtt_ms"
	MetricSRTJitterMS                MetricName = "srt.jitter_ms"
	MetricSRTBandwidthMbps           MetricName = "srt.bandwidth_mbps"
	MetricRTPPacketLossPercent       MetricName = "rtp.packet_loss_percent"
	MetricRTPJitterMS                MetricName = "rtp.jitter_ms"
	MetricMediaInputBitrateKbps      MetricName = "media.input_bitrate_kbps"
	MetricMediaInputTimeoutSec       MetricName = "media.input_timeout_sec"
	MetricDiscordGatewayConnected    MetricName = "discord.gateway_connected"
	MetricDiscordVoiceConnected      MetricName = "discord.voice_connected"
	MetricDiscordAudioReceiving      MetricName = "discord.audio_receiving"
	MetricDiscordAudioPacketsTotal   MetricName = "discord.audio_packets_total"
	MetricDiscordAudioForwardedTotal MetricName = "discord.audio_forwarded_total"
	MetricDiscordAudioForwardErrors  MetricName = "discord.audio_forward_errors_total"
	MetricDiscordAudioLastPacketAge  MetricName = "discord.audio_last_packet_age_sec"
	MetricDiscordAudioLastForwardAge MetricName = "discord.audio_last_forward_age_sec"
	MetricDiscordParticipantCount    MetricName = "discord.participant_count"
	MetricDiscordWorkerEventFailures MetricName = "discord.worker_event_publish_failures_total"
	MetricDiscordReconnectCount      MetricName = "discord.reconnect_count"
	MetricDiscordVoiceDisconnects    MetricName = "discord.voice_disconnect_count"
	MetricWorkerHeartbeatAgeSec      MetricName = "worker.heartbeat_age_sec"
	MetricWorkerOverlayEventsTotal   MetricName = "worker.overlay_events_total"
	MetricWorkerCaptionEventsTotal   MetricName = "worker.caption_events_total"
	MetricWorkerSceneUpdatesTotal    MetricName = "worker.scene_updates_total"
	MetricWorkerEventSendFailures    MetricName = "worker.event_send_failures_total"
	MetricArchiveFinalMKVExists      MetricName = "archive.final_mkv_exists"
	MetricArchiveFinalMP4Exists      MetricName = "archive.final_mp4_exists"
	MetricArchivePackageStatus       MetricName = "archive.package_status"
	MetricGDriveUploadStatus         MetricName = "gdrive.upload_status"
	MetricGDriveUploadProgress       MetricName = "gdrive.upload_progress_percent"
	MetricGDriveUploadRetryCount     MetricName = "gdrive.upload_retry_count"
	MetricGDriveUploadDurationSec    MetricName = "gdrive.upload_duration_sec"
	MetricGDriveUploadFileCount      MetricName = "gdrive.upload_file_count"
	MetricGDriveUploadFolderProof    MetricName = "gdrive.upload_folder_fingerprint_present"
	MetricGDriveUploadFinalMP4Proof  MetricName = "gdrive.upload_final_mp4_fingerprint_present"
	MetricGDriveUploadMetadataProof  MetricName = "gdrive.upload_metadata_fingerprint_present"
	MetricHostCPUPercent             MetricName = "host.cpu_percent"
	MetricHostMemoryPercent          MetricName = "host.memory_percent"
	MetricHostDiskFreeBytes          MetricName = "host.disk_free_bytes"
	MetricHostNetworkTxBPS           MetricName = "host.network_tx_bps"
	MetricHostNetworkRxBPS           MetricName = "host.network_rx_bps"
)

type IncidentRule string

const (
	RuleHeartbeatTimeout          IncidentRule = "heartbeat_timeout"
	RuleEncoderProcessExited      IncidentRule = "encoder_process_exited"
	RuleRecorderNotWriting        IncidentRule = "recorder_not_writing"
	RuleArchiveRemuxSlow          IncidentRule = "archive_remux_slow"
	RuleArchivePackageFailed      IncidentRule = "archive_package_failed"
	RuleGDriveUploadFailed        IncidentRule = "gdrive_upload_failed"
	RuleGDriveUploadRetryHigh     IncidentRule = "gdrive_upload_retry_high"
	RuleHighPacketLoss            IncidentRule = "high_packet_loss"
	RuleRTMPSReconnectLoop        IncidentRule = "rtmps_reconnect_loop"
	RuleEncoderLowFPS             IncidentRule = "encoder_low_fps"
	RuleEncoderBitrateLow         IncidentRule = "encoder_bitrate_low"
	RuleEncoderDroppedFrames      IncidentRule = "encoder_dropped_frames_high"
	RuleAudioSilence              IncidentRule = "audio_silence"
	RuleAudioClipping             IncidentRule = "audio_clipping"
	RuleDiscordAudioNotReceiving  IncidentRule = "discord_audio_not_receiving"
	RuleDiscordAudioForwardFailed IncidentRule = "discord_audio_forward_failed"
	RuleDiscordReconnectLoop      IncidentRule = "discord_reconnect_loop"
	RuleDiscordVoiceDisconnected  IncidentRule = "discord_voice_disconnected"
	RuleMediaInputTimeout         IncidentRule = "media_input_timeout"
	RuleDiskLow                   IncidentRule = "disk_low"
	RuleStreamStartTimeout        IncidentRule = "stream_start_timeout"
	RuleStreamStopTimeout         IncidentRule = "stream_stop_timeout"
	RuleUnexpectedStopped         IncidentRule = "unexpected_stopped"
	RuleWorkerEventSendFailed     IncidentRule = "worker_event_send_failed"
)

type IncidentSeverity string

const (
	SeverityInfo     IncidentSeverity = "info"
	SeverityWarning  IncidentSeverity = "warning"
	SeverityError    IncidentSeverity = "error"
	SeverityCritical IncidentSeverity = "critical"
)

type IncidentStatus string

const (
	IncidentOpen          IncidentStatus = "open"
	IncidentAcknowledged  IncidentStatus = "acknowledged"
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentMitigated     IncidentStatus = "mitigated"
	IncidentResolved      IncidentStatus = "resolved"
	IncidentIgnored       IncidentStatus = "ignored"
)

type Incident struct {
	ID         string           `json:"id"`
	Rule       IncidentRule     `json:"rule"`
	Severity   IncidentSeverity `json:"severity"`
	Status     IncidentStatus   `json:"status"`
	SummaryJA  string           `json:"summary_ja"`
	ServiceID  string           `json:"service_id"`
	StreamID   string           `json:"stream_id,omitempty"`
	SignalID   string           `json:"signal_id,omitempty"`
	Report     DiagnosticReport `json:"diagnostic_report"`
	OpenedAt   time.Time        `json:"opened_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`
}

type DiagnosticRerunOutcome string

const (
	DiagnosticRerunOutcomeEvaluated    DiagnosticRerunOutcome = "evaluated"
	DiagnosticRerunOutcomeInconclusive DiagnosticRerunOutcome = "inconclusive"
)

type DiagnosticRerunReason string

const (
	DiagnosticRerunReasonSavedSignalMissing         DiagnosticRerunReason = "saved_signal_missing"
	DiagnosticRerunReasonSavedSignalNotFound        DiagnosticRerunReason = "saved_signal_not_found"
	DiagnosticRerunReasonSavedSignalNoLongerMatches DiagnosticRerunReason = "saved_signal_no_longer_matches_rule"
	DiagnosticRerunReasonIncidentUpdatedDuringRerun DiagnosticRerunReason = "incident_updated_during_rerun"
)

// DiagnosticRerunResponse reports a diagnostic-only re-evaluation. Incident
// lifecycle and remediation state are intentionally unchanged.
type DiagnosticRerunResponse struct {
	Incident Incident               `json:"incident"`
	Outcome  DiagnosticRerunOutcome `json:"outcome"`
	Reason   DiagnosticRerunReason  `json:"reason,omitempty"`
}

type DiagnosticReport struct {
	Summary            string   `json:"summary"`
	LikelyCause        string   `json:"likely_cause"`
	Confidence         float64  `json:"confidence"`
	Evidence           []string `json:"evidence"`
	Impact             string   `json:"impact"`
	RecommendedActions []string `json:"recommended_actions"`
	SafeAutoCandidates []string `json:"safe_auto_remediation_candidates"`
	ApprovalRequired   []string `json:"actions_requiring_approval"`
}

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

type NotificationEventType string

const (
	NotificationStreamStarted              NotificationEventType = "stream.started"
	NotificationStreamLive                 NotificationEventType = "stream.live"
	NotificationStreamCompleted            NotificationEventType = "stream.completed"
	NotificationStreamFailed               NotificationEventType = "stream.failed"
	NotificationStreamWarning              NotificationEventType = "stream.warning"
	NotificationStreamError                NotificationEventType = "stream.error"
	NotificationIncidentOpened             NotificationEventType = "incident.opened"
	NotificationIncidentUpdated            NotificationEventType = "incident.updated"
	NotificationIncidentResolved           NotificationEventType = "incident.resolved"
	NotificationDiagnosticCreated          NotificationEventType = "diagnostic.created"
	NotificationRemediationPendingApproval NotificationEventType = "remediation.pending_approval"
	NotificationRemediationExecuted        NotificationEventType = "remediation.executed"
	NotificationArchiveUploadCompleted     NotificationEventType = "archive.upload.completed"
	NotificationArchiveUploadFailed        NotificationEventType = "archive.upload.failed"
	NotificationServiceOffline             NotificationEventType = "service.offline"
	NotificationServiceRecovered           NotificationEventType = "service.recovered"
	NotificationAdminAudit                 NotificationEventType = "admin.audit"
)

// NotificationEventWriteRequest is the secret-free event envelope accepted by
// Observability's notification event endpoint. Metadata is deliberately not
// part of this contract; service_id/details carry only display-safe context.
type NotificationEventWriteRequest struct {
	EventType     NotificationEventType `json:"event_type"`
	Severity      string                `json:"severity,omitempty"`
	Status        string                `json:"status,omitempty"`
	Action        string                `json:"action"`
	ServiceID     string                `json:"service_id,omitempty"`
	ResourceType  string                `json:"resource_type,omitempty"`
	ResourceID    string                `json:"resource_id,omitempty"`
	ActorUsername string                `json:"actor_username,omitempty"`
	Summary       string                `json:"summary,omitempty"`
	Details       string                `json:"details,omitempty"`
	Timestamp     string                `json:"timestamp,omitempty"`
}

// NotificationDeliveryResult is a secret-safe delivery attempt returned by
// notification-events. Target and Error must already be masked or sanitized.
type NotificationDeliveryResult struct {
	EventType NotificationEventType `json:"event_type"`
	Channel   string                `json:"channel"`
	Target    string                `json:"target"`
	Status    string                `json:"status"`
	Error     string                `json:"error,omitempty"`
}

type NotificationChannel struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Type                   string    `json:"type"`
	Enabled                bool      `json:"enabled"`
	UsesGlobalSMTP         bool      `json:"uses_global_smtp"`
	MaskedWebhookURL       string    `json:"masked_webhook_url,omitempty"`
	SMTPPasswordConfigured bool      `json:"smtp_password_configured,omitempty"`
	MaskedEmailTarget      string    `json:"masked_email_target,omitempty"`
	SeverityFilter         []string  `json:"severity_filter,omitempty"`
	EventTypeFilter        []string  `json:"event_type_filter,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type NotificationChannelWriteRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	UsesGlobalSMTP  *bool    `json:"uses_global_smtp,omitempty"`
	// Deprecated: direct SMTP settings are retained only for existing
	// Observability channels. New email channels use UsesGlobalSMTP.
	SMTPHost        string   `json:"smtp_host,omitempty"`
	SMTPPort        int      `json:"smtp_port,omitempty"`
	SMTPTLS         bool     `json:"smtp_tls,omitempty"`
	SMTPFrom        string   `json:"smtp_from,omitempty"`
	SMTPUsername    string   `json:"smtp_username,omitempty"`
	SMTPPassword    string   `json:"smtp_password,omitempty"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	EventTypeFilter []string `json:"event_type_filter,omitempty"`
}

// ControlNotificationChannelUpdateRequest is the browser-facing Control Panel
// write shape. It deliberately excludes per-channel SMTP configuration and the
// global/legacy SMTP mode selector. Omitted recipients preserve the existing
// masked recipient set and delivery mode.
type ControlNotificationChannelUpdateRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Enabled         bool     `json:"enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	EventTypeFilter []string `json:"event_type_filter,omitempty"`
}

// ControlNotificationChannelCreateRequest has the same wire fields as an
// update. The create schema additionally requires recipients for email and the
// backend always selects globally managed SMTP for new email channels.
type ControlNotificationChannelCreateRequest = ControlNotificationChannelUpdateRequest

type OAuthProvider struct {
	ID                     string    `json:"id"`
	ProviderType           string    `json:"provider_type"`
	Name                   string    `json:"name"`
	Enabled                bool      `json:"enabled"`
	ClientID               string    `json:"client_id"`
	ClientSecretConfigured bool      `json:"client_secret_configured"`
	Scopes                 []string  `json:"scopes"`
	AllowedDomains         []string  `json:"allowed_domains"`
	AutoProvision          bool      `json:"auto_provision"`
	DefaultRoleIDs         []string  `json:"default_role_ids,omitempty"`
	RedirectURI            string    `json:"redirect_uri"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type OAuthProviderWriteRequest struct {
	ProviderType   string   `json:"provider_type"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	AutoProvision  bool     `json:"auto_provision,omitempty"`
	DefaultRoleIDs []string `json:"default_role_ids,omitempty"`
	// RedirectURI must match AUTOSTREAM_PUBLIC_URL scheme/host in production.
	// /auth/oauth/callback is used for login providers.
	// /integrations/oauth-accounts/callback is only for Google Drive/YouTube connected accounts.
	RedirectURI string `json:"redirect_uri"`
}

// OAuthAccountPurpose is derived from the effective OAuth scopes. It is
// returned by connected-account reads and OAuth consent starts; it never
// contains an OAuth scope or credential.
type OAuthAccountPurpose string

const (
	OAuthAccountPurposeDrive        OAuthAccountPurpose = "drive"
	OAuthAccountPurposeYouTube      OAuthAccountPurpose = "youtube"
	OAuthAccountPurposeDriveYouTube OAuthAccountPurpose = "drive_youtube"
	OAuthAccountPurposeUnknown      OAuthAccountPurpose = "unknown"
)

type OAuthAccount struct {
	ID                               string              `json:"id"`
	ProviderID                       string              `json:"provider_id"`
	ProviderType                     string              `json:"provider_type"`
	ProviderName                     string              `json:"provider_name,omitempty"`
	AccountLabel                     string              `json:"account_label"`
	AccountPurpose                   OAuthAccountPurpose `json:"account_purpose"`
	DisplayName                      string              `json:"display_name,omitempty"`
	Subject                          string              `json:"subject,omitempty"`
	Email                            string              `json:"email,omitempty"`
	Scopes                           []string            `json:"scopes"`
	RefreshTokenConfigured           bool                `json:"refresh_token_configured"`
	TokenFingerprint                 string              `json:"token_fingerprint,omitempty"`
	RefreshTokenUpdatedAt            string              `json:"refresh_token_updated_at,omitempty"`
	AccessTokenRefreshedAt           string              `json:"access_token_refreshed_at,omitempty"`
	AccessTokenRefreshAttemptedAt    string              `json:"access_token_refresh_attempted_at,omitempty"`
	AccessTokenRefreshFailedAt       string              `json:"access_token_refresh_failed_at,omitempty"`
	AccessTokenRefreshFailureCode    string              `json:"access_token_refresh_failure_code,omitempty"`
	AccessTokenRefreshRelinkRequired bool                `json:"access_token_refresh_relink_required"`
	CreatedAt                        time.Time           `json:"created_at"`
	UpdatedAt                        time.Time           `json:"updated_at"`
}

type OAuthAccountWriteRequest struct {
	ProviderID   string   `json:"provider_id"`
	ProviderType string   `json:"provider_type"`
	AccountLabel string   `json:"account_label"`
	Subject      string   `json:"subject,omitempty"`
	Email        string   `json:"email,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
}

type OAuthAccountConnectionStartRequest struct {
	ProviderID     string `json:"provider_id,omitempty"`
	OAuthAccountID string `json:"oauth_account_id,omitempty"`
	AccountLabel   string `json:"account_label,omitempty"`
	AccountPurpose string `json:"account_purpose,omitempty"`
	RedirectAfter  string `json:"redirect_after,omitempty"`
}

type OAuthAccountConnectionStartResponse struct {
	Provider         OAuthLoginProvider  `json:"provider"`
	AuthorizationURL string              `json:"authorization_url"`
	State            string              `json:"state"`
	Nonce            string              `json:"nonce"`
	ExpiresAt        time.Time           `json:"expires_at"`
	AccountLabel     string              `json:"account_label"`
	AccountPurpose   OAuthAccountPurpose `json:"account_purpose"`
	Relink           bool                `json:"relink"`
	Scopes           []string            `json:"scopes"`
}

type OAuthAccountConnectionCallbackRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	State        string `json:"state"`
	Code         string `json:"code"`
	AccountLabel string `json:"account_label,omitempty"`
}

type DriveDestination struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	AuthMode            string    `json:"auth_mode"`
	OAuthAccountID      string    `json:"oauth_account_id,omitempty"`
	FolderIDConfigured  bool      `json:"folder_id_configured"`
	FolderIDFingerprint string    `json:"folder_id_fingerprint,omitempty"`
	MaskedFolderID      string    `json:"masked_folder_id,omitempty"`
	SharedDrive         bool      `json:"shared_drive"`
	BasePath            string    `json:"base_path"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type DriveDestinationWriteRequest struct {
	Name           string `json:"name"`
	AuthMode       string `json:"auth_mode"`
	OAuthAccountID string `json:"oauth_account_id,omitempty"`
	FolderID       string `json:"folder_id,omitempty"`
	SharedDrive    bool   `json:"shared_drive"`
	BasePath       string `json:"base_path"`
}

type OAuthLoginProvider struct {
	ID           string   `json:"id"`
	ProviderType string   `json:"provider_type"`
	Name         string   `json:"name"`
	Scopes       []string `json:"scopes"`
	RedirectURI  string   `json:"redirect_uri"`
}

type OAuthLoginStartRequest struct {
	RedirectAfter string `json:"redirect_after,omitempty"`
}

type OAuthLoginStartResponse struct {
	Provider         OAuthLoginProvider `json:"provider"`
	AuthorizationURL string             `json:"authorization_url"`
	State            string             `json:"state"`
	Nonce            string             `json:"nonce"`
	ExpiresAt        time.Time          `json:"expires_at"`
}

type OAuthLoginCallbackRequest struct {
	ProviderID string `json:"provider_id,omitempty"`
	State      string `json:"state"`
	Code       string `json:"code"`
}

type OAuthUserLink struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ProviderID   string    `json:"provider_id"`
	ProviderType string    `json:"provider_type"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OAuthUserLinkWriteRequest struct {
	ProviderID   string `json:"provider_id"`
	ProviderType string `json:"provider_type,omitempty"`
	Subject      string `json:"subject"`
	Email        string `json:"email,omitempty"`
}

type NotificationDelivery struct {
	ID         string                `json:"id"`
	EventType  NotificationEventType `json:"event_type"`
	Channel    string                `json:"channel"`
	Target     string                `json:"target"`
	IncidentID string                `json:"incident_id,omitempty"`
	Status     string                `json:"status"`
	Error      string                `json:"error,omitempty"`
	Metadata   map[string]any        `json:"metadata,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
}

type ServiceRemediationExecuteRequest struct {
	ActionID   string `json:"action_id"`
	Action     string `json:"action"`
	IncidentID string `json:"incident_id"`
	StreamID   string `json:"stream_id"`
}
