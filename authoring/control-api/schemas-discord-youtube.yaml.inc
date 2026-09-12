    DiscordConfig:
      type: object
      additionalProperties: false
      required: [ id, name, created_at, updated_at ]
      properties:
        id:
          type: string
        name:
          type: string
        service_id:
          type: string
          description: Discord Bot service allowed to receive this config through runtime config.
        guild_id:
          type: string
        voice_channel_id:
          type: string
        text_channel_id:
          type: string
        bot_token_configured:
          type: boolean
        bot_token_fingerprint:
          type: string
        caption_enabled:
          type: boolean
        stt_profile_id:
          type: string
        reconnect_enabled:
          type: boolean
        reconnect_max_attempts:
          type: integer
          minimum: 1
        reconnect_base_delay:
          type: string
          description: Go duration string for the initial voice rejoin backoff, for example 2s.
        reconnect_max_delay:
          type: string
          description: Go duration string for the maximum voice rejoin backoff, for example 30s.
        audio_forward_enabled:
          type: boolean
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    DiscordConfigWriteRequest:
      type: object
      additionalProperties: false
      required: [ name ]
      properties:
        name:
          type: string
          minLength: 1
        service_id:
          type: string
        guild_id:
          type: string
        voice_channel_id:
          type: string
        text_channel_id:
          type: string
        bot_token:
          type: string
          writeOnly: true
          description: Write-only Discord Bot token. Never returned by Control API, UI, logs, or audit metadata.
        caption_enabled:
          type: boolean
        stt_profile_id:
          type: string
        reconnect_enabled:
          type: boolean
        reconnect_max_attempts:
          type: integer
          minimum: 1
        reconnect_base_delay:
          type: string
          description: Go duration string for the initial voice rejoin backoff, for example 2s.
        reconnect_max_delay:
          type: string
          description: Go duration string for the maximum voice rejoin backoff, for example 30s.
        audio_forward_enabled:
          type: boolean
    YouTubeOutput:
      type: object
      additionalProperties: false
      required: [ id, name, mode, created_at, updated_at ]
      allOf:
        - if:
            properties:
              mode: { const: live_api_relay_static }
            required: [ mode ]
          then:
            not:
              anyOf:
                - required: [ rtmp_url ]
                - required: [ stream_key_configured ]
                - required: [ stream_key_fingerprint ]
                - required: [ watch_url ]
      properties:
        id:
          type: string
        name:
          type: string
        mode:
          type: string
          enum: [ stream_key, live_api_dry_run, live_api, live_api_relay_static ]
        rtmp_url:
          type: string
          pattern: "^rtmps://"
          description: Returned without the stream key. Runtime stream keys are delivered only to the assigned Encoder/Recorder.
        stream_key_configured:
          type: boolean
        stream_key_fingerprint:
          type: string
        watch_url:
          type: string
          format: uri
          pattern: "^https://www\\.youtube\\.com/watch\\?v=[A-Za-z0-9_-]{6,32}$"
          description: Canonical public watch URL used by stream_key outputs. It contains no stream key or OAuth credential.
        oauth_account_id:
          type: string
        relay_binding_id:
          type: string
          pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
          description: Non-secret fixed Encoder/Recorder relay binding identity for live_api_relay_static. It is never an RTMPS URL or stream key.
        reusable_live_stream_id:
          type: string
          description: Non-secret reusable YouTube LiveStream identity for live_api_relay_static. Ingestion addresses and stream keys are never returned.
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
          description: When true, Control Panel completes the YouTube Live broadcast during normal stream stop.
        complete_retry_count:
          type: integer
          minimum: 0
          description: Number of failed YouTube Live API complete attempts retained for automatic backoff retry.
        complete_next_retry_at:
          type: string
          format: date-time
          description: Next automatic YouTube Live API complete retry time, when a retained runtime is waiting for retry.
        complete_last_error:
          type: string
          description: Safe error code from the latest YouTube Live API complete failure. Raw OAuth or stream key material must not be included.
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    YouTubeOutputWriteRequest:
      type: object
      additionalProperties: false
      required: [ name, mode ]
      oneOf:
        - required: [ name, mode ]
          properties:
            mode:
              enum: [ stream_key ]
        - required: [ name, mode, oauth_account_id ]
          properties:
            mode:
              enum: [ live_api_dry_run, live_api ]
        - required: [ name, mode, oauth_account_id, relay_binding_id, reusable_live_stream_id ]
          properties:
            mode:
              enum: [ live_api_relay_static ]
          not:
            anyOf:
              - required: [ rtmp_url ]
              - required: [ stream_key ]
              - required: [ watch_url ]
      properties:
        name:
          type: string
          minLength: 1
        mode:
          type: string
          enum: [ stream_key, live_api_dry_run, live_api, live_api_relay_static ]
        rtmp_url:
          type: string
          pattern: "^rtmps://"
        stream_key:
          type: string
          writeOnly: true
          description: Write-only secret. Required when creating stream_key outputs, optional on update to keep the existing encrypted key. Never returned by Control API, UI, logs, or audit metadata.
        watch_url:
          type: string
          format: uri
          description: YouTube watch or share URL for stream_key mode. The server validates supported YouTube hosts and video IDs, rejects credentials, fragments, and non-default ports, then stores a canonical watch URL. New UI-created profiles require it; existing API profiles without this field remain compatible.
        oauth_account_id:
          type: string
          minLength: 1
          description: Required for live_api, live_api_dry_run, and live_api_relay_static modes.
        relay_binding_id:
          type: string
          pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
          description: Required for live_api_relay_static. Non-secret fixed Encoder/Recorder relay binding identity; never an RTMPS URL or stream key.
        reusable_live_stream_id:
          type: string
          minLength: 1
          description: Required for live_api_relay_static. Non-secret reusable YouTube LiveStream identity; ingestion details and stream keys are not accepted.
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
          description: When true, Control Panel completes the YouTube Live broadcast during normal stream stop.
    YouTubeRuntimeConfig:
      type: object
      additionalProperties: false
      required: [ mode ]
      allOf:
        - if:
            properties:
              mode: { const: live_api_relay_static }
            required: [ mode ]
          then:
            required: [ mode, output_id, oauth_account_id, relay_binding_id, reusable_live_stream_id, broadcast_id, live_stream_id, complete_on_stop ]
            properties:
              output_id: { type: string, minLength: 1 }
              oauth_account_id: { type: string, minLength: 1 }
              relay_binding_id: { type: string, pattern: "^relay-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" }
              reusable_live_stream_id: { type: string, minLength: 1 }
              broadcast_id: { type: string, minLength: 1 }
              live_stream_id: { type: string, minLength: 1 }
              complete_on_stop: { const: true }
            not:
              anyOf:
                - required: [ rtmp_url ]
                - required: [ stream_key_secret_name ]
      properties:
        mode:
          type: string
          enum: [ stream_key, live_api_dry_run, live_api, live_api_relay_static ]
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
          description: Non-secret fixed relay binding identity for live_api_relay_static.
        reusable_live_stream_id:
          type: string
          description: Non-secret reusable YouTube LiveStream identity for live_api_relay_static. No ingestion address or stream key is present.
        rtmp_url:
          type: string
          pattern: "^rtmps://"
          description: Prepared RTMPS ingest URL for recovering an active YouTube Live API runtime. Stream key material remains in stream_key_secret_name.
        stream_key_secret_name:
          type: string
          description: Short-lived runtime secret reference for YouTube Live API stream key. The raw key must not be included in this object.
        watch_url:
          type: string
          format: uri
          pattern: "^https://www\\.youtube\\.com/watch\\?v=[A-Za-z0-9_-]{6,32}$"
          description: Canonical public watch URL. Live API modes derive it from broadcast_id; live_api_relay_static runtime may carry it. stream_key mode uses the configured output watch_url.
        dry_run:
          type: boolean
        complete_on_stop:
          type: boolean
          description: When true, Control Panel completes the YouTube Live broadcast during normal stream stop.
    YouTubeRelayStaticRecoveryResolveRequest:
      $ref: "../schemas/youtube-relay-static-recovery-resolve-request.schema.json"
    YouTubeRelayStaticRecoveryResolveResponse:
      $ref: "../schemas/youtube-relay-static-recovery-resolve-response.schema.json"
