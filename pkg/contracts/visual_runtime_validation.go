package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"unicode"
	"unicode/utf8"

	contractschemas "github.com/example/autostream-contracts/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	errEncoderVideoCoverPathBodyLinkage = errors.New("encoder video cover path/body stream linkage invalid")
	errEncoderVideoCoverRequest         = errors.New("encoder video cover request invalid")
	errEncoderVideoCoverResponseStatus  = errors.New("encoder video cover response status class invalid")
	errEncoderVideoCoverResponseShape   = errors.New("encoder video cover response structure invalid")
	errEncoderVideoCoverResponseLinkage = errors.New("encoder video cover response linkage invalid")
	errEncoderVideoCoverRuntimeState    = errors.New("encoder video cover runtime state invalid")
	errEncoderVideoCoverSchema          = errors.New("encoder video cover canonical schema unavailable")
)

const (
	encoderVideoCoverCatalogSchemaName     = "discord-bot-start-job-request.schema.json"
	encoderVideoCoverCatalogSchemaID       = "https://schemas.autostream.example.com/discord-bot-start-job-request.schema.json"
	encoderVideoCoverRequestSchemaName     = "encoder-video-cover-apply-request.schema.json"
	encoderVideoCoverResponseSchemaName    = "encoder-video-cover-apply-response.schema.json"
	encoderVideoCoverRuntimeSchemaName     = "encoder-video-cover-runtime-state.schema.json"
	encoderVideoCoverUnavailableSchemaName = "encoder-video-cover-unavailable-response.schema.json"
)

var encoderVideoCoverCanonicalSchemas struct {
	once        sync.Once
	request     *jsonschema.Schema
	response    *jsonschema.Schema
	runtime     *jsonschema.Schema
	unavailable *jsonschema.Schema
	err         error
}

// ValidateEncoderVideoCoverApplyRequest is the single public request
// validation seam. It rejects malformed, duplicate, trailing, null, and
// unknown JSON; applies the canonical schema while field presence is intact;
// then enforces path linkage and typed semantic invariants.
func ValidateEncoderVideoCoverApplyRequest(pathStreamID string, payload []byte) error {
	requestSchema, _, _, _, err := loadEncoderVideoCoverCanonicalSchemas()
	if err != nil {
		return errEncoderVideoCoverSchema
	}
	var request EncoderVideoCoverApplyRequest
	document, err := decodeEncoderVideoCoverStrictJSON(payload, &request)
	if err != nil || requestSchema.Validate(document) != nil {
		return errEncoderVideoCoverRequest
	}
	return validateEncoderVideoCoverApplyRequestSemantics(pathStreamID, request)
}

// ValidateEncoderVideoCoverApplyResponse is the single public response
// validation seam. It preserves required-field presence through canonical
// schema validation before applying status-class and cross-field semantics.
func ValidateEncoderVideoCoverApplyResponse(statusCode int, payload []byte) error {
	_, responseSchema, _, unavailableSchema, err := loadEncoderVideoCoverCanonicalSchemas()
	if err != nil {
		return errEncoderVideoCoverSchema
	}
	if statusCode == 404 {
		var response EncoderVideoCoverUnavailableResponse
		document, err := decodeEncoderVideoCoverStrictJSON(payload, &response)
		if err != nil || unavailableSchema.Validate(document) != nil || response.Code != VisualErrorCapabilityRequired {
			return errEncoderVideoCoverResponseShape
		}
		return nil
	}
	var response EncoderVideoCoverApplyResponse
	document, err := decodeEncoderVideoCoverStrictJSON(payload, &response)
	if err != nil || responseSchema.Validate(document) != nil {
		return errEncoderVideoCoverResponseShape
	}
	return validateEncoderVideoCoverApplyResponseSemantics(statusCode, response)
}

