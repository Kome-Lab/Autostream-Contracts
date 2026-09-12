  /auth/login:
    post:
      security: []
      responses:
        "200":
          description: login succeeded
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/oauth/providers:
    get:
      security: []
      responses:
        "200":
          description: enabled OAuth login providers without client secrets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/OAuthLoginProvider"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/oauth/{id}/start:
    post:
      security: []
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthLoginStartRequest"
      responses:
        "200":
          description: OAuth authorization URL and one-time state
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OAuthLoginStartResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/oauth/callback:
    post:
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthLoginCallbackRequest"
      responses:
        "200":
          description: OAuth login succeeded and session cookie was issued
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "202":
          description: MFA challenge required
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: OAuth identity is not linked to a local user or not allowed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/logout:
    post:
      responses:
        "200":
          description: logout succeeded
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/session/refresh:
    post:
      description: Explicitly extends the idle expiry for an authenticated active browser session without extending its absolute lifetime.
      parameters:
        - $ref: "#/components/parameters/csrfToken"
      responses:
        "200":
          description: session idle expiry refreshed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SessionRefreshResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: session is missing, idle-expired, or absolute-expired
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/me:
    get:
      responses:
        "200":
          description: current user
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/change-password:
    post:
      responses:
        "200":
          description: password changed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/mfa/enroll:
    post:
      responses:
        "202":
          description: TOTP enrollment started
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/mfa/verify:
    post:
      responses:
        "200":
          description: MFA verified
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/mfa/disable:
    post:
      responses:
        "200":
          description: MFA disabled
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/recovery-codes/regenerate:
    post:
      responses:
        "200":
          description: recovery codes regenerated
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys/register/start:
    post:
      parameters:
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              properties:
                display_name:
                  type: string
                  maxLength: 255
      responses:
        "201":
          description: Passkey registration challenge created. Response must be sent with Cache-Control no-store.
          headers:
            Cache-Control:
              schema:
                type: string
              description: no-store
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PasskeyRegistrationStartResponse"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys/register/finish:
    post:
      parameters:
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PasskeyRegistrationFinishRequest"
      responses:
        "201":
          description: Passkey credential registered for the current authenticated user
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PasskeyCredential"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: registration challenge is invalid, expired, or verification failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys/login/start:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PasskeyLoginStartRequest"
      responses:
        "200":
          description: Passkey login challenge created. Response must be sent with Cache-Control no-store and must not enumerate users.
          headers:
            Cache-Control:
              schema:
                type: string
              description: no-store
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PasskeyLoginStartResponse"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys/login/finish:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PasskeyLoginFinishRequest"
      responses:
        "200":
          description: Passkey login succeeded and session cookie was issued
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "202":
          description: Passkey assertion succeeded but TOTP MFA challenge is still required
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: challenge is invalid, expired, or assertion verification failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: MFA enrollment is required before login can continue
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys:
    get:
      responses:
        "200":
          description: current user's public passkey credentials
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/PasskeyCredential"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /auth/passkeys/{id}:
    delete:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - $ref: "#/components/parameters/csrfToken"
      responses:
        "204":
          description: passkey deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: passkey not found for current user
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
