package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateEncoderVideoCoverApplyJSONRejectsPresenceUnknownTrailingAndNullMutants(t *testing.T) {
	requestPayload := marshalVideoCoverValidationFixture(t, validEncoderVideoCoverApplyRequestGoFixture())
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", requestPayload); err != nil {
		t.Fatalf("valid raw request rejected: %v", err)
	}
	requestObject := decodeVideoCoverValidationObject(t, requestPayload)
	requestObject["hide_confirmed"] = false
	explicitForbiddenHide := marshalVideoCoverValidationFixture(t, requestObject)
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", explicitForbiddenHide); err == nil {
		t.Fatal("active request with explicitly present hide_confirmed:false accepted")
	}
	requestObject = decodeVideoCoverValidationObject(t, requestPayload)
	requestObject["unknown"] = true
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", marshalVideoCoverValidationFixture(t, requestObject)); err == nil {
		t.Fatal("request with unknown field accepted")
	}
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", append(requestPayload, []byte(" {}")...)); err == nil {
		t.Fatal("request with trailing JSON value accepted")
	}
	duplicateStream := append([]byte(`{"stream_id":"stream-duplicate",`), requestPayload[1:]...)
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", duplicateStream); err == nil {
		t.Fatal("request with duplicate field accepted")
	}
	if err := ValidateEncoderVideoCoverApplyRequest("stream-1", []byte("null")); err == nil {
		t.Fatal("null request accepted")
	}

	responsePayload := marshalVideoCoverValidationFixture(t, validVideoCoverResponseForStatus(409))
	if err := ValidateEncoderVideoCoverApplyResponse(409, responsePayload); err != nil {
		t.Fatalf("valid raw 409 response rejected: %v", err)
	}
	for _, required := range []string{"accepted", "applied"} {
		mutant := decodeVideoCoverValidationObject(t, responsePayload)
		delete(mutant, required)
		if err := ValidateEncoderVideoCoverApplyResponse(409, marshalVideoCoverValidationFixture(t, mutant)); err == nil {
			t.Fatalf("409 response missing required %q accepted", required)
		}
	}
	for _, requiredZero := range []string{
		"process_restart",
		"audio_encoder_restart",
		"audio_mux_restart",
		"graph_rebuild",
		"reconnect",
		"sequence_loss",
		"timestamp_discontinuity",
		"intentional_mute_insertion",
	} {
		mutant := decodeVideoCoverValidationObject(t, responsePayload)
		audioContinuity := mutant["actual"].(map[string]any)["pipeline"].(map[string]any)["audio_continuity"].(map[string]any)
		delete(audioContinuity, requiredZero)
		if err := ValidateEncoderVideoCoverApplyResponse(409, marshalVideoCoverValidationFixture(t, mutant)); err == nil {
			t.Fatalf("409 response missing required zero-valued audio continuity field %q accepted", requiredZero)
		}
	}
	responseObject := decodeVideoCoverValidationObject(t, responsePayload)
	responseObject["unknown"] = true
	if err := ValidateEncoderVideoCoverApplyResponse(409, marshalVideoCoverValidationFixture(t, responseObject)); err == nil {
		t.Fatal("response with unknown field accepted")
	}
	if err := ValidateEncoderVideoCoverApplyResponse(409, append(responsePayload, []byte(" {}")...)); err == nil {
		t.Fatal("response with trailing JSON value accepted")
	}
	duplicateAccepted := append([]byte(`{"accepted":false,`), responsePayload[1:]...)
	if err := ValidateEncoderVideoCoverApplyResponse(409, duplicateAccepted); err == nil {
		t.Fatal("response with duplicate field accepted")
	}
	if err := ValidateEncoderVideoCoverApplyResponse(409, []byte("null")); err == nil {
		t.Fatal("null response accepted")
	}
	responseObject = decodeVideoCoverValidationObject(t, responsePayload)
	responseObject["actual"].(map[string]any)["pipeline"].(map[string]any)["audio_continuity"] = nil
	if err := ValidateEncoderVideoCoverApplyResponse(409, marshalVideoCoverValidationFixture(t, responseObject)); err == nil {
		t.Fatal("response with null required object accepted")
	}
}

