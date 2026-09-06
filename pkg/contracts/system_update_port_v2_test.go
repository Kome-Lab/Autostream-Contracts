package contracts

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func portV2TestPlan(t *testing.T, mode SystemUpdatePortMode, docker, noOp bool) SystemUpdatePortReconfiguration {
	t.Helper()
	before := SystemUpdatePortSnapshotRef{
		SnapshotID: "ps1:" + strings.Repeat("a", 64), SnapshotSHA256: "sha256:" + strings.Repeat("a", 64),
		SourcePolicyRevision: 11, ProjectionRevision: 17, ExecutorPolicyRevision: 23, ExecutorPolicySHA256: "sha256:" + strings.Repeat("1", 64),
		EndpointRevision: 3, AppliedEndpointRevision: 2, ConfigRevision: 31, ConfigSHA256: "sha256:" + strings.Repeat("2", 64),
		AdvertisedPort: 443, AdvertisedEndpointSHA256: "sha256:" + strings.Repeat("3", 64), LocalListenPort: 18081,
	}
	if docker {
		before.Docker = &SystemUpdatePortDockerSnapshot{PublishedHostIP: "127.0.0.1", PublishedPort: 18081, ContainerPort: 8080, HealthPort: 18081, ComposePolicySHA256: "sha256:" + strings.Repeat("4", 64), ComposeRevision: 7, VersionEnvSHA256: "sha256:" + strings.Repeat("5", 64), ImageID: "sha256:" + strings.Repeat("6", 64), RepositoryDigest: "sha256:" + strings.Repeat("7", 64)}
	}
	clone := func(ref SystemUpdatePortSnapshotRef) *SystemUpdatePortSnapshotRef {
		data, _ := json.Marshal(ref)
		var copy SystemUpdatePortSnapshotRef
		_ = json.Unmarshal(data, &copy)
		return &copy
	}
	plan := SystemUpdatePortReconfiguration{PortContractVersion: 2, Mode: mode, NetworkNamespace: "host", Protocol: SystemUpdatePortProtocolTCP, Before: clone(before), Target: clone(before), Rollback: clone(before)}
	if docker {
		plan.DockerBaseline = &SystemUpdatePortDockerBaseline{ExpectedContainerID: strings.Repeat("8", 64), ExpectedImageID: before.Docker.ImageID, ExpectedRepositoryDigest: before.Docker.RepositoryDigest, ExpectedVersionEnvSHA256: before.Docker.VersionEnvSHA256, ApprovedComposeConfigSHA256: strings.Repeat("9", 64), ApprovedComposeRevision: 7}
	}
	if !noOp {
		plan.Target.SourcePolicyRevision = 12
		plan.Target.ProjectionRevision = 18
		plan.Target.ExecutorPolicyRevision = 24
		plan.Target.ConfigRevision = 32
		plan.Target.LocalListenPort = 18084
		plan.Target.ConfigSHA256 = "sha256:" + strings.Repeat("b", 64)
		plan.Target.ExecutorPolicySHA256 = "sha256:" + strings.Repeat("c", 64)
		plan.Target.SnapshotID = "ps1:" + strings.Repeat("d", 64)
		plan.Target.SnapshotSHA256 = "sha256:" + strings.Repeat("d", 64)
		plan.Rollback.SourcePolicyRevision = 13
		plan.Rollback.ProjectionRevision = 19
		plan.Rollback.ExecutorPolicyRevision = 25
		plan.Rollback.ConfigRevision = 33
		plan.Rollback.ConfigSHA256 = "sha256:" + strings.Repeat("e", 64)
		plan.Rollback.ExecutorPolicySHA256 = "sha256:" + strings.Repeat("f", 64)
		plan.Rollback.SnapshotID = "ps1:" + strings.Repeat("0", 64)
		plan.Rollback.SnapshotSHA256 = "sha256:" + strings.Repeat("0", 64)
		if docker {
			plan.Target.Docker.PublishedPort = 18084
			plan.Target.Docker.HealthPort = 18084
			plan.Target.Docker.ComposeRevision = 8
			plan.Rollback.Docker.ComposeRevision = 9
		}
		if mode == SystemUpdatePortModeLocalAndAdvertised {
			plan.Target.AdvertisedPort = 8443
			plan.Target.AdvertisedEndpointSHA256 = "sha256:" + strings.Repeat("a", 64)
			plan.Target.EndpointRevision = 4
			plan.Target.AppliedEndpointRevision = 4
			plan.Rollback.EndpointRevision = 5
			plan.Rollback.AppliedEndpointRevision = 5
		}
	}
	var err error
	plan.PortPlanSHA256, err = ComputeSystemUpdatePortPlanSHA256(plan)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func portV2TestResult(ref *SystemUpdatePortSnapshotRef, result SystemUpdatePortReconfigurationResult) SystemUpdatePortResultV2 {
	return SystemUpdatePortResultV2{Result: result, ObservedSnapshotID: ref.SnapshotID, ObservedSnapshotSHA256: ref.SnapshotSHA256, ObservedConfigRevision: ref.ConfigRevision, ObservedConfigSHA256: ref.ConfigSHA256, ObservedExecutorPolicyRevision: ref.ExecutorPolicyRevision, ObservedExecutorPolicySHA256: ref.ExecutorPolicySHA256, Observation: SystemUpdatePortObservation{PolicyDiskVerified: true, PolicyMemoryVerified: true, AgentProjectionVerified: true, ListenerVerified: true, ObservedAt: testUpdaterTime().Add(time.Minute)}}
}

func TestSystemUpdatePortV2ModesAndIndependentCounters(t *testing.T) {
	for _, mode := range []SystemUpdatePortMode{SystemUpdatePortModeLocalOnly, SystemUpdatePortModeLocalAndAdvertised} {
		for _, docker := range []bool{false, true} {
			for _, noOp := range []bool{false, true} {
				plan := portV2TestPlan(t, mode, docker, noOp)
				if err := ValidateSystemUpdatePortPlanJSON(testUpdaterJSON(t, plan)); err != nil {
					t.Fatalf("mode=%s docker=%t no_op=%t: %v", mode, docker, noOp, err)
				}
				if SystemUpdatePortPlanIsNoOp(plan) != noOp {
					t.Fatal("no-op branch mismatch")
				}
			}
		}
	}
	for name, mutate := range map[string]func(*SystemUpdatePortReconfiguration){
		"advertised-only":   func(p *SystemUpdatePortReconfiguration) { p.Target.LocalListenPort = p.Before.LocalListenPort },
		"counter-coalesced": func(p *SystemUpdatePortReconfiguration) { p.Target.ProjectionRevision = p.Target.SourcePolicyRevision },
		"rollback-old-config": func(p *SystemUpdatePortReconfiguration) {
			p.Rollback.ConfigRevision = p.Before.ConfigRevision
			p.Rollback.ConfigSHA256 = p.Before.ConfigSHA256
		},
		"applied-endpoint-confused": func(p *SystemUpdatePortReconfiguration) { p.Before.AppliedEndpointRevision = 4 },
		"endpoint-not-monotonic":    func(p *SystemUpdatePortReconfiguration) { p.Rollback.EndpointRevision = p.Before.EndpointRevision },
		"mixed-legacy":              func(p *SystemUpdatePortReconfiguration) { p.NewPort = 18084 },
		"result-in-plan":            func(p *SystemUpdatePortReconfiguration) { p.Result = SystemUpdatePortReconfigurationApplied },
		"bad-snapshot-id":           func(p *SystemUpdatePortReconfiguration) { p.Target.SnapshotID = p.Before.SnapshotID },
		"overflow":                  func(p *SystemUpdatePortReconfiguration) { p.Before.ConfigRevision = 9007199254740991 },
	} {
		t.Run(name, func(t *testing.T) {
			plan := portV2TestPlan(t, SystemUpdatePortModeLocalAndAdvertised, false, false)
			mutate(&plan)
			plan.PortPlanSHA256, _ = ComputeSystemUpdatePortPlanSHA256(plan)
			if ValidateSystemUpdatePortPlan(plan) == nil {
				t.Fatal("invalid delta accepted")
			}
		})
	}
	plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, false, false)
	object := testUpdaterObject(t, testUpdaterJSON(t, plan))
	object["new_port"] = 0
	if ValidateSystemUpdatePortPlanJSON(testUpdaterJSON(t, object)) == nil {
		t.Fatal("explicit zero legacy field accepted")
	}
}

