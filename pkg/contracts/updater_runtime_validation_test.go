package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestComputeUpdaterCommandCanonicalDigestUsesRFC8785Projection(t *testing.T) {
	command := testUpdaterSoftwareCommand(t)
	digest, err := ComputeUpdaterCommandCanonicalDigest(
		command.MutationAuthorization.Target,
		command.MutationAuthorization.DesiredRevision,
		command.MutationAuthorization.Fence,
		command.DesiredOperation,
	)
	if err != nil {
		t.Fatalf("ComputeUpdaterCommandCanonicalDigest: %v", err)
	}
	canonical := `{"desired_operation":{"operation":"software_update","software_update":{"expected_current_version":"v1.2.3","strategy":"when_idle","target_version":"v1.2.4"}},"desired_revision":7,"fence":9,"target":{"deployment_mode":"systemd","expected_config_revision":4,"service_id":"worker-1","service_type":"worker","target_kind":"application"}}`
	wantSum := sha256.Sum256([]byte(canonical))
	want := "sha256:" + hex.EncodeToString(wantSum[:])
	if digest != want {
		t.Fatalf("digest=%q, want RFC 8785 vector %q", digest, want)
	}
}

func TestValidateUpdaterCommandEnvelopeRejectsStrictJSONAndPresenceMutants(t *testing.T) {
	command := testUpdaterSoftwareCommand(t)
	valid := testUpdaterJSON(t, command)
	if err := ValidateUpdaterCommandEnvelope(valid); err != nil {
		t.Fatalf("valid command rejected: %v", err)
	}

	mutants := map[string][]byte{
		"duplicate":          []byte(strings.Replace(string(valid), `"command_id":"command-1"`, `"command_id":"command-1","command_id":"command-1"`, 1)),
		"unknown":            testUpdaterSetRawField(t, valid, "shell", "echo forbidden"),
		"trailing":           append(append([]byte{}, valid...), []byte(` {}`)...),
		"null":               testUpdaterSetRawField(t, valid, "desired_operation", nil),
		"missing desired":    testUpdaterDeleteRawField(t, valid, "desired_operation"),
		"missing one_time":   testUpdaterDeleteNestedRawField(t, valid, "mutation_authorization", "one_time"),
		"explicit shell":     testUpdaterSetDesiredRawField(t, valid, "shell", "echo forbidden"),
		"explicit argv":      testUpdaterSetDesiredRawField(t, valid, "argv", []any{"echo"}),
		"explicit env":       testUpdaterSetDesiredRawField(t, valid, "env", map[string]any{"A": "B"}),
		"explicit path":      testUpdaterSetDesiredRawField(t, valid, "path", "/tmp/forbidden"),
		"explicit url":       testUpdaterSetDesiredRawField(t, valid, "url", "https://example.invalid"),
		"explicit token":     testUpdaterSetDesiredRawField(t, valid, "token", "forbidden"),
		"mismatched variant": testUpdaterSetNestedRawField(t, valid, "desired_operation", "operation", "bootstrap"),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterCommandEnvelope(mutant); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}
}

