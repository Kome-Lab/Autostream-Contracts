  /services/runtime-config:
    get:
      security:
        - bearerToken: []
      description: Service-token endpoint for a registered service to fetch only its own assignments and Control Panel-managed non-secret runtime profiles. Requires service.config.read.
      parameters:
        - name: service_id
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: runtime config for the authenticated service
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store because runtime config includes assignments and runtime secret reference names.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ServiceRuntimeConfig"
        "403":
          description: token does not own the requested service_id or lacks service.config.read
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/runtime-secrets/resolve:
    post:
      security:
        - bearerToken: []
      description: Service-token endpoint for resolving a short-lived raw runtime secret that is explicitly referenced by the authenticated service runtime config or an assigned stream archive runtime config. Requires service.secret.resolve. A server-side lease prevents the same service/context/secret from being resolved repeatedly during the returned expires_in_sec window. This endpoint is not for browser UI use and responses must not be cached, logged, audited with raw values, or exposed to the frontend.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceRuntimeSecretResolveRequest"
      responses:
        "200":
          description: one-time runtime secret for the authenticated service
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ServiceRuntimeSecretResolveResponse"
        "403":
          description: token does not own service_id, lacks service.secret.resolve, or secret_name is not referenced by that service runtime config
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: service or configured secret was not found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: runtime secret lease is still active for the same service/context/secret
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/notifications/email:
    post:
      summary: Send a notification email with global SMTP
      operationId: sendServiceNotificationEmail
      security:
        - bearerToken: []
      description: Delivers a required plain-text notification and an optional HTML alternative with the globally managed application SMTP settings. This service endpoint requires the dedicated notifications.email.send scope, a registered non-pending service owned by the token, and service_type=observability. SMTP configuration and credentials are loaded only inside Control Panel and are never copied into notification channels, runtime config, responses, audit metadata, or logs.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceNotificationEmailRequest"
      responses:
        "202":
          description: the configured SMTP server accepted the message delivery attempt
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ServiceNotificationEmailResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: bad_request or invalid_email_notification; recipients, subject, text, or optional HTML violated the size, bare-mailbox, or control-character contract
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing_service_token or invalid_service_token
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: missing_service_scope, service_type_not_allowed, or service_token_not_registered
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: smtp_not_configured or smtp_requires_tls
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "429":
          description: rate_limited
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: list_services_failed or app_settings_failed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: smtp_dial_failed, smtp_starttls_failed, smtp_auth_failed, smtp_from_rejected, smtp_recipient_rejected, smtp_data_failed, smtp_write_failed, smtp_close_failed, or send_failed; the response never includes raw SMTP failure details, credentials, or recipient addresses
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: secret_encryption_key_required
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
