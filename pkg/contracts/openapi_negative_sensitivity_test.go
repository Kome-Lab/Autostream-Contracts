package contracts

import (
	"strings"
	"testing"
)

func TestV2CoreContractNegativeSensitivity(t *testing.T) {
	contractError := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/V2ContractError")
	assertV2SchemaFixture(t, contractError, map[string]any{
		"contract_major": 2, "code": "contract_major_unsupported",
		"message": "supported contract major is 2", "retryable": false, "expected_major": 2,
	}, true)
	assertV2SchemaFixture(t, contractError, map[string]any{
		"contract_major": 1, "code": "contract_major_unsupported",
		"message": "supported contract major is 2", "retryable": false, "expected_major": 2,
	}, false)

	capability := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/V2CapabilityNegotiation")
	readyCapability := map[string]any{
		"protocol_version": 2, "contract_major": 2,
		"capabilities":         []any{map[string]any{"name": "host.update", "required": true, "supported": true}},
		"unknown_capabilities": []any{}, "readiness": "ready", "revision": 3,
	}
	assertV2SchemaFixture(t, capability, readyCapability, true)
	unknownCapability := map[string]any{
		"protocol_version": 2, "contract_major": 2,
		"capabilities":         []any{map[string]any{"name": "future.capability", "required": true, "supported": false}},
		"unknown_capabilities": []any{"future.capability"}, "readiness": "not_ready", "revision": 3,
		"safe_error": map[string]any{
			"contract_major": 2, "code": "semantic_validation_failed",
			"message": "required capability is unknown", "retryable": false,
		},
	}
	assertV2SchemaFixture(t, capability, unknownCapability, true)
	unknownMarkedReady := cloneV2Fixture(t, unknownCapability)
	unknownMarkedReady["readiness"] = "ready"
	delete(unknownMarkedReady, "safe_error")
	assertV2SchemaFixture(t, capability, unknownMarkedReady, false)

	command := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterCommandEnvelope")
	validCommand := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "idempotency_key": "idem-1",
		"issuer": map[string]any{
			"service_id": "control-panel-1", "service_type": "control_panel",
			"authentication": "assignment_bound_rotating_service_identity", "permission": "updates.authorize",
		},
		"canonical_payload_digest": "sha256:" + strings.Repeat("a", 64),
		"audit_correlation_id":     "audit-1",
		"desired_operation": map[string]any{
			"operation": "software_update",
			"software_update": map[string]any{
				"expected_current_version": "v2.0.0", "target_version": "v2.0.1", "strategy": "when_idle",
			},
		},
		"mutation_authorization": map[string]any{
			"authorization_id": "authorization-1", "nonce_id": "nonce-12345678901234567890",
			"job_id": "job-1", "updater_id": "updater-1", "host_id": "host-1", "action_type": "host.update",
			"target": map[string]any{
				"target_kind": "application", "service_id": "worker-1", "service_type": "worker", "deployment_mode": "systemd",
				"expected_config_revision": 8,
			},
			"canonical_argument_digest": "sha256:" + strings.Repeat("b", 64),
			"desired_revision":          9, "fence": 4, "expires_at": "2026-08-31T01:00:00Z",
			"required_capability": "host.update", "one_time": true,
		},
	}
	assertV2SchemaFixture(t, command, validCommand, true)
	for name, mutate := range map[string]func(map[string]any){
		"unsupported_protocol": func(value map[string]any) { value["protocol_version"] = 1 },
		"arbitrary_shell":      func(value map[string]any) { value["shell"] = "powershell" },
		"shared_database":      func(value map[string]any) { value["database_credentials"] = "forbidden" },
		"missing_fence": func(value map[string]any) {
			delete(value["mutation_authorization"].(map[string]any), "fence")
		},
		"missing_identity": func(value map[string]any) {
			authorization := value["mutation_authorization"].(map[string]any)
			delete(authorization["target"].(map[string]any), "service_id")
		},
		"mismatched_capability": func(value map[string]any) {
			value["mutation_authorization"].(map[string]any)["required_capability"] = "host.port"
		},
		"transplanted_host": func(value map[string]any) {
			delete(value["mutation_authorization"].(map[string]any), "host_id")
		},
		"missing_desired_operation": func(value map[string]any) {
			delete(value, "desired_operation")
		},
		"generic_desired_token": func(value map[string]any) {
			value["desired_operation"].(map[string]any)["token"] = "forbidden"
		},
		"multiple_desired_variants": func(value map[string]any) {
			value["desired_operation"].(map[string]any)["bootstrap"] = map[string]any{
				"expected_state": "absent", "target_version": "v2.0.1",
			}
		},
	} {
		t.Run("updater_command_"+name, func(t *testing.T) {
			fixture := cloneV2Fixture(t, validCommand)
			mutate(fixture)
			assertV2SchemaFixture(t, command, fixture, false)
		})
	}

	result := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterResultEnvelope")
	validResult := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "job_id": "job-1",
		"updater_id": "updater-1", "host_id": "host-1", "lease_id": "lease-1", "lease_generation": 3, "idempotency_key": "idem-1",
		"canonical_payload_digest": "sha256:" + strings.Repeat("a", 64), "authorization_id": "authorization-1",
		"desired_revision": 9, "applied_revision": 8, "fence": 4,
		"outcome": "ambiguous", "status": "reconciling", "automatic_resend_allowed": false,
		"audit_correlation_id": "audit-1",
		"safe_error": map[string]any{
			"code": "outcome_ambiguous", "message": "updater outcome requires reconciliation", "retryable": false,
		},
		"evidence": []any{map[string]any{
			"evidence_code": "outcome_ambiguous", "observed_at": "2026-08-31T00:30:00Z", "observed_revision": 8,
		}},
	}
	assertV2SchemaFixture(t, result, validResult, true)
	resultWithCallerText := cloneV2Fixture(t, validResult)
	resultWithCallerText["safe_error"].(map[string]any)["message"] = "/var/lib/autostream/runtime-token"
	assertV2SchemaFixture(t, result, resultWithCallerText, false)
	ambiguousReportedSucceeded := cloneV2Fixture(t, validResult)
	ambiguousReportedSucceeded["status"] = "succeeded"
	assertV2SchemaFixture(t, result, ambiguousReportedSucceeded, false)
	resultWithRawOutput := cloneV2Fixture(t, validResult)
	resultWithRawOutput["evidence"].([]any)[0].(map[string]any)["stdout"] = "raw output"
	assertV2SchemaFixture(t, result, resultWithRawOutput, false)
	contradictoryTerminal := cloneV2Fixture(t, validResult)
	contradictoryTerminal["outcome"] = "succeeded"
	contradictoryTerminal["status"] = "failed"
	delete(contradictoryTerminal, "safe_error")
	assertV2SchemaFixture(t, result, contradictoryTerminal, false)
	failedWithoutSafeError := cloneV2Fixture(t, validResult)
	failedWithoutSafeError["outcome"] = "failed"
	failedWithoutSafeError["status"] = "failed"
	delete(failedWithoutSafeError, "safe_error")
	assertV2SchemaFixture(t, result, failedWithoutSafeError, false)

	lease := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterLeaseEnvelope")
	validLease := map[string]any{
		"protocol_version": 2, "lease_id": "lease-1", "lease_generation": 3,
		"lease_expires_at": "2026-08-31T00:45:00Z",
		"command":          validCommand,
	}
	assertV2SchemaFixture(t, lease, validLease, true)
	staleLease := cloneV2Fixture(t, validLease)
	staleLease["lease_generation"] = 0
	assertV2SchemaFixture(t, lease, staleLease, false)

	progress := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/UpdaterProgressEnvelope")
	validProgress := map[string]any{
		"protocol_version": 2, "command_id": "command-1", "job_id": "job-1",
		"updater_id": "updater-1", "host_id": "host-1", "lease_id": "lease-1", "lease_generation": 3, "sequence": 2,
		"phase": "executing", "progress": 50, "desired_revision": 9, "fence": 4,
		"audit_correlation_id": "audit-1", "observed_at": "2026-08-31T00:30:00Z",
	}
	assertV2SchemaFixture(t, progress, validProgress, true)
	progressWithLog := cloneV2Fixture(t, validProgress)
	progressWithLog["raw_log"] = "forbidden"
	assertV2SchemaFixture(t, progress, progressWithLog, false)

	grant := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/RemediationGrant")
	validGrant := map[string]any{
		"authorization_id": "authorization-1", "authorization_nonce_id": "nonce-12345678901234567890",
		"proposal_id": "proposal-1",
		"request_origin": map[string]any{
			"origin_type": "service", "principal_id": "observability-1", "service_id": "observability-1",
			"service_type": "observability", "permission": "remediation.execute",
		},
		"executor": map[string]any{
			"service_id": "updater-1", "authority": "updater", "execution_scope": "host_system",
		},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "idempotency_key": "idem-1",
		"canonical_argument_digest": "sha256:" + strings.Repeat("c", 64), "desired_revision": 9,
		"fence": 4, "expires_at": "2026-08-31T01:00:00Z", "capability": "host.update",
		"one_time": true, "audit_correlation_id": "audit-1",
	}
	assertV2SchemaFixture(t, grant, validGrant, true)
	observabilityExecutor := cloneV2Fixture(t, validGrant)
	observabilityExecutor["executor"].(map[string]any)["authority"] = "observability"
	assertV2SchemaFixture(t, grant, observabilityExecutor, false)
	grantWithCredential := cloneV2Fixture(t, validGrant)
	grantWithCredential["credentials"] = "forbidden"
	assertV2SchemaFixture(t, grant, grantWithCredential, false)
	wrongPermission := cloneV2Fixture(t, validGrant)
	wrongPermission["request_origin"].(map[string]any)["permission"] = "updates.authorize"
	assertV2SchemaFixture(t, grant, wrongPermission, false)
	mismatchedGrantCapability := cloneV2Fixture(t, validGrant)
	mismatchedGrantCapability["capability"] = "host.port"
	assertV2SchemaFixture(t, grant, mismatchedGrantCapability, false)
	applicationGrant := cloneV2Fixture(t, validGrant)
	applicationGrant["action_type"] = "application.retry_package_remux"
	applicationGrant["capability"] = "application.retry_package_remux"
	applicationGrant["executor"] = map[string]any{
		"authority": "target_application", "execution_scope": "application_local",
	}
	applicationGrant["target"] = map[string]any{
		"service_id": "encoder-1", "service_type": "encoder_recorder", "incident_id": "incident-1",
	}
	assertV2SchemaFixture(t, grant, applicationGrant, true)
	contradictoryApplicationExecutor := cloneV2Fixture(t, applicationGrant)
	contradictoryApplicationExecutor["executor"].(map[string]any)["service_id"] = "worker-1"
	assertV2SchemaFixture(t, grant, contradictoryApplicationExecutor, false)
	wrongApplicationTarget := cloneV2Fixture(t, applicationGrant)
	wrongApplicationTarget["target"].(map[string]any)["service_type"] = "worker"
	assertV2SchemaFixture(t, grant, wrongApplicationTarget, false)
	hostWithApplicationExecutor := cloneV2Fixture(t, validGrant)
	hostWithApplicationExecutor["executor"] = applicationGrant["executor"]
	assertV2SchemaFixture(t, grant, hostWithApplicationExecutor, false)

	remediationResult := compileNormalizedOpenAPISchema(t, "control-api.json", "/components/schemas/RemediationResultEvidence")
	validRemediationResult := map[string]any{
		"authorization_id": "authorization-1", "proposal_id": "proposal-1",
		"executor": map[string]any{
			"service_id": "updater-1", "authority": "updater", "execution_scope": "host_system",
		},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "idempotency_key": "idem-1",
		"canonical_argument_digest": "sha256:" + strings.Repeat("c", 64), "desired_revision": 9,
		"applied_revision": 8, "fence": 4, "result": "ambiguous", "reconciliation_required": true,
		"automatic_resend_allowed": false,
		"evidence": []any{map[string]any{
			"evidence_code": "outcome_ambiguous", "observed_at": "2026-08-31T00:30:00Z", "observed_revision": 8,
		}},
		"audit_correlation_id": "audit-1", "completed_at": "2026-08-31T00:31:00Z",
	}
	assertV2SchemaFixture(t, remediationResult, validRemediationResult, true)
	blindResend := cloneV2Fixture(t, validRemediationResult)
	blindResend["automatic_resend_allowed"] = true
	assertV2SchemaFixture(t, remediationResult, blindResend, false)
	falseReconciliation := cloneV2Fixture(t, validRemediationResult)
	falseReconciliation["reconciliation_required"] = false
	assertV2SchemaFixture(t, remediationResult, falseReconciliation, false)

	proposal := compileNormalizedOpenAPISchema(t, "observability-api.json", "/components/schemas/RemediationProposal")
	validProposal := map[string]any{
		"proposal_id": "proposal-1", "incident_id": "incident-1",
		"detector":    map[string]any{"service_id": "observability-1", "service_type": "observability"},
		"target":      map[string]any{"service_id": "worker-1", "service_type": "worker", "host_id": "host-1"},
		"action_type": "host.update", "proposal_revision": 2, "required_capability": "host.update",
		"evidence": []any{map[string]any{
			"evidence_code": "host_symptom_confirmed", "observed_at": "2026-08-31T00:15:00Z", "observed_revision": 2,
		}},
		"audit_correlation_id": "audit-1", "observed_at": "2026-08-31T00:15:00Z",
		"control_panel_authorization_required": true,
	}
	assertV2SchemaFixture(t, proposal, validProposal, true)
	proposalWithExecutor := cloneV2Fixture(t, validProposal)
	proposalWithExecutor["executor"] = map[string]any{"authority": "updater"}
	assertV2SchemaFixture(t, proposal, proposalWithExecutor, false)
	wrongDetector := cloneV2Fixture(t, validProposal)
	wrongDetector["detector"].(map[string]any)["service_type"] = "worker"
	assertV2SchemaFixture(t, proposal, wrongDetector, false)
	proposalCapabilityMismatch := cloneV2Fixture(t, validProposal)
	proposalCapabilityMismatch["required_capability"] = "host.port"
	assertV2SchemaFixture(t, proposal, proposalCapabilityMismatch, false)

	localTransition := compileNormalizedOpenAPISchema(t, "observability-api.json", "/components/schemas/ObservabilityLocalRemediationTransition")
	validLocalTransition := map[string]any{
		"action_id": "action-1", "action_type": "rerun_diagnostics", "expected_revision": 2,
		"fence": 2, "idempotency_key": "idem-1", "audit_correlation_id": "audit-1",
	}
	assertV2SchemaFixture(t, localTransition, validLocalTransition, true)
	hostActionInObservability := cloneV2Fixture(t, validLocalTransition)
	hostActionInObservability["action_type"] = "host.update"
	assertV2SchemaFixture(t, localTransition, hostActionInObservability, false)

	probeCases := []struct {
		bundle      string
		serviceType string
	}{
		{bundle: "control-api.json", serviceType: "control_panel"},
		{bundle: "discord-bot-api.json", serviceType: "discord_bot"},
		{bundle: "encoder-recorder-api.json", serviceType: "encoder_recorder"},
		{bundle: "observability-api.json", serviceType: "observability"},
	}
	for _, test := range probeCases {
		t.Run("application_probe_"+test.serviceType, func(t *testing.T) {
			probe := compileNormalizedOpenAPISchema(t, test.bundle,
				"/paths/~1updater~1version/get/responses/200/content/application~1json/schema")
			validProbe := map[string]any{
				"version": "v2.0.0", "service_id": test.serviceType + "-1",
				"service_type": test.serviceType, "config_revision": 7,
			}
			assertV2SchemaFixture(t, probe, validProbe, true)
			missingRevision := cloneV2Fixture(t, validProbe)
			delete(missingRevision, "config_revision")
			assertV2SchemaFixture(t, probe, missingRevision, false)
			wrongIdentity := cloneV2Fixture(t, validProbe)
			wrongIdentity["service_type"] = "updater"
			assertV2SchemaFixture(t, probe, wrongIdentity, false)
			updaterHealthSubstitution := cloneV2Fixture(t, validProbe)
			delete(updaterHealthSubstitution, "service_id")
			updaterHealthSubstitution["updater_health"] = map[string]any{"status": "ready"}
			assertV2SchemaFixture(t, probe, updaterHealthSubstitution, false)
		})
	}
	workerProbe := compileNormalizedOpenAPISchema(t, "control-api.json",
		"/components/pathItems/WorkerApplicationRuntimeIdentityProbe/get/responses/200/content/application~1json/schema")
	validWorkerProbe := map[string]any{
		"version": "v2.0.0", "service_id": "worker-1", "service_type": "worker", "config_revision": 7,
	}
	assertV2SchemaFixture(t, workerProbe, validWorkerProbe, true)
	workerHealthSubstitution := cloneV2Fixture(t, validWorkerProbe)
	delete(workerHealthSubstitution, "service_id")
	workerHealthSubstitution["updater_health"] = map[string]any{"status": "ready"}
	assertV2SchemaFixture(t, workerProbe, workerHealthSubstitution, false)
}
