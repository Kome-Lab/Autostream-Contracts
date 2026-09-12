  /streams/{id}/mark-failed:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream marked failed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/retry-upload:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "202":
          description: archive upload retry queued
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: assigned encoder_recorder service is missing; no package retry dispatch is attempted
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
  /streams/{id}/encoder-preflight:
    get:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: assigned Encoder/Recorder preflight checks with secret values redacted
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
  /streams/{id}/preview/{name}:
    get:
      summary: Read an active stream HLS preview asset
      description: >
        Proxies the selected primary Encoder/Recorder preview through Control Panel. Requires streams.read. Only index.m3u8 and segment-NNNNNN.ts are accepted, and the stream must be starting, live, or stopping. The assigned Node token is never returned to the browser. Preview retrieval never rolls the live stream.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/previewAssetName"
      responses:
        "200":
          description: HLS playlist or MPEG-TS segment
          headers:
            Cache-Control:
              schema:
                type: string
              description: no-store for playlists; private, max-age=30 for segments
            Referrer-Policy:
              schema:
                type: string
              description: no-referrer
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/vnd.apple.mpegurl:
              schema:
                type: string
            video/mp2t:
              schema:
                type: string
                format: binary
        "400":
          description: invalid preview asset name
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: stream or preview asset is not ready
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: stream is not active or required primary assignment is missing
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: Encoder/Recorder preview fetch failed or returned an unsafe playlist
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/preview-links:
    post:
      summary: Create a VLC-compatible signed HLS preview link
      description: >
        Creates a bearer-capability URL for the active stream. Requires streams.read and CSRF protection. The link expires after 12 hours at most and becomes invalid as soon as the stream is no longer starting, live, or stopping. The URL itself must not be written to logs, audit metadata, tickets, or chat.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      responses:
        "201":
          description: signed public HLS link created
          headers:
            Cache-Control:
              schema:
                type: string
              description: no-store
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/StreamPreviewLink"
        "409":
          description: stream is not active or required primary assignment is missing
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: preview proxy or signing key is unavailable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /stream-previews/{token}/{name}:
    get:
      security: []
      summary: Read a signed public HLS preview asset
      description: >
        Public capability endpoint used by VLC and other HLS players. The signature is bound to one stream and external_player purpose. Playlist URIs remain relative so segment requests reuse the same signed path. The URL expires after 12 hours at most and returns 410 after the stream stops.
      parameters:
        - name: token
          in: path
          required: true
          description: Signed bearer capability. Never log or expose this value.
          schema:
            type: string
            minLength: 1
        - $ref: "#/components/parameters/previewAssetName"
      responses:
        "200":
          description: HLS playlist or MPEG-TS segment
          headers:
            Cache-Control:
              schema:
                type: string
              description: no-store for playlists; private, max-age=30 for segments
            Referrer-Policy:
              schema:
                type: string
              description: no-referrer
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/vnd.apple.mpegurl:
              schema:
                type: string
            video/mp2t:
              schema:
                type: string
                format: binary
        "404":
          description: signed token or preview asset is invalid, expired, or not ready
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "410":
          description: stream is no longer active
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /stream-previews/{token}/participants:
    get:
      security: []
      summary: Read the current Discord voice participants for a signed preview
      description: >
        Returns non-secret display metadata from the latest worker participant snapshot. The preview bearer token authorizes this request; Discord credentials and service tokens are never returned.
      parameters:
        - name: token
          in: path
          required: true
          description: Signed bearer capability. Never log or expose this value.
          schema:
            type: string
            minLength: 1
      responses:
        "200":
          description: Current voice participants and active speaker state
          content:
            application/json:
              schema:
                type: object
                required: [ participants ]
                properties:
                  participants:
                    type: array
                    items:
                      type: object
                      required: [ user_id ]
                      properties:
                        user_id: { type: string }
                        display_name: { type: string }
                        avatar_url: { type: string, format: uri }
                        is_bot: { type: boolean }
                        speaking: { type: boolean }
                  active_speaker_id: { type: string }
                  updated_at: { type: string, format: date-time }
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: signed token is invalid or expired
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "410":
          description: stream is no longer active
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: worker participant events could not be read
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: preview participant service is unavailable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