func TestSystemUpdatePortV2CreateStrictModes(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-create-request.schema.json")
	base := `{"protocol_version":2,"operation":"port_reconfigure","port_contract_version":2,"mode":"local_only","target_id":"worker-a","expected_snapshot_id":"ps1:` + strings.Repeat("a", 64) + `","expected_endpoint_revision":3,"desired_revision":32,"fence":7,"required_capability":"host.port","idempotency_key":"port-intent-a","new_local_listen_port":18084}`
	variants := []string{base, strings.Replace(base, `"mode":"local_only"`, `"mode":"local_and_advertised","new_advertised_port":8443`, 1), strings.Replace(base, `"new_local_listen_port":18084`, `"new_published_port":18084,"new_container_port":8080`, 1)}
	for _, body := range variants {
		if ValidateSystemUpdatePortCreateRequest([]byte(body)) != nil {
			t.Fatal("valid create rejected")
		}
		validatePortContractJSON(t, schema, body, true)
	}
	mutants := map[string]string{
		"legacy":             strings.Replace(base, `"new_local_listen_port":18084`, `"new_port":18084`, 1),
		"missing-local":      strings.Replace(base, `,"new_local_listen_port":18084`, "", 1),
		"missing-version":    strings.Replace(base, `"port_contract_version":2,`, "", 1),
		"null":               strings.Replace(base, `"new_local_listen_port":18084`, `"new_local_listen_port":null`, 1),
		"zero":               strings.Replace(base, `"new_local_listen_port":18084`, `"new_local_listen_port":0`, 1),
		"unknown":            strings.Replace(base, `"new_local_listen_port":18084`, `"new_local_listen_port":18084,"root_policy":{}`, 1),
		"mode-field":         strings.Replace(base, `"new_local_listen_port":18084`, `"new_local_listen_port":18084,"new_advertised_port":8443`, 1),
		"missing-advertised": strings.Replace(base, `"mode":"local_only"`, `"mode":"local_and_advertised"`, 1),
		"mixed-deployment":   strings.Replace(base, `"new_local_listen_port":18084`, `"new_local_listen_port":18084,"new_published_port":18084,"new_container_port":8080`, 1),
	}
	for name, body := range mutants {
		t.Run(name, func(t *testing.T) {
			if ValidateSystemUpdatePortCreateRequest([]byte(body)) == nil {
				t.Fatal("invalid create accepted")
			}
			validatePortContractJSON(t, schema, body, false)
		})
	}
	duplicate := strings.Replace(base, `"mode":"local_only"`, `"mode":"local_only","mode":"local_only"`, 1)
	if ValidateSystemUpdatePortCreateRequest([]byte(duplicate)) == nil {
		t.Fatal("duplicate key accepted")
	}
}

