"""Source checks for the isolated MediaRelease1 contract surface."""
from __future__ import annotations

import ast
import hashlib
import json
import re
import stat
import subprocess
from pathlib import Path

BASELINE = "d32ea3514b1c1010391caae0348b0ff9e751f447"
MOD_DIR = "v3mediarelease"
MODULE = "github.com/Kome-Lab/Autostream-Contracts/" + MOD_DIR
JOB = "v3-media-release-contract-checks"
WORKFLOW = ".github/workflows/v3-evidence-ci.yml"
OPENAPI = "openapi/media-release-v1.yaml"
CHARACTERIZATION = (
    "pkg/contracts/openapi_characterization_test.go",
    "scripts/characterize-contracts.mjs",
)
LEGACY_APIS = (
    "openapi/control-api.yaml",
    "openapi/discord-bot-api.yaml",
    "openapi/encoder-recorder-api.yaml",
    "openapi/observability-api.yaml",
)
# Before/after canonical source bytes for the fifth-entrypoint comparison and
# its conformance checks. An unrelated edit in either broad registry file must
# still fail even when its tests happen to pass.
REGISTRY_OUTPUT_HASHES = {
    "pkg/contracts/openapi_characterization_test.go": (
        "d6ef6712352afef74e94242877639ce415179c9d46d28748242a8fc69e61cb53",
        "e5045eaa5135b1d532c7cb780c5f9f80a5d19809e60adca8d878847800371eba",
    ),
    "scripts/characterize-contracts.mjs": (
        "b0d9aeed53a7b39335510a177bdf1d30056511337aad1294ddfa47af4080711b",
        "b6331a342d978d1765b7d0fb90c8b2e66a9a1bb1c51a2fa84848565efd8f53fd",
    ),
}
CI_PATHS = frozenset((
    "scripts/ci/mediarelease_source_boundary.py",
    "scripts/ci/mediarelease_test_events.py",
))
# Exact paths: no new executable, nested module, schema or CI helper can ride
# along with a contract addition. Every required artifact must be present.
MODULE_FILES = frozenset((
    "go.mod", "types.go", "validate.go", "codec.go", "model.go", "negotiation.go",
    "consumption.go", "codec_test.go", "source_boundary_test.go", "conformance_test.go",
    "testdata/handoff.json", "testdata/required-roots.txt",
))
NEW_PATHS = frozenset(MOD_DIR + "/" + p for p in MODULE_FILES) | CI_PATHS | {OPENAPI}
EXTENSION_PATHS = NEW_PATHS | set(CHARACTERIZATION) | {
    WORKFLOW, "scripts/ci/lifecycle_source_boundary.py", "scripts/ci/lifecycle_ci_test.py",
}
# Parsing supplied UTC values is deterministic. The AST gate checks time
# selectors independently and denies access to timers and the process clock.
ADDITIONAL_IMPORTS = frozenset(("crypto/sha256", "encoding/hex", "time"))
AUTHORITY_NAME = re.compile(
    r"^(?:(?:new|issue|mint|consume).*(?:permit|authority|approval|recovery|qualification)"
    r"|qualify(?:production|seal))$", re.I,
)
SENSITIVE_NAMES = frozenset((
    "password", "passwordhash", "mfasecret", "recoverycode", "recoveryhash",
    "sessiontoken", "csrftoken", "oauthsecret", "privatekey", "encryptionkey",
    "rawpayload", "rawaudio", "rawvideo", "rawmedia", "credential", "credentials",
    "notificationbody", "notificationurl", "connectionurl", "accesstoken", "refreshtoken",
))
PERSONAL_PATH = re.compile(r"/(?:home|Users)/[A-Za-z0-9_.-]+/")
TUPLE = {
    "family": "MediaRelease1", "wire_version": "1",
    "profile_id": "MEDIA_RELEASE_HANDOFF_V1", "profile_revision": "1",
    "encoding_id": "MR1_JSON_ASCII_V1",
}
KINDS = {
    "CPAdmissionProof": ("CC10", "CP_ADMISSION_PROOF", "ADMISSION_AUTHORITY"),
    "WorkerNonMediaBinding": ("CC10", "WORKER_NON_MEDIA_BINDING", "NON_MEDIA_BINDING"),
    "AppliedCoverProof": ("CC11", "APPLIED_COVER_PROOF", "COVER_ATTESTATION"),
    "ReleasePermit": ("CC12", "MEDIA_RELEASE_PERMIT", "AUTHORIZE_OPEN"),
    "OneTimeConsumption": ("CC13", "MEDIA_RELEASE_CONSUMPTION", "CONSUME_OPEN"),
    "ReleaseReceipt": ("CC14", "MEDIA_RELEASE_RECEIPT", "REPORT_RESULT"),
}
OPERATIONS = {
    "/capabilities": ("get", "getMediaReleaseCapabilities"),
    "/admissions": ("post", "issueStartAdmissionProof"),
    "/worker-bindings": ("post", "bindNonMediaWorker"),
    "/applied-cover-proofs": ("post", "attestAppliedCover"),
    "/permits": ("post", "issueMediaReleasePermit"),
    "/permit-revocations": ("post", "revokeMediaReleasePermit"),
    "/opens": ("post", "openMediaRelease"),
    "/fences": ("post", "fenceMediaRelease"),
    "/records/inspect": ("post", "inspectMediaReleaseRecord"),
    "/receipts/inspect": ("post", "inspectMediaReleaseReceipt"),
}
OPERATION_SCHEMAS = {
    "/capabilities": (None, "Capability"),
    "/admissions": ("AdmissionRequest", "CPAdmissionProof"),
    "/worker-bindings": ("BindingRequest", "WorkerNonMediaBinding"),
    "/applied-cover-proofs": ("CoverRequest", "AppliedCoverProof"),
    "/permits": ("PermitRequest", "ReleasePermit"),
    "/permit-revocations": ("FenceRequest", "FenceObservation"),
    "/opens": ("OpenRequest", "OneTimeConsumption"),
    "/fences": ("FenceRequest", "FenceObservation"),
    "/records/inspect": ("InspectRequest", "RecordInspection"),
    "/receipts/inspect": ("ReceiptInspectRequest", "ReceiptInspection"),
}
COMMON_IDENTITY = frozenset(
    "family wire_version encoding_id profile_id profile_revision cc kind record_key issuer "
    "domain original audience purpose issued_at clock_ref".split()
)
KIND_REQUIRED = {
    "CPAdmissionProof": "claim_ref worker_original_ref encoder_original_ref worker_assignment encoder_assignment "
                        "membership_ref membership_snapshot_ref member_revision current_frame_refs reservation_ref "
                        "domain_closure_ref admission_epoch hold_ref stop_fence_ref window",
    "WorkerNonMediaBinding": "cp_admission_ref worker_original_ref encoder_original_ref worker_assignment "
                             "reservation_ref admission_epoch membership_snapshot_ref manager_process_id "
                             "worker_generation local_binding_ref non_media_fence_ref window",
    "AppliedCoverProof": "cp_admission_ref worker_binding_ref preparation_id preparation_generation worker_generation "
                         "graph_id graph_generation graph_config_ref desired_cover_ref requested_revision "
                         "applied_revision applied_active feed_acceptance_ref output_progress_ref qualified_sink_ref "
                         "source_frame_ref closures observed_at window",
    "ReleasePermit": "permit_id release_attempt_id logical_effect_key connection_operation_id cp_admission_ref "
                     "worker_binding_ref applied_cover_ref scope closure_refs authority_ref revocation_ref "
                     "stop_fence_ref lease_ref window consumption_key receipt_key expected_slot_revision slot_reservation_ref",
    "OneTimeConsumption": "permit_ref logical_effect_key connection_operation_id receipt_key expected_slot_revision open_request_digest",
    "ReleaseReceipt": "permit_ref consumption_ref logical_effect_key scope store_record_id",
}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError("media release source boundary: " + message)


