"""Check the independent pure module and preserve every pre-IU-CC-01 blob.

This is a source gate, not runtime authority, provider qualification, or a secret
manager. Structural DTO validation remains the Go package's responsibility.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

BASELINE = "45aaa9932bcd92c728c86c95349853724fe1accc"
MODULE = "github.com/Kome-Lab/Autostream-Contracts/v3lifecycle"
WORKFLOW = ".github/workflows/v3-evidence-ci.yml"
SCRIPTS = frozenset((
    "scripts/ci/lifecycle-required-tests.txt",
    "scripts/ci/lifecycle_test_events.py",
    "scripts/ci/lifecycle_source_boundary.py",
    "scripts/ci/lifecycle_ci_test.py",
))
PURE_IMPORTS = frozenset((
    "bytes", "encoding/json", "errors", "fmt", "io", "math", "reflect",
    "slices", "sort", "strconv", "strings", "unicode", "unicode/utf8",
))
SECRET_PATTERNS = (
    re.compile(r"-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,})"),
    re.compile(r"(?:AKIA|ASIA)[A-Z0-9]{16}"),
    re.compile(r"xox[baprs]-[0-9A-Za-z-]{20,}"),
    re.compile(r"[A-Za-z]:[\\/](?:Users|home)[\\/][^\s\"'<>]+", re.IGNORECASE),
)
FORBIDDEN_FIELDS = frozenset((
    "Password", "PasswordHash", "MFASecret", "RecoveryCode", "RecoveryHash",
    "SessionToken", "CSRFToken", "OAuthSecret", "PrivateKey", "EncryptionKey",
    "RawPayload", "RawAudio", "NotificationBody", "NotificationURL",
))
# Capture the whole identifier list, including comma-separated declarations.
FIELD_DECLARATION = re.compile(
    r"(?m)(?:^|[;{])\s*([A-Za-z_]\w*(?:\s*,\s*[A-Za-z_]\w*)*)\s+(?:\*?\w|\[)"
)
GO_NONCODE = re.compile(
    r'(?P<string>"(?:\\.|[^"\\])*"|\x60[^\x60]*\x60|\'(?:\\.|[^\'\\])*\')'
    r'|(?P<line>//[^\r\n]*)|(?P<block>/\*.*?\*/)',
    re.DOTALL,
)
AUTHORITY_FUNCTION = re.compile(
    r"\bfunc\s+(?:\([^)]*\)\s+)?(?:NewPermit|NewAuthority|IssuePermit|"
    r"IssueMutation|IssueRecovery|MintPermit|MintAuthority|ConsumeApproval|"
    r"QualifySeal|QualifyProduction)\s*\("
)


def git(repo: Path, *args: str) -> bytes:
    result = subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True)
    return result.stdout


def allowed_change(path: str) -> bool:
    return path == WORKFLOW or path in SCRIPTS or path.startswith("v3lifecycle/")


def mask_go_noncode(text: str, preserve_strings: bool) -> str:
    def replace(match):
        if preserve_strings and match.group("string") is not None:
            return match.group()
        return re.sub(r"[^\r\n]", " ", match.group())
    return GO_NONCODE.sub(replace, text)


def production_imports(text: str) -> list[str]:
    # Locate keywords outside comments/strings, then read their actual import
    # declarations. This also handles aliases, inline blocks and comments that
    # contain ')' without silently dropping a dependency.
    code = mask_go_noncode(text, preserve_strings=False)
    comments_removed = mask_go_noncode(text, preserve_strings=True)
    imports = []
    for keyword in re.finditer(r"\bimport\b", code):
        start = keyword.end()
        while start < len(text) and comments_removed[start].isspace():
            start += 1
        if start >= len(text):
            raise ValueError("incomplete production import")
        if code[start] == "(":
            end = code.find(")", start + 1)
            if end == -1:
                raise ValueError("incomplete production import block")
            declaration = comments_removed[start + 1:end]
        else:
            end = comments_removed.find("\n", start)
            declaration = comments_removed[start:] if end == -1 else comments_removed[start:end]
        # Raw/escaped path spellings are unnecessary in this independent module.
        # Refuse unsupported forms rather than accepting an unexamined import.
        if chr(96) in declaration:
            raise ValueError("raw-string production imports are not supported")
        quoted = re.findall(r'"(?:\\.|[^"\\])*"', declaration)
        if not quoted:
            raise ValueError("unrecognized production import")
        imports.extend(value[1:-1] for value in quoted)
    return imports


def check_source(path: Path, relative: str) -> None:
    text = path.read_text(encoding="utf-8")
    for pattern in SECRET_PATTERNS:
        if pattern.search(text):
            raise ValueError(f"sensitive material or personal path pattern: {relative}")
    if path.suffix != ".go" or path.name.endswith("_test.go"):
        return
    # The Go AST conformance root independently checks every production package.
    # This lightweight pre-test gate is conservative about import spellings.
    imports = production_imports(text)
    for imported in imports:
        if imported not in PURE_IMPORTS:
            raise ValueError(f"non-pure production import {imported}: {relative}")
    code = mask_go_noncode(text, preserve_strings=False)
    if re.search(r"\bfunc\s+(?:main|init)\s*\(", code):
        raise ValueError(f"implicit/executable entry point: {relative}")
    if AUTHORITY_FUNCTION.search(code):
        raise ValueError(f"authority constructor/issuer: {relative}")
    for declaration in FIELD_DECLARATION.finditer(code):
        names = (name.strip() for name in declaration.group(1).split(","))
        if any(name in FORBIDDEN_FIELDS for name in names):
            raise ValueError(f"secret/raw payload DTO field: {relative}")


def verify(repo: Path) -> dict:
    # This compares Git's canonical working/index contents, respecting checkout
    # EOL rules. It does not rewrite or stage existing files.
    changed = git(repo, "diff", "--name-only", "-z", BASELINE, "--").decode().split("\0")
    untracked = git(repo, "ls-files", "--others", "--exclude-standard", "-z").decode().split("\0")
    for relative in filter(None, changed + untracked):
        if not allowed_change(relative):
            raise ValueError(f"out-of-scope change: {relative}")
    baseline_workflow = git(repo, "show", BASELINE + ":" + WORKFLOW)
    workflow = (repo / WORKFLOW).read_bytes().replace(b"\r\n", b"\n")
    if not workflow.startswith(baseline_workflow):
        raise ValueError("existing workflow/job/steps/pins were altered")
    if workflow[len(baseline_workflow):].count(b"  v3-lifecycle-contract-checks:") != 1:
        raise ValueError("exactly one appended lifecycle job required")
    # All baseline files except the append-only workflow must remain identical.
    old_changes = [p for p in filter(None, changed) if p != WORKFLOW and
                   git(repo, "ls-tree", BASELINE, "--", p)]
    if old_changes:
        raise ValueError("baseline source/module/API/pin changes: " + ", ".join(old_changes))
    module_dir = repo / "v3lifecycle"
    expected_mod = f"module {MODULE}\n\ngo 1.26.5\n"
    if (module_dir / "go.mod").read_text(encoding="utf-8") != expected_mod:
        raise ValueError("exact independent module/Go version and zero dependencies required")
    if (module_dir / "go.sum").exists() or (module_dir / "go.work").exists():
        raise ValueError("independent stdlib module must not add dependency/workspace files")
    checked = []
    for path in sorted(module_dir.rglob("*")):
        relative = path.relative_to(repo).as_posix()
        if path.is_symlink():
            raise ValueError(f"symlink forbidden: {relative}")
        if not path.is_file():
            continue
        if path.name != "go.mod" and path.suffix not in (".go", ".md", ".json", ".txt"):
            raise ValueError(f"generated/unexpected file: {relative}")
        check_source(path, relative)
        checked.append(relative)
    if not any(p.endswith(".go") and not p.endswith("_test.go") for p in checked):
        raise ValueError("production lifecycle source is missing")
    check_source(repo / WORKFLOW, WORKFLOW)
    for relative in sorted(SCRIPTS):
        check_source(repo / relative, relative)
    modules = []
    for path in sorted(repo.rglob("go.mod")):
        if ".git" in path.parts:
            continue
        modules.append({
            "path": path.relative_to(repo).as_posix(),
            "sha256": hashlib.sha256(path.read_bytes().replace(b"\r\n", b"\n")).hexdigest(),
            "validation_job": "v3-lifecycle-contract-checks" if path.parent == module_dir else
                              "v3-evidence-checks" if path.parent.name == "v3evidence" else "v3-lifecycle-contract-checks (unchanged root regression)",
        })
    return {
        "status": "PASS", "gate": "SOURCE_BOUNDARY", "baseline": BASELINE,
        "candidate_commit": git(repo, "rev-parse", "HEAD").decode().strip(),
        "candidate_tree": git(repo, "rev-parse", "HEAD^{tree}").decode().strip(),
        "baseline_source_api_enums_module_pins": "UNCHANGED",
        "existing_workflow": "UNCHANGED_PREFIX", "pure_import_allowlist": sorted(PURE_IMPORTS),
        "checked_files": checked, "modules": modules, "production_qualification": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args()
    try:
        summary = verify(args.repo.resolve())
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"lifecycle source boundary: {error}", file=sys.stderr)
        return 1
    print(json.dumps(summary, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