func TestSystemUpdatePortV2ResultWirePreservesMeaning(t *testing.T) {
	for _, kind := range []SystemUpdatePortReconfigurationResult{SystemUpdatePortReconfigurationApplied, SystemUpdatePortReconfigurationUnchanged, SystemUpdatePortReconfigurationRolledBack, SystemUpdatePortReconfigurationRollbackFailed} {
		t.Run(string(kind), func(t *testing.T) {
			plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, false, kind == SystemUpdatePortReconfigurationUnchanged)
			command := testUpdaterPortCommand(t)
			command.DesiredOperation.PortReconfigure = &plan
			command.MutationAuthorization.DesiredRevision = plan.Target.ConfigRevision
			command.MutationAuthorization.Target.ExpectedConfigRevision = plan.Before.ConfigRevision
			testUpdaterRefreshDigest(t, &command)
			lease := testUpdaterLease(t, command)
			result := testUpdaterSucceededResult(lease, "application_probe_verified")
			ref := plan.Target
			if kind == SystemUpdatePortReconfigurationRolledBack {
				ref = plan.Rollback
			}
			port := portV2TestResult(ref, kind)
			if kind == SystemUpdatePortReconfigurationRolledBack {
				result.Status = SystemUpdateRolledBack
				result.Outcome = UpdaterOutcomeRolledBack
				result.AppliedRevision = 33
				result.Evidence[0].ObservedRevision = 33
				result.Evidence = append(result.Evidence, UpdaterEvidence{EvidenceCode: "rollback_verified", ObservedAt: port.Observation.ObservedAt, ObservedRevision: 33})
			}
			if kind == SystemUpdatePortReconfigurationRollbackFailed {
				port = SystemUpdatePortResultV2{Result: kind, Observation: SystemUpdatePortObservation{ObservedAt: testUpdaterTime().Add(time.Minute)}}
				result.Status = SystemUpdateFailed
				result.Outcome = UpdaterOutcomeFailed
				result.AppliedRevision = 0
				result.SafeError = &V2UpdaterSafeError{Code: "execution_failed", Message: "updater execution failed", Retryable: false}
			}
			result.PortReconfigure = &port
			encoded := testUpdaterJSON(t, result)
			if err := ValidateUpdaterResultEnvelope(lease, encoded); err != nil {
				t.Fatalf("typed result rejected: %v", err)
			}
			var decoded UpdaterResultEnvelope
			if json.Unmarshal(encoded, &decoded) != nil || !EqualSystemUpdatePortResults(*decoded.PortReconfigure, port) {
				t.Fatal("wire erased typed result")
			}
			if IsAcceptedSystemUpdatePortResult(port) != (kind != SystemUpdatePortReconfigurationRollbackFailed) {
				t.Fatal("failed recovery consumed accepted slot")
			}
			mutant := testUpdaterObject(t, encoded)
			delete(mutant, "port_reconfigure")
			if kind != SystemUpdatePortReconfigurationRollbackFailed && ValidateUpdaterResultEnvelope(lease, testUpdaterJSON(t, mutant)) == nil {
				t.Fatal("missing typed success accepted")
			}
			mutant = testUpdaterObject(t, encoded)
			observation := mutant["port_reconfigure"].(map[string]any)["observation"].(map[string]any)
			delete(observation, "policy_disk_verified")
			if ValidateUpdaterResultEnvelope(lease, testUpdaterJSON(t, mutant)) == nil {
				t.Fatal("missing observation boolean accepted")
			}
			changed := port
			changed.Observation.ObservedAt = changed.Observation.ObservedAt.Add(time.Second)
			if EqualSystemUpdatePortResults(port, changed) {
				t.Fatal("first observation time overwritten")
			}
		})
	}
	plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, false, false)
	if ValidateSystemUpdatePortResult(plan, portV2TestResult(plan.Before, SystemUpdatePortReconfigurationUnchanged)) == nil {
		t.Fatal("failed changed intent converted to unchanged")
	}
}

func TestSystemUpdatePortV2IntentAndRuntimeDigests(t *testing.T) {
	plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, false, false)
	intent := plan.PortPlanSHA256
	runtime, err := ComputeSystemUpdatePortRuntimePlanSHA256(plan, "job-a", "host-a", "worker-a", "worker", 7, 1, "session-0000000001")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := ComputeSystemUpdatePortRuntimePlanSHA256(plan, "job-a", "host-a", "worker-a", "worker", 7, 2, "session-0000000002")
	if err != nil {
		t.Fatal(err)
	}
	if runtime == intent || runtime == changed || plan.PortPlanSHA256 != intent {
		t.Fatal("intent and runtime identity conflated")
	}
	plan.Target.LocalListenPort = 18085
	if _, err := ComputeSystemUpdatePortRuntimePlanSHA256(plan, "job-a", "host-a", "worker-a", "worker", 7, 1, "session-0000000001"); err == nil {
		t.Fatal("modified immutable plan accepted")
	}
}

