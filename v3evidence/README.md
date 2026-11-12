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
