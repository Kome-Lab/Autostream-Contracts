    "resolvedDiscordTarget": {
      "type": "object",
      "additionalProperties": false,
      "required": ["guild_id", "text_channel_id", "voice_channel_id"],
      "properties": {
        "guild_id": {
          "type": "string",
          "pattern": "^[0-9]{1,32}$"
        },
        "text_channel_id": {
          "type": "string",
          "pattern": "^[0-9]{1,32}$"
        },
        "voice_channel_id": {
          "type": "string",
          "pattern": "^[0-9]{1,32}$"
        }
      }
    },
    "discordTargetSelection": {
      "type": "object",
      "additionalProperties": false,
      "required": ["mode"],
      "properties": {
        "mode": { "enum": ["inherit", "preset", "manual"] },
        "preset_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        },
        "manual_target": { "$ref": "#/$defs/resolvedDiscordTarget" }
      },
      "allOf": [
        {
          "if": {
            "required": ["mode"],
            "properties": { "mode": { "const": "preset" } }
          },
          "then": { "required": ["preset_id"] },
          "else": { "not": { "required": ["preset_id"] } }
        },
        {
          "if": {
            "required": ["mode"],
            "properties": { "mode": { "const": "manual" } }
          },
          "then": { "required": ["manual_target"] },
          "else": { "not": { "required": ["manual_target"] } }
        }
      ]
    },
    "discordTargetStoredSnapshot": {
      "type": "object",
      "additionalProperties": false,
      "required": ["mode", "revision", "resolved"],
      "properties": {
        "mode": { "enum": ["inherit", "preset", "manual"] },
        "preset_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        },
        "preset_revision": { "type": "integer", "minimum": 1 },
        "revision": { "type": "integer", "minimum": 1 },
        "resolved": { "$ref": "#/$defs/resolvedDiscordTarget" }
      },
      "allOf": [
        {
          "if": {
            "required": ["mode"],
            "properties": { "mode": { "const": "preset" } }
          },
          "then": { "required": ["preset_id", "preset_revision"] },
          "else": {
            "not": {
              "anyOf": [
                { "required": ["preset_id"] },
                { "required": ["preset_revision"] }
              ]
            }
          }
        }
      ],
      "description": "Control Panel-owned immutable stream snapshot. Preset edits do not mutate this record without an explicit revisioned action."
    },
    "discordTargetSnapshot": {
      "type": "object",
      "additionalProperties": false,
      "required": ["revision", "resolved"],
      "properties": {
        "revision": { "type": "integer", "minimum": 1 },
        "resolved": { "$ref": "#/$defs/resolvedDiscordTarget" }
      },
      "description": "Bot-facing resolved snapshot. Preset IDs, inheritance metadata, and manual selection inputs are deliberately absent."
    },
