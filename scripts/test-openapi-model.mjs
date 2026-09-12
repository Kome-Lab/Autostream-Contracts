import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import {
  analyzeOpenAPI,
  buildRefLayoutIndependentProjection,
  canonicalCompact,
  normalizeBundle,
  sha256,
} from "./characterization/openapi-model.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const fixtureRoot = path.join(root, "testdata/characterization/generated/openapi");
const inventory = JSON.parse(readFileSync(path.join(fixtureRoot, "semantic-inventory.json"), "utf8"));
const bundleFor = (api) => JSON.parse(readFileSync(
  path.join(fixtureRoot, "normalized", path.basename(api, ".yaml") + ".json"), "utf8",
));

for (const expected of inventory.apis) {
  test(`pure model preserves fixed ${expected.api} semantic and fingerprint oracle`, () => {
    const normalized = normalizeBundle(bundleFor(expected.api));
    const actual = analyzeOpenAPI(expected.api, normalized);
    actual.normalized_bundle_sha256 = sha256(canonicalCompact(normalized));
    actual.ref_layout_independent_sha256 = sha256(canonicalCompact(buildRefLayoutIndependentProjection(normalized)));
    assert.deepEqual(actual, expected);
  });
}

test("new unclassified public operation still fails closed", () => {
  const root = bundleFor("openapi/control-api.yaml");
  root.paths["/unclassified-model-negative"] = {
    get: { security: [], responses: { "200": { description: "negative fixture" } } },
  };
  assert.throws(() => analyzeOpenAPI("openapi/control-api.yaml", root), /classify unauthenticated operation/);
});

test("unresolved references remain observable in the real model", () => {
  const root = bundleFor("openapi/control-api.yaml");
  root.components.schemas.ModelNegative = { $ref: "#/components/schemas/DoesNotExist" };
  assert.ok(analyzeOpenAPI("openapi/control-api.yaml", root).unresolved_refs.some(
    (item) => item.ref === "#/components/schemas/DoesNotExist",
  ));
});
