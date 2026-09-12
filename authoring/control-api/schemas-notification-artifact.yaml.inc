    ServiceNotificationEmailRequest:
      type: object
      additionalProperties: false
      required: [ recipients, subject, text ]
      properties:
        recipients:
          type: array
          minItems: 1
          maxItems: 20
          uniqueItems: true
          items:
            type: string
            format: email
            maxLength: 320
            pattern: '^[^\r\n]+$'
            description: Bare mailbox address only; display names and CR/LF are rejected.
          description: Email envelope recipients. Responses and errors never reflect these addresses raw.
        subject:
          type: string
          minLength: 1
          maxLength: 200
          pattern: '^[^\r\n]*$'
          description: Email subject. The server counts Unicode code points and rejects CR/LF to prevent header injection.
        text:
          type: string
          minLength: 1
          maxLength: 16384
          pattern: '^[^\u0000]*$'
          description: Plain-text body. The server enforces a maximum of 16,384 UTF-8 bytes and rejects NUL.
        html:
          type: string
          maxLength: 65536
          pattern: '^[^\u0000]*$'
          description: Optional HTML alternative. The server enforces a maximum of 65,536 UTF-8 bytes and rejects NUL; callers must keep text as the plain-text fallback.
    ServiceNotificationEmailResponse:
      type: object
      additionalProperties: false
      required: [ status, recipient_count ]
      properties:
        status:
          type: string
          const: sent
        recipient_count:
          type: integer
          minimum: 1
          maximum: 20
          description: Number of recipients submitted without returning their addresses.
    StreamArtifact:
      type: object
      additionalProperties: false
      required: [ kind, name, relative_path, size_bytes ]
      oneOf:
        - properties: { kind: { const: archive }, name: { const: final.mp4 } }
        - properties: { kind: { const: caption }, name: { const: captions.vtt } }
        - properties: { kind: { const: transcript }, name: { const: transcript.json } }
        - properties: { kind: { const: metadata }, name: { const: metadata.json } }
        - properties: { kind: { const: logs }, name: { const: logs.jsonl } }
      properties:
        kind:
          enum: [ archive, caption, transcript, metadata, logs ]
        name:
          type: string
          minLength: 1
          maxLength: 255
          pattern: "^[^/\\\\]+$"
        relative_path:
          type: string
          minLength: 1
          maxLength: 1024
          pattern: "^final/[^/\\\\]+/[^/\\\\]+/[^/\\\\]+$"
          description: Run-scoped final/{stream_id}/{archive_run_id}/{name} path; the server verifies the embedded identifiers and file name.
        size_bytes:
          type: integer
          minimum: 0
    ServiceArtifactReport:
      type: object
      additionalProperties: false
      required: [ service_id, stream_id, archive_run_id, archive_started_at, artifacts ]
      dependentRequired:
        archive_run_id: [ archive_started_at ]
        archive_started_at: [ archive_run_id ]
      properties:
        service_id:
          type: string
          minLength: 1
        stream_id:
          type: string
          minLength: 1
        archive_run_id:
          type: string
          minLength: 1
          maxLength: 128
          pattern: "^[A-Za-z0-9][A-Za-z0-9._-]*$"
          not: { pattern: "\\.\\." }
          description: Stable identifier for one broadcast/archive run of the stream.
        archive_started_at:
          type: string
          format: date-time
          description: Start timestamp of the archive run.
        artifacts:
          type: array
          minItems: 1
          maxItems: 20
          items:
            $ref: "#/components/schemas/StreamArtifact"
    ControlNotificationChannelUpdateRequest:
      type: object
      additionalProperties: false
      required: [ name, type, enabled ]
      description: Control Panel notification-channel update. Per-channel SMTP fields and uses_global_smtp are not accepted. migrate_to_global_smtp is the only supported delivery-mode transition; true migrates an existing legacy email channel to globally managed application SMTP, while omission or false preserves the current mode. Omitting email_recipients preserves the existing masked recipient set.
      properties:
        name:
          type: string
          minLength: 1
        type:
          enum: [ discord, slack, generic, email ]
        enabled:
          type: boolean
        webhook_url:
          type: string
          format: uri
          pattern: "^https?://"
          writeOnly: true
          description: Optional write-only webhook replacement. Omission preserves the existing encrypted URL; it is never returned raw.
        email_recipients:
          type: array
          minItems: 1
          maxItems: 20
          uniqueItems: true
          items:
            type: string
            format: email
            maxLength: 320
            pattern: '^[^\r\n]+$'
            description: Bare mailbox address only; display names and CR/LF are rejected.
          description: Optional replacement recipient set. Omission preserves existing recipients; an explicitly empty array is invalid.
        migrate_to_global_smtp:
          type: boolean
          writeOnly: true
          description: Set true to migrate an existing legacy email channel from per-channel SMTP to globally managed application SMTP. Omission or false preserves the current delivery mode; this request flag is never returned.
        severity_filter:
          type: array
          uniqueItems: true
          items:
            enum: [ info, warning, error, critical ]
        event_type_filter:
          type: array
          uniqueItems: true
          items:
            type: string
    ControlNotificationChannelCreateRequest:
      description: Control Panel notification-channel creation. Email channels always use globally managed application SMTP settings, enforced by the backend. Per-channel SMTP fields and uses_global_smtp are not accepted.
      allOf:
        - $ref: "#/components/schemas/ControlNotificationChannelUpdateRequest"
        - not:
            required: [ migrate_to_global_smtp ]
        - if:
            required: [ type ]
            properties:
              type:
                const: email
          then:
            required: [ email_recipients ]
            properties:
              email_recipients: {}
    NotificationDeliveryResult:
      type: object
      additionalProperties: false
      required: [ event_type, channel, target, status ]
      properties:
        event_type:
          type: string
          enum: [ stream.started, stream.live, stream.completed, stream.failed, stream.warning, stream.error, incident.opened, incident.updated, incident.resolved, diagnostic.created, remediation.pending_approval, remediation.executed, archive.upload.completed, archive.upload.failed, service.offline, service.recovered, admin.audit ]
        channel:
          type: string
          enum: [ discord, slack, generic, email ]
        target:
          type: string
          minLength: 1
          maxLength: 512
          pattern: '^[^\u0000-\u001F\u007F]+$'
          description: Masked delivery target only. Raw webhook URLs and email recipient addresses are never returned.
        status:
          type: string
          enum: [ success, failure ]
        error:
          type: string
          maxLength: 512
          pattern: '^[^\u0000-\u001F\u007F]*$'
          description: Optional sanitized error. Raw provider responses, target URLs, credentials, and tokens are never returned.
    NotificationChannel:
      type: object
      additionalProperties: false
      required: [ id, name, type, enabled, uses_global_smtp, created_at, updated_at ]
      properties:
        id:
          type: string
        name:
          type: string
        type:
          enum: [ discord, slack, generic, email ]
        enabled:
          type: boolean
        uses_global_smtp:
          type: boolean
          description: Whether this email channel delegates delivery to Control Panel using globally managed application SMTP settings. This reveals no SMTP credentials.
        masked_webhook_url:
          type: string
          description: Raw webhook URL is never returned.
        masked_email_target:
          type: string
          description: Raw email recipients and SMTP settings are never returned as unmasked response fields.
        severity_filter:
          type: array
          items:
            enum: [ info, warning, error, critical ]
        event_type_filter:
          type: array
          items:
            type: string
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
