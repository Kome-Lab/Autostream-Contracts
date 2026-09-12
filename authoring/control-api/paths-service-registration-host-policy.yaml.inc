  /services/register:
    post:
      security:
        - bearerToken: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ServiceRegistrationRequest"
      responses:
        "202":
          description: service registered
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/RegisteredService"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid service registration payload or outbound-policy-blocked public URL
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
  /services/host-agent/policy:
    post:
      operationId: fetchHostAgentPolicy
      security:
        - bearerToken: []
      description: Fetches the pull_v2 policy for the authenticated Host Agent. Requires service.config.read. Server-owned host binding fields are never accepted from the request.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostAgentPolicyRequest"
      responses:
        "200":
          description: newer server-owned Host Agent policy
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
                $ref: "#/components/schemas/HostAgentPolicyResponse"
        "204":
          description: policy is unchanged or no policy is configured
          headers:
            Cache-Control:
              schema:
                type: string
              description: Always no-store.
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "400":
          description: invalid_host_agent_policy_request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid service token
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: token does not own service_id or lacks service.config.read
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: host-agent transport binding mismatch or client revision is ahead
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
  /services/host-agent/self-updates/{id}/grants:
    post:
      operationId: issueHostSelfUpdateGrant
      security:
        - bearerToken: []
      description: Issues one short-lived single-logical-consume credential to the authenticated Host Agent for one exact self-update attempt, lifecycle revision, host/agent ownership and policy fence, release identity, protocol set, directive time, plan hash, session, and stage or reconcile operation. Generic update-job grants are not reusable here.
      parameters:
        - $ref: "#/components/parameters/id"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostSelfUpdateGrantIssueRequest"
      responses:
        "201":
          description: first issuance returns the transient credential; an idempotent replay returns only the same non-secret receipt with issued=false
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
                $ref: "#/components/schemas/HostSelfUpdateGrantIssueResponse"
        "400":
          description: invalid_host_self_update_grant_request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "401":
          description: missing or invalid Host Agent service token
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "403":
          description: service token does not own the bound Host Agent
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
          description: lifecycle revision, ownership, policy, plan, operation, expiry, or active-host fence conflict
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
  /services/host-agent/self-update-grants/consume:
    post:
      operationId: consumeHostSelfUpdateGrant
      security: []
      description: The root-owned Local Executor consumes the transient credential against the exact complete grant binding and fixed Control Panel origin. The credential is the authorization proof and must not enter argv, environment, logs, or persistent state. Repeating the same credential and exact binding after response loss returns consumed=false with the same receipt; any changed binding fails closed.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HostSelfUpdateGrantConsumeRequest"
      responses:
        "200":
          description: grant consumed now, or exact response-loss replay reconciled
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
                $ref: "#/components/schemas/HostSelfUpdateGrantConsumeResponse"
        "400":
          description: invalid_host_self_update_grant_consume
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
          headers:
            X-AutoStream-Contract-Major:
              $ref: "#/components/headers/contractMajor"
        "409":
          description: host_self_update_grant_expired, host_self_update_grant_consumed, or exact binding/fence mismatch
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
