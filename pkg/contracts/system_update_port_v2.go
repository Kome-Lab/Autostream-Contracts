package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	SystemUpdatePortContractVersion         = 2
	SystemUpdatePortPolicyTransitionVersion = 1
	SystemUpdatePortPolicySnapshotMaxBytes  = 1 << 20
)

type SystemUpdatePortMode string

const (
	SystemUpdatePortModeLocalOnly          SystemUpdatePortMode = "local_only"
	SystemUpdatePortModeLocalAndAdvertised SystemUpdatePortMode = "local_and_advertised"
)

// Snapshot references contain verified applied state. EndpointRevision is the
// desired endpoint counter; AppliedEndpointRevision is the root policy counter.
type SystemUpdatePortSnapshotRef struct {
	SnapshotID               string                          `json:"snapshot_id"`
	SnapshotSHA256           string                          `json:"snapshot_sha256"`
	SourcePolicyRevision     int64                           `json:"source_policy_revision"`
	ProjectionRevision       int64                           `json:"projection_revision"`
	ExecutorPolicyRevision   int64                           `json:"executor_policy_revision"`
	ExecutorPolicySHA256     string                          `json:"executor_policy_sha256"`
	EndpointRevision         int64                           `json:"endpoint_revision"`
	AppliedEndpointRevision  int64                           `json:"applied_endpoint_revision"`
	ConfigRevision           int64                           `json:"config_revision"`
	ConfigSHA256             string                          `json:"config_sha256"`
	AdvertisedPort           int                             `json:"advertised_port"`
	AdvertisedEndpointSHA256 string                          `json:"advertised_endpoint_sha256"`
	LocalListenPort          int                             `json:"local_listen_port"`
	Docker                   *SystemUpdatePortDockerSnapshot `json:"docker,omitempty"`
}

type SystemUpdatePortDockerSnapshot struct {
	PublishedHostIP     string `json:"published_host_ip"`
	PublishedPort       int    `json:"published_port"`
	ContainerPort       int    `json:"container_port"`
	HealthPort          int    `json:"health_port"`
	ComposePolicySHA256 string `json:"compose_policy_sha256"`
	ComposeRevision     int64  `json:"compose_revision"`
	VersionEnvSHA256    string `json:"version_env_sha256"`
	ImageID             string `json:"image_id"`
	RepositoryDigest    string `json:"repository_digest"`
}

type SystemUpdatePortDockerBaseline struct {
	ExpectedContainerID         string `json:"expected_container_id"`
	ExpectedImageID             string `json:"expected_image_id"`
	ExpectedRepositoryDigest    string `json:"expected_repository_digest"`
	ExpectedVersionEnvSHA256    string `json:"expected_version_env_sha256"`
	ApprovedComposeConfigSHA256 string `json:"approved_compose_config_sha256"`
	ApprovedComposeRevision     int64  `json:"approved_compose_revision"`
}

type SystemUpdatePortObservation struct {
	PolicyDiskVerified      bool      `json:"policy_disk_verified"`
	PolicyMemoryVerified    bool      `json:"policy_memory_verified"`
	AgentProjectionVerified bool      `json:"agent_projection_verified"`
	ListenerVerified        bool      `json:"listener_verified"`
	ObservedAt              time.Time `json:"observed_at"`
}

type SystemUpdatePortRuntimeInstance struct {
	ContainerID      string `json:"container_id"`
	ImageID          string `json:"image_id"`
	RepositoryDigest string `json:"repository_digest"`
}

// SystemUpdatePortResultV2 is separate from the immutable plan. RollbackFailed
// is an attempt observation, never an accepted result or a released host hold.
type SystemUpdatePortResultV2 struct {
	Result                         SystemUpdatePortReconfigurationResult `json:"result"`
	ObservedSnapshotID             string                                `json:"observed_snapshot_id,omitempty"`
	ObservedSnapshotSHA256         string                                `json:"observed_snapshot_sha256,omitempty"`
	ObservedConfigRevision         int64                                 `json:"observed_config_revision,omitempty"`
	ObservedConfigSHA256           string                                `json:"observed_config_sha256,omitempty"`
	ObservedExecutorPolicyRevision int64                                 `json:"observed_executor_policy_revision,omitempty"`
	ObservedExecutorPolicySHA256   string                                `json:"observed_executor_policy_sha256,omitempty"`
	Observation                    SystemUpdatePortObservation           `json:"observation"`
	RuntimeInstance                *SystemUpdatePortRuntimeInstance      `json:"runtime_instance,omitempty"`
}

type UpdaterPortPolicyBaseline struct {
	PortContractVersion     int                               `json:"port_contract_version"`
	PolicyTransitionVersion int                               `json:"policy_transition_version"`
	AgentUID                uint32                            `json:"agent_uid"`
	AgentGID                uint32                            `json:"agent_gid"`
	SourcePolicyRevision    int64                             `json:"source_policy_revision"`
	ProjectionRevision      int64                             `json:"projection_revision"`
	ExecutorPolicyRevision  int64                             `json:"executor_policy_revision"`
	ExecutorPolicySHA256    string                            `json:"executor_policy_sha256"`
	Targets                 []UpdaterPortPolicyBaselineTarget `json:"targets"`
	ObservedAt              time.Time                         `json:"observed_at"`
}

type UpdaterPortPolicyBaselineTarget struct {
	ServiceID        string                          `json:"service_id"`
	ServiceType      SystemUpdateTargetType          `json:"service_type"`
	DeploymentMode   SystemUpdateDeploymentMode      `json:"deployment_mode"`
	EndpointRevision int64                           `json:"endpoint_revision"`
	ConfigRevision   int64                           `json:"config_revision"`
	ConfigSHA256     string                          `json:"config_sha256"`
	LocalListenPort  int                             `json:"local_listen_port"`
	Docker           *SystemUpdatePortDockerSnapshot `json:"docker,omitempty"`
	DockerRoot       *UpdaterPortDockerRootBaseline  `json:"docker_root,omitempty"`
}

// UpdaterPortDockerRootBaseline carries only existing non-secret profile
// metadata required to reproduce the already-bound root policy bytes.
type UpdaterPortDockerRootBaseline struct {
	ComposeConfigSHA256 string `json:"compose_config_sha256"`
	CurrentVersion      string `json:"current_version,omitempty"`
}

// SystemUpdatePortPolicySnapshot is an internal, secret-free persistence DTO,
// never a create-command or arbitrary root-policy input. Policy contains the
// complete source policy without updated_at. Bindings are kept independently
// so json:"-" source fields cannot disappear from the comparison authority.
type SystemUpdatePortPolicySnapshot struct {
	Policy               json.RawMessage                     `json:"policy"`
	UpdaterID            string                              `json:"updater_id"`
	HostID               string                              `json:"host_id"`
	OwnershipEpoch       int64                               `json:"ownership_epoch"`
	CredentialReferences []string                            `json:"credential_references"`
	Bindings             []SystemUpdatePortPolicyBinding     `json:"bindings"`
	Targets              []SystemUpdatePortPolicyTargetState `json:"targets"`
	RootPolicy           json.RawMessage                     `json:"root_policy"`
}

type SystemUpdatePortPolicyBinding struct {
	TargetID              string  `json:"target_id"`
	ServiceID             string  `json:"service_id"`
	HostID                string  `json:"host_id"`
	BindingPolicyRevision int64   `json:"binding_policy_revision"`
	DatabaseName          *string `json:"database_name,omitempty"`
	LocalListenPort       *int    `json:"local_listen_port,omitempty"`
}

type SystemUpdatePortEndpoint struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSLEnabled bool   `json:"ssl_enabled"`
	PublicURL  string `json:"public_url"`
}

type SystemUpdatePortPolicyTargetState struct {
	TargetID                string                          `json:"target_id"`
	ServiceID               string                          `json:"service_id"`
	HostID                  string                          `json:"host_id"`
	ServiceType             SystemUpdateTargetType          `json:"service_type"`
	DeploymentMode          SystemUpdateDeploymentMode      `json:"deployment_mode"`
	DesiredEndpoint         SystemUpdatePortEndpoint        `json:"desired_endpoint"`
	AppliedEndpoint         SystemUpdatePortEndpoint        `json:"applied_endpoint"`
	EndpointRevision        int64                           `json:"endpoint_revision"`
	AppliedEndpointRevision int64                           `json:"applied_endpoint_revision"`
	ConfigRevision          int64                           `json:"config_revision"`
	ConfigSHA256            string                          `json:"config_sha256"`
	LocalListenPort         int                             `json:"local_listen_port"`
	Docker                  *SystemUpdatePortDockerSnapshot `json:"docker,omitempty"`
}

