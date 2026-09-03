package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	dockerPortComposeSHA256    = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	dockerPortVersionEnvDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	dockerPortImageDigest      = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	dockerPortRepoDigest       = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

func dockerPortReconfigurationPlanJSON() string {
	return `{
		"network_namespace":"host",
		"protocol":"tcp",
		"old_port":80,
		"new_port":18080,
		"expected_endpoint_revision":7,
		"target_endpoint_revision":8,
		"expected_config_revision":11,
		"target_config_revision":12,
		"expected_config_sha256":"` + portContractConfigDigest + `",
		"target_config_sha256":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"expected_source_policy_revision":19,
		"expected_updater_policy_revision":23,
		"expected_executor_policy_revision":5,
		"expected_executor_policy_sha256":"` + portContractExecutorDigest + `",
		"port_plan_sha256":"` + portContractPlanSHA256 + `",
		"docker":{
			"published_host_ip":"127.0.0.1",
			"old_published_port":18084,
			"new_published_port":28084,
			"old_container_port":8084,
			"new_container_port":8085,
			"old_health_port":18084,
			"new_health_port":28084,
			"approved_compose_config_sha256":"` + dockerPortComposeSHA256 + `",
			"approved_compose_revision":4,
			"expected_version_env_sha256":"` + dockerPortVersionEnvDigest + `",
			"expected_container_id":"0123456789ab",
			"expected_image_id":"` + dockerPortImageDigest + `",
			"expected_repository_digest":"` + dockerPortRepoDigest + `"
		}
	}`
}

func dockerPortReconfigurationJobJSON() string {
	return `{` + portContractV2JobFields + `
		"id":"job-docker-port-1",
		"target_id":"worker-a",
		"target_type":"worker",
		"host_id":"host-a",
		"transport_mode":"pull_v2",
		"ownership_epoch":2,
		"policy_revision":23,
		"deployment_mode":"docker",
		"current_version":"v1.2.3",
		"target_version":"v1.2.3",
		"strategy":"maintenance",
		"status":"queued",
		"idempotency_key":"docker-port-worker-a-28084",
		"lease_generation":1,
		"sequence":0,
		"progress":0,
		"created_at":"2026-07-28T00:00:00Z",
		"updated_at":"2026-07-28T00:00:00Z",
		"operation":"port_reconfigure",
		"port_reconfigure":` + dockerPortReconfigurationPlanJSON() + `
	}`
}

func TestDockerPortCreateRequestIsDisjointFromSystemdPortRequest(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-create-request.schema.json")

	dockerRequest := `{` + portContractV2CreateFields + `
		"operation":"port_reconfigure",
		"target_id":"worker-a",
		"new_advertised_port":18080,
		"new_published_port":28084,
		"new_container_port":8085,
		"expected_endpoint_revision":7,
		"idempotency_key":"docker-port-worker-a-28084"
	}`
	validatePortContractJSON(t, schema, dockerRequest, true)

	for _, invalid := range []string{
		strings.Replace(dockerRequest, `"new_advertised_port":18080,`, "", 1),
		strings.Replace(dockerRequest, `"new_published_port":28084,`, "", 1),
		strings.Replace(dockerRequest, `"new_container_port":8085,`, "", 1),
		strings.Replace(dockerRequest, `"new_advertised_port":18080`, `"new_advertised_port":0`, 1),
		strings.Replace(dockerRequest, `"new_published_port":28084`, `"new_published_port":1023`, 1),
		strings.Replace(dockerRequest, `"new_container_port":8085`, `"new_container_port":65536`, 1),
		strings.Replace(dockerRequest, `"new_advertised_port":18080,`, `"new_advertised_port":18080,"new_port":18080,`, 1),
	} {
		validatePortContractJSON(t, schema, invalid, false)
	}
}

func TestDockerPortJobRequiresCompleteDockerBinding(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-job.schema.json")
	job := dockerPortReconfigurationJobJSON()
	validatePortContractJSON(t, schema, job, true)

	for _, invalid := range []string{
		strings.Replace(job, `"docker":{`, `"docker":{"unexpected":"value",`, 1),
		strings.Replace(job, `"published_host_ip":"127.0.0.1"`, `"published_host_ip":"0.0.0.0"`, 1),
		strings.Replace(job, `"approved_compose_config_sha256":"`+dockerPortComposeSHA256+`"`, `"approved_compose_config_sha256":"sha256:`+dockerPortComposeSHA256+`"`, 1),
		strings.Replace(job, `"approved_compose_revision":4`, `"approved_compose_revision":0`, 1),
		strings.Replace(job, `"expected_version_env_sha256":"`+dockerPortVersionEnvDigest+`",`, "", 1),
		strings.Replace(job, `"expected_container_id":"0123456789ab"`, `"expected_container_id":"ABCDEF012345"`, 1),
		strings.Replace(job, `"expected_image_id":"`+dockerPortImageDigest+`"`, `"expected_image_id":"1111"`, 1),
		strings.Replace(job, `"expected_repository_digest":"`+dockerPortRepoDigest+`"`, `"expected_repository_digest":"2222"`, 1),
		strings.Replace(job, `"docker":{`, `"docker_missing":{`, 1),
	} {
		validatePortContractJSON(t, schema, invalid, false)
	}

	systemdJob := portReconfigurationJobJSON()
	validatePortContractJSON(t, schema, strings.Replace(systemdJob, `"port_plan_sha256":"`+portContractPlanSHA256+`"`, `"port_plan_sha256":"`+portContractPlanSHA256+`","docker":{}`, 1), false)
	validatePortContractJSON(t, schema, strings.Replace(systemdJob, `"old_port":8084`, `"old_port":80`, 1), false)
}

