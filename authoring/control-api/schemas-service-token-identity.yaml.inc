    ServiceEndpoint:
      type: object
      additionalProperties: false
      required: [ host, port, ssl_enabled, public_url ]
      properties:
        host:
          type: string
          minLength: 1
        port:
          type: integer
          minimum: 1
          maximum: 65535
        ssl_enabled:
          type: boolean
        public_url:
          type: string
          format: uri
          pattern: "^https?://"
    RegisteredService:
      type: object
      required: [ service_id, service_type, service_name, ssl_enabled, version, status, capabilities, created_at, updated_at ]
      allOf:
        - if:
            required: [ service_type ]
            properties:
              service_type:
                const: update_agent
          then:
            required: [ transport_mode, execution_host_id, ownership_epoch ]
            not:
              anyOf:
                - required: [ host ]
                - required: [ port ]
                - required: [ public_url ]
                - required: [ desired_endpoint ]
                - required: [ applied_endpoint ]
                - required: [ reported_endpoint ]
                - required: [ endpoint_revision ]
                - required: [ endpoint_status ]
                - required: [ applied_config_revision ]
                - required: [ applied_config_sha256 ]
          else:
            required: [ public_url ]
            not:
              anyOf:
                - required: [ transport_mode ]
                - required: [ execution_host_id ]
                - required: [ ownership_epoch ]
      properties:
        ssl_enabled:
          type: boolean
        description:
          type: string
        host:
          type: string
        port:
          type: integer
          minimum: 0
          maximum: 65535
        reported_version:
          type: string
        reported_commit:
          type: string
        reported_build_date:
          type: string
        last_reported_at:
          type: string
          format: date-time
          nullable: true
        reported_hostname:
          type: string
        reported_os:
          type: string
        reported_arch:
          type: string
        reported_capabilities:
          $ref: "#/components/schemas/EncoderOutputRelayCapabilities"
        service_id:
          type: string
        service_type:
          enum: [ discord_bot, encoder_recorder, worker, observability, update_agent ]
        transport_mode:
          const: pull_v2
          description: Protocol-major-2 transport for update_agent records. An update_agent omits host, port, and public_url.
        execution_host_id:
          type: string
          minLength: 1
          maxLength: 191
          pattern: "^[^\\x00-\\x1F\\x7F]+$"
          description: Stable execution-host identity owned by a pull_v2 update agent.
        ownership_epoch:
          type: integer
          minimum: 0
          description: Server-owned execution-host ownership epoch. Zero is the observe-only, not-yet-owned sentinel.
        service_name:
          type: string
        public_url:
          type: string
          format: uri
          pattern: "^https?://"
        desired_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        applied_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        reported_endpoint:
          $ref: "#/components/schemas/ServiceEndpoint"
        endpoint_revision:
          type: integer
          minimum: 1
        endpoint_status:
          type: string
          minLength: 1
          maxLength: 32
        applied_config_revision:
          type: integer
          format: int64
          minimum: 1
        applied_config_sha256:
          type: string
          pattern: "^sha256:[a-f0-9]{64}$"
        version:
          type: string
        status:
          enum: [ unknown, pending, registered, assigned, online, updating, offline, degraded, starting, stopping, restart_requested ]
        assignment_role:
          enum: [ primary, standby ]
          description: present only when returned as a stream assignment. Only primary services receive dispatch; standby services are failover candidates and cannot resolve stream-scoped runtime secrets until promoted.
        last_heartbeat_at:
          type: string
          format: date-time
          nullable: true
        current_stream_id:
          type: string
          nullable: true
        capabilities:
          $ref: "#/components/schemas/EncoderOutputRelayCapabilities"
        metrics:
          type: object
          additionalProperties: true
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    ServiceTokenCreateRequest:
      type: object
      additionalProperties: false
      required: [ service_type, scopes ]
      properties:
        service_type:
          enum: [ discord_bot, encoder_recorder, worker, observability, update_agent ]
        scopes:
          type: array
          items:
            enum:
              - service.register
              - service.heartbeat
              - service.logs.write
              - service.status.write
              - service.config.read
              - service.secret.resolve
              - worker.events.write
              - encoder.status.write
              - discord.status.write
              - streams.start
              - streams.stop
              - observability.ingest
              - notifications.email.send
              - remediation.execute
              - updates.claim
              - updates.report
              - updates.authorize
          uniqueItems: true
        service_id:
          type: string
          description: Optional service ID to pre-create as a pending registry entry. Requires service.register scope.
        service_name:
          type: string
        public_url:
          type: string
          format: uri
          pattern: "^https?://"
        version:
          type: string
        capabilities:
          $ref: "#/components/schemas/EncoderOutputRelayCapabilities"
          description: Non-secret capability hints. Secret-like keys and values are redacted before persistence.
    ServiceToken:
      type: object
      required: [ id, service_type, scopes, created_at ]
      properties:
        id:
          type: string
        service_type:
          enum: [ discord_bot, encoder_recorder, worker, observability, update_agent ]
        scopes:
          type: array
          items:
            enum:
              - service.register
              - service.heartbeat
              - service.logs.write
              - service.status.write
              - service.config.read
              - service.secret.resolve
              - worker.events.write
              - encoder.status.write
              - discord.status.write
              - streams.start
              - streams.stop
              - observability.ingest
              - notifications.email.send
              - remediation.execute
              - updates.claim
              - updates.report
              - updates.authorize
        revoked_at:
          type: string
          format: date-time
          nullable: true
        created_at:
          type: string
          format: date-time
        token:
          type: string
          description: One-time raw token returned only on create or rotate responses. Never returned by list endpoints.
