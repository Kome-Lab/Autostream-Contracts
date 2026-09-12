  /nodes/action-permissions:
    get:
      operationId: getNodeActionPermissionProjection
      summary: Project the current caller's permission for one Node credential action
      x-autostream-required-permission: api_tokens.create
      description: >
        Read-only, audit-free projection that reuses the Node credential mutation
        authority. The response never returns a Node, service-token, credential, raw
        scope, or private endpoint identifier. evaluated_at is diagnostic only;
        projection_revision is the opaque authority value used by the UI for
        pre-submit revalidation.
      parameters:
        - name: action
          in: query
          required: true
          schema:
            type: string
            enum: [ registration_token, configure_token_regenerate, runtime_token_rotate ]
        - name: node_type
          in: query
          required: false
          schema:
            type: string
            minLength: 1
            maxLength: 64
            pattern: '^[a-z][a-z0-9_]*$'
        - name: node_id
          in: query
          required: false
          schema:
            type: string
            minLength: 1
            maxLength: 128
            pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        - name: allow_runtime_secrets
          in: query
          required: false
          schema: { type: boolean, default: false }
        - name: allow_remediation
          in: query
          required: false
          schema: { type: boolean, default: false }
      responses:
        "200":
          description: strict permission projection
          headers:
            Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
            Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
            Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
            Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
            X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
          content:
            application/json:
              schema: { $ref: '../schemas/node-action-permission-projection.schema.json' }
        "400": { $ref: '#/components/responses/nodeProjectionBadRequest' }
        "401": { $ref: '#/components/responses/nodeProjectionUnauthorized' }
        "403": { $ref: '#/components/responses/nodeProjectionForbidden' }
        "404": { $ref: '#/components/responses/nodeProjectionNotFound' }
        "500": { $ref: '#/components/responses/nodeProjectionInternalError' }
        "503": { $ref: '#/components/responses/nodeProjectionUnavailable' }
        "426": { $ref: '#/components/responses/nodeProjectionContractMajorUnsupported' }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /api-tokens:
    get:
      responses:
        "200":
          description: list service/API tokens without raw token values
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/ServiceToken"
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
              $ref: "#/components/schemas/ServiceTokenCreateRequest"
      responses:
        "201":
          description: create token and show raw token once. If service_id is present, a pending service registry entry is also created.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ServiceToken"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: service_id already exists when precreating a service registry entry
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
  /api-tokens/{id}:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "201":
          description: revoke the old token, create a replacement token, rebind linked services, and show the new raw token once
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ServiceToken"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: caller lacks api_tokens.revoke
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: token not found or already revoked
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: token revoked
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
