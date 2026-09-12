openapi: 3.1.0
info:
  title: AutoStream Control API
  version: 0.2.0
x-autostream-contract-policy:
  contract_major: 2
  supported_contract_majors: [ 2 ]
  unsupported_contract_major_status: 426
  invalid_payload_status: 400
  final_state: v2_only
  uri_strategy: preserve_existing_paths_no_v2_prefix
  request_header: X-AutoStream-Contract-Major
  response_header: X-AutoStream-Contract-Major
  request_header_value: "2"
  response_header_value: "2"
  request_scope: json_and_multipart_except_health_static_and_oauth_mechanics
  unknown_field_behavior: reject_fail_closed
  unknown_payload_behavior: reject_fail_closed
  unknown_capability_behavior: non_ready
  stable_error_codes:
    - contract_major_unsupported
    - protocol_version_unsupported
    - request_schema_invalid
    - revision_conflict
    - stale_generation
    - stale_fence
    - semantic_validation_failed
  temporary_compatibility:
    owner: V2-COMPAT-EOL-CONTRACTS-001
    removal_wave: Execution Bundle 8
    final_state: absent
    missing_major_behavior: accepted_only_by_explicit_legacy_adapter
  rollback_boundary: whole_release_before_consumer_migration
  stable_error_schema: { $ref: '#/components/schemas/V2ContractError' }
  capability_negotiation_schema: { $ref: '#/components/schemas/V2CapabilityNegotiation' }
  worker_application_probe_path_item: { $ref: '#/components/pathItems/WorkerApplicationRuntimeIdentityProbe' }
x-autostream-updater-v2-authority:
  protocol_version: 2
  orchestration_authority: control_panel
  host_execution_authority: updater
  local_journal_owner: updater
  application_verification_consumer: local_executor
  updater_health_substitutes_for_application_probe: false
  arbitrary_shell_allowed: false
  shared_control_panel_database_credentials_allowed: false
  observability_host_execution_allowed: false
  command_envelope_schema: { $ref: '#/components/schemas/UpdaterCommandEnvelope' }
  desired_operation_schema: { $ref: '#/components/schemas/UpdaterDesiredOperation' }
  lease_envelope_schema: { $ref: '#/components/schemas/UpdaterLeaseEnvelope' }
  progress_envelope_schema: { $ref: '#/components/schemas/UpdaterProgressEnvelope' }
  result_envelope_schema: { $ref: '#/components/schemas/UpdaterResultEnvelope' }
  heartbeat_schema: { $ref: '#/components/schemas/UpdaterHeartbeat' }
  local_journal_schema: { $ref: '#/components/schemas/UpdaterLocalJournalBoundary' }
  canonical_command_digest:
    algorithm: sha256
    serialization: RFC8785_JCS
    projection_fields: [ target, desired_revision, fence, desired_operation ]
    timestamp_lexical_form: canonical_utc_rfc3339nano_no_trailing_fractional_zeroes
    payload_and_argument_digests_must_match: true
  runtime_token_rotation:
    credential_claim_request_schema: { $ref: '#/components/schemas/UpdaterRuntimeTokenRotationCredentialClaimRequest' }
    raw_replacement_token_surface: credential_claim_response_only
    cache_control: no-store
    command_allowed: false
    local_journal_allowed: false
  local_executor_mutation_grant:
    binding_schema: { $ref: '#/components/schemas/UpdaterMutationGrantBinding' }
    issue_request_schema: { $ref: '#/components/schemas/UpdaterMutationGrantIssueRequest' }
    issue_response_schema: { $ref: '#/components/schemas/UpdaterMutationGrantIssueResponse' }
    consume_request_schema: { $ref: '#/components/schemas/UpdaterMutationGrantConsumeRequest' }
    opaque_grant_surface: issue_response_and_authorization_header_only
    cache_control: no-store
    command_allowed: false
    local_journal_allowed: false
x-autostream-remediation-authority:
  detection_proposal_evidence: observability
  authorization_orchestration_audit: control_panel
  host_system_execution: updater
  application_local_execution: explicit_target_service
  observability_role: detect_propose_evidence
  observability_may_mint_grant: false
  observability_may_execute_host_runtime: false
  observability_may_call_updater_directly: false
  denied_actions: [ delete_archives, rotate_credentials, change_roles, stop_live_stream, recreate_youtube_broadcast, revoke_service_tokens ]
  grant_schema: { $ref: '#/components/schemas/RemediationGrant' }
  result_evidence_schema: { $ref: '#/components/schemas/RemediationResultEvidence' }
servers:
  - url: https://control.example.com
security:
  - cookieSession: []
paths:
