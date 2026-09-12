"""Negative checks for the production assembler's exact authoring inventory."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location(
    "control_api_assembly", Path(__file__).with_name("assemble-control-api.py")
)
ASSEMBLY = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ASSEMBLY)


class ControlAPIAssemblyTest(unittest.TestCase):
    def test_public_bytes_are_reproducible(self):
        original = ASSEMBLY.OUTPUT.read_bytes()
        self.assertEqual(ASSEMBLY.assemble(), original)
        self.assertEqual(ASSEMBLY.assemble(), ASSEMBLY.assemble())

    def test_missing_duplicate_unlisted_and_escaping_sources_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            (source / "authoring").mkdir()
            (source / "authoring/domain.yaml.inc").write_bytes(b"paths:\n")
            (source / "authoring/schema.json.inc").write_bytes(b"{}\n")
            manifest = source / "authoring/artifact-layout.json"
            def write_manifest(names):
                manifest.write_text(json.dumps({
                    "format_version": 1, "repository": "Autostream-Contracts",
                    "artifacts": [
                        {"output": "openapi/control-api.yaml", "fragments": names},
                        {"output": "schemas/discord-bot-start-job-request.schema.json",
                         "fragments": ["authoring/schema.json.inc"]},
                    ],
                }))
            for names in ([], ["absent.yaml"],
                          ["domain.yaml", "domain.yaml"],
                          ["../domain.yaml"]):
                with self.subTest(fragments=names):
                    write_manifest(["authoring/" + name + ".inc" for name in names])
                    with self.assertRaises(ValueError):
                        ASSEMBLY.assemble(source)
            write_manifest(["authoring/domain.yaml.inc"])
            (source / "authoring/unlisted.yaml.inc").write_bytes(b"info:\n")
            with self.assertRaises(ValueError):
                ASSEMBLY.assemble(source)


if __name__ == "__main__":
    unittest.main()
