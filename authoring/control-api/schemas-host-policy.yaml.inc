    HostAgentPolicyRequest:
      type: object
      additionalProperties: false
      required: [ service_id, current_revision ]
      description: Server-owned transport, execution_host_id, and ownership_epoch fields are never accepted from this request.
      properties:
        service_id:
          type: string
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        current_revision:
          type: integer
          minimum: 0
    HostAgentPolicyTarget:
      type: object
      additionalProperties: false
      required: [ service_id, service_type, deployment_mode ]
      properties:
        service_id:
          type: string
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        service_type:
          enum: [ control_panel, worker, encoder_recorder, discord_bot, observability ]
        deployment_mode:
          enum: [ systemd, docker ]
        desired_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        applied_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        local_listen_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        local_health_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        applied_config_revision:
          type: integer
          format: int64
          minimum: 1
        applied_config_sha256:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
    HostAgentPolicyResponse:
      type: object
      additionalProperties: false
      required: [ service_id, transport_mode, execution_host_id, ownership_epoch, revision, source_policy_revision, local_executor_policy_revision, observe_only, targets ]
      allOf:
        - if:
            properties:
              ownership_epoch: { const: 0 }
          then:
            properties:
              observe_only: { const: true }
          else:
            properties:
              observe_only: { const: false }
        - if:
            properties:
              ownership_epoch: { const: 0 }
          then:
            oneOf:
              - properties:
                  local_executor_policy_revision: { const: 0 }
                not:
                  required: [ local_executor_policy_sha256 ]
              - required: [ local_executor_policy_sha256 ]
                properties:
                  local_executor_policy_revision: { minimum: 1 }
          else:
            required: [ local_executor_policy_sha256 ]
            properties:
              local_executor_policy_revision: { minimum: 1 }
        - if:
            required: [ self_update_status ]
          then:
            required: [ self_update_id, self_update_revision, self_update_status ]
            allOf:
              - if:
                  required: [ self_update_status ]
                  properties:
                    self_update_status: { const: cancel_requested }
                then:
                  not:
                    anyOf:
                      - required: [ runtime_requirement ]
                      - required: [ self_update ]
                else:
                  required: [ runtime_requirement, self_update ]
          else:
            not:
              anyOf:
                - required: [ self_update_id ]
                - required: [ self_update_revision ]
                - required: [ runtime_requirement ]
                - required: [ self_update ]
        - if:
            anyOf:
              - required: [ self_update_id ]
              - required: [ self_update_revision ]
              - required: [ runtime_requirement ]
              - required: [ self_update ]
          then:
            required: [ self_update_status ]
      properties:
        service_id:
          type: string
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        transport_mode:
          const: pull_v2
          description: Server-owned transport binding.
        execution_host_id:
          type: string
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$"
          description: Server-owned execution-host binding.
        ownership_epoch:
          type: integer
          minimum: 0
          description: Server-owned monotonic fencing epoch. Zero is the observe-only, not-yet-owned sentinel and is never valid for mutation.
        revision:
          type: integer
          minimum: 1
          description: Projection cursor for this host-agent policy.
        source_policy_revision:
          type: integer
          format: int64
          minimum: 1
        local_executor_policy_revision:
          type: integer
          format: int64
          minimum: 0
          description: Zero is allowed only for an observe-only epoch-zero policy with no policy digest.
        local_executor_policy_sha256:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
        observe_only:
          type: boolean
        targets:
          type: array
          uniqueItems: true
          items:
            $ref: "#/components/schemas/HostAgentPolicyTarget"
        self_update_id:
          type: string
          minLength: 1
          maxLength: 128
          pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        self_update_revision:
          type: integer
          format: int64
          minimum: 1
        self_update_status:
          enum: [ queued, staging, activating, verifying, rolling_back, cancel_requested ]
        runtime_requirement:
          $ref: "#/components/schemas/HostAgentRuntimeRequirement"
        self_update:
          $ref: "#/components/schemas/HostAgentSelfUpdateDirective"
    HostAgentRuntimeRequirement:
      type: object
      additionalProperties: false
      required: [ minimum_agent_version, minimum_executor_version, agent_protocol_version, executor_protocol_version, mutation_protocol_version, recovery_protocol_version ]
      properties:
        minimum_agent_version:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        minimum_executor_version:
          type: string
          minLength: 6
          maxLength: 128
          pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
        agent_protocol_version: { type: integer, minimum: 1 }
        executor_protocol_version: { type: integer, minimum: 1 }
        mutation_protocol_version: { type: integer, minimum: 1 }
        recovery_protocol_version: { type: integer, minimum: 1 }
    HostAgentSelfUpdateDirective:
      $ref: "../schemas/host-agent-self-update-directive.schema.json"
    HostSelfUpdateReleaseBinding:
      $ref: "../schemas/host-self-update-release-binding.schema.json"
