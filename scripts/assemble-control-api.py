#!/usr/bin/env python3
"""Compatibility CLI for the Control API's byte-preserving authoring assembly."""
import argparse
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts/ci"))
from artifact_assembly import Source, assemble as assemble_artifacts

SOURCE = ROOT
OUTPUT = ROOT / "openapi/control-api.yaml"


def assemble(source: Path = SOURCE) -> bytes:
    return assemble_artifacts(Source(source), "Autostream-Contracts")["openapi/control-api.yaml"]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    data = assemble()
    if args.check:
        if OUTPUT.read_bytes() != data:
            parser.error("control-api.yaml differs from authoring; regenerate it")
        print("Control API assembly matches the public single-file contract")
    else:
        OUTPUT.write_bytes(data)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
