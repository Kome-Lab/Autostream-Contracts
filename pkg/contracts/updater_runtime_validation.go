package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	contractschemas "github.com/example/autostream-contracts/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	errUpdaterCommandInvalid               = errors.New("updater command envelope invalid")
	errUpdaterLeaseInvalid                 = errors.New("updater lease envelope invalid")
	errUpdaterProgressInvalid              = errors.New("updater progress envelope invalid")
	errUpdaterResultInvalid                = errors.New("updater result envelope invalid")
	errUpdaterMutationGrantInvalid         = errors.New("updater mutation grant binding invalid")
	errUpdaterMutationGrantResponseInvalid = errors.New("updater mutation grant issue response invalid")
	errUpdateAgentClearInvalid             = errors.New("update agent clear-active-job response invalid")
	errUpdaterDesiredSchema                = errors.New("updater desired-operation canonical schema unavailable")
	errUpdaterCanonicalJSONInvalid         = errors.New("updater canonical JSON projection invalid")
)

const (
	updaterDesiredSchemaName                = "updater-desired-operation.schema.json"
	updaterDesiredSchemaID                  = "https://schemas.autostream.example.com/updater-desired-operation.schema.json"
	updaterSystemUpdateSchemaName           = "system-update-job.schema.json"
	updaterSelfUpdateSchemaName             = "host-agent-self-update-directive.schema.json"
	updaterReleaseBindingSchemaName         = "host-self-update-release-binding.schema.json"
	updaterSchemaCanonicalBase              = "https://schemas.autostream.example.com/"
	updaterMaxJCSSafeInteger        float64 = 9007199254740991
)

var (
	updaterIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	updaterHostIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,190}$`)
	updaterNoncePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$`)
	updaterDigestPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	updaterRawSHA256Pattern  = regexp.MustCompile(`^[a-f0-9]{64}$`)

	updaterDesiredCanonicalSchema struct {
		once   sync.Once
		schema *jsonschema.Schema
		err    error
	}
)