// ValidateEncoderVideoCoverRuntimeState is the single public reconciliation
// response seam. It preserves required false and zero-valued field presence,
// binds the body stream to the requested path, and validates graph linkage.
func ValidateEncoderVideoCoverRuntimeState(pathStreamID string, payload []byte) error {
	_, _, runtimeSchema, _, err := loadEncoderVideoCoverCanonicalSchemas()
	if err != nil {
		return errEncoderVideoCoverSchema
	}
	var state VideoCoverRuntimeState
	document, err := decodeEncoderVideoCoverStrictJSON(payload, &state)
	if err != nil || runtimeSchema.Validate(document) != nil {
		return errEncoderVideoCoverRuntimeState
	}
	if pathStreamID == "" || state.StreamID == "" || pathStreamID != state.StreamID {
		return errEncoderVideoCoverPathBodyLinkage
	}
	if !validVideoCoverRuntimeState(state) {
		return errEncoderVideoCoverRuntimeState
	}
	return nil
}

func loadEncoderVideoCoverCanonicalSchemas() (*jsonschema.Schema, *jsonschema.Schema, *jsonschema.Schema, *jsonschema.Schema, error) {
	encoderVideoCoverCanonicalSchemas.once.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(denyEncoderVideoCoverExternalSchemaLoader{})
		for _, name := range []string{
			encoderVideoCoverCatalogSchemaName,
			encoderVideoCoverRequestSchemaName,
			encoderVideoCoverResponseSchemaName,
			encoderVideoCoverRuntimeSchemaName,
			encoderVideoCoverUnavailableSchemaName,
		} {
			body, err := contractschemas.RuntimeValidationFS.ReadFile(name)
			if err != nil {
				encoderVideoCoverCanonicalSchemas.err = err
				return
			}
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
			if err != nil {
				encoderVideoCoverCanonicalSchemas.err = err
				return
			}
			if err := compiler.AddResource(name, document); err != nil {
				encoderVideoCoverCanonicalSchemas.err = err
				return
			}
			var identity struct {
				ID string `json:"$id"`
			}
			if err := json.Unmarshal(body, &identity); err != nil {
				encoderVideoCoverCanonicalSchemas.err = err
				return
			}
			if identity.ID != "" && identity.ID != name {
				if err := compiler.AddResource(identity.ID, document); err != nil {
					encoderVideoCoverCanonicalSchemas.err = err
					return
				}
			}
			if name == encoderVideoCoverCatalogSchemaName {
				if err := compiler.AddResource(encoderVideoCoverCatalogSchemaID, document); err != nil {
					encoderVideoCoverCanonicalSchemas.err = err
					return
				}
			}
		}
		encoderVideoCoverCanonicalSchemas.request, encoderVideoCoverCanonicalSchemas.err = compiler.Compile(encoderVideoCoverRequestSchemaName)
		if encoderVideoCoverCanonicalSchemas.err != nil {
			return
		}
		encoderVideoCoverCanonicalSchemas.response, encoderVideoCoverCanonicalSchemas.err = compiler.Compile(encoderVideoCoverResponseSchemaName)
		if encoderVideoCoverCanonicalSchemas.err != nil {
			return
		}
		encoderVideoCoverCanonicalSchemas.runtime, encoderVideoCoverCanonicalSchemas.err = compiler.Compile(encoderVideoCoverRuntimeSchemaName)
		if encoderVideoCoverCanonicalSchemas.err != nil {
			return
		}
		encoderVideoCoverCanonicalSchemas.unavailable, encoderVideoCoverCanonicalSchemas.err = compiler.Compile(encoderVideoCoverUnavailableSchemaName)
	})
	return encoderVideoCoverCanonicalSchemas.request, encoderVideoCoverCanonicalSchemas.response,
		encoderVideoCoverCanonicalSchemas.runtime, encoderVideoCoverCanonicalSchemas.unavailable, encoderVideoCoverCanonicalSchemas.err
}

type denyEncoderVideoCoverExternalSchemaLoader struct{}

func (denyEncoderVideoCoverExternalSchemaLoader) Load(string) (any, error) {
	return nil, errEncoderVideoCoverSchema
}

func decodeEncoderVideoCoverStrictJSON(payload []byte, target any) (any, error) {
	if !utf8.Valid(payload) {
		return nil, errEncoderVideoCoverResponseShape
	}
	if err := validateEncoderVideoCoverJSONTokens(payload); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, err
	}
	if err := requireEncoderVideoCoverJSONEOF(decoder); err != nil {
		return nil, err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return document, nil
}

func validateEncoderVideoCoverJSONTokens(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := validateEncoderVideoCoverJSONValue(decoder); err != nil {
		return err
	}
	return requireEncoderVideoCoverJSONEOF(decoder)
}

func validateEncoderVideoCoverJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return errEncoderVideoCoverResponseShape
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errEncoderVideoCoverResponseShape
			}
			if _, duplicate := seen[key]; duplicate {
				return errEncoderVideoCoverResponseShape
			}
			seen[key] = struct{}{}
			if err := validateEncoderVideoCoverJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errEncoderVideoCoverResponseShape
		}
	case '[':
		for decoder.More() {
			if err := validateEncoderVideoCoverJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errEncoderVideoCoverResponseShape
		}
	default:
		return errEncoderVideoCoverResponseShape
	}
	return nil
}

func requireEncoderVideoCoverJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return errEncoderVideoCoverResponseShape
	}
	return nil
}

// validateEncoderVideoCoverApplyRequestSemantics may only be called after the
// public JSON seam has completed canonical schema validation while presence is
// intact.
func validateEncoderVideoCoverApplyRequestSemantics(pathStreamID string, request EncoderVideoCoverApplyRequest) error {
	if pathStreamID == "" || request.StreamID == "" || pathStreamID != request.StreamID {
		return errEncoderVideoCoverPathBodyLinkage
	}
	if request.JobGeneration == 0 || request.ExpectedGeneration == 0 || request.Revision == 0 ||
		!validVisualIdempotencyKey(request.IdempotencyKey) {
		return errEncoderVideoCoverRequest
	}
	if request.Active {
		if request.HideConfirmed || request.CoverAsset == nil ||
			!validMediaAssetDescriptor(*request.CoverAsset) ||
			request.CoverAsset.Usage != "video_cover" || request.CoverAsset.Readiness != VisualReadinessReady {
			return errEncoderVideoCoverRequest
		}
		return nil
	}
	if !request.HideConfirmed || request.CoverAsset != nil {
		return errEncoderVideoCoverRequest
	}
	return nil
}

// validateEncoderVideoCoverApplyResponseSemantics may only be called after the
// public JSON seam has completed canonical schema validation while presence is
// intact.
func validateEncoderVideoCoverApplyResponseSemantics(statusCode int, response EncoderVideoCoverApplyResponse) error {
	if response.StreamID == "" || response.JobGeneration == 0 || response.RequestedRevision == 0 ||
		response.ActualGeneration == 0 || response.StreamID != response.Actual.StreamID ||
		response.JobGeneration != response.Actual.JobGeneration ||
		response.ActualGeneration != response.Actual.Generation ||
		!validVideoCoverRuntimeState(response.Actual) {
		return errEncoderVideoCoverResponseShape
	}
	if response.Error != nil && !validVisualSafeError(*response.Error) {
		return errEncoderVideoCoverResponseShape
	}

	switch statusCode {
	case 200:
		if response.Outcome != EncoderVideoCoverApplyOutcomeApplied || !response.Accepted || response.Rejected ||
			!response.Applied || response.Error != nil {
			return errEncoderVideoCoverResponseStatus
		}
		return validateAppliedVideoCoverResponseLinkage(response)
	case 202:
		if response.Outcome != EncoderVideoCoverApplyOutcomeAmbiguous || !response.Accepted || response.Rejected ||
			response.Applied || response.Error == nil || response.Error.Code != VisualErrorCoverApplyAmbiguous ||
			response.Actual.Readiness != VisualReadinessUnknown || response.Actual.Applied.State != "unknown" ||
			response.Actual.AppliedWitness != nil || response.Actual.Error == nil ||
			response.Actual.Error.Code != VisualErrorCoverApplyAmbiguous {
			return errEncoderVideoCoverResponseStatus
		}
		return nil
	case 409:
		if !isRejectedVideoCoverResponse(response) || !isVideoCoverFenceError(response.Error.Code) {
			return errEncoderVideoCoverResponseStatus
		}
		return nil
	case 502:
		if !isRejectedVideoCoverResponse(response) || !isVideoCoverGraphOrAssetError(response.Error.Code) ||
			response.Actual.Readiness == VisualReadinessReady || response.Actual.Error == nil ||
			response.Actual.Error.Code != response.Error.Code {
			return errEncoderVideoCoverResponseStatus
		}
		return nil
	default:
		return errEncoderVideoCoverResponseStatus
	}
}

