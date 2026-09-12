    RemediationExecutorIdentity:
      type: object
      additionalProperties: false
      required: [ authority, execution_scope ]
      properties:
        service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        authority: { type: string, enum: [ updater, target_application ] }
        execution_scope: { type: string, enum: [ host_system, application_local ] }
      allOf:
        - if:
            required: [ authority ]
            properties: { authority: { const: updater } }
          then:
            required: [ service_id ]
            properties: { execution_scope: { const: host_system } }
        - if:
            required: [ authority ]
            properties: { authority: { const: target_application } }
          then:
            properties: { execution_scope: { const: application_local } }
            not: { required: [ service_id ] }
      description: Observability is deliberately absent. Application-local execution derives the exact executor service from target, so a contradictory second service identity is not representable.
    RemediationTargetIdentity:
      type: object
      additionalProperties: false
      required: [ service_id, service_type ]
      properties:
        service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        service_type: { type: string, enum: [ control_panel, worker, encoder_recorder, discord_bot, observability ] }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        incident_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
    RemediationEvidence:
      type: object
      additionalProperties: false
      required: [ evidence_code, observed_at, observed_revision ]
      properties:
        evidence_code:
          type: string
          enum: [ incident_detected, diagnostic_confirmed, retry_eligible, host_symptom_confirmed, execution_verified, rollback_verified, outcome_ambiguous ]
        observed_at: { type: string, format: date-time }
        observed_revision: { type: integer, format: int64, minimum: 1 }
        evidence_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
      description: Bounded secret-safe evidence. Raw logs, payloads, credentials, configuration, stdout, and stderr are prohibited.
    RemediationRequestOrigin:
      type: object
      additionalProperties: false
      required: [ origin_type, principal_id, permission ]
      properties:
        origin_type: { type: string, enum: [ operator, service ] }
        principal_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:@-]{0,127}$' }
        service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        service_type: { type: string, enum: [ control_panel, observability ] }
        permission: { type: string, const: remediation.execute }
      allOf:
        - if:
            required: [ origin_type ]
            properties: { origin_type: { const: service } }
          then:
            required: [ service_id, service_type ]
          else:
            not:
              anyOf:
                - required: [ service_id ]
                - required: [ service_type ]
      description: Authenticated operator or service origin verified by Control Panel. Request origin never implies authorization.
    RemediationGrant:
      type: object
      additionalProperties: false
      required: [ authorization_id, authorization_nonce_id, proposal_id, request_origin, executor, target, action_type, idempotency_key, canonical_argument_digest, desired_revision, fence, expires_at, capability, one_time, audit_correlation_id ]
      properties:
        authorization_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        authorization_nonce_id: { type: string, minLength: 16, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$' }
        proposal_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        request_origin: { $ref: '#/components/schemas/RemediationRequestOrigin' }
        executor: { $ref: '#/components/schemas/RemediationExecutorIdentity' }
        target: { $ref: '#/components/schemas/RemediationTargetIdentity' }
        action_type:
          type: string
          enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update, application.retry_gdrive_upload, application.retry_package_remux ]
        idempotency_key: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        canonical_argument_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        fence: { type: integer, format: int64, minimum: 1 }
        expires_at: { type: string, format: date-time }
        capability:
          type: string
          enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update, application.retry_gdrive_upload, application.retry_package_remux ]
        one_time: { type: boolean, const: true }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
      allOf:
        - if:
            required: [ action_type ]
            properties:
              action_type: { enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update ] }
          then:
            properties:
              executor:
                properties: { authority: { const: updater }, execution_scope: { const: host_system } }
              target:
                required: [ host_id ]
        - if:
            required: [ action_type ]
            properties:
              action_type: { enum: [ application.retry_gdrive_upload, application.retry_package_remux ] }
          then:
            properties:
              executor:
                properties: { authority: { const: target_application }, execution_scope: { const: application_local } }
              target:
                properties: { service_type: { const: encoder_recorder } }
        - if: { properties: { action_type: { const: host.systemd } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.systemd } } }
        - if: { properties: { action_type: { const: host.docker } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.docker } } }
        - if: { properties: { action_type: { const: host.update } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.update } } }
        - if: { properties: { action_type: { const: host.bootstrap } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.bootstrap } } }
        - if: { properties: { action_type: { const: host.port } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.port } } }
        - if: { properties: { action_type: { const: host.self_update } }, required: [ action_type ] }
          then: { properties: { capability: { const: host.self_update } } }
        - if: { properties: { action_type: { const: application.retry_gdrive_upload } }, required: [ action_type ] }
          then: { properties: { capability: { const: application.retry_gdrive_upload } } }
        - if: { properties: { action_type: { const: application.retry_package_remux } }, required: [ action_type ] }
          then: { properties: { capability: { const: application.retry_package_remux } } }
      description: Control Panel-issued bounded grant. It cannot encode arbitrary shell, argv, environment, credentials, or an Observability executor.
    RemediationResultEvidence:
      type: object
      additionalProperties: false
      required: [ authorization_id, proposal_id, executor, target, action_type, idempotency_key, canonical_argument_digest, desired_revision, fence, result, reconciliation_required, automatic_resend_allowed, evidence, audit_correlation_id, completed_at ]
      properties:
        authorization_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        proposal_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        executor: { $ref: '#/components/schemas/RemediationExecutorIdentity' }
        target: { $ref: '#/components/schemas/RemediationTargetIdentity' }
        action_type:
          type: string
          enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update, application.retry_gdrive_upload, application.retry_package_remux ]
        idempotency_key: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        canonical_argument_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
        desired_revision: { type: integer, format: int64, minimum: 1 }
        applied_revision: { type: integer, format: int64, minimum: 0 }
        fence: { type: integer, format: int64, minimum: 1 }
        result: { type: string, enum: [ succeeded, failed, rolled_back, ambiguous ] }
        reconciliation_required: { type: boolean }
        automatic_resend_allowed: { type: boolean, const: false }
        evidence:
          type: array
          minItems: 1
          maxItems: 32
          items: { $ref: '#/components/schemas/RemediationEvidence' }
        safe_error: { $ref: '#/components/schemas/RemediationSafeError' }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        completed_at: { type: string, format: date-time }
      allOf:
        - if:
            required: [ result ]
            properties: { result: { const: ambiguous } }
          then:
            properties: { reconciliation_required: { const: true } }
          else:
            properties: { reconciliation_required: { const: false } }
        - if:
            required: [ action_type ]
            properties:
              action_type: { enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update ] }
          then:
            properties:
              executor:
                properties: { authority: { const: updater }, execution_scope: { const: host_system } }
              target:
                required: [ host_id ]
        - if:
            required: [ action_type ]
            properties:
              action_type: { enum: [ application.retry_gdrive_upload, application.retry_package_remux ] }
          then:
            properties:
              executor:
                properties: { authority: { const: target_application }, execution_scope: { const: application_local } }
              target:
                properties: { service_type: { const: encoder_recorder } }
    RemediationSafeError:
      type: object
      additionalProperties: false
      required: [ code, message, retryable ]
      properties:
        code:
          type: string
          enum: [ proposal_stale, authorization_expired, authorization_replayed, request_origin_invalid, permission_denied, executor_identity_mismatch, target_identity_mismatch, action_not_allowed, capability_missing, revision_conflict, stale_fence, direct_observability_updater_forbidden, outcome_ambiguous ]
        message: { type: string, minLength: 1, maxLength: 240 }
        retryable: { type: boolean }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
      description: Secret-safe remediation failure metadata; raw evidence and execution output are excluded.