// ComputeUpdaterCommandCanonicalDigest returns the RFC 8785/JCS SHA-256
// digest of exactly {target, desired_revision, fence, desired_operation}.
// Callers must use the result for both canonical_payload_digest and the
// mutation authorization's canonical_argument_digest.
func ComputeUpdaterCommandCanonicalDigest(target UpdaterTargetIdentity, desiredRevision, fence int64, desiredOperation UpdaterDesiredOperation) (string, error) {
	if desiredRevision < 1 || fence < 1 {
		return "", errUpdaterCanonicalJSONInvalid
	}
	targetDocument, err := updaterCanonicalDocument(target)
	if err != nil {
		return "", errUpdaterCanonicalJSONInvalid
	}
	desiredDocument, err := updaterCanonicalDocument(desiredOperation)
	if err != nil {
		return "", errUpdaterCanonicalJSONInvalid
	}
	projection := map[string]any{
		"target":            targetDocument,
		"desired_revision":  desiredRevision,
		"fence":             fence,
		"desired_operation": desiredDocument,
	}
	var canonical bytes.Buffer
	if err := appendUpdaterJCS(&canonical, projection); err != nil {
		return "", errUpdaterCanonicalJSONInvalid
	}
	digest := sha256.Sum256(canonical.Bytes())
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// ValidateUpdaterCommandEnvelope validates a raw command while required field
// presence, duplicate members, and unknown members are still observable.
func ValidateUpdaterCommandEnvelope(payload []byte) error {
	var command UpdaterCommandEnvelope
	document, err := decodeUpdaterStrictJSON(payload, &command)
	if err != nil || !validateUpdaterCommandDocument(document) {
		return errUpdaterCommandInvalid
	}
	return ValidateUpdaterCommand(command)
}

// ValidateUpdaterLeaseEnvelope validates a raw lease and its nested raw
// command, then applies expiry and command-binding semantics at now.
func ValidateUpdaterLeaseEnvelope(now time.Time, payload []byte) error {
	var lease UpdaterLeaseEnvelope
	document, err := decodeUpdaterStrictJSON(payload, &lease)
	if err != nil || !validateUpdaterLeaseDocument(document) {
		return errUpdaterLeaseInvalid
	}
	leaseDocument, ok := document.(map[string]any)
	if !ok || !validateUpdaterCommandDocument(leaseDocument["command"]) {
		return errUpdaterLeaseInvalid
	}
	return ValidateUpdaterLease(now, lease)
}

// ValidateUpdaterProgressEnvelope validates raw progress and binds it to the
// supplied validated lease generation.
func ValidateUpdaterProgressEnvelope(lease UpdaterLeaseEnvelope, payload []byte) error {
	var progress UpdaterProgressEnvelope
	document, err := decodeUpdaterStrictJSON(payload, &progress)
	if err != nil || !validateUpdaterProgressDocument(document) {
		return errUpdaterProgressInvalid
	}
	return ValidateUpdaterProgress(lease, progress)
}

// ValidateUpdaterResultEnvelope validates raw terminal output and binds it to
// the supplied validated lease generation and command intent.
func ValidateUpdaterResultEnvelope(lease UpdaterLeaseEnvelope, payload []byte) error {
	var result UpdaterResultEnvelope
	document, err := decodeUpdaterStrictJSON(payload, &result)
	if err != nil || !validateUpdaterResultDocument(document) {
		return errUpdaterResultInvalid
	}
	return ValidateUpdaterResult(lease, result)
}

// ValidateUpdaterMutationGrantIssueRequest validates a credential-free raw
// grant request. The opaque grant itself is response-only and is not accepted
// in this JSON body.
func ValidateUpdaterMutationGrantIssueRequest(now time.Time, payload []byte) error {
	var request UpdaterMutationGrantIssueRequest
	document, err := decodeUpdaterStrictJSON(payload, &request)
	if err != nil || !validateUpdaterMutationGrantDocument(document) {
		return errUpdaterMutationGrantInvalid
	}
	return ValidateUpdaterMutationGrantBinding(now, request.Binding)
}

// ValidateUpdaterMutationGrantConsumeRequest validates the Local Executor's
// credential-free consume binding. The opaque grant belongs in the
// authorization channel and is never accepted in this body.
func ValidateUpdaterMutationGrantConsumeRequest(now time.Time, payload []byte) error {
	var request UpdaterMutationGrantConsumeRequest
	document, err := decodeUpdaterStrictJSON(payload, &request)
	if err != nil || !validateUpdaterMutationGrantDocument(document) {
		return errUpdaterMutationGrantInvalid
	}
	return ValidateUpdaterMutationGrantBinding(now, request.Binding)
}

// ValidateUpdaterMutationGrantIssueResponse validates the exact secret-bearing
// response object without ever including the token value in an error. The
// caller must keep the response no-store and out of logs and persistent state.
func ValidateUpdaterMutationGrantIssueResponse(now time.Time, payload []byte) error {
	var response UpdaterMutationGrantIssueResponse
	document, err := decodeUpdaterStrictJSON(payload, &response)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !requireUpdaterFields(object, "grant_token", "expires_at") ||
		!validUpdaterMutationGrantIssueResponse(now, response) {
		return errUpdaterMutationGrantResponseInvalid
	}
	return nil
}

// ValidateUpdateAgentClearActiveJobResponse validates the non-lease v2 claim
// variant before a boolean zero value can erase the required true sentinel.
func ValidateUpdateAgentClearActiveJobResponse(payload []byte) error {
	var response UpdateAgentClearActiveJobResponse
	document, err := decodeUpdaterStrictJSON(payload, &response)
	object, ok := document.(map[string]any)
	if err != nil || !ok || !requireUpdaterFields(object, "clear_active_job_id") || !response.ClearActiveJobID {
		return errUpdateAgentClearInvalid
	}
	return nil
}

// ValidateUpdaterCommand applies typed semantic validation. Network entry
// points should call ValidateUpdaterCommandEnvelope first so required zero and
// false presence cannot be lost by decoding.
func ValidateUpdaterCommand(command UpdaterCommandEnvelope) error {
	if command.ProtocolVersion != 2 || !validUpdaterIdentifier(command.CommandID) ||
		!validUpdaterIdempotencyKey(command.IdempotencyKey) ||
		!validUpdaterIdentifier(command.AuditCorrelationID) ||
		command.Issuer.ServiceType != "control_panel" ||
		command.Issuer.Authentication != "assignment_bound_rotating_service_identity" ||
		command.Issuer.Permission != "updates.authorize" ||
		!validUpdaterIdentifier(command.Issuer.ServiceID) {
		return errUpdaterCommandInvalid
	}
	authorization := command.MutationAuthorization
	if !validUpdaterIdentifier(authorization.AuthorizationID) ||
		!updaterNoncePattern.MatchString(authorization.NonceID) ||
		!validUpdaterIdentifier(authorization.JobID) ||
		!validUpdaterIdentifier(authorization.UpdaterID) ||
		!updaterHostIDPattern.MatchString(authorization.HostID) ||
		authorization.DesiredRevision < 1 || authorization.Fence < 1 ||
		authorization.ExpiresAt.IsZero() || !authorization.OneTime {
		return errUpdaterCommandInvalid
	}
	expectedCapability, ok := updaterCapabilityForOperation(command.DesiredOperation.Operation)
	if !ok || authorization.ActionType != expectedCapability || authorization.RequiredCapability != expectedCapability {
		return errUpdaterCommandInvalid
	}
	if err := validateUpdaterDesiredOperation(command.DesiredOperation); err != nil {
		return errUpdaterCommandInvalid
	}
	if !validUpdaterTargetForOperation(authorization.Target, command.DesiredOperation.Operation, authorization.HostID) {
		return errUpdaterCommandInvalid
	}
	if command.DesiredOperation.Operation == UpdaterDesiredPortReconfigure &&
		!validUpdaterPortTargetBinding(authorization.Target, command.DesiredOperation.PortReconfigure) {
		return errUpdaterCommandInvalid
	}
	digest, err := ComputeUpdaterCommandCanonicalDigest(
		authorization.Target,
		authorization.DesiredRevision,
		authorization.Fence,
		command.DesiredOperation,
	)
	if err != nil || !updaterDigestPattern.MatchString(command.CanonicalPayloadDigest) ||
		command.CanonicalPayloadDigest != authorization.CanonicalArgumentDigest ||
		command.CanonicalPayloadDigest != digest {
		return errUpdaterCommandInvalid
	}
	return nil
}

// ValidateUpdaterLease applies typed lease expiry and nested command
// semantics. Raw entry points should use ValidateUpdaterLeaseEnvelope.
func ValidateUpdaterLease(now time.Time, lease UpdaterLeaseEnvelope) error {
	if err := validateUpdaterLeaseStatic(lease); err != nil || now.IsZero() ||
		!now.Before(lease.LeaseExpiresAt) {
		return errUpdaterLeaseInvalid
	}
	return nil
}

// ValidateUpdaterProgress applies typed lease-generation and command-fence
// linkage. Raw entry points should use ValidateUpdaterProgressEnvelope.
func ValidateUpdaterProgress(lease UpdaterLeaseEnvelope, progress UpdaterProgressEnvelope) error {
	if err := validateUpdaterLeaseStatic(lease); err != nil || progress.ProtocolVersion != 2 ||
		progress.CommandID != lease.Command.CommandID ||
		progress.JobID != lease.Command.MutationAuthorization.JobID ||
		progress.UpdaterID != lease.Command.MutationAuthorization.UpdaterID ||
		progress.HostID != lease.Command.MutationAuthorization.HostID ||
		progress.LeaseID != lease.LeaseID || progress.LeaseGeneration != lease.LeaseGeneration ||
		progress.DesiredRevision != lease.Command.MutationAuthorization.DesiredRevision ||
		progress.Fence != lease.Command.MutationAuthorization.Fence ||
		progress.AuditCorrelationID != lease.Command.AuditCorrelationID ||
		progress.Sequence < 0 || progress.Progress < 0 || progress.Progress > 100 ||
		!validUpdaterProgressPhase(progress.Phase) || progress.ObservedAt.IsZero() ||
		progress.ObservedAt.After(lease.LeaseExpiresAt) {
		return errUpdaterProgressInvalid
	}
	return nil
}

// ValidateUpdaterResult applies typed lease-generation, command, terminal
// outcome, revision and bounded-evidence semantics. Raw entry points should use
// ValidateUpdaterResultEnvelope.
func ValidateUpdaterResult(lease UpdaterLeaseEnvelope, result UpdaterResultEnvelope) error {
	if err := validateUpdaterLeaseStatic(lease); err != nil || result.ProtocolVersion != 2 ||
		result.CommandID != lease.Command.CommandID ||
		result.JobID != lease.Command.MutationAuthorization.JobID ||
		result.UpdaterID != lease.Command.MutationAuthorization.UpdaterID ||
		result.HostID != lease.Command.MutationAuthorization.HostID ||
		result.LeaseID != lease.LeaseID || result.LeaseGeneration != lease.LeaseGeneration ||
		result.IdempotencyKey != lease.Command.IdempotencyKey ||
		result.CanonicalPayloadDigest != lease.Command.CanonicalPayloadDigest ||
		result.AuthorizationID != lease.Command.MutationAuthorization.AuthorizationID ||
		result.DesiredRevision != lease.Command.MutationAuthorization.DesiredRevision ||
		result.Fence != lease.Command.MutationAuthorization.Fence ||
		result.AuditCorrelationID != lease.Command.AuditCorrelationID ||
		result.AutomaticResendAllowed || len(result.Evidence) < 1 || len(result.Evidence) > 32 ||
		!validUpdaterEvidence(result.Evidence, lease) ||
		(result.SafeError != nil && !validUpdaterSafeError(result.SafeError, result.AuditCorrelationID)) {
		return errUpdaterResultInvalid
	}

	switch result.Outcome {
	case UpdaterOutcomeSucceeded:
		if result.Status != SystemUpdateSucceeded || result.SafeError != nil ||
			result.AppliedRevision != result.DesiredRevision {
			return errUpdaterResultInvalid
		}
		if lease.Command.MutationAuthorization.Target.TargetKind == UpdaterTargetApplication &&
			!hasUpdaterEvidenceAtRevision(result.Evidence, "application_probe_verified", result.AppliedRevision) {
			return errUpdaterResultInvalid
		}
		if lease.Command.DesiredOperation.Operation == UpdaterDesiredHostSelfUpdate &&
			!hasUpdaterEvidenceAtRevision(result.Evidence, "host_runtime_verified", result.AppliedRevision) {
			return errUpdaterResultInvalid
		}
	case UpdaterOutcomeRolledBack:
		if result.Status != SystemUpdateRolledBack || !hasUpdaterEvidence(result.Evidence, "rollback_verified") {
			return errUpdaterResultInvalid
		}
	case UpdaterOutcomeFailed:
		if result.Status != SystemUpdateFailed || !validUpdaterSafeError(result.SafeError, result.AuditCorrelationID) {
			return errUpdaterResultInvalid
		}
	case UpdaterOutcomeAmbiguous:
		if result.Status != SystemUpdateReconciling || !validUpdaterSafeError(result.SafeError, result.AuditCorrelationID) ||
			result.SafeError.Code != "outcome_ambiguous" ||
			!hasUpdaterEvidence(result.Evidence, "outcome_ambiguous") {
			return errUpdaterResultInvalid
		}
	default:
		return errUpdaterResultInvalid
	}
	return nil
}

// ValidateUpdaterMutationGrantBinding applies typed lease, session and closed
// Local Executor operation linkage. Raw entry points must validate presence
// before calling this seam.
func ValidateUpdaterMutationGrantBinding(now time.Time, binding UpdaterMutationGrantBinding) error {
	if ValidateUpdaterLease(now, binding.Lease) != nil || !updaterNoncePattern.MatchString(binding.SessionID) ||
		!validUpdaterMutationForDesired(binding.Operation, binding.Lease.Command.DesiredOperation.Operation) {
		return errUpdaterMutationGrantInvalid
	}
	return nil
}

// ValidateUpdaterMutationGrantIssueResponseForLease binds a decoded issue
// response to the lease and authorization expiry that authorized issuance.
// Network entry points should call the raw response validator first.
func ValidateUpdaterMutationGrantIssueResponseForLease(now time.Time, lease UpdaterLeaseEnvelope, response UpdaterMutationGrantIssueResponse) error {
	if ValidateUpdaterLease(now, lease) != nil || !validUpdaterMutationGrantIssueResponse(now, response) ||
		response.ExpiresAt.After(lease.LeaseExpiresAt) ||
		response.ExpiresAt.After(lease.Command.MutationAuthorization.ExpiresAt) {
		return errUpdaterMutationGrantResponseInvalid
	}
	return nil
}

func validateUpdaterLeaseStatic(lease UpdaterLeaseEnvelope) error {
	if lease.ProtocolVersion != 2 || !validUpdaterIdentifier(lease.LeaseID) || lease.LeaseGeneration < 1 ||
		lease.LeaseExpiresAt.IsZero() || ValidateUpdaterCommand(lease.Command) != nil ||
		lease.LeaseExpiresAt.After(lease.Command.MutationAuthorization.ExpiresAt) {
		return errUpdaterLeaseInvalid
	}
	return nil
}

func updaterCapabilityForOperation(operation UpdaterDesiredOperationType) (UpdaterCapability, bool) {
	switch operation {
	case UpdaterDesiredSoftwareUpdate:
		return UpdaterCapabilityUpdate, true
	case UpdaterDesiredBootstrap:
		return UpdaterCapabilityBootstrap, true
	case UpdaterDesiredPortReconfigure:
		return UpdaterCapabilityPort, true
	case UpdaterDesiredHostSelfUpdate:
		return UpdaterCapabilitySelfUpdate, true
	default:
		return "", false
	}
}

func validUpdaterMutationForDesired(mutation UpdaterMutationOperation, desired UpdaterDesiredOperationType) bool {
	switch desired {
	case UpdaterDesiredSoftwareUpdate:
		return mutation == UpdaterMutationApply || mutation == UpdaterMutationReconcile
	case UpdaterDesiredPortReconfigure:
		return mutation == UpdaterMutationPortReconfigure || mutation == UpdaterMutationPortReconfigureReconcile
	case UpdaterDesiredBootstrap:
		return mutation == UpdaterMutationBootstrap || mutation == UpdaterMutationBootstrapReconcile
	case UpdaterDesiredHostSelfUpdate:
		return mutation == UpdaterMutationHostSelfUpdateStage || mutation == UpdaterMutationHostSelfUpdateActivate ||
			mutation == UpdaterMutationHostSelfUpdateReconcile
	default:
		return false
	}
}

func validateUpdaterDesiredOperation(desired UpdaterDesiredOperation) error {
	schema, err := loadUpdaterDesiredCanonicalSchema()
	if err != nil {
		return errUpdaterDesiredSchema
	}
	body, err := json.Marshal(desired)
	if err != nil {
		return errUpdaterCommandInvalid
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil || schema.Validate(document) != nil {
		return errUpdaterCommandInvalid
	}
	if desired.Operation == UpdaterDesiredSoftwareUpdate && desired.SoftwareUpdate != nil &&
		desired.SoftwareUpdate.ExpectedCurrentVersion == desired.SoftwareUpdate.TargetVersion {
		return errUpdaterCommandInvalid
	}
	if desired.Operation == UpdaterDesiredPortReconfigure && desired.PortReconfigure != nil {
		plan := desired.PortReconfigure
		if plan.Result != "" || plan.OldPort == plan.NewPort {
			return errUpdaterCommandInvalid
		}
	}
	return nil
}

func validUpdaterTargetForOperation(target UpdaterTargetIdentity, operation UpdaterDesiredOperationType, hostID string) bool {
	if !validUpdaterIdentifier(target.ServiceID) || !validUpdaterDeploymentMode(target.DeploymentMode) {
		return false
	}
	switch target.TargetKind {
	case UpdaterTargetApplication:
		return (operation == UpdaterDesiredSoftwareUpdate || operation == UpdaterDesiredPortReconfigure) &&
			validUpdaterApplicationServiceType(target.ServiceType) && target.ExpectedConfigRevision >= 1 &&
			target.ExecutionHostID == ""
	case UpdaterTargetUpdateAgent:
		return operation == UpdaterDesiredBootstrap && target.ServiceType == SystemUpdateTargetUpdateAgent &&
			target.DeploymentMode == SystemUpdateDeploymentSystemd && target.ExpectedConfigRevision == 0 &&
			updaterHostIDPattern.MatchString(target.ExecutionHostID) && target.ExecutionHostID == hostID
	case UpdaterTargetHostRuntime:
		return operation == UpdaterDesiredHostSelfUpdate && target.ServiceType == SystemUpdateTargetUpdateAgent &&
			target.DeploymentMode == SystemUpdateDeploymentSystemd && target.ExpectedConfigRevision == 0 &&
			updaterHostIDPattern.MatchString(target.ExecutionHostID) && target.ExecutionHostID == hostID
	default:
		return false
	}
}

func validUpdaterApplicationServiceType(serviceType SystemUpdateTargetType) bool {
	switch serviceType {
	case SystemUpdateTargetControlPanel, SystemUpdateTargetDiscordBot, SystemUpdateTargetEncoderRecorder,
		SystemUpdateTargetObservability, SystemUpdateTargetWorker:
		return true
	default:
		return false
	}
}

func validUpdaterDeploymentMode(mode SystemUpdateDeploymentMode) bool {
	return mode == SystemUpdateDeploymentSystemd || mode == SystemUpdateDeploymentDocker
}

func validUpdaterPortTargetBinding(target UpdaterTargetIdentity, plan *SystemUpdatePortReconfiguration) bool {
	if plan == nil || !validUpdaterPortPlanDigest(*plan) {
		return false
	}
	switch target.DeploymentMode {
	case SystemUpdateDeploymentSystemd:
		return plan.Docker == nil && plan.OldPort >= 1024 && plan.NewPort >= 1024
	case SystemUpdateDeploymentDocker:
		if plan.Docker == nil {
			return false
		}
		docker := plan.Docker
		return docker.OldHealthPort == docker.OldPublishedPort &&
			docker.NewHealthPort == docker.NewPublishedPort
	default:
		return false
	}
}

func validUpdaterIdentifier(value string) bool {
	return updaterIdentifierPattern.MatchString(value)
}

func validUpdaterIdempotencyKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r <= 0x1f || r == 0x7f {
			return false
		}
	}
	return true
}

