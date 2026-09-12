  /services/observability/signals:
    post:
      security:
        - bearerToken: []
      description: Worker and Encoder/Recorder report Observability signals through Control Panel with a Node Runtime Token. Requires observability.ingest. Control Panel overwrites service_id and service_type from the registered service before forwarding to Observability.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceObservabilitySignal"
      responses:
        "202":
          description: signal accepted for forwarding
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid signal payload
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid service token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: wrong service type, pending service, or missing observability.ingest scope
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: Observability API is unavailable or rejected the forwarded signal
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/stream-events:
    post:
      security:
        - bearerToken: []
      responses:
        "202":
          description: service stream event accepted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/stream-artifacts:
    post:
      security:
        - bearerToken: []
      description: Assigned Encoder/Recorder reports logical archive artifacts. Requires encoder.status.write.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceArtifactReport"
      responses:
        "202":
          description: artifact report accepted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid artifact metadata or unsafe relative path
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: wrong service type, missing scope, or service is not assigned to the stream
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: service_or_stream_not_found, service_not_registered, or stream_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: stream_artifact_store_failed; a non-transient persistence failure prevented the artifact report from being stored
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: stream_artifact_store_unavailable; a transient database connection failure prevented the artifact report from being stored and the caller may retry
          headers:
            Retry-After:
              description: Number of seconds the caller should wait before retrying the artifact report.
              schema:
                type: integer
                minimum: 1
              example: 1
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/streams/{id}/start:
    post:
      security:
        - bearerToken: []
      description: Starts a waiting stream from its primary assigned Discord Bot after a configured Discord voice-channel join. Requires streams.start. The authenticated service token must be discord_bot, be registered as the stream's primary Discord Bot, and match the saved Discord configuration; request data cannot override the saved stream settings.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream start lifecycle accepted or completed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid service token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: missing streams.start scope, non-Discord service, or token is not the primary assigned Discord Bot
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: stream is not configured for Discord VC auto-start or is not currently startable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/streams/{id}/stop:
    post:
      security:
        - bearerToken: []
      description: Stops an active Discord VC-triggered stream after its primary assigned Discord Bot observes that the configured voice channel is empty. Requires streams.stop. The authenticated service token must be the stream's primary Discord Bot; request data cannot override the saved stream settings. completed and stopping streams are idempotent successes.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream stopped, or already stopped
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "202":
          description: stop is already in progress
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid service token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: missing streams.stop scope, non-Discord service, or token is not the primary assigned Discord Bot
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: stream is not configured for Discord VC auto-stop or is not currently stoppable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/remediation-actions/execute:
    post:
      security:
        - bearerToken: []
      description: Observability requests an approved remediation dispatch. Requires remediation.execute. action_id, incident_id, and stream_id are required correlation context. Control Panel verifies the context against Observability read API and rejects duplicate action_id replays before service dispatch.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceRemediationExecuteRequest"
      responses:
        "202":
          description: remediation dispatch accepted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: missing action_id, incident_id, stream_id, or unsupported action
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: wrong service type, missing remediation scope, or remediation context not verified
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: duplicate action_id replay or missing stream assignment
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /service-health:
    get:
      responses:
        "200":
          description: service health list
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/{id}/assign:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceAssignmentWriteRequest"
      responses:
        "200":
          description: service assigned to a stream
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/RegisteredService"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/{id}/assignment:
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: service unassigned from its current stream
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/{id}:
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: service deleted from the registry; assignments are removed and the linked service token is revoked
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: service not found
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
