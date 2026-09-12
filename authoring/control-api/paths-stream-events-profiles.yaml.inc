  /streams/{id}/audio-status:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Discord audio bridge status from the assigned Encoder/Recorder
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: assigned Encoder/Recorder service is missing
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/worker-events:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: recent Worker events returned through the assigned Encoder/Recorder with secret values redacted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: assigned Encoder/Recorder service is missing
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/worker-events/test:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "202":
          description: test Worker event dispatched to the assigned Encoder/Recorder
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: assigned Encoder/Recorder service is missing
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/logs:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream logs
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/artifacts:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream artifacts
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/encoder:
    get: { $ref: "#/components/pathItems/listCreateProfile/get" }
    post: { $ref: "#/components/pathItems/listCreateProfile/post" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/encoder/{id}:
    get: { $ref: "#/components/pathItems/itemProfile/get" }
    put: { $ref: "#/components/pathItems/itemProfile/put" }
    delete: { $ref: "#/components/pathItems/itemProfile/delete" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/archive:
    get: { $ref: "#/components/pathItems/listCreateProfile/get" }
    post: { $ref: "#/components/pathItems/listCreateProfile/post" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/archive/{id}:
    get: { $ref: "#/components/pathItems/itemProfile/get" }
    put: { $ref: "#/components/pathItems/itemProfile/put" }
    delete: { $ref: "#/components/pathItems/itemProfile/delete" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/caption:
    get: { $ref: "#/components/pathItems/listCreateProfile/get" }
    post: { $ref: "#/components/pathItems/listCreateProfile/post" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/caption/{id}:
    get: { $ref: "#/components/pathItems/itemProfile/get" }
    put:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: caption profile updated; any referencing live streams were refreshed successfully
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: caption profile was saved, but at least one referencing live stream could not apply it immediately; code is caption_profile_saved_runtime_apply_failed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete: { $ref: "#/components/pathItems/itemProfile/delete" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/overlay:
    get: { $ref: "#/components/pathItems/listCreateProfile/get" }
    post: { $ref: "#/components/pathItems/listCreateProfile/post" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /profiles/overlay/{id}:
    get: { $ref: "#/components/pathItems/itemProfile/get" }
    put: { $ref: "#/components/pathItems/itemProfile/put" }
    delete: { $ref: "#/components/pathItems/itemProfile/delete" }
    parameters:
      - $ref: "#/components/parameters/contractMajor"