func validateAppliedVideoCoverResponseLinkage(response EncoderVideoCoverApplyResponse) error {
	actual := response.Actual
	witness := actual.AppliedWitness
	if actual.Readiness != VisualReadinessReady || actual.Applied.State != "known" || actual.Applied.Active == nil ||
		witness == nil || response.ActualGeneration != witness.Generation {
		return errEncoderVideoCoverResponseLinkage
	}

	revision := response.RequestedRevision
	if actual.Desired.Revision != revision || actual.Applied.Revision != revision || actual.Cover.Revision != revision ||
		witness.Revision != revision || witness.Cover.Revision != revision {
		return errEncoderVideoCoverResponseLinkage
	}

	active := actual.Desired.Active
	if *actual.Applied.Active != active || actual.Cover.Enabled != active || witness.Active != active ||
		witness.Cover.Enabled != active || actual.Cover != witness.Cover || actual.Watermark != witness.Watermark ||
		!equalVisualPipelineInvariant(actual.Pipeline, witness.Pipeline) {
		return errEncoderVideoCoverResponseLinkage
	}

	variantID := actual.Desired.VariantID
	if actual.Applied.VariantID != variantID || actual.Cover.VariantID != variantID || witness.Cover.VariantID != variantID {
		return errEncoderVideoCoverResponseLinkage
	}
	if active {
		if actual.CoverAsset == nil || actual.CoverAsset.VariantID != variantID {
			return errEncoderVideoCoverResponseLinkage
		}
	} else if variantID != "" || actual.CoverAsset != nil {
		return errEncoderVideoCoverResponseLinkage
	}
	return nil
}

func validVideoCoverRuntimeState(state VideoCoverRuntimeState) bool {
	if state.StreamID == "" || state.JobGeneration == 0 || state.Generation == 0 ||
		state.Capability != CapabilityLiveVideoCoverV1 || !validVisualReadiness(state.Readiness) ||
		!validVideoCoverDesiredState(state.Desired) || !validVideoCoverAppliedState(state.Applied) ||
		!validVideoVisualLayerState(state.Cover) || !validVideoVisualLayerState(state.Watermark) ||
		!validVisualPipelineInvariant(state.Pipeline) || !state.NoAutomaticResend {
		return false
	}
	if state.Desired.Active {
		if state.CoverAsset == nil || !validMediaAssetDescriptor(*state.CoverAsset) ||
			state.CoverAsset.Usage != "video_cover" || state.CoverAsset.VariantID != state.Desired.VariantID {
			return false
		}
	} else if state.CoverAsset != nil {
		return false
	}
	if state.Error != nil && !validVisualSafeError(*state.Error) {
		return false
	}
	if state.Readiness == VisualReadinessReady {
		if state.Error != nil || state.Applied.State != "known" {
			return false
		}
	} else if state.Error == nil {
		return false
	}
	if state.LastGoodApplied != nil && !validKnownVideoCoverAppliedState(*state.LastGoodApplied) {
		return false
	}

	switch state.Applied.State {
	case "known":
		if state.AppliedWitness == nil || !validVideoCoverAppliedWitness(*state.AppliedWitness) ||
			state.Applied.Active == nil || state.AppliedWitness.Generation != state.Generation ||
			state.AppliedWitness.Revision != state.Applied.Revision ||
			state.AppliedWitness.Active != *state.Applied.Active || state.Cover != state.AppliedWitness.Cover ||
			state.Watermark != state.AppliedWitness.Watermark ||
			!equalVisualPipelineInvariant(state.Pipeline, state.AppliedWitness.Pipeline) ||
			state.Cover.Enabled != *state.Applied.Active || state.Cover.Revision != state.Applied.Revision ||
			state.Cover.VariantID != state.Applied.VariantID {
			return false
		}
		if state.Readiness == VisualReadinessReady &&
			(state.Desired.Active != *state.Applied.Active || state.Desired.Revision != state.Applied.Revision ||
				state.Desired.VariantID != state.Applied.VariantID) {
			return false
		}
	case "unknown":
		if state.Readiness == VisualReadinessReady || state.AppliedWitness != nil || state.LastGoodApplied == nil {
			return false
		}
	default:
		return false
	}
	return true
}

