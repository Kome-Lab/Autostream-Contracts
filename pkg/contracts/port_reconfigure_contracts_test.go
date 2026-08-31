package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	portContractConfigDigest   = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	portContractExecutorDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	portContractPlanSHA256     = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func validatePortContractJSON(t *testing.T, schema *jsonschema.Schema, body string, wantValid bool) {
	t.Helper()
	var instance any
	if err := json.Unmarshal([]byte(body), &instance); err != nil {
		t.Fatal(err)
	}
	err := schema.Validate(instance)
	if wantValid && err != nil {
		t.Fatalf("expected valid port-reconfiguration contract for %s: %v", body, err)
	}
	if !wantValid && err == nil {
		t.Fatalf("expected port-reconfiguration contract rejection for %s", body)
	}
}

func portReconfigurationPlanJSON() string {
	return `{
		"network_namespace":"host",
		"protocol":"tcp",
		"old_port":8084,
		"new_port":18084,
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
		"port_plan_sha256":"` + portContractPlanSHA256 + `"
	}`
}

func portReconfigurationJobJSON() string {
	return `{
		"id":"job-port-1",
		"target_id":"worker-a",
		"target_type":"worker",
		"host_id":"host-a",
		"transport_mode":"pull_v2",
		"ownership_epoch":2,
		"policy_revision":23,
		"deployment_mode":"systemd",
		"current_version":"v1.2.3",
		"target_version":"v1.2.3",
		"strategy":"maintenance",
		"status":"queued",
		"idempotency_key":"port-worker-a-18084",
		"lease_generation":0,
		"sequence":0,
		"progress":0,
		"created_at":"2026-07-28T00:00:00Z",
		"updated_at":"2026-07-28T00:00:00Z",
		"operation":"port_reconfigure",
		"port_reconfigure":` + portReconfigurationPlanJSON() + `
	}`
}

func TestSystemUpdateCreateRequestSeparatesSoftwareAndPortOperations(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-create-request.schema.json")

	validatePortContractJSON(t, schema, `{
		"target_id":"worker-a",
		"strategy":"when_idle",
		"idempotency_key":"software-worker-a"
	}`, true)
	validatePortContractJSON(t, schema, `{
		"operation":"software_update",
		"target_id":"worker-a",
		"strategy":"maintenance",
		"idempotency_key":"software-worker-a-explicit"
	}`, true)
	validatePortContractJSON(t, schema, `{
		"operation":"port_reconfigure",
		"target_id":"worker-a",
		"new_port":18084,
		"expected_endpoint_revision":7,
		"idempotency_key":"port-worker-a-18084"
	}`, true)

	for _, invalid := range []string{
		`{"operation":"port_reconfigure","target_id":"worker-a","new_port":18084,"idempotency_key":"missing-fence"}`,
		`{"operation":"port_reconfigure","target_id":"worker-a","new_port":1023,"expected_endpoint_revision":7,"idempotency_key":"privileged-port"}`,
		`{"operation":"port_reconfigure","target_id":"worker-a","new_port":65536,"expected_endpoint_revision":7,"idempotency_key":"invalid-port"}`,
		`{"operation":"port_reconfigure","target_id":"worker-a","new_port":18084,"expected_endpoint_revision":7,"strategy":"maintenance","idempotency_key":"mixed-operation"}`,
		`{"operation":"port_reconfigure","target_id":"worker-a","new_port":18084,"expected_endpoint_revision":7,"old_port":8084,"idempotency_key":"client-controls-old-port"}`,
		`{"operation":"software_update","target_id":"worker-a","new_port":18084,"expected_endpoint_revision":7,"strategy":"maintenance","idempotency_key":"software-with-port"}`,
	} {
		validatePortContractJSON(t, schema, invalid, false)
	}
}