func TestDockerPortMutationGrantsBindTheCompleteDockerPlan(t *testing.T) {
	issueSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	consumeSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-consume-request.schema.json")
	plan := dockerPortReconfigurationPlanJSON()

	issue := `{
		"service_id":"host-agent-a",
		"lease_token":"` + strings.Repeat("l", 43) + `",
		"lease_generation":2,
		"host_id":"host-a",
		"transport_mode":"pull_v2",
		"ownership_epoch":2,
		"policy_revision":23,
		"target_id":"worker-a",
		"service_type":"worker",
		"target_version":"v1.2.3",
		"deployment_mode":"docker",
		"job_operation":"port_reconfigure",
		"operation":"port_reconfigure",
		"plan_sha256":"` + portContractPlanSHA256 + `",
		"session_id":"docker-port-0001",
		"port_reconfigure":` + plan + `
	}`
	consume := `{
		"lease_generation":2,
		"host_id":"host-a",
		"transport_mode":"pull_v2",
		"ownership_epoch":2,
		"policy_revision":23,
		"target_id":"worker-a",
		"service_type":"worker",
		"target_version":"v1.2.3",
		"deployment_mode":"docker",
		"job_operation":"port_reconfigure",
		"operation":"port_reconfigure_reconcile",
		"plan_sha256":"` + portContractPlanSHA256 + `",
		"session_id":"docker-port-0001",
		"port_reconfigure":` + plan + `
	}`
	validatePortContractJSON(t, issueSchema, issue, true)
	validatePortContractJSON(t, consumeSchema, consume, true)
	validatePortContractJSON(t, issueSchema, strings.Replace(issue, `"expected_container_id":"0123456789ab",`, "", 1), false)
	validatePortContractJSON(t, consumeSchema, strings.Replace(consume, `"docker":{`, `"docker_missing":{`, 1), false)
}

func TestSystemUpdateTargetDockerPortMappingIsAllowlisted(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-target.schema.json")
	applied := `{` + portContractV2TargetFields + `
		"target_id":"worker-a",
		"target_type":"worker",
		"name":"Worker A",
		"host_id":"host-a",
		"update_available":false,
		"deployment_mode":"docker",
		"updater_online":true,
		"eligible":true,
		"busy":false,
		"port_mapping":{
			"mode":"docker",
			"state":"applied",
			"advertised_port":18080,
			"published_port":28084,
			"container_port":8085,
			"health_port":28084,
			"config_revision":12,
			"published_host_ip":"127.0.0.1",
			"reported_at":"2026-07-28T00:00:00Z"
		}
	}`
	validatePortContractJSON(t, schema, applied, true)
	validatePortContractJSON(t, schema, strings.Replace(applied, `"reported_at":"2026-07-28T00:00:00Z"`, `"unexpected":"value"`, 1), false)
	validatePortContractJSON(t, schema, strings.Replace(applied, `"mode":"docker"`, `"mode":"systemd"`, 1), false)
	validatePortContractJSON(t, schema, strings.Replace(applied, `"state":"applied"`, `"state":"unknown"`, 1), false)
	validatePortContractJSON(t, schema, strings.Replace(applied, `"published_port":28084,`, "", 1), false)

	for _, state := range []string{"drifted", "unavailable"} {
		validatePortContractJSON(t, schema, `{`+portContractV2TargetFields+`
			"host_id":"host-a",
			"target_id":"worker-a",
			"target_type":"worker",
			"name":"Worker A",
			"update_available":false,
			"deployment_mode":"docker",
			"updater_online":true,
			"eligible":false,
			"busy":false,
			"port_mapping":{"mode":"docker","state":"`+state+`"}
		}`, true)
	}
}

