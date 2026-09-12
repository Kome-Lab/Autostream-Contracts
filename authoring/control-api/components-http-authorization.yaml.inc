components:
  securitySchemes:
    cookieSession:
      type: apiKey
      in: cookie
      name: autostream_session
    bearerToken:
      type: http
      scheme: bearer
  parameters:
    contractMajor:
      name: X-AutoStream-Contract-Major
      in: header
      required: true
      description: Required exact clean-break contract major for every non-exempt JSON or multipart API operation.
      schema: { type: string, const: "2" }
    id:
      name: id
      in: path
      required: true
      schema:
        type: string
    notificationId:
      name: notification_id
      in: path
      required: true
      schema:
        type: string
    limit:
      name: limit
      in: query
      required: false
      schema:
        type: integer
        minimum: 1
        maximum: 500
    csrfToken:
      name: X-CSRF-Token
      in: header
      required: true
      description: Required for unsafe cookie-authenticated requests.
      schema:
        type: string
    previewAssetName:
      name: name
      in: path
      required: true
      description: HLS playlist or six-digit MPEG-TS segment name.
      schema:
        type: string
        pattern: "^(index\\.m3u8|segment-[0-9]{6}\\.ts)$"
    auditActionGroup:
      name: action_group
      in: query
      required: false
      schema:
        type: string
        enum: [ all, service_assignment, service_runtime, stream_lifecycle, security, secrets, notifications ]
    auditAction:
      name: action
      in: query
      required: false
      description: comma-separated audit action names
      schema:
        type: string
    auditResult:
      name: result
      in: query
      required: false
      schema:
        type: string
        enum: [ success, failure ]
    auditQuery:
      name: q
      in: query
      required: false
      schema:
        type: string
  headers:
    contractMajor:
      required: true
      description: Exact contract major echoed by every non-exempt API response.
      schema: { type: string, const: "2" }
    noStore:
      required: true
      description: Identity probes must never be cached.
      schema: { type: string, const: no-store }
    nodeProjectionJSONContentType:
      required: true
      schema: { type: string, const: application/json }
    nodeProjectionCacheControl:
      required: true
      schema: { type: string, const: 'no-store, no-cache' }
    nodeProjectionPragma:
      required: true
      schema: { type: string, const: no-cache }
    nodeProjectionReferrerPolicy:
      required: true
      schema: { type: string, const: no-referrer }
  responses:
    contractMajorUnsupported:
      description: The request omitted contract major 2 or supplied an unsupported major.
      headers:
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema: { $ref: '#/components/schemas/V2ContractError' }
    nodeProjectionBadRequest:
      description: invalid Node action projection query
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code: { type: string, const: invalid_node_action_projection_request }
    nodeProjectionUnauthorized:
      description: unauthenticated
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code: { type: string, const: unauthorized }
    nodeProjectionForbidden:
      description: password change required or caller lacks api_tokens.create
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code: { type: string, enum: [ password_change_required, permission_denied ] }
    nodeProjectionNotFound:
      description: Node or current Node Runtime Token not found
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code: { type: string, enum: [ node_not_found, runtime_token_not_found ] }
    nodeProjectionInternalError:
      description: Node or permission authority could not be read
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code:
                type: string
                enum: [ get_node_failed, list_service_tokens_failed, check_system_update_emergency_recovery_failed ]
    nodeProjectionUnavailable:
      description: updater identity fence authority is unavailable
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            required: [ code ]
            properties:
              code: { type: string, const: system_update_identity_fence_unavailable }
    nodeProjectionContractMajorUnsupported:
      description: exact contract major 2 is required
      headers:
        Content-Type: { $ref: '#/components/headers/nodeProjectionJSONContentType' }
        Cache-Control: { $ref: '#/components/headers/nodeProjectionCacheControl' }
        Pragma: { $ref: '#/components/headers/nodeProjectionPragma' }
        Referrer-Policy: { $ref: '#/components/headers/nodeProjectionReferrerPolicy' }
        X-AutoStream-Contract-Major: { $ref: '#/components/headers/contractMajor' }
      content:
        application/json:
          schema: { $ref: '#/components/schemas/V2ContractError' }
  pathItems:
    WorkerApplicationRuntimeIdentityProbe:
      get:
        summary: Read the Worker application runtime identity
        description: Standard reusable Worker-owned route contract consumed by Local Executor. It is not Updater health and does not create a Control Panel route.
        x-autostream-semantic-name: application_runtime_identity_probe
        x-autostream-current-source-cache-control: no_explicit_route_level_no_store
        x-autostream-v2-target-cache-control: no-store
        security: []
        responses:
          "200":
            description: Worker runtime identity
            headers:
              Cache-Control: { $ref: '#/components/headers/noStore' }
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
            content:
              application/json:
                schema: { $ref: '#/components/schemas/WorkerApplicationRuntimeIdentityProbe' }
          "503":
            description: Worker runtime identity is unavailable
            headers:
              Cache-Control: { $ref: '#/components/headers/noStore' }
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
            content:
              application/json:
                schema: { $ref: '#/components/schemas/WorkerApplicationRuntimeIdentityProbeSafeError' }
          "405":
            description: method not allowed
            headers:
              Cache-Control: { $ref: '#/components/headers/noStore' }
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
            content:
              application/json:
                schema:
                  type: object
                  additionalProperties: false
                  required: [ code, message ]
                  properties:
                    code: { type: string, const: method_not_allowed }
                    message: { type: string, maxLength: 240 }
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
      parameters:
        - $ref: "#/components/parameters/contractMajor"
    listCreateProfile:
      get:
        responses:
          "200":
            description: list resources
            headers:
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
      post:
        responses:
          "201":
            description: create resource
            headers:
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
    itemProfile:
      get:
        parameters: [ { $ref: "#/components/parameters/id" } ]
        responses:
          "200":
            description: resource detail
            headers:
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
      put:
        parameters: [ { $ref: "#/components/parameters/id" } ]
        responses:
          "200":
            description: resource updated
            headers:
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
      delete:
        parameters: [ { $ref: "#/components/parameters/id" } ]
        responses:
          "200":
            description: resource deleted
            headers:
              X-AutoStream-Contract-Major:
                $ref: "#/components/headers/contractMajor"
          "426":
            $ref: "#/components/responses/contractMajorUnsupported"
  schemas:
