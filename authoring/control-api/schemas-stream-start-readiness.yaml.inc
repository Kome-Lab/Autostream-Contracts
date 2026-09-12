    StartStreamRequest:
      type: object
      additionalProperties: false
      properties:
        discord_config_id:
          type: string
          description: Discord Bot Config selected for this stream. Required before stream-specific guild/channel overrides can be used. Control Panel resolves guild/channel server-side and verifies it matches the assigned primary Discord Bot service.
        discord_guild_id:
          type: string
          description: Stream-specific Discord guild ID override. Valid only with discord_config_id. If omitted, the selected Discord Config default is used.
        discord_voice_channel_id:
          type: string
          description: Stream-specific Discord voice channel ID override. Valid only with discord_config_id. If omitted, the selected Discord Config default is used.
        discord_text_channel_id:
          type: string
          description: Optional stream-specific Discord text channel ID override. Valid only with discord_config_id. If omitted, the selected Discord Config default is used.
        encoder_input_url:
          type: string
        encoder_rtmp_url:
          type: string
        encoder_profile_id:
          type: string
        caption_profile_id:
          type: string
        overlay_profile_id:
          type: string
        encoder_audio_gain_db:
          type: number
          minimum: -60
          maximum: 24
          default: 0
        archive_profile_id:
          type: string
        youtube_output_id:
          type: string
          description: YouTube output profile selected for this stream. Stream key is resolved server-side from a secret reference and is never accepted or returned by the public Control API.
    DiscordYouTubeLiveNotification:
      type: object
      additionalProperties: false
      required:
        - id
        - stream_id
        - watch_url
        - discord_service_id
        - discord_text_channel_id
        - youtube_mode
        - state
        - attempt_count
        - dispatch_attempt_count
        - created_at
        - updated_at
      properties:
        id:
          type: string
        stream_id:
          type: string
        watch_url:
          type: string
          format: uri
          description: Canonical public YouTube watch URL; no RTMP URL or stream key is included.
        discord_service_id:
          type: string
        discord_text_channel_id:
          type: string
        youtube_mode:
          type: string
          enum: [ stream_key, live_api, live_api_relay_static ]
        youtube_broadcast_id:
          type: string
        lifecycle_status:
          type: string
          description: Provider status when known; stream_key uses legacy_unverified by explicit compatibility policy.
        state:
          type: string
          enum: [ awaiting_youtube_live, dispatch_pending, dispatching, bot_dispatching, delivered, delivery_unknown, suppressed ]
        attempt_count:
          type: integer
          minimum: 0
        dispatch_attempt_count:
          type: integer
          minimum: 0
        next_attempt_at:
          type: string
          format: date-time
        last_error:
          type: string
          description: Stable, secret-safe delivery state code; raw Bot errors are not exposed.
        discord_message_id:
          type: string
          description: Bot-confirmed Discord receipt only.
        delivered_at:
          type: string
          format: date-time
        recovery_of_id:
          type: string
          description: Prior delivery_unknown record acknowledged by an operator.
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    DiscordYouTubeLiveNotificationResponse:
      type: object
      additionalProperties: false
      required: [ notification ]
      properties:
        notification:
          $ref: "#/components/schemas/DiscordYouTubeLiveNotification"
    DiscordYouTubeLiveNotificationRecoveryRequest:
      type: object
      additionalProperties: false
      required: [ acknowledge_possible_duplicate ]
      properties:
        acknowledge_possible_duplicate:
          type: boolean
          const: true
          description: Explicit operator attestation that the original request may already have posted to Discord.
    StreamSettingsWriteRequest:
      type: object
      additionalProperties: false
      properties:
        discord_config_id:
          type: string
          description: Control Panel managed Discord Bot Config. Required when Discord guild/channel overrides are provided.
        discord_guild_id:
          type: string
          description: Stream-specific Discord guild ID override. Valid only with discord_config_id.
        discord_voice_channel_id:
          type: string
          description: Stream-specific Discord voice channel ID override. Valid only with discord_config_id.
        discord_text_channel_id:
          type: string
          description: Optional stream-specific Discord text channel ID override. Valid only with discord_config_id.
        auto_start_trigger:
          type: string
          enum: [ "", discord_voice_join ]
          description: Set to discord_voice_join to allow this waiting stream to start automatically when a user joins the configured Discord voice channel. Empty or omitted disables automatic start.
        encoder_profile_id:
          type: string
        caption_profile_id:
          type: string
        overlay_profile_id:
          type: string
        encoder_audio_gain_db:
          type: number
          minimum: -60
          maximum: 24
          default: 0
        archive_profile_id:
          type: string
        youtube_output_id:
          type: string
        encoder_input_url:
          type: string
    StreamWriteRequest:
      type: object
      additionalProperties: false
      required: [ name ]
      properties:
        name:
          type: string
          minLength: 1
        discord_config_id:
          type: string
          description: Control Panel managed Discord Bot Config. Required when Discord guild/channel overrides are provided.
        discord_guild_id:
          type: string
          description: Stream-specific Discord guild ID override. Valid only with discord_config_id.
        discord_voice_channel_id:
          type: string
          description: Stream-specific Discord voice channel ID override. Valid only with discord_config_id.
        discord_text_channel_id:
          type: string
          description: Optional stream-specific Discord text channel ID override. Valid only with discord_config_id.
        auto_start_trigger:
          type: string
          enum: [ "", discord_voice_join ]
          description: Set to discord_voice_join to allow this waiting stream to start automatically when a user joins the configured Discord voice channel. Empty or omitted disables automatic start.
        encoder_profile_id:
          type: string
        caption_profile_id:
          type: string
        overlay_profile_id:
          type: string
        encoder_audio_gain_db:
          type: number
          minimum: -60
          maximum: 24
          default: 0
        archive_profile_id:
          type: string
        youtube_output_id:
          type: string
        encoder_input_url:
          type: string
    EncoderRuntimeSettingsRequest:
      $ref: "../schemas/encoder-runtime-settings-request.schema.json"
    MissingStreamAssignmentsResponse:
      type: object
      additionalProperties: false
      required:
        - code
        - missing_service_types
      properties:
        code:
          type: string
          enum:
            - missing_stream_assignments
        missing_service_types:
          type: array
          items:
            type: string
            enum:
              - discord_bot
              - worker
              - encoder_recorder
    ReadinessIssue:
      type: object
      additionalProperties: false
      required:
        - code
        - message
      properties:
        service_id:
          type: string
        service_type:
          type: string
        code:
          type: string
          enum:
            - missing_stream_assignments
            - service_call_token_missing
            - service_public_url_invalid
            - service_public_url_blocked
            - encoder_public_url_missing
            - encoder_public_url_invalid
            - encoder_public_url_blocked
            - service_offline
            - service_heartbeat_stale
            - discord_audio_forward_unavailable
            - discord_audio_capture_unavailable
            - discord_caption_audio_forward_unavailable
            - worker_deepgram_transcription_unavailable
            - discord_config_required
            - discord_config_not_found
            - discord_config_invalid
            - discord_config_service_mismatch
            - youtube_output_not_found
            - youtube_output_invalid_config
            - youtube_stream_key_unavailable
            - youtube_live_api_unavailable
            - youtube_oauth_account_unavailable
            - archive_profile_not_found
            - archive_profile_invalid_config
            - drive_destination_not_found
            - drive_destination_unavailable
            - drive_oauth_account_unavailable
        message:
          type: string
    StartReadinessResponse:
      type: object
      additionalProperties: false
      required:
        - stream_id
        - ready
        - missing_service_types
        - issues
        - assigned_service_count
        - primary_service_count
      properties:
        stream_id:
          type: string
        ready:
          type: boolean
        missing_service_types:
          type: array
          items:
            type: string
            enum:
              - discord_bot
              - worker
              - encoder_recorder
        issues:
          type: array
          items:
            $ref: "#/components/schemas/ReadinessIssue"
        assigned_service_count:
          type: integer
          minimum: 0
        primary_service_count:
          type: integer
          minimum: 0
          description: Number of primary assignments eligible for start dispatch. Standby assignments are listed but excluded until promoted.
        assignments:
          type: array
          description: Public registered service summaries. Service token binding IDs and raw tokens are never returned.
          items:
            $ref: "#/components/schemas/RegisteredService"
    ExternalE2EConfigResponse:
      type: object
      additionalProperties: false
      required: [ schema_version, stream_id, runtime_config, service_assignments, confirmations, readiness ]
      properties:
        schema_version:
          type: integer
          enum: [ 1 ]
          description: Version of the non-secret external verification confirmation export schema.
        stream_id:
          type: string
        runtime_config:
          type: object
          additionalProperties: false
          description: Control Panel-managed runtime config IDs only. Raw Discord channel IDs, Drive folder IDs, OAuth tokens, RTMPS URLs, stream keys, and session cookies are never returned.
          required: [ youtube_output_id, drive_destination_id, discord_config_id, encoder_profile_id, archive_profile_id ]
          properties:
            youtube_output_id:
              type: string
            drive_destination_id:
              type: string
            discord_config_id:
              type: string
            encoder_profile_id:
              type: string
            archive_profile_id:
              type: string
        service_assignments:
          type: object
          additionalProperties: false
          description: Assigned service IDs only. Service token binding IDs and raw service tokens are never returned.
          required: [ discord_bot_service_id, encoder_recorder_primary_service_id, worker_primary_service_id, encoder_recorder_standby_service_id, worker_standby_service_id ]
          properties:
            discord_bot_service_id:
              type: string
            encoder_recorder_primary_service_id:
              type: string
            worker_primary_service_id:
              type: string
            encoder_recorder_standby_service_id:
              type: string
            worker_standby_service_id:
              type: string
        confirmations:
          type: object
          additionalProperties: false
          required: [ youtube_output_saved, drive_destination_saved, discord_config_saved, primary_assignments_saved, runtime_config_distribution_enabled ]
          properties:
            youtube_output_saved:
              type: boolean
            drive_destination_saved:
              type: boolean
            discord_config_saved:
              type: boolean
            primary_assignments_saved:
              type: boolean
            runtime_config_distribution_enabled:
              type: boolean
        readiness:
          type: object
          additionalProperties: false
          description: Secret-safe external verification readiness summary. Missing lists contain field names, service types, or capability names only.
          required: [ ready, missing_confirmations, missing_runtime_ids, missing_primary_services, missing_runtime_config_capabilities ]
          properties:
            ready:
              type: boolean
            missing_confirmations:
              type: array
              items:
                type: string
            missing_runtime_ids:
              type: array
              items:
                type: string
            missing_primary_services:
              type: array
              items:
                type: string
            missing_runtime_config_capabilities:
              type: array
              items:
                type: string
