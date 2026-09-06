package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"strings"
)

func ComputeSystemUpdatePortBytesSHA256(payload []byte) string {
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// ComputeSystemUpdatePortEndpointSHA256 binds the complete advertised endpoint,
// including the hostname, TLS choice and PublicURL's non-port components.
func ComputeSystemUpdatePortEndpointSHA256(endpoint SystemUpdatePortEndpoint) (string, error) {
	if !validPortSnapshotEndpoint(endpoint) {
		return "", errSystemUpdatePortContract
	}
	value, err := updaterCanonicalDocument(endpoint)
	if err != nil {
		return "", errSystemUpdatePortContract
	}
	var body bytes.Buffer
	if appendUpdaterJCS(&body, value) != nil {
		return "", errSystemUpdatePortContract
	}
	return ComputeSystemUpdatePortBytesSHA256(body.Bytes()), nil
}

// This narrow mirror preserves the existing root serializer's field order.
// Deployment profiles are immutable opaque bytes here; the owning Executor's
// secure parser must validate its installed profile before and after the delta.
type portRootPolicy struct {
	SchemaVersion        int               `json:"schema_version"`
	ProtocolVersion      int               `json:"protocol_version"`
	HostID               string            `json:"host_id"`
	AgentUID             uint32            `json:"agent_uid"`
	AgentGID             uint32            `json:"agent_gid"`
	SocketPath           string            `json:"socket_path"`
	SourcePolicyRevision int64             `json:"source_policy_revision,omitempty"`
	ProjectionRevision   int64             `json:"projection_revision,omitempty"`
	PolicyRevision       int64             `json:"policy_revision"`
	Mutation             *portRootMutation `json:"mutation,omitempty"`
	Targets              []portRootTarget  `json:"targets"`
}

type portRootMutation struct {
	PanelURL string `json:"panel_url"`
}

type portRootTarget struct {
	ServiceID        string           `json:"service_id"`
	ServiceType      string           `json:"service_type"`
	DeploymentMode   string           `json:"deployment_mode"`
	DatabaseName     string           `json:"database_name,omitempty"`
	EndpointRevision int64            `json:"endpoint_revision,omitempty"`
	ConfigRevision   int64            `json:"config_revision"`
	ConfigSHA256     string           `json:"config_sha256,omitempty"`
	LocalListen      portRootEndpoint `json:"local_listen_endpoint"`
	Systemd          json.RawMessage  `json:"systemd,omitempty"`
	Docker           *portRootDocker  `json:"docker,omitempty"`
}

type portRootEndpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type portRootDocker struct {
	DockerPath              string   `json:"docker_path"`
	ComposeProject          string   `json:"compose_project"`
	ProjectDir              string   `json:"project_dir"`
	ComposeFiles            []string `json:"compose_files"`
	Service                 string   `json:"service"`
	ImageRepo               string   `json:"image_repo"`
	ImageVariable           string   `json:"image_variable"`
	BaseEnvFile             string   `json:"base_env_file,omitempty"`
	VersionEnvFile          string   `json:"version_env_file"`
	PortEnvFile             string   `json:"port_env_file,omitempty"`
	ComposeConfigSHA256     string   `json:"compose_config_sha256"`
	PortComposePolicySHA256 string   `json:"port_compose_policy_sha256,omitempty"`
	PortComposeRevision     int64    `json:"port_compose_revision,omitempty"`
	CurrentVersion          string   `json:"current_version,omitempty"`
	Channel                 string   `json:"channel,omitempty"`
}

func decodeSystemUpdatePortRootPolicy(payload []byte) (portRootPolicy, error) {
	var policy portRootPolicy
	if len(payload) == 0 || len(payload) > SystemUpdatePortPolicySnapshotMaxBytes {
		return policy, errSystemUpdatePortContract
	}
	if _, err := decodeUpdaterStrictJSON(payload, &policy); err != nil {
		return policy, errSystemUpdatePortContract
	}
	canonical, err := json.Marshal(policy)
	if err != nil || !bytes.Equal(canonical, payload) || policy.SchemaVersion != 2 || policy.ProtocolVersion != 2 ||
		!validUpdaterIdentifier(policy.HostID) || policy.AgentUID == 0 || policy.AgentGID == 0 ||
		policy.SocketPath != "/run/autostream-local-executor/executor.sock" || policy.Mutation == nil ||
		!validSystemUpdatePortRevision(policy.SourcePolicyRevision) || !validSystemUpdatePortRevision(policy.ProjectionRevision) ||
		!validSystemUpdatePortRevision(policy.PolicyRevision) || len(policy.Targets) == 0 || len(policy.Targets) > 1024 {
		return policy, errSystemUpdatePortContract
	}
	origin, err := url.Parse(policy.Mutation.PanelURL)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" ||
		policy.Mutation.PanelURL != strings.TrimSpace(policy.Mutation.PanelURL) {
		return policy, errSystemUpdatePortContract
	}
	loopback := strings.EqualFold(origin.Hostname(), "localhost")
	if address := net.ParseIP(origin.Hostname()); address != nil {
		loopback = address.IsLoopback()
	}
	if origin.Scheme != "https" && (origin.Scheme != "http" || !loopback) {
		return policy, errSystemUpdatePortContract
	}
	seen := map[string]bool{}
	for _, target := range policy.Targets {
		if seen[target.ServiceID] || !validUpdaterIdentifier(target.ServiceID) || !validUpdaterApplicationServiceType(SystemUpdateTargetType(target.ServiceType)) ||
			!validSystemUpdatePortRevision(target.EndpointRevision) || !validSystemUpdatePortRevision(target.ConfigRevision) ||
			!updaterDigestPattern.MatchString(target.ConfigSHA256) || target.LocalListen.Host != "127.0.0.1" || !validSystemUpdatePort(target.LocalListen.Port, 1024) {
			return policy, errSystemUpdatePortContract
		}
		if target.DeploymentMode == "systemd" {
			if len(target.Systemd) == 0 || target.Docker != nil {
				return policy, errSystemUpdatePortContract
			}
		} else if target.DeploymentMode == "docker" {
			if len(target.Systemd) != 0 || target.Docker == nil {
				return policy, errSystemUpdatePortContract
			}
		} else {
			return policy, errSystemUpdatePortContract
		}
		seen[target.ServiceID] = true
	}
	return policy, nil
}

func portRootTargetMatchesRef(target portRootTarget, ref SystemUpdatePortSnapshotRef) bool {
	if target.EndpointRevision != ref.AppliedEndpointRevision || target.ConfigRevision != ref.ConfigRevision ||
		target.ConfigSHA256 != ref.ConfigSHA256 || target.LocalListen.Port != ref.LocalListenPort || (target.Docker == nil) != (ref.Docker == nil) {
		return false
	}
	return target.Docker == nil || target.Docker.PortComposeRevision == ref.Docker.ComposeRevision &&
		target.Docker.PortComposePolicySHA256 == strings.TrimPrefix(ref.Docker.ComposePolicySHA256, "sha256:")
}

// ApplySystemUpdatePortPolicyDelta transforms only the installed B policy's
// bounded target delta. It takes no replacement policy, path, origin or profile.
// The output digest must equal the pre-authorized snapshot ref. Callers deriving
// T/R initially may leave after.ExecutorPolicySHA256 empty, then bind returned
// bytes before publishing the immutable plan; execution must provide it.
func ApplySystemUpdatePortPolicyDelta(payload []byte, serviceID string, before, after SystemUpdatePortSnapshotRef) ([]byte, error) {
	policy, err := decodeSystemUpdatePortRootPolicy(payload)
	if err != nil || !validUpdaterIdentifier(serviceID) || ComputeSystemUpdatePortBytesSHA256(payload) != before.ExecutorPolicySHA256 ||
		policy.SourcePolicyRevision != before.SourcePolicyRevision || policy.ProjectionRevision != before.ProjectionRevision || policy.PolicyRevision != before.ExecutorPolicyRevision {
		return nil, errSystemUpdatePortContract
	}
	step := after.SourcePolicyRevision - before.SourcePolicyRevision
	if step < 0 || step > 2 || after.ProjectionRevision-before.ProjectionRevision != step || after.ExecutorPolicyRevision-before.ExecutorPolicyRevision != step ||
		after.ConfigRevision-before.ConfigRevision != step || !validSystemUpdatePortRevision(after.SourcePolicyRevision) ||
		!validSystemUpdatePortRevision(after.ProjectionRevision) || !validSystemUpdatePortRevision(after.ExecutorPolicyRevision) ||
		!validSystemUpdatePortRevision(after.ConfigRevision) || !validSystemUpdatePort(after.LocalListenPort, 1024) ||
		!validSystemUpdatePortRevision(after.AppliedEndpointRevision) || !updaterDigestPattern.MatchString(after.ConfigSHA256) ||
		(before.Docker == nil) != (after.Docker == nil) {
		return nil, errSystemUpdatePortContract
	}
	if after.EndpointRevision != before.EndpointRevision && after.EndpointRevision != before.EndpointRevision+step {
		return nil, errSystemUpdatePortContract
	}
	if after.AppliedEndpointRevision != before.AppliedEndpointRevision && after.AppliedEndpointRevision != after.EndpointRevision {
		return nil, errSystemUpdatePortContract
	}
	if step == 1 && systemUpdatePortLocalEqual(before, after) ||
		step == 2 && (after.AdvertisedPort != before.AdvertisedPort || after.AdvertisedEndpointSHA256 != before.AdvertisedEndpointSHA256) {
		return nil, errSystemUpdatePortContract
	}
	matched := false
	for i := range policy.Targets {
		target := &policy.Targets[i]
		if target.ServiceID != serviceID {
			continue
		}
		if target.ServiceType == "control_panel" || !portRootTargetMatchesRef(*target, before) {
			return nil, errSystemUpdatePortContract
		}
		if step == 0 {
			if !reflect.DeepEqual(before, after) {
				return nil, errSystemUpdatePortContract
			}
			return append([]byte(nil), payload...), nil
		}
		if step == 2 && !systemUpdatePortLocalEqual(before, after) {
			return nil, errSystemUpdatePortContract
		}
		var config []byte
		if after.Docker != nil {
			config, err = SystemUpdatePortDockerConfig(SystemUpdateTargetType(target.ServiceType), after.Docker.PublishedPort, after.Docker.ContainerPort, after.ConfigRevision)
		} else {
			config, err = SystemUpdatePortListenerConfig(SystemUpdateTargetType(target.ServiceType), SystemUpdateDeploymentMode(target.DeploymentMode), after.LocalListenPort, after.ConfigRevision)
		}
		if err != nil || ComputeSystemUpdatePortBytesSHA256(config) != after.ConfigSHA256 {
			return nil, errSystemUpdatePortContract
		}
		target.EndpointRevision = after.AppliedEndpointRevision
		target.ConfigRevision = after.ConfigRevision
		target.ConfigSHA256 = after.ConfigSHA256
		target.LocalListen.Port = after.LocalListenPort
		if target.Docker != nil {
			if !validSystemUpdatePortDockerSnapshot(*after.Docker, after.LocalListenPort) ||
				after.Docker.ComposeRevision != before.Docker.ComposeRevision+step || after.Docker.ComposePolicySHA256 != before.Docker.ComposePolicySHA256 ||
				after.Docker.VersionEnvSHA256 != before.Docker.VersionEnvSHA256 || after.Docker.ImageID != before.Docker.ImageID || after.Docker.RepositoryDigest != before.Docker.RepositoryDigest {
				return nil, errSystemUpdatePortContract
			}
			target.Docker.PortComposeRevision = after.Docker.ComposeRevision
		}
		matched = true
	}
	if !matched {
		return nil, errSystemUpdatePortContract
	}
	policy.SourcePolicyRevision = after.SourcePolicyRevision
	policy.ProjectionRevision = after.ProjectionRevision
	policy.PolicyRevision = after.ExecutorPolicyRevision
	result, err := json.Marshal(policy)
	if err != nil || len(result) > SystemUpdatePortPolicySnapshotMaxBytes ||
		after.ExecutorPolicySHA256 != "" && ComputeSystemUpdatePortBytesSHA256(result) != after.ExecutorPolicySHA256 {
		return nil, errSystemUpdatePortContract
	}
	return result, nil
}

// The source policy mirror deliberately has no timestamp or credential value.
type portSnapshotSourcePolicy struct {
	UpdaterID                   string                     `json:"updater_id"`
	Revision                    int64                      `json:"revision"`
	ProjectionRevision          int64                      `json:"projection_revision,omitempty"`
	LocalExecutorPolicyRevision int64                      `json:"local_executor_policy_revision,omitempty"`
	TransportMode               string                     `json:"transport_mode"`
	ExecutionHostID             string                     `json:"execution_host_id,omitempty"`
	LocalExecutorPolicySHA256   string                     `json:"local_executor_policy_sha256,omitempty"`
	PollIntervalSeconds         int                        `json:"poll_interval_seconds"`
	HeartbeatIntervalSeconds    int                        `json:"heartbeat_interval_seconds"`
	Hosts                       []portSnapshotSourceHost   `json:"hosts,omitempty"`
	Targets                     []portSnapshotSourceTarget `json:"targets"`
}

type portSnapshotSourceHost struct {
	HostID        string `json:"host_id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Arch          string `json:"arch"`
	HostPublicKey string `json:"host_public_key"`
}

type portSnapshotSourceTarget struct {
	TargetID       string `json:"target_id"`
	ServiceID      string `json:"service_id"`
	HostID         string `json:"host_id"`
	ServiceType    string `json:"service_type"`
	DeploymentMode string `json:"deployment_mode"`
}

func ValidateSystemUpdatePortPolicySnapshot(snapshot SystemUpdatePortPolicySnapshot) error {
	if !validUpdaterIdentifier(snapshot.UpdaterID) || !validUpdaterIdentifier(snapshot.HostID) || !validSystemUpdatePortRevision(snapshot.OwnershipEpoch) ||
		len(snapshot.Policy) == 0 || len(snapshot.Policy) > SystemUpdatePortPolicySnapshotMaxBytes || snapshot.Bindings == nil || snapshot.Targets == nil || snapshot.CredentialReferences == nil {
		return errSystemUpdatePortContract
	}
	var policy portSnapshotSourcePolicy
	if _, err := decodeUpdaterStrictJSON(snapshot.Policy, &policy); err != nil {
		return errSystemUpdatePortContract
	}
	root, err := decodeSystemUpdatePortRootPolicy(snapshot.RootPolicy)
	if err != nil || policy.TransportMode != "pull_v2" || policy.UpdaterID != snapshot.UpdaterID || policy.ExecutionHostID != snapshot.HostID || root.HostID != snapshot.HostID ||
		policy.Revision != root.SourcePolicyRevision || policy.ProjectionRevision != root.ProjectionRevision || policy.LocalExecutorPolicyRevision != root.PolicyRevision ||
		policy.LocalExecutorPolicySHA256 != ComputeSystemUpdatePortBytesSHA256(snapshot.RootPolicy) || policy.PollIntervalSeconds < 1 || policy.HeartbeatIntervalSeconds < 1 ||
		len(policy.Targets) == 0 || len(policy.Targets) != len(snapshot.Targets) || len(policy.Targets) != len(snapshot.Bindings) || len(policy.Targets) != len(root.Targets) {
		return errSystemUpdatePortContract
	}
	seenReferences := map[string]bool{}
	for _, reference := range snapshot.CredentialReferences {
		if !validUpdaterIdentifier(reference) || seenReferences[reference] {
			return errSystemUpdatePortContract
		}
		seenReferences[reference] = true
	}
	seenHosts := map[string]bool{}
	for _, host := range policy.Hosts {
		if !validUpdaterIdentifier(host.HostID) || seenHosts[host.HostID] {
			return errSystemUpdatePortContract
		}
		seenHosts[host.HostID] = true
	}
	bindings := map[string]SystemUpdatePortPolicyBinding{}
	for _, binding := range snapshot.Bindings {
		if _, exists := bindings[binding.TargetID]; exists || binding.BindingPolicyRevision != policy.Revision || binding.HostID != snapshot.HostID ||
			!validUpdaterIdentifier(binding.TargetID) || !validUpdaterIdentifier(binding.ServiceID) {
			return errSystemUpdatePortContract
		}
		bindings[binding.TargetID] = binding
	}
	targets := map[string]SystemUpdatePortPolicyTargetState{}
	for _, target := range snapshot.Targets {
		if _, exists := targets[target.TargetID]; exists || target.HostID != snapshot.HostID || !validUpdaterIdentifier(target.TargetID) ||
			!validUpdaterIdentifier(target.ServiceID) || !validUpdaterApplicationServiceType(target.ServiceType) || !validUpdaterDeploymentMode(target.DeploymentMode) ||
			!validSystemUpdatePortRevision(target.EndpointRevision) || !validSystemUpdatePortRevision(target.AppliedEndpointRevision) ||
			target.AppliedEndpointRevision > target.EndpointRevision || !validSystemUpdatePortRevision(target.ConfigRevision) ||
			!updaterDigestPattern.MatchString(target.ConfigSHA256) || !validSystemUpdatePort(target.LocalListenPort, 1024) ||
			!validPortSnapshotEndpoint(target.DesiredEndpoint) || !validPortSnapshotEndpoint(target.AppliedEndpoint) ||
			!reflect.DeepEqual(target.DesiredEndpoint, target.AppliedEndpoint) {
			return errSystemUpdatePortContract
		}
		targets[target.TargetID] = target
	}
	rootTargets := map[string]portRootTarget{}
	for _, target := range root.Targets {
		rootTargets[target.ServiceID] = target
	}
	seen := map[string]bool{}
	for _, source := range policy.Targets {
		target, ok := targets[source.TargetID]
		binding, bound := bindings[source.TargetID]
		rootTarget, rooted := rootTargets[source.ServiceID]
		if !ok || !bound || !rooted || seen[source.TargetID] || source.HostID != snapshot.HostID || source.ServiceID != target.ServiceID ||
			source.ServiceType != string(target.ServiceType) || source.DeploymentMode != string(target.DeploymentMode) || binding.ServiceID != source.ServiceID ||
			rootTarget.ServiceType != source.ServiceType || rootTarget.DeploymentMode != source.DeploymentMode {
			return errSystemUpdatePortContract
		}
		if source.ServiceType == "control_panel" {
			if binding.LocalListenPort != nil && *binding.LocalListenPort != 0 {
				return errSystemUpdatePortContract
			}
		} else if source.DeploymentMode == "docker" {
			// The listener binding table belongs to systemd targets. Docker's
			// published listener is bound by the installed root policy and the
			// complete Docker snapshot, both checked below.
			if binding.LocalListenPort != nil {
				return errSystemUpdatePortContract
			}
		} else if binding.LocalListenPort == nil || *binding.LocalListenPort != target.LocalListenPort {
			return errSystemUpdatePortContract
		}
		if rootTarget.DatabaseName == "" {
			if binding.DatabaseName != nil {
				return errSystemUpdatePortContract
			}
		} else if binding.DatabaseName == nil || *binding.DatabaseName != rootTarget.DatabaseName {
			return errSystemUpdatePortContract
		}
		ref := SystemUpdatePortSnapshotRef{AppliedEndpointRevision: target.AppliedEndpointRevision, ConfigRevision: target.ConfigRevision, ConfigSHA256: target.ConfigSHA256, LocalListenPort: target.LocalListenPort, Docker: target.Docker}
		if !portRootTargetMatchesRef(rootTarget, ref) || (target.DeploymentMode == SystemUpdateDeploymentDocker) != (target.Docker != nil) ||
			target.Docker != nil && !validSystemUpdatePortDockerSnapshot(*target.Docker, target.LocalListenPort) {
			return errSystemUpdatePortContract
		}
		seen[source.TargetID] = true
	}
	body, err := json.Marshal(snapshot)
	if err != nil || len(body) > SystemUpdatePortPolicySnapshotMaxBytes {
		return errSystemUpdatePortContract
	}
	return nil
}

func validPortSnapshotEndpoint(endpoint SystemUpdatePortEndpoint) bool {
	if endpoint.Host == "" || endpoint.Host != strings.TrimSpace(endpoint.Host) || !validSystemUpdatePort(endpoint.Port, 1) {
		return false
	}
	if endpoint.PublicURL == "" {
		return true
	}
	parsed, err := url.Parse(endpoint.PublicURL)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.User == nil
}

// ComputeSystemUpdatePortRuntimePlanSHA256 is distinct from the stable intent
// digest: it also binds the current job, ownership, lease and root session.
func ComputeSystemUpdatePortRuntimePlanSHA256(plan SystemUpdatePortReconfiguration, jobID, hostID, targetID, serviceType string, ownershipEpoch int64, leaseGeneration uint64, sessionID string) (string, error) {
	if ValidateSystemUpdatePortPlan(plan) != nil || !validUpdaterIdentifier(jobID) || !validUpdaterIdentifier(hostID) ||
		!validUpdaterIdentifier(targetID) || !validUpdaterApplicationServiceType(SystemUpdateTargetType(serviceType)) || serviceType == "control_panel" ||
		!validSystemUpdatePortRevision(ownershipEpoch) || leaseGeneration == 0 || leaseGeneration > uint64(updaterMaxJCSSafeInteger) || !updaterNoncePattern.MatchString(sessionID) {
		return "", errSystemUpdatePortContract
	}
	value, err := updaterCanonicalDocument(struct {
		Plan            SystemUpdatePortReconfiguration `json:"plan"`
		JobID           string                          `json:"job_id"`
		HostID          string                          `json:"host_id"`
		TargetID        string                          `json:"target_id"`
		ServiceType     string                          `json:"service_type"`
		OwnershipEpoch  int64                           `json:"ownership_epoch"`
		LeaseGeneration uint64                          `json:"lease_generation"`
		SessionID       string                          `json:"session_id"`
	}{plan, jobID, hostID, targetID, serviceType, ownershipEpoch, leaseGeneration, sessionID})
	if err != nil {
		return "", errSystemUpdatePortContract
	}
	var body bytes.Buffer
	if appendUpdaterJCS(&body, value) != nil {
		return "", errSystemUpdatePortContract
	}
	return strings.TrimPrefix(ComputeSystemUpdatePortBytesSHA256(body.Bytes()), "sha256:"), nil
}
