    "readiness": {
      "enum": ["ready", "not_ready", "unknown"]
    },
    "safeError": {
      "type": "object",
      "additionalProperties": false,
      "required": ["code"],
      "properties": {
        "code": {
          "enum": [
            "invalid_theme_id",
            "media_asset_format_unsupported",
            "media_asset_too_large",
            "media_asset_decode_failed",
            "media_asset_aspect_ratio_invalid",
            "media_asset_variant_processing",
            "media_asset_variant_failed",
            "media_asset_unauthorized",
            "media_asset_not_found",
            "media_asset_hash_mismatch",
            "media_asset_dimension_mismatch",
            "media_asset_timeout",
            "discord_target_invalid",
            "preset_not_found",
            "preset_revision_conflict",
            "stale_job_generation",
            "stale_cover_generation",
            "stale_cover_revision",
            "idempotency_conflict",
            "cover_apply_ambiguous",
            "cover_graph_unavailable",
            "revision_payload_conflict",
            "capability_required"
          ]
        },
        "request_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        }
      }
    },
