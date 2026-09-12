    V2ContractError:
      type: object
      additionalProperties: false
      required: [ contract_major, code, message, retryable ]
      properties:
        contract_major: { type: integer, const: 2 }
        code:
          type: string
          enum:
            - contract_major_unsupported
            - protocol_version_unsupported
            - request_schema_invalid
            - revision_conflict
            - stale_generation
            - stale_fence
            - semantic_validation_failed
        message: { type: string, minLength: 1, maxLength: 240 }
        retryable: { type: boolean }
        expected_major: { type: integer, const: 2 }
        audit_correlation_id:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
      description: Stable v2 error envelope. It never includes a raw payload, credential, token, hash input, command output, or configuration value.
    V2CapabilityNegotiation:
      type: object
      additionalProperties: false
      required: [ protocol_version, contract_major, capabilities, unknown_capabilities, readiness, revision ]
      properties:
        protocol_version: { type: integer, const: 2 }
        contract_major: { type: integer, const: 2 }
        capabilities:
          type: array
          minItems: 1
          maxItems: 64
          items:
            type: object
            additionalProperties: false
            required: [ name, required, supported ]
            properties:
              name: { type: string, minLength: 1, maxLength: 64, pattern: '^[a-z][a-z0-9_.-]*$' }
              required: { type: boolean, const: true }
              supported: { type: boolean }
        unknown_capabilities:
          type: array
          maxItems: 64
          uniqueItems: true
          items: { type: string, minLength: 1, maxLength: 64, pattern: '^[a-z][a-z0-9_.-]*$' }
        readiness: { type: string, enum: [ ready, not_ready, unknown ] }
        revision: { type: integer, format: int64, minimum: 1 }
        safe_error: { $ref: '#/components/schemas/V2ContractError' }
      allOf:
        - if:
            required: [ readiness ]
            properties: { readiness: { const: ready } }
          then:
            properties:
              unknown_capabilities: { maxItems: 0 }
              capabilities:
                items:
                  required: [ supported ]
                  properties: { supported: { const: true } }
        - if:
            required: [ readiness ]
            properties: { readiness: { enum: [ not_ready, unknown ] } }
          then:
            required: [ safe_error ]
      description: Each required capability carries its own support decision. Ready is mechanically impossible when any required capability is unsupported or unknown.
    WorkerApplicationRuntimeIdentityProbe:
      type: object
      additionalProperties: false
      required: [ version, service_id, service_type, config_revision ]
      properties:
        version:
          type: string
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        service_id:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        service_type: { type: string, const: worker }
        config_revision: { type: integer, format: int64, minimum: 1 }
      description: Application-owned Worker probe consumed by Local Executor verification. This is independent from Updater health.
    WorkerApplicationRuntimeIdentityProbeSafeError:
      type: object
      additionalProperties: false
      required: [ code, message ]
      properties:
        code: { type: string, enum: [ application_probe_unavailable, application_probe_identity_invalid, application_probe_revision_unknown ] }
        message: { type: string, minLength: 1, maxLength: 240 }
      description: Bounded safe Worker probe error with no raw configuration or Updater health payload.
