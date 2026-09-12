    "sceneAppearance": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "generation",
        "revision",
        "capability",
        "readiness",
        "background_mode",
        "header_title_mode"
      ],
      "properties": {
        "generation": {
          "type": "integer",
          "minimum": 1,
          "description": "Control Panel-owned stream visual runtime generation. It advances for every stream start and is the same epoch value as Video Cover job_generation for that start; it is not the Worker-issued job generation.",
          "x-autostream-owner": "control_panel",
          "x-autostream-epoch": "video_cover_job_generation",
          "x-autostream-advance": "every_stream_start"
        },
        "revision": {
          "type": "integer",
          "minimum": 1,
          "description": "Exact stream_visual_settings.revision captured in this immutable start snapshot.",
          "x-autostream-source": "stream_visual_settings.revision"
        },
        "capability": {
          "const": "scene_appearance_v1"
        },
        "readiness": {
          "$ref": "#/$defs/readiness"
        },
        "background_mode": {
          "enum": ["default", "image"]
        },
        "background": {
          "$ref": "#/$defs/mediaAssetDescriptor",
          "properties": { "usage": { "const": "scene_background" } }
        },
        "header_title_mode": {
          "enum": ["default", "custom"]
        },
        "custom_title": {
          "type": "string",
          "minLength": 1,
          "maxLength": 80,
          "pattern": "^[^\\r\\n]+$"
        },
        "error": {
          "$ref": "#/$defs/safeError"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["background_mode"],
            "properties": {
              "background_mode": {
                "const": "image"
              }
            }
          },
          "then": {
            "required": ["background"]
          },
          "else": {
            "not": {
              "required": ["background"]
            }
          }
        },
        {
          "if": {
            "required": ["header_title_mode"],
            "properties": {
              "header_title_mode": {
                "const": "custom"
              }
            }
          },
          "then": {
            "required": ["custom_title"]
          },
          "else": {
            "not": {
              "required": ["custom_title"]
            }
          }
        },
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
            "required": ["readiness", "background_mode"],
            "properties": {
              "readiness": { "const": "ready" },
              "background_mode": { "const": "image" }
            }
          },
          "then": {
            "properties": {
              "background": {
                "required": ["readiness"],
                "properties": { "readiness": { "const": "ready" } }
              }
            }
          }
        },
        {
          "if": {
            "required": ["background"],
            "properties": {
              "background": {
                "required": ["readiness"],
                "properties": { "readiness": { "enum": ["not_ready", "unknown"] } }
              }
            }
          },
          "then": {
            "properties": { "readiness": { "enum": ["not_ready", "unknown"] } },
            "required": ["error"]
          }
        }
      ]
    },
