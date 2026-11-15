"""Negative tests for the mandatory lifecycle Go JSON event gate."""
import importlib.util
import unittest
from pathlib import Path

SPEC = importlib.util.spec_from_file_location("lifecycle_test_events", Path(__file__).with_name("lifecycle_test_events.py"))
CHECK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECK)


def complete_events():
    events = [{"Action": "start", "Package": CHECK.MODULE}]
    for root in CHECK.REQUIRED:
        events.extend([
            {"Action": "run", "Package": CHECK.MODULE, "Test": root},
            {"Action": "pass", "Package": CHECK.MODULE, "Test": root},
        ])
    events.append({"Action": "pass", "Package": CHECK.MODULE})
    return events


class LifecycleEventGateTests(unittest.TestCase):
    def test_complete_inventory_passes(self):
        summary = CHECK.verify_events(complete_events(), list(CHECK.REQUIRED))
        self.assertEqual(summary["status"], "PASS")
        self.assertEqual(summary["required_roots"], 24)
        self.assertEqual(summary["skip"], 0)
        self.assertEqual(summary["fail"], 0)

    def test_missing_root_fails(self):
        events = [e for e in complete_events() if e.get("Test") != CHECK.REQUIRED[0]]
        with self.assertRaises(ValueError):
            CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_skip_any_subtest_fails(self):
        events = complete_events()
        events.insert(-1, {"Action": "skip", "Package": CHECK.MODULE, "Test": CHECK.REQUIRED[0] + "/negative"})
        with self.assertRaises(ValueError):
            CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_failure_any_package_fails(self):
        events = complete_events()
        events.insert(-1, {"Action": "fail", "Package": CHECK.MODULE + "/consumer", "Test": "TestConsumer"})
        with self.assertRaises(ValueError):
            CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_duplicate_root_fails(self):
        for action in ("run", "pass"):
            with self.subTest(action=action):
                events = complete_events()
                events.insert(-1, {"Action": action, "Package": CHECK.MODULE, "Test": CHECK.REQUIRED[0]})
                with self.assertRaises(ValueError):
                    CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_wrong_module_cannot_supply_root(self):
        events = complete_events()
        events[1]["Package"] = "github.com/Kome-Lab/Autostream-Contracts/v3evidence"
        with self.assertRaises(ValueError):
            CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_root_pass_requires_earlier_run(self):
        events = complete_events()
        events[1], events[2] = events[2], events[1]
        with self.assertRaises(ValueError):
            CHECK.verify_events(events, list(CHECK.REQUIRED))

    def test_required_inventory_cannot_be_reduced(self):
        with self.assertRaises(ValueError):
            CHECK.verify_events(complete_events(), list(CHECK.REQUIRED)[1:])

    def test_package_end_is_required(self):
        with self.assertRaises(ValueError):
            CHECK.verify_events(complete_events()[:-1], list(CHECK.REQUIRED))

    def test_empty_inventory_and_events_fail(self):
        with self.assertRaises(ValueError):
            CHECK.verify_events([], [])

BOUNDARY_SPEC = importlib.util.spec_from_file_location(
    "lifecycle_source_boundary", Path(__file__).with_name("lifecycle_source_boundary.py"))
BOUNDARY = importlib.util.module_from_spec(BOUNDARY_SPEC)
BOUNDARY_SPEC.loader.exec_module(BOUNDARY)