func validUpdaterMutationGrantIssueResponse(now time.Time, response UpdaterMutationGrantIssueResponse) bool {
	if now.IsZero() || !now.Before(response.ExpiresAt) || len(response.GrantToken) < 32 || len(response.GrantToken) > 256 {
		return false
	}
	for _, r := range response.GrantToken {
		if r <= 0x1f || r == 0x7f {
			return false
		}
	}
	return true
}

func validUpdaterProgressPhase(phase string) bool {
	switch phase {
	case "accepted", "journaled", "preparing", "executing", "verifying", "rolling_back", "reconciling":
		return true
	default:
		return false
	}
}

func validUpdaterEvidence(evidence []UpdaterEvidence, lease UpdaterLeaseEnvelope) bool {
	for _, item := range evidence {
		if !validUpdaterEvidenceCode(item.EvidenceCode) || item.ObservedAt.IsZero() ||
			item.ObservedAt.After(lease.LeaseExpiresAt) || item.ObservedRevision < 1 ||
			(item.ArtifactDigest != "" && !updaterDigestPattern.MatchString(item.ArtifactDigest)) {
			return false
		}
	}
	return true
}

func validUpdaterEvidenceCode(code string) bool {
	switch code {
	case "command_accepted", "phase_observed", "application_probe_verified", "host_runtime_verified", "rollback_verified", "outcome_ambiguous":
		return true
	default:
		return false
	}
}

