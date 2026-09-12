    "visualPipelineInvariant": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "layers",
        "watermark_topmost",
        "cover_watermark_independent",
        "output_parity",
        "audio_continuity"
      ],
      "properties": {
        "layers": {
          "type": "array",
          "prefixItems": [
            {
              "const": "base_or_worker_scene"
            },
            {
              "const": "video_cover"
            },
            {
              "const": "watermark"
            },
            {
              "const": "video_encode"
            },
            {
              "const": "tee_live_archive_preview"
            }
          ],
          "items": false,
          "minItems": 5,
          "maxItems": 5
        },
        "watermark_topmost": {
          "const": true
        },
        "cover_watermark_independent": {
          "const": true
        },
        "output_parity": {
          "type": "array",
          "prefixItems": [
            {
              "const": "live"
            },
            {
              "const": "archive"
            },
            {
              "const": "preview"
            }
          ],
          "items": false,
          "minItems": 3,
          "maxItems": 3
        },
        "audio_continuity": {
          "type": "object",
          "additionalProperties": false,
          "required": [
            "process_restart",
            "audio_encoder_restart",
            "audio_mux_restart",
            "graph_rebuild",
            "reconnect",
            "sequence_loss",
            "timestamp_discontinuity",
            "intentional_mute_insertion"
          ],
          "properties": {
            "process_restart": {
              "const": 0
            },
            "audio_encoder_restart": {
              "const": 0
            },
            "audio_mux_restart": {
              "const": 0
            },
            "graph_rebuild": {
              "const": 0
            },
            "reconnect": {
              "const": 0
            },
            "sequence_loss": {
              "const": 0
            },
            "timestamp_discontinuity": {
              "const": 0
            },
            "intentional_mute_insertion": {
              "const": 0
            }
          }
        }
      }
    },