func validVideoCoverDesiredState(state VideoCoverDesiredState) bool {
	if state.Revision == 0 {
		return false
	}
	if state.Active {
		return (state.Source == "preset" || state.Source == "upload") && validVisualIdentifier(state.VariantID)
	}
	return state.Source == "none" && state.VariantID == ""
}

func validVideoCoverAppliedState(state VideoCoverAppliedState) bool {
	switch state.State {
	case "known":
		if state.Active == nil || state.Revision == 0 {
			return false
		}
		if *state.Active {
			return validVisualIdentifier(state.VariantID)
		}
		return state.VariantID == ""
	case "unknown":
		return state.Active == nil && state.Revision == 0 && state.VariantID == ""
	default:
		return false
	}
}

func validKnownVideoCoverAppliedState(state VideoCoverAppliedState) bool {
	return state.State == "known" && validVideoCoverAppliedState(state)
}

func validVideoVisualLayerState(state VideoVisualLayerState) bool {
	if state.Revision == 0 {
		return false
	}
	if state.Enabled {
		return validVisualIdentifier(state.VariantID)
	}
	return state.VariantID == ""
}

func validVisualPipelineInvariant(pipeline VisualPipelineInvariant) bool {
	return equalStrings(pipeline.Layers, []string{
		"base_or_worker_scene", "video_cover", "watermark", "video_encode", "tee_live_archive_preview",
	}) && pipeline.WatermarkTopmost && pipeline.CoverWatermarkIndependent &&
		equalStrings(pipeline.OutputParity, []string{"live", "archive", "preview"}) &&
		pipeline.AudioContinuity == (VisualAudioContinuity{})
}

func validVideoCoverAppliedWitness(witness VideoCoverAppliedWitness) bool {
	return witness.GraphApplied && witness.Generation > 0 && witness.Revision > 0 &&
		validVideoVisualLayerState(witness.Cover) && validVideoVisualLayerState(witness.Watermark) &&
		validVisualPipelineInvariant(witness.Pipeline) && witness.Active == witness.Cover.Enabled &&
		witness.Revision == witness.Cover.Revision
}

func validMediaAssetDescriptor(asset MediaAssetDescriptor) bool {
	if !validVisualIdentifier(asset.AssetID) || !validVisualIdentifier(asset.VariantID) ||
		(asset.Usage != "scene_background" && asset.Usage != "video_cover") ||
		(asset.MediaType != "image/png" && asset.MediaType != "image/jpeg" && asset.MediaType != "image/webp") ||
		asset.Width < 1 || asset.Width > 8192 || asset.Height < 1 || asset.Height > 8192 ||
		asset.ByteSize < 1 || asset.ByteSize > 20971520 || asset.PixelCount < 1 || asset.PixelCount > 40000000 ||
		asset.PixelCount != int64(asset.Width)*int64(asset.Height) || asset.Animated ||
		!validVisualSHA256(asset.SHA256) || asset.Revision == 0 || !validVisualReadiness(asset.Readiness) {
		return false
	}
	if asset.Error != nil && !validVisualSafeError(*asset.Error) {
		return false
	}
	if asset.Readiness == VisualReadinessReady {
		if asset.Error != nil {
			return false
		}
	} else if asset.Error == nil {
		return false
	}
	if asset.Usage == "video_cover" {
		return asset.AspectRatioErrorPPM != nil && *asset.AspectRatioErrorPPM >= 0 &&
			*asset.AspectRatioErrorPPM <= 1000 && asset.Opaque != nil && *asset.Opaque
	}
	return asset.AspectRatioErrorPPM == nil && asset.Opaque == nil
}

func validVisualSafeError(safeError VisualSafeError) bool {
	if !isVisualSafeErrorCode(safeError.Code) {
		return false
	}
	return safeError.RequestID == "" || validVisualIdentifier(safeError.RequestID)
}