func hasUpdaterEvidence(evidence []UpdaterEvidence, code string) bool {
	for _, item := range evidence {
		if item.EvidenceCode == code {
			return true
		}
	}
	return false
}

func hasUpdaterEvidenceAtRevision(evidence []UpdaterEvidence, code string, revision int64) bool {
	for _, item := range evidence {
		if item.EvidenceCode == code && item.ObservedRevision == revision {
			return true
		}
	}
	return false
}

func validUpdaterSafeError(safeError *V2UpdaterSafeError, auditCorrelationID string) bool {
	if safeError == nil {
		return false
	}
	message, ok := CanonicalUpdaterSafeErrorMessage(safeError.Code)
	if !ok || safeError.Message != message {
		return false
	}
	return safeError.AuditCorrelationID == "" || safeError.AuditCorrelationID == auditCorrelationID
}

// CanonicalUpdaterSafeErrorMessage returns the only message permitted for a
// stable updater error code. Keeping this surface closed prevents credentials,
// paths, command output, and other caller-controlled text from entering the
// wire error envelope.
func CanonicalUpdaterSafeErrorMessage(code string) (string, bool) {
	switch code {
	case "protocol_version_unsupported":
		return "updater protocol version is unsupported", true
	case "issuer_authentication_failed":
		return "updater issuer authentication failed", true
	case "updater_identity_mismatch":
		return "updater identity does not match assignment", true
	case "authorization_expired":
		return "updater authorization expired", true
	case "authorization_replayed":
		return "updater authorization was already used", true
	case "idempotency_payload_conflict":
		return "updater idempotency payload conflicts", true
	case "capability_missing":
		return "updater capability is unavailable", true
	case "revision_conflict":
		return "updater revision conflicts", true
	case "stale_fence":
		return "updater fence is stale", true
	case "command_expired":
		return "updater command expired", true
	case "outcome_ambiguous":
		return "updater outcome requires reconciliation", true
	case "local_journal_unavailable":
		return "updater local journal is unavailable", true
	case "execution_failed":
		return "updater execution failed", true
	default:
		return "", false
	}
}