func TestValidateUpdaterCommandRejectsBindingMutants(t *testing.T) {
	tests := map[string]func(*UpdaterCommandEnvelope){
		"action": func(command *UpdaterCommandEnvelope) {
			command.MutationAuthorization.ActionType = UpdaterCapabilityPort
		},
		"capability": func(command *UpdaterCommandEnvelope) {
			command.MutationAuthorization.RequiredCapability = UpdaterCapabilityPort
		},
		"target kind": func(command *UpdaterCommandEnvelope) {
			command.MutationAuthorization.Target.TargetKind = UpdaterTargetHostRuntime
		},
		"payload digest": func(command *UpdaterCommandEnvelope) {
			command.CanonicalPayloadDigest = "sha256:" + strings.Repeat("0", 64)
		},
		"argument digest": func(command *UpdaterCommandEnvelope) {
			command.MutationAuthorization.CanonicalArgumentDigest = "sha256:" + strings.Repeat("0", 64)
		},
		"one time false": func(command *UpdaterCommandEnvelope) {
			command.MutationAuthorization.OneTime = false
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			command := testUpdaterSoftwareCommand(t)
			mutate(&command)
			if err := ValidateUpdaterCommand(command); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}
}

func TestValidateUpdaterDesiredOperationVariants(t *testing.T) {
	commands := map[string]UpdaterCommandEnvelope{
		"software":    testUpdaterSoftwareCommand(t),
		"bootstrap":   testUpdaterBootstrapCommand(t),
		"port":        testUpdaterPortCommand(t),
		"self update": testUpdaterSelfUpdateCommand(t),
	}
	for name, command := range commands {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterCommandEnvelope(testUpdaterJSON(t, command)); err != nil {
				t.Fatalf("valid variant rejected: %v", err)
			}
		})
	}

	t.Run("noncanonical self-update timestamp lexical form", func(t *testing.T) {
		payload := testUpdaterJSON(t, testUpdaterSelfUpdateCommand(t))
		object := testUpdaterObject(t, payload)
		desired := object["desired_operation"].(map[string]any)
		selfUpdate := desired["host_self_update"].(map[string]any)
		release := selfUpdate["release"].(map[string]any)
		release["published_at"] = "2026-09-02T00:00:00.000Z"
		payload = testUpdaterRefreshRawDigest(t, object)
		if err := ValidateUpdaterCommandEnvelope(payload); err == nil {
			t.Fatal("noncanonical RFC3339 fractional timestamp accepted")
		}
	})

	t.Run("two variants", func(t *testing.T) {
		command := testUpdaterSoftwareCommand(t)
		command.DesiredOperation.Bootstrap = &UpdaterBootstrapDesiredOperation{ExpectedState: "absent", TargetVersion: "v1.2.4"}
		testUpdaterRefreshDigest(t, &command)
		if err := ValidateUpdaterCommandEnvelope(testUpdaterJSON(t, command)); err == nil {
			t.Fatal("two desired variants accepted")
		}
	})
	t.Run("bootstrap state", func(t *testing.T) {
		command := testUpdaterBootstrapCommand(t)
		command.DesiredOperation.Bootstrap.ExpectedState = "present"
		testUpdaterRefreshDigest(t, &command)
		if err := ValidateUpdaterCommandEnvelope(testUpdaterJSON(t, command)); err == nil {
			t.Fatal("non-absent bootstrap state accepted")
		}
	})
	t.Run("port result", func(t *testing.T) {
		command := testUpdaterPortCommand(t)
		command.DesiredOperation.PortReconfigure.Result = SystemUpdatePortReconfigurationApplied
		testUpdaterRefreshDigest(t, &command)
		if err := ValidateUpdaterCommandEnvelope(testUpdaterJSON(t, command)); err == nil {
			t.Fatal("port desired result accepted")
		}
	})
	t.Run("non-application explicit expected revision", func(t *testing.T) {
		payload := testUpdaterJSON(t, testUpdaterBootstrapCommand(t))
		payload = testUpdaterSetTargetRawField(t, payload, "expected_config_revision", float64(0))
		if err := ValidateUpdaterCommandEnvelope(payload); err == nil {
			t.Fatal("non-application expected_config_revision accepted")
		}
	})
}

func TestValidateUpdaterLeaseEnvelopeRejectsExpiryAndStrictMutants(t *testing.T) {
	now := testUpdaterTime()
	lease := testUpdaterLease(t, testUpdaterSoftwareCommand(t))
	valid := testUpdaterJSON(t, lease)
	if err := ValidateUpdaterLeaseEnvelope(now, valid); err != nil {
		t.Fatalf("valid lease rejected: %v", err)
	}

	mutants := map[string][]byte{
		"missing expiry":     testUpdaterDeleteRawField(t, valid, "lease_expires_at"),
		"missing generation": testUpdaterDeleteRawField(t, valid, "lease_generation"),
		"null command":       testUpdaterSetRawField(t, valid, "command", nil),
		"unknown":            testUpdaterSetRawField(t, valid, "token", "forbidden"),
		"duplicate":          []byte(strings.Replace(string(valid), `"lease_id":"lease-1"`, `"lease_id":"lease-1","lease_id":"lease-1"`, 1)),
		"trailing":           append(append([]byte{}, valid...), []byte(` {}`)...),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterLeaseEnvelope(now, mutant); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}

	expired := lease
	expired.LeaseExpiresAt = now
	if err := ValidateUpdaterLease(now, expired); err == nil {
		t.Fatal("expired lease accepted")
	}
	afterAuthorization := lease
	afterAuthorization.LeaseExpiresAt = lease.Command.MutationAuthorization.ExpiresAt.Add(time.Second)
	if err := ValidateUpdaterLease(now, afterAuthorization); err == nil {
		t.Fatal("lease extending past authorization accepted")
	}
}

