  /observability/incidents:
    get:
      responses:
        "200":
          description: proxied incidents
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/incidents/{id}/acknowledge:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: proxied incident acknowledge
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/incidents/{id}/resolve:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: proxied incident resolve
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/incidents/{id}/diagnostics/rerun:
    post:
      description: Proxies a report-only diagnostic re-evaluation to Observability. Requires diagnostics.run; it never resolves the incident or executes remediation.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: proxied diagnostic re-evaluation result
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiagnosticRerunResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid incident identifier
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or expired authenticated session
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: missing diagnostics.run permission or failed CSRF check
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: incident not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "429":
          description: Observability rate limited the re-evaluation request
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: Observability request or authentication failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: Observability is not configured or unavailable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/diagnostics:
    get:
      responses:
        "200":
          description: proxied diagnostic reports
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/metrics:
    get:
      responses:
        "200":
          description: proxied metric snapshots
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/remediation-actions:
    get:
      responses:
        "200":
          description: proxied remediation actions
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/remediation-actions/{id}/approve:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: remediation approved
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/remediation-actions/{id}/execute:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: remediation execution requested
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/notification-deliveries:
    get:
      responses:
        "200":
          description: proxied notification delivery history
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/notification-channels:
    get:
      responses:
        "200":
          description: proxied notification channels without raw webhook URLs or raw SMTP settings
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/NotificationChannel"
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
              $ref: "#/components/schemas/ControlNotificationChannelCreateRequest"
      responses:
        "201":
          description: proxied notification channel created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/NotificationChannel"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid notification channel or webhook URL
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "429":
          description: Observability rate limit reached
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: Observability Node authentication failed or the service could not be reached
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: Observability is unavailable or its secret encryption key is not configured
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/notification-channels/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: proxied notification channel detail without raw webhook URL or raw SMTP settings
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/NotificationChannel"
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
              $ref: "#/components/schemas/ControlNotificationChannelUpdateRequest"
      responses:
        "200":
          description: proxied notification channel updated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/NotificationChannel"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid notification channel or webhook URL
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "429":
          description: Observability rate limit reached
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: Observability Node authentication failed or the service could not be reached
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: Observability is unavailable or its secret encryption key is not configured
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: proxied notification channel deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /observability/notification-channels/{id}/test:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "202":
          description: proxied notification channel test
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/NotificationDeliveryResult"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
