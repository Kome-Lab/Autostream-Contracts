    "videoCoverStartSnapshot": {
      "type": "object",
      "additionalProperties": false,
      "required": ["job_generation", "revision", "active", "idempotency_key"],
      "properties": {
        "job_generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Control Panel-owned stream visual start epoch shared with Scene Appearance generation.",
          "x-autostream-owner": "control_panel",
          "x-autostream-epoch": "video_cover_job_generation"
        },
        "revision": {
          "type": "integer",
          "minimum": 1,
          "description": "Desired Video Cover revision captured for this start."
        },
        "active": { "type": "boolean" },
        "idempotency_key": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF](?:[^\\x00-\\x1F\\x7F]*[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF])?$"
        },
        "cover_asset": {
          "$ref": "#/$defs/mediaAssetDescriptor",
          "properties": {
            "usage": { "const": "video_cover" },
            "readiness": { "const": "ready" }
          }
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": true } }
          },
          "then": { "required": ["cover_asset"] },
          "else": { "not": { "required": ["cover_asset"] } }
        }
      ],
      "description": "Desired Video Cover start snapshot. New Control Panel producers send it for both active and inactive desired state so Encoder binds job_generation and revision before GET/reconcile. active=true requires cover_asset; active=false forbids cover_asset. Whole-field omission remains only for legacy callers during compatibility."
    },
    "encoderVideoCoverApplyRequest": {
      "type": "object",
      "additionalProperties": false,
      "x-autostream-path-body-equality": {
        "path_parameter": "id",
        "body_property": "stream_id",
        "comparison": "exact",
        "validator": "contracts.ValidateEncoderVideoCoverApplyRequest"
      },
      "required": [
        "stream_id",
        "job_generation",
        "expected_generation",
        "revision",
        "active",
        "idempotency_key"
      ],
      "properties": {
        "stream_id": { "type": "string", "minLength": 1 },
        "job_generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Control Panel-owned stream visual start epoch."
        },
        "expected_generation": {
          "type": "integer",
          "minimum": 1,
          "description": "CAS fence for the Encoder-owned graph instance generation currently observed by Control Panel.",
          "x-autostream-owner": "encoder_recorder",
          "x-autostream-fence": "exact_current_generation"
        },
        "revision": {
          "type": "integer",
          "minimum": 1,
          "description": "Desired Video Cover revision."
        },
        "active": { "type": "boolean" },
        "idempotency_key": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF](?:[^\\x00-\\x1F\\x7F]*[^\\x00-\\x20\\x7F\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF])?$"
        },
        "cover_asset": {
          "$ref": "#/$defs/mediaAssetDescriptor",
          "properties": {
            "usage": { "const": "video_cover" },
            "readiness": { "const": "ready" }
          }
        },
        "hide_confirmed": {
          "const": true,
          "description": "Required for hide. An ambiguous hide outcome is reconciled by GET and is never automatically resent."
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": true } }
          },
          "then": {
            "required": ["cover_asset"],
            "not": { "required": ["hide_confirmed"] }
          },
          "else": {
            "required": ["hide_confirmed"],
            "not": { "required": ["cover_asset"] }
          }
        }
      ],
      "description": "Strict Control Panel to Encoder Video Cover CAS. It can mutate only the Cover layer; Watermark fields are intentionally not representable. OpenAPI path id and body stream_id equality is enforced by the named behavioral validator because JSON Schema cannot express that relation."
    },
    "videoCoverAppliedWitness": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "graph_applied",
        "generation",
        "revision",
        "active",
        "cover",
        "watermark",
        "pipeline"
      ],
      "properties": {
        "graph_applied": {
          "const": true,
          "description": "True only after the Encoder graph has applied the Cover state. Acceptance or asset validation alone cannot produce this witness."
        },
        "generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Encoder-owned graph instance generation that applied this state."
        },
        "revision": { "type": "integer", "minimum": 1 },
        "active": { "type": "boolean" },
        "cover": { "$ref": "#/$defs/layerState" },
        "watermark": {
          "$ref": "#/$defs/layerState",
          "description": "Independently observed Watermark layer; a Cover mutation cannot change it."
        },
        "pipeline": { "$ref": "#/$defs/visualPipelineInvariant" }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": true } }
          },
          "then": {
            "properties": {
              "cover": {
                "required": ["enabled", "variant_id"],
                "properties": { "enabled": { "const": true } }
              }
            }
          },
          "else": {
            "properties": {
              "cover": {
                "required": ["enabled"],
                "properties": { "enabled": { "const": false } },
                "not": { "required": ["variant_id"] }
              }
            }
          }
        }
      ]
    },
