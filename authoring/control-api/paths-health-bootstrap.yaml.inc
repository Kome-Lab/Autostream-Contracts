  /health:
    get:
      security: []
      responses:
        "200":
          description: healthy
      x-autostream-contract-major-exempt: health_non_json
  /updater/version:
    get:
      operationId: getUpdaterVersion
      summary: Read the Control Panel application runtime identity
      description: >
        Historical route name retained without introducing a /v2 URI prefix. This is an application-owned runtime identity probe consumed by the Local Executor during update verification; it is not Updater health. The current source has no explicit route-level no-store behavior. A later consumer implementation must emit the exact Cache-Control no-store response declared here.
      x-autostream-semantic-name: application_runtime_identity_probe
      x-autostream-current-source-cache-control: no_explicit_route_level_no_store
      x-autostream-v2-target-cache-control: no-store
      security: []
      responses:
        "200":
          description: Control Panel runtime identity
          headers:
            Cache-Control:
              required: true
              schema: { type: string, const: no-store }
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [ version, service_id, service_type, config_revision ]
                allOf:
                  - required: [ version ]
                properties:
                  version:
                    type: string
                    pattern: '^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$'
                    example: v2.0.0
                  service_id:
                    type: string
                    minLength: 1
                    maxLength: 128
                    pattern: '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
                  service_type:
                    type: string
                    const: control_panel
                  config_revision:
                    type: integer
                    minimum: 1
        "500":
          description: Control Panel runtime identity is unavailable; the response is safe and contains no raw configuration
          headers:
            Cache-Control:
              required: true
              schema: { type: string, const: no-store }
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [ code, message ]
                properties:
                  code: { type: string, enum: [ application_probe_unavailable, application_probe_identity_invalid, application_probe_revision_unknown ] }
                  message: { type: string, maxLength: 240 }
        "405":
          description: method not allowed
          headers:
            Cache-Control:
              required: true
              schema: { type: string, const: no-store }
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

  /setup/first-admin:
    post:
      security: []
      responses:
        "201":
          description: first admin created
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