var errSystemUpdatePortContract = errors.New("system update port contract invalid")

var systemUpdatePortBaselineVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)

func validSystemUpdatePortRevision(value int64) bool {
	return value > 0 && value <= int64(updaterMaxJCSSafeInteger)
}

func validSystemUpdatePort(value int, minimum int) bool { return value >= minimum && value <= 65535 }

func validSystemUpdatePortSnapshotRef(ref SystemUpdatePortSnapshotRef) bool {
	if !updaterDigestPattern.MatchString(ref.SnapshotSHA256) ||
		ref.SnapshotID != "ps1:"+strings.TrimPrefix(ref.SnapshotSHA256, "sha256:") ||
		!validSystemUpdatePortRevision(ref.SourcePolicyRevision) || !validSystemUpdatePortRevision(ref.ProjectionRevision) ||
		!validSystemUpdatePortRevision(ref.ExecutorPolicyRevision) || !validSystemUpdatePortRevision(ref.EndpointRevision) ||
		!validSystemUpdatePortRevision(ref.AppliedEndpointRevision) || ref.AppliedEndpointRevision > ref.EndpointRevision ||
		!validSystemUpdatePortRevision(ref.ConfigRevision) || !updaterDigestPattern.MatchString(ref.ExecutorPolicySHA256) ||
		!updaterDigestPattern.MatchString(ref.ConfigSHA256) || !updaterDigestPattern.MatchString(ref.AdvertisedEndpointSHA256) ||
		!validSystemUpdatePort(ref.AdvertisedPort, 1) || !validSystemUpdatePort(ref.LocalListenPort, 1024) {
		return false
	}
	return ref.Docker == nil || validSystemUpdatePortDockerSnapshot(*ref.Docker, ref.LocalListenPort)
}

func validSystemUpdatePortDockerSnapshot(docker SystemUpdatePortDockerSnapshot, localPort int) bool {
	return docker.PublishedHostIP == "127.0.0.1" && docker.PublishedPort == localPort &&
		validSystemUpdatePort(docker.PublishedPort, 1024) && validSystemUpdatePort(docker.ContainerPort, 1024) &&
		docker.HealthPort == docker.PublishedPort && validSystemUpdatePortRevision(docker.ComposeRevision) &&
		updaterDigestPattern.MatchString(docker.ComposePolicySHA256) && updaterDigestPattern.MatchString(docker.VersionEnvSHA256) &&
		updaterDigestPattern.MatchString(docker.ImageID) && updaterDigestPattern.MatchString(docker.RepositoryDigest)
}

func systemUpdatePortLocalEqual(left, right SystemUpdatePortSnapshotRef) bool {
	if left.LocalListenPort != right.LocalListenPort || (left.Docker == nil) != (right.Docker == nil) {
		return false
	}
	return left.Docker == nil || left.Docker.PublishedPort == right.Docker.PublishedPort && left.Docker.ContainerPort == right.Docker.ContainerPort
}

// SystemUpdatePortPlanIsNoOp identifies the exact B=T=R branch. It does not
// replace the fresh disk/memory/Agent/listener observations required to finish.
func SystemUpdatePortPlanIsNoOp(plan SystemUpdatePortReconfiguration) bool {
	return plan.PortContractVersion == 2 && plan.Before != nil && plan.Target != nil && plan.Rollback != nil &&
		reflect.DeepEqual(*plan.Before, *plan.Target) && reflect.DeepEqual(*plan.Before, *plan.Rollback)
}