func portV2TestPolicySnapshot(t *testing.T) SystemUpdatePortPolicySnapshot {
	t.Helper()
	workerConfig, err := SystemUpdatePortListenerConfig(SystemUpdateTargetWorker, SystemUpdateDeploymentSystemd, 18081, 31)
	if err != nil {
		t.Fatal(err)
	}
	observabilityConfig, err := SystemUpdatePortListenerConfig(SystemUpdateTargetObservability, SystemUpdateDeploymentSystemd, 18090, 71)
	if err != nil {
		t.Fatal(err)
	}
	root := portRootPolicy{SchemaVersion: 2, ProtocolVersion: 2, HostID: "host-a", AgentUID: 1001, AgentGID: 1001, SocketPath: "/run/autostream-local-executor/executor.sock", SourcePolicyRevision: 11, ProjectionRevision: 17, PolicyRevision: 23, Mutation: &portRootMutation{PanelURL: "https://panel.example.com"}, Targets: []portRootTarget{
		{ServiceID: "worker-a", ServiceType: "worker", DeploymentMode: "systemd", EndpointRevision: 2, ConfigRevision: 31, ConfigSHA256: ComputeSystemUpdatePortBytesSHA256(workerConfig), LocalListen: portRootEndpoint{Host: "127.0.0.1", Port: 18081}, Systemd: json.RawMessage(`{"systemctl_path":"/usr/bin/systemctl","runuser_path":"/usr/sbin/runuser","smoke_user":"autostream","unit":"autostream-worker.service","release_root":"/opt/autostream/worker/releases","current_link":"/opt/autostream/worker/current","binary_path":"bin/autostream-worker"}`)},
		{ServiceID: "observability-a", ServiceType: "observability", DeploymentMode: "systemd", DatabaseName: "metrics", EndpointRevision: 5, ConfigRevision: 71, ConfigSHA256: ComputeSystemUpdatePortBytesSHA256(observabilityConfig), LocalListen: portRootEndpoint{Host: "127.0.0.1", Port: 18090}, Systemd: json.RawMessage(`{"systemctl_path":"/usr/bin/systemctl","runuser_path":"/usr/sbin/runuser","smoke_user":"autostream","unit":"autostream-observability.service","release_root":"/opt/autostream/observability/releases","current_link":"/opt/autostream/observability/current","binary_path":"bin/autostream-observability"}`)},
	}}
	rootBytes, _ := json.Marshal(root)
	policy := portSnapshotSourcePolicy{UpdaterID: "agent-a", Revision: 11, ProjectionRevision: 17, LocalExecutorPolicyRevision: 23, TransportMode: "pull_v2", ExecutionHostID: "host-a", LocalExecutorPolicySHA256: ComputeSystemUpdatePortBytesSHA256(rootBytes), PollIntervalSeconds: 15, HeartbeatIntervalSeconds: 30, Targets: []portSnapshotSourceTarget{{TargetID: "worker-a", ServiceID: "worker-a", HostID: "host-a", ServiceType: "worker", DeploymentMode: "systemd"}, {TargetID: "observability-a", ServiceID: "observability-a", HostID: "host-a", ServiceType: "observability", DeploymentMode: "systemd"}}}
	policyBytes, _ := json.Marshal(policy)
	workerPort, observabilityPort, database := 18081, 18090, "metrics"
	workerEndpoint := SystemUpdatePortEndpoint{Host: "worker.example.com", Port: 443, SSLEnabled: true, PublicURL: "https://worker.example.com/api?view=ready"}
	observabilityEndpoint := SystemUpdatePortEndpoint{Host: "metrics.example.com", Port: 443, SSLEnabled: true, PublicURL: "https://metrics.example.com"}
	return SystemUpdatePortPolicySnapshot{Policy: policyBytes, UpdaterID: "agent-a", HostID: "host-a", OwnershipEpoch: 7, CredentialReferences: []string{"credential-ref-a"}, Bindings: []SystemUpdatePortPolicyBinding{{TargetID: "worker-a", ServiceID: "worker-a", HostID: "host-a", BindingPolicyRevision: 11, LocalListenPort: &workerPort}, {TargetID: "observability-a", ServiceID: "observability-a", HostID: "host-a", BindingPolicyRevision: 11, LocalListenPort: &observabilityPort, DatabaseName: &database}}, Targets: []SystemUpdatePortPolicyTargetState{{TargetID: "worker-a", ServiceID: "worker-a", HostID: "host-a", ServiceType: SystemUpdateTargetWorker, DeploymentMode: SystemUpdateDeploymentSystemd, DesiredEndpoint: workerEndpoint, AppliedEndpoint: workerEndpoint, EndpointRevision: 3, AppliedEndpointRevision: 2, ConfigRevision: 31, ConfigSHA256: root.Targets[0].ConfigSHA256, LocalListenPort: 18081}, {TargetID: "observability-a", ServiceID: "observability-a", HostID: "host-a", ServiceType: SystemUpdateTargetObservability, DeploymentMode: SystemUpdateDeploymentSystemd, DesiredEndpoint: observabilityEndpoint, AppliedEndpoint: observabilityEndpoint, EndpointRevision: 5, AppliedEndpointRevision: 5, ConfigRevision: 71, ConfigSHA256: root.Targets[1].ConfigSHA256, LocalListenPort: 18090}}, RootPolicy: rootBytes}
}