func TestPortReconfigurationJobClaimAndReportUseNestedWireShapes(t *testing.T) {
	jobSchema := compileContractJSONSchema(t, "system-update-job.schema.json")
	claimSchema := compileContractJSONSchema(t, "update-agent-claim-response.schema.json", "system-update-job.schema.json")
	reportSchema := compileContractJSONSchema(t, "update-agent-report-request.schema.json")

	job := portReconfigurationJobJSON()
	validatePortContractJSON(t, jobSchema, job, true)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"protocol":"tcp"`, `"protocol":"sctp"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"protocol":"tcp"`, `"protocol":"udp"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"network_namespace":"host"`, `"network_namespace":"container"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"old_port":8084,`, "", 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"ownership_epoch":2`, `"ownership_epoch":0`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"target_type":"worker"`, `"target_type":"control_panel"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"transport_mode":"pull_v2"`, `"transport_mode":"ssh_v1"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"transport_mode":"pull_v2",`, "", 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"deployment_mode":"systemd"`, `"deployment_mode":"docker"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"expected_source_policy_revision":19,`, "", 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"expected_executor_policy_revision":5`, `"expected_executor_policy_revision":0`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(job, `"port_plan_sha256":"`+portContractPlanSHA256+`"`, `"port_plan_sha256":"`+strings.ToUpper(portContractPlanSHA256)+`"`, 1), false)
	succeededJob := strings.Replace(job, `"status":"queued"`, `"status":"succeeded"`, 1)
	succeededJob = strings.Replace(succeededJob, `"port_plan_sha256":"`+portContractPlanSHA256+`"`, `"port_plan_sha256":"`+portContractPlanSHA256+`","result":"applied"`, 1)
	validatePortContractJSON(t, jobSchema, succeededJob, true)
	validatePortContractJSON(t, jobSchema, strings.Replace(succeededJob, `"result":"applied"`, `"result":"rolled_back"`, 1), false)
	validatePortContractJSON(t, jobSchema, strings.Replace(succeededJob, `"status":"succeeded"`, `"status":"installing"`, 1), false)

	claim := `{
		"job":` + job + `,
		"lease_token":"` + strings.Repeat("l", 43) + `",
		"lease_expires_at":"2026-07-28T00:01:00Z",
		"lease_generation":1,
		"report_sequence":1,
		"recovery_required":false,
		"last_status":"queued"
	}`
	validatePortContractJSON(t, claimSchema, claim, true)

	for _, terminal := range []struct {
		status string
		result string
	}{
		{status: "succeeded", result: "applied"},
		{status: "succeeded", result: "unchanged"},
		{status: "rolled_back", result: "rolled_back"},
		{status: "failed", result: "rollback_failed"},
	} {
		report := `{
			"service_id":"host-agent-a",
			"lease_token":"` + strings.Repeat("l", 43) + `",
			"lease_generation":1,
			"sequence":1,
			"status":"` + terminal.status + `",
			"port_reconfigure":{"result":"` + terminal.result + `"}
		}`
		validatePortContractJSON(t, reportSchema, report, true)
	}
	reportPrefix := `{
		"service_id":"host-agent-a",
		"lease_token":"` + strings.Repeat("l", 43) + `",
		"lease_generation":1,
		"sequence":1,`
	for _, invalid := range []string{
		reportPrefix + `"status":"failed","port_reconfigure":{"result":"partially_applied"}}`,
		reportPrefix + `"status":"installing","port_reconfigure":{"result":"applied"}}`,
		reportPrefix + `"status":"succeeded","port_reconfigure":{"result":"rolled_back"}}`,
		reportPrefix + `"status":"rolled_back","port_reconfigure":{"result":"unchanged"}}`,
		reportPrefix + `"status":"failed","port_reconfigure":{"result":"applied"}}`,
		reportPrefix + `"status":"succeeded","port_reconfigure":{"result":"applied","new_port":18084}}`,
		reportPrefix + `"status":"failed","ownership_epoch":2}`,
	} {
		validatePortContractJSON(t, reportSchema, invalid, false)
	}
	validatePortContractJSON(t, reportSchema, reportPrefix+`"status":"installing","progress":70}`, true)
}

func TestPortReconfigurationMutationGrantsBindTheImmutableJobPlan(t *testing.T) {
	issueSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-issue-request.schema.json")
	consumeSchema := compileContractJSONSchema(t, "update-agent-mutation-grant-consume-request.schema.json")

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
		"deployment_mode":"systemd",
		"job_operation":"port_reconfigure",
		"operation":"port_reconfigure",
		"plan_sha256":"` + portContractPlanSHA256 + `",
		"session_id":"port-contract-0001",
		"port_reconfigure":` + portReconfigurationPlanJSON() + `
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
		"deployment_mode":"systemd",
		"job_operation":"port_reconfigure",
		"operation":"port_reconfigure_reconcile",
		"plan_sha256":"` + portContractPlanSHA256 + `",
		"session_id":"port-contract-0001",
		"port_reconfigure":` + portReconfigurationPlanJSON() + `
	}`
	validatePortContractJSON(t, issueSchema, issue, true)
	validatePortContractJSON(t, consumeSchema, consume, true)

	for _, invalid := range []string{
		strings.Replace(issue, `"job_operation":"port_reconfigure",`, "", 1),
		strings.Replace(issue, `"service_type":"worker",`, "", 1),
		strings.Replace(issue, `"service_type":"worker"`, `"service_type":"control_panel"`, 1),
		strings.Replace(issue, `"expected_source_policy_revision":19,`, "", 1),
		strings.Replace(issue, `"port_reconfigure":`+portReconfigurationPlanJSON(), `"port_reconfigure":{"new_port":18084}`, 1),
		strings.Replace(issue, `"job_operation":"port_reconfigure"`, `"job_operation":"software_update"`, 1),
		strings.Replace(issue, `"operation":"port_reconfigure"`, `"operation":"apply"`, 1),
		strings.Replace(issue, `"ownership_epoch":2`, `"ownership_epoch":0`, 1),
		strings.Replace(issue, `"transport_mode":"pull_v2"`, `"transport_mode":"ssh_v1"`, 1),
		strings.Replace(issue, `"transport_mode":"pull_v2",`, "", 1),
		strings.Replace(issue, `"deployment_mode":"systemd"`, `"deployment_mode":"docker"`, 1),
		strings.Replace(issue, `"protocol":"tcp"`, `"protocol":"udp"`, 1),
		strings.Replace(issue, `"network_namespace":"host"`, `"network_namespace":"container"`, 1),
		strings.Replace(issue, `"port_plan_sha256":"`+portContractPlanSHA256+`"`, `"port_plan_sha256":"`+portContractPlanSHA256+`","result":"applied"`, 1),
	} {
		validatePortContractJSON(t, issueSchema, invalid, false)
	}
	for _, invalid := range []string{
		strings.Replace(consume, `"job_operation":"port_reconfigure",`, "", 1),
		strings.Replace(consume, `"service_type":"worker",`, "", 1),
		strings.Replace(consume, `"service_type":"worker"`, `"service_type":"control_panel"`, 1),
		strings.Replace(consume, `"job_operation":"port_reconfigure"`, `"job_operation":"software_update"`, 1),
		strings.Replace(consume, `"operation":"port_reconfigure_reconcile"`, `"operation":"reconcile"`, 1),
		strings.Replace(consume, `"transport_mode":"pull_v2"`, `"transport_mode":"ssh_v1"`, 1),
		strings.Replace(consume, `"deployment_mode":"systemd"`, `"deployment_mode":"docker"`, 1),
	} {
		validatePortContractJSON(t, consumeSchema, invalid, false)
	}
}

