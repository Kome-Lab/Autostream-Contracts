    PublicAppSettings:
      type: object
      additionalProperties: false
      required: [ app_name, timezone ]
      description: Public browser bootstrap settings. SMTP and all secret values are excluded.
      properties:
        app_name:
          type: string
          minLength: 1
          maxLength: 80
        timezone:
          type: string
          minLength: 1
          maxLength: 64
          description: IANA time zone name used by the Control Panel UI.
        turnstile_enabled:
          type: boolean
        turnstile_site_key:
          type: string
          maxLength: 255
        turnstile_configured:
          type: boolean
          description: Configured status only. The Turnstile secret is never returned.
        google_analytics_enabled:
          type: boolean
        google_analytics_measurement_id:
          type: string
          pattern: "^G-[A-Z0-9]{4,22}$"
          description: Public GA4 measurement ID, present only when analytics is enabled and valid.
        updated_at:
          type: string
          format: date-time
    ManagedAppSettings:
      type: object
      additionalProperties: false
      required: [ app_name, timezone, smtp_enabled, smtp_starttls ]
      description: Administrator application settings. Secret values are represented only by configured status.
      properties:
        app_name:
          type: string
          minLength: 1
          maxLength: 80
        timezone:
          type: string
          minLength: 1
          maxLength: 64
        google_analytics_enabled:
          type: boolean
        google_analytics_measurement_id:
          type: string
          pattern: "^G-[A-Z0-9]{4,22}$"
        smtp_enabled:
          type: boolean
        smtp_host:
          type: string
          maxLength: 255
        smtp_port:
          type: integer
          minimum: 1
          maximum: 65535
        smtp_starttls:
          type: boolean
        smtp_from:
          type: string
          maxLength: 512
        smtp_username:
          type: string
          maxLength: 255
        smtp_password_configured:
          type: boolean
          description: Configured status only. The SMTP password is never returned.
        turnstile_enabled:
          type: boolean
        turnstile_site_key:
          type: string
          maxLength: 255
        turnstile_configured:
          type: boolean
          description: Configured status only. The Turnstile secret is never returned.
        updated_at:
          type: string
          format: date-time
    AppSettingsWriteRequest:
      type: object
      additionalProperties: false
      required: [ app_name, timezone ]
      allOf:
        - if:
            properties:
              google_analytics_enabled:
                const: true
            required: [ google_analytics_enabled ]
          then:
            required: [ google_analytics_measurement_id ]
      properties:
        app_name:
          type: string
          minLength: 1
          maxLength: 80
        timezone:
          type: string
          minLength: 1
          maxLength: 64
        google_analytics_enabled:
          type: boolean
        google_analytics_measurement_id:
          type: string
          pattern: "^[Gg]-[A-Za-z0-9]{4,22}$"
          description: Control Panel normalizes accepted GA4 IDs to uppercase and clears the value when analytics is disabled.
        smtp_enabled:
          type: boolean
        smtp_host:
          type: string
          maxLength: 255
        smtp_port:
          type: integer
          minimum: 1
          maximum: 65535
        smtp_starttls:
          type: boolean
        smtp_from:
          type: string
          maxLength: 512
        smtp_username:
          type: string
          maxLength: 255
        smtp_password:
          type: string
          writeOnly: true
          description: Write-only SMTP password. It is encrypted at rest and never returned by API, UI, logs, or audit metadata.
        turnstile_enabled:
          type: boolean
        turnstile_site_key:
          type: string
          maxLength: 255
        turnstile_secret:
          type: string
          writeOnly: true
          description: Write-only Turnstile secret. It is never returned by API, UI, logs, or audit metadata.
    StreamPreviewLink:
      type: object
      additionalProperties: false
      required: [ stream_id, url, expires_at ]
      description: The URL is a bearer capability and must not be written to access logs, tickets, chat, screenshots, or audit metadata.
      properties:
        stream_id:
          type: string
          minLength: 1
        url:
          type: string
          format: uri-reference
          description: Signed index.m3u8 URL or same-origin path. The web client resolves a same-origin path before copying it to an external player. It expires after 12 hours at most and is invalidated when the stream stops.
        expires_at:
          type: string
          format: date-time
    SecuritySettings:
      type: object
      required: [ password_min_length, password_hash, login_lockout_threshold, session_idle_timeout_min, session_absolute_lifetime_h, remember_me_enabled, mfa_mode ]
      additionalProperties: false
      properties:
        password_min_length:
          type: integer
          minimum: 8
        password_hash:
          enum: [ argon2id ]
        login_lockout_threshold:
          type: integer
          minimum: 3
        session_idle_timeout_min:
          type: integer
          minimum: 5
        session_absolute_lifetime_h:
          type: integer
          minimum: 1
        remember_me_enabled:
          type: boolean
          const: false
        mfa_mode:
          enum: [ disabled, totp, passkey ]
        mfa_required_roles:
          type: array
          description: Empty means an enabled mfa_mode applies to all users. Otherwise only users with one of these roles must complete that MFA policy.
          items:
            type: string
        mfa_supported_methods:
          type: array
          readOnly: true
          items:
            enum: [ totp, passkey ]
        passkey_status:
          enum: [ planned, credential_store_ready, available ]
          readOnly: true
          description: available means WebAuthn registration, passwordless login, and passkey-required login policy are available.
        updated_at:
          type: string
          format: date-time
