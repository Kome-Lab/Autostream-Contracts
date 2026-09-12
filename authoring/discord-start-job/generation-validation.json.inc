  "allOf": [
    {
      "if": { "required": ["schema_version"] },
      "then": {
        "required": ["schema_version", "discord_target"],
        "not": {
          "anyOf": [
            { "required": ["guild_id"] },
            { "required": ["text_channel_id"] },
            { "required": ["voice_channel_id"] }
          ]
        }
      },
      "else": {
        "required": ["guild_id", "voice_channel_id"],
        "not": { "required": ["discord_target"] }
      }
    }
  ],
