package contracts

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
	CapabilitySceneFramesMJPEGSRT       = "scene_frames_mjpeg_srt"
	CapabilityWorkerFrameIngestMJPEGSRT = "worker_frame_ingest_mjpeg_srt"
	CapabilitySceneAppearanceV1         = "scene_appearance_v1"
	CapabilityLiveVideoCoverV1          = "live_video_cover_v1"
	CapabilityDiscordResolvedTargetV2   = "discord_resolved_target_v2"
)

// SupportsDiscordResolvedTargetV2 is the fail-closed mixed-fleet negotiation
// boundary for Discord Bot start requests. Only an actual boolean true in the
// assigned Bot's current capability map authorizes the strict v2 DTO; absence,
// false, unknown types, and a version string alone retain the legacy DTO.
func SupportsDiscordResolvedTargetV2(capabilities map[string]any) bool {
	supported, ok := capabilities[CapabilityDiscordResolvedTargetV2].(bool)
	return ok && supported
}

type UpdateTransportMode string

const UpdateTransportPullV2 UpdateTransportMode = "pull_v2"

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

type ErrorResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}