func TestValidateUpdaterProgressEnvelopeRequiresLeaseAndZeroCounters(t *testing.T) {
	lease := testUpdaterLease(t, testUpdaterSoftwareCommand(t))
	progress := UpdaterProgressEnvelope{
		ProtocolVersion:    2,
		CommandID:          lease.Command.CommandID,
		JobID:              lease.Command.MutationAuthorization.JobID,
		UpdaterID:          lease.Command.MutationAuthorization.UpdaterID,
		HostID:             lease.Command.MutationAuthorization.HostID,
		LeaseID:            lease.LeaseID,
		LeaseGeneration:    lease.LeaseGeneration,
		Sequence:           0,
		Phase:              "accepted",
		Progress:           0,
		DesiredRevision:    lease.Command.MutationAuthorization.DesiredRevision,
		Fence:              lease.Command.MutationAuthorization.Fence,
		AuditCorrelationID: lease.Command.AuditCorrelationID,
		ObservedAt:         testUpdaterTime().Add(time.Minute),
	}
	valid := testUpdaterJSON(t, progress)
	if err := ValidateUpdaterProgressEnvelope(lease, valid); err != nil {
		t.Fatalf("valid progress rejected: %v", err)
	}
	for _, field := range []string{"lease_id", "lease_generation", "sequence", "progress"} {
		t.Run("missing "+field, func(t *testing.T) {
			if err := ValidateUpdaterProgressEnvelope(lease, testUpdaterDeleteRawField(t, valid, field)); err == nil {
				t.Fatal("required field omission accepted")
			}
		})
	}
	for name, mutant := range map[string][]byte{
		"unknown":   testUpdaterSetRawField(t, valid, "raw_log", "forbidden"),
		"null":      testUpdaterSetRawField(t, valid, "phase", nil),
		"duplicate": []byte(strings.Replace(string(valid), `"phase":"accepted"`, `"phase":"accepted","phase":"accepted"`, 1)),
		"trailing":  append(append([]byte{}, valid...), []byte(` {}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterProgressEnvelope(lease, mutant); err == nil {
				t.Fatal("strict JSON mutant accepted")
			}
		})
	}
	progress.LeaseGeneration++
	if err := ValidateUpdaterProgress(lease, progress); err == nil {
		t.Fatal("wrong lease generation accepted")
	}
}

func TestValidateUpdaterResultEnvelopeRejectsTerminalMutants(t *testing.T) {
	lease := testUpdaterLease(t, testUpdaterSoftwareCommand(t))
	result := testUpdaterSucceededResult(lease, "application_probe_verified")
	valid := testUpdaterJSON(t, result)
	if err := ValidateUpdaterResultEnvelope(lease, valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	for _, field := range []string{"lease_id", "lease_generation", "applied_revision", "automatic_resend_allowed"} {
		t.Run("missing "+field, func(t *testing.T) {
			if err := ValidateUpdaterResultEnvelope(lease, testUpdaterDeleteRawField(t, valid, field)); err == nil {
				t.Fatal("required field omission accepted")
			}
		})
	}
	for name, mutant := range map[string][]byte{
		"unknown":   testUpdaterSetRawField(t, valid, "stdout", "forbidden"),
		"null":      testUpdaterSetRawField(t, valid, "evidence", nil),
		"duplicate": []byte(strings.Replace(string(valid), `"outcome":"succeeded"`, `"outcome":"succeeded","outcome":"succeeded"`, 1)),
		"trailing":  append(append([]byte{}, valid...), []byte(` {}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterResultEnvelope(lease, mutant); err == nil {
				t.Fatal("strict JSON mutant accepted")
			}
		})
	}

	mutants := map[string]func(*UpdaterResultEnvelope){
		"wrong applied revision": func(result *UpdaterResultEnvelope) { result.AppliedRevision-- },
		"automatic resend":       func(result *UpdaterResultEnvelope) { result.AutomaticResendAllowed = true },
		"missing application evidence": func(result *UpdaterResultEnvelope) {
			result.Evidence[0].EvidenceCode = "phase_observed"
		},
	}
	for name, mutate := range mutants {
		t.Run(name, func(t *testing.T) {
			mutant := result
			mutant.Evidence = append([]UpdaterEvidence(nil), result.Evidence...)
			mutate(&mutant)
			if err := ValidateUpdaterResult(lease, mutant); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}

	rolledBack := result
	rolledBack.Outcome = UpdaterOutcomeRolledBack
	rolledBack.Status = SystemUpdateRolledBack
	rolledBack.Evidence = []UpdaterEvidence{{EvidenceCode: "phase_observed", ObservedAt: testUpdaterTime().Add(time.Minute), ObservedRevision: result.DesiredRevision}}
	if err := ValidateUpdaterResult(lease, rolledBack); err == nil {
		t.Fatal("rolled_back result without rollback evidence accepted")
	}
	ambiguous := result
	ambiguous.Outcome = UpdaterOutcomeAmbiguous
	ambiguous.Status = SystemUpdateReconciling
	ambiguous.AppliedRevision = 0
	ambiguous.Evidence = []UpdaterEvidence{{EvidenceCode: "outcome_ambiguous", ObservedAt: testUpdaterTime().Add(time.Minute), ObservedRevision: result.DesiredRevision}}
	ambiguous.SafeError = &V2UpdaterSafeError{Code: "outcome_ambiguous", Message: "updater outcome requires reconciliation", Retryable: false}
	if err := ValidateUpdaterResult(lease, ambiguous); err != nil {
		t.Fatalf("valid ambiguous result rejected: %v", err)
	}
	ambiguousPayload := testUpdaterJSON(t, ambiguous)
	if err := ValidateUpdaterResultEnvelope(lease, testUpdaterDeleteNestedRawField(t, ambiguousPayload, "safe_error", "retryable")); err == nil {
		t.Fatal("ambiguous result accepted without required false retryable field")
	}
	unsafeMessage := ambiguous
	unsafeMessage.SafeError = &V2UpdaterSafeError{
		Code:      "outcome_ambiguous",
		Message:   `C:\\private\\runtime-token.txt`,
		Retryable: false,
	}
	if err := ValidateUpdaterResult(lease, unsafeMessage); err == nil {
		t.Fatal("caller-controlled path accepted as a safe error message")
	}
	mismatchedMessage := ambiguous
	mismatchedMessage.SafeError = &V2UpdaterSafeError{
		Code:      "execution_failed",
		Message:   "updater outcome requires reconciliation",
		Retryable: false,
	}
	if err := ValidateUpdaterResult(lease, mismatchedMessage); err == nil {
		t.Fatal("safe message accepted for the wrong stable error code")
	}
	ambiguous.AutomaticResendAllowed = true
	if err := ValidateUpdaterResult(lease, ambiguous); err == nil {
		t.Fatal("automatic ambiguous resend accepted")
	}
}

func TestValidateUpdaterSelfUpdateSuccessRequiresHostRuntimeEvidence(t *testing.T) {
	lease := testUpdaterLease(t, testUpdaterSelfUpdateCommand(t))
	result := testUpdaterSucceededResult(lease, "host_runtime_verified")
	if err := ValidateUpdaterResult(lease, result); err != nil {
		t.Fatalf("valid self-update result rejected: %v", err)
	}
	result.Evidence[0].EvidenceCode = "phase_observed"
	if err := ValidateUpdaterResult(lease, result); err == nil {
		t.Fatal("self-update success without host_runtime_verified accepted")
	}
}

func TestUpdaterRuntimeTokenRotationClaimRequestIsSecretFree(t *testing.T) {
	payload := string(testUpdaterJSON(t, UpdaterRuntimeTokenRotationCredentialClaimRequest{
		ExpectedRevision: 8,
		ClaimID:          "claim-runtime-0001",
	}))
	if payload != `{"expected_revision":8,"claim_id":"claim-runtime-0001"}` {
		t.Fatalf("claim request wire shape=%s", payload)
	}
	for _, forbidden := range []string{"runtime_token", "replacement_token", "token_value", "secret"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("claim request exposed forbidden field %q", forbidden)
		}
	}
}

func TestValidateUpdaterMutationGrantBindsLeasePlanAndClosedOperation(t *testing.T) {
	now := testUpdaterTime()
	binding := UpdaterMutationGrantBinding{
		Lease:     testUpdaterLease(t, testUpdaterSoftwareCommand(t)),
		Operation: UpdaterMutationApply,
		SessionID: "session-0000000001",
	}
	issue := testUpdaterJSON(t, UpdaterMutationGrantIssueRequest{Binding: binding})
	consume := testUpdaterJSON(t, UpdaterMutationGrantConsumeRequest{Binding: binding})
	if err := ValidateUpdaterMutationGrantIssueRequest(now, issue); err != nil {
		t.Fatalf("valid issue request rejected: %v", err)
	}
	if err := ValidateUpdaterMutationGrantConsumeRequest(now, consume); err != nil {
		t.Fatalf("valid consume request rejected: %v", err)
	}

	wrongOperation := binding
	wrongOperation.Operation = UpdaterMutationPortReconfigure
	if err := ValidateUpdaterMutationGrantBinding(now, wrongOperation); err == nil {
		t.Fatal("port mutation accepted for software desired operation")
	}
	forbiddenToken := testUpdaterSetRawField(t, issue, "grant_token", "forbidden")
	if err := ValidateUpdaterMutationGrantIssueRequest(now, forbiddenToken); err == nil {
		t.Fatal("grant token accepted in issue request body")
	}
	missingOperation := testUpdaterDeleteGrantBindingRawField(t, issue, "operation")
	if err := ValidateUpdaterMutationGrantIssueRequest(now, missingOperation); err == nil {
		t.Fatal("missing closed mutation operation accepted")
	}
	for name, candidate := range map[string]UpdaterMutationGrantBinding{
		"bootstrap": {
			Lease: testUpdaterLease(t, testUpdaterBootstrapCommand(t)), Operation: UpdaterMutationBootstrap,
			SessionID: "session-0000000001",
		},
		"port": {
			Lease: testUpdaterLease(t, testUpdaterPortCommand(t)), Operation: UpdaterMutationPortReconfigure,
			SessionID: "session-0000000001",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterMutationGrantBinding(now, candidate); err != nil {
				t.Fatalf("valid mutation binding rejected: %v", err)
			}
		})
	}

	selfBinding := UpdaterMutationGrantBinding{
		Lease:     testUpdaterLease(t, testUpdaterSelfUpdateCommand(t)),
		Operation: UpdaterMutationHostSelfUpdateStage,
		SessionID: "session-0000000001",
	}
	if err := ValidateUpdaterMutationGrantBinding(now, selfBinding); err != nil {
		t.Fatalf("valid self-update stage binding rejected: %v", err)
	}
	selfBinding.Operation = UpdaterMutationApply
	if err := ValidateUpdaterMutationGrantBinding(now, selfBinding); err == nil {
		t.Fatal("generic apply accepted for self-update desired operation")
	}
}