func isVisualSafeErrorCode(code VisualSafeErrorCode) bool {
	switch code {
	case VisualErrorInvalidThemeID,
		VisualErrorMediaAssetFormatUnsupported,
		VisualErrorMediaAssetTooLarge,
		VisualErrorMediaAssetDecodeFailed,
		VisualErrorMediaAssetAspectRatioInvalid,
		VisualErrorMediaAssetVariantProcessing,
		VisualErrorMediaAssetVariantFailed,
		VisualErrorMediaAssetUnauthorized,
		VisualErrorMediaAssetNotFound,
		VisualErrorMediaAssetHashMismatch,
		VisualErrorMediaAssetDimensionMismatch,
		VisualErrorMediaAssetTimeout,
		VisualErrorDiscordTargetInvalid,
		VisualErrorPresetNotFound,
		VisualErrorPresetRevisionConflict,
		VisualErrorStaleJobGeneration,
		VisualErrorStaleCoverGeneration,
		VisualErrorStaleCoverRevision,
		VisualErrorIdempotencyConflict,
		VisualErrorCoverApplyAmbiguous,
		VisualErrorCoverGraphUnavailable,
		VisualErrorRevisionPayloadConflict,
		VisualErrorCapabilityRequired:
		return true
	default:
		return false
	}
}

func validVisualReadiness(readiness VisualReadiness) bool {
	return readiness == VisualReadinessReady || readiness == VisualReadinessNotReady || readiness == VisualReadinessUnknown
}

func validVisualIdempotencyKey(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	if length < 1 || length > 128 {
		return false
	}
	first, _ := utf8.DecodeRuneInString(value)
	last, _ := utf8.DecodeLastRuneInString(value)
	if visualIdempotencyEdgeSpace(first) || visualIdempotencyEdgeSpace(last) {
		return false
	}
	for _, character := range value {
		if character <= 0x1f || character == 0x7f {
			return false
		}
	}
	return true
}

func visualIdempotencyEdgeSpace(character rune) bool {
	return unicode.IsSpace(character) || character == '\ufeff'
}

func validVisualIdentifier(value string) bool {
	if len(value) < 1 || len(value) > 128 || !visualIdentifierAlphaNumeric(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if !visualIdentifierAlphaNumeric(character) && character != '.' && character != '_' && character != ':' && character != '-' {
			return false
		}
	}
	return true
}

func visualIdentifierAlphaNumeric(character byte) bool {
	return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
}

func validVisualSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for index := range value {
		if !((value[index] >= 'a' && value[index] <= 'f') || (value[index] >= '0' && value[index] <= '9')) {
			return false
		}
	}
	return true
}

func isRejectedVideoCoverResponse(response EncoderVideoCoverApplyResponse) bool {
	return response.Outcome == EncoderVideoCoverApplyOutcomeRejected && !response.Accepted && response.Rejected &&
		!response.Applied && response.Error != nil
}

func isVideoCoverFenceError(code VisualSafeErrorCode) bool {
	switch code {
	case VisualErrorStaleJobGeneration,
		VisualErrorStaleCoverGeneration,
		VisualErrorStaleCoverRevision,
		VisualErrorIdempotencyConflict,
		VisualErrorRevisionPayloadConflict:
		return true
	default:
		return false
	}
}

func isVideoCoverGraphOrAssetError(code VisualSafeErrorCode) bool {
	switch code {
	case VisualErrorMediaAssetFormatUnsupported,
		VisualErrorMediaAssetTooLarge,
		VisualErrorMediaAssetDecodeFailed,
		VisualErrorMediaAssetAspectRatioInvalid,
		VisualErrorMediaAssetVariantProcessing,
		VisualErrorMediaAssetVariantFailed,
		VisualErrorMediaAssetUnauthorized,
		VisualErrorMediaAssetNotFound,
		VisualErrorMediaAssetHashMismatch,
		VisualErrorMediaAssetDimensionMismatch,
		VisualErrorMediaAssetTimeout,
		VisualErrorCoverGraphUnavailable,
		VisualErrorCapabilityRequired:
		return true
	default:
		return false
	}
}

func equalVisualPipelineInvariant(left, right VisualPipelineInvariant) bool {
	return equalStrings(left.Layers, right.Layers) && left.WatermarkTopmost == right.WatermarkTopmost &&
		left.CoverWatermarkIndependent == right.CoverWatermarkIndependent &&
		equalStrings(left.OutputParity, right.OutputParity) && left.AudioContinuity == right.AudioContinuity
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
