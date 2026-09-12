    UpdaterLeaseEnvelope:
      type: object
      additionalProperties: false
      required: [ protocol_version, lease_id, lease_generation, lease_expires_at, command ]
      properties:
        protocol_version: { type: integer, const: 2 }
        lease_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        lease_generation: { type: integer, format: int64, minimum: 1 }
        lease_expires_at: { type: string, format: date-time }
        command: { $ref: '#/components/schemas/UpdaterCommandEnvelope' }
      x-autostream-behavioral-validator: contracts.ValidateUpdaterLeaseEnvelope
      description: Outbound pull lease. Updater, host, expiry and fence are read only from the nested one-time mutation grant, preventing contradictory duplicate bindings.
    UpdaterProgressEnvelope:
      type: object
      additionalProperties: false
      required: [ protocol_version, command_id, job_id, updater_id, host_id, lease_id, lease_generation, sequence, phase, progress, desired_revision, fence, audit_correlation_id, observed_at ]
      properties:
        protocol_version: { type: integer, const: 2 }
        command_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        job_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        lease_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        lease_generation: { type: integer, format: int64, minimum: 1 }
        sequence: { type: integer, format: int64, minimum: 0 }
        phase: { type: string, enum: [ accepted, journaled, preparing, executing, verifying, rolling_back, reconciling ] }
        progress: { type: integer, minimum: 0, maximum: 100 }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        fence: { type: integer, format: int64, minimum: 1 }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        observed_at: { type: string, format: date-time }
      x-autostream-behavioral-validator: contracts.ValidateUpdaterProgressEnvelope
      description: Bounded typed progress only. Raw command output and arbitrary log payloads are not representable.
    UpdaterResultEnvelope:
      type: object
      additionalProperties: false
      required:
        - protocol_version
        - command_id
        - job_id
        - updater_id
        - host_id
        - lease_id
        - lease_generation
        - idempotency_key
        - canonical_payload_digest
        - authorization_id
        - desired_revision
        - fence
        - outcome
        - status
        - automatic_resend_allowed
        - audit_correlation_id
        - evidence
      properties:
        protocol_version: { type: integer, const: 2 }
        command_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        job_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        lease_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        lease_generation: { type: integer, format: int64, minimum: 1 }
        idempotency_key: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        canonical_payload_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
        authorization_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        applied_revision: { type: integer, format: int64, minimum: 0 }
        fence: { type: integer, format: int64, minimum: 1 }
        outcome: { type: string, enum: [ succeeded, failed, rolled_back, ambiguous, canceled ] }
        status: { type: string, enum: [ reconciling, succeeded, failed, rolled_back, canceled ] }
        automatic_resend_allowed: { type: boolean, const: false }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        evidence:
          type: array
          minItems: 1
          maxItems: 32
          items: { $ref: '#/components/schemas/UpdaterEvidence' }
        safe_error: { $ref: '#/components/schemas/UpdaterSafeError' }
        port_reconfigure: { $ref: '#/components/schemas/SystemUpdatePortResultV2' }
      allOf:
        - oneOf:
            - properties: { outcome: { const: succeeded }, status: { const: succeeded } }
              required: [ applied_revision ]
              not: { required: [ safe_error ] }
            - properties: { outcome: { const: rolled_back }, status: { const: rolled_back } }
            - properties: { outcome: { const: canceled }, status: { const: canceled } }
              not: { required: [ port_reconfigure ] }
            - properties: { outcome: { const: failed }, status: { const: failed } }
              required: [ safe_error ]
            - properties:
                outcome: { const: ambiguous }
                status: { const: reconciling }
                safe_error:
                  required: [ code ]
                  properties: { code: { const: outcome_ambiguous } }
              required: [ safe_error ]
      x-autostream-behavioral-validator: contracts.ValidateUpdaterResultEnvelope
      description: Result bound to the exact lease generation. Version 2 port results preserve applied, unchanged and rolled_back with complete observed snapshot evidence. Rolled-back applied_revision is immutable R.C, while desired_revision remains T.C. rollback_failed is an unaccepted recovery attempt that retains the host hold; accepted result identity and its first observation time cannot be overwritten. Ambiguous outcomes remain reconciling and automatic resend is always false.
    UpdaterRuntimeTokenRotationCredentialClaimRequest:
      type: object
      additionalProperties: false
      required: [ expected_revision, claim_id ]
      properties:
        expected_revision: { type: integer, format: int64, minimum: 1 }
        claim_id: { type: string, minLength: 16, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$' }
      description: Secret-free one-time claim request binding. The raw replacement token exists only in the credential claim response, which is Cache-Control no-store, and is forbidden from Updater commands, progress, results, evidence, and local journal records.
    UpdaterMutationGrantBinding:
      type: object
      additionalProperties: false
      required: [ lease, operation, session_id ]
      properties:
        lease: { $ref: '#/components/schemas/UpdaterLeaseEnvelope' }
        operation:
          type: string
          enum: [ apply, reconcile, port_reconfigure, port_reconfigure_reconcile, bootstrap, bootstrap_reconcile, host_self_update_stage, host_self_update_activate, host_self_update_reconcile ]
        session_id: { type: string, minLength: 16, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$' }
      x-autostream-operation-desired-map:
        apply: software_update
        reconcile: software_update
        port_reconfigure: port_reconfigure
        port_reconfigure_reconcile: port_reconfigure
        bootstrap: bootstrap
        bootstrap_reconcile: bootstrap
        host_self_update_stage: host_self_update
        host_self_update_activate: host_self_update
        host_self_update_reconcile: host_self_update
      description: Complete credential-free Local Executor binding. The nested lease carries the exact command, authorization, target, desired revision, fence, JCS digest and desired plan. The closed operation is semantically restricted by desired_operation.
    UpdaterMutationGrantIssueRequest:
      type: object
      additionalProperties: false
      required: [ binding ]
      properties:
        binding: { $ref: '#/components/schemas/UpdaterMutationGrantBinding' }
      x-autostream-behavioral-validator: contracts.ValidateUpdaterMutationGrantIssueRequest
      description: Authenticated Updater request for one opaque Local Executor grant. Grant credentials, tokens, shell, argv, environment, path and URL are prohibited.
    UpdaterMutationGrantIssueResponse:
      $ref: '#/components/schemas/UpdateAgentMutationGrantIssueResponse'
      x-autostream-behavioral-validator: contracts.ValidateUpdaterMutationGrantIssueResponse
      description: Exact no-store secret-bearing response. The opaque token is bounded, short-lived, never logged or persisted, and its expiry may not exceed the authorizing lease or mutation authorization.
    UpdaterMutationGrantConsumeRequest:
      type: object
      additionalProperties: false
      required: [ binding ]
      properties:
        binding: { $ref: '#/components/schemas/UpdaterMutationGrantBinding' }
      x-autostream-behavioral-validator: contracts.ValidateUpdaterMutationGrantConsumeRequest
      description: Exact Local Executor consume binding. The opaque grant is supplied only as authorization proof, never in this body, argv, environment, logs or persistent journal state.
    UpdaterHeartbeat:
      type: object
      additionalProperties: false
      required: [ protocol_version, updater_id, host_id, service_id, authentication, sequence, capabilities, desired_revision, applied_revision, fence, status, observed_at ]
      properties:
        protocol_version: { type: integer, const: 2 }
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        authentication: { type: string, const: assignment_bound_rotating_service_identity }
        sequence: { type: integer, format: int64, minimum: 0 }
        capabilities:
          type: array
          minItems: 1
          uniqueItems: true
          items: { type: string, enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update ] }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        applied_revision: { type: integer, format: int64, minimum: 0 }
        fence: { type: integer, format: int64, minimum: 1 }
        status: { type: string, enum: [ ready, busy, not_ready, unknown ] }
        observed_at: { type: string, format: date-time }
        safe_error: { $ref: '#/components/schemas/UpdaterSafeError' }
        port_contract_version: { type: integer, const: 2 }
        policy_transition_version: { type: integer, const: 1 }
        port_policy_baseline: { $ref: '#/components/schemas/UpdaterPortPolicyBaseline' }
      allOf:
        - if:
            anyOf:
              - required: [ port_contract_version ]
              - required: [ policy_transition_version ]
              - required: [ port_policy_baseline ]
          then:
            required: [ port_contract_version, policy_transition_version ]
      description: Updater health and capability evidence only; it cannot satisfy an application runtime identity probe. Port baseline metadata is accepted only against the already-bound installed root policy digest, never as a new trust anchor or arbitrary root policy.
    UpdaterPortPolicyBaseline:
      $ref: '../schemas/system-update-port-v2.schema.json#/$defs/policyBaseline'
    SystemUpdatePortResultV2:
      $ref: '../schemas/system-update-port-v2.schema.json#/$defs/result'
    SystemUpdatePortSnapshotRef:
      $ref: '../schemas/system-update-port-v2.schema.json#/$defs/snapshotRef'
    UpdaterSafeError:
      type: object
      additionalProperties: false
      required: [ code, message, retryable ]
      properties:
        code:
          type: string
          enum: [ protocol_version_unsupported, issuer_authentication_failed, updater_identity_mismatch, authorization_expired, authorization_replayed, idempotency_payload_conflict, capability_missing, revision_conflict, stale_fence, command_expired, outcome_ambiguous, local_journal_unavailable, execution_failed ]
        message:
          type: string
          enum:
            - updater protocol version is unsupported
            - updater issuer authentication failed
            - updater identity does not match assignment
            - updater authorization expired
            - updater authorization was already used
            - updater idempotency payload conflicts
            - updater capability is unavailable
            - updater revision conflicts
            - updater fence is stale
            - updater command expired
            - updater outcome requires reconciliation
            - updater local journal is unavailable
            - updater execution failed
        retryable: { type: boolean }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
      description: Secret-safe error metadata. Message is a closed canonical phrase paired to code by the behavioral validator, so credentials, raw payloads, commands, paths, stdout, stderr, and environment values are not representable.
    UpdaterLocalJournalBoundary:
      type: object
      additionalProperties: false
      required: [ journal_id, updater_id, host_id, command_id, authorization_id, idempotency_key, canonical_payload_digest, desired_revision, fence, phase, recovery_state, recorded_at ]
      properties:
        journal_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        command_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        authorization_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        idempotency_key: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        canonical_payload_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        fence: { type: integer, format: int64, minimum: 1 }
        phase: { type: string, enum: [ accepted, prepared, executing, verifying, rolling_back, terminal ] }
        recovery_state: { type: string, enum: [ known, resume_required, rollback_required, ambiguous ] }
        recorded_at: { type: string, format: date-time }
      description: Local Updater durability boundary. The Control Panel database is not the host executor's journal.
