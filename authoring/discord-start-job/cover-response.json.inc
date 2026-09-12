    "encoderVideoCoverApplyResponse": {
      "type": "object",
      "additionalProperties": false,
      "x-autostream-behavioral-validator": "contracts.ValidateEncoderVideoCoverApplyResponse",
      "x-autostream-applied-linkage": [
        "stream_id=actual.stream_id",
        "job_generation=actual.job_generation",
        "actual_generation=actual.generation=actual.applied_witness.generation",
        "requested_revision=actual.desired.revision=actual.applied.revision=actual.cover.revision=actual.applied_witness.revision=actual.applied_witness.cover.revision",
        "actual.desired.active=actual.applied.active=actual.cover.enabled=actual.applied_witness.active=actual.applied_witness.cover.enabled",
        "actual.cover=actual.applied_witness.cover",
        "actual.watermark=actual.applied_witness.watermark",
        "actual.pipeline=actual.applied_witness.pipeline"
      ],
      "required": [
        "stream_id",
        "job_generation",
        "requested_revision",
        "actual_generation",
        "accepted",
        "rejected",
        "applied",
        "outcome",
        "actual"
      ],
      "properties": {
        "stream_id": { "type": "string", "minLength": 1 },
        "job_generation": { "type": "integer", "minimum": 1 },
        "requested_revision": { "type": "integer", "minimum": 1 },
        "actual_generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Encoder-owned graph instance generation observed when producing this response.",
          "x-autostream-owner": "encoder_recorder",
          "x-autostream-scope": "job_generation"
        },
        "accepted": { "type": "boolean" },
        "rejected": { "type": "boolean" },
        "applied": { "type": "boolean" },
        "outcome": { "enum": ["applied", "rejected", "ambiguous"] },
        "actual": { "$ref": "#/$defs/videoCoverRuntimeState" },
        "error": { "$ref": "#/$defs/safeError" }
      },
      "allOf": [
        {
          "if": {
            "required": ["outcome"],
            "properties": { "outcome": { "const": "applied" } }
          },
          "then": {
            "properties": {
              "accepted": { "const": true },
              "rejected": { "const": false },
              "applied": { "const": true },
              "actual": {
                "required": ["readiness", "applied_witness"],
                "properties": { "readiness": { "const": "ready" } }
              }
            },
            "not": { "required": ["error"] }
          }
        },
        {
          "if": {
            "required": ["outcome"],
            "properties": { "outcome": { "const": "rejected" } }
          },
          "then": {
            "required": ["error"],
            "properties": {
              "accepted": { "const": false },
              "rejected": { "const": true },
              "applied": { "const": false }
            }
          }
        },
        {
          "if": {
            "required": ["outcome"],
            "properties": { "outcome": { "const": "ambiguous" } }
          },
          "then": {
            "required": ["error"],
            "properties": {
              "accepted": { "const": true },
              "rejected": { "const": false },
              "applied": { "const": false },
              "actual": {
                "required": ["readiness", "applied"],
                "properties": {
                  "readiness": { "const": "unknown" },
                  "applied": {
                    "required": ["state"],
                    "properties": { "state": { "const": "unknown" } }
                  }
                }
              }
            }
          }
        }
      ],
      "description": "Strict Encoder apply result. Applied=true is impossible without an actual runtime state carrying a graph-applied witness. Cross-field equality is enforced by the named behavioral validator because JSON Schema cannot express value equality between object locations."
    },
    "videoCoverStateRequest": {
      "type": "object",
      "additionalProperties": false,
      "required": ["active", "expected_job_generation", "expected_revision", "idempotency_key"],
      "properties": {
        "active": { "type": "boolean" },
        "expected_job_generation": { "type": "integer", "minimum": 1 },
        "expected_revision": { "type": "integer", "minimum": 1 },
        "idempotency_key": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF](?:[^\\x00-\\x1F\\x7F]*[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF])?$"
        },
        "hide_confirmed": { "const": true }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": false } }
          },
          "then": { "required": ["hide_confirmed"] },
          "else": { "not": { "required": ["hide_confirmed"] } }
        }
      ],
      "description": "Revisioned public Cover state CAS. Ambiguous results are reconciled through state read and never automatically resent."
    },
