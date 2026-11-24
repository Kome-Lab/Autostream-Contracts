"""Negative tests for the mandatory lifecycle Go JSON event gate."""
import importlib.util
import json
import shutil
import tempfile
import unittest
from pathlib import Path
from unittest import mock

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


REPO = Path(__file__).resolve().parents[2]
MEDIA_DIR = "v3mediarelease"
MEDIA_MODULE = "github.com/Kome-Lab/Autostream-Contracts/v3mediarelease"
MEDIA_OPENAPI = "openapi/media-release-v1.yaml"
MEDIA_JOB = "v3-media-release-contract-checks"


class MediaReleaseSourceGateTests(unittest.TestCase):
    def test_only_registered_media_paths_are_in_scope(self):
        for path in ("v3mediarelease/types.go", "v3mediarelease/go.mod", MEDIA_OPENAPI):
            with self.subTest(path=path):
                self.assertTrue(BOUNDARY.allowed_change(path))
        for path in ("v3mediarelease2/types.go", "v3mediarelease-old/types.go",
                     "V3MediaRelease/types.go", "v3mediarelease/../v3evidence/request.go",
                     "openapi/media-release-v1.yaml.bak", "openapi/media-release-v1.yml",
                     "openapi/other.yaml", "scripts/ci/mediarelease_misc.py"):
            with self.subTest(path=path):
                self.assertFalse(BOUNDARY.allowed_change(path))

    def check_go(self, text):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "validate.go"
            path.write_text(text, encoding="utf-8")
            BOUNDARY.check_source(path, "v3mediarelease/validate.go")

    def test_media_validator_uses_only_allowed_imports(self):
        self.check_go('package mediarelease\nimport "crypto/sha256"\n')
        self.check_go('package mediarelease\nimport "time"\nvar parse = time.Parse\n')
        for name in ("net/http", "os/exec", "database/sql", "unsafe", "C", "example.test/remote"):
            with self.subTest(name=name), self.assertRaises(ValueError):
                self.check_go(f'package mediarelease\nimport (\n remote "{name}"\n)\n')

    def test_media_predicates_cannot_read_wall_clock(self):
        with self.assertRaises(ValueError):
            self.check_go('package mediarelease\nimport "time"\nfunc sample() time.Time { return time.Now() }\n')

    def test_media_source_rejects_authority_and_executable_entrypoints(self):
        for declaration in ("func IssuePermit() {}", "func (x Model) MintAuthority() {}",
                            "func main() {}", "func init() {}"):
            with self.subTest(declaration=declaration), self.assertRaises(ValueError):
                self.check_go("package mediarelease\n" + declaration)

    def test_media_source_rejects_secret_and_payload_fields(self):
        for name in ("RawPayload", "RawAudio", "SessionToken", "PrivateKey"):
            with self.subTest(field=name), self.assertRaises(ValueError):
                self.check_go("package mediarelease\ntype Input struct { Explicit, " + name + " []byte }\n")

    def test_media_openapi_examples_are_scanned_for_private_content(self):
        for example in ("ghp_" + "Q" * 36, "Z:" + "/Users/" + "fixture/private.key"):
            with self.subTest(example_type=example[:1]), tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "media-release-v1.yaml"
                path.write_text(json.dumps({"example": example}), encoding="utf-8")
                with self.assertRaises(ValueError):
                    BOUNDARY.check_source(path, MEDIA_OPENAPI)

    def test_new_module_manifest_and_dependencies(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            module = repo / MEDIA_DIR
            shutil.copytree(REPO / MEDIA_DIR, module)
            manifest = module / "go.mod"
            original = manifest.read_text(encoding="utf-8")
            BOUNDARY.mediarelease.verify_module(repo, BOUNDARY.check_source)
            for text in (original.replace(MEDIA_MODULE, MEDIA_MODULE + "/unexpected"),
                         original.replace("1.26.5", "1.25.5"),
                         original + "\nrequire example.test/dependency v1.0.0\n",
                         original + "\nreplace example.test/unreviewed => ../external\n"):
                with self.subTest(text=text):
                    self.assertNotEqual(original, text)
                    manifest.write_text(text, encoding="utf-8")
                    with self.assertRaises(ValueError):
                        BOUNDARY.mediarelease.verify_module(repo, BOUNDARY.check_source)
            manifest.write_text(original, encoding="utf-8")
            for unexpected in ("go.sum", "go.work", "payload.exe", "nested/go.mod"):
                with self.subTest(unexpected=unexpected):
                    extra = module / unexpected
                    extra.parent.mkdir(parents=True, exist_ok=True)
                    extra.write_bytes(b"")
                    with self.assertRaises(ValueError):
                        BOUNDARY.mediarelease.verify_module(repo, BOUNDARY.check_source)
                    extra.unlink()

    def test_media_module_refuses_link_even_with_allowed_name(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            module = repo / MEDIA_DIR
            shutil.copytree(REPO / MEDIA_DIR, module)
            source = module / "types.go"
            target = repo / "outside.go"
            source.rename(target)
            try:
                source.symlink_to(target)
            except OSError:
                # Some Windows runners lack link creation privileges. Exercise
                # the same detector without weakening the rejection assertion.
                source.write_bytes(target.read_bytes())
                original = Path.is_symlink
                with mock.patch.object(Path, "is_symlink", lambda p: p == source or original(p)):
                    with self.assertRaises(ValueError):
                        BOUNDARY.mediarelease.verify_module(repo, BOUNDARY.check_source)
            else:
                with self.assertRaises(ValueError):
                    BOUNDARY.mediarelease.verify_module(repo, BOUNDARY.check_source)

    def test_pinned_source_rejects_staged_or_worktree_drift_and_git_links(self):
        def check_records(changed=b"", staged=b"", index=b""):
            def read_git(_repo, *args):
                if args[0] == "diff":
                    return staged if "--cached" in args else changed
                if args[0] == "ls-files":
                    return index if "--stage" in args else b""
                self.fail(f"unexpected preservation query: {args}")
            BOUNDARY.mediarelease.verify_pinned_paths(REPO, read_git)

        target = b"v3mediarelease/types.go"
        digest = b"0" * 40
        check_records(changed=target + b"\0", index=b"100644 " + digest + b" 0\t" + target + b"\0")
        for change in ({"changed": b"v3lifecycle/types.go\0"},
                       {"staged": b"v3evidence/request.go\0"},
                       {"staged": b"testdata/characterization/generated/openapi/fingerprints.json\0"},
                       {"changed": b"testdata/characterization/generated/openapi/source-inventory.json\0"},
                       {"changed": b"scripts/characterization/openapi-model.mjs\0"},
                       {"changed": b"pkg/contracts/other_characterization_test.go\0"},
                       {"changed": b"openapi/observability-api.yaml\0"},
                       {"index": b"120000 " + digest + b" 0\t" + target + b"\0"},
                       {"index": b"100644 " + digest + b" 2\t" + target + b"\0"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                check_records(**change)

    def test_old_and_new_modules_keep_their_validation_jobs(self):
        inventory = BOUNDARY.verify(REPO)["modules"]
        jobs = {r["path"]: r["validation_job"] for r in inventory}
        self.assertEqual(len(jobs), len(inventory))
        self.assertEqual(jobs["v3evidence/go.mod"], "v3-evidence-checks")
        self.assertEqual(jobs["v3lifecycle/go.mod"], "v3-lifecycle-contract-checks")
        self.assertEqual(jobs["v3mediarelease/go.mod"], MEDIA_JOB)
        self.assertIn("unchanged root regression", jobs["go.mod"])


class MediaReleaseCharacterizationGateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.gate = BOUNDARY.mediarelease
        cls.ledgers = {}
        for path in ("pkg/contracts/openapi_characterization_test.go", "scripts/characterize-contracts.mjs"):
            pinned = BOUNDARY.git(REPO, "show", cls.gate.BASELINE + ":" + path)
            current = (REPO / path).read_bytes().replace(b"\r\n", b"\n")
            cls.ledgers[path] = pinned, current

    def test_both_registries_require_exact_media_extension(self):
        for path, (pinned, current) in self.ledgers.items():
            with self.subTest(registry=path):
                self.gate.check_characterization_delta(path, pinned, current)
                with self.assertRaises(ValueError):
                    self.gate.check_characterization_delta(path, pinned, pinned)

    def test_changed_registry_literals_cannot_replace_an_old_or_new_entrypoint(self):
        for path, (pinned, current) in self.ledgers.items():
            for name in (b"media-release-v1.yaml", b"control-api.yaml"):
                with self.subTest(registry=path, entrypoint=name):
                    self.assertIn(name, current)
                    mutation = current.replace(name, b"unexpected-api.yaml", 1)
                    with self.assertRaises(ValueError):
                        self.gate.check_characterization_delta(path, pinned, mutation)

    def test_public_entrypoint_set_requires_all_four_originals_and_only_one_addition(self):
        expected = ("control-api.yaml", "discord-bot-api.yaml", "encoder-recorder-api.yaml",
                    "observability-api.yaml", "media-release-v1.yaml")
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            api_dir = repo / "openapi"
            api_dir.mkdir()
            for name in expected:
                (api_dir / name).touch()
            self.gate.check_entrypoints(repo)
            for name in expected:
                entrypoint = api_dir / name
                entrypoint.unlink()
                with self.subTest(missing=name), self.assertRaises(ValueError):
                    self.gate.check_entrypoints(repo)
                entrypoint.touch()
            for name in ("unregistered.yaml", "unregistered.yml"):
                extra = api_dir / name
                extra.touch()
                with self.subTest(extra=name), self.assertRaises(ValueError):
                    self.gate.check_entrypoints(repo)
                extra.unlink()

    def test_unrelated_edit_in_scoped_characterization_source_is_rejected(self):
        for path, (pinned, current) in self.ledgers.items():
            with self.subTest(registry=path), self.assertRaises(ValueError):
                self.gate.check_characterization_delta(path, pinned, current + b"\n// unrelated change\n")


class MediaReleaseWorkflowGateTests(unittest.TestCase):
    def setUp(self):
        self.workflow = (REPO / BOUNDARY.WORKFLOW).read_bytes().replace(b"\r\n", b"\n")
        self.header = ("  " + MEDIA_JOB + ":").encode("ascii")
        self.assertEqual(self.workflow.count(b"\n" + self.header + b"\n"), 1)
        self.old, _, self.job_body = self.workflow.partition(b"\n" + self.header + b"\n")
        self.old += b"\n"

    def check(self, workflow):
        return BOUNDARY.mediarelease.check_workflow(REPO, workflow)

    def test_complete_job_is_accepted(self):
        self.check(self.workflow)

    def test_missing_duplicate_or_commented_job_rejected(self):
        for workflow in (self.old,
                         self.workflow + b"\n" + self.header + b"\n" + self.job_body,
                         self.old + b"  # " + self.header.strip() + b"\n" + self.job_body):
            with self.subTest(bytes=len(workflow)), self.assertRaises(ValueError):
                self.check(workflow)

    def test_new_job_cannot_be_disabled_or_retargeted(self):
        for body in (b"    if: false\n" + self.job_body,
                     self.job_body.replace(b"working-directory: v3mediarelease",
                                           b"working-directory: v3lifecycle"),
                     self.job_body.replace(b"1.26.5", b"1.25.5")):
            self.assertNotEqual(body, self.job_body)
            with self.subTest(body_prefix=body[:32]), self.assertRaises(ValueError):
                self.check(self.old + self.header + b"\n" + body)

    def test_older_jobs_cannot_be_substituted(self):
        for label in (b"  v3-evidence-checks:", b"  v3-lifecycle-contract-checks:"):
            with self.subTest(label=label), self.assertRaises(ValueError):
                self.check(self.workflow.replace(label, label.replace(b"checks", b"removed")))


class MediaReleaseOpenAPIGateTests(unittest.TestCase):
    def check_mutation(self, mutate):
        document = json.loads((REPO / MEDIA_OPENAPI).read_bytes())
        mutate(document)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "media-release-v1.yaml"
            path.write_text(json.dumps(document), encoding="utf-8")
            with self.assertRaises(ValueError):
                BOUNDARY.mediarelease.check_openapi(path)

    def test_published_contract_parses(self):
        BOUNDARY.mediarelease.check_openapi(REPO / MEDIA_OPENAPI)

    def test_unsupported_api_and_family_tuples_are_rejected(self):
        self.check_mutation(lambda d: d.update(openapi="2.0"))
        for key, value in (("family", "LegacyLifecycle"), ("wire_version", "2"),
                           ("profile_id", "FUTURE_PROFILE"), ("profile_revision", 1),
                           ("encoding_id", "UNREVIEWED")):
            with self.subTest(key=key):
                self.check_mutation(lambda d: d["x-autostream-contract"].update({key: value}))

    def test_required_record_schemas_cannot_be_removed(self):
        for name in ("CPAdmissionProof", "WorkerNonMediaBinding", "AppliedCoverProof",
                     "ReleasePermit", "OneTimeConsumption", "ReleaseReceipt"):
            with self.subTest(schema=name):
                self.check_mutation(lambda d: d["components"]["schemas"].pop(name))

    def test_missing_operation_and_dangling_refs_are_rejected(self):
        self.check_mutation(lambda d: d["paths"].pop(next(iter(d["paths"]))))
        self.check_mutation(lambda d: d["components"]["schemas"].update(
            ReleasePermit={"$ref": "#/components/schemas/UnknownPermit"}))
        self.check_mutation(lambda d: d["paths"]["/opens"]["post"]["requestBody"]["content"]
                            ["application/json"].update(schema={"$ref": "#/components/schemas/ReleaseReceipt"}))
        self.check_mutation(lambda d: d["paths"]["/records/inspect"]["post"].update(security=[]))

    def test_open_envelope_and_wrong_discriminator_are_rejected(self):
        self.check_mutation(lambda d: d["components"]["schemas"]["OneTimeConsumption"].update(
            additionalProperties=True))
        self.check_mutation(lambda d: d["components"]["schemas"]["ReleasePermitIdentity"]
                            ["properties"]["cc"].update(const="CC14"))
        self.check_mutation(lambda d: d["components"]["schemas"]["ReleaseReceipt"]["required"]
                            .remove("record_digest"))
        self.check_mutation(lambda d: d["components"]["schemas"]["WorkerNonMediaBindingIdentity"]
                            ["required"].remove("worker_generation"))
        self.check_mutation(lambda d: d["components"]["schemas"]["ReleasePermitIdentity"]
                            ["required"].remove("expected_slot_revision"))

    def test_duplicate_json_member_is_rejected(self):
        original = (REPO / MEDIA_OPENAPI).read_text(encoding="utf-8").lstrip()
        self.assertTrue(original.startswith("{"))
        duplicate = '{"openapi":"3.1.0",' + original[1:]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "media-release-v1.yaml"
            path.write_text(duplicate, encoding="utf-8")
            with self.assertRaises(ValueError):
                BOUNDARY.mediarelease.check_openapi(path)


class MediaReleaseEventGateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location(
            "mediarelease_test_events", Path(__file__).with_name("mediarelease_test_events.py"))
        cls.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.gate)
        cls.inventory = (REPO / MEDIA_DIR / "testdata" / "required-roots.txt").read_text(
            encoding="utf-8").splitlines()

    def complete(self):
        children = self.gate.REQUIRED_CHILDREN
        events = [{"Action": "start", "Package": MEDIA_MODULE}]
        def emit_test(name):
            events.append({"Action": "run", "Package": MEDIA_MODULE, "Test": name})
            direct_children = {child[len(name) + 1:].split("/", 1)[0]
                               for child in children if child.startswith(name + "/")}
            for child in sorted(direct_children):
                emit_test(name + "/" + child)
            events.append({"Action": "pass", "Package": MEDIA_MODULE, "Test": name})
        for name in self.inventory:
            emit_test(name)
        events.append({"Action": "pass", "Package": MEDIA_MODULE})
        return events

    def test_complete_media_inventory_passes(self):
        self.assertEqual(self.inventory, list(self.gate.REQUIRED))
        self.assertTrue(self.gate.REQUIRED_CHILDREN)
        summary = self.gate.verify_events(self.complete(), self.inventory)
        self.assertEqual(summary["required_roots"], len(self.inventory))
        self.assertEqual((summary["skip"], summary["fail"]), (0, 0))

    def test_missing_root_or_child_and_shortened_inventory_fail(self):
        for test in (self.inventory[0], self.gate.REQUIRED_CHILDREN[0]):
            with self.subTest(test=test), self.assertRaises(ValueError):
                self.gate.verify_events([event for event in self.complete() if event.get("Test") != test],
                                        self.inventory)
        with self.assertRaises(ValueError):
            self.gate.verify_events(self.complete(), self.inventory[:-1])

    def test_duplicate_root_and_reversed_child_lifecycle_fail(self):
        for event_action in ("run", "pass"):
            events = self.complete()
            index = next(i for i, e in enumerate(events)
                         if e.get("Test") == self.inventory[0] and e["Action"] == event_action)
            events.insert(index, dict(events[index]))
            with self.subTest(action=event_action), self.assertRaises(ValueError):
                self.gate.verify_events(events, self.inventory)
        events = self.complete()
        leaf = self.gate.REQUIRED_CHILDREN[0]
        run, passed = [i for i, e in enumerate(events) if e.get("Test") == leaf]
        events[run], events[passed] = events[passed], events[run]
        with self.assertRaises(ValueError):
            self.gate.verify_events(events, self.inventory)

    def test_skip_or_fail_is_never_accepted(self):
        for action in ("skip", "fail"):
            events = self.complete()
            next(e for e in events if e.get("Test") == self.gate.REQUIRED_CHILDREN[0]
                 and e["Action"] == "pass")["Action"] = action
            with self.subTest(action=action), self.assertRaises(ValueError):
                self.gate.verify_events(events, self.inventory)

    def test_module_and_package_execution_boundaries(self):
        events = self.complete()
        events[1]["Package"] = CHECK.MODULE
        early_pass = self.complete()
        early_pass.insert(2, early_pass.pop())
        for malformed in (events, early_pass, self.complete()[:-1]):
            with self.subTest(length=len(malformed)), self.assertRaises(ValueError):
                self.gate.verify_events(malformed, self.inventory)

    def test_duplicate_json_event_members_are_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "events.json"
            path.write_text('{"Action":"skip","Action":"pass","Package":"test"}\n', encoding="utf-8")
            with self.assertRaises(ValueError):
                self.gate.read_events(path)


if __name__ == "__main__":
    unittest.main()