def check_production(code: str, text: str, relative: str) -> None:
    # The native AST conformance root independently inspects actual declaration
    # and struct-tag nodes, including grouped fields and package aliases.
    for name in re.findall(r"\bfunc\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*(?:\[|\()", code):
        require(not AUTHORITY_NAME.match(name), "authority issuer forbidden: " + relative)
    require(not re.search(r"(?m)^\s*//(?:go:|line )", text), "production directive forbidden: " + relative)
    require(not re.search(r"\bpackage\s+main\b", code), "executable package: " + relative)
    for selector in re.findall(r"\btime\s*\.\s*([A-Za-z_]\w*)", code):
        require(selector in {"Parse", "Time", "RFC3339Nano", "Duration", "Nanosecond"},
                "ambient clock, timers or location access: " + relative)
    for names in re.findall(r"(?m)(?:^|[;{])\s*([A-Za-z_]\w*(?:\s*,\s*[A-Za-z_]\w*)*)\s+(?:\*?\w|\[)", code):
        require(not any(n.strip().casefold().replace("_", "") in SENSITIVE_NAMES for n in names.split(",")),
                "secret/raw field: " + relative)
    for tag in re.findall(r'\bjson:"([^",]+)', text):
        require(tag.casefold().replace("_", "") not in SENSITIVE_NAMES, "secret/raw wire name: " + relative)