func validateUpdaterCommandDocument(document any) bool {
	command, ok := document.(map[string]any)
	if !ok || !requireUpdaterFields(command,
		"protocol_version", "command_id", "issuer", "idempotency_key", "canonical_payload_digest",
		"mutation_authorization", "desired_operation", "audit_correlation_id") {
		return false
	}
	issuer, ok := command["issuer"].(map[string]any)
	if !ok || !requireUpdaterFields(issuer, "service_id", "service_type", "authentication", "permission") {
		return false
	}
	authorization, ok := command["mutation_authorization"].(map[string]any)
	if !ok || !requireUpdaterFields(authorization,
		"authorization_id", "nonce_id", "job_id", "updater_id", "host_id", "action_type", "target",
		"canonical_argument_digest", "desired_revision", "fence", "expires_at", "required_capability", "one_time") {
		return false
	}
	target, ok := authorization["target"].(map[string]any)
	if !ok || !requireUpdaterFields(target, "target_kind", "service_id", "service_type", "deployment_mode") {
		return false
	}
	targetKind, _ := target["target_kind"].(string)
	switch UpdaterTargetKind(targetKind) {
	case UpdaterTargetApplication:
		if !requireUpdaterFields(target, "expected_config_revision") || hasUpdaterField(target, "execution_host_id") {
			return false
		}
	case UpdaterTargetUpdateAgent, UpdaterTargetHostRuntime:
		if !requireUpdaterFields(target, "execution_host_id") || hasUpdaterField(target, "expected_config_revision") {
			return false
		}
	default:
		return false
	}
	desiredSchema, err := loadUpdaterDesiredCanonicalSchema()
	return err == nil && desiredSchema.Validate(command["desired_operation"]) == nil
}

