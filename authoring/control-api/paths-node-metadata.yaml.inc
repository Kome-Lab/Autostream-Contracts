  /nodes:
    get:
      operationId: listNodesWithMetadataRevision
      summary: Read registered Nodes with semantic metadata editing authority
      x-autostream-required-permission: api_tokens.create
      responses:
        "200":
          description: Node metadata and current monitoring observations
          content:
            application/json:
              schema:
                type: array
                items: { $ref: "#/components/schemas/NodeMetadataView" }
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /nodes/{id}:
    put:
      operationId: updateNodeMetadata
      summary: Edit Node metadata under its original optional semantic baseline
      description: With expected_metadata_revision, an actual endpoint change checks the latest updater ownership and metadata baseline in the same coordinated write. Name/description edits and exact endpoint no-ops retain the ordinary metadata CAS. Omitting the expectation preserves the legacy preliminary ownership guard; it does not disable that guard.
      x-autostream-required-permission: api_tokens.create
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/NodeMetadataWriteRequest" }
      responses:
        "200":
          description: Node metadata saved; existing endpoint management guards remain in force
          content:
            application/json:
              schema: { $ref: "#/components/schemas/NodeMetadataView" }
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "400":
          description: invalid_metadata_revision for a malformed supplied baseline, or an existing Node input validation error
        "409":
          description: node_metadata_conflict before metadata changes, or node_endpoint_managed_by_updater when updater management is confirmed
        "503":
          description: node_endpoint_ownership_unavailable before writes when a stamped endpoint change cannot obtain a supported, matching and consistent ownership authority
        "426": { $ref: "#/components/responses/contractMajorUnsupported" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
