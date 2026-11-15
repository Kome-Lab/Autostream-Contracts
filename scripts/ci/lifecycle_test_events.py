"""Fail closed unless every G0-V-01..24 root really ran and passed once.

This reads native go test -json events, never cross-build output. Expected test
names and package are reviewed literals, not candidate-generated expectations.
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path

MODULE = "github.com/Kome-Lab/Autostream-Contracts/v3lifecycle"
REQUIRED = (
    "TestContractRegisterClosed",
    "TestMembershipFreezeAndAliasCoverage",
    "TestRevisionVectorOwnerResourceAndIncarnation",
    "TestAdmissionCutAndLineageModel",
    "TestCompletionFrontierAndPhysicalObligations",
    "TestReadySealPriorWorkOnly",
    "TestFenceObservationIsNotAuthority",
    "TestEffectBoundaryOutcomeModel",
    "TestUnknownPreservesKnownBoundaries",
    "TestPhaseHintDoesNotProveCommit",
    "TestInspectionIsObservationOnly",
    "TestRecordOnlyReconciliationCASModel",
    "TestRecordTransferIsNotEffectReplay",
    "TestRestoreFreshnessIsNotEquality",
    "TestTimeoutCancellationDoNotReopen",
    "TestLegacyProjectionCannotMintEvidence",
    "TestIdentityClassAndPrincipalKindRemainSeparate",
    "TestFixtureCannotQualifyProduction",
    "TestProducerConsumerProfileConformance",
    "TestExplicitLimitsAndMissingValues",
    "TestSourceBoundaryNoAuthorityConstructors",
    "TestBoundarySubkindRegistryClosed",
    "TestImmutableAttemptCorrelation",
    "TestFanoutOutcomeObligations",
)
ACTIONS = frozenset(("start", "run", "pause", "cont", "pass", "bench", "fail", "output", "skip"))


def verify_events(events: list[dict], inventory: list[str]) -> dict:
    if inventory != list(REQUIRED):
        raise ValueError("required inventory must match all 24 reviewed roots in order")
    runs: Counter[str] = Counter()
    passes: Counter[str] = Counter()
    package_passes: Counter[str] = Counter()
    packages: set[str] = set()
    test_passes = 0
    for index, event in enumerate(events, 1):
        if not isinstance(event, dict):
            raise ValueError(f"event {index}: object required")
        action, package, test = event.get("Action"), event.get("Package"), event.get("Test")
        if action not in ACTIONS:
            raise ValueError(f"event {index}: unrecognized action")
        if not isinstance(package, str) or not (package == MODULE or package.startswith(MODULE + "/")):
            raise ValueError(f"event {index}: unexpected module")
        packages.add(package)
        if action in ("skip", "fail"):
            raise ValueError(f"event {index}: {action} is forbidden ({test or package})")
        if test is not None and (not isinstance(test, str) or not test):
            raise ValueError(f"event {index}: invalid test name")
        if action == "pass" and test is None:
            package_passes[package] += 1
        if action == "pass" and test is not None:
            test_passes += 1
        if package == MODULE and test in REQUIRED:
            if action == "run":
                runs[test] += 1
                if runs[test] != 1 or passes[test]:
                    raise ValueError(f"duplicate/out-of-order root run: {test}")
            elif action == "pass":
                passes[test] += 1
                if runs[test] != 1 or passes[test] != 1:
                    raise ValueError(f"root pass without exactly one earlier run: {test}")
    for test in REQUIRED:
        if runs[test] != 1 or passes[test] != 1:
            raise ValueError(f"required root did not run/pass exactly once: {test}")
    if not packages or any(package_passes[p] != 1 for p in packages):
        raise ValueError("every observed package must finish with exactly one package pass")
    return {
        "status": "PASS", "module": MODULE, "execution": "NATIVE",
        "required_roots": len(REQUIRED), "test_passes_including_subtests": test_passes,
        "skip": 0, "fail": 0,
        "roots": [{"id": f"G0-V-{i:02d}", "test": test, "run": runs[test], "pass": passes[test]}
                  for i, test in enumerate(REQUIRED, 1)],
    }


def unique_object(pairs: list[tuple]) -> dict:
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate JSON event field")
        value[key] = item
    return value


def read_events(path: Path) -> list[dict]:
    events = []
    for line_number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            raise ValueError(f"event line {line_number}: blank event")
        try:
            events.append(json.loads(line, object_pairs_hook=unique_object))
        except (json.JSONDecodeError, ValueError) as error:
            raise ValueError(f"event line {line_number}: malformed event") from error
    return events


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("events", type=Path)
    parser.add_argument("inventory", type=Path)
    args = parser.parse_args()
    try:
        inventory = args.inventory.read_text(encoding="utf-8").splitlines()
        summary = verify_events(read_events(args.events), inventory)
    except (OSError, ValueError) as error:
        print(f"lifecycle required-test gate: {error}", file=sys.stderr)
        return 1
    print(json.dumps(summary, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
