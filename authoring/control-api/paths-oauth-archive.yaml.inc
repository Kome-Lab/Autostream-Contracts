  /integrations/oauth-providers:
    get:
      responses:
        "200":
          description: list OAuth provider configs without raw client secrets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/OAuthProvider"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthProviderWriteRequest"
      responses:
        "201":
          description: OAuth provider created
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /integrations/oauth-providers/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: OAuth provider detail without raw client secret
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthProviderWriteRequest"
      responses:
        "200":
          description: OAuth provider updated
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: OAuth provider deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: oauth_provider_in_use when connected accounts or login user links still reference this provider
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /integrations/oauth-accounts:
    get:
      responses:
        "200":
          description: list connected OAuth accounts without raw refresh tokens
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/OAuthAccount"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /integrations/oauth-accounts/start:
    post:
      summary: Start OAuth consent for a connected operational account
      description: >
        Returns a one-time authorization URL. Supplying oauth_account_id starts a re-link flow that preserves the existing account ID; provider_id is then derived when omitted. Raw client secrets and refresh tokens are never returned.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthAccountConnectionStartRequest"
      responses:
        "200":
          description: OAuth consent URL
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OAuthAccountConnectionStartResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: Invalid request or unsupported account_purpose
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: Missing integrations.create or integrations.update permission
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: Referenced OAuth account or provider not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: Referenced OAuth account/provider mismatch or provider cannot be used for connected accounts
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: OAuth provider or one-time state lookup/create failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /integrations/oauth-accounts/callback:
    get:
      summary: Browser OAuth callback for connected accounts
      description: Consumes state/code, stores the refresh token encrypted, then redirects to a safe local path.
      parameters:
        - name: state
          in: query
          required: true
          schema: { type: string }
        - name: code
          in: query
          required: true
          schema: { type: string }
        - name: provider_id
          in: query
          required: false
          schema: { type: string }
      responses:
        "303":
          description: Connected account created and browser redirected
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    post:
      summary: API OAuth callback for connected accounts
      description: >
        Consumes state/code and returns public connected-account metadata only. A newly created account returns 201; a re-linked account keeps its account ID and returns 200. The returned OAuthAccount carries account_purpose and scopes.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthAccountConnectionCallbackRequest"
      responses:
        "200":
          description: Connected account relinked while retaining its account ID
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OAuthAccount"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "201":
          description: Connected account created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OAuthAccount"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: Invalid callback request or connected-account persistence failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: Invalid, expired, or browser-state-mismatched OAuth state/code
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: Missing required permission or verified OAuth identity is not allowed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: Target OAuth account no longer exists during a re-link
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: Provider, identity, redirect, scope, or refresh-token condition prevents completion
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: OAuth state/account persistence failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: Encryption material required to persist the refresh token is unavailable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /integrations/oauth-accounts/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: OAuth account detail without raw refresh token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/OAuthAccountLabelUpdateRequest"
      responses:
        "200":
          description: OAuth account label updated without changing identity or refresh token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: manual OAuth refresh token update is disabled
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: OAuth account deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: oauth_account_in_use when Drive destinations, YouTube outputs, or active YouTube runtime metadata still reference this account
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /archive/destinations:
    get:
      responses:
        "200":
          description: list Drive archive destinations without raw folder IDs
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/DriveDestination"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/DriveDestinationWriteRequest"
      responses:
        "201":
          description: Drive archive destination created
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /archive/destinations/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Drive archive destination detail without raw folder ID
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/DriveDestinationWriteRequest"
      responses:
        "200":
          description: Drive archive destination updated
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Drive archive destination deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
