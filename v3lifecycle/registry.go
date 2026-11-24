package lifecycle

// ContractDescriptor identifies schema ownership, not fact or authority ownership.
type ContractDescriptor struct {
	ID                    ContractID
	OwnerRepository, Name string
}

func ContractRegister() []ContractDescriptor {
	return []ContractDescriptor{
		{CC01, "Kome-Lab/Autostream-Contracts", "Domain/membership"},
		{CC02, "Kome-Lab/Autostream-Contracts", "Generation/Revision"},
		{CC03, "Kome-Lab/Autostream-Contracts", "Admission/Drain"},
		{CC04, "Kome-Lab/Autostream-Contracts", "Completion Proof"},
		{CC05, "Kome-Lab/Autostream-Contracts", "READY Seal"},
		{CC06, "Kome-Lab/Autostream-Contracts", "Final Fence"},
		{CC07, "Kome-Lab/Autostream-Contracts", "External Effect intent/dispatch/receipt"},
		{CC08, "Kome-Lab/Autostream-Contracts", "UNKNOWN/Inspect"},
		{CC09, "Kome-Lab/Autostream-Contracts", "Recovery/reconcile responsibility"},
	}
}

// ConsumerExpectation is a conformance role, never an installed adapter claim.
type ConsumerExpectation struct {
	Repository          string
	Component           ComponentID
	Workstream          string
	Profile             ProfileID
	ContractRevision    uint64
	Expected            ResultKind
	ProductionQualified bool
}

func ConsumerExpectations() []ConsumerExpectation {
	return []ConsumerExpectation{
		{"Kome-Lab/Autostream-ControlPanel", "C01", "WS1", ApprovalLocal, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-ControlPanel", "C01", "WS2", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-ControlPanel", "C02", "WS7", StopLookup, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-ControlPanel", "C14", "WS7", RecordRepair, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS1", ApprovalLocal, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS2", StopLookup, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS3", MediaIngress, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS4", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS5", EffectOutcome, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS6", RecordRepair, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Contracts", "C03", "WS7", ExplicitResume, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Encoder-Recorder", "C04", "WS3", MediaIngress, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Encoder-Recorder", "C05", "WS3", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Worker", "C06", "WS3", MediaIngress, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-DiscordBot", "C07", "WS3", MediaIngress, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Observability", "C08", "WS5", EffectOutcome, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Updater", "C09", "WS4", MediaIngress, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Updater", "C10", "WS4", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Updater", "C11", "WS6", RecordRepair, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Docker", "C12", "WS6", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Docker", "C14", "WS6", ClosureAssess, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Docs", "C13", "WS7", StopLookup, 1, StructurallyCoherent, false},
		{"Kome-Lab/Autostream-Docs", "C14", "WS7", RecordRepair, 1, StructurallyCoherent, false},
	}
}

// LegacyProjection deliberately provides no automatic new-state -> Evidence3 cast.
// Old admission/drain values and the original Evidence API remain in v3evidence.
func LegacyProjection(lane AdmissionLane, drain DrainState, gate Gate, profile ProfileRef) Result {
	if !member(lane, Open, Closing, Closed) || !member(drain, Waiting, Proved) || !validGate(gate) || !validProfile(profile.ProfileID) || profile.Revision != 1 {
		return result(RefusedUnchanged, "INVALID_NEW_LIFECYCLE_STATE", "Projection")
	}
	return result(NeedsDecision, "NO_PROJECTION", "Projection")
}
