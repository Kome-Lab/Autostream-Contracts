package contracts

import (
	"time"
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
	SystemUpdateTargetUpdateAgent     SystemUpdateTargetType = "update_agent"
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

// SystemUpdatePortReconfiguration is a versioned immutable job/command plan.
// Version 2 uses Before/Target/Rollback and a separate SystemUpdatePortResultV2.
// Flat fields and Result remain solely for exact decoding/recovery of saved
// legacy plans; validators reject every mixed-version payload.
type SystemUpdatePortReconfiguration struct {
	PortContractVersion            int                                    `json:"port_contract_version,omitempty"`
	Mode                           SystemUpdatePortMode                   `json:"mode,omitempty"`
	Before                         *SystemUpdatePortSnapshotRef           `json:"before,omitempty"`
	Target                         *SystemUpdatePortSnapshotRef           `json:"target,omitempty"`
	Rollback                       *SystemUpdatePortSnapshotRef           `json:"rollback,omitempty"`
	DockerBaseline                 *SystemUpdatePortDockerBaseline        `json:"docker_baseline,omitempty"`
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
	ProtocolVersion          int                   `json:"protocol_version"`
	Operation                SystemUpdateOperation `json:"operation,omitempty"`
	TargetID                 string                `json:"target_id"`
	Strategy                 SystemUpdateStrategy  `json:"strategy,omitempty"`
	NewPort                  int                   `json:"new_port,omitempty"`
	PortContractVersion      int                   `json:"port_contract_version,omitempty"`
	Mode                     SystemUpdatePortMode  `json:"mode,omitempty"`
	ExpectedSnapshotID       string                `json:"expected_snapshot_id,omitempty"`
	NewLocalListenPort       int                   `json:"new_local_listen_port,omitempty"`
	NewAdvertisedPort        int                   `json:"new_advertised_port,omitempty"`
	NewPublishedPort         int                   `json:"new_published_port,omitempty"`
	NewContainerPort         int                   `json:"new_container_port,omitempty"`
	ExpectedEndpointRevision int64                 `json:"expected_endpoint_revision,omitempty"`
	IdempotencyKey           string                `json:"idempotency_key"`
	DesiredRevision          int64                 `json:"desired_revision"`
	Fence                    int64                 `json:"fence"`
	RequiredCapability       UpdaterCapability     `json:"required_capability"`
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
	ProtocolVersion         int                              `json:"protocol_version"`
	TargetID                string                           `json:"target_id"`
	TargetType              SystemUpdateTargetType           `json:"target_type"`
	Name                    string                           `json:"name"`
	HostID                  string                           `json:"host_id"`
	CurrentVersion          string                           `json:"current_version,omitempty"`
	LatestVersion           string                           `json:"latest_version,omitempty"`
	UpdateAvailable         bool                             `json:"update_available"`
	DeploymentMode          SystemUpdateDeploymentMode       `json:"deployment_mode,omitempty"`
	UpdaterID               string                           `json:"updater_id"`
	UpdaterOnline           bool                             `json:"updater_online"`
	Capabilities            []UpdaterCapability              `json:"capabilities"`
	DesiredRevision         int64                            `json:"desired_revision"`
	AppliedRevision         int64                            `json:"applied_revision"`
	Fence                   int64                            `json:"fence"`
	UpdaterHealth           *UpdaterHealth                   `json:"updater_health"`
	ApplicationProbe        *ApplicationRuntimeIdentityProbe `json:"application_probe"`
	Eligible                bool                             `json:"eligible"`
	BlockedReason           string                           `json:"blocked_reason,omitempty"`
	EligibleOperations      []SystemUpdateOperation          `json:"eligible_operations,omitempty"`
	OperationBlockedReasons map[string]string                `json:"operation_blocked_reasons,omitempty"`
	Busy                    bool                             `json:"busy"`
	CurrentStreamID         string                           `json:"current_stream_id,omitempty"`
	UpdateCheckSource       string                           `json:"update_check_source,omitempty"`
	UpdateCheckError        string                           `json:"update_check_error,omitempty"`
	SafeError               *V2UpdaterSafeError              `json:"safe_error,omitempty"`
	PortMapping             *SystemUpdatePortMapping         `json:"port_mapping,omitempty"`
	PortContractVersion     int                              `json:"port_contract_version,omitempty"`
	PortPolicySnapshotID    string                           `json:"port_policy_snapshot_id,omitempty"`
	LocalListenPort         int                              `json:"local_listen_port,omitempty"`
	EndpointRevision        int64                            `json:"endpoint_revision,omitempty"`
	AppliedEndpointRevision int64                            `json:"applied_endpoint_revision,omitempty"`
	AppliedConfigRevision   int64                            `json:"applied_config_revision,omitempty"`
	OwnershipEpoch          int64                            `json:"ownership_epoch,omitempty"`
	PortModes               []SystemUpdatePortMode           `json:"port_modes,omitempty"`
}

type SystemUpdateJob struct {
	ProtocolVersion         int                              `json:"protocol_version"`
	ID                      string                           `json:"id"`
	TargetID                string                           `json:"target_id"`
	TargetType              SystemUpdateTargetType           `json:"target_type"`
	ExecutionHostID         string                           `json:"host_id"`
	TransportMode           UpdateTransportMode              `json:"transport_mode"`
	OwnershipEpoch          int64                            `json:"ownership_epoch"`
	PolicyRevision          int64                            `json:"policy_revision"`
	DeploymentMode          SystemUpdateDeploymentMode       `json:"deployment_mode"`
	CurrentVersion          string                           `json:"current_version"`
	TargetVersion           string                           `json:"target_version"`
	Strategy                SystemUpdateStrategy             `json:"strategy"`
	Status                  SystemUpdateStatus               `json:"status"`
	IdempotencyKey          string                           `json:"idempotency_key"`
	UpdaterID               string                           `json:"updater_id"`
	AuthorizationID         string                           `json:"authorization_id"`
	CanonicalPayloadDigest  string                           `json:"canonical_payload_digest"`
	DesiredRevision         int64                            `json:"desired_revision"`
	Fence                   int64                            `json:"fence"`
	Outcome                 string                           `json:"outcome"`
	RequiredCapability      UpdaterCapability                `json:"required_capability"`
	AutomaticResendAllowed  *bool                            `json:"automatic_resend_allowed"`
	SafeError               *V2UpdaterSafeError              `json:"safe_error,omitempty"`
	RequestedBy             string                           `json:"requested_by,omitempty"`
	LeaseGeneration         int64                            `json:"lease_generation"`
	LeaseExpiresAt          *time.Time                       `json:"lease_expires_at,omitempty"`
	Sequence                int64                            `json:"sequence"`
	Progress                int                              `json:"progress"`
	Code                    string                           `json:"code,omitempty"`
	Message                 string                           `json:"message,omitempty"`
	ArtifactDigest          string                           `json:"artifact_digest,omitempty"`
	PreviousDigest          string                           `json:"previous_digest,omitempty"`
	Operation               SystemUpdateOperation            `json:"operation,omitempty"`
	PortReconfigure         *SystemUpdatePortReconfiguration `json:"port_reconfigure,omitempty"`
	PortResult              *SystemUpdatePortResultV2        `json:"port_result,omitempty"`
	RecoveryRequired        bool                             `json:"recovery_required,omitempty"`
	LastRecoveryObservation *SystemUpdatePortResultV2        `json:"last_recovery_observation,omitempty"`
	CreatedAt               time.Time                        `json:"created_at"`
	UpdatedAt               time.Time                        `json:"updated_at"`
	ClaimedAt               *time.Time                       `json:"claimed_at,omitempty"`
	CompletedAt             *time.Time                       `json:"completed_at,omitempty"`
	CanceledAt              *time.Time                       `json:"canceled_at,omitempty"`
}

type SystemUpdatesResponse struct {
	Updaters []SystemUpdateAgentStatus `json:"updaters"`
	Hosts    []SystemUpdateHostStatus  `json:"hosts"`
	Targets  []SystemUpdateTarget      `json:"targets"`
	Jobs     []SystemUpdateJob         `json:"jobs"`
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
	ProtocolMajor       int                        `json:"protocol_major,omitempty"`
	MinimumAgentVersion string                     `json:"minimum_agent_version,omitempty"`
	Components          []ReleaseManifestComponent `json:"components"`
}

type ReleaseManifestComponent struct {
	Service            string            `json:"service"`
	SourceVersion      string            `json:"source_version"`
	Commit             string            `json:"commit,omitempty"`
	ProtocolMajor      int               `json:"protocol_major,omitempty"`
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
