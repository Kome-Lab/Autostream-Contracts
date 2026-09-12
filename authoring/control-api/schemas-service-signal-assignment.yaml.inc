    ServiceObservabilitySignal:
      type: object
      additionalProperties: true
      required: [ type, name ]
      properties:
        type:
          enum: [ event, metric ]
        name:
          type: string
          minLength: 1
        stream_id:
          type: string
        status:
          type: string
        value:
          type: number
        attributes:
          type: object
          additionalProperties: true
        timestamp:
          type: string
          format: date-time
      description: service_id and service_type, if supplied by the caller, are ignored and replaced with the authenticated registered service identity.
    DiagnosticRerunResponse:
      $ref: "../schemas/diagnostic-rerun-response.schema.json"
    ServiceRemediationExecuteRequest:
      type: object
      additionalProperties: false
      required: [ action_id, action, incident_id, stream_id ]
      properties:
        action_id:
          type: string
          minLength: 1
          description: Remediation action ID created by Observability. Must be unique for each dispatch attempt.
        action:
          enum: [ retry_gdrive_upload, retry_package_remux ]
        incident_id:
          type: string
          minLength: 1
          description: Incident ID associated with the remediation action.
        stream_id:
          type: string
          minLength: 1
          description: Stream ID from the incident context.
    StreamServiceAssignment:
      type: object
      description: Stream assignment returned to the authenticated service only. Primary services receive dispatch; standby services are failover candidates and cannot resolve stream-scoped secrets until promoted.
      required: [ stream_id, service_id, service_type, assignment_role, assigned_at ]
      properties:
        stream_id:
          type: string
        service_id:
          type: string
        service_type:
          enum: [ discord_bot, encoder_recorder, worker, observability ]
        assignment_role:
          enum: [ primary, standby ]
        assigned_at:
          type: string
          format: date-time
    ServiceAssignmentWriteRequest:
      type: object
      additionalProperties: false
      required: [ stream_id ]
      properties:
        stream_id:
          type: string
          minLength: 1
        assignment_role:
          type: string
          enum: [ primary, standby ]
          default: primary
          description: primary is dispatched on stream start; standby is registered as a failover candidate but is not started automatically.
