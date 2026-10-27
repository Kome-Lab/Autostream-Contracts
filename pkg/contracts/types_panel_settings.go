package contracts

// PanelEncoderRuntimeSettingsWriteRequest is the Control Panel persistence/CAS
// request. The downstream Encoder request remains EncoderRuntimeSettingsRequest;
// its Node contract does not acquire the Panel's editing authority.
type PanelEncoderRuntimeSettingsWriteRequest struct {
	ExpectedSettingsRevision *string `json:"expected_settings_revision,omitempty"`
	EncoderAudioGainDB       float64 `json:"encoder_audio_gain_db"`
	OverlayProfileID         string  `json:"overlay_profile_id"`
}
