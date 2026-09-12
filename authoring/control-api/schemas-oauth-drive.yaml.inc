    OAuthLoginProvider:
      type: object
      additionalProperties: false
      required: [ id, provider_type, name, scopes, redirect_uri ]
      properties:
        id: { type: string }
        provider_type:
          enum: [ google, github, discord ]
        name: { type: string }
        scopes:
          type: array
          items: { type: string }
        redirect_uri:
          type: string
          format: uri
    OAuthLoginStartRequest:
      type: object
      additionalProperties: false
      properties:
        redirect_after:
          type: string
          description: same-site relative path to return to after login
    OAuthLoginStartResponse:
      type: object
      additionalProperties: false
      required: [ provider, authorization_url, state, nonce, expires_at ]
      properties:
        provider:
          $ref: "#/components/schemas/OAuthLoginProvider"
        authorization_url:
          type: string
          format: uri
        state:
          type: string
          description: one-time OAuth state token; stored hashed server-side
        nonce:
          type: string
          description: one-time nonce for OIDC-capable providers
        expires_at:
          type: string
          format: date-time
    OAuthLoginCallbackRequest:
      type: object
      additionalProperties: false
      required: [ state, code ]
      properties:
        provider_id: { type: string }
        state: { type: string }
        code:
          type: string
          writeOnly: true
    OAuthUserLink:
      type: object
      additionalProperties: false
      required: [ id, user_id, provider_id, provider_type, subject, created_at, updated_at ]
      properties:
        id: { type: string }
        user_id: { type: string }
        provider_id: { type: string }
        provider_type:
          enum: [ google, github, discord ]
        subject: { type: string }
        email: { type: string }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    OAuthProvider:
      type: object
      additionalProperties: false
      required: [ id, provider_type, name, enabled, client_id, client_secret_configured, scopes, allowed_domains, auto_provision, redirect_uri, created_at, updated_at ]
      properties:
        id: { type: string }
        provider_type:
          enum: [ google, github, discord ]
        name: { type: string }
        enabled: { type: boolean }
        client_id: { type: string }
        client_secret_configured: { type: boolean }
        scopes:
          type: array
          items: { type: string }
        allowed_domains:
          type: array
          items: { type: string }
        auto_provision:
          type: boolean
          description: When true, an allowed but unlinked OAuth identity may create a local user with default_role_ids on first login.
        default_role_ids:
          type: array
          items: { type: string }
          description: Role IDs assigned to auto-provisioned users. Raw permissions are resolved server-side.
        redirect_uri:
          type: string
          format: uri
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    OAuthProviderWriteRequest:
      type: object
      additionalProperties: false
      description: >
        OAuth provider configuration. Login providers use /auth/oauth/callback. Google connected accounts for Drive/YouTube use /integrations/oauth-accounts/callback; that path requires provider_type=google, a Drive or YouTube API scope, and no auto_provision/default_role_ids.
      required: [ provider_type, name, enabled, client_id, redirect_uri ]
      properties:
        provider_type:
          enum: [ google, github, discord ]
        name: { type: string }
        enabled: { type: boolean }
        client_id: { type: string }
        client_secret:
          type: string
          writeOnly: true
          description: encrypted at rest; never returned by API
        scopes:
          type: array
          items: { type: string }
        allowed_domains:
          type: array
          items: { type: string }
        auto_provision:
          type: boolean
          description: Enable first-login user creation for allowed OAuth identities.
        default_role_ids:
          type: array
          items: { type: string }
          description: Required when auto_provision is true. Server requires roles.assign for this field.
        redirect_uri:
          type: string
          format: uri
          description: Must match AUTOSTREAM_PUBLIC_URL scheme/host in production.
    OAuthAccount:
      type: object
      additionalProperties: false
      required: [ id, provider_id, provider_type, account_label, account_purpose, scopes, refresh_token_configured, created_at, updated_at ]
      properties:
        id: { type: string }
        provider_id: { type: string }
        provider_type:
          enum: [ google, github, discord ]
        provider_name:
          type: string
          description: Configured OAuth provider name used to build a legacy display fallback.
        account_label: { type: string }
        account_purpose:
          type: string
          enum: [ drive, youtube, drive_youtube, unknown ]
          description: Effective connected-account capability derived from the returned scopes. unknown is returned only when no Drive or YouTube capability is present.
        display_name:
          type: string
          description: Resolved operator-facing name. A configured account label takes precedence; legacy rows use the provider name plus a stable short account reference so multiple accounts remain distinguishable.
        subject: { type: string }
        email: { type: string }
        scopes:
          type: array
          items: { type: string }
        refresh_token_configured: { type: boolean }
        token_fingerprint:
          type: string
          description: short fingerprint for change tracking, not a secret value
        refresh_token_updated_at:
          type: string
          description: RFC3339 timestamp when the stored OAuth refresh token was first registered or replaced; legacy accounts without recorded history return an empty string
        access_token_refreshed_at:
          type: string
          description: RFC3339 timestamp when the short-lived OAuth access token was last refreshed automatically; the access token itself is never returned or persisted
        access_token_refresh_attempted_at:
          type: string
          description: RFC3339 timestamp when automatic access-token refresh was last attempted; legacy accounts without recorded history return an empty string
        access_token_refresh_failed_at:
          type: string
          description: RFC3339 timestamp of the active automatic refresh failure; empty after a successful refresh or when no failure is recorded
        access_token_refresh_failure_code:
          type: string
          enum: [ "", unknown, credentials_unavailable, provider_not_ready, provider_unavailable, provider_credentials_invalid, reauthorization_required, timeout, invalid_response ]
          description: Bounded non-secret operational failure class. Provider response bodies, error descriptions, and tokens are never returned.
        access_token_refresh_relink_required:
          type: boolean
          description: True only when the recorded automatic refresh failure requires reconnecting this existing OAuth account; reconnecting retains the account ID and its references.
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    OAuthAccountLabelUpdateRequest:
      type: object
      additionalProperties: false
      required: [ account_label ]
      properties:
        account_label:
          type: string
          description: Operator-facing label. Provider identity, scopes, and refresh token remain immutable outside OAuth callback.
    OAuthAccountConnectionStartRequest:
      type: object
      additionalProperties: false
      anyOf:
        - required: [ provider_id ]
        - required: [ oauth_account_id ]
      properties:
        provider_id:
          type: string
          minLength: 1
          description: OAuth provider configured in the Integration Registry
        oauth_account_id:
          type: string
          minLength: 1
          description: Existing connected-account ID to re-link. The server derives provider_id when it is omitted.
        account_label:
          type: string
          description: optional operator label; callback may derive a label from the verified email
        account_purpose:
          type: string
          description: Optional requested connected-account capability. Canonical values are drive, youtube, and drive_youtube; documented compatibility aliases are accepted and normalized by the Control Panel.
        redirect_after:
          type: string
          description: safe local path to redirect to after browser callback
    OAuthAccountConnectionStartResponse:
      $ref: "../schemas/oauth-account-connection-start-response.schema.json"
    OAuthAccountConnectionCallbackRequest:
      type: object
      additionalProperties: false
      required: [ state, code ]
      properties:
        provider_id:
          type: string
          description: optional explicit provider id; must match state when supplied
        state:
          type: string
        code:
          type: string
        account_label:
          type: string
    DriveDestination:
      type: object
      additionalProperties: false
      required: [ id, name, auth_mode, folder_id_configured, shared_drive, created_at, updated_at ]
      properties:
        id: { type: string }
        name: { type: string }
        auth_mode:
          enum: [ oauth2 ]
        oauth_account_id: { type: string }
        folder_id_configured: { type: boolean }
        folder_id_fingerprint:
          type: string
          description: short fingerprint for change tracking, not the raw folder ID
        masked_folder_id: { type: string }
        shared_drive:
          type: boolean
          description: use supportsAllDrives=true when uploading
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    DriveDestinationWriteRequest:
      type: object
      additionalProperties: false
      required: [ name, auth_mode, oauth_account_id ]
      properties:
        name: { type: string }
        auth_mode:
          enum: [ oauth2 ]
        oauth_account_id:
          type: string
          minLength: 1
          description: Required for oauth2 destinations and must reference a Google connected account with Drive scope.
        folder_id:
          type: string
          writeOnly: true
          description: required when creating a destination, optional when updating an existing destination, encrypted at rest, and never returned by API
        shared_drive:
          type: boolean