func TestSystemUpdatePortV2CompleteSnapshot(t *testing.T) {
	before := portV2TestPolicySnapshot(t)
	identity, digest, err := ComputeSystemUpdatePortSnapshotIdentity(before)
	if err != nil || identity == "" || digest == "" {
		t.Fatal("complete snapshot rejected")
	}
	for name, mutate := range map[string]func(*SystemUpdatePortPolicySnapshot){
		"missing-listener":      func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].LocalListenPort = nil },
		"missing-database":      func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[1].DatabaseName = nil },
		"stale-binding":         func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[1].BindingPolicyRevision = 10 },
		"foreign-host":          func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].HostID = "host-b" },
		"orphan":                func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].TargetID = "missing" },
		"duplicate":             func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0] = s.Bindings[1] },
		"lost-applied-revision": func(s *SystemUpdatePortPolicySnapshot) { s.Targets[0].AppliedEndpointRevision = 0 },
		"pending-desired":       func(s *SystemUpdatePortPolicySnapshot) { s.Targets[0].DesiredEndpoint.Port = 8443 },
		"source-timestamp": func(s *SystemUpdatePortPolicySnapshot) {
			s.Policy = append([]byte(`{"updated_at":"2026-09-06T00:00:00Z",`), s.Policy[1:]...)
		},
		"credential-value": func(s *SystemUpdatePortPolicySnapshot) {
			s.Policy = append([]byte(`{"runtime_token":"synthetic-forbidden",`), s.Policy[1:]...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			mutant := portV2TestPolicySnapshot(t)
			mutate(&mutant)
			if ValidateSystemUpdatePortPolicySnapshot(mutant) == nil {
				t.Fatal("incomplete snapshot accepted")
			}
		})
	}
	reordered := portV2TestPolicySnapshot(t)
	reordered.Bindings[0], reordered.Bindings[1] = reordered.Bindings[1], reordered.Bindings[0]
	reordered.Targets[0], reordered.Targets[1] = reordered.Targets[1], reordered.Targets[0]
	var policy portSnapshotSourcePolicy
	_ = json.Unmarshal(reordered.Policy, &policy)
	policy.Targets[0], policy.Targets[1] = policy.Targets[1], policy.Targets[0]
	reordered.Policy, _ = json.Marshal(policy)
	if !EqualSystemUpdatePortPolicySnapshots(before, reordered) {
		t.Fatal("set ordering changed snapshot identity")
	}
	left, le := MarshalSystemUpdatePortPolicySnapshot(before)
	right, re := MarshalSystemUpdatePortPolicySnapshot(reordered)
	if le != nil || re != nil || !bytes.Equal(left, right) {
		t.Fatal("equivalent snapshots persisted different set order")
	}
}

func portV2TestDockerPolicySnapshot(t *testing.T) SystemUpdatePortPolicySnapshot {
	t.Helper()
	snapshot := portV2TestPolicySnapshot(t)
	docker := *portV2TestPlan(t, SystemUpdatePortModeLocalOnly, true, false).Before.Docker
	configSHA, err := SystemUpdateDockerPortConfigSHA256(SystemUpdateTargetWorker, docker.PublishedPort, docker.ContainerPort, snapshot.Targets[0].ConfigRevision)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Bindings[0].LocalListenPort = nil
	snapshot.Targets[0].DeploymentMode = SystemUpdateDeploymentDocker
	snapshot.Targets[0].ConfigSHA256 = configSHA
	snapshot.Targets[0].Docker = &docker
	var root portRootPolicy
	_ = json.Unmarshal(snapshot.RootPolicy, &root)
	root.Targets[0].DeploymentMode = "docker"
	root.Targets[0].Systemd = nil
	root.Targets[0].ConfigSHA256 = configSHA
	root.Targets[0].Docker = &portRootDocker{DockerPath: "/usr/bin/docker", ComposeProject: "autostream", ProjectDir: "/opt/autostream/docker", ComposeFiles: []string{"/opt/autostream/docker/compose.yml"}, Service: "worker", ImageRepo: "ghcr.io/autostream/worker", ImageVariable: "AUTOSTREAM_WORKER_IMAGE", VersionEnvFile: "/opt/autostream/docker/worker-version.env", PortEnvFile: "/opt/autostream/docker/worker-port.env", ComposeConfigSHA256: strings.Repeat("9", 64), PortComposePolicySHA256: strings.TrimPrefix(docker.ComposePolicySHA256, "sha256:"), PortComposeRevision: docker.ComposeRevision, CurrentVersion: "v1.2.3", Channel: "stable"}
	snapshot.RootPolicy, _ = json.Marshal(root)
	var policy portSnapshotSourcePolicy
	_ = json.Unmarshal(snapshot.Policy, &policy)
	policy.Targets[0].DeploymentMode = "docker"
	policy.LocalExecutorPolicySHA256 = ComputeSystemUpdatePortBytesSHA256(snapshot.RootPolicy)
	snapshot.Policy, _ = json.Marshal(policy)
	return snapshot
}