func TestValidateEncoderVideoCoverRuntimeStateRejectsStrictPresenceAndLinkageMutants(t *testing.T) {
	runtimePayload := marshalVideoCoverValidationFixture(t, validAppliedVideoCoverResponseGoFixture().Actual)
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", runtimePayload); err != nil {
		t.Fatalf("valid raw runtime state rejected: %v", err)
	}
	if err := ValidateEncoderVideoCoverRuntimeState("stream-2", runtimePayload); err == nil {
		t.Fatal("runtime state with path stream A and body stream B accepted")
	}

	semanticMutants := []struct {
		name   string
		mutate func(*VideoCoverRuntimeState)
	}{
		{
			name: "applied_witness_revision_mismatch",
			mutate: func(value *VideoCoverRuntimeState) {
				value.Applied.Revision++
			},
		},
		{
			name: "cover_linkage_mismatch",
			mutate: func(value *VideoCoverRuntimeState) {
				value.Cover.Revision++
			},
		},
		{
			name: "watermark_linkage_mismatch",
			mutate: func(value *VideoCoverRuntimeState) {
				value.Watermark.Revision++
			},
		},
		{
			name: "pipeline_linkage_mismatch",
			mutate: func(value *VideoCoverRuntimeState) {
				value.AppliedWitness.Pipeline.WatermarkTopmost = false
			},
		},
		{
			name: "audio_linkage_mismatch",
			mutate: func(value *VideoCoverRuntimeState) {
				value.AppliedWitness.Pipeline.AudioContinuity.Reconnect = 1
			},
		},
	}
	for _, mutant := range semanticMutants {
		t.Run(mutant.name, func(t *testing.T) {
			value := validAppliedVideoCoverResponseGoFixture().Actual
			mutant.mutate(&value)
			if err := ValidateEncoderVideoCoverRuntimeState("stream-1", marshalVideoCoverValidationFixture(t, value)); err == nil {
				t.Fatal("runtime semantic linkage mutant accepted")
			}
		})
	}

	runtimeObject := decodeVideoCoverValidationObject(t, runtimePayload)
	runtimeObject["unknown"] = true
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", marshalVideoCoverValidationFixture(t, runtimeObject)); err == nil {
		t.Fatal("runtime state with unknown field accepted")
	}
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", append(runtimePayload, []byte(" {}")...)); err == nil {
		t.Fatal("runtime state with trailing JSON value accepted")
	}
	duplicateStream := append([]byte(`{"stream_id":"stream-duplicate",`), runtimePayload[1:]...)
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", duplicateStream); err == nil {
		t.Fatal("runtime state with duplicate field accepted")
	}
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", []byte("null")); err == nil {
		t.Fatal("null runtime state accepted")
	}

	missingRequiredFalse := decodeVideoCoverValidationObject(t, runtimePayload)
	delete(missingRequiredFalse["cover_asset"].(map[string]any), "animated")
	if err := ValidateEncoderVideoCoverRuntimeState("stream-1", marshalVideoCoverValidationFixture(t, missingRequiredFalse)); err == nil {
		t.Fatal("runtime state missing required animated:false accepted")
	}
	for _, requiredZero := range []string{
		"process_restart",
		"audio_encoder_restart",
		"audio_mux_restart",
		"graph_rebuild",
		"reconnect",
		"sequence_loss",
		"timestamp_discontinuity",
		"intentional_mute_insertion",
	} {
		mutant := decodeVideoCoverValidationObject(t, runtimePayload)
		audioContinuity := mutant["pipeline"].(map[string]any)["audio_continuity"].(map[string]any)
		delete(audioContinuity, requiredZero)
		if err := ValidateEncoderVideoCoverRuntimeState("stream-1", marshalVideoCoverValidationFixture(t, mutant)); err == nil {
			t.Fatalf("runtime state missing required zero-valued audio continuity field %q accepted", requiredZero)
		}
	}
}

