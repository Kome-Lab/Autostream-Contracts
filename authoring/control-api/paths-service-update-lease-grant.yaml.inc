  /services/heartbeat:
    post:
      security:
        - bearerToken: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: "#/components/schemas/UpdaterHeartbeat"
                - $ref: "#/components/schemas/Heartbeat"
      responses:
        "202":
          description: heartbeat accepted
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/update-jobs/claim:
    post:
      security:
        - bearerToken: []
      description: Atomically claims the next eligible job as a v2 command assigned to the authenticated update_agent and optional host_id execution slot. Requires updates.claim. If active_job_id is present, the server returns only that command's renewed typed lease or an explicit clear_active_job_id instruction and never claims a different command.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateAgentClaimRequest"
      responses:
        "200":
          description: job claimed or renewed with a time-bounded lease, or the reported active job must be durably cleared
          content:
            application/json:
              schema:
                oneOf:
                  - $ref: "#/components/schemas/UpdaterLeaseEnvelope"
                  - $ref: "#/components/schemas/UpdateAgentClearActiveJobResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "204":
          description: active_job_id was omitted and no eligible job is currently available
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: token is not bound to the requested update_agent or lacks updates.claim
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/update-jobs/{id}/report:
    post:
      security:
        - bearerToken: []
      description: Idempotently reports monotonic progress for the claimed job. Requires updates.report and the active lease. Arbitrary logs and secrets are rejected or sanitized.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: "#/components/schemas/UpdaterProgressEnvelope"
                - $ref: "#/components/schemas/UpdaterResultEnvelope"
                - $ref: "#/components/schemas/UpdateAgentReportRequest"
      responses:
        "200":
          description: progress accepted and lease renewed when applicable
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SystemUpdateJob"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: lease expired, sequence regressed, or state transition is invalid
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/update-jobs/{id}/mutation-grants:
    post:
      security:
        - bearerToken: []
      description: Issues a short-lived one-time mutation grant for the destination helper after validating the authenticated update_agent, updates.authorize scope, active lease, execution host, target, version, deployment mode, operation, immutable plan hash, and remote session. The job identity is taken only from the id path parameter. Connection and privileged execution configuration remain server-side.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: "#/components/schemas/UpdaterMutationGrantIssueRequest"
                - $ref: "#/components/schemas/UpdateAgentMutationGrantIssueRequest"
      responses:
        "201":
          description: one-time mutation grant issued for the exact immutable binding
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store because the response contains a one-time grant credential.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UpdaterMutationGrantIssueResponse"
        "400":
          description: bad_request or invalid_system_update_mutation_grant
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid service token
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: token lacks updates.authorize or is not bound to service_id
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "404":
          description: system_update_job_not_found
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: mutation grant state, lease, host, target, version, mode, operation, plan hash, or session binding does not match
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
  /services/update-jobs/{id}/mutation-grants/consume:
    post:
      security:
        - bearerToken: []
      description: Consumes the mutation grant supplied as the Bearer credential against the exact job path and immutable host, target, version, deployment mode, operation, plan hash, lease generation, and session binding. The first matching use and an idempotent replay of the same token, session, and binding return 204; any different reuse returns 409. The grant token is never accepted in the JSON body.
      parameters: [ { $ref: "#/components/parameters/id" } ]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: "#/components/schemas/UpdaterMutationGrantConsumeRequest"
                - $ref: "#/components/schemas/UpdateAgentMutationGrantConsumeRequest"
      responses:
        "204":
          description: grant consumed, or the exact same grant, session, and binding was already consumed; response has no body
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: bad_request or invalid_system_update_mutation_grant_consumption
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: mutation grant Bearer credential is missing or malformed
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: mutation grant is unknown, expired, no longer bound to the current job lease, or was reused with a different job, session, or binding
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "426":
          $ref: "#/components/responses/contractMajorUnsupported"
    parameters:
      - $ref: "#/components/parameters/contractMajor"