func validateUpdaterLeaseDocument(document any) bool {
	lease, ok := document.(map[string]any)
	return ok && requireUpdaterFields(lease, "protocol_version", "lease_id", "lease_generation", "lease_expires_at", "command")
}

func validateUpdaterProgressDocument(document any) bool {
	progress, ok := document.(map[string]any)
	return ok && requireUpdaterFields(progress,
		"protocol_version", "command_id", "job_id", "updater_id", "host_id", "lease_id", "lease_generation",
		"sequence", "phase", "progress", "desired_revision", "fence", "audit_correlation_id", "observed_at")
}

func validateUpdaterResultDocument(document any) bool {
	result, ok := document.(map[string]any)
	if !ok || !requireUpdaterFields(result,
		"protocol_version", "command_id", "job_id", "updater_id", "host_id", "lease_id", "lease_generation",
		"idempotency_key", "canonical_payload_digest", "authorization_id", "desired_revision", "fence", "outcome",
		"status", "automatic_resend_allowed", "audit_correlation_id", "evidence") {
		return false
	}
	outcome, _ := result["outcome"].(string)
	if outcome == string(UpdaterOutcomeSucceeded) && !hasUpdaterField(result, "applied_revision") {
		return false
	}
	if (outcome == string(UpdaterOutcomeFailed) || outcome == string(UpdaterOutcomeAmbiguous)) && !hasUpdaterField(result, "safe_error") {
		return false
	}
	if safeError, present := result["safe_error"]; present {
		object, ok := safeError.(map[string]any)
		if !ok || !requireUpdaterFields(object, "code", "message", "retryable") {
			return false
		}
	}
	evidence, ok := result["evidence"].([]any)
	if !ok || len(evidence) < 1 || len(evidence) > 32 {
		return false
	}
	for _, rawEvidence := range evidence {
		object, ok := rawEvidence.(map[string]any)
		if !ok || !requireUpdaterFields(object, "evidence_code", "observed_at", "observed_revision") {
			return false
		}
	}
	return true
}

