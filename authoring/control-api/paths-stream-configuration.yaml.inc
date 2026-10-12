  /streams:
    get:
      responses:
        "200":
          description: list streams
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    post:
      description: Create a stream, its visual settings and both primary assignments atomically. Requires streams.create, services.assign and workers.assign. Offline services may be reserved; missing, wrong-type or occupied services are rejected without partial persistence.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/StreamWriteRequest"
      responses:
        "201":
          description: stream and both assignments committed
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
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      description: >
        Soft-delete a stopped or unstarted stream and release only its own assignments. Requires streams.delete. Active/uncertain execution, recording finalization and relay claims remain protected; retained recordings and other streams are not deleted.
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
          description: stream settings updated
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: stream not found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: stream_status_not_editable or youtube_relay_binding_release_pending; a fixed relay binding must be released through its safe lifecycle before changing the selected YouTube output.
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
              $ref: "#/components/schemas/EncoderRuntimeSettingsRequest"
      responses:
        "200": { description: settings persisted and applied when the stream is live, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "400": { description: invalid gain or watermark profile, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "404": { description: stream not found, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: stream is starting or stopping, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
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
  /streams/{id}/visual-settings:
    get:
      summary: Read saved stream visual settings
      operationId: getStreamVisualSettings
      description: Read saved visual configuration with streams.read.
      parameters:
        - { $ref: "#/components/parameters/id" }
      responses:
        "200":
          description: Validated canonical state
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "../schemas/stream-operations.schema.json#/$defs/VisualSettings"
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      description: Save candidates and immutable preset snapshots with streams.update. Running or uncertain execution remains protected.
      summary: Save stream visual settings
      operationId: updateStreamVisualSettings
      parameters:
        - { $ref: "#/components/parameters/id" }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "../schemas/stream-operations.schema.json#/$defs/VisualUpdate"
      responses:
        "200":
          description: Validated canonical state
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "../schemas/stream-operations.schema.json#/$defs/VisualSettings"
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/video-cover-state:
    get:
      summary: Read and reconcile stream cover state
      operationId: getStreamVideoCoverState
      description: Read requested and Encoder-confirmed state with streams.read. A pending action is reconciled by a generation/revision/image-matched Encoder GET without resending.
      parameters:
        - { $ref: "#/components/parameters/id" }
      responses:
        "200":
          description: Validated canonical state
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "../schemas/stream-operations.schema.json#/$defs/CoverState"
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    put:
      description: Show or switch one saved candidate with streams.show_cover; hide requires streams.hide_cover and hide_confirmed. Permission, capability, generation, revisions and duplicate fencing remain authoritative.
      summary: Show switch or hide a stream cover
      operationId: updateStreamVideoCoverState
      parameters:
        - { $ref: "#/components/parameters/id" }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "../schemas/stream-operations.schema.json#/$defs/CoverAction"
      responses:
        "200":
          description: Validated canonical state
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "../schemas/stream-operations.schema.json#/$defs/CoverState"
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/media-assets/{variant_id}:
    get:
      summary: Read an authorized stream image
      operationId: getStreamMediaAsset
      description: Read only image variants referenced by this stream with streams.read.
      parameters:
        - { $ref: "#/components/parameters/id" }
        - { name: variant_id, in: path, required: true, schema: { type: string } }
      responses:
        "200":
          description: Authorized image bytes
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            image/png:
              schema: { type: string, format: binary }
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /video-cover-presets/{id}/media-assets/{variant_id}:
    get:
      summary: Read an authorized cover preset image
      operationId: getVideoCoverPresetMediaAsset
      description: Read current saved preset image with video_cover_presets.read, independent of streams.create.
      parameters:
        - { $ref: "#/components/parameters/id" }
        - { name: variant_id, in: path, required: true, schema: { type: string } }
      responses:
        "200":
          description: Authorized image bytes
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            image/png:
              schema: { type: string, format: binary }
        "403":
          description: permission denied
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "409":
          description: stale revision, pending action, foreign image or protected execution
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/stop-outcome:
    get:
      summary: Reconcile an exact normal stop request
      operationId: getStreamStopOutcome
      description: Read the durable positive normal-stop receipt for the exact pre-request updated_at. ready alone does not prove a stop. No automatic stop resend is performed.
      parameters:
        - { $ref: "#/components/parameters/id" }
        - { name: expected_updated_at, in: query, required: true, schema: { type: string, format: date-time } }
      responses:
        "200":
          description: streams.read-authorized exact source-version receipt and current stream
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/StopOutcome" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /video-cover-presets:
    get:
      summary: List saved cover presets
      operationId: listVideoCoverPresets
      description: Requires video_cover_presets.read. Preserves permission, expected revision and asset ownership checks.
      responses:
        "200":
          description: Canonical preset state
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/PresetList" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision or asset ownership conflict, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    post:
      description: Requires video_cover_presets.create. Preserves permission, expected revision and asset ownership checks.
      summary: Create a cover preset
      operationId: createVideoCoverPreset
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/PresetWrite" }
      responses:
        "201":
          description: Canonical preset state
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/Preset" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision or asset ownership conflict, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /video-cover-presets/{id}:
    get:
      summary: Read a saved cover preset
      operationId: getVideoCoverPreset
      description: Requires video_cover_presets.read. Preserves permission, expected revision and asset ownership checks.
      responses:
        "200":
          description: Canonical preset state
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/Preset" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision or asset ownership conflict, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    put:
      description: Requires video_cover_presets.update. Preserves permission, expected revision and asset ownership checks.
      summary: Update a cover preset
      operationId: updateVideoCoverPreset
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/PresetWrite" }
      responses:
        "200":
          description: Canonical preset state
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/Preset" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision or asset ownership conflict, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    delete:
      description: Requires video_cover_presets.delete. Preserves permission, expected revision and asset ownership checks.
      summary: Delete a cover preset
      operationId: deleteVideoCoverPreset
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/PresetDelete" }
      responses:
        "200":
          description: Canonical preset state
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
          content:
            application/json:
              schema: { $ref: "../schemas/stream-operations.schema.json#/$defs/Preset" }
        "403": { description: permission denied, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "409": { description: revision or asset ownership conflict, headers: { X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" } } }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    parameters:
      - $ref: "#/components/parameters/id"
      - $ref: "#/components/parameters/contractMajor"