func TestValidateEncoderVideoCoverApplyRequestIsStandaloneFailClosed(t *testing.T) {
	request := validEncoderVideoCoverApplyRequestGoFixture()
	assertEncoderVideoCoverApplyRequestMatchesSchema(t, request)
	if err := validateEncoderVideoCoverApplyRequestFixture("stream-1", request); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	mutations := map[string]func(*EncoderVideoCoverApplyRequest){
		"empty_path": func(*EncoderVideoCoverApplyRequest) {},
		"different_path_stream": func(value *EncoderVideoCoverApplyRequest) {
			value.StreamID = "stream-2"
		},
		"empty_stream":             func(value *EncoderVideoCoverApplyRequest) { value.StreamID = "" },
		"zero_job_generation":      func(value *EncoderVideoCoverApplyRequest) { value.JobGeneration = 0 },
		"zero_expected_generation": func(value *EncoderVideoCoverApplyRequest) { value.ExpectedGeneration = 0 },
		"zero_revision":            func(value *EncoderVideoCoverApplyRequest) { value.Revision = 0 },
		"empty_idempotency":        func(value *EncoderVideoCoverApplyRequest) { value.IdempotencyKey = "" },
		"long_idempotency": func(value *EncoderVideoCoverApplyRequest) {
			value.IdempotencyKey = strings.Repeat("a", 129)
		},
		"control_idempotency": func(value *EncoderVideoCoverApplyRequest) {
			value.IdempotencyKey = "cover\napply"
		},
		"active_without_asset":          func(value *EncoderVideoCoverApplyRequest) { value.CoverAsset = nil },
		"active_with_hide_confirmation": func(value *EncoderVideoCoverApplyRequest) { value.HideConfirmed = true },
		"asset_wrong_usage": func(value *EncoderVideoCoverApplyRequest) {
			value.CoverAsset.Usage = "scene_background"
		},
		"asset_not_ready": func(value *EncoderVideoCoverApplyRequest) {
			value.CoverAsset.Readiness = VisualReadinessUnknown
			value.CoverAsset.Error = &VisualSafeError{Code: VisualErrorMediaAssetVariantProcessing}
		},
		"asset_zero_width": func(value *EncoderVideoCoverApplyRequest) { value.CoverAsset.Width = 0 },
		"asset_bad_sha":    func(value *EncoderVideoCoverApplyRequest) { value.CoverAsset.SHA256 = "bad" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			value := validEncoderVideoCoverApplyRequestGoFixture()
			mutate(&value)
			pathStreamID := "stream-1"
			if name == "empty_path" {
				pathStreamID = ""
			}
			if err := validateEncoderVideoCoverApplyRequestFixture(pathStreamID, value); err == nil {
				t.Fatal("structurally invalid request accepted")
			}
		})
	}

	hide := validEncoderVideoCoverApplyRequestGoFixture()
	hide.Active = false
	hide.Revision = 5
	hide.IdempotencyKey = "hide-cover-5"
	hide.CoverAsset = nil
	hide.HideConfirmed = true
	assertEncoderVideoCoverApplyRequestMatchesSchema(t, hide)
	if err := validateEncoderVideoCoverApplyRequestFixture("stream-1", hide); err != nil {
		t.Fatalf("valid hide request rejected: %v", err)
	}
	hide.HideConfirmed = false
	if err := validateEncoderVideoCoverApplyRequestFixture("stream-1", hide); err == nil {
		t.Fatal("hide without confirmation accepted")
	}
	hide.HideConfirmed = true
	hide.CoverAsset = validCoverAssetGoFixture()
	if err := validateEncoderVideoCoverApplyRequestFixture("stream-1", hide); err == nil {
		t.Fatal("inactive request with asset accepted")
	}
}

func TestValidateEncoderVideoCoverApplyResponseFixturesAreSchemaValidAndStatusBound(t *testing.T) {
	for _, status := range []int{200, 202, 409, 502} {
		t.Run(statusName(status), func(t *testing.T) {
			fixture := validVideoCoverResponseForStatus(status)
			assertEncoderVideoCoverApplyResponseMatchesStatusSchema(t, status, fixture)
			if err := validateEncoderVideoCoverApplyResponseFixture(status, fixture); err != nil {
				t.Fatalf("status %d valid response rejected: %v", status, err)
			}
		})
	}
}

