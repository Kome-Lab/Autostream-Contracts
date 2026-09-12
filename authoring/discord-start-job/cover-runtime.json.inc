    "videoCoverRuntimeState": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "stream_id",
        "job_generation",
        "generation",
        "capability",
        "readiness",
        "desired",
        "applied",
        "cover",
        "watermark",
        "pipeline",
        "no_automatic_resend"
      ],
      "properties": {
        "stream_id": {
          "type": "string",
          "minLength": 1
        },
        "job_generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Control Panel-owned stream visual start epoch shared with Scene Appearance generation.",
          "x-autostream-owner": "control_panel",
          "x-autostream-epoch": "video_cover_job_generation"
        },
        "generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Encoder-owned graph instance generation. It increases monotonically within one job_generation whenever a new graph instance becomes authoritative.",
          "x-autostream-owner": "encoder_recorder",
          "x-autostream-scope": "job_generation",
          "x-autostream-monotonic": true
        },
        "capability": {
          "const": "live_video_cover_v1"
        },
        "readiness": {
          "$ref": "#/$defs/readiness"
        },
        "desired": {
          "$ref": "#/$defs/desiredCoverState"
        },
        "applied": {
          "$ref": "#/$defs/appliedCoverState"
        },
        "cover": {
          "$ref": "#/$defs/layerState",
          "description": "Current Cover imagefeed layer state, independent from Watermark."
        },
        "cover_asset": {
          "$ref": "#/$defs/mediaAssetDescriptor",
          "properties": { "usage": { "const": "video_cover" } }
        },
        "watermark": {
          "$ref": "#/$defs/layerState",
          "description": "Current Watermark imagefeed layer state, independent from Cover."
        },
        "pipeline": {
          "$ref": "#/$defs/visualPipelineInvariant"
        },
        "applied_witness": {
          "$ref": "#/$defs/videoCoverAppliedWitness"
        },
        "no_automatic_resend": {
          "const": true,
          "description": "Unknown or ambiguous applied state requires an explicit operator resend with a new revision."
        },
        "last_good_applied": {
          "$ref": "#/$defs/knownAppliedCoverState"
        },
        "error": {
          "$ref": "#/$defs/safeError"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["readiness"],
            "properties": { "readiness": { "enum": ["not_ready", "unknown"] } }
          },
          "then": { "required": ["error"] },
          "else": { "not": { "required": ["error"] } }
        },
        {
          "if": {
            "required": ["applied"],
            "properties": {
              "applied": {
                "required": ["state"],
                "properties": { "state": { "const": "unknown" } }
              }
            }
          },
          "then": {
            "required": ["last_good_applied", "error"],
            "properties": { "readiness": { "enum": ["not_ready", "unknown"] } },
            "not": { "required": ["applied_witness"] }
          }
        },
        {
          "if": {
            "required": ["applied"],
            "properties": {
              "applied": {
                "required": ["state"],
                "properties": { "state": { "const": "known" } }
              }
            }
          },
          "then": { "required": ["applied_witness"] }
        },
        {
          "if": {
            "required": ["readiness"],
            "properties": { "readiness": { "const": "ready" } }
          },
          "then": {
            "properties": {
              "applied": {
                "required": ["state"],
                "properties": { "state": { "const": "known" } }
              }
            }
          }
        },
        {
          "if": {
            "required": ["desired"],
            "properties": {
              "desired": {
                "required": ["active"],
                "properties": { "active": { "const": true } }
              }
            }
          },
          "then": { "required": ["cover_asset"] },
          "else": { "not": { "required": ["cover_asset"] } }
        }
      ]
    }
  },
