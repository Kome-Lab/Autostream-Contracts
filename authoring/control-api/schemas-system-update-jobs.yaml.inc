    SystemUpdateCreateRequest:
      $ref: ../schemas/system-update-create-request.schema.json
    SystemUpdatePortMapping:
      type: object
      additionalProperties: false
      required: [ mode, state ]
      allOf:
        - if:
            required: [ state ]
            properties:
              state: { const: applied }
          then:
            required:
              - advertised_port
              - published_port
              - container_port
              - health_port
              - config_revision
              - published_host_ip
              - reported_at
      properties:
        mode: { type: string, const: docker }
        state:
          type: string
          enum: [ applied, drifted, unavailable ]
        advertised_port: { type: integer, minimum: 1, maximum: 65535 }
        published_port: { type: integer, minimum: 1024, maximum: 65535 }
        container_port: { type: integer, minimum: 1024, maximum: 65535 }
        health_port: { type: integer, minimum: 1024, maximum: 65535 }
        config_revision: { type: integer, format: int64, minimum: 1 }
        published_host_ip: { type: string, const: 127.0.0.1 }
        reported_at: { type: string, format: date-time }
    SystemUpdateTarget:
      $ref: ../schemas/system-update-target.schema.json
    SystemUpdateDockerPortReconfiguration:
      type: object
      additionalProperties: false
      required:
        - published_host_ip
        - old_published_port
        - new_published_port
        - old_container_port
        - new_container_port
        - old_health_port
        - new_health_port
        - approved_compose_config_sha256
        - approved_compose_revision
        - expected_version_env_sha256
        - expected_container_id
        - expected_image_id
        - expected_repository_digest
      properties:
        published_host_ip: { type: string, const: 127.0.0.1 }
        old_published_port: { type: integer, minimum: 1024, maximum: 65535 }
        new_published_port: { type: integer, minimum: 1024, maximum: 65535 }
        old_container_port: { type: integer, minimum: 1024, maximum: 65535 }
        new_container_port: { type: integer, minimum: 1024, maximum: 65535 }
        old_health_port: { type: integer, minimum: 1024, maximum: 65535 }
        new_health_port: { type: integer, minimum: 1024, maximum: 65535 }
        approved_compose_config_sha256:
          type: string
          pattern: "^[a-f0-9]{64}$"
        approved_compose_revision: { type: integer, format: int64, minimum: 1 }
        expected_version_env_sha256:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
        expected_container_id:
          type: string
          pattern: "^[a-f0-9]{12,64}$"
        expected_image_id:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
        expected_repository_digest:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
      description: Immutable Docker runtime binding. The server additionally requires health ports to equal their published ports and rejects a no-op across advertised, published, and container ports.
    SystemUpdatePortReconfiguration:
      $ref: '../schemas/system-update-job.schema.json#/$defs/portReconfigurationPlan'
    SystemUpdatePortMutationGrantBinding:
      allOf:
        - $ref: '../schemas/system-update-job.schema.json#/$defs/portReconfigurationPlan'
        - not: { required: [ result ] }
      description: Version 2 binds immutable B/T/R intent. The transport plan_sha256 is a distinct lease/session-bound digest; reconcile never repeats the initial B-to-T consume.
    SystemUpdateJob:
      $ref: ../schemas/system-update-job.schema.json
    SystemUpdateAgentStatus:
      $ref: ../schemas/system-update-agent-status.schema.json
    SystemUpdateHostStatus:
      $ref: ../schemas/system-update-host-status.schema.json
    SystemUpdatesResponse:
      type: object
      additionalProperties: false
      required: [ updaters, hosts, targets, jobs ]
      properties:
        updaters:
          type: array
          items: { $ref: "#/components/schemas/SystemUpdateAgentStatus" }
        hosts:
          type: array
          items: { $ref: "#/components/schemas/SystemUpdateHostStatus" }
        targets:
          type: array
          items: { $ref: "#/components/schemas/SystemUpdateTarget" }
        jobs:
          type: array
          items: { $ref: "#/components/schemas/SystemUpdateJob" }
    UpdateAgentClaimRequest:
      type: object
      additionalProperties: false
      required: [ updater_id, host_id, lease_generation, fence ]
      properties:
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$" }
        host_id:
          type: string
          minLength: 1
          maxLength: 191
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$"
        lease_generation: { type: integer, format: int64, minimum: 1 }
        fence: { type: integer, format: int64, minimum: 1 }
        active_job_id:
          type: string
          minLength: 1
          maxLength: 64
          pattern: '\S'
          description: Optional locally active job. When present, the server returns only that job's renewed lease or an explicit clear_active_job_id response; it never claims a different job.
    UpdateAgentClaimResponse:
      oneOf:
        - $ref: "#/components/schemas/UpdateAgentLeaseClaimResponse"
        - $ref: "#/components/schemas/UpdateAgentClearActiveJobResponse"
    UpdateAgentLeaseClaimResponse:
      type: object
      additionalProperties: false
      required: [ job, lease_token, lease_expires_at, lease_generation, report_sequence, recovery_required, last_status ]
      properties:
        job: { $ref: "#/components/schemas/SystemUpdateJob" }
        lease_token:
          type: string
          minLength: 32
          writeOnly: true
        lease_expires_at: { type: string, format: date-time }
        lease_generation: { type: integer, format: int64, minimum: 1 }
        report_sequence: { type: integer, format: int64, minimum: 1 }
        recovery_required: { type: boolean }
        last_status:
          type: string
          enum: [ queued, claimed, downloading, verifying, staging, stopping, installing, starting, health_checking, rolling_back, reconciling, succeeded, rolled_back, failed, canceled ]
    UpdateAgentClearActiveJobResponse:
      type: object
      additionalProperties: false
      required: [ clear_active_job_id ]
      properties:
        clear_active_job_id:
          type: boolean
          const: true
          description: The reported active job is terminal, missing, or not owned by this updater and must be durably cleared before another claim.
      x-autostream-behavioral-validator: contracts.ValidateUpdateAgentClearActiveJobResponse
    UpdateAgentReportRequest:
      type: object
      additionalProperties: false
      required: [ updater_id, host_id, lease_token, lease_generation, fence, sequence, status ]
      allOf:
        - if:
            required: [ port_reconfigure ]
          then:
            oneOf:
              - properties:
                  status: { const: succeeded }
                  port_reconfigure:
                    properties:
                      result:
                        enum: [ applied, unchanged ]
              - properties:
                  status: { const: rolled_back }
                  port_reconfigure:
                    properties:
                      result: { const: rolled_back }
              - properties:
                  status: { const: failed }
                  port_reconfigure:
                    properties:
                      result: { const: rollback_failed }
      properties:
        updater_id: { type: string, minLength: 1, maxLength: 128, pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$" }
        host_id: { type: string, minLength: 1, maxLength: 191, pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$" }
        fence: { type: integer, format: int64, minimum: 1 }
        lease_token:
          type: string
          minLength: 32
          writeOnly: true
        lease_generation: { type: integer, format: int64, minimum: 1 }
        sequence: { type: integer, format: int64, minimum: 1 }
        status:
          type: string
          enum: [ claimed, downloading, verifying, staging, stopping, installing, starting, health_checking, rolling_back, reconciling, succeeded, rolled_back, failed ]
        progress: { type: integer, minimum: 0, maximum: 100 }
        code: { type: string, maxLength: 128, pattern: "^[a-z0-9_.-]+$" }
        message: { type: string, maxLength: 500 }
        artifact_digest: { type: string, pattern: "^sha256:[a-f0-9]{64}$" }
        previous_digest: { type: string, pattern: "^sha256:[a-f0-9]{64}$" }
        port_reconfigure:
          type: object
          additionalProperties: false
          required: [ result ]
          properties:
            result:
              type: string
              enum: [ applied, rolled_back, unchanged, rollback_failed ]
    UpdateAgentMutationGrantIssueRequest:
      $ref: '../schemas/update-agent-mutation-grant-issue-request.schema.json'
    UpdateAgentMutationGrantIssueResponse:
      type: object
      additionalProperties: false
      required: [ grant_token, expires_at ]
      properties:
        grant_token:
          type: string
          minLength: 32
          maxLength: 256
          pattern: '^[^\u0000-\u001F\u007F]+$'
          writeOnly: true
          description: Returned once and supplied as the Bearer credential when the destination helper consumes the grant. It must not be logged or persisted.
        expires_at: { type: string, format: date-time }
    UpdateAgentMutationGrantConsumeRequest:
      $ref: '../schemas/update-agent-mutation-grant-consume-request.schema.json'
