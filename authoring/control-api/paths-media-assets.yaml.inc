  /media-assets/upload-sessions:
    parameters: [ { $ref: "#/components/parameters/contractMajor" } ]
    post:
      operationId: createMediaUploadSession
      summary: Create an owner-bound temporary image upload session
      x-autostream-permissions-any: [ streams.create, streams.update ]
      responses:
        "201":
          description: A temporary draft; creating it does not create a stream
          content:
            application/json:
              schema: { $ref: "#/components/schemas/MediaUploadSession" }
        "403": { description: required stream permission or CSRF validation failed }
  /media-assets:
    parameters: [ { $ref: "#/components/parameters/contractMajor" } ]
    post:
      operationId: uploadMediaAsset
      summary: Upload one original image to a temporary session
      x-autostream-permissions-any: [ streams.create, streams.update ]
      description: >
        Metadata parts precede the file. Optional client_operation_id is a lowercase
        UUID bound to the session, user and usage. Repeating the same operation while
        the draft is valid returns its existing asset; it does not consume another
        cover slot. A deleted operation is not reused. Four distinct video_cover
        originals are enforced atomically, including concurrent requests; background
        assets, shared presets and generated variants do not consume the cover quota.
        A missing response is unknown, not evidence of failure. Inspect the exact
        session and operation before an explicit retry; never create a new operation
        merely because the earlier response was lost.
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              additionalProperties: false
              required: [ session_id, usage_type, file ]
              properties:
                session_id: { type: string, format: uuid }
                usage_type: { type: string, enum: [ scene_background, video_cover ] }
                client_operation_id:
                  type: string
                  pattern: '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
                file: { type: string, format: binary }
      responses:
        "201":
          description: Validated original asset; variant conversion and stream save are separate
          content:
            application/json:
              schema: { $ref: "#/components/schemas/PanelMediaAsset" }
        "400": { description: "Invalid image, metadata, operation UUID or multipart field order" }
        "403": { description: "Owner, session or usage mismatch" }
        "409": { description: "Expired or claimed session, integrity conflict, or cover_upload_limit" }
        "500":
          description: media_asset_operation_failed; result may be unknown and must be reconciled by reads
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
  /media-assets/upload-sessions/{session_id}/assets:
    parameters: [ { $ref: "#/components/parameters/contractMajor" } ]
    get:
      operationId: inspectMediaUploadOperation
      summary: Inspect one upload operation without mutating it
      x-autostream-permissions-any: [ streams.create, streams.update ]
      description: >
        Cookie-authenticated and restricted to the session owner. Returns committed
        asset/variant metadata even when the session has expired or been claimed, so
        callers can distinguish those states from a valid draft. An empty assets
        array is not proof that the upload was unapplied: an in-flight request can
        commit later. This endpoint does not authorize automatic upload or create.
      parameters:
        - name: session_id
          in: path
          required: true
          schema: { type: string, format: uuid }
        - name: operation_id
          in: query
          required: true
          schema:
            type: string
            pattern: '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
          description: Exactly one canonical lowercase UUID
      responses:
        "200":
          description: Owner-bound committed metadata; never a terminal non-application receipt
          headers:
            Cache-Control: { schema: { type: string, const: no-store } }
          content:
            application/json:
              schema: { $ref: "#/components/schemas/MediaUploadOperationResponse" }
        "400": { description: "Missing, repeated or invalid operation_id" }
        "403": { description: Required stream permission missing }
        "404": { description: The owner's upload session was not found }
        "500": { description: media_asset_operation_failed; no mutation is performed by inspection }
  /streams/{id}/visual-settings:
    parameters: [ { $ref: "#/components/parameters/contractMajor" } ]
    get:
      operationId: getStreamVisualSettings
      summary: Read a stream's saved visual settings and cover candidates
      x-autostream-required-permission: streams.read
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Saved visual settings including all cover candidates and selected start image
        "403": { description: Required streams.read permission missing }
        "404": { description: The stream or its visual settings were not found }
    put:
      operationId: updateStreamVisualSettings
      summary: Update saved visual settings and exact draft image claims
      parameters: [ { $ref: "#/components/parameters/id" } ]
      description: >
        Requires streams.update. The saved visual revision remains a separate CAS
        authority. Full candidate names support 255 Unicode characters. Existing
        over-limit upload collections may be retained or reduced; no implicit
        deletion or quota bypass is introduced.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              allOf:
                - $ref: "#/components/schemas/StreamVisualWriteSettings"
                - required: [ expected_revision ]
      responses:
        "200": { description: Visual settings and exact draft claims committed }
        "400": { description: Known invalid_visual_settings rejection; draft remains editable }
        "409": { description: "Revision, state, claim or uploaded-cover quota conflict" }
        "500": { description: stream_visual_operation_failed; a missing or ambiguous commit receipt must not be automatically resent }