func TestValidateEncoderVideoCoverApplyResponseRejectsStructuralMutants(t *testing.T) {
	mutations := map[string]func(*EncoderVideoCoverApplyResponse){
		"empty_outer_stream":         func(value *EncoderVideoCoverApplyResponse) { value.StreamID = "" },
		"zero_outer_job_generation":  func(value *EncoderVideoCoverApplyResponse) { value.JobGeneration = 0 },
		"zero_requested_revision":    func(value *EncoderVideoCoverApplyResponse) { value.RequestedRevision = 0 },
		"zero_actual_generation":     func(value *EncoderVideoCoverApplyResponse) { value.ActualGeneration = 0 },
		"empty_actual_stream":        func(value *EncoderVideoCoverApplyResponse) { value.Actual.StreamID = "" },
		"zero_actual_job_generation": func(value *EncoderVideoCoverApplyResponse) { value.Actual.JobGeneration = 0 },
		"zero_runtime_generation":    func(value *EncoderVideoCoverApplyResponse) { value.Actual.Generation = 0 },
		"missing_capability":         func(value *EncoderVideoCoverApplyResponse) { value.Actual.Capability = "" },
		"missing_readiness":          func(value *EncoderVideoCoverApplyResponse) { value.Actual.Readiness = "" },
		"zero_desired_revision":      func(value *EncoderVideoCoverApplyResponse) { value.Actual.Desired.Revision = 0 },
		"missing_desired_source":     func(value *EncoderVideoCoverApplyResponse) { value.Actual.Desired.Source = "" },
		"missing_applied_state":      func(value *EncoderVideoCoverApplyResponse) { value.Actual.Applied.State = "" },
		"zero_cover_revision":        func(value *EncoderVideoCoverApplyResponse) { value.Actual.Cover.Revision = 0 },
		"zero_watermark_revision":    func(value *EncoderVideoCoverApplyResponse) { value.Actual.Watermark.Revision = 0 },
		"missing_pipeline_layers":    func(value *EncoderVideoCoverApplyResponse) { value.Actual.Pipeline.Layers = nil },
		"false_watermark_topmost": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.Pipeline.WatermarkTopmost = false
		},
		"missing_output_parity": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Pipeline.OutputParity = nil },
		"false_no_automatic_resend": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.NoAutomaticResend = false
		},
	}
	for _, status := range []int{200, 202, 409, 502} {
		for name, mutate := range mutations {
			t.Run(statusName(status)+"_"+name, func(t *testing.T) {
				value := validVideoCoverResponseForStatus(status)
				mutate(&value)
				if err := validateEncoderVideoCoverApplyResponseFixture(status, value); err == nil {
					t.Fatal("structurally invalid response accepted")
				}
			})
		}
	}

	statusMutations := map[int]map[string]func(*EncoderVideoCoverApplyResponse){
		200: {
			"missing_witness": func(value *EncoderVideoCoverApplyResponse) { value.Actual.AppliedWitness = nil },
			"invalid_asset":   func(value *EncoderVideoCoverApplyResponse) { value.Actual.CoverAsset.Width = 0 },
		},
		202: {
			"missing_actual_error": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Error = nil },
			"missing_last_good":    func(value *EncoderVideoCoverApplyResponse) { value.Actual.LastGoodApplied = nil },
		},
		409: {
			"missing_ready_witness": func(value *EncoderVideoCoverApplyResponse) { value.Actual.AppliedWitness = nil },
			"invalid_outer_error": func(value *EncoderVideoCoverApplyResponse) {
				value.Error.Code = VisualSafeErrorCode("unknown")
			},
		},
		502: {
			"missing_actual_error": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Error = nil },
			"missing_last_good":    func(value *EncoderVideoCoverApplyResponse) { value.Actual.LastGoodApplied = nil },
			"invalid_asset":        func(value *EncoderVideoCoverApplyResponse) { value.Actual.CoverAsset.Width = 0 },
		},
	}
	for status, cases := range statusMutations {
		for name, mutate := range cases {
			t.Run(statusName(status)+"_"+name, func(t *testing.T) {
				value := validVideoCoverResponseForStatus(status)
				mutate(&value)
				if err := validateEncoderVideoCoverApplyResponseFixture(status, value); err == nil {
					t.Fatal("status-specific structural mutant accepted")
				}
			})
		}
	}
}