func TestDockerPortGoTypesAndOpenAPIAreAdditive(t *testing.T) {
	body, err := json.Marshal(SystemUpdateCreateRequest{
		ProtocolVersion: 2, DesiredRevision: 12, Fence: 3, RequiredCapability: UpdaterCapabilityPort,
		Operation:                SystemUpdateOperationPortReconfigure,
		TargetID:                 "worker-a",
		NewAdvertisedPort:        18080,
		NewPublishedPort:         28084,
		NewContainerPort:         8085,
		ExpectedEndpointRevision: 7,
		IdempotencyKey:           "docker-port-worker-a-28084",
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-create-request.schema.json"), string(body), true)

	reportedAt := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	dockerPlan := &SystemUpdatePortReconfiguration{
		NetworkNamespace: "host", Protocol: SystemUpdatePortProtocolTCP,
		OldPort: 80, NewPort: 18080,
		ExpectedEndpointRevision: 7, TargetEndpointRevision: 8,
		ExpectedConfigRevision: 11, TargetConfigRevision: 12,
		ExpectedConfigSHA256:         portContractConfigDigest,
		TargetConfigSHA256:           "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		ExpectedSourcePolicyRevision: 19, ExpectedUpdaterPolicyRevision: 23,
		ExpectedExecutorPolicyRevision: 5, ExpectedExecutorPolicySHA256: portContractExecutorDigest,
		PortPlanSHA256: portContractPlanSHA256,
		Docker: &SystemUpdateDockerPortReconfiguration{
			PublishedHostIP:  "127.0.0.1",
			OldPublishedPort: 18084, NewPublishedPort: 28084,
			OldContainerPort: 8084, NewContainerPort: 8085,
			OldHealthPort: 18084, NewHealthPort: 28084,
			ApprovedComposeConfigSHA256: dockerPortComposeSHA256,
			ApprovedComposeRevision:     4,
			ExpectedVersionEnvSHA256:    dockerPortVersionEnvDigest,
			ExpectedContainerID:         "0123456789ab",
			ExpectedImageID:             dockerPortImageDigest,
			ExpectedRepositoryDigest:    dockerPortRepoDigest,
		},
	}
	jobBody, err := json.Marshal(SystemUpdateJob{
		ProtocolVersion: 2, UpdaterID: "host-agent-a", DesiredRevision: 12, Fence: 3,
		Outcome: "pending", RequiredCapability: UpdaterCapabilityPort, AuthorizationID: "authorization-1",
		CanonicalPayloadDigest: portContractConfigDigest, AutomaticResendAllowed: systemUpdateAutomaticResendDisabledFixture(),
		LeaseGeneration: 1,
		ID:              "job-docker-port-1", TargetID: "worker-a", TargetType: SystemUpdateTargetWorker,
		ExecutionHostID: "host-a", TransportMode: UpdateTransportPullV2,
		OwnershipEpoch: 2, PolicyRevision: 23, DeploymentMode: SystemUpdateDeploymentDocker,
		CurrentVersion: "v1.2.3", TargetVersion: "v1.2.3", Strategy: SystemUpdateMaintenance,
		Status: SystemUpdateQueued, IdempotencyKey: "docker-port-worker-a-28084",
		Operation: SystemUpdateOperationPortReconfigure, PortReconfigure: dockerPlan,
		CreatedAt: reportedAt, UpdatedAt: reportedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-job.schema.json"), string(jobBody), true)

	targetBody, err := json.Marshal(SystemUpdateTarget{
		ProtocolVersion: 2, HostID: "host-a", UpdaterID: "host-agent-a", Capabilities: []UpdaterCapability{UpdaterCapabilityPort},
		DesiredRevision: 12, AppliedRevision: 11, Fence: 3,
		UpdaterHealth:    &UpdaterHealth{Status: "ready", Revision: 12},
		ApplicationProbe: &ApplicationRuntimeIdentityProbe{Version: "v1.2.3", ServiceID: "worker-a", ServiceType: SystemUpdateTargetWorker, ConfigRevision: 11},
		TargetID:         "worker-a", TargetType: SystemUpdateTargetWorker, Name: "Worker A",
		DeploymentMode: SystemUpdateDeploymentDocker, UpdaterOnline: true, Eligible: true,
		PortMapping: &SystemUpdatePortMapping{
			Mode: SystemUpdateDeploymentDocker, State: SystemUpdatePortMappingApplied,
			AdvertisedPort: 18080, PublishedPort: 28084, ContainerPort: 8085, HealthPort: 28084,
			ConfigRevision: 12, PublishedHostIP: "127.0.0.1",
			ReportedAt: &reportedAt,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-target.schema.json"), string(targetBody), true)

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"SystemUpdateDockerPortReconfiguration:",
		"SystemUpdatePortMapping:",
		"approved_compose_config_sha256:",
		"expected_repository_digest:",
		"published_host_ip:",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing Docker port marker %q", marker)
		}
	}
	requireControlOpenAPISystemUpdateCreateRequest(t)
}