func validateUpdaterMutationGrantDocument(document any) bool {
	request, ok := document.(map[string]any)
	if !ok || !requireUpdaterFields(request, "binding") {
		return false
	}
	binding, ok := request["binding"].(map[string]any)
	if !ok || !requireUpdaterFields(binding, "lease", "operation", "session_id") {
		return false
	}
	lease, ok := binding["lease"].(map[string]any)
	return ok && validateUpdaterLeaseDocument(lease) && validateUpdaterCommandDocument(lease["command"])
}

func requireUpdaterFields(object map[string]any, fields ...string) bool {
	for _, field := range fields {
		if !hasUpdaterField(object, field) {
			return false
		}
	}
	return true
}

func hasUpdaterField(object map[string]any, field string) bool {
	_, ok := object[field]
	return ok
}

func loadUpdaterDesiredCanonicalSchema() (*jsonschema.Schema, error) {
	updaterDesiredCanonicalSchema.once.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(denyUpdaterExternalSchemaLoader{})
		for _, name := range []string{
			updaterDesiredSchemaName,
			updaterSystemUpdateSchemaName,
			updaterSelfUpdateSchemaName,
			updaterReleaseBindingSchemaName,
		} {
			body, err := contractschemas.RuntimeValidationFS.ReadFile(name)
			if err != nil {
				updaterDesiredCanonicalSchema.err = err
				return
			}
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
			if err != nil {
				updaterDesiredCanonicalSchema.err = err
				return
			}
			if err := compiler.AddResource(name, document); err != nil {
				updaterDesiredCanonicalSchema.err = err
				return
			}
			if err := compiler.AddResource(updaterSchemaCanonicalBase+name, document); err != nil {
				updaterDesiredCanonicalSchema.err = err
				return
			}
		}
		updaterDesiredCanonicalSchema.schema, updaterDesiredCanonicalSchema.err = compiler.Compile(updaterDesiredSchemaID)
	})
	return updaterDesiredCanonicalSchema.schema, updaterDesiredCanonicalSchema.err
}