func TestValidateEncoderVideoCoverApplyResponseBindsAppliedWitness(t *testing.T) {
	if err := validateEncoderVideoCoverApplyResponseFixture(200, validAppliedVideoCoverResponseGoFixture()); err != nil {
		t.Fatalf("valid applied response rejected: %v", err)
	}
	inactive := validAppliedVideoCoverResponseGoFixture()
	inactive.RequestedRevision = 5
	inactive.Actual.Desired = VideoCoverDesiredState{Active: false, Revision: 5, Source: "none"}
	inactiveApplied := false
	inactive.Actual.Applied = VideoCoverAppliedState{State: "known", Active: &inactiveApplied, Revision: 5}
	inactive.Actual.Cover = VideoVisualLayerState{Enabled: false, Revision: 5}
	inactive.Actual.CoverAsset = nil
	inactive.Actual.AppliedWitness.Revision = 5
	inactive.Actual.AppliedWitness.Active = false
	inactive.Actual.AppliedWitness.Cover = VideoVisualLayerState{Enabled: false, Revision: 5}
	if err := validateEncoderVideoCoverApplyResponseFixture(200, inactive); err != nil {
		t.Fatalf("valid inactive applied response rejected: %v", err)
	}

	mutations := map[string]func(*EncoderVideoCoverApplyResponse){
		"outer_actual_stream": func(value *EncoderVideoCoverApplyResponse) { value.Actual.StreamID = "stream-2" },
		"outer_actual_job_generation": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.JobGeneration++
		},
		"outer_actual_generation": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Generation++ },
		"outer_witness_generation": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.AppliedWitness.Generation++
		},
		"desired_revision": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Desired.Revision++ },
		"applied_revision": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Applied.Revision++ },
		"cover_revision":   func(value *EncoderVideoCoverApplyResponse) { value.Actual.Cover.Revision++ },
		"witness_revision": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.AppliedWitness.Revision++
		},
		"desired_active": func(value *EncoderVideoCoverApplyResponse) { value.Actual.Desired.Active = false },
		"applied_active": func(value *EncoderVideoCoverApplyResponse) {
			inactive := false
			value.Actual.Applied.Active = &inactive
		},
		"witness_cover_layer": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.AppliedWitness.Cover.VariantID = "variant-2"
		},
		"witness_watermark_layer": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.AppliedWitness.Watermark.Revision++
		},
		"witness_pipeline": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.AppliedWitness.Pipeline.Layers[0] = "different"
		},
		"cover_asset_variant": func(value *EncoderVideoCoverApplyResponse) {
			value.Actual.CoverAsset.VariantID = "variant-2"
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			value := validAppliedVideoCoverResponseGoFixture()
			mutate(&value)
			if err := validateEncoderVideoCoverApplyResponseFixture(200, value); err == nil {
				t.Fatal("contradictory applied response accepted")
			}
		})
	}
}

func TestValidateEncoderVideoCoverApplyResponseEnforcesHTTPStatusClass(t *testing.T) {
	fixtures := map[int]EncoderVideoCoverApplyResponse{
		200: validVideoCoverResponseForStatus(200),
		202: validVideoCoverResponseForStatus(202),
		409: validVideoCoverResponseForStatus(409),
		502: validVideoCoverResponseForStatus(502),
	}
	for status, fixture := range fixtures {
		for otherStatus := range fixtures {
			if otherStatus == status {
				continue
			}
			if err := validateEncoderVideoCoverApplyResponseFixture(otherStatus, fixture); err == nil {
				t.Errorf("status %d response accepted for status %d", status, otherStatus)
			}
		}
	}
	if err := validateEncoderVideoCoverApplyResponseFixture(418, fixtures[200]); err == nil {
		t.Fatal("unknown response status accepted")
	}
}

