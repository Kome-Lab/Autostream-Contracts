    Profile:
      type: object
      required: [ id, kind, name, config, created_at, updated_at ]
      properties:
        id:
          type: string
        kind:
          enum: [ encoder, archive, caption, overlay, discord_config, youtube_output ]
        name:
          type: string
        config:
          type: object
          additionalProperties: true
          description: Secret references may be present, but raw secret values must not be returned.
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    ServiceRuntimeConfig:
      type: object
      description: Runtime config for the authenticated service. It contains only public service metadata, that service's assignments, and Control Panel-managed non-secret profiles filtered to that service.
      required: [ service, assignments, profiles ]
      properties:
        service:
          $ref: "#/components/schemas/RegisteredService"
        assignments:
          type: array
          items:
            $ref: "#/components/schemas/StreamServiceAssignment"
        profiles:
          type: object
          description: Non-secret runtime profiles. Raw bot tokens, stream keys, refresh tokens, Drive folder IDs, webhook URLs, SMTP settings, and credential material must be represented only by configured status, fingerprints, or runtime secret reference names.
          additionalProperties:
            type: array
            items:
              $ref: "#/components/schemas/Profile"
        stream_discord_configs:
          type: array
          description: Resolved per-stream Discord guild/channel settings for the authenticated Discord Bot service only. Raw bot tokens are never included.
          items:
            type: object
            additionalProperties: false
            required: [ stream_id, assignment_role, discord_config_id, guild_id, voice_channel_id ]
            properties:
              stream_id:
                type: string
              assignment_role:
                type: string
                enum: [ primary, standby ]
              discord_config_id:
                type: string
              guild_id:
                type: string
              voice_channel_id:
                type: string
              text_channel_id:
                type: string
              auto_start_trigger:
                type: string
                enum: [ discord_voice_join ]
                description: Present only when this stream is explicitly allowed to auto-start from a Discord voice-channel join.
        stream_archive_configs:
          type: array
          description: Resolved per-stream archive settings for the authenticated Encoder/Recorder service only. It includes runtime secret reference names and shared-drive flags, but never raw Drive folder IDs, OAuth refresh tokens, client secrets, or credential JSON.
          items:
            type: object
            additionalProperties: false
            required: [ stream_id, assignment_role, archive_profile_id, ready ]
            properties:
              stream_id:
                type: string
              assignment_role:
                type: string
                enum: [ primary, standby ]
              archive_profile_id:
                type: string
              ready:
                type: boolean
              readiness_code:
                type: string
              readiness_message:
                type: string
              archive_config:
                type: object
                description: Non-secret archive dispatch config. Secret material is represented by *_secret_name fields only.
                additionalProperties: true
                not:
                  required: [ base_path ]
                properties:
                  drive_destination_id:
                    type: string
                  archive_profile_id:
                    type: string
                  auth_mode:
                    type: string
                    enum: [ oauth2 ]
                  oauth_account_id:
                    type: string
                  oauth_provider_id:
                    type: string
                  folder_id_secret_name:
                    type: string
                  shared_drive:
                    type: boolean
                  client_id:
                    type: string
                  client_secret_secret_name:
                    type: string
                  refresh_token_secret_name:
                    type: string
        stream_youtube_configs:
          type: array
          description: Resolved per-stream YouTube output settings for the authenticated Encoder/Recorder service only. It includes RTMPS endpoint metadata and runtime secret reference names, but never raw stream keys, OAuth refresh tokens, or YouTube credential material.
          items:
            type: object
            additionalProperties: false
            required: [ stream_id, assignment_role, youtube_output_id, ready ]
            allOf:
              - if:
                  required: [ ready, youtube_config ]
                  properties:
                    ready: { const: true }
                    youtube_config:
                      type: object
                      required: [ mode ]
                      properties:
                        mode: { const: live_api_relay_static }
                then:
                  properties:
                    youtube_config:
                      required: [ mode, output_id, oauth_account_id, relay_binding_id, reusable_live_stream_id, complete_on_stop ]
                      properties:
                        output_id: { type: string, minLength: 1 }
                        oauth_account_id: { type: string, minLength: 1 }
                        relay_binding_id: { type: string, pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" }
                        reusable_live_stream_id: { type: string, minLength: 1 }
                        complete_on_stop: { const: true }
              - if:
                  required: [ active_runtime ]
                  properties:
                    active_runtime:
                      type: object
                      required: [ mode ]
                      properties:
                        mode: { const: live_api_relay_static }
                then:
                  properties:
                    active_runtime:
                      required: [ mode, output_id, oauth_account_id, broadcast_id, live_stream_id, complete_on_stop ]
                      properties:
                        output_id: { type: string, minLength: 1 }
                        oauth_account_id: { type: string, minLength: 1 }
                        broadcast_id: { type: string, minLength: 1 }
                        live_stream_id: { type: string, minLength: 1 }
                        complete_on_stop: { const: true }
            properties:
              stream_id:
                type: string
              assignment_role:
                type: string
                enum: [ primary, standby ]
              youtube_output_id:
                type: string
              ready:
                type: boolean
              readiness_code:
                type: string
              readiness_message:
                type: string
              youtube_config:
                type: object
                description: Non-secret YouTube dispatch config. Stream key material is represented by stream_key_secret_name only.
                additionalProperties: true
                allOf:
                  - if:
                      properties:
                        mode: { const: live_api_relay_static }
                      required: [ mode ]
                    then:
                      not:
                        anyOf:
                          - required: [ rtmp_url ]
                          - required: [ stream_key_secret_name ]
                          - required: [ stream_key ]
                properties:
                  mode:
                    type: string
                    enum: [ stream_key, live_api, live_api_dry_run, live_api_relay_static ]
                  output_id:
                    type: string
                  rtmp_url:
                    type: string
                    pattern: "^rtmps://"
                  stream_key_secret_name:
                    type: string
                  oauth_account_id:
                    type: string
                  relay_binding_id:
                    type: string
                    pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
                    description: Non-secret fixed relay binding identity for live_api_relay_static.
                  reusable_live_stream_id:
                    type: string
                    description: Non-secret reusable YouTube LiveStream identity for live_api_relay_static.
                  broadcast_title_template:
                    type: string
                  broadcast_description:
                    type: string
                  privacy_status:
                    type: string
                    enum: [ private, unlisted, public ]
                  latency_preference:
                    type: string
                    enum: [ normal, low, ultra_low ]
                  enable_auto_start:
                    type: boolean
                  enable_auto_stop:
                    type: boolean
                  complete_on_stop:
                    type: boolean
                  watch_url:
                    type: string
                    format: uri
                    pattern: "^https://www\\.youtube\\.com/watch\\?v=[A-Za-z0-9_-]{6,32}$"
                    description: Canonical public YouTube watch URL derived for the live API runtime. It contains no stream key or OAuth credential.
              active_runtime:
                type: object
                description: Current non-secret YouTube runtime state. Runtime stream keys are represented by stream_key_secret_name only.
                additionalProperties: true
                allOf:
                  - if:
                      properties:
                        mode: { const: live_api_relay_static }
                      required: [ mode ]
                    then:
                      not:
                        anyOf:
                          - required: [ rtmp_url ]
                          - required: [ stream_key_secret_name ]
                          - required: [ stream_key ]
                properties:
                  mode:
                    type: string
                    enum: [ stream_key, live_api, live_api_dry_run, live_api_relay_static ]
                  output_id:
                    type: string
                  oauth_account_id:
                    type: string
                  broadcast_id:
                    type: string
                  live_stream_id:
                    type: string
                  relay_binding_id:
                    type: string
                    pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
                  reusable_live_stream_id:
                    type: string
                  rtmp_url:
                    type: string
                    pattern: "^rtmps://"
                  stream_key_secret_name:
                    type: string
                  dry_run:
                    type: boolean
                  complete_on_stop:
                    type: boolean
                  watch_url:
                    type: string
                    format: uri
                    pattern: "^https://www\\.youtube\\.com/watch\\?v=[A-Za-z0-9_-]{6,32}$"
                    description: Canonical public YouTube watch URL derived for the live API runtime. It contains no stream key or OAuth credential.
    ServiceRuntimeSecretResolveRequest:
      type: object
      additionalProperties: false
      required: [ service_id, secret_name ]
      properties:
        service_id:
          type: string
          minLength: 1
        stream_id:
          type: string
          description: Required for stream-scoped integration secrets such as Drive destination folder IDs.
        archive_profile_id:
          type: string
          description: Required for archive runtime secret references delivered in Encoder/Recorder start payloads.
        secret_name:
          type: string
          minLength: 1
          maxLength: 128
          description: Secret reference name previously returned in the authenticated service runtime config.
    ServiceRuntimeSecretResolveResponse:
      type: object
      additionalProperties: false
      required: [ secret_name, value, expires_in_sec ]
      properties:
        secret_name:
          type: string
        value:
          type: string
          description: Raw runtime secret. This is returned only to the authenticated service and must not be reused in UI, audit metadata, diagnostics, or logs.
        expires_in_sec:
          type: integer
          minimum: 1
          description: Lease lifetime for this resolved value. During this window, the same service/context/secret returns runtime_secret_lease_active instead of the raw value.
