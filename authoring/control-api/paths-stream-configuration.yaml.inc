  /streams:
    get:
      responses:
        "200":
          description: list streams
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  required: [ settings_revision ]
                  properties:
                    settings_revision: { $ref: "#/components/schemas/StreamSettingsRevision" }
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
              $ref: "#/components/schemas/StreamCreateV2Request"
      responses:
        "201":
          description: Configuration, visuals, selected image claims and primary Node/Bot reservations committed together. A lost response is not safe to automatically repeat.
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream detail
          content:
            application/json:
              schema:
                type: object
                required: [ settings_revision ]
                properties:
                  settings_revision: { $ref: "#/components/schemas/StreamSettingsRevision" }
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      description: >
        Delete a non-active stream, its persisted settings, logs, artifacts, runtime state, and service assignments. Requires streams.delete. Active streams remain protected and return 409 until stopped.
      responses:
        "200":
          description: stream deleted
          content:
            application/json:
              schema:
                type: object
                additionalProperties: false
                required: [ status, stream_id ]
                properties:
                  status:
                    type: string
                    enum: [ deleted ]
                  stream_id:
                    type: string
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: authenticated user does not have streams.delete
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: stream not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: active stream cannot be deleted or youtube_relay_binding_release_pending; a fixed relay binding must be released through its safe lifecycle before deletion.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: youtube_relay_binding_claim_check_failed; Control Panel could not determine whether a fixed relay binding remains claimed.
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
  /streams/{id}/settings:
    put:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/StreamSettingsWriteRequest"
      responses:
        "200":
          description: Stream settings updated. settings_revision is the settings-only editing authority; polling lifecycle and heartbeat values do not replace the editing baseline. Conditional edits use managed archive destinations/profiles updated under their own CAS, followed by a stream reference edit; the legacy direct-archive materialization request cannot be combined with expected_settings_revision.
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: stream_settings_archive_revision_unsupported before writes when expected_settings_revision is supplied together with a requested legacy direct-archive materialization. Omitted legacy revisions and empty clearing fields retain their prior behavior.
        "404":
          description: stream not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: revision_conflict, stream_status_not_editable or youtube_relay_binding_release_pending; settings conflicts preserve the draft and require explicit reconciliation.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: youtube_relay_binding_claim_check_failed; Control Panel could not determine whether a fixed relay binding remains claimed.
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
  /streams/{id}/runtime-settings:
    put:
      summary: Persist and, when live, immediately apply Encoder gain and watermark settings
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PanelEncoderRuntimeSettingsWriteRequest"
      responses:
        "200": { description: settings persisted and applied when the stream is live, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "400": { description: invalid gain or watermark profile, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "404": { description: stream not found, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision_conflict or stream is starting or stopping; a stale editing baseline never overwrites the current settings, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "502": { description: assigned Encoder could not apply the live settings; desired settings remain persisted, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/start-readiness:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: false
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/StartStreamRequest"
      responses:
        "200":
          description: start readiness check result; no stream status change or service dispatch is attempted
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/StartReadinessResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/external-e2e-config:
    get:
      summary: Export non-secret external verification Control Panel confirmation
      description: >
        Returns the Control Panel-managed IDs and boolean confirmations needed by the external verification handoff file. Requires a cookie-authenticated user with streams.read. The response is always Cache-Control: no-store and never returns raw Discord guild/channel IDs, Drive folder IDs, OAuth refresh tokens, client secrets, RTMPS URLs, stream keys, service tokens, or session cookies.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: non-secret external verification confirmation export
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store because the response confirms runtime integration state.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ExternalE2EConfigResponse"
        "403":
          description: authenticated user does not have streams.read
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: stream not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/start:
    post:
      description: >
        Starts every assigned primary service. When a public YouTube watch URL and Discord text target are selected, Control Panel atomically enqueues a durable notification intent with the live stream transition; a successful start response means queued, not delivered. live_api and live_api_relay_static wait for the provider Broadcast lifecycle to be live. stream_key remains a legacy_unverified compatibility path. A transport-ambiguous Bot result becomes delivery_unknown and never auto-retries; an operator may explicitly attest and request a new event through the recovery path. Audit metadata stores only a watch URL fingerprint.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: false
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/StartStreamRequest"
      responses:
        "200":
          description: stream started
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: required service assignments are missing; no dispatch is attempted
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MissingStreamAssignmentsResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