// ValidateSystemUpdatePortPlan closes the new version against mixed legacy
// fields and checks independent B/T/R counters. Legacy saved plans retain their
// original decoder and are not silently promoted to this contract.
func ValidateSystemUpdatePortPlan(plan SystemUpdatePortReconfiguration) error {
	if plan.PortContractVersion != 2 || (plan.Mode != SystemUpdatePortModeLocalOnly && plan.Mode != SystemUpdatePortModeLocalAndAdvertised) ||
		plan.NetworkNamespace != "host" || plan.Protocol != SystemUpdatePortProtocolTCP ||
		plan.Before == nil || plan.Target == nil || plan.Rollback == nil || plan.Result != "" || plan.Docker != nil ||
		plan.OldPort != 0 || plan.NewPort != 0 || plan.ExpectedEndpointRevision != 0 || plan.TargetEndpointRevision != 0 ||
		plan.ExpectedConfigRevision != 0 || plan.TargetConfigRevision != 0 || plan.ExpectedConfigSHA256 != "" || plan.TargetConfigSHA256 != "" ||
		plan.ExpectedSourcePolicyRevision != 0 || plan.ExpectedUpdaterPolicyRevision != 0 || plan.ExpectedExecutorPolicyRevision != 0 ||
		plan.ExpectedExecutorPolicySHA256 != "" || !updaterRawSHA256Pattern.MatchString(plan.PortPlanSHA256) {
		return errSystemUpdatePortContract
	}
	b, t, r := *plan.Before, *plan.Target, *plan.Rollback
	if !validSystemUpdatePortSnapshotRef(b) || !validSystemUpdatePortSnapshotRef(t) || !validSystemUpdatePortSnapshotRef(r) ||
		(b.Docker == nil) != (t.Docker == nil) || (b.Docker == nil) != (r.Docker == nil) ||
		(b.Docker == nil) != (plan.DockerBaseline == nil) {
		return errSystemUpdatePortContract
	}
	if b.Docker != nil {
		baseline := plan.DockerBaseline
		if !validSystemUpdatePortContainerID(baseline.ExpectedContainerID) || baseline.ExpectedImageID != b.Docker.ImageID ||
			baseline.ExpectedRepositoryDigest != b.Docker.RepositoryDigest || baseline.ExpectedVersionEnvSHA256 != b.Docker.VersionEnvSHA256 ||
			!updaterRawSHA256Pattern.MatchString(baseline.ApprovedComposeConfigSHA256) || baseline.ApprovedComposeRevision != b.Docker.ComposeRevision {
			return errSystemUpdatePortContract
		}
		for _, next := range []*SystemUpdatePortDockerSnapshot{t.Docker, r.Docker} {
			if next.PublishedHostIP != b.Docker.PublishedHostIP || next.ImageID != b.Docker.ImageID ||
				next.RepositoryDigest != b.Docker.RepositoryDigest || next.VersionEnvSHA256 != b.Docker.VersionEnvSHA256 ||
				next.ComposePolicySHA256 != b.Docker.ComposePolicySHA256 {
				return errSystemUpdatePortContract
			}
		}
	}
	if !SystemUpdatePortPlanIsNoOp(plan) {
		if systemUpdatePortLocalEqual(b, t) || !systemUpdatePortLocalEqual(b, r) || r.AdvertisedPort != b.AdvertisedPort ||
			r.AdvertisedEndpointSHA256 != b.AdvertisedEndpointSHA256 {
			return errSystemUpdatePortContract
		}
		advertisedDelta := int64(0)
		if t.AdvertisedPort != b.AdvertisedPort {
			advertisedDelta = 1
		}
		if plan.Mode == SystemUpdatePortModeLocalOnly && advertisedDelta != 0 ||
			advertisedDelta == 0 && t.AdvertisedEndpointSHA256 != b.AdvertisedEndpointSHA256 ||
			advertisedDelta != 0 && t.AdvertisedEndpointSHA256 == b.AdvertisedEndpointSHA256 {
			return errSystemUpdatePortContract
		}
		for index, next := range []SystemUpdatePortSnapshotRef{t, r} {
			step := int64(index + 1)
			appliedEndpoint := b.AppliedEndpointRevision
			if advertisedDelta != 0 {
				appliedEndpoint = b.EndpointRevision + step
			}
			if next.SourcePolicyRevision != b.SourcePolicyRevision+step || next.ProjectionRevision != b.ProjectionRevision+step ||
				next.ExecutorPolicyRevision != b.ExecutorPolicyRevision+step || next.ConfigRevision != b.ConfigRevision+step ||
				next.EndpointRevision != b.EndpointRevision+step*advertisedDelta || next.AppliedEndpointRevision != appliedEndpoint ||
				next.ExecutorPolicySHA256 == b.ExecutorPolicySHA256 || next.ConfigSHA256 == b.ConfigSHA256 ||
				next.SnapshotID == b.SnapshotID || b.Docker != nil && next.Docker.ComposeRevision != b.Docker.ComposeRevision+step {
				return errSystemUpdatePortContract
			}
		}
		if t.ConfigSHA256 == r.ConfigSHA256 || t.ExecutorPolicySHA256 == r.ExecutorPolicySHA256 || t.SnapshotID == r.SnapshotID {
			return errSystemUpdatePortContract
		}
	}
	digest, err := ComputeSystemUpdatePortPlanSHA256(plan)
	if err != nil || digest != plan.PortPlanSHA256 {
		return errSystemUpdatePortContract
	}
	return nil
}