func TestValidateUpdaterMutationGrantIssueResponseIsExactAndLeaseBound(t *testing.T) {
	now := testUpdaterTime()
	lease := testUpdaterLease(t, testUpdaterSoftwareCommand(t))
	response := UpdaterMutationGrantIssueResponse{
		GrantToken: strings.Repeat("g", 48),
		ExpiresAt:  now.Add(2 * time.Minute),
	}
	valid := testUpdaterJSON(t, response)
	if err := ValidateUpdaterMutationGrantIssueResponse(now, valid); err != nil {
		t.Fatalf("valid grant response rejected: %v", err)
	}
	if err := ValidateUpdaterMutationGrantIssueResponseForLease(now, lease, response); err != nil {
		t.Fatalf("valid lease-bound grant response rejected: %v", err)
	}
	mutants := map[string][]byte{
		"missing token":  testUpdaterDeleteRawField(t, valid, "grant_token"),
		"missing expiry": testUpdaterDeleteRawField(t, valid, "expires_at"),
		"null token":     testUpdaterSetRawField(t, valid, "grant_token", nil),
		"unknown":        testUpdaterSetRawField(t, valid, "command", "forbidden"),
		"duplicate":      []byte(strings.Replace(string(valid), `"grant_token":`, `"grant_token":"`+strings.Repeat("g", 48)+`","grant_token":`, 1)),
		"trailing":       append(append([]byte{}, valid...), []byte(` {}`)...),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdaterMutationGrantIssueResponse(now, mutant); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}
	expired := response
	expired.ExpiresAt = now
	if err := ValidateUpdaterMutationGrantIssueResponseForLease(now, lease, expired); err == nil {
		t.Fatal("expired grant accepted")
	}
	afterLease := response
	afterLease.ExpiresAt = lease.LeaseExpiresAt.Add(time.Second)
	if err := ValidateUpdaterMutationGrantIssueResponseForLease(now, lease, afterLease); err == nil {
		t.Fatal("grant outliving lease accepted")
	}
}

