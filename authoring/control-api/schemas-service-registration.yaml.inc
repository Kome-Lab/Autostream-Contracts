    EncoderOutputRelayMode:
      type: string
      enum: [ direct, live_api_relay_static ]
      description: >
        Non-secret v2 Encoder/Recorder output relay routing vocabulary. direct uses the assignment-scoped runtime target; live_api_relay_static requires the matching fixed relay binding. Unknown values are rejected fail closed. This value never carries an RTMPS URL, relay URL, or stream key.
    EncoderOutputRelayCapabilities:
      type: object
      additionalProperties: true
      not:
        anyOf:
          - required: [ scene_video_srt ]
          - required: [ worker_video_ingest_srt ]
      description: Non-secret service capability hints for output relay routing, negotiated Worker scene-video transport, and strict Discord Bot start-envelope selection.
      allOf:
        - if:
            required: [ output_relay_mode ]
            properties:
              output_relay_mode: { const: live_api_relay_static }
          then:
            required: [ output_relay_binding_id ]
      properties:
        output_relay_mode:
          $ref: "#/components/schemas/EncoderOutputRelayMode"
        output_relay_binding_id:
          type: string
          pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
          description: Required with live_api_relay_static. Non-secret fixed relay binding identity that must match the selected output; never an RTMPS address or stream key.
        scene_frames_mjpeg_srt:
          type: boolean
          const: true
          description: Worker can render the complete scene as low-rate MJPEG images over the negotiated job-scoped SRT ingest without video encoding.
        worker_frame_ingest_mjpeg_srt:
          type: boolean
          const: true
          description: Encoder/Recorder can receive Worker-rendered JPEG images, hold the latest frame, and perform the only video encode and audio mux.
        scene_appearance_v1:
          type: boolean
          const: true
          description: Worker implements the complete scene appearance start snapshot, authenticated asset retrieval, integrity verification, cache, rendering, and exact default parity.
        live_video_cover_v1:
          type: boolean
          const: true
          description: Encoder/Recorder implements generation-fenced Video Cover apply and reconcile with independent Cover and Watermark imagefeeds and graph-applied witnesses.
        discord_resolved_target_v2:
          type: boolean
          const: true
          description: Advertises actual Discord Bot support for the strict schema_version=2 resolved-target start DTO. Missing, false, unknown values, and version strings make the service ineligible for start dispatch.
    ServiceRegistrationRequest:
      type: object
      additionalProperties: false
      required: [ service_id, service_type, service_name, version, capabilities ]
      allOf:
        - if:
            required: [ service_type ]
            properties:
              service_type:
                const: update_agent
          then:
            required: [ transport_mode ]
            not:
              anyOf:
                - required: [ host ]
                - required: [ port ]
                - required: [ public_url ]
          else:
            required: [ public_url ]
            not:
              required: [ transport_mode ]
      properties:
        service_id:
          type: string
          minLength: 1
        service_type:
          enum: [ discord_bot, encoder_recorder, worker, observability, update_agent ]
        transport_mode:
          const: pull_v2
          description: Required protocol-major-2 transport for update_agent registration. An update_agent must omit host, port, and public_url. Server-owned execution_host_id and ownership_epoch are never accepted from registration input.
        service_name:
          type: string
          minLength: 1
        description:
          type: string
        host:
          type: string
        port:
          type: integer
          minimum: 0
          maximum: 65535
        ssl_enabled:
          type: boolean
        public_url:
          type: string
          format: uri
          pattern: "^https?://"
          description: Control Panel から到達可能な http/https absolute URL。
        version:
          type: string
        commit:
          type: string
        build_date:
          type: string
        capabilities:
          $ref: "#/components/schemas/EncoderOutputRelayCapabilities"
        hostname:
          type: string
        os:
          type: string
        arch:
          type: string
    Heartbeat:
      type: object
      additionalProperties: false
      required: [ service_id, status ]
      properties:
        service_id: { type: string, minLength: 1 }
        node_id: { type: string, minLength: 1 }
        nodeId: { type: string, minLength: 1 }
        current_stream_id: { type: string }
        status:
          type: string
          enum: [ unknown, online, updating, degraded, offline, ready, running, stopping, failed ]
        metrics:
          type: object
          additionalProperties: { type: number }
        version: { type: string }
        commit: { type: string }
        build_date: { type: string }
        hostname: { type: string }
        os: { type: string }
        arch: { type: string }
        capabilities:
          $ref: "#/components/schemas/EncoderOutputRelayCapabilities"
        api:
          type: object
          additionalProperties: false
          required: [ host, port, sslEnabled ]
          properties:
            host: { type: string, minLength: 1 }
            port: { type: integer, minimum: 1, maximum: 65535 }
            sslEnabled: { type: boolean }
        timestamp: { type: string, format: date-time }
