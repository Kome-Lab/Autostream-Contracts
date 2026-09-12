  "properties": {
    "schema_version": {
      "const": 2
    },
    "stream_id": {
      "type": "string",
      "minLength": 1
    },
    "job_generation": {
      "type": "integer",
      "minimum": 1,
      "description": "Worker-issued positive uint64 generation fence for this exact stream job. Control Panel must pass it through unchanged, and consumers must reject missing, zero, or values outside the uint64 range before starting the job."
    },
    "discord_target": {
      "$ref": "#/$defs/discordTargetSnapshot"
    },
    "guild_id": {
      "type": "string",
      "minLength": 1,
      "description": "Temporary legacy adapter field removed in Execution Bundle 8. v2 payloads cannot include it."
    },
    "text_channel_id": {
      "type": "string",
      "minLength": 1,
      "description": "Temporary legacy adapter field removed in Execution Bundle 8. v2 payloads cannot include it."
    },
    "voice_channel_id": {
      "type": "string",
      "minLength": 1,
      "description": "Temporary legacy adapter field removed in Execution Bundle 8. v2 payloads cannot include it."
    },
    "encoder_audio_url": {
      "type": "string",
      "format": "uri",
      "description": "Encoder/Recorder audio ingest endpoint. Treat as runtime-only and do not expose it in public status or logs."
    },
    "caption_audio_url": {
      "type": "string",
      "format": "uri",
      "description": "Optional caption/STT audio ingest endpoint. Treat as runtime-only and do not expose it in public status or logs."
    },
    "caption_audio_token": {
      "type": "string",
      "minLength": 1,
      "writeOnly": true,
      "description": "Short-lived stream-scoped token authorizing Discord Opus ingest at the selected Worker. It must never be returned by status endpoints, logs, audit metadata, or frontend API responses."
    },
    "caption_audio_flush_ms": {
      "type": "integer",
      "minimum": 10,
      "maximum": 1000,
      "description": "Optional start snapshot of caption_profile_config.caption_audio_flush_ms. Bounds are owned by the canonical Caption Profile contract."
    },
    "caption_audio_max_batch_packets": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "description": "Optional start snapshot of caption_profile_config.caption_audio_max_batch_packets. Bounds are owned by the canonical Caption Profile contract."
    },
    "unresolved_ssrc_buffer_ms": {
      "type": "integer",
      "minimum": 0,
      "maximum": 5000,
      "description": "Optional start snapshot of caption_profile_config.unresolved_ssrc_buffer_ms. Bounds are owned by the canonical Caption Profile contract."
    },
    "stream_ingest_token": {
      "type": "string",
      "minLength": 1,
      "writeOnly": true,
      "description": "Short-lived stream-scoped ingest token issued by Control Panel. It must never be returned by status endpoints, logs, audit metadata, or frontend API responses."
    },
    "worker_events_url": {
      "type": "string",
      "format": "uri",
      "description": "Worker event endpoint selected by this stream's Worker assignment. Treat as runtime-only and do not expose it in public status or logs."
    },
    "worker_events_token": {
      "type": "string",
      "minLength": 1,
      "writeOnly": true,
      "description": "Short-lived stream-scoped token for Discord Bot to publish participant, active speaker, and chat events to the selected Worker. It must never be returned by status endpoints, logs, audit metadata, or frontend API responses."
    }
  }
}
