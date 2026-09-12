  /secrets/status:
    get:
      responses:
        "200":
          description: secret configured status only
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /secrets/{name}:
    put:
      parameters:
        - name: name
          in: path
          required: true
          schema: { type: string }
      responses:
        "200":
          description: secret value updated without returning raw value
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /audit-logs:
    get:
      parameters:
        - $ref: "#/components/parameters/auditActionGroup"
        - $ref: "#/components/parameters/auditAction"
        - $ref: "#/components/parameters/auditResult"
        - $ref: "#/components/parameters/auditQuery"
        - $ref: "#/components/parameters/limit"
      responses:
        "200":
          description: audit log list
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /audit-logs/export:
    get:
      parameters:
        - $ref: "#/components/parameters/auditActionGroup"
        - $ref: "#/components/parameters/auditAction"
        - $ref: "#/components/parameters/auditResult"
        - $ref: "#/components/parameters/auditQuery"
        - $ref: "#/components/parameters/limit"
      responses:
        "200":
          description: audit log export
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /security/settings:
    get:
      responses:
        "200":
          description: security settings
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SecuritySettings"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SecuritySettings"
      responses:
        "200":
          description: security settings updated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SecuritySettings"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /settings/app:
    get:
      security: []
      summary: Read public application settings
      description: >
        Returns only browser bootstrap settings: application name, time zone, public Turnstile fields/configured status, and Google Analytics fields. SMTP settings and all secret values are excluded.
      responses:
        "200":
          description: public application settings
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PublicAppSettings"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      summary: Update managed application settings
      parameters:
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/AppSettingsWriteRequest"
      responses:
        "200":
          description: managed application settings updated; raw secrets are not returned
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ManagedAppSettings"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid application settings or GA4 measurement ID
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /settings/app/manage:
    get:
      summary: Read administrator application settings
      description: Requires system_settings.update. Secret values are represented only by configured status.
      responses:
        "200":
          description: managed application settings without raw secrets
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ManagedAppSettings"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: authenticated user does not have system_settings.update
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
