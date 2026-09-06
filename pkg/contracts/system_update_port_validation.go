package contracts

import (
	"encoding/json"
	"strings"
)

// SystemUpdatePortCreateRequestV2 keeps presence for mode-specific inputs. It
// cannot represent legacy new_port, software strategy or arbitrary authority.
type SystemUpdatePortCreateRequestV2 struct {
	ProtocolVersion          int                   `json:"protocol_version"`
	Operation                SystemUpdateOperation `json:"operation"`
	PortContractVersion      int                   `json:"port_contract_version"`
	Mode                     SystemUpdatePortMode  `json:"mode"`
	TargetID                 string                `json:"target_id"`
	ExpectedSnapshotID       string                `json:"expected_snapshot_id"`
	ExpectedEndpointRevision int64                 `json:"expected_endpoint_revision"`
	DesiredRevision          int64                 `json:"desired_revision"`
	Fence                    int64                 `json:"fence"`
	RequiredCapability       UpdaterCapability     `json:"required_capability"`
	IdempotencyKey           string                `json:"idempotency_key"`
	NewLocalListenPort       *int                  `json:"new_local_listen_port,omitempty"`
	NewAdvertisedPort        *int                  `json:"new_advertised_port,omitempty"`
	NewPublishedPort         *int                  `json:"new_published_port,omitempty"`
	NewContainerPort         *int                  `json:"new_container_port,omitempty"`
}

func ValidateSystemUpdatePortCreateRequest(payload []byte) error {
	var request SystemUpdatePortCreateRequestV2
	document, err := decodeUpdaterStrictJSON(payload, &request)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !requireUpdaterFields(object, "protocol_version", "operation", "port_contract_version", "mode", "target_id", "expected_snapshot_id", "expected_endpoint_revision", "desired_revision", "fence", "required_capability", "idempotency_key") {
		return errSystemUpdatePortContract
	}
	return ValidateSystemUpdatePortCreate(request)
}

func ValidateSystemUpdatePortCreate(request SystemUpdatePortCreateRequestV2) error {
	if request.ProtocolVersion != 2 || request.PortContractVersion != 2 || request.Operation != SystemUpdateOperationPortReconfigure ||
		!validUpdaterIdentifier(request.TargetID) || !strings.HasPrefix(request.ExpectedSnapshotID, "ps1:") ||
		!updaterRawSHA256Pattern.MatchString(strings.TrimPrefix(request.ExpectedSnapshotID, "ps1:")) ||
		!validSystemUpdatePortRevision(request.ExpectedEndpointRevision) || !validSystemUpdatePortRevision(request.DesiredRevision) ||
		!validSystemUpdatePortRevision(request.Fence) || request.RequiredCapability != UpdaterCapabilityPort || !validUpdaterIdempotencyKey(request.IdempotencyKey) {
		return errSystemUpdatePortContract
	}
	if request.NewLocalListenPort != nil {
		if !validSystemUpdatePort(*request.NewLocalListenPort, 1024) || request.NewPublishedPort != nil || request.NewContainerPort != nil {
			return errSystemUpdatePortContract
		}
	} else if request.NewPublishedPort == nil || request.NewContainerPort == nil || !validSystemUpdatePort(*request.NewPublishedPort, 1024) || !validSystemUpdatePort(*request.NewContainerPort, 1024) {
		return errSystemUpdatePortContract
	}
	switch request.Mode {
	case SystemUpdatePortModeLocalOnly:
		if request.NewAdvertisedPort != nil {
			return errSystemUpdatePortContract
		}
	case SystemUpdatePortModeLocalAndAdvertised:
		if request.NewAdvertisedPort == nil || !validSystemUpdatePort(*request.NewAdvertisedPort, 1) {
			return errSystemUpdatePortContract
		}
	default:
		return errSystemUpdatePortContract
	}
	return nil
}

func ValidateSystemUpdatePortPlanJSON(payload []byte) error {
	var plan SystemUpdatePortReconfiguration
	document, err := decodeUpdaterStrictJSON(payload, &plan)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !validateSystemUpdatePortPlanDocument(object) {
		return errSystemUpdatePortContract
	}
	return ValidateSystemUpdatePortPlan(plan)
}

func validateSystemUpdatePortPlanDocument(plan map[string]any) bool {
	if !requireUpdaterFields(plan, "port_contract_version", "mode", "network_namespace", "protocol", "before", "target", "rollback", "port_plan_sha256") {
		return false
	}
	allowed := map[string]bool{"port_contract_version": true, "mode": true, "network_namespace": true, "protocol": true, "before": true, "target": true, "rollback": true, "port_plan_sha256": true, "docker_baseline": true}
	for key := range plan {
		if !allowed[key] {
			return false
		}
	}
	for _, name := range []string{"before", "target", "rollback"} {
		ref, ok := plan[name].(map[string]any)
		if !ok || !requireUpdaterFields(ref, "snapshot_id", "snapshot_sha256", "source_policy_revision", "projection_revision", "executor_policy_revision", "executor_policy_sha256", "endpoint_revision", "applied_endpoint_revision", "config_revision", "config_sha256", "advertised_port", "advertised_endpoint_sha256", "local_listen_port") {
			return false
		}
		if docker, present := ref["docker"]; present {
			object, ok := docker.(map[string]any)
			if !ok || !requireUpdaterFields(object, "published_host_ip", "published_port", "container_port", "health_port", "compose_policy_sha256", "compose_revision", "version_env_sha256", "image_id", "repository_digest") {
				return false
			}
		}
	}
	if baseline, present := plan["docker_baseline"]; present {
		object, ok := baseline.(map[string]any)
		if !ok || !requireUpdaterFields(object, "expected_container_id", "expected_image_id", "expected_repository_digest", "expected_version_env_sha256", "approved_compose_config_sha256", "approved_compose_revision") {
			return false
		}
	}
	return true
}