func TestSystemUpdateTargetAdvertisesOperationSpecificEligibility(t *testing.T) {
	schema := compileContractJSONSchema(t, "system-update-target.schema.json")
	valid := `{
		"target_id":"worker-a",
		"target_type":"worker",
		"name":"Worker A",
		"host_id":"host-a",
		"update_available":false,
		"deployment_mode":"systemd",
		"updater_online":true,
		"eligible":false,
		"blocked_reason":"system_update_release_unavailable",
		"eligible_operations":["port_reconfigure"],
		"operation_blocked_reasons":{"software_update":"system_update_release_unavailable"},
		"busy":false
	}`
	validatePortContractJSON(t, schema, valid, true)

	for _, invalid := range []string{
		strings.Replace(valid, `["port_reconfigure"]`, `["port_reconfigure","port_reconfigure"]`, 1),
		strings.Replace(valid, `["port_reconfigure"]`, `["delete_service"]`, 1),
		strings.Replace(valid, `{"software_update":"system_update_release_unavailable"}`, `{"delete_service":"blocked"}`, 1),
	} {
		validatePortContractJSON(t, schema, invalid, false)
	}

	body, err := json.Marshal(SystemUpdateTarget{
		TargetID: "worker-a", TargetType: SystemUpdateTargetWorker, Name: "Worker A",
		HostID: "host-a", DeploymentMode: SystemUpdateDeploymentSystemd,
		UpdaterOnline: true, Eligible: false,
		BlockedReason:      "system_update_release_unavailable",
		EligibleOperations: []SystemUpdateOperation{SystemUpdateOperationPortReconfigure},
		OperationBlockedReasons: map[string]string{
			string(SystemUpdateOperationSoftwareUpdate): "system_update_release_unavailable",
		},
		Busy: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, schema, string(body), true)
}

func TestPortReconfigurationGoTypesPreserveSoftwareCompatibility(t *testing.T) {
	software, err := json.Marshal(SystemUpdateCreateRequest{
		TargetID: "worker-a", Strategy: SystemUpdateWhenIdle, IdempotencyKey: "software-worker-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(software), "operation") || strings.Contains(string(software), "port_reconfigure") || strings.Contains(string(software), "new_port") {
		t.Fatalf("legacy software request leaked additive port fields: %s", software)
	}

	portRequest, err := json.Marshal(SystemUpdateCreateRequest{
		Operation:                SystemUpdateOperationPortReconfigure,
		TargetID:                 "worker-a",
		NewPort:                  18084,
		ExpectedEndpointRevision: 7,
		IdempotencyKey:           "port-worker-a-18084",
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-create-request.schema.json"), string(portRequest), true)

	portPlan := &SystemUpdatePortReconfiguration{
		NetworkNamespace:               "host",
		Protocol:                       "tcp",
		OldPort:                        8084,
		NewPort:                        18084,
		ExpectedEndpointRevision:       7,
		TargetEndpointRevision:         8,
		ExpectedConfigRevision:         11,
		TargetConfigRevision:           12,
		ExpectedConfigSHA256:           portContractConfigDigest,
		TargetConfigSHA256:             "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		ExpectedSourcePolicyRevision:   19,
		ExpectedUpdaterPolicyRevision:  23,
		ExpectedExecutorPolicyRevision: 5,
		ExpectedExecutorPolicySHA256:   portContractExecutorDigest,
		PortPlanSHA256:                 portContractPlanSHA256,
	}
	now := time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC)
	portJob, err := json.Marshal(SystemUpdateJob{
		ID: "job-port-1", TargetID: "worker-a", TargetType: SystemUpdateTargetWorker,
		ExecutionHostID: "host-a", TransportMode: UpdateTransportPullV2,
		OwnershipEpoch: 2, PolicyRevision: 23, DeploymentMode: SystemUpdateDeploymentSystemd,
		CurrentVersion: "v1.2.3", TargetVersion: "v1.2.3", Strategy: SystemUpdateMaintenance,
		Status: SystemUpdateQueued, IdempotencyKey: "port-worker-a-18084",
		Operation: SystemUpdateOperationPortReconfigure, PortReconfigure: portPlan,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "system-update-job.schema.json"), string(portJob), true)

	report, err := json.Marshal(UpdateAgentReportRequest{
		ServiceID: "host-agent-a", LeaseToken: strings.Repeat("l", 43), LeaseGeneration: 1,
		Sequence: 1, Status: SystemUpdateSucceeded,
		PortReconfigure: &SystemUpdatePortReconfiguration{Result: SystemUpdatePortReconfigurationApplied},
	})
	if err != nil {
		t.Fatal(err)
	}
	validatePortContractJSON(t, compileContractJSONSchema(t, "update-agent-report-request.schema.json"), string(report), true)
}

func TestPortPolicyProjectionAndAppliedConfigFieldsAreAdditive(t *testing.T) {
	policySchema := compileContractJSONSchema(
		t,
		"host-agent-policy-response.schema.json",
		"host-agent-self-update-directive.schema.json",
		"host-self-update-release-binding.schema.json",
	)
	validatePortContractJSON(t, policySchema, `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":0,
		"revision":17,
		"source_policy_revision":23,
		"local_executor_policy_revision":5,
		"local_executor_policy_sha256":"`+portContractExecutorDigest+`",
		"observe_only":true,
		"targets":[{
			"service_id":"worker-a",
			"service_type":"worker",
			"deployment_mode":"systemd",
			"applied_config_revision":11,
			"applied_config_sha256":"`+portContractConfigDigest+`"
		}]
	}`, true)
	validatePortContractJSON(t, policySchema, `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":0,
		"revision":1,
		"observe_only":false,
		"targets":[]
	}`, false)
	validatePortContractJSON(t, policySchema, `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":1,
		"revision":2,
		"source_policy_revision":1,
		"local_executor_policy_revision":1,
		"local_executor_policy_sha256":"`+portContractExecutorDigest+`",
		"observe_only":false,
		"targets":[]
	}`, true)
	validatePortContractJSON(t, policySchema, `{
		"service_id":"host-agent-a",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":1,
		"revision":2,
		"source_policy_revision":1,
		"local_executor_policy_revision":1,
		"local_executor_policy_sha256":"`+portContractExecutorDigest+`",
		"observe_only":true,
		"targets":[]
	}`, false)

	registeredSchema := compileContractJSONSchema(t, "registered-service.schema.json", "encoder-output-relay-capabilities.schema.json")
	validatePortContractJSON(t, registeredSchema, `{
		"service_id":"worker-a",
		"service_type":"worker",
		"service_name":"Worker A",
		"ssl_enabled":false,
		"public_url":"http://worker.example.com:8084",
		"applied_config_revision":11,
		"applied_config_sha256":"`+portContractConfigDigest+`",
		"version":"v1.2.3",
		"status":"online",
		"capabilities":{},
		"created_at":"2026-07-28T00:00:00Z",
		"updated_at":"2026-07-28T00:00:00Z"
	}`, true)
	validatePortContractJSON(t, registeredSchema, `{
		"service_id":"host-agent-a",
		"service_type":"update_agent",
		"service_name":"Host Agent A",
		"transport_mode":"pull_v2",
		"execution_host_id":"host-a",
		"ownership_epoch":0,
		"ssl_enabled":false,
		"version":"v2.0.0",
		"status":"pending",
		"capabilities":{},
		"created_at":"2026-07-28T00:00:00Z",
		"updated_at":"2026-07-28T00:00:00Z"
	}`, true)

	registrationSchema := compileContractJSONSchema(t, "service-registration.schema.json", "encoder-output-relay-capabilities.schema.json")
	validatePortContractJSON(t, registrationSchema, `{
		"service_id":"host-agent-a",
		"service_type":"update_agent",
		"service_name":"Host Agent A",
		"transport_mode":"pull_v2",
		"ownership_epoch":0,
		"version":"v2.0.0",
		"capabilities":{}
	}`, false)

	openAPI, err := os.ReadFile(filepath.Join("..", "..", "openapi", "control-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"SystemUpdatePortReconfiguration:",
		"SystemUpdatePortMutationGrantBinding:",
		"port_reconfigure:",
		"source_policy_revision:",
		"local_executor_policy_revision:",
		"local_executor_policy_sha256:",
		"applied_config_revision:",
		"applied_config_sha256:",
		"const: host",
		"const: tcp",
		"port_reconfigure_reconcile",
		"For port operations it must equal port_reconfigure.port_plan_sha256",
	} {
		if !strings.Contains(string(openAPI), marker) {
			t.Fatalf("control OpenAPI is missing port-reconfiguration marker %q", marker)
		}
	}
	requireControlOpenAPISystemUpdateTargetProjection(t)
}
