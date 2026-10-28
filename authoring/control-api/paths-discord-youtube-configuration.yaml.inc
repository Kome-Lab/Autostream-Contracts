  /discord/configs:
    get:
      responses:
        "200":
          description: list Discord Bot configs without raw bot tokens
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/DiscordConfig"
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
              $ref: "#/components/schemas/DiscordConfigWriteRequest"
      responses:
        "201":
          description: create Discord Bot config
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiscordConfig"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /discord/configs/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Discord Bot config detail without raw bot token
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiscordConfig"
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
              $ref: "#/components/schemas/DiscordConfigWriteRequest"
      responses:
        "200":
          description: Update Discord Bot config and its optional encrypted token in the same transaction. expected_updated_at protects the original editing baseline.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiscordConfig"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: revision_conflict; neither profile nor token was changed
          headers:
            X-AutoStream-Contract-Major: { $ref: "#/components/headers/contractMajor" }
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: Discord Bot config deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /youtube/outputs:
    get:
      responses:
        "200":
          description: list YouTube outputs without raw stream keys or OAuth refresh tokens
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/YouTubeOutput"
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
              $ref: "#/components/schemas/YouTubeOutputWriteRequest"
      responses:
        "201":
          description: create YouTube output
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/YouTubeOutput"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /youtube/outputs/{id}:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: YouTube output detail without raw stream key or OAuth refresh token
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/YouTubeOutput"
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
              $ref: "#/components/schemas/YouTubeOutputWriteRequest"
      responses:
        "200":
          description: Update YouTube output and its optional encrypted stream-key change together under the original expected_updated_at baseline.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/YouTubeOutput"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: revision_conflict or youtube_relay_binding_release_pending; neither output configuration nor its encrypted key changes on these rejections.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: youtube_relay_binding_claim_check_failed; Control Panel could not determine whether the fixed relay binding remains claimed.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    delete:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: YouTube output deleted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: youtube_relay_binding_release_pending; the fixed relay binding must be released through its safe lifecycle before this output is deleted.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: youtube_relay_binding_claim_check_failed; Control Panel could not determine whether the fixed relay binding remains claimed.
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