func TestValidateUpdateAgentClearActiveJobResponseRejectsStrictMutants(t *testing.T) {
	valid := []byte(`{"clear_active_job_id":true}`)
	if err := ValidateUpdateAgentClearActiveJobResponse(valid); err != nil {
		t.Fatalf("valid clear response rejected: %v", err)
	}
	mutants := map[string][]byte{
		"false":     []byte(`{"clear_active_job_id":false}`),
		"missing":   []byte(`{}`),
		"null":      []byte(`{"clear_active_job_id":null}`),
		"unknown":   []byte(`{"clear_active_job_id":true,"lease_id":"forbidden"}`),
		"duplicate": []byte(`{"clear_active_job_id":true,"clear_active_job_id":true}`),
		"trailing":  []byte(`{"clear_active_job_id":true}{}`),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if err := ValidateUpdateAgentClearActiveJobResponse(mutant); err == nil {
				t.Fatal("mutant accepted")
			}
		})
	}
}

func testUpdaterSoftwareCommand(t *testing.T) UpdaterCommandEnvelope {
	t.Helper()
	command := testUpdaterCommandBase()
	command.MutationAuthorization.ActionType = UpdaterCapabilityUpdate
	command.MutationAuthorization.RequiredCapability = UpdaterCapabilityUpdate
	command.MutationAuthorization.Target = UpdaterTargetIdentity{
		TargetKind:             UpdaterTargetApplication,
		ServiceID:              "worker-1",
		ServiceType:            SystemUpdateTargetWorker,
		DeploymentMode:         SystemUpdateDeploymentSystemd,
		ExpectedConfigRevision: 4,
	}
	command.DesiredOperation = UpdaterDesiredOperation{
		Operation: UpdaterDesiredSoftwareUpdate,
		SoftwareUpdate: &UpdaterSoftwareUpdateDesiredOperation{
			ExpectedCurrentVersion: "v1.2.3",
			TargetVersion:          "v1.2.4",
			Strategy:               SystemUpdateWhenIdle,
		},
	}
	testUpdaterRefreshDigest(t, &command)
	return command
}

