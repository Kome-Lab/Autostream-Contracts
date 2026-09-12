    "themePreference": {
      "type": "object",
      "additionalProperties": false,
      "required": ["theme_id", "color_mode", "revision", "readiness"],
      "properties": {
        "theme_id": {
          "enum": [
            "autostream",
            "slate",
            "ocean",
            "cyan",
            "indigo",
            "violet",
            "magenta",
            "rose",
            "crimson",
            "amber",
            "emerald",
            "monochrome"
          ]
        },
        "color_mode": {
          "enum": ["system", "light", "dark"]
        },
        "revision": {
          "type": "integer",
          "minimum": 1
        },
        "readiness": {
          "$ref": "#/$defs/readiness"
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
        }
      ]
    },
    "themePreferenceWrite": {
      "type": "object",
      "additionalProperties": false,
      "required": ["theme_id", "color_mode", "expected_revision"],
      "properties": {
        "theme_id": {
          "enum": ["autostream", "slate", "ocean", "cyan", "indigo", "violet", "magenta", "rose", "crimson", "amber", "emerald", "monochrome"]
        },
        "color_mode": { "enum": ["system", "light", "dark"] },
        "expected_revision": { "type": "integer", "minimum": 0 }
      },
      "description": "Authenticated self-update CAS input. Revision mismatch is revision_conflict; admin reset uses the existing users.update authority."
    },
    "mediaAssetDescriptor": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "asset_id",
        "variant_id",
        "usage",
        "media_type",
        "width",
        "height",
        "byte_size",
        "pixel_count",
        "animated",
        "sha256",
        "revision",
        "readiness"
      ],
      "properties": {
        "asset_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        },
        "variant_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        },
        "usage": { "enum": ["scene_background", "video_cover"] },
        "media_type": {
          "enum": ["image/png", "image/jpeg", "image/webp"]
        },
        "width": {
          "type": "integer",
          "minimum": 1,
          "maximum": 8192
        },
        "height": {
          "type": "integer",
          "minimum": 1,
          "maximum": 8192
        },
        "byte_size": {
          "type": "integer",
          "minimum": 1,
          "maximum": 20971520
        },
        "pixel_count": {
          "type": "integer",
          "minimum": 1,
          "maximum": 40000000,
          "description": "The producer must verify this equals decoded width multiplied by height before declaring readiness."
        },
        "animated": {
          "const": false
        },
        "aspect_ratio_error_ppm": {
          "type": "integer",
          "minimum": 0,
          "maximum": 1000
        },
        "opaque": { "type": "boolean" },
        "sha256": {
          "type": "string",
          "pattern": "^[a-f0-9]{64}$",
          "description": "Public integrity digest only; never an authentication credential."
        },
        "revision": {
          "type": "integer",
          "minimum": 1,
          "description": "Immutable processed-variant revision. It is copied from media_asset_variants.processor_revision and never changes for an existing variant identity.",
          "x-autostream-source": "media_asset_variants.processor_revision",
          "x-autostream-immutability": "processed_variant"
        },
        "readiness": {
          "$ref": "#/$defs/readiness"
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
            "required": ["usage"],
            "properties": { "usage": { "const": "video_cover" } }
          },
          "then": {
            "required": ["aspect_ratio_error_ppm", "opaque"],
            "properties": { "opaque": { "const": true } }
          },
          "else": {
            "not": {
              "anyOf": [
                { "required": ["aspect_ratio_error_ppm"] },
                { "required": ["opaque"] }
              ]
            }
          }
        }
      ],
      "description": "Static PNG, JPEG, or WebP metadata only. SVG, GIF, animation, filesystem paths, URLs, and Base64 payloads are not representable."
    },