class LifecycleSourceGateTests(unittest.TestCase):
    def assert_source(self, source, valid):
        import tempfile
        with tempfile.TemporaryDirectory(prefix="lifecycle-source-gate-") as directory:
            candidate = Path(directory) / "candidate.go"
            candidate.write_text(source, encoding="utf-8")
            if valid:
                BOUNDARY.check_source(candidate, "v3lifecycle/candidate.go")
            else:
                with self.assertRaises(ValueError):
                    BOUNDARY.check_source(candidate, "v3lifecycle/candidate.go")

    def test_pure_validator_allowed(self):
        self.assert_source('package lifecycle\nimport "strings"\nfunc Valid(s string) bool { return strings.Contains(s,"x") }\n', True)

    def test_provider_database_environment_clock_imports_forbidden(self):
        for imported in ("net/http", "database/sql", "os", "time", "unsafe", "C", "example.test/provider"):
            with self.subTest(imported=imported):
                self.assert_source(f'package lifecycle\nimport "{imported}"\n', False)

    def test_multiline_import_alias_is_checked(self):
        self.assert_source('package lifecycle\nimport (\n backend "database/sql"\n)\n', False)

    def test_secret_and_raw_dto_fields_forbidden(self):
        for field in ("Password", "SessionToken", "MFASecret", "RecoveryCode", "PrivateKey", "RawPayload", "RawAudio", "NotificationBody"):
            with self.subTest(field=field):
                self.assert_source(f"package lifecycle\ntype Value struct {{\n {field} string\n}}\n", False)

    def test_inline_secret_field_forbidden(self):
        self.assert_source("package lifecycle\ntype Value struct { RawPayload []byte }\n", False)

    def test_authority_issuer_forbidden(self):
        for name in ("NewPermit", "MintAuthority", "ConsumeApproval", "IssueRecovery", "QualifySeal"):
            with self.subTest(name=name):
                self.assert_source(f"package lifecycle\nfunc {name}() {{}}\n", False)

    def test_implicit_entry_points_forbidden(self):
        for name in ("init", "main"):
            with self.subTest(name=name):
                self.assert_source(f"package lifecycle\nfunc {name}() {{}}\n", False)

    def test_unknown_repo_changes_forbidden(self):
        self.assertTrue(BOUNDARY.allowed_change("v3lifecycle/validate.go"))
        self.assertTrue(BOUNDARY.allowed_change(".github/workflows/v3-evidence-ci.yml"))
        for path in ("go.mod", "v3evidence/request.go", "scripts/ci/request_api_compat.sh", "internal/production.go"):
            with self.subTest(path=path):
                self.assertFalse(BOUNDARY.allowed_change(path))

    def test_raw_string_imports_forbidden(self):
        quote = chr(96)
        for statement in (f"import {quote}os{quote}", f"import _ {quote}os{quote}",
                          f"import . {quote}os{quote}", f"import (\n backend {quote}os{quote}\n)",
                          f"import ({quote}os{quote})"):
            with self.subTest(statement=statement):
                self.assert_source(f"package lifecycle\n{statement}\n", False)

    def test_comma_declared_secret_fields_forbidden(self):
        for declaration in ("RawPayload, Other []byte", "Other, Password string",
                            "Other, RawAudio, Last []byte", "Other,\n SessionToken string"):
            with self.subTest(declaration=declaration):
                self.assert_source(f"package lifecycle\ntype Value struct {{\n {declaration}\n}}\n", False)

    def test_import_comments_do_not_hide_dependency(self):
        quote = chr(96)
        self.assert_source(f"package lifecycle\nimport (\n /* ) */\n . {quote}os{quote}\n)\n", False)

    def test_comments_and_string_documentation_are_not_fields(self):
        quote = chr(96)
        self.assert_source(f"package lifecycle\n/*\nRawPayload []byte\n*/\nconst Description = {quote}\nPassword string\n{quote}\ntype Value struct {{ Ref string }}\n", True)

    def test_import_words_in_strings_and_comments_are_ignored(self):
        quote = chr(96)
        self.assert_source(f'package lifecycle\n// import "os"\nconst Description = {quote}\nimport "net/http"\n{quote}\n', True)

    def test_escaped_import_spelling_fails_closed(self):
        self.assert_source('package lifecycle\nimport "o\\x73"\n', False)

    def test_pure_alias_and_inline_imports_allowed(self):
        for statement in ('import . "fmt"', 'import alias "strings"', 'import ("bytes"; "encoding/json")'):
            with self.subTest(statement=statement):
                self.assert_source(f"package lifecycle\n{statement}\n", True)


if __name__ == "__main__":
    unittest.main()