def check_workflow(repo: Path, workflow: bytes) -> None:
    prefix = subprocess.check_output(["git", "-C", str(repo), "show", BASELINE + ":" + WORKFLOW])
    require(workflow.startswith(prefix), "existing CI jobs or action pins changed")
    suffix = workflow[len(prefix):]
    lines = suffix.decode("utf-8").splitlines()
    require([s for s in lines if re.match(r"^  [^ ]", s)] == ["  " + JOB + ":"],
            "expected exactly one independent MediaRelease1 job")
    for setting in ("    name: " + JOB, "        working-directory: " + MOD_DIR, "          go-version: '1.26.5'"):
        require(lines.count(setting) == 1, "CI module, Go pin or job identity is missing")
    # Pin the executable suffix that was reviewed with the test inventory.
    # Matching command substrings alone cannot distinguish a mandatory step
    # from one made conditional, commented out or swallowed by shell control flow.
    require(hashlib.sha256(suffix).hexdigest() == "53a91b12acc1acc8382729b85102e4b327d78385f300177095964f01979346e8",
            "reviewed CI steps, conditions, commands or action pins changed")


def regular_path(repo: Path, relative: str) -> Path:
    path = repo
    for part in Path(relative).parts:
        path /= part
        require(not path.is_symlink() and not path.is_junction(), "link forbidden: " + relative)
    require(path.is_file() and stat.S_ISREG(path.stat().st_mode), "regular file required: " + relative)
    return path


def verify_pinned_paths(repo: Path, git) -> None:
    # Preserve everything in the last reviewed tree except the three files
    # explicitly extended for this family. Check both staged and unstaged
    # source, and include untracked paths in the same scope check.
    for args in (("diff", "--name-only", "--no-renames", "-z", BASELINE, "--"),
                 ("diff", "--cached", "--name-only", "--no-renames", "-z", BASELINE, "--"),
                 ("ls-files", "--others", "--exclude-standard", "-z")):
        for path in filter(None, git(repo, *args).decode().split("\0")):
            require(path in EXTENSION_PATHS, "pinned source change outside extension: " + path)
    for item in filter(None, git(repo, "ls-files", "--stage", "-z").decode().split("\0")):
        metadata, path = item.split("\t", 1)
        if path in EXTENSION_PATHS:
            # core.symlinks=false can check out a Git symlink as an ordinary
            # local file. Reject its index representation as well.
            mode, _, stage = metadata.split()
            require(mode == "100644" and stage == "0", "unexpected index mode/stage: " + path)


def check_legacy_tests(repo: Path, git) -> None:
    path = "scripts/ci/lifecycle_ci_test.py"
    old = ast.parse(git(repo, "show", BASELINE + ":" + path))
    current = ast.parse(regular_path(repo, path).read_bytes())
    for node in old.body:
        if isinstance(node, (ast.FunctionDef, ast.ClassDef)):
            same_name = [n for n in current.body if isinstance(n, type(node)) and n.name == node.name]
            require(len(same_name) == 1 and ast.dump(same_name[0]) == ast.dump(node),
                    "previous CI boundary/event tests changed: " + node.name)


def check_characterization_delta(relative: str, baseline_bytes: bytes, candidate_bytes: bytes) -> None:
    expected = REGISTRY_OUTPUT_HASHES.get(relative)
    require(expected is not None, "unreviewed registry edit")
    before = baseline_bytes.replace(b"\r\n", b"\n")
    after = candidate_bytes.replace(b"\r\n", b"\n")
    actual = (hashlib.sha256(before).hexdigest(), hashlib.sha256(after).hexdigest())
    require(actual == expected, "registry source differs from reviewed fifth-entry change: " + relative)


def check_entrypoints(repo: Path) -> None:
    discovered = sorted(p.relative_to(repo).as_posix() for p in (repo / "openapi").iterdir()
                        if p.suffix.lower() in (".yaml", ".yml"))
    require(discovered == sorted((*LEGACY_APIS, OPENAPI)), "unregistered or missing API entrypoint")