func TestSystemUpdatePortV2CompleteDockerSnapshot(t *testing.T) {
	complete := portV2TestDockerPolicySnapshot(t)
	if complete.Bindings[0].LocalListenPort != nil || complete.Targets[0].LocalListenPort != 18081 || complete.Targets[0].Docker.PublishedPort != 18081 {
		t.Fatal("fixture does not model a Docker root listener without a systemd binding")
	}
	if id, digest, err := ComputeSystemUpdatePortSnapshotIdentity(complete); err != nil || id == "" || digest == "" {
		t.Fatal("complete Docker snapshot rejected")
	}
	if payload, err := MarshalSystemUpdatePortPolicySnapshot(complete); err != nil || len(payload) == 0 {
		t.Fatal("complete Docker snapshot persistence rejected")
	}
	for name, mutate := range map[string]func(*SystemUpdatePortPolicySnapshot){
		"systemd-only-listener-row":   func(s *SystemUpdatePortPolicySnapshot) { port := 18081; s.Bindings[0].LocalListenPort = &port },
		"zero-listener-row":           func(s *SystemUpdatePortPolicySnapshot) { port := 0; s.Bindings[0].LocalListenPort = &port },
		"missing-binding":             func(s *SystemUpdatePortPolicySnapshot) { s.Bindings = s.Bindings[1:] },
		"missing-systemd-listener":    func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[1].LocalListenPort = nil },
		"missing-database":            func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[1].DatabaseName = nil },
		"orphan-binding":              func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].TargetID = "orphan-a" },
		"duplicate-binding":           func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[1] = s.Bindings[0] },
		"foreign-host":                func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].HostID = "host-b" },
		"stale-binding":               func(s *SystemUpdatePortPolicySnapshot) { s.Bindings[0].BindingPolicyRevision-- },
		"missing-docker-state":        func(s *SystemUpdatePortPolicySnapshot) { s.Targets[0].Docker = nil },
		"foreign-target-host":         func(s *SystemUpdatePortPolicySnapshot) { s.Targets[0].HostID = "host-b" },
		"published-listener-mismatch": func(s *SystemUpdatePortPolicySnapshot) { s.Targets[0].Docker.PublishedPort++ },
		"root-listener-mismatch": func(s *SystemUpdatePortPolicySnapshot) {
			s.Targets[0].LocalListenPort++
			s.Targets[0].Docker.PublishedPort++
			s.Targets[0].Docker.HealthPort++
		},
	} {
		t.Run(name, func(t *testing.T) {
			mutant := portV2TestDockerPolicySnapshot(t)
			mutate(&mutant)
			if ValidateSystemUpdatePortPolicySnapshot(mutant) == nil {
				t.Fatal("incomplete or inconsistent Docker snapshot accepted")
			}
		})
	}
}

func TestSystemUpdatePortV2RootDeltaPreservesAuthorityAndGeneratesRollback(t *testing.T) {
	snapshot := portV2TestPolicySnapshot(t)
	plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, false, false)
	before := *plan.Before
	before.ExecutorPolicySHA256 = ComputeSystemUpdatePortBytesSHA256(snapshot.RootPolicy)
	before.ConfigSHA256 = snapshot.Targets[0].ConfigSHA256
	var rootBefore portRootPolicy
	_ = json.Unmarshal(snapshot.RootPolicy, &rootBefore)
	for _, step := range []int64{1, 2} {
		after := *plan.Target
		if step == 2 {
			after = *plan.Rollback
		}
		after.ExecutorPolicySHA256 = ""
		config, err := SystemUpdatePortListenerConfig(SystemUpdateTargetWorker, SystemUpdateDeploymentSystemd, after.LocalListenPort, after.ConfigRevision)
		if err != nil {
			t.Fatal(err)
		}
		after.ConfigSHA256 = ComputeSystemUpdatePortBytesSHA256(config)
		body, err := ApplySystemUpdatePortPolicyDelta(snapshot.RootPolicy, "worker-a", before, after)
		if err != nil {
			t.Fatal(err)
		}
		var output portRootPolicy
		_ = json.Unmarshal(body, &output)
		if output.SourcePolicyRevision != 11+step || output.ProjectionRevision != 17+step || output.PolicyRevision != 23+step || output.Targets[0].ConfigRevision != 31+step {
			t.Fatal("independent counters changed incorrectly")
		}
		if !reflect.DeepEqual(output.Targets[1], rootBefore.Targets[1]) || output.HostID != rootBefore.HostID || output.AgentUID != rootBefore.AgentUID || output.AgentGID != rootBefore.AgentGID || output.SocketPath != rootBefore.SocketPath || !reflect.DeepEqual(output.Mutation, rootBefore.Mutation) || !bytes.Equal(output.Targets[0].Systemd, rootBefore.Targets[0].Systemd) {
			t.Fatal("port delta changed invariant authority")
		}
		if step == 2 && (output.Targets[0].LocalListen.Port != 18081 || output.Targets[0].ConfigSHA256 == before.ConfigSHA256 || bytes.Equal(body, snapshot.RootPolicy)) {
			t.Fatal("rollback reused old bytes or changed restored listener")
		}
		after.ExecutorPolicySHA256 = ComputeSystemUpdatePortBytesSHA256(body)
		if _, err := ApplySystemUpdatePortPolicyDelta(snapshot.RootPolicy, "worker-a", before, after); err != nil {
			t.Fatal("bound output rejected")
		}
		tampered := bytes.Replace(snapshot.RootPolicy, []byte("https://panel.example.com"), []byte("https://other.example.com"), 1)
		if _, err := ApplySystemUpdatePortPolicyDelta(tampered, "worker-a", before, after); err == nil {
			t.Fatal("tampered root origin accepted")
		}
		after.ExecutorPolicySHA256 = ""
		after.ConfigSHA256 = "sha256:" + strings.Repeat("9", 64)
		if _, err := ApplySystemUpdatePortPolicyDelta(snapshot.RootPolicy, "worker-a", before, after); err == nil {
			t.Fatal("config digest without corresponding listener bytes accepted")
		}
	}
	advertisedOnly := *plan.Target
	advertisedOnly.ExecutorPolicySHA256 = ""
	advertisedOnly.LocalListenPort = before.LocalListenPort
	if _, err := ApplySystemUpdatePortPolicyDelta(snapshot.RootPolicy, "worker-a", before, advertisedOnly); err == nil {
		t.Fatal("root delta accepted unchanged local tuple")
	}
}

