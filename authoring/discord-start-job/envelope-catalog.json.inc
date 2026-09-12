{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "DiscordBotStartJobRequest",
  "description": "AutoStream Discord Bot start envelope. Control Panel may send schema_version=2 only when the assigned Bot's current capability map advertises discord_resolved_target_v2 as boolean true; a version string alone, an absent capability, false, or an unknown value is not support evidence. schema_version=2 delivers target IDs only as one server-resolved snapshot and rejects legacy flat target fields. Otherwise Control Panel sends the legacy flat target payload and omits every v2-only field. Payloads without schema_version retain a narrowly scoped compatibility branch in which legacy flat target fields remain accepted until Execution Bundle 8; the Bot never reads preset resources.",
  "type": "object",
  "required": ["stream_id", "job_generation"],
  "additionalProperties": false,
  "x-autostream-contract-major": 2,
  "x-autostream-v2-transition-compatibility": {
    "owner": "V2-COMPAT-EOL-CONTRACTS-001",
    "removal_wave": "Execution Bundle 8",
    "final_state": "absent"
  },
  "x-autostream-visual-contract-definitions": [
    "themePreference",
    "themePreferenceWrite",
    "mediaAssetDescriptor",
    "sceneAppearance",
    "discordTargetSelection",
    "discordTargetStoredSnapshot",
    "discordTargetSnapshot",
    "videoCoverStartSnapshot",
    "encoderVideoCoverApplyRequest",
    "encoderVideoCoverApplyResponse",
    "videoCoverAppliedWitness",
    "videoCoverRuntimeState",
    "videoCoverStateRequest",
    "visualPipelineInvariant"
  ],
  "x-autostream-visual-contract-catalog": {
    "theme_preference": { "$ref": "#/$defs/themePreference" },
    "theme_preference_write": { "$ref": "#/$defs/themePreferenceWrite" },
    "scene_appearance": { "$ref": "#/$defs/sceneAppearance" },
    "discord_target_selection": { "$ref": "#/$defs/discordTargetSelection" },
    "discord_target_stored_snapshot": { "$ref": "#/$defs/discordTargetStoredSnapshot" },
    "video_cover_start_snapshot": { "$ref": "#/$defs/videoCoverStartSnapshot" },
    "encoder_video_cover_apply_request": { "$ref": "#/$defs/encoderVideoCoverApplyRequest" },
    "encoder_video_cover_apply_response": { "$ref": "#/$defs/encoderVideoCoverApplyResponse" },
    "video_cover_state_request": { "$ref": "#/$defs/videoCoverStateRequest" },
    "video_cover_runtime_state": { "$ref": "#/$defs/videoCoverRuntimeState" }
  },
  "$defs": {
