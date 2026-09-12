    SystemUpdatePullOwnershipActivateRequest:
      $ref: ../schemas/system-update-pull-ownership-activate-request.schema.json
    SystemUpdatePullOwnershipActivateResponse:
      $ref: ../schemas/system-update-pull-ownership-activate-response.schema.json
    SystemUpdatePullOwnershipDeactivateRequest:
      $ref: ../schemas/system-update-pull-ownership-deactivate-request.schema.json
    SystemUpdatePullOwnershipDeactivateResponse:
      $ref: ../schemas/system-update-pull-ownership-deactivate-response.schema.json
    HostSelfUpdateCreateRequest:
      type: object
      additionalProperties: false
      required: [ target_version, idempotency_key ]
      description: The host binding comes only from the path. Release commit, artifact, digest, location, and credentials are resolved by the server.
      properties:
        target_version:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        idempotency_key:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[^\u0000-\u001F\u007F]+$'
    HostSelfUpdateRetryRequest:
      type: object
      additionalProperties: false
      required: [ idempotency_key ]
      properties:
        idempotency_key:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[^\u0000-\u001F\u007F]+$'
    HostSelfUpdateCancelRequest:
      type: object
      additionalProperties: false
      required: [ expected_revision ]
      properties:
        expected_revision:
          type: integer
          format: int64
          minimum: 1
    HostSelfUpdateRelease:
      type: object
      additionalProperties: false
      description: Verified immutable public release identity. No download location or credential is persisted or returned.
      required:
        - tag
        - commit
        - published_at
        - manifest_asset_id
        - manifest_asset_name
        - manifest_sha256
        - manifest_checksum_asset_id
        - manifest_checksum_sha256
        - archive_asset_id
        - archive_asset_name
        - archive_size
        - archive_sha256
        - archive_checksum_asset_id
        - archive_checksum_sha256
        - arch
        - agent_protocol_version
        - executor_protocol_version
        - mutation_protocol_version
        - recovery_protocol_version
        - minimum_panel_version
        - attestation_verified_at
      properties:
        tag:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        commit: { type: string, pattern: '^[a-f0-9]{40}$' }
        published_at:
          type: string
          format: date-time
          pattern: '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.[0-9]{0,8}[1-9])?Z$'
        manifest_asset_id: { type: integer, format: int64, minimum: 1 }
        manifest_asset_name:
          type: string
          minLength: 1
          maxLength: 255
          pattern: '^[A-Za-z0-9][A-Za-z0-9._+-]{0,254}$'
        manifest_sha256: { type: string, pattern: '^[a-f0-9]{64}$' }
        manifest_checksum_asset_id: { type: integer, format: int64, minimum: 1 }
        manifest_checksum_sha256: { type: string, pattern: '^[a-f0-9]{64}$' }
        archive_asset_id: { type: integer, format: int64, minimum: 1 }
        archive_asset_name:
          type: string
          minLength: 1
          maxLength: 255
          pattern: '^[A-Za-z0-9][A-Za-z0-9._+-]{0,254}$'
        archive_size: { type: integer, format: int64, minimum: 1 }
        archive_sha256: { type: string, pattern: '^[a-f0-9]{64}$' }
        archive_checksum_asset_id: { type: integer, format: int64, minimum: 1 }
        archive_checksum_sha256: { type: string, pattern: '^[a-f0-9]{64}$' }
        arch: { enum: [ amd64, arm64 ] }
        agent_protocol_version: { type: integer, minimum: 1 }
        executor_protocol_version: { type: integer, minimum: 1 }
        mutation_protocol_version: { type: integer, minimum: 1 }
        recovery_protocol_version: { type: integer, minimum: 1 }
        minimum_panel_version:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        attestation_verified_at: { type: string, format: date-time }
    HostSelfUpdate:
      type: object
      additionalProperties: false
      description: Dedicated lifecycle; it is never represented as a generic update job.
      required:
        - id
        - execution_host_id
        - agent_service_id
        - target_version
        - status
        - revision
        - idempotency_key
        - attempt_generation
        - expected_ownership_epoch
        - expected_source_policy_revision
        - expected_projection_revision
        - expected_local_executor_policy_revision
        - expected_local_executor_policy_sha256
        - previous_agent_version
        - previous_executor_version
        - previous_agent_protocol_version
        - previous_executor_protocol_version
        - previous_mutation_protocol_version
        - previous_recovery_protocol_version
        - release
        - issued_at
        - observation_state
        - created_at
        - updated_at
      properties:
        id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        execution_host_id: { type: string, minLength: 1, maxLength: 191, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$' }
        agent_service_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        target_version:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        status:
          enum: [ queued, staging, activating, verifying, rolling_back, cancel_requested, succeeded, rolled_back, failed, canceled ]
        revision: { type: integer, format: int64, minimum: 1 }
        idempotency_key:
          type: string
          minLength: 1
          maxLength: 128
          pattern: '^[^\u0000-\u001F\u007F]+$'
        requested_by:
          type: string
          minLength: 1
          maxLength: 255
          pattern: '^[^\u0000-\u001F\u007F]+$'
        retry_of_id: { type: string, minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' }
        attempt_generation: { type: string, pattern: '^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$' }
        expected_ownership_epoch: { type: integer, format: int64, minimum: 1 }
        expected_source_policy_revision: { type: integer, format: int64, minimum: 1 }
        expected_projection_revision: { type: integer, format: int64, minimum: 1 }
        expected_local_executor_policy_revision: { type: integer, format: int64, minimum: 1 }
        expected_local_executor_policy_sha256: { type: string, pattern: '^sha256:[a-f0-9]{64}$' }
        previous_agent_version: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        previous_executor_version: { type: string, minLength: 1, maxLength: 128, pattern: '^[^\u0000-\u001F\u007F]+$' }
        previous_agent_protocol_version: { type: integer, minimum: 1 }
        previous_executor_protocol_version: { type: integer, minimum: 1 }
        previous_mutation_protocol_version: { type: integer, minimum: 1 }
        previous_recovery_protocol_version: { type: integer, minimum: 1 }
        release: { $ref: "#/components/schemas/HostSelfUpdateRelease" }
        issued_at: { type: string, format: date-time }
        observation_state: { enum: [ known, stalled, unknown ] }
        reported_phase: { type: string, minLength: 1, maxLength: 64, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$' }
        last_heartbeat_at: { type: string, format: date-time }
        stalled_since: { type: string, format: date-time }
        cancel_requested_at: { type: string, format: date-time }
        started_at: { type: string, format: date-time }
        completed_at: { type: string, format: date-time }
        code: { type: string, minLength: 1, maxLength: 64, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$' }
        message: { type: string, minLength: 1, maxLength: 255, pattern: '^[^\u0000-\u001F\u007F]+$' }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    HostSelfUpdateListResponse:
      type: object
      additionalProperties: false
      required: [ self_updates ]
      properties:
        self_updates:
          type: array
          maxItems: 500
          items:
            $ref: "#/components/schemas/HostSelfUpdate"
    HostSelfUpdateGrant:
      $ref: "../schemas/host-self-update-grant.schema.json"
    HostSelfUpdateGrantIssueRequest:
      type: object
      additionalProperties: false
      required: [ expected_revision, operation, plan_sha256, session_id ]
      properties:
        expected_revision: { type: integer, format: int64, minimum: 1 }
        operation: { enum: [ stage, reconcile ] }
        plan_sha256: { type: string, pattern: '^[a-f0-9]{64}$' }
        session_id: { type: string, minLength: 16, maxLength: 128, pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$' }
    HostSelfUpdateGrantIssueResponse:
      $ref: "../schemas/host-self-update-grant-issue-response.schema.json"
    HostSelfUpdateGrantConsumeRequest:
      $ref: "../schemas/host-self-update-grant-consume-request.schema.json"
    HostSelfUpdateGrantConsumeResponse:
      $ref: "../schemas/host-self-update-grant-consume-response.schema.json"