func validSystemUpdatePortContainerID(id string) bool {
	if len(id) < 12 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

func ComputeSystemUpdatePortPlanSHA256(plan SystemUpdatePortReconfiguration) (string, error) {
	plan.PortPlanSHA256 = ""
	value, err := updaterCanonicalDocument(plan)
	if err != nil {
		return "", errSystemUpdatePortContract
	}
	var body bytes.Buffer
	if appendUpdaterJCS(&body, value) != nil {
		return "", errSystemUpdatePortContract
	}
	return strings.TrimPrefix(ComputeSystemUpdatePortBytesSHA256(body.Bytes()), "sha256:"), nil
}

// ValidateSystemUpdatePortResult verifies the observation against immutable
// intent. Failed recovery deliberately has no successful snapshot identity.
func ValidateSystemUpdatePortResult(plan SystemUpdatePortReconfiguration, result SystemUpdatePortResultV2) error {
	if ValidateSystemUpdatePortPlan(plan) != nil || result.Observation.ObservedAt.IsZero() {
		return errSystemUpdatePortContract
	}
	if result.Result == SystemUpdatePortReconfigurationRollbackFailed {
		if result.ObservedSnapshotID != "" || result.ObservedSnapshotSHA256 != "" || result.ObservedConfigRevision != 0 ||
			result.ObservedConfigSHA256 != "" || result.ObservedExecutorPolicyRevision != 0 || result.ObservedExecutorPolicySHA256 != "" ||
			result.RuntimeInstance != nil || SystemUpdatePortPlanIsNoOp(plan) || systemUpdatePortObservationVerified(result.Observation) {
			return errSystemUpdatePortContract
		}
		return nil
	}
	var observed *SystemUpdatePortSnapshotRef
	switch result.Result {
	case SystemUpdatePortReconfigurationApplied:
		if SystemUpdatePortPlanIsNoOp(plan) {
			return errSystemUpdatePortContract
		}
		observed = plan.Target
	case SystemUpdatePortReconfigurationUnchanged:
		if !SystemUpdatePortPlanIsNoOp(plan) {
			return errSystemUpdatePortContract
		}
		observed = plan.Before
	case SystemUpdatePortReconfigurationRolledBack:
		if SystemUpdatePortPlanIsNoOp(plan) {
			return errSystemUpdatePortContract
		}
		observed = plan.Rollback
	default:
		return errSystemUpdatePortContract
	}
	if !systemUpdatePortObservationVerified(result.Observation) || result.ObservedSnapshotID != observed.SnapshotID ||
		result.ObservedSnapshotSHA256 != observed.SnapshotSHA256 || result.ObservedConfigRevision != observed.ConfigRevision ||
		result.ObservedConfigSHA256 != observed.ConfigSHA256 || result.ObservedExecutorPolicyRevision != observed.ExecutorPolicyRevision ||
		result.ObservedExecutorPolicySHA256 != observed.ExecutorPolicySHA256 {
		return errSystemUpdatePortContract
	}
	if observed.Docker == nil {
		if result.RuntimeInstance != nil {
			return errSystemUpdatePortContract
		}
	} else if result.RuntimeInstance == nil || !validSystemUpdatePortContainerID(result.RuntimeInstance.ContainerID) ||
		result.RuntimeInstance.ImageID != observed.Docker.ImageID || result.RuntimeInstance.RepositoryDigest != observed.Docker.RepositoryDigest ||
		result.Result == SystemUpdatePortReconfigurationUnchanged && result.RuntimeInstance.ContainerID != plan.DockerBaseline.ExpectedContainerID {
		return errSystemUpdatePortContract
	}
	return nil
}

func systemUpdatePortObservationVerified(observation SystemUpdatePortObservation) bool {
	return observation.PolicyDiskVerified && observation.PolicyMemoryVerified && observation.AgentProjectionVerified && observation.ListenerVerified
}

func IsAcceptedSystemUpdatePortResult(result SystemUpdatePortResultV2) bool {
	return result.Result == SystemUpdatePortReconfigurationApplied || result.Result == SystemUpdatePortReconfigurationUnchanged || result.Result == SystemUpdatePortReconfigurationRolledBack
}

// EqualSystemUpdatePortResults includes the first accepted observation time;
// transport renewal must not silently rewrite any accepted result field.
func EqualSystemUpdatePortResults(left, right SystemUpdatePortResultV2) bool {
	l, le := json.Marshal(left)
	r, re := json.Marshal(right)
	return le == nil && re == nil && bytes.Equal(l, r)
}

func ValidateUpdaterPortPolicyBaseline(baseline UpdaterPortPolicyBaseline) error {
	if baseline.PortContractVersion != 2 || baseline.PolicyTransitionVersion != 1 || baseline.AgentUID == 0 || baseline.AgentGID == 0 ||
		!validSystemUpdatePortRevision(baseline.SourcePolicyRevision) || !validSystemUpdatePortRevision(baseline.ProjectionRevision) ||
		!validSystemUpdatePortRevision(baseline.ExecutorPolicyRevision) || !updaterDigestPattern.MatchString(baseline.ExecutorPolicySHA256) ||
		baseline.ObservedAt.IsZero() || len(baseline.Targets) == 0 || len(baseline.Targets) > 1024 {
		return errSystemUpdatePortContract
	}
	seen := map[string]bool{}
	for _, target := range baseline.Targets {
		if !validUpdaterIdentifier(target.ServiceID) || seen[target.ServiceID] || !validUpdaterApplicationServiceType(target.ServiceType) ||
			!validUpdaterDeploymentMode(target.DeploymentMode) || !validSystemUpdatePortRevision(target.EndpointRevision) ||
			!validSystemUpdatePortRevision(target.ConfigRevision) || !updaterDigestPattern.MatchString(target.ConfigSHA256) ||
			!validSystemUpdatePort(target.LocalListenPort, 1024) ||
			(target.DeploymentMode == SystemUpdateDeploymentDocker) != (target.Docker != nil) ||
			(target.DeploymentMode == SystemUpdateDeploymentDocker) != (target.DockerRoot != nil) ||
			target.DockerRoot != nil && (!updaterRawSHA256Pattern.MatchString(target.DockerRoot.ComposeConfigSHA256) ||
				target.DockerRoot.CurrentVersion != "" && !validSystemUpdatePortBaselineVersion(target.DockerRoot.CurrentVersion)) ||
			target.Docker != nil && !validSystemUpdatePortDockerSnapshot(*target.Docker, target.LocalListenPort) {
			return errSystemUpdatePortContract
		}
		seen[target.ServiceID] = true
	}
	return nil
}

func validSystemUpdatePortBaselineVersion(version string) bool {
	return len(version) <= 128 && systemUpdatePortBaselineVersionPattern.MatchString(version)
}

// SystemUpdatePortListenerConfig returns the existing canonical Node listener
// bytes. For Docker this is the derived container listener projection; the root
// ConfigSHA256 binds SystemUpdatePortDockerConfig's mapping bytes instead.
// Revision is included in the bytes, including newly generated R.
func SystemUpdatePortListenerConfig(serviceType SystemUpdateTargetType, deployment SystemUpdateDeploymentMode, port int, revision int64) ([]byte, error) {
	if serviceType == SystemUpdateTargetControlPanel || !validUpdaterApplicationServiceType(serviceType) ||
		!validSystemUpdatePort(port, 1024) || !validSystemUpdatePortRevision(revision) || !validUpdaterDeploymentMode(deployment) {
		return nil, errSystemUpdatePortContract
	}
	host := "127.0.0.1"
	if deployment == SystemUpdateDeploymentDocker {
		host = "0.0.0.0"
	}
	return MarshalNodeListenerConfig(NodeListenerConfig{SchemaVersion: 2, ServiceType: string(serviceType), BindAddress: net.JoinHostPort(host, fmt.Sprint(port)), ConfigRevision: revision})
}

// SystemUpdatePortDockerConfig preserves the installed Docker port mapping's
// three-line LF encoding. The caller cannot supply environment variable names.
func SystemUpdatePortDockerConfig(serviceType SystemUpdateTargetType, publishedPort, containerPort int, revision int64) ([]byte, error) {
	if !validSystemUpdatePort(publishedPort, 1024) || !validSystemUpdatePort(containerPort, 1024) || !validSystemUpdatePortRevision(revision) {
		return nil, errSystemUpdatePortContract
	}
	var prefix string
	switch serviceType {
	case SystemUpdateTargetWorker:
		prefix = "AUTOSTREAM_WORKER"
	case SystemUpdateTargetEncoderRecorder:
		prefix = "AUTOSTREAM_ENCODER_RECORDER"
	case SystemUpdateTargetDiscordBot:
		prefix = "AUTOSTREAM_DISCORD_BOT"
	case SystemUpdateTargetObservability:
		prefix = "AUTOSTREAM_OBSERVABILITY"
	default:
		return nil, errSystemUpdatePortContract
	}
	return []byte(fmt.Sprintf("%s_PORT=%d\n%s_CONTAINER_PORT=%d\nAUTOSTREAM_CONFIG_REVISION=%d\n", prefix, publishedPort, prefix, containerPort, revision)), nil
}

func SystemUpdateDockerPortConfigSHA256(serviceType SystemUpdateTargetType, publishedPort, containerPort int, revision int64) (string, error) {
	payload, err := SystemUpdatePortDockerConfig(serviceType, publishedPort, containerPort, revision)
	if err != nil {
		return "", err
	}
	return ComputeSystemUpdatePortBytesSHA256(payload), nil
}

// ComputeSystemUpdatePortSnapshotIdentity is order-independent for sets, but
// includes exact root payload bytes through its digest. It excludes timestamps,
// job/lease/session fields by construction of the persistence DTO.
func ComputeSystemUpdatePortSnapshotIdentity(snapshot SystemUpdatePortPolicySnapshot) (string, string, error) {
	var err error
	snapshot, err = normalizeSystemUpdatePortPolicySnapshot(snapshot)
	if err != nil {
		return "", "", errSystemUpdatePortContract
	}
	value, err := updaterCanonicalDocument(snapshot)
	if err != nil {
		return "", "", errSystemUpdatePortContract
	}
	var body bytes.Buffer
	if appendUpdaterJCS(&body, value) != nil || body.Len() > SystemUpdatePortPolicySnapshotMaxBytes {
		return "", "", errSystemUpdatePortContract
	}
	digest := ComputeSystemUpdatePortBytesSHA256(body.Bytes())
	return "ps1:" + strings.TrimPrefix(digest, "sha256:"), digest, nil
}

func normalizeSystemUpdatePortPolicySnapshot(snapshot SystemUpdatePortPolicySnapshot) (SystemUpdatePortPolicySnapshot, error) {
	if ValidateSystemUpdatePortPolicySnapshot(snapshot) != nil {
		return SystemUpdatePortPolicySnapshot{}, errSystemUpdatePortContract
	}
	snapshot.Bindings = append([]SystemUpdatePortPolicyBinding{}, snapshot.Bindings...)
	snapshot.Targets = append([]SystemUpdatePortPolicyTargetState{}, snapshot.Targets...)
	snapshot.CredentialReferences = append([]string{}, snapshot.CredentialReferences...)
	sort.Slice(snapshot.Bindings, func(i, j int) bool { return snapshot.Bindings[i].TargetID < snapshot.Bindings[j].TargetID })
	sort.Slice(snapshot.Targets, func(i, j int) bool { return snapshot.Targets[i].TargetID < snapshot.Targets[j].TargetID })
	sort.Strings(snapshot.CredentialReferences)
	var policy portSnapshotSourcePolicy
	if _, err := decodeUpdaterStrictJSON(snapshot.Policy, &policy); err != nil {
		return SystemUpdatePortPolicySnapshot{}, errSystemUpdatePortContract
	}
	sort.Slice(policy.Hosts, func(i, j int) bool { return policy.Hosts[i].HostID < policy.Hosts[j].HostID })
	sort.Slice(policy.Targets, func(i, j int) bool { return policy.Targets[i].TargetID < policy.Targets[j].TargetID })
	snapshot.Policy, _ = json.Marshal(policy)
	return snapshot, nil
}
