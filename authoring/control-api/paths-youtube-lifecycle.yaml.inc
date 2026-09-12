  /streams/{id}/youtube-live-notification:
    get:
      operationId: getDiscordYouTubeLiveNotification
      description: >
        Returns the latest durable Discord YouTube-live notification state for a stream. It contains only the public watch URL, Discord target IDs, lifecycle state, and a confirmed Discord message receipt where available. Event IDs, OAuth credentials, stream keys, and raw Bot errors are never returned.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: latest notification state
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiscordYouTubeLiveNotificationResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: authenticated user does not have streams.read
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: not_found or discord_youtube_notification_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: get_stream_failed or get_discord_youtube_notification_failed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: discord_youtube_notification_outbox_unavailable
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
  /streams/{id}/youtube-live-notifications/{notification_id}/recover:
    post:
      operationId: recoverDiscordYouTubeLiveNotification
      description: >
        Requires streams.start and explicit acknowledgement that the original ambiguous request might already have posted to Discord. Only delivery_unknown records on a still-live stream may be recovered. This creates a new event ID and does not reopen or automatically retry the original event.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/notificationId"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/DiscordYouTubeLiveNotificationRecoveryRequest"
      responses:
        "202":
          description: recovery notification intent created with a new event ID
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/DiscordYouTubeLiveNotificationResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: duplicate_risk_acknowledgement_required
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: authenticated user does not have streams.start
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: discord_youtube_notification_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: discord_youtube_notification_recovery_not_allowed or discord_youtube_notification_recovery_already_requested
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: recover_discord_youtube_notification_failed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: discord_youtube_notification_outbox_unavailable
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
  /streams/{id}/stop:
    post:
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: stream stopped
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: required service assignments are missing; no stop dispatch is attempted
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
  /streams/{id}/youtube/complete:
    post:
      description: Manually retry completion of a retained YouTube Live API runtime after stop or start-cleanup failure. Control Panel also retries retained runtimes automatically with backoff. Raw OAuth tokens and stream keys are never returned.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      responses:
        "200":
          description: YouTube runtime completion retried or no retained runtime exists
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: youtube_relay_static_completion_requires_completed_stream; a fixed-relay runtime may be completed only after its stream is completed.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: YouTube Live API completion failed; runtime is retained for another retry
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /streams/{id}/youtube/relay-static/recovery/resolve:
    post:
      operationId: resolveYouTubeRelayStaticRecovery
      description: >
        Explicitly resolves a live_api_relay_static recovery claim. Requires streams.stop. When Control Panel knows an unstarted YouTube Broadcast ID, it deletes that Broadcast (a provider 404 is also confirmed cleanup). For a possibly-dispatched stream, it first requires the assigned Encoder to acknowledge stop, then completes the Broadcast. When a broadcast insert response was lost, confirm_external_cleanup must attest to out-of-band cleanup. After a confirmed Encoder stop, the same attestation may also resolve a known Broadcast whose provider completion outcome could not be confirmed. This endpoint never returns a stream key, OAuth credential, provider error body, or RTMPS ingest address.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/YouTubeRelayStaticRecoveryResolveRequest"
      responses:
        "200":
          description: Fixed relay recovery claim resolved and released.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/YouTubeRelayStaticRecoveryResolveResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: bad_request or youtube_relay_static_external_cleanup_confirmation_required
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: permission_denied; requires streams.stop
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: not_found or youtube_relay_static_recovery_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: stream_relay_recovery_not_safe_while_active, youtube_relay_static_recovery_not_required, youtube_relay_static_recovery_encoder_stop_unavailable, youtube_relay_static_recovery_broadcast_unknown, youtube_relay_static_recovery_dispatch_state_invalid, or youtube_oauth_account_unavailable
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: get_stream_failed, get_youtube_relay_static_recovery_failed, youtube_relay_static_recovery_credentials_failed, or resolve_youtube_relay_static_recovery_failed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "502":
          description: youtube_relay_static_recovery_cleanup_failed, youtube_relay_static_recovery_encoder_stop_unconfirmed, or youtube_relay_static_recovery_complete_failed; the relay claim is retained.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "503":
          description: youtube_relay_binding_store_unavailable or youtube_relay_static_recovery_cleanup_unavailable
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
