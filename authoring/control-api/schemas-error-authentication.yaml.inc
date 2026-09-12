    ErrorResponse:
      type: object
      required: [ code ]
      properties:
        request_id:
          type: string
        code:
          type: string
          description: Stable, secret-safe Control API error or readiness issue code. Start-readiness checks are side-effect-free and do not read or return raw secrets.
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
            - live_api_requires_managed_output_relay
            - youtube_relay_static_unavailable
            - youtube_relay_static_binding_unavailable
            - youtube_relay_binding_store_unavailable
            - youtube_relay_binding_in_use
            - youtube_relay_binding_claim_check_failed
            - youtube_relay_binding_release_pending
            - youtube_relay_static_config_changed_reload
            - youtube_relay_static_completion_requires_completed_stream
            - youtube_relay_static_recovery_required
            - youtube_relay_static_external_cleanup_confirmation_required
            - youtube_relay_static_recovery_not_found
            - stream_relay_recovery_not_safe_while_active
            - youtube_relay_static_recovery_not_required
            - youtube_relay_static_recovery_encoder_stop_unavailable
            - youtube_relay_static_recovery_encoder_stop_unconfirmed
            - youtube_relay_static_recovery_broadcast_unknown
            - youtube_relay_static_recovery_dispatch_state_invalid
            - youtube_relay_static_recovery_complete_failed
            - youtube_relay_static_recovery_cleanup_failed
            - youtube_relay_static_recovery_cleanup_unavailable
            - youtube_relay_static_recovery_credentials_failed
            - resolve_youtube_relay_static_recovery_failed
            - get_youtube_relay_static_recovery_failed
            - archive_profile_not_found
            - archive_profile_invalid_config
            - drive_destination_not_found
            - drive_destination_unavailable
            - drive_oauth_account_unavailable
            - runtime_secret_transport_insecure
            - runtime_secret_lease_active
            - runtime_secret_not_allowed
            - bad_request
            - not_found
            - permission_denied
            - caption_profile_saved_runtime_apply_failed
            - get_stream_failed
            - discord_youtube_notification_enqueue_failed
            - discord_youtube_notification_outbox_unavailable
            - discord_youtube_notification_not_found
            - get_discord_youtube_notification_failed
            - duplicate_risk_acknowledgement_required
            - discord_youtube_notification_recovery_not_allowed
            - discord_youtube_notification_recovery_already_requested
            - recover_discord_youtube_notification_failed
            - invalid_email_notification
            - missing_service_token
            - invalid_service_token
            - missing_service_scope
            - service_type_not_allowed
            - service_token_not_registered
            - service_or_stream_not_found
            - service_not_registered
            - stream_not_found
            - stream_artifact_store_failed
            - stream_artifact_store_unavailable
            - rate_limited
            - list_services_failed
            - app_settings_failed
            - secret_encryption_key_required
            - smtp_not_configured
            - smtp_requires_tls
            - smtp_dial_failed
            - smtp_starttls_failed
            - smtp_auth_failed
            - smtp_from_rejected
            - smtp_recipient_rejected
            - smtp_data_failed
            - smtp_write_failed
            - smtp_close_failed
            - send_failed
        message:
          type: string
    User:
      type: object
      additionalProperties: false
      required: [ id, username, email, status, roles, role_ids ]
      properties:
        id:
          type: string
        username:
          type: string
        email:
          type: string
        status:
          type: string
        roles:
          type: array
          items:
            type: string
          description: Operator-facing role names.
        role_ids:
          type: array
          items:
            type: string
          description: Stable role IDs used to initialize the administrative user edit form.
        last_login_at:
          type: [ string, "null" ]
          format: date-time
        last_login_ip:
          type: string
    UserCreateRequest:
      type: object
      additionalProperties: false
      required: [ username, email, temporary_password ]
      properties:
        username:
          type: string
          minLength: 1
        email:
          type: string
          format: email
        temporary_password:
          type: string
          format: password
        role_ids:
          type: array
          items:
            type: string
        send_welcome_email:
          type: boolean
    UserUpdateRequest:
      type: object
      additionalProperties: false
      properties:
        username:
          type: string
          minLength: 1
        email:
          type: string
          format: email
        role_ids:
          type: array
          items:
            type: string
          description: Omit to preserve current role assignments. Updating this field requires roles.assign and cannot target the current user's own roles.
    PasskeyCredential:
      type: object
      description: Public WebAuthn credential metadata. Raw credential ID and public key material are never returned.
      required: [ id, user_id, name, credential_id_hash, sign_count ]
      properties:
        id:
          type: string
        user_id:
          type: string
        name:
          type: string
        credential_id_hash:
          type: string
        sign_count:
          type: integer
          format: int64
        transports:
          type: array
          items:
            type: string
        aaguid:
          type: string
        backup_eligible:
          type: boolean
        backed_up:
          type: boolean
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
        last_used_at:
          type: string
          format: date-time
    PasskeyRegistrationStartResponse:
      type: object
      required: [ registration_token, expires_at, public_key ]
      properties:
        registration_token:
          type: string
          description: Short-lived one-time token returned only to the current authenticated user.
        expires_at:
          type: string
          format: date-time
        public_key:
          type: object
          required: [ challenge, rp, user, pubKeyCredParams, timeout, attestation, authenticatorSelection ]
          properties:
            challenge:
              type: string
            rp:
              type: object
              required: [ id, name ]
              properties:
                id:
                  type: string
                name:
                  type: string
            user:
              type: object
              required: [ id, name, displayName ]
              properties:
                id:
                  type: string
                name:
                  type: string
                displayName:
                  type: string
            pubKeyCredParams:
              type: array
              items:
                type: object
                required: [ type, alg ]
                properties:
                  type:
                    const: public-key
                  alg:
                    type: integer
            timeout:
              type: integer
            attestation:
              type: string
            authenticatorSelection:
              type: object
              required: [ residentKey, userVerification ]
              properties:
                residentKey:
                  type: string
                userVerification:
                  type: string
    PasskeyRegistrationFinishRequest:
      type: object
      additionalProperties: false
      required: [ registration_token, credential ]
      properties:
        registration_token:
          type: string
          description: One-time registration token returned by /auth/passkeys/register/start.
        name:
          type: string
          maxLength: 255
          description: Optional operator-visible passkey label.
        credential:
          type: object
          additionalProperties: true
          description: WebAuthn attestation response from navigator.credentials.create().
    PasskeyLoginStartRequest:
      type: object
      additionalProperties: false
      properties:
        username:
          type: string
          description: Optional username hint. Empty requests create a discoverable login challenge without enumerating users.
    PasskeyLoginStartResponse:
      type: object
      required: [ challenge_token, expires_at, public_key ]
      properties:
        challenge_token:
          type: string
          description: Short-lived one-time token returned only to the browser starting the login ceremony.
        expires_at:
          type: string
          format: date-time
        public_key:
          type: object
          additionalProperties: true
          description: PublicKeyCredentialRequestOptions for navigator.credentials.get().
    PasskeyLoginFinishRequest:
      type: object
      additionalProperties: false
      required: [ challenge_token, credential ]
      properties:
        challenge_token:
          type: string
          description: One-time login token returned by /auth/passkeys/login/start.
        credential:
          type: object
          additionalProperties: true
          description: WebAuthn assertion response from navigator.credentials.get().
    SessionRefreshResponse:
      type: object
      additionalProperties: false
      required: [ status, idle_expires_at, absolute_expires_at ]
      properties:
        status:
          type: string
          enum: [ refreshed ]
        idle_expires_at:
          type: string
          format: date-time
        absolute_expires_at:
          type: string
          format: date-time
          description: Hard session expiry. Activity refresh never moves this timestamp.
