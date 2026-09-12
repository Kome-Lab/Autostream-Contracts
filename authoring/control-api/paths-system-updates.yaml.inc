  /system-updates:
    get:
      description: Lists server-resolved updater status, execution-host reachability, update targets, and durable update jobs. Requires system_updates.read. Docker bundle versions remain separate from service source versions.
      responses:
        "200":
          description: update target inventory and job history
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdatesResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    post:
      description: Creates one durable update job using the trusted latest release resolved by Control Panel. Requires system_updates.execute. The request cannot supply a URL, path, image, command, digest, version, or systemd unit.
      parameters:
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SystemUpdateCreateRequest"
      responses:
        "202":
          description: update job persisted before execution
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdateJob"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: target is not eligible, is in maintenance, or already has an active update
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /system-updates/{id}/cancel:
    post:
      description: Cancels a queued update before an updater claims it. Requires system_updates.execute.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      responses:
        "200":
          description: queued job canceled
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdateJob"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: job is no longer cancelable
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /system-updates/host-self-updates:
    get:
      operationId: listHostSelfUpdates
      description: Lists the dedicated Host Agent and Local Executor self-update lifecycle. Requires system_updates.read. Raw grants, release credentials, and download locations are never returned.
      responses:
        "200":
          description: bounded Host self-update history
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HostSelfUpdateListResponse"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /system-updates/hosts/{host_id}/self-updates:
    post:
      operationId: createHostSelfUpdate
      description: Creates one dedicated Host self-update after the server resolves and verifies the fixed public immutable Kome-Lab release. The request supplies only the target version and idempotency key; it cannot supply a host binding, URL, credential, commit, digest, path, command, or artifact.
      parameters:
        - name: host_id
          in: path
          required: true
          schema:
            type: string
            minLength: 1
            maxLength: 191
            pattern: "^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostSelfUpdateCreateRequest"
      responses:
        "202":
          description: new lifecycle persisted, or the exact idempotent result replayed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HostSelfUpdate"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid_host_self_update_request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: host_self_update_host_not_found or host_self_update_agent_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: release verification, host readiness, idempotency, ownership, or active-lifecycle conflict
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
  /system-updates/host-self-updates/{id}/retry:
    post:
      operationId: retryHostSelfUpdate
      description: Creates a new random-generation attempt for the same immutable release and exact host fences after a terminal failed or rolled-back lifecycle. Requires system_updates.execute.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostSelfUpdateRetryRequest"
      responses:
        "202":
          description: retry lifecycle persisted, or the exact idempotent result replayed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HostSelfUpdate"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid_host_self_update_retry
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: host_self_update_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: lifecycle, idempotency, revision, ownership, or active-host conflict
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
  /system-updates/host-self-updates/{id}/cancel:
    post:
      operationId: cancelHostSelfUpdate
      description: Cancels a queued lifecycle or requests cooperative cancellation from an executing Host Agent. Executing cancellation returns 409 when the registered agent has no cancellation capability. Requires system_updates.execute.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostSelfUpdateCancelRequest"
      responses:
        "200":
          description: queued attempt canceled or executing attempt moved to cancel_requested
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HostSelfUpdate"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid_host_self_update_cancel
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: host_self_update_not_found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: host_self_update_cancel_unsupported or lifecycle revision/state conflict
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
  /system-updates/updaters/{id}/pull-ownership/activate:
    post:
      operationId: activateSystemUpdatePullOwnership
      description: >
        Atomically activates the already-registered and readiness-verified pull_v2 Host Agent for one execution host. Requires system_updates.execute. The id path parameter selects the pre-created updater; every request field is a compare-and-swap expectation. The server chooses the next ownership epoch and never accepts a replacement owner, credential, command, path, URL, image, or release input.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SystemUpdatePullOwnershipActivateRequest"
      responses:
        "200":
          description: pull_v2 ownership activated with server-owned fencing values and no secret
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdatePullOwnershipActivateResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: bad_request or invalid_pull_activation_request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: permission_denied
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: update_agent_not_registered
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: updater_policy_revision_conflict, system_update_ownership_conflict, system_update_execution_host_busy, host_agent_not_ready, or update_agent_inactive
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: activate_pull_ownership_failed
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
  /system-updates/updaters/{id}/pull-ownership/deactivate:
    post:
      operationId: deactivateSystemUpdatePullOwnership
      description: >
        Atomically returns one active pull_v2 Host Agent to observer epoch zero. Requires system_updates.execute. The id path parameter selects the updater; every request field is a compare-and-swap expectation. The client cannot select a replacement owner, transport, next epoch, credential, command, path, URL, image, or release.
      parameters:
        - $ref: "#/components/parameters/id"
        - $ref: "#/components/parameters/csrfToken"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SystemUpdatePullOwnershipDeactivateRequest"
      responses:
        "200":
          description: pull_v2 updater returned to observer epoch zero
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdatePullOwnershipDeactivateResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: bad_request or invalid_pull_deactivation_request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: permission_denied
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: update_agent_not_registered
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: updater_policy_revision_conflict, system_update_ownership_conflict, host_lifecycle_busy, host_agent_not_ready, or update_agent_inactive
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "500":
          description: deactivate_pull_ownership_failed
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