func testUpdaterBootstrapCommand(t *testing.T) UpdaterCommandEnvelope {
	t.Helper()
	command := testUpdaterCommandBase()
	command.MutationAuthorization.ActionType = UpdaterCapabilityBootstrap
	command.MutationAuthorization.RequiredCapability = UpdaterCapabilityBootstrap
	command.MutationAuthorization.Target = UpdaterTargetIdentity{
		TargetKind:      UpdaterTargetUpdateAgent,
		ServiceID:       "update-agent-1",
		ServiceType:     SystemUpdateTargetUpdateAgent,
		DeploymentMode:  SystemUpdateDeploymentSystemd,
		ExecutionHostID: "host-1",
	}
	command.DesiredOperation = UpdaterDesiredOperation{
		Operation: UpdaterDesiredBootstrap,
		Bootstrap: &UpdaterBootstrapDesiredOperation{
			ExpectedState: "absent",
			TargetVersion: "v1.2.4",
		},
	}
	testUpdaterRefreshDigest(t, &command)
	return command
}

func testUpdaterPortCommand(t *testing.T) UpdaterCommandEnvelope {
	t.Helper()
	command := testUpdaterCommandBase()
	command.MutationAuthorization.ActionType = UpdaterCapabilityPort
	command.MutationAuthorization.RequiredCapability = UpdaterCapabilityPort
	command.MutationAuthorization.Target = UpdaterTargetIdentity{
		TargetKind:             UpdaterTargetApplication,
		ServiceID:              "worker-1",
		ServiceType:            SystemUpdateTargetWorker,
		DeploymentMode:         SystemUpdateDeploymentSystemd,
		ExpectedConfigRevision: 4,
	}
	command.DesiredOperation = UpdaterDesiredOperation{
		Operation: UpdaterDesiredPortReconfigure,
		PortReconfigure: &SystemUpdatePortReconfiguration{
			NetworkNamespace:               "host",
			Protocol:                       SystemUpdatePortProtocolTCP,
			OldPort:                        4000,
			NewPort:                        4001,
			ExpectedEndpointRevision:       3,
			TargetEndpointRevision:         4,
			ExpectedConfigRevision:         4,
			TargetConfigRevision:           5,
			ExpectedConfigSHA256:           "sha256:" + strings.Repeat("1", 64),
			TargetConfigSHA256:             "sha256:" + strings.Repeat("2", 64),
			ExpectedSourcePolicyRevision:   6,
			ExpectedUpdaterPolicyRevision:  7,
			ExpectedExecutorPolicyRevision: 8,
			ExpectedExecutorPolicySHA256:   "sha256:" + strings.Repeat("3", 64),
			PortPlanSHA256:                 strings.Repeat("4", 64),
		},
	}
	testUpdaterRefreshDigest(t, &command)
	return command
}

