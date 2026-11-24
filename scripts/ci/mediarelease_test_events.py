"""Verify native MediaRelease1 Go test events against the reviewed inventory."""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from lifecycle_test_events import read_events

MODULE = "github.com/Kome-Lab/Autostream-Contracts/v3mediarelease"
REQUIRED = (
    "TestAdmissionAuthorityVector",
    "TestWorkerBindingVector",
    "TestAppliedCoverVector",
    "TestTargetPermitVector",
    "TestConsumptionVectors",
    "TestReleaseReceiptVector",
    "TestCanonicalWireVectors",
    "TestImmutableEqualityVectors",
    "TestWireFormatRejections",
    "TestUnknownVersionAndFamily",
    "TestSameAttemptHandoff",
    "TestPhysicalOwnerCorrelation",
    "TestWorkerGenerationCorrelation",
    "TestRevisionCASVectors",
    "TestReleaseWindowBoundaries",
    "TestRevocationAndStopFence",
    "TestDuplicateReleaseRefused",
    "TestRestartAfterConsumption",
    "TestUnknownEffectAndPartialWrite",
    "TestOriginalReceiptInspection",
    "TestNegotiatedFamily",
    "TestMixedAndOldConsumers",
    "TestSyntheticFactsNeverAuthorize",
    "TestMediaReleaseProductionSourceBoundary",
)
REQUIRED_CHILDREN = (
    "TestReleaseWindowBoundaries/AtExpiry",
    "TestRevocationAndStopFence/CPRevoked",
    "TestRevocationAndStopFence/OwnerStopped",
    "TestDuplicateReleaseRefused/NewIDsSamePhysicalSlot",
    "TestDuplicateReleaseRefused/OverlappingTracks",
    "TestRestartAfterConsumption/LostCommitACK",
    "TestRestartAfterConsumption/RolledBackClaim",
    "TestUnknownEffectAndPartialWrite/PartialWrite",
    "TestUnknownEffectAndPartialWrite/RemoteTerminal",
    "TestOriginalReceiptInspection/UnknownStore",
    "TestOriginalReceiptInspection/OriginalKeyAfterACKLoss",
    "TestMixedAndOldConsumers/OldFamily",
    "TestMixedAndOldConsumers/FutureOnly",
    "TestMixedAndOldConsumers/MissingInspect",
    "TestWireFormatRejections/DuplicateMember",
    "TestWireFormatRejections/UnsortedSet",
)


def verify_events(events: list[dict], inventory: list[str]) -> dict:
    if inventory != list(REQUIRED):
        raise ValueError("required root inventory differs from the reviewed set")
    started, finished = False, False
    active: set[str] = set()
    passed: set[str] = set()
    for index, event in enumerate(events, 1):
        if not isinstance(event, dict) or event.get("Package") != MODULE:
            raise ValueError(f"event {index}: unexpected package")
        action = event.get("Action")
        test = event.get("Test")
        if test is not None and (not isinstance(test, str) or not test):
            raise ValueError(f"event {index}: invalid test name")
        if action == "start" and test is None:
            if started:
                raise ValueError("duplicate package start")
            started = True
        elif not started or finished:
            raise ValueError(f"event {index}: outside package execution")
        elif action == "run" and test is not None:
            parent = test.rpartition("/")[0]
            if test in active or test in passed or (parent and parent not in active):
                raise ValueError(f"event {index}: unexpected test run")
            active.add(test)
        elif action == "pass" and test is not None:
            if test not in active or any(name.startswith(test + "/") for name in active):
                raise ValueError(f"event {index}: unexpected test pass")
            active.remove(test)
            passed.add(test)
        elif action == "pass" and test is None:
            if active:
                raise ValueError("package finished with incomplete tests")
            finished = True
        elif action == "output":
            continue
        elif action in ("pause", "cont") and test in active:
            continue
        else:
            raise ValueError(f"event {index}: fail, skip, or unexpected event")
    missing = (set(REQUIRED) | set(REQUIRED_CHILDREN)) - passed
    if missing or not finished:
        raise ValueError("incomplete suite: " + ", ".join(sorted(missing)))
    return {"status": "PASS", "module": MODULE, "required_roots": len(REQUIRED),
            "required_children": len(REQUIRED_CHILDREN),
            "total_roots": sum("/" not in name for name in passed),
            "total_children": sum("/" in name for name in passed),
            "test_passes_including_subtests": len(passed), "skip": 0, "fail": 0}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("events", type=Path)
    parser.add_argument("inventory", type=Path)
    args = parser.parse_args()
    try:
        print(json.dumps(verify_events(read_events(args.events),
                                      args.inventory.read_text(encoding="utf-8").splitlines())))
    except (OSError, ValueError) as error:
        print(f"media release test inventory: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