func TestSystemUpdatePortV2DockerConfigMappingAndRootDelta(t *testing.T) {
	for _, tc := range []struct {
		service SystemUpdateTargetType
		prefix  string
	}{
		{SystemUpdateTargetWorker, "AUTOSTREAM_WORKER"},
		{SystemUpdateTargetEncoderRecorder, "AUTOSTREAM_ENCODER_RECORDER"},
		{SystemUpdateTargetDiscordBot, "AUTOSTREAM_DISCORD_BOT"},
		{SystemUpdateTargetObservability, "AUTOSTREAM_OBSERVABILITY"},
	} {
		t.Run(string(tc.service), func(t *testing.T) {
			payload, err := SystemUpdatePortDockerConfig(tc.service, 18081, 8080, 31)
			want := tc.prefix + "_PORT=18081\n" + tc.prefix + "_CONTAINER_PORT=8080\nAUTOSTREAM_CONFIG_REVISION=31\n"
			if err != nil || string(payload) != want {
				t.Fatal("Docker mapping differs from installed three-line encoding")
			}
			digest, err := SystemUpdateDockerPortConfigSHA256(tc.service, 18081, 8080, 31)
			if err != nil || digest != ComputeSystemUpdatePortBytesSHA256([]byte(want)) {
				t.Fatal("Docker root config digest does not bind mapping bytes")
			}
			derived, err := SystemUpdatePortListenerConfig(tc.service, SystemUpdateDeploymentDocker, 8080, 31)
			if err != nil || digest == ComputeSystemUpdatePortBytesSHA256(derived) {
				t.Fatal("Docker root mapping and derived Node projection were conflated")
			}
		})
	}
	for _, tc := range []struct {
		service              SystemUpdateTargetType
		published, container int
		revision             int64
	}{
		{SystemUpdateTargetControlPanel, 18081, 8080, 31},
		{SystemUpdateTargetWorker, 80, 8080, 31},
		{SystemUpdateTargetWorker, 18081, 80, 31},
		{SystemUpdateTargetWorker, 18081, 8080, 0},
	} {
		if _, err := SystemUpdatePortDockerConfig(tc.service, tc.published, tc.container, tc.revision); err == nil {
			t.Fatal("invalid Docker mapping accepted")
		}
	}

	snapshot := portV2TestPolicySnapshot(t)
	plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, true, false)
	before := *plan.Before
	before.ConfigSHA256, _ = SystemUpdateDockerPortConfigSHA256(SystemUpdateTargetWorker, before.Docker.PublishedPort, before.Docker.ContainerPort, before.ConfigRevision)
	var rootBefore portRootPolicy
	_ = json.Unmarshal(snapshot.RootPolicy, &rootBefore)
	rootBefore.Targets[0].DeploymentMode = "docker"
	rootBefore.Targets[0].Systemd = nil
	rootBefore.Targets[0].ConfigSHA256 = before.ConfigSHA256
	rootBefore.Targets[0].Docker = &portRootDocker{DockerPath: "/usr/bin/docker", ComposeProject: "autostream", ProjectDir: "/opt/autostream/docker", ComposeFiles: []string{"/opt/autostream/docker/compose.yml"}, Service: "worker", ImageRepo: "ghcr.io/autostream/worker", ImageVariable: "AUTOSTREAM_WORKER_IMAGE", VersionEnvFile: "/opt/autostream/docker/worker-version.env", PortEnvFile: "/opt/autostream/docker/worker-port.env", ComposeConfigSHA256: strings.Repeat("9", 64), PortComposePolicySHA256: strings.TrimPrefix(before.Docker.ComposePolicySHA256, "sha256:"), PortComposeRevision: before.Docker.ComposeRevision, CurrentVersion: "v1.2.3", Channel: "stable"}
	rootBytes, _ := json.Marshal(rootBefore)
	before.ExecutorPolicySHA256 = ComputeSystemUpdatePortBytesSHA256(rootBytes)
	for _, afterRef := range []*SystemUpdatePortSnapshotRef{plan.Target, plan.Rollback} {
		after := *afterRef
		after.ExecutorPolicySHA256 = ""
		after.ConfigSHA256, _ = SystemUpdateDockerPortConfigSHA256(SystemUpdateTargetWorker, after.Docker.PublishedPort, after.Docker.ContainerPort, after.ConfigRevision)
		body, err := ApplySystemUpdatePortPolicyDelta(rootBytes, "worker-a", before, after)
		if err != nil {
			t.Fatal("installed Docker mapping delta rejected")
		}
		var observed portRootPolicy
		_ = json.Unmarshal(body, &observed)
		if observed.Targets[0].ConfigSHA256 != after.ConfigSHA256 || observed.Targets[0].LocalListen.Port != after.Docker.PublishedPort || observed.Targets[0].Docker.PortComposeRevision != after.Docker.ComposeRevision {
			t.Fatal("Docker root did not adopt the authorized mapping")
		}
		expectedProfile := *rootBefore.Targets[0].Docker
		expectedProfile.PortComposeRevision = after.Docker.ComposeRevision
		if !reflect.DeepEqual(observed.Targets[0].Docker, &expectedProfile) || !reflect.DeepEqual(observed.Targets[1], rootBefore.Targets[1]) {
			t.Fatal("Docker delta changed existing non-port authority")
		}
		derived, _ := SystemUpdatePortListenerConfig(SystemUpdateTargetWorker, SystemUpdateDeploymentDocker, after.Docker.ContainerPort, after.ConfigRevision)
		after.ConfigSHA256 = ComputeSystemUpdatePortBytesSHA256(derived)
		if _, err := ApplySystemUpdatePortPolicyDelta(rootBytes, "worker-a", before, after); err == nil {
			t.Fatal("Docker root accepted derived Node JSON as the mapping digest")
		}
	}
}