func testUpdaterSelfUpdateCommand(t *testing.T) UpdaterCommandEnvelope {
	t.Helper()
	command := testUpdaterCommandBase()
	command.MutationAuthorization.ActionType = UpdaterCapabilitySelfUpdate
	command.MutationAuthorization.RequiredCapability = UpdaterCapabilitySelfUpdate
	command.MutationAuthorization.Target = UpdaterTargetIdentity{
		TargetKind:      UpdaterTargetHostRuntime,
		ServiceID:       "update-agent-1",
		ServiceType:     SystemUpdateTargetUpdateAgent,
		DeploymentMode:  SystemUpdateDeploymentSystemd,
		ExecutionHostID: "host-1",
	}
	command.DesiredOperation = UpdaterDesiredOperation{
		Operation: UpdaterDesiredHostSelfUpdate,
		HostSelfUpdate: &HostAgentSelfUpdateDirective{
			Generation:              "123e4567-e89b-42d3-a456-426614174000",
			AgentVersion:            "v1.2.4",
			ExecutorVersion:         "v1.2.4",
			Commit:                  strings.Repeat("a", 40),
			ArtifactSHA256:          "sha256:" + strings.Repeat("b", 64),
			AgentProtocolVersion:    2,
			ExecutorProtocolVersion: 2,
			MutationProtocolVersion: 2,
			RecoveryProtocolVersion: 2,
			Release: HostSelfUpdateReleaseBinding{
				Tag:                     "v1.2.4",
				Commit:                  strings.Repeat("a", 40),
				PublishedAt:             testUpdaterTime(),
				ManifestAssetID:         10,
				ManifestAssetName:       "host-agent-manifest.json",
				ManifestSHA256:          strings.Repeat("c", 64),
				ManifestChecksumAssetID: 11,
				ManifestChecksumSHA256:  strings.Repeat("d", 64),
				ArchiveAssetID:          12,
				ArchiveAssetName:        "autostream-host-agent_v1.2.4_linux_amd64.tar.gz",
				ArchiveSize:             1024,
				ArchiveSHA256:           strings.Repeat("e", 64),
				ArchiveChecksumAssetID:  13,
				ArchiveChecksumSHA256:   strings.Repeat("f", 64),
				Arch:                    "amd64",
				AgentProtocolVersion:    2,
				ExecutorProtocolVersion: 2,
				MutationProtocolVersion: 2,
				RecoveryProtocolVersion: 2,
				MinimumPanelVersion:     "v1.2.3",
			},
			StagedAt: testUpdaterTime().Add(time.Minute),
		},
	}
	testUpdaterRefreshDigest(t, &command)
	return command
}

func testUpdaterCommandBase() UpdaterCommandEnvelope {
	return UpdaterCommandEnvelope{
		ProtocolVersion: 2,
		CommandID:       "command-1",
		Issuer: UpdaterCommandIssuer{
			ServiceID:      "control-panel-1",
			ServiceType:    "control_panel",
			Authentication: "assignment_bound_rotating_service_identity",
			Permission:     "updates.authorize",
		},
		IdempotencyKey: "idempotency-1",
		MutationAuthorization: UpdaterMutationAuthorization{
			AuthorizationID: "authorization-1",
			NonceID:         "nonce-0000000001",
			JobID:           "job-1",
			UpdaterID:       "updater-1",
			HostID:          "host-1",
			DesiredRevision: 7,
			Fence:           9,
			ExpiresAt:       testUpdaterTime().Add(10 * time.Minute),
			OneTime:         true,
		},
		AuditCorrelationID: "audit-1",
	}
}

func testUpdaterLease(t *testing.T, command UpdaterCommandEnvelope) UpdaterLeaseEnvelope {
	t.Helper()
	return UpdaterLeaseEnvelope{
		ProtocolVersion: 2,
		LeaseID:         "lease-1",
		LeaseGeneration: 3,
		LeaseExpiresAt:  testUpdaterTime().Add(5 * time.Minute),
		Command:         command,
	}
}