func ValidateSystemUpdatePortResultJSON(plan SystemUpdatePortReconfiguration, payload []byte) error {
	var result SystemUpdatePortResultV2
	document, err := decodeUpdaterStrictJSON(payload, &result)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !validateSystemUpdatePortResultDocument(object) {
		return errSystemUpdatePortContract
	}
	return ValidateSystemUpdatePortResult(plan, result)
}

func validateSystemUpdatePortResultDocument(result map[string]any) bool {
	if !requireUpdaterFields(result, "result", "observation") {
		return false
	}
	observation, ok := result["observation"].(map[string]any)
	if !ok || !requireUpdaterFields(observation, "policy_disk_verified", "policy_memory_verified", "agent_projection_verified", "listener_verified", "observed_at") {
		return false
	}
	if result["result"] == string(SystemUpdatePortReconfigurationRollbackFailed) {
		return len(result) == 2
	}
	if !requireUpdaterFields(result, "observed_snapshot_id", "observed_snapshot_sha256", "observed_config_revision", "observed_config_sha256", "observed_executor_policy_revision", "observed_executor_policy_sha256") {
		return false
	}
	if runtime, present := result["runtime_instance"]; present {
		object, ok := runtime.(map[string]any)
		if !ok || !requireUpdaterFields(object, "container_id", "image_id", "repository_digest") {
			return false
		}
	}
	return true
}

func ValidateUpdaterPortPolicyBaselineJSON(payload []byte) error {
	var baseline UpdaterPortPolicyBaseline
	document, err := decodeUpdaterStrictJSON(payload, &baseline)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !requireUpdaterFields(object, "port_contract_version", "policy_transition_version", "agent_uid", "agent_gid", "source_policy_revision", "projection_revision", "executor_policy_revision", "executor_policy_sha256", "targets", "observed_at") {
		return errSystemUpdatePortContract
	}
	targets, ok := object["targets"].([]any)
	if !ok {
		return errSystemUpdatePortContract
	}
	for _, raw := range targets {
		target, ok := raw.(map[string]any)
		if !ok || !requireUpdaterFields(target, "service_id", "service_type", "deployment_mode", "endpoint_revision", "config_revision", "config_sha256", "local_listen_port") {
			return errSystemUpdatePortContract
		}
	}
	return ValidateUpdaterPortPolicyBaseline(baseline)
}

func validateUpdaterPortResultBinding(lease UpdaterLeaseEnvelope, result UpdaterResultEnvelope) bool {
	plan := lease.Command.DesiredOperation.PortReconfigure
	if lease.Command.DesiredOperation.Operation != UpdaterDesiredPortReconfigure || plan == nil || plan.PortContractVersion != 2 {
		return result.PortReconfigure == nil
	}
	if result.PortReconfigure == nil {
		return result.AppliedRevision == 0 && (result.Outcome == UpdaterOutcomeFailed || result.Outcome == UpdaterOutcomeAmbiguous || result.Status == SystemUpdateCanceled)
	}
	port := *result.PortReconfigure
	if ValidateSystemUpdatePortResult(*plan, port) != nil || port.Observation.ObservedAt.After(lease.LeaseExpiresAt) {
		return false
	}
	switch port.Result {
	case SystemUpdatePortReconfigurationApplied, SystemUpdatePortReconfigurationUnchanged:
		return result.Outcome == UpdaterOutcomeSucceeded && result.Status == SystemUpdateSucceeded && result.AppliedRevision == port.ObservedConfigRevision
	case SystemUpdatePortReconfigurationRolledBack:
		return result.Outcome == UpdaterOutcomeRolledBack && result.Status == SystemUpdateRolledBack && result.AppliedRevision == plan.Rollback.ConfigRevision &&
			hasUpdaterEvidenceAtRevision(result.Evidence, "rollback_verified", result.AppliedRevision) && hasUpdaterEvidenceAtRevision(result.Evidence, "application_probe_verified", result.AppliedRevision)
	case SystemUpdatePortReconfigurationRollbackFailed:
		return result.Outcome == UpdaterOutcomeFailed && result.Status == SystemUpdateFailed && result.AppliedRevision == 0
	default:
		return false
	}
}

// EqualSystemUpdatePortPolicySnapshots compares the complete normalized DTO,
// including independently restored bindings, rather than policy JSON alone.
func EqualSystemUpdatePortPolicySnapshots(left, right SystemUpdatePortPolicySnapshot) bool {
	li, ld, le := ComputeSystemUpdatePortSnapshotIdentity(left)
	ri, rd, re := ComputeSystemUpdatePortSnapshotIdentity(right)
	return le == nil && re == nil && li == ri && ld == rd
}

// MarshalSystemUpdatePortPolicySnapshot validates before persistence; consumers
// may retain the exact returned bytes without a second, divergent JSON shape.
func MarshalSystemUpdatePortPolicySnapshot(snapshot SystemUpdatePortPolicySnapshot) ([]byte, error) {
	normalized, err := normalizeSystemUpdatePortPolicySnapshot(snapshot)
	if err != nil {
		return nil, errSystemUpdatePortContract
	}
	return json.Marshal(normalized)
}
