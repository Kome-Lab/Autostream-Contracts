# AutoStream v3 G0 lifecycle contracts

Independent module: `github.com/Kome-Lab/Autostream-Contracts/v3lifecycle`.
Go 1.26.5, `GOWORK=off`, standard library only, no dependencies.

Normative logical source: [Master R2 §24, G0-LOGICAL-20261115-R1](https://chatgpt.com/space/page_ac2afef3ac148191bb661aa4eff4c6bf).
This is IU-CC-01 / P-R02. The shared schema owner is Contracts. The
implementation and conformance freeze is limited to G0-C01 through G0-C07;
the final candidate SHA, checked tree, test inventory, and CI evidence are
recorded in [PR #1](https://github.com/Kome-Lab/Autostream-Contracts/pull/1).

## Meaning of a result

`STRUCTURALLY_COHERENT` means that supplied, bounded claims agree with a
supplied logical profile. It is not authentication, independent freshness,
a durable guard, evidence issuance, a READY capability, or execution permission.
Every `Result.ProductionQualified()` and `Result.AuthorizesMutation()` returns
false. A fixture remains test data even when its Gate is VERIFIED_FOR_PROFILE.

Product owners retain their DB, transaction, private participant, provider,
payload, clock, trust root, authorization, and all-actor enforcement. This
module does not perform IO or inspect the environment. No constructor can
issue or consume a permit, approval, recovery capability, or qualified seal.
Nonsecret authority/capability references are only claims that a product
adapter must independently authenticate for its exact target and purpose.

## Files and contract coverage

| Contract | Immutable payload | Validation / model | Required vectors |
| --- | --- | --- | --- |
| CC01 Domain/membership | MembershipRecord, DomainRef, Member, Alias | ValidateRecord, CompareImmutable; explicit complete/empty roster and alias coverage | V01, V02, V19 |
| CC02 Generation/Revision | RevisionVector, AxisKey, VersionedValue, ComparisonProfile | ValidateProfile, ValidateVector, CompareVectors | V03, V20 |
| CC03 Admission/Drain | AdmissionRecord, ReservationRefs, Namespace, WorkRef | ReduceAdmission; durable child registration and handoff before dispatch; all guard ACKs before CLOSED | V04, V15 |
| CC04 Completion Proof | CompletionRecord, PartitionProof, RegistrationClosure, PhysicalObligation, EffectRequirement | AssessCompletion, AssessCompletionWithEffects; typed final frontier and physical/remote obligations | V05, V24 |
| CC05 READY Seal | SealRecord, ProofRef; explicit release/config/policy/trust AxisKeys | ValidateRecord, ValidateSealAgainstCompletion; same immutable attempt/domain/reservations and prior work only | V06, V16 |
| CC06 Final Fence | FenceRecord; target, consumed-authority reference, target vector, D1 semantics | ValidateRecord; target-bound ownership/fence epochs; observation only | V07, V21 |
| CC07 External Effect | EffectRecord, BoundaryObservation, BoundaryDetails | ReduceEffect, AssessObservations, AssessObservationHistory, AssessFanout | V08, V22, V24 |
| CC08 UNKNOWN/Inspect | QueryTarget, InspectionRecord, ObservationSet | InspectSameAttempt, ValidateObservationSet; scoped frame/owner/revision binding | V09, V10, V11, V23 |
| CC09 Recovery/reconcile responsibility | ReconcileRecord, MetadataDelta, typed ExpectedProjection | CompareProjectionCAS, ValidateRecordTransfer, AssessRestoreFreshness | V12, V13, V14 |

`types.go` and `enums.go` contain DTOs and closed registries; `validate.go`,
`profile.go`, `record.go`, and `observation.go` validate their respective
boundaries. `model.go` contains pure state/CAS proposals. `codec.go` only
decodes internal conformance data; it is not an adopted external or signing wire.
`registry.go` records shared ownership and consumer expectations.

## Explicit profiles and limits

All seven logical profiles are revision 1:
G0-P-APPROVAL-LOCAL, G0-P-MEDIA-INGRESS, G0-P-CLOSURE-ASSESS,
G0-P-STOP-LOOKUP, G0-P-EFFECT-OUTCOME, G0-P-RECORD-REPAIR,
G0-P-EXPLICIT-RESUME. A caller supplies the complete frozen profile, exact
resource-bound required/permitted keys, owner roles, typed value rules,
purposes, required boundaries, terminal policy, and explicit limits.
Minimum logical axes are checked; no profile, limit, terminal policy, missing
observation, or current revision is filled from a default.

The 19 axes remain separate, including RELEASE and CONFIG. A/C/Identity,
Membership/Job/Connection/Writer/Effect/Cover use uint64 values; the remaining
registered axes use literal references. Literal `"1"` is not uint64 `1`.
Each axis key is (Axis, OwnerID, ResourceRef); an incarnation change cannot
create a second entry with the same key. Numeric zero is a present value.
Revision increments refuse overflow. Comparisons are exact, never a global
greater-than rule or a wall-clock ordering.

Limits require positive MaxIDBytes, MaxMembers, MaxAxes, MaxRefs,
MaxObligations, MaxPartitions, MaxRanges, and MaxDepth. The JSON decoder also
bounds document bytes and object/array members using these explicit values.
The small fixture limits are not production budgets or deployment policy.

A Fact is OBSERVED with value and owner/frame/scope/revision, or UNAVAILABLE
with a reason and missing references. Optional references explicitly distinguish
PRESENT, ABSENT with an observed absence fact, and NOT_OBSERVED. An unavailable
branch has no fabricated current frame. Absence, missing evidence, unknown
outcome, and proven no-effect are different states.

New IDs preserve valid UTF-8 literals without trimming, case conversion, or
Unicode normalization. The internal decoder rejects duplicate keys (including
Unicode case aliases), unknown fields, invalid UTF-8, unpaired surrogate escapes,
overflow, excessive depth, and ambiguous unions. Diagnostics contain only code
and structural path, never submitted values.

## State and persistence obligations

Admission has independent OPEN → CLOSING → CLOSED and WAITING/PROVED axes.
A cut closes new independent admission; already accepted lineage may register
bounded children under the original scope. The local owner must durably register
a child before dispatch and durably hand off before releasing the parent duty.
The reducer computes a proposal and does not install a real guard.

Work sequences start at 1. EMPTY requires an explicit owner fact; THROUGH
requires a positive sequence. CutFrontier is separate from FinalAcceptedFrontier,
which includes accepted child continuation after the cut. Holes, pending
registrations, live callbacks, FD/connection releases, and remote effects
prevent structural closure. Registration closure binds snapshot, admission epoch,
partition, and registration epoch. A queue length of zero is not completion.
Safety termination is separate from business success and valid media output.

An effect progresses from NO_INTENT to INTENT_RECORDED to DISPATCH_POSSIBLE
only through explicit durable-fact CAS proposals. Product owners must finish
local intent/dispatch persistence before network IO and release SQL locks before
external calls. The module does not send anything. ACK loss after dispatch
possibility remains UNKNOWN; it does not reopen dispatch, choose a new ID,
release a reservation, reconsume approval, or retry the business operation.

Effect models declare an explicit local record owner and exact expected boundary
addresses. Provider observations bind the exact effect target; local persistence
and projection may have other explicitly declared resources. Immutable known
history cannot be overwritten by a contradictory current value. History is
append-only; the current view has unique boundary addresses. Returned mutable
slices, facts, and typed detail pointers are copied.

The six boundaries are CP_LIFECYCLE, LEDGER, BACKEND_MUTATION, PROVIDER,
LOCAL_PERSISTENCE, and PROJECTION. Sub-boundaries are closed. Raw writes carry
typed extent metadata, durability carries an explicit logical profile and witness,
and queue observations carry work/partition/sequence references. Raw bytes and
secrets are not fields. A transport response, remote acceptance, remote terminal
result, local DB commit, participant ACK, and result projection remain distinct.

Known SQL/CP commit can coexist with UNKNOWN connection release, Provider outcome,
or projection. PhaseHint records only a last attempted boundary. It cannot prove
a commit. Timeout, cancellation, restart, and inspection never implicitly OPEN,
force READY, stop all media, replay, repair, release HELD, or mint authority.

## Immutable equality and record-only CAS

CompareImmutable validates both records and compares typed immutable content.
Declared collections are bounded sets: order does not matter and duplicates are
rejected. RecordRevision alone is mutable; old request/target/profile/correlation,
proof, membership, and source facts cannot be replaced under the same key.
Equality is not a hash, authenticity check, or independent-freshness proof.

A reconciliation plan requires its original immutable correlation, authentic
source references (verified by the consumer), current typed vector, fresh
purpose-specific record capability reference, typed projection target, exact
expected projection revision, and fixed metadata delta. CAS loss is STALE;
the model never merges, rebases, or writes. Receipt transfer is a separate
RECORD_TRANSFER purpose and is not a Provider/business retry. Real Recovery,
explicit resume, root rotation, and formal v2 return have different authority.

## Consumer and ownership register

The independent test fixtures contain 23 consumer-role rows, covering all
9 repositories, all 14 logical components, and all 7 workstreams. Every row
uses the same logical revision, profile input, and independently written expected
outcomes. This tests shared interpretation, not installed product adapters.

| Repository | Components | Product responsibility retained |
| --- | --- | --- |
| R01 ControlPanel | C01 C02 C14 | CP identity/business Tx, authorization/consumption, HTTP/UI |
| R02 Contracts | C03 | Shared non-authority schema, validators, profiles, vectors |
| R03 Encoder-Recorder | C04 C05 | Ingest, raw/queue/mux/sink, recording/archive facts |
| R04 Worker | C06 | Job/scene/caption generation and child obligations |
| R05 DiscordBot | C07 | Gateway/VC/connection, sink-specific dispatch |
| R06 Observability | C08 | Incident and per-channel notification effects/history |
| R07 Updater | C09 C10 C11 | Host/target backend, durable receipts, separate recovery |
| R08 Docker | C12 C14 | Deployment/release declarations and packaging |
| R09 Docs | C13 C14 | Feature/scenario/runbook mapping |

WS1 Trust, WS2 Control Plane/Identity, WS3 Media, WS4 Execution, WS5
Observability, WS6 Release/Recovery/Migration, and WS7 UI/Documentation can
prepare adapters in parallel against this module's fixed commit. None of those
repository changes is authorized or completed by IU-CC-01.

## Compatibility and versioning

The existing root module and v3evidence module are unchanged from
`45aaa9932bcd92c728c86c95349853724fe1accc`. Canonical Request1, Binding1,
Evidence3, their 26 mutable semantic fields and original literal bytes,
APIs, diagnostics, errors, old Admission/Drain enums, and existing pins remain
owned by v3evidence. No automatic new-state conversion is provided.
LegacyProjection returns NO_PROJECTION/NEEDS_DECISION for valid new states,
including PROVED; it never emits an old READY envelope.

Existing fixed disposable interop baselines remain Updater
`6f71ee3a0906665f0a639abf69c27749ba211dc7`, ControlPanel
`9a673d052a9a954570de8901044f1eb68790a870`, and original Request API
baseline `790be850e98c06119330907a8fc2ae091e2bdc85`.
Test overlays and temporary replacements remain confined to CI's disposable
copies. Product repositories, remotes, branches, pins, and PRs are not modified.

A new consumer must pin this independent module at the reviewed commit and
supply the same ContractRef logical revision/spec/module-commit reference.
The synthetic fixture commit is deliberately not a real consumer pin. A
ModuleCommitRef is only checked syntactically; it is not source authenticity.
Registry/semantic additions require a reviewed logical revision and conformance
update. Existing wire or old evidence enums must not silently absorb them.

## Verification

From this directory with Go 1.26.5 and GOWORK=off:

```sh
go test -count=1 -json ./... > lifecycle-tests.json
python3 -B ../scripts/ci/lifecycle_test_events.py lifecycle-tests.json ../scripts/ci/lifecycle-required-tests.txt
go vet ./...
go build ./...
CGO_ENABLED=0 go test -count=1 ./...
CGO_ENABLED=1 go test -race -count=1 ./...
python3 -B ../scripts/ci/lifecycle_source_boundary.py
python3 -B ../scripts/ci/lifecycle_ci_test.py
```

Write logs outside the source directory in CI; generated files must not be
committed. The required inventory fixes G0-V-01 through G0-V-24 to their exact
root test names. Every native suite's JSON gate checks one run and one pass for
each root, no skipped/failed event anywhere, exact package identity, and completed
package execution. The checker itself has adversarial negative tests.

The workflow appends v3-lifecycle-contract-checks with its own job-level
working directory. It runs all module tests, vet/build, Race, CGO-free tests,
Linux amd64/arm64 cross-builds, source/privacy checks, and module/pin invariance.
It also tests the unchanged root Go module on Linux. The old
v3-evidence-checks job and all its fixed interop/diagnostic steps remain intact.
Cross-built binaries are not executed and do not prove native arm64 or host,
storage, codec, transport, independent-clock, or Provider qualification.

Windows checkout with core.autocrlf=true causes existing root OpenAPI tests
that compare literal LF strings to fail. The same fixed Git tree, exported with
core.autocrlf=false into a separate disposable directory, passes those root tests.
No existing source or test is rewritten to hide that environment difference.

## Remaining release blockers and handoff

Not implemented here: real provider/DB adapters; private transaction/fence handles;
real Approval Origin/Consumption/Ceiling; actual membership discovery/all-actor
guards; independent L2; D4 secret commitment; qualified Root/Clock/storage/host;
production READY/Final Fence; actual recovery/restore/formal v2 return; real media
or notification delivery. Model tests do not prove HTTP, independent DB/process,
barrier races, real ACK loss, partial restore, or Provider behavior.

UD1–UD4, audio failure authority DP-06, and the unadopted release/recovery model
remain unresolved. No Root, quorum, clock, storage product, F5 method, deployment
profile, or real recovery authority is selected by this code.

For P-R01 and P-R03 onward: pin the final reviewed module commit; use the
same profile/vector or propose a reviewed revision; preserve original IDs,
B0/B1/B2/S13 and lock ordering; authenticate exact owner frames and resources;
add actual typed adapters and HTTP/DB/process/provider failure evidence under
separate explicit repository permissions. Keep legacy entrypoints compatible.
Media/Execution preparation does not wait for all WS1 production qualifications,
and no preparation bypasses the missing trust gate.

Initial v3 still requires all planned features implemented, integrated, tested,
and accepted. L2 placement NOT_QUALIFIED; real L2/product acceptance NOT_RUN;
Phase1 remains open; production transition unauthorized; formal v2 return
NOT_CERTIFIED. AWS/new hosts, deployment, merge, tag, and release are outside
this change.