func testUpdaterSucceededResult(lease UpdaterLeaseEnvelope, evidenceCode string) UpdaterResultEnvelope {
	return UpdaterResultEnvelope{
		ProtocolVersion:        2,
		CommandID:              lease.Command.CommandID,
		JobID:                  lease.Command.MutationAuthorization.JobID,
		UpdaterID:              lease.Command.MutationAuthorization.UpdaterID,
		HostID:                 lease.Command.MutationAuthorization.HostID,
		LeaseID:                lease.LeaseID,
		LeaseGeneration:        lease.LeaseGeneration,
		IdempotencyKey:         lease.Command.IdempotencyKey,
		CanonicalPayloadDigest: lease.Command.CanonicalPayloadDigest,
		AuthorizationID:        lease.Command.MutationAuthorization.AuthorizationID,
		DesiredRevision:        lease.Command.MutationAuthorization.DesiredRevision,
		AppliedRevision:        lease.Command.MutationAuthorization.DesiredRevision,
		Fence:                  lease.Command.MutationAuthorization.Fence,
		Outcome:                UpdaterOutcomeSucceeded,
		Status:                 SystemUpdateSucceeded,
		AutomaticResendAllowed: false,
		AuditCorrelationID:     lease.Command.AuditCorrelationID,
		Evidence: []UpdaterEvidence{{
			EvidenceCode:     evidenceCode,
			ObservedAt:       testUpdaterTime().Add(time.Minute),
			ObservedRevision: lease.Command.MutationAuthorization.DesiredRevision,
		}},
	}
}

func testUpdaterRefreshDigest(t *testing.T, command *UpdaterCommandEnvelope) {
	t.Helper()
	digest, err := ComputeUpdaterCommandCanonicalDigest(
		command.MutationAuthorization.Target,
		command.MutationAuthorization.DesiredRevision,
		command.MutationAuthorization.Fence,
		command.DesiredOperation,
	)
	if err != nil {
		t.Fatalf("compute digest fixture: %v", err)
	}
	command.CanonicalPayloadDigest = digest
	command.MutationAuthorization.CanonicalArgumentDigest = digest
}

func testUpdaterRefreshRawDigest(t *testing.T, command map[string]any) []byte {
	t.Helper()
	authorization := command["mutation_authorization"].(map[string]any)
	projection := map[string]any{
		"target":            authorization["target"],
		"desired_revision":  authorization["desired_revision"],
		"fence":             authorization["fence"],
		"desired_operation": command["desired_operation"],
	}
	var encoded bytes.Buffer
	if err := appendUpdaterJCS(&encoded, projection); err != nil {
		t.Fatalf("canonicalize raw fixture: %v", err)
	}
	digestBytes := sha256.Sum256(encoded.Bytes())
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	command["canonical_payload_digest"] = digest
	authorization["canonical_argument_digest"] = digest
	return testUpdaterJSON(t, command)
}

func testUpdaterTime() time.Time {
	return time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
}

func testUpdaterJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return payload
}

func testUpdaterObject(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return object
}

func testUpdaterSetRawField(t *testing.T, payload []byte, field string, value any) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	object[field] = value
	return testUpdaterJSON(t, object)
}

func testUpdaterDeleteRawField(t *testing.T, payload []byte, field string) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	delete(object, field)
	return testUpdaterJSON(t, object)
}

func testUpdaterDeleteNestedRawField(t *testing.T, payload []byte, parent, field string) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	nested, ok := object[parent].(map[string]any)
	if !ok {
		t.Fatalf("fixture %s is not an object", parent)
	}
	delete(nested, field)
	return testUpdaterJSON(t, object)
}

func testUpdaterSetNestedRawField(t *testing.T, payload []byte, parent, field string, value any) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	nested, ok := object[parent].(map[string]any)
	if !ok {
		t.Fatalf("fixture %s is not an object", parent)
	}
	nested[field] = value
	return testUpdaterJSON(t, object)
}

func testUpdaterSetDesiredRawField(t *testing.T, payload []byte, field string, value any) []byte {
	t.Helper()
	return testUpdaterSetNestedRawField(t, payload, "desired_operation", field, value)
}

func testUpdaterSetTargetRawField(t *testing.T, payload []byte, field string, value any) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	authorization, ok := object["mutation_authorization"].(map[string]any)
	if !ok {
		t.Fatal("fixture mutation_authorization is not an object")
	}
	target, ok := authorization["target"].(map[string]any)
	if !ok {
		t.Fatal("fixture target is not an object")
	}
	target[field] = value
	return testUpdaterJSON(t, object)
}

func testUpdaterDeleteGrantBindingRawField(t *testing.T, payload []byte, field string) []byte {
	t.Helper()
	object := testUpdaterObject(t, payload)
	binding, ok := object["binding"].(map[string]any)
	if !ok {
		t.Fatal("fixture binding is not an object")
	}
	delete(binding, field)
	return testUpdaterJSON(t, object)
}
