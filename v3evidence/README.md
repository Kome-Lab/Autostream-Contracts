# v3 P-A evidence contract

This independent Go1.26.5 module owns the v3 P-A Guard evidence transport types.
It does not change the legacy root module, v2 APIs, schemas, OpenAPI or releases.
The initial source and synthetic tests are the previously verified candidate from
Updater f62469a4e319ca8487f1ac5502979ca15521d9d6.

Protocol3 has13 closed Kinds and exactly one discriminated payload:
CP authorization, Policy, Trust, membership, ownership, fence, generation,
Boot, process, P-Admission, Drain, Candidate and Business Admission. Each record is
a meaningful request-bound Scope/Execution/Binding/BackendReadFrame attestation,
not a global policy publication or a Host operation with dummy Stream/Run IDs.

Decode/Encode/Validate check structure and correlation only. Unknown Version/Kind,
duplicate keys (including escaped aliases), wrong case, extra/missing/null fields,
invalid UTF-8/trailing JSON, ambiguous payloads and invalid/overflowing integers
are rejected. Inputs are bounded at256KiB and depth32. A single valid field name
may use a JSON Unicode escape. Original wire bytes must remain available to the
future Integrity verifier. Serialization is not an adopted signature
canonicalization. Clone detaches payloads, time points and resource arrays.

SPECIFIED validity is only a claimed interval; UNKNOWN validity/revocation/state
can be transported but cannot qualify execution. EvidenceID, proof text, successful
Decode and self-declared status never authenticate an issuer or authorize Mutation.

Issuer/audience/payload integrity, key/profile/rotation/revocation, current
applicability and qualified clock checks remain mandatory verifier responsibilities.
No actual Producer, transport, signature/MAC algorithm, key product, Token format,
clock/TTL/skew or successful verifier default is supplied. Evidence issuance/expiry,
D1 decision, Backend commit and Context cancellation remain separate concepts.

CP owns business authorization and issuance. Agent delivers immutable evidence
without expanding authority. Executor independently validates current conditions.
Independent Continuity owns Witness/Outstanding/restore freshness separately;
Guard evidence must not be substituted for it. B/P/D remain distinct.

The actual v3 CP/Policy/Trust/Domain/ownership/Fence/Admission/Drain/Candidate
producers are not implemented by these types. Existing v2 grant/policy routes
are not converted or reused. Raw OS Boot/PID/start ticks do not themselves prove
a v3 incarnation, trusted clock or current authority.

Run from this module directory: go test ./..., go vet ./..., go build ./....
Dedicated v3 CI pinsGo1.26.5 and checks gofmt, tests, CGO-free tests, Race and
Linux amd64/arm64 test cross-builds. Tests are synthetic Codec contracts only;
they do not certify a deployed CP producer, real cryptography or native ARM64.

Reference/Receipt/History mutation, GuardControl, independent Continuity Store,
service apply, Media/Provider, host/DB/credential/backup/restore/deploy are absent.
Product-wide acceptance NOT_RUN; Phase1 closure on hold; production migration
unauthorized; formal v2 return NOT_CERTIFIED. All preservation/manual-hold/
old-authority-nonreuse/UNKNOWN/HELD constraints remain.

## Canonical Request / Binding ownership

This module now owns the shared **current v3 P-A** request contract. Request
schema1 is not AutoStream v1 runtime compatibility. The source of the exact
existing contract is Updater `6f71ee3a0906665f0a639abf69c27749ba211dc7`,
`internal/localexecutor/registry/types.go` and `validation.go` (CanonicalBinding,
ValidateRequest and DecodeRequest). No legacy v2 Grant or authority is converted.

| Version axis | Current meaning | Rejection / migration |
|---|---|---|
| Evidence ProtocolVersion | 3, Envelope transport | Existing Codec rules unchanged |
| CanonicalRequestVersion | 1, `Command.Schema` on wire | Only Schema1 accepted |
| CanonicalBindingVersion | 1, fixed v1 equality profile | Schema1 maps ONLY to binding1; CanonicalForm rejects any other profile |

The last two have equal numeric values but different meanings. Do not infer them
from Evidence Protocol3, a bare Binding string, a hash or a stored Receipt. The
optional `CanonicalForm` retains both versions, original canonical bytes and the
literal binding for explicit comparison; it is not a new wire field or proof.
Future changes to meaning, ordering, escaping or canonicalization require a new
Schema AND an explicit Schema-to-Binding-profile dispatch. Never reinterpret
stored schema1/profile1 bytes. Keep a separately scoped, reviewed one-time
conversion plan if ever required; no fallback or conversion is implemented here.

### Closed v1 encoding

`Command` reuses the existing public Scope/Execution types. They retain their
snake_case Evidence JSON tags. Canonical Request encoding uses fixed PascalCase
wire shapes and a closed explicit encoder, not the ordinary Marshal output of an
arbitrary struct or map. Public Command/Request JSON methods delegate to that
qualified codec so generic JSON use cannot accidentally choose the Evidence tags.

Canonical object field order is fixed:

| Object | Ordered fields |
|---|---|
| Request | Command, Binding |
| Command | Schema, Operation, Scope, Execution, RequestingPrincipal, AuthorizationIssuer, CandidateRevision, CandidateContentPin, Expected, RequiredGuard, Timing, OriginalBudget |
| Scope | Realm, Host, Resource, Backend |
| Execution | RequestID, ExecutionKey, OperationID, AttemptID, StepID |
| Expected | Revision, StateVersion |
| RequiredGuard | Policy, Trust, Membership, OwnershipEpoch, FenceEpoch, Generation, BootID, ProcessIncarnation |
| Timing | TimeBase, DeadlineTick |

All fields are mandatory, including a zero StateVersion. Operation is exactly
`registry.reference_only`. Every other string in Command is an ASCII opaque ID:
`[a-z0-9][a-z0-9._:-]{0,127}`. There is no trimming, case folding or Unicode
normalization. Unicode IDs, U+FFFD and lone surrogates are rejected; JSON Unicode
escapes which decode to permitted ASCII values/keys are accepted. Keys are
exact-case and duplicates are detected after escape decoding. Input object order
and insignificant whitespace can vary; canonical output cannot.

StateVersion is an exact uint64 in `0..MaxUint64-1`; DeadlineTick is in
`1..MaxUint64`. Schema is uint32 and only1 is known. Decimal integer output has no
leading zero, sign, exponent or fraction; integer JSON strings, null, overflow and
floating conversion are rejected. Canonical bytes are UTF-8/ASCII without BOM,
whitespace or a final newline. The entire input Request is limited to64KiB; the
shared strict walker bounds JSON depth at32 (the fixed schema is shallower).
Null, unknown/extra, duplicate, missing, wrong-case fields, malformed UTF-8 and
trailing JSON are rejected. Evidence Codec's existing behavior remains unchanged.

`ComputeBinding(Command)` is **the canonical Command JSON string itself**, exactly
matching the existing C10 Binding. It is not a new hash or a signature.
`VerifyBinding` requires full byte/string equality. Binding itself, delivery IDs,
BackendReadFrame, Consume/Seal/Receipt/Witness and later observations never enter
the Command preimage. EncodeRequest refuses a stale binding; it does not silently
recompute it. `VerifyCanonicalForm` additionally checks both explicit versions and
the original canonical bytes. `ValidateRequestFor` correlates a validated request
with a trusted Scope/Execution; it does not authenticate that target or caller.

### CP semantic mapping

The CP semantic Intent remains a consumer view, not the owner of this contract.
The mapping is lossless and does not introduce a new CP-only binding format.

| CP Intent | Shared contract | In canonical Binding? |
|---|---|---|
| Scope / Execution | Command.Scope / Execution | All4 /5 fields |
| Action | Command.Operation | Yes |
| RequestingPrincipal / AuthorizationIssuer | Same-named Command fields | Yes; identity/authenticity still requires CP authentication |
| CandidateRevision / CandidateContentPin | Same-named Command fields | Yes; these are not Candidate approval Evidence |
| ExpectedRevision / ExpectedStateVersion | Expected.Revision / StateVersion | Yes |
| Required.Policy / Trust / Membership | RequiredGuard.Policy / Trust / Membership | Yes |
| Required.OwnershipEpoch / FenceEpoch | RequiredGuard.OwnershipEpoch / FenceEpoch | Yes |
| Required.Generation / BootID / ProcessIncarnation | Same-named RequiredGuard fields | Yes |
| TimeBase / DeadlineTick | Timing.TimeBase / DeadlineTick | Yes; no clock or D1 implementation supplied |
| OriginalBudget | Command.OriginalBudget | Yes; no reset or extra Budget authorized |
| Binding | Request.Binding | No self-reference; exact derived equality value |
| BackendReadFrame / later Evidence | Outside Command | No |

Authentication and business/target authorization remain CP responsibilities.
Matching Binding does not prove principal authenticity, current permission,
uniqueness, unconsumed authority, target freshness or a qualified clock. No key,
crypto, time, revocation, Continuity or Mutation implementation is added here.
The pure Codec has no Context or I/O; cancellation is checked at the CP consumer
boundary rather than reinterpreted as Validity or D1.

### Three-repository verification boundary

`scripts/ci/canonical_*_interop_test.go` are copied ONLY into disposable pinned
Updater/CP checkouts. Their temporary module replacement points to the candidate
Contracts checkout and is never committed in either product repository. The tests
compare actual C10 bytes/Binding/full-field mutations and strict decoding, then
use the shared Decoder in a temporary CP Resolver and reverse-map every Intent
field. The actual CP Service issues/saves/reads an Envelope which is returned to
the actual C10 Guard Codec/source/authenticator/Preflight boundary.

That temporary Resolver is not a persisted CP product connection. Updater's local
CanonicalBinding and CP's production dependency pin/Resolver remain unchanged in
this task. Their future integration must pin the reviewed shared module and use
explicit typed conversions (no Updater internal import). The remaining CP
Authority, Integrity, Validity/revocation, other12 Producers, storage collaborators
and independent Continuity in the bridge remain synthetic test doubles. A green
bridge does not make real CP issuance or C10 ELIGIBLE available in production.