func validEncoderVideoCoverApplyRequestGoFixture() EncoderVideoCoverApplyRequest {
	return EncoderVideoCoverApplyRequest{
		StreamID: "stream-1", JobGeneration: 9, ExpectedGeneration: 3, Revision: 4,
		Active: true, IdempotencyKey: "show-cover-4", CoverAsset: validCoverAssetGoFixture(),
	}
}

func validVideoCoverResponseForStatus(status int) EncoderVideoCoverApplyResponse {
	switch status {
	case 200:
		return validAppliedVideoCoverResponseGoFixture()
	case 202:
		return validAmbiguousVideoCoverResponseGoFixture()
	case 409:
		return validRejectedVideoCoverResponseGoFixture(VisualErrorStaleCoverRevision, false)
	case 502:
		return validRejectedVideoCoverResponseGoFixture(VisualErrorCoverGraphUnavailable, true)
	default:
		panic("unsupported test status")
	}
}

func validAppliedVideoCoverResponseGoFixture() EncoderVideoCoverApplyResponse {
	active := true
	cover := VideoVisualLayerState{Enabled: true, Revision: 4, VariantID: "variant-1"}
	watermark := VideoVisualLayerState{Enabled: true, Revision: 2, VariantID: "watermark-1"}
	pipeline := validVisualPipelineGoFixture()
	witnessPipeline := validVisualPipelineGoFixture()
	return EncoderVideoCoverApplyResponse{
		StreamID: "stream-1", JobGeneration: 9, RequestedRevision: 4, ActualGeneration: 3,
		Accepted: true, Applied: true, Outcome: EncoderVideoCoverApplyOutcomeApplied,
		Actual: VideoCoverRuntimeState{
			StreamID: "stream-1", JobGeneration: 9, Generation: 3,
			Capability: CapabilityLiveVideoCoverV1, Readiness: VisualReadinessReady,
			Desired: VideoCoverDesiredState{Active: true, Revision: 4, Source: "upload", VariantID: "variant-1"},
			Applied: VideoCoverAppliedState{State: "known", Active: &active, Revision: 4, VariantID: "variant-1"},
			Cover:   cover, CoverAsset: validCoverAssetGoFixture(), Watermark: watermark,
			Pipeline: pipeline, NoAutomaticResend: true,
			AppliedWitness: &VideoCoverAppliedWitness{
				GraphApplied: true, Generation: 3, Revision: 4, Active: true,
				Cover: cover, Watermark: watermark, Pipeline: witnessPipeline,
			},
		},
	}
}

func validAmbiguousVideoCoverResponseGoFixture() EncoderVideoCoverApplyResponse {
	active := true
	return EncoderVideoCoverApplyResponse{
		StreamID: "stream-1", JobGeneration: 9, RequestedRevision: 5, ActualGeneration: 4,
		Accepted: true, Outcome: EncoderVideoCoverApplyOutcomeAmbiguous,
		Actual: VideoCoverRuntimeState{
			StreamID: "stream-1", JobGeneration: 9, Generation: 4,
			Capability: CapabilityLiveVideoCoverV1, Readiness: VisualReadinessUnknown,
			Desired:   VideoCoverDesiredState{Active: false, Revision: 5, Source: "none"},
			Applied:   VideoCoverAppliedState{State: "unknown"},
			Cover:     VideoVisualLayerState{Enabled: false, Revision: 5},
			Watermark: VideoVisualLayerState{Enabled: true, Revision: 2, VariantID: "watermark-1"},
			Pipeline:  validVisualPipelineGoFixture(), NoAutomaticResend: true,
			LastGoodApplied: &VideoCoverAppliedState{State: "known", Active: &active, Revision: 4, VariantID: "variant-1"},
			Error:           &VisualSafeError{Code: VisualErrorCoverApplyAmbiguous},
		},
		Error: &VisualSafeError{Code: VisualErrorCoverApplyAmbiguous},
	}
}

