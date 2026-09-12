    UpdaterTargetIdentity:
      type: object
      additionalProperties: false
      required: [ target_kind, service_id, service_type, deployment_mode ]
      properties:
        target_kind: { type: string, enum: [ application, update_agent, host_runtime ] }
        service_id:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        service_type:
          type: string
          enum: [ control_panel, worker, encoder_recorder, discord_bot, observability, update_agent ]
        deployment_mode: { type: string, enum: [ systemd, docker ] }
        expected_config_revision: { type: integer, format: int64, minimum: 1 }
        execution_host_id:
          type: string
          minLength: 1
          maxLength: 191
          pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$'
      oneOf:
        - required: [ expected_config_revision ]
          properties:
            target_kind: { const: application }
            service_type: { enum: [ control_panel, worker, encoder_recorder, discord_bot, observability ] }
          not: { required: [ execution_host_id ] }
        - required: [ execution_host_id ]
          properties:
            target_kind: { const: update_agent }
            service_type: { const: update_agent }
            deployment_mode: { const: systemd }
          not: { required: [ expected_config_revision ] }
        - required: [ execution_host_id ]
          properties:
            target_kind: { const: host_runtime }
            service_type: { const: update_agent }
            deployment_mode: { const: systemd }
          not: { required: [ expected_config_revision ] }
      description: Strict identity union. Application targets carry expected_config_revision; update_agent and host_runtime targets carry execution_host_id and forbid expected_config_revision.
    UpdaterEvidence:
      type: object
      additionalProperties: false
      required: [ evidence_code, observed_at, observed_revision ]
      properties:
        evidence_code:
          type: string
          enum: [ command_accepted, phase_observed, application_probe_verified, host_runtime_verified, rollback_verified, outcome_ambiguous ]
        observed_at: { type: string, format: date-time }
        observed_revision: { type: integer, format: int64, minimum: 1 }
        artifact_digest: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
      description: Bounded evidence only. Raw stdout, stderr, environment, paths, payloads, credentials, and configuration are prohibited.
    UpdaterCommandIssuer:
      type: object
      additionalProperties: false
      required: [ service_id, service_type, authentication, permission ]
      properties:
        service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        service_type: { type: string, const: control_panel }
        authentication: { type: string, const: assignment_bound_rotating_service_identity }
        permission: { type: string, const: updates.authorize }
      description: Authenticated Control Panel command issuer. No credential value is carried in the payload.
    UpdaterMutationAuthorization:
      type: object
      additionalProperties: false
      required: [ authorization_id, nonce_id, job_id, updater_id, host_id, action_type, target, canonical_argument_digest, desired_revision, fence, expires_at, required_capability, one_time ]
      properties:
        authorization_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        nonce_id: { type: string, minLength: 16, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$' }
        job_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        action_type:
          type: string
          enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update ]
        target: { $ref: '#/components/schemas/UpdaterTargetIdentity' }
        canonical_argument_digest:
          type: string
          pattern: '^sha256:[a-f0-9]{64}$'
          description: RFC 8785/JCS SHA-256 digest of exactly target, desired_revision, fence, and desired_operation; identical to canonical_payload_digest.
        desired_revision: { type: integer, format: int64, minimum: 1 }
        fence: { type: integer, format: int64, minimum: 1 }
        expires_at: { type: string, format: date-time }
        required_capability:
          type: string
          enum: [ host.systemd, host.docker, host.update, host.bootstrap, host.port, host.self_update ]
        one_time: { type: boolean, const: true }
      allOf:
        - if: { properties: { action_type: { const: host.systemd } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.systemd } } }
        - if: { properties: { action_type: { const: host.docker } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.docker } } }
        - if: { properties: { action_type: { const: host.update } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.update } } }
        - if: { properties: { action_type: { const: host.bootstrap } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.bootstrap } } }
        - if: { properties: { action_type: { const: host.port } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.port } } }
        - if: { properties: { action_type: { const: host.self_update } }, required: [ action_type ] }
          then: { properties: { required_capability: { const: host.self_update } } }
      description: Complete short-lived one-time mutation grant claim. Job, updater, host, action, target, digest, revision, fence, expiry, nonce, and capability are bound in this single object; the bearer credential itself is never represented here.
    UpdaterDesiredOperation:
      $ref: '../schemas/updater-desired-operation.schema.json'
    UpdaterCommandEnvelope:
      type: object
      additionalProperties: false
      required:
        - protocol_version
        - command_id
        - issuer
        - idempotency_key
        - canonical_payload_digest
        - mutation_authorization
        - desired_operation
        - audit_correlation_id
      properties:
        protocol_version: { type: integer, const: 2 }
        command_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        issuer: { $ref: '#/components/schemas/UpdaterCommandIssuer' }
        idempotency_key: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        canonical_payload_digest:
          type: string
          pattern: '^sha256:[a-f0-9]{64}$'
          description: RFC 8785/JCS SHA-256 digest of exactly target, desired_revision, fence, and desired_operation; identical to mutation_authorization.canonical_argument_digest.
        mutation_authorization: { $ref: '#/components/schemas/UpdaterMutationAuthorization' }
        desired_operation: { $ref: '#/components/schemas/UpdaterDesiredOperation' }
        audit_correlation_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
      allOf:
        - if:
            required: [ desired_operation ]
            properties: { desired_operation: { properties: { operation: { const: software_update } } } }
          then:
            properties:
              mutation_authorization:
                properties:
                  action_type: { const: host.update }
                  required_capability: { const: host.update }
                  target: { properties: { target_kind: { const: application } } }
        - if:
            required: [ desired_operation ]
            properties: { desired_operation: { properties: { operation: { const: bootstrap } } } }
          then:
            properties:
              mutation_authorization:
                properties:
                  action_type: { const: host.bootstrap }
                  required_capability: { const: host.bootstrap }
                  target: { properties: { target_kind: { const: update_agent } } }
        - if:
            required: [ desired_operation ]
            properties: { desired_operation: { properties: { operation: { const: port_reconfigure } } } }
          then:
            properties:
              mutation_authorization:
                properties:
                  action_type: { const: host.port }
                  required_capability: { const: host.port }
                  target: { properties: { target_kind: { const: application } } }
        - if:
            required: [ desired_operation ]
            properties: { desired_operation: { properties: { operation: { const: host_self_update } } } }
          then:
            properties:
              mutation_authorization:
                properties:
                  action_type: { const: host.self_update }
                  required_capability: { const: host.self_update }
                  target: { properties: { target_kind: { const: host_runtime } } }
      x-autostream-behavioral-validator: contracts.ValidateUpdaterCommandEnvelope
      description: Typed host-execution intent whose privileged identity and fence are carried exactly once by mutation_authorization. The desired operation is a closed exactly-one union. Arbitrary shell, argv, environment, path, URL, token, shared Control Panel database credentials, and unbounded command output are not representable.