def check_characterization(repo: Path, git) -> dict:
    def pinned(path):
        return git(repo, "show", BASELINE + ":" + path)
    for path in CHARACTERIZATION:
        check_characterization_delta(path, pinned(path), regular_path(repo, path).read_bytes())
    ledger_path = "testdata/characterization/generated/openapi/source-inventory.json"
    ledger = pinned(ledger_path)
    require(regular_path(repo, ledger_path).read_bytes().replace(b"\r\n", b"\n") == ledger,
            "historical source inventory changed")
    require(json.loads(ledger)["entrypoints"] == list(LEGACY_APIS), "unexpected legacy source inventory")
    check_entrypoints(repo)
    hashes = {}
    for path in LEGACY_APIS:
        fixture = "testdata/characterization/generated/openapi/normalized/" + Path(path).stem + ".json"
        for artifact in (path, fixture):
            current = regular_path(repo, artifact).read_bytes().replace(b"\r\n", b"\n")
            require(current == pinned(artifact), "legacy OpenAPI source/normalized fixture changed: " + artifact)
            hashes[artifact] = hashlib.sha256(current).hexdigest()
    return hashes


def verify_module(repo: Path, check_source) -> list[str]:
    mod = repo / MOD_DIR
    files = set()
    require(mod.is_dir() and not mod.is_symlink() and not mod.is_junction(), "invalid module root")
    for path in mod.rglob("*"):
        relative = path.relative_to(mod).as_posix()
        require(not path.is_symlink() and not path.is_junction(), "link forbidden: " + relative)
        if path.is_dir():
            require(relative == "testdata", "unexpected module directory: " + relative)
        else:
            require(relative in MODULE_FILES, "unexpected module path: " + relative)
            files.add(relative)
    require(files == MODULE_FILES, "incomplete module inventory: " + ", ".join(sorted(MODULE_FILES - files)))
    for name in sorted(files):
        relative = MOD_DIR + "/" + name
        path = regular_path(repo, relative)
        check_source(path, relative)
    require((mod / "go.mod").read_text(encoding="utf-8") == f"module {MODULE}\n\ngo 1.26.5\n",
            "module identity, Go pin or dependency set differs")
    return sorted(MOD_DIR + "/" + path for path in files)


def read_object(pairs: list[tuple]) -> dict:
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate OpenAPI member")
        result[key] = value
    return result


def local_ref(schemas: dict, value: str) -> str:
    prefix = "#/components/schemas/"
    require(isinstance(value, str) and value.startswith(prefix) and value[len(prefix):] in schemas,
            "unresolved or external schema reference")
    return value[len(prefix):]


def walk_nodes(value):
    if isinstance(value, dict):
        yield value
        for item in value.values():
            yield from walk_nodes(item)
    elif isinstance(value, list):
        for item in value:
            yield from walk_nodes(item)


def check_openapi(path: Path) -> None:
    try:
        _check_openapi_document(path)
    except (KeyError, TypeError, AttributeError, IndexError, RecursionError) as exc:
        raise ValueError("invalid MediaRelease1 OpenAPI object structure") from exc