func TestSystemUpdatePortV2BaselineAndDockerResult(t *testing.T) {
	for _, docker := range []bool{false, true} {
		plan := portV2TestPlan(t, SystemUpdatePortModeLocalOnly, docker, false)
		target := UpdaterPortPolicyBaselineTarget{ServiceID: "worker-a", ServiceType: SystemUpdateTargetWorker, DeploymentMode: SystemUpdateDeploymentSystemd, EndpointRevision: plan.Before.AppliedEndpointRevision, ConfigRevision: plan.Before.ConfigRevision, ConfigSHA256: plan.Before.ConfigSHA256, LocalListenPort: plan.Before.LocalListenPort}
		if docker {
			target.DeploymentMode = SystemUpdateDeploymentDocker
			target.Docker = plan.Before.Docker
			target.DockerRoot = &UpdaterPortDockerRootBaseline{ComposeConfigSHA256: strings.Repeat("9", 64), CurrentVersion: "v1.2.3"}
		}
		baseline := UpdaterPortPolicyBaseline{PortContractVersion: 2, PolicyTransitionVersion: 1, AgentUID: 1001, AgentGID: 1001, SourcePolicyRevision: 11, ProjectionRevision: 17, ExecutorPolicyRevision: 23, ExecutorPolicySHA256: plan.Before.ExecutorPolicySHA256, Targets: []UpdaterPortPolicyBaselineTarget{target}, ObservedAt: testUpdaterTime()}
		if ValidateUpdaterPortPolicyBaselineJSON(testUpdaterJSON(t, baseline)) != nil {
			t.Fatal("complete metadata baseline rejected")
		}
		for _, key := range []string{"agent_uid", "executor_policy_sha256", "policy_transition_version"} {
			mutant := testUpdaterObject(t, testUpdaterJSON(t, baseline))
			delete(mutant, key)
			if ValidateUpdaterPortPolicyBaselineJSON(testUpdaterJSON(t, mutant)) == nil {
				t.Fatal("incomplete baseline accepted")
			}
		}
		mutant := testUpdaterObject(t, testUpdaterJSON(t, baseline))
		mutant["root_policy"] = map[string]any{"path": "/unapproved"}
		if ValidateUpdaterPortPolicyBaselineJSON(testUpdaterJSON(t, mutant)) == nil {
			t.Fatal("arbitrary policy metadata accepted")
		}
		if docker {
			baseline.Targets[0].DockerRoot = nil
			if ValidateUpdaterPortPolicyBaseline(baseline) == nil {
				t.Fatal("missing existing Docker root metadata accepted")
			}
			result := portV2TestResult(plan.Target, SystemUpdatePortReconfigurationApplied)
			if ValidateSystemUpdatePortResult(plan, result) == nil {
				t.Fatal("Docker success without instance accepted")
			}
			result.RuntimeInstance = &SystemUpdatePortRuntimeInstance{ContainerID: strings.Repeat("a", 64), ImageID: plan.Target.Docker.ImageID, RepositoryDigest: plan.Target.Docker.RepositoryDigest}
			if ValidateSystemUpdatePortResultJSON(plan, testUpdaterJSON(t, result)) != nil {
				t.Fatal("complete Docker result rejected")
			}
			result.RuntimeInstance.ImageID = "sha256:" + strings.Repeat("b", 64)
			if ValidateSystemUpdatePortResult(plan, result) == nil {
				t.Fatal("changed image accepted as port-only operation")
			}
		}
	}
}
