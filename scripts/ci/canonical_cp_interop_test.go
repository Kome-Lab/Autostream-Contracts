//go:build canonical_request_interop

// Copied only into the pinned disposable ControlPanel checkout. This qualified
// shared-contract Resolver is TEST-ONLY: CP PR12 still has no persisted Resolver
// connection. Business Authority/Integrity/Validity/revocation remain doubles.
package v3authorization

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	evidence "github.com/Kome-Lab/Autostream-Contracts/v3evidence"
)

type canonicalCPFixture struct {
	Request        json.RawMessage           `json:"request"`
	CanonicalBytes string                    `json:"canonical_bytes"`
	Scope          evidence.Scope            `json:"scope"`
	Execution      evidence.Execution        `json:"execution"`
	Frame          evidence.BackendReadFrame `json:"backend_read_frame"`
	Issuer         string                    `json:"issuer"`
	Audience       string                    `json:"audience"`
}

// This is a candidate CP adapter, not another Binding algorithm. DecodeRequest
// verifies the original canonical binding before ANY CP semantic translation.
type canonicalCPResolver struct{}

func (canonicalCPResolver) Resolve(ctx context.Context, raw json.RawMessage) (Intent, error) {
	if err := ctx.Err(); err != nil {
		return Intent{}, err
	}
	request, err := evidence.DecodeRequest(raw)
	if err != nil {
		return Intent{}, ErrConflict
	}
	c := request.Command
	intent := Intent{Scope: c.Scope, Execution: c.Execution, Binding: request.Binding, Action: c.Operation,
		RequestingPrincipal: c.RequestingPrincipal, AuthorizationIssuer: c.AuthorizationIssuer,
		CandidateRevision: c.CandidateRevision, CandidateContentPin: c.CandidateContentPin,
		ExpectedRevision: c.Expected.Revision, ExpectedStateVersion: c.Expected.StateVersion,
		Required: GuardRequirement{Policy: c.RequiredGuard.Policy, Trust: c.RequiredGuard.Trust, Membership: c.RequiredGuard.Membership,
			OwnershipEpoch: c.RequiredGuard.OwnershipEpoch, FenceEpoch: c.RequiredGuard.FenceEpoch, Generation: c.RequiredGuard.Generation,
			BootID: c.RequiredGuard.BootID, ProcessIncarnation: c.RequiredGuard.ProcessIncarnation},
		TimeBase: c.Timing.TimeBase, DeadlineTick: c.Timing.DeadlineTick, OriginalBudget: c.OriginalBudget}
	if !validIntent(intent) {
		return Intent{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Intent{}, err
	}
	return intent, nil
}

// The reverse map exists only to prove no CP-specific reinterpretation loses a
// C10 field. It does not define a second public Request or canonical serializer.
func canonicalCPCommand(intent Intent) evidence.Command {
	return evidence.Command{Schema: evidence.CanonicalRequestVersion, Operation: intent.Action, Scope: intent.Scope, Execution: intent.Execution,
		RequestingPrincipal: intent.RequestingPrincipal, AuthorizationIssuer: intent.AuthorizationIssuer,
		CandidateRevision: intent.CandidateRevision, CandidateContentPin: intent.CandidateContentPin,
		Expected: evidence.ExpectedReference{Revision: intent.ExpectedRevision, StateVersion: intent.ExpectedStateVersion},
		RequiredGuard: evidence.RequiredGuard{Policy: intent.Required.Policy, Trust: intent.Required.Trust, Membership: intent.Required.Membership,
			OwnershipEpoch: intent.Required.OwnershipEpoch, FenceEpoch: intent.Required.FenceEpoch, Generation: intent.Required.Generation,
			BootID: intent.Required.BootID, ProcessIncarnation: intent.Required.ProcessIncarnation},
		Timing: evidence.CommandTiming{TimeBase: intent.TimeBase, DeadlineTick: intent.DeadlineTick}, OriginalBudget: intent.OriginalBudget}
}

func canonicalCPInput(t *testing.T) (canonicalCPFixture, evidence.Request, Intent) {
	t.Helper()
	directory := os.Getenv("CANONICAL_INTEROP_DIR")
	if directory == "" || !filepath.IsAbs(directory) {
		t.Fatal("explicit absolute synthetic interop directory required")
	}
	raw, err := os.ReadFile(filepath.Join(directory, "c10-canonical-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture canonicalCPFixture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	request, err := evidence.DecodeRequest(fixture.Request)
	if err != nil {
		t.Fatal("actual C10 request rejected", err)
	}
	intent, err := (canonicalCPResolver{}).Resolve(context.Background(), fixture.Request)
	if err != nil {
		t.Fatal(err)
	}
	if intent.Scope != fixture.Scope || intent.Execution != fixture.Execution || intent.AuthorizationIssuer != fixture.Issuer {
		t.Fatal("exported target/execution/issuer lost")
	}
	if !reflect.DeepEqual(canonicalCPCommand(intent), request.Command) {
		t.Fatal("CP semantic conversion dropped or changed a field")
	}
	canonical, err := evidence.CanonicalBytes(canonicalCPCommand(intent))
	if err != nil || string(canonical) != fixture.CanonicalBytes || string(canonical) != request.Binding || intent.Binding != request.Binding {
		t.Fatal("CP and actual C10 canonical bytes/binding differ", err)
	}
	if err = evidence.VerifyBinding(canonicalCPCommand(intent), fixture.CanonicalBytes); err != nil {
		t.Fatal(err)
	}
	return fixture, request, intent
}

func canonicalCPServiceFixture(t *testing.T, fixture canonicalCPFixture, intent Intent) *serviceFixture {
	t.Helper()
	f := makeTestFixture(t)
	f.request = IssueRequest{OriginalRequest: append(json.RawMessage(nil), fixture.Request...), Frame: fixture.Frame}
	configureIntent(f, intent)
	f.deps.Resolver = canonicalCPResolver{}
	f.service = NewService(Config{Issuer: fixture.Issuer, Audience: fixture.Audience}, f.deps)
	return f
}

func TestCanonicalSharedCPServiceInterop(t *testing.T) {
	fixture, request, intent := canonicalCPInput(t)
	encoded, err := evidence.EncodeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	// The shared Encoder preserves the exact existing C10 Request wire as well as
	// canonical Command bytes; no special CP-only request representation is used.
	if !bytes.Equal(encoded, fixture.Request) {
		t.Fatal("shared Request encoding changed actual C10 wire")
	}
	f := canonicalCPServiceFixture(t, fixture, intent)
	issued := f.issue()
	if issued.Status != Issued || issued.AuthorizesExecution {
		t.Fatal("actual CP Service did not issue", issued.Status, issued.Code)
	}
	envelope, err := evidence.Decode(issued.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Binding != request.Binding || envelope.Scope != request.Command.Scope || envelope.Execution != request.Command.Execution ||
		envelope.CPAuthorization.PolicyRevision != request.Command.RequiredGuard.Policy || envelope.CPAuthorization.TrustRevision != request.Command.RequiredGuard.Trust {
		t.Fatal("actual CP Producer lost shared semantic correlation")
	}
	lookup := Lookup{Scope: intent.Scope, Execution: intent.Execution, EvidenceID: envelope.EvidenceID}
	read := f.service.Read(context.Background(), f.principal, lookup)
	if read.Status != Replay || !bytes.Equal(read.Envelope, issued.Envelope) || f.security.seals != 1 || f.store.finishes != 1 {
		t.Fatal("CP read regenerated or changed shared-contract evidence")
	}
	if err = os.WriteFile(filepath.Join(os.Getenv("CANONICAL_INTEROP_DIR"), "cp-canonical-envelope.json"), read.Envelope, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("actual CP Service with disposable shared Resolver mapped ALL Command fields without loss; security/authority/time/store synthetic, CP persistent source remains unconnected")
}

func TestCanonicalSharedCPRejectsBindingScopeExecutionAndPrincipal(t *testing.T) {
	fixture, request, intent := canonicalCPInput(t)
	for _, name := range []string{"binding", "scope-stale", "execution-stale", "scope-valid-binding", "execution-valid-binding", "principal", "context"} {
		t.Run(name, func(t *testing.T) {
			f := canonicalCPServiceFixture(t, fixture, intent)
			ctx := context.Background()
			candidate := request
			switch name {
			case "binding":
				candidate.Binding = "tampered"
			case "scope-stale", "scope-valid-binding":
				candidate.Command.Scope.Host = "another-host"
			case "execution-stale", "execution-valid-binding":
				candidate.Command.Execution.AttemptID = "another-attempt"
			case "principal":
				candidate.Command.RequestingPrincipal = "another-principal"
			case "context":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if name == "scope-valid-binding" || name == "execution-valid-binding" || name == "principal" {
				var err error
				candidate.Binding, err = evidence.ComputeBinding(candidate.Command)
				if err != nil {
					t.Fatal(err)
				}
			}
			// Marshal deliberately creates invalid proposals for the Decoder, rather than
			// asking the shared Encoder to accept a stale/tampered binding.
			var body map[string]json.RawMessage
			if err := json.Unmarshal(fixture.Request, &body); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "binding":
				body["Binding"], _ = json.Marshal(candidate.Binding)
			case "context":
			default:
				valid := candidate
				valid.Binding, _ = evidence.ComputeBinding(candidate.Command)
				encoded, err := evidence.EncodeRequest(valid)
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(encoded, &body); err != nil {
					t.Fatal(err)
				}
				if name == "scope-stale" || name == "execution-stale" {
					body["Binding"], _ = json.Marshal(request.Binding)
				}
			}
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			if name == "execution-valid-binding" {
				// A genuinely different execution is not a cryptographic collision; the CP
				// business Authority must reject it. Its explicit current refusal is separate
				// from equality syntax, which is never permission.
				f.authority.decision.State = "DENIED"
			}
			result := f.service.Issue(ctx, f.principal, IssueRequest{OriginalRequest: raw, Frame: fixture.Frame})
			if result.Status == Issued || result.Status == Replay || len(result.Envelope) != 0 || f.security.seals != 0 || f.store.finishes != 0 {
				t.Fatal("invalid/unauthorized/canceled shared request issued", name, result.Status, result.Code)
			}
		})
	}
}