def _check_openapi_document(path: Path) -> None:
    def invalid_constant(_):
        raise ValueError("non-JSON numeric constant")
    # JSON is valid YAML 1.2; the published document uses this subset so that
    # the source boundary never installs or implicitly resolves a YAML parser.
    spec = json.loads(path.read_bytes().decode("utf-8"), object_pairs_hook=read_object,
                      parse_constant=invalid_constant)
    require(isinstance(spec, dict) and spec.get("openapi") == "3.1.0", "OpenAPI 3.1.0 required")
    require(spec.get("info", {}).get("version") == "1" and spec.get("x-autostream-contract") == TUPLE,
            "wrong MediaRelease1 wire/profile tuple")
    servers = spec.get("servers", [])
    require(len(servers) == 1 and servers[0].get("url") == "/v3/media-release/v1",
            "wrong API version namespace")
    schemas = spec["components"]["schemas"]
    require(isinstance(schemas, dict) and KINDS.keys() <= schemas.keys(), "missing record schemas")
    for name, literals in (
        ("ConsumptionState", {"NOT_CONSUMED", "CONSUMPTION_PENDING", "CONSUMED"}),
        ("EffectState", {"EFFECT_UNKNOWN", "RELEASE_CONFIRMED", "RELEASE_FAILED"}),
    ):
        enum = schemas[name]
        require(enum.get("type") == "string" and len(enum["enum"]) == len(literals)
                and set(enum["enum"]) == literals, "incorrect state axis: " + name)
    digest = schemas["Digest"]
    require(digest.get("type") == "string" and digest.get("pattern") == "^[0-9a-f]{64}$"
            and digest.get("minLength") == digest.get("maxLength") == 64, "digest constraints changed")
    require(spec.get("security") == [{"serviceToken": []}]
            and spec["components"]["securitySchemes"]["serviceToken"].get("type") == "http"
            and spec["components"]["securitySchemes"]["serviceToken"].get("scheme") == "bearer",
            "missing authenticated service principal")
    for node in walk_nodes(spec):
        if "$ref" in node:
            value = node["$ref"]
            require(isinstance(value, str), "invalid reference")
            parts = value.split("/")
            require(len(parts) >= 4 and parts[:2] == ["#", "components"] and parts[2] in ("schemas", "responses"),
                    "unresolved or external reference")
            target = spec
            for token in parts[1:]:
                require(re.search(r"~(?![01])", token) is None, "invalid JSON pointer escape")
                target = target[token.replace("~1", "/").replace("~0", "~")]
            require(isinstance(target, dict), "reference must identify an OpenAPI object")
        if "properties" in node:
            props = node["properties"]
            require(isinstance(props, dict), "schema properties must be an object")
            require(not any(k.casefold().replace("_", "") in SENSITIVE_NAMES for k in props),
                    "secret/raw schema field")
            required = node.get("required", [])
            require(isinstance(required, list) and len(set(required)) == len(required)
                    and set(required) <= props.keys(), "invalid required schema member")
    prefix = "#/components/schemas/"
    alternatives = schemas["Record"]["oneOf"]
    require(len(alternatives) == len(KINDS)
            and {s["$ref"] for s in alternatives} == {prefix + s for s in KINDS},
            "record union differs from six closed record kinds")
    for envelope, (cc, kind, purpose) in KINDS.items():
        schema = schemas[envelope]
        fields = {"identity", "snapshot", "immutable_digest", "record_digest"}
        require(schema.get("type") == "object" and schema.get("additionalProperties") is False
                and set(schema["properties"]) == fields and set(schema.get("required", [])) == fields,
                "incomplete or open envelope: " + envelope)
        snapshot = {"OneTimeConsumption": "ConsumptionSnapshot", "ReleaseReceipt": "ReceiptSnapshot"}.get(envelope, "SealedSnapshot")
        require(schema["properties"]["snapshot"] == {"$ref": prefix + snapshot}
                and all(schema["properties"][k] == {"$ref": prefix + "Digest"}
                        for k in ("immutable_digest", "record_digest")), "wrong snapshot/digest fields: " + envelope)
        identity = local_ref(schemas, schema["properties"]["identity"]["$ref"])
        require(identity == envelope + "Identity", "identity ref mismatch: " + envelope)
        ident_schema = schemas[identity]
        require(ident_schema.get("type") == "object" and ident_schema.get("additionalProperties") is False,
                "identity is open: " + identity)
        required = COMMON_IDENTITY | set(KIND_REQUIRED[envelope].split())
        conditional = {"asset_ref", "variant_id", "hidden_observation_ref"} if envelope == "AppliedCoverProof" else set()
        require(set(ident_schema["required"]) == required and set(ident_schema["properties"]) == required | conditional,
                "incomplete identity field inventory: " + identity)
        discriminators = dict(TUPLE, cc=cc, kind=kind, purpose=purpose)
        for field, literal in discriminators.items():
            require(field in ident_schema["required"] and ident_schema["properties"][field].get("const") == literal,
                    "incorrect identity discriminator: " + identity + "." + field)
    paths = spec["paths"]
    require(paths.keys() == OPERATIONS.keys(), "incorrect endpoint set")
    for relative, (method, operation_id) in OPERATIONS.items():
        require(set(paths[relative]) == {method}, "unexpected HTTP method for " + relative)
        op = paths[relative][method]
        require(op.get("operationId") == operation_id, "incorrect operationId: " + relative)
        require("servers" not in op, "operation changed version namespace")
        require("security" not in op or op["security"] == spec["security"], "operation changed authentication")
        req, result = OPERATION_SCHEMAS[relative]
        def content(name):
            return {"application/json": {"schema": {"$ref": prefix + name}}}
        require(op.get("responses", {}).get("200", {}).get("content") == content(result),
                "missing successful response schema: " + relative)
        if req is not None:
            request = op.get("requestBody", {})
            require(request.get("required") is True and request.get("content") == content(req),
                    "missing required request body: " + relative)
        else:
            require("requestBody" not in op, "capability query cannot take a write body")