func validRejectedVideoCoverResponseGoFixture(code VisualSafeErrorCode, graphFailure bool) EncoderVideoCoverApplyResponse {
	if !graphFailure {
		response := validAppliedVideoCoverResponseGoFixture()
		response.RequestedRevision = 5
		response.Accepted = false
		response.Rejected = true
		response.Applied = false
		response.Outcome = EncoderVideoCoverApplyOutcomeRejected
		response.Error = &VisualSafeError{Code: code}
		return response
	}
	active := true
	return EncoderVideoCoverApplyResponse{
		StreamID: "stream-1", JobGeneration: 9, RequestedRevision: 5, ActualGeneration: 4,
		Rejected: true, Outcome: EncoderVideoCoverApplyOutcomeRejected,
		Actual: VideoCoverRuntimeState{
			StreamID: "stream-1", JobGeneration: 9, Generation: 4,
			Capability: CapabilityLiveVideoCoverV1, Readiness: VisualReadinessNotReady,
			Desired:    VideoCoverDesiredState{Active: true, Revision: 5, Source: "upload", VariantID: "variant-1"},
			Applied:    VideoCoverAppliedState{State: "unknown"},
			Cover:      VideoVisualLayerState{Enabled: true, Revision: 4, VariantID: "variant-1"},
			CoverAsset: validCoverAssetGoFixture(),
			Watermark:  VideoVisualLayerState{Enabled: true, Revision: 2, VariantID: "watermark-1"},
			Pipeline:   validVisualPipelineGoFixture(), NoAutomaticResend: true,
			LastGoodApplied: &VideoCoverAppliedState{State: "known", Active: &active, Revision: 4, VariantID: "variant-1"},
			Error:           &VisualSafeError{Code: code},
		},
		Error: &VisualSafeError{Code: code},
	}
}

func validVisualPipelineGoFixture() VisualPipelineInvariant {
	return VisualPipelineInvariant{
		Layers:           []string{"base_or_worker_scene", "video_cover", "watermark", "video_encode", "tee_live_archive_preview"},
		WatermarkTopmost: true, CoverWatermarkIndependent: true,
		OutputParity: []string{"live", "archive", "preview"},
	}
}

func assertEncoderVideoCoverApplyRequestMatchesSchema(t *testing.T, request EncoderVideoCoverApplyRequest) {
	t.Helper()
	schema := compileContractJSONSchema(t, "encoder-video-cover-apply-request.schema.json", visualCatalogSchema)
	assertGoValueMatchesSchema(t, schema, request)
}

func assertEncoderVideoCoverApplyResponseMatchesStatusSchema(t *testing.T, status int, response EncoderVideoCoverApplyResponse) {
	t.Helper()
	fragment := "/paths/~1streams~1{id}~1video-cover-state/put/responses/" + statusName(status) + "/content/application~1json/schema"
	schema := compileNormalizedOpenAPISchema(t, "encoder-recorder-api.json", fragment)
	assertGoValueMatchesSchema(t, schema, response)
}

func assertGoValueMatchesSchema(t *testing.T, schema interface{ Validate(any) error }, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONPayloadAgainstSchema(t, schema, payload, true)
}

func marshalVideoCoverValidationFixture(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func decodeVideoCoverValidationObject(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func validateEncoderVideoCoverApplyRequestFixture(pathStreamID string, value EncoderVideoCoverApplyRequest) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ValidateEncoderVideoCoverApplyRequest(pathStreamID, payload)
}

func validateEncoderVideoCoverApplyResponseFixture(status int, value EncoderVideoCoverApplyResponse) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ValidateEncoderVideoCoverApplyResponse(status, payload)
}

func statusName(status int) string {
	switch status {
	case 200:
		return "200"
	case 202:
		return "202"
	case 409:
		return "409"
	case 502:
		return "502"
	default:
		return "unknown"
	}
}