type denyUpdaterExternalSchemaLoader struct{}

func (denyUpdaterExternalSchemaLoader) Load(string) (any, error) {
	return nil, errUpdaterDesiredSchema
}

func decodeUpdaterStrictJSON(payload []byte, target any) (any, error) {
	if !utf8.Valid(payload) || validateUpdaterJSONTokens(payload) != nil {
		return nil, errUpdaterCommandInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, err
	}
	if err := requireUpdaterJSONEOF(decoder); err != nil {
		return nil, err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return document, nil
}

func validateUpdaterJSONTokens(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := validateUpdaterJSONValue(decoder); err != nil {
		return err
	}
	return requireUpdaterJSONEOF(decoder)
}

func validateUpdaterJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return errUpdaterCommandInvalid
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
				return errUpdaterCommandInvalid
			}
			if _, duplicate := seen[key]; duplicate {
				return errUpdaterCommandInvalid
			}
			seen[key] = struct{}{}
			if err := validateUpdaterJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errUpdaterCommandInvalid
		}
	case '[':
		for decoder.More() {
			if err := validateUpdaterJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errUpdaterCommandInvalid
		}
	default:
		return errUpdaterCommandInvalid
	}
	return nil
}

func requireUpdaterJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return errUpdaterCommandInvalid
	}
	return nil
}

func updaterCanonicalDocument(value any) (any, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(body))
}

// appendUpdaterJCS implements the RFC 8785 surface needed by the bounded
// Updater projection: objects, arrays, Unicode strings, booleans, and I-JSON
// safe integers. Schemas reject floating-point desired values.
func appendUpdaterJCS(destination *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		destination.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				destination.WriteByte(',')
			}
			if err := appendUpdaterJCSString(destination, key); err != nil {
				return err
			}
			destination.WriteByte(':')
			if err := appendUpdaterJCS(destination, typed[key]); err != nil {
				return err
			}
		}
		destination.WriteByte('}')
	case []any:
		destination.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				destination.WriteByte(',')
			}
			if err := appendUpdaterJCS(destination, item); err != nil {
				return err
			}
		}
		destination.WriteByte(']')
	case string:
		return appendUpdaterJCSString(destination, typed)
	case bool:
		if typed {
			destination.WriteString("true")
		} else {
			destination.WriteString("false")
		}
	case json.Number:
		return appendUpdaterJCSNumber(destination, typed.String())
	case float64:
		return appendUpdaterJCSFloat(destination, typed)
	case int:
		return appendUpdaterJCSFloat(destination, float64(typed))
	case int64:
		return appendUpdaterJCSFloat(destination, float64(typed))
	case nil:
		return errUpdaterCanonicalJSONInvalid
	default:
		return errUpdaterCanonicalJSONInvalid
	}
	return nil
}

func appendUpdaterJCSString(destination *bytes.Buffer, value string) error {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return err
	}
	destination.Write(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'}))
	return nil
}

func appendUpdaterJCSNumber(destination *bytes.Buffer, value string) error {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errUpdaterCanonicalJSONInvalid
	}
	return appendUpdaterJCSFloat(destination, number)
}

func appendUpdaterJCSFloat(destination *bytes.Buffer, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value ||
		math.Abs(value) > updaterMaxJCSSafeInteger {
		return errUpdaterCanonicalJSONInvalid
	}
	if value == 0 {
		destination.WriteByte('0')
		return nil
	}
	// The accepted projection contains integers only and is restricted to the
	// I-JSON safe range, which is below ECMAScript's 1e21 exponential threshold.
	// RFC 8785 therefore requires the ordinary decimal form.
	destination.WriteString(strconv.FormatFloat(value, 'f', 0, 64))
	return nil
}

func validUpdaterPortPlanDigest(plan SystemUpdatePortReconfiguration) bool {
	return updaterRawSHA256Pattern.MatchString(plan.PortPlanSHA256)
}
