#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  realpathSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(SCRIPT_DIR, "..");
const COMMITTED_ROOT = path.join(REPO_ROOT, "testdata", "characterization", "generated");
const REDOCLY_PACKAGE = "@redocly/cli@2.39.0";
const REDOCLY_VERSION = "2.39.0";
const API_ENTRYPOINTS = [
  "openapi/control-api.yaml",
  "openapi/discord-bot-api.yaml",
  "openapi/encoder-recorder-api.yaml",
  "openapi/observability-api.yaml",
];
const HTTP_METHODS = ["delete", "get", "head", "options", "patch", "post", "put", "trace"];
const VOLATILE_GENERATOR_KEYS = new Set([
  "x-generated-at",
  "x-generated-by",
  "x-generator",
  "x-redocly-generated-at",
]);

const NO_AUTH_CLASSIFICATION = new Map([
  ["control-api.yaml|GET /health", "unauthenticated_health_probe"],
  ["control-api.yaml|GET /updater/version", "loopback_only"],
  ["control-api.yaml|POST /auth/login", "intended_public"],
  ["control-api.yaml|GET /auth/oauth/providers", "intended_public"],
  ["control-api.yaml|POST /auth/oauth/{id}/start", "intended_public"],
  ["control-api.yaml|POST /auth/oauth/callback", "intended_public"],
  ["control-api.yaml|GET /settings/app", "intended_public"],
  ["control-api.yaml|POST /setup/first-admin", "conditional_bootstrap_public"],
  ["control-api.yaml|GET /stream-previews/{token}/{name}", "capability_url"],
  ["control-api.yaml|GET /stream-previews/{token}/participants", "capability_url"],
  ["control-api.yaml|POST /services/host-agent/self-update-grants/consume", "grant_token_header"],
  ["discord-bot-api.yaml|GET /updater/version", "loopback_only"],
  ["encoder-recorder-api.yaml|GET /updater/version", "loopback_only"],
  ["observability-api.yaml|GET /health", "unauthenticated_health_probe"],
  ["observability-api.yaml|GET /status", "unauthenticated_internal_status"],
  ["observability-api.yaml|GET /updater/version", "loopback_only"],
]);

const command = process.argv[2] ?? "verify";
const options = parseOptions(process.argv.slice(3));

try {
  switch (command) {
    case "update":
      updateCommittedCharacterization();
      break;
    case "verify":
      verifyCommittedCharacterization();
      break;
    case "snapshot":
      if (!options.output) {
        throw new Error("snapshot requires --output <absolute-directory>");
      }
      captureSnapshot(requireAbsolutePath(options.output, "--output"));
      break;
    case "compare":
      if (!options.before || !options.after) {
        throw new Error("compare requires --before <directory> --after <directory>");
      }
      compareMoveOnlySnapshots(
        requireAbsolutePath(options.before, "--before"),
        requireAbsolutePath(options.after, "--after"),
      );
      break;
    default:
      throw new Error(`unknown command ${JSON.stringify(command)}\n${usage()}`);
  }
} catch (error) {
  process.stderr.write(`characterize-contracts: ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
}

function usage() {
  return [
    "usage:",
    "  node scripts/characterize-contracts.mjs verify",
    "  node scripts/characterize-contracts.mjs update",
    "  node scripts/characterize-contracts.mjs snapshot --output <absolute-directory>",
    "  node scripts/characterize-contracts.mjs compare --before <directory> --after <directory>",
  ].join("\n");
}

function parseOptions(args) {
  const parsed = {};
  for (let index = 0; index < args.length; index += 1) {
    const token = args[index];
    if (!token.startsWith("--") || index + 1 >= args.length) {
      throw new Error(`invalid option ${JSON.stringify(token)}\n${usage()}`);
    }
    parsed[token.slice(2)] = args[index + 1];
    index += 1;
  }
  return parsed;
}

function requireAbsolutePath(value, optionName) {
  if (!path.isAbsolute(value)) {
    throw new Error(`${optionName} must be absolute: ${JSON.stringify(value)}`);
  }
  return path.resolve(value);
}

function updateCommittedCharacterization() {
  const tempRoot = makeExternalTempDirectory("autostream-contracts-characterization-update-");
  try {
    const snapshotRoot = path.join(tempRoot, "snapshot");
    captureSnapshot(snapshotRoot);
    assertExactManagedRoot(COMMITTED_ROOT);
    rmSync(COMMITTED_ROOT, { recursive: true, force: true });
    mkdirSync(path.dirname(COMMITTED_ROOT), { recursive: true });
    cpSync(snapshotRoot, COMMITTED_ROOT, { recursive: true, errorOnExist: false });
    process.stdout.write(`updated ${toRepositoryPath(COMMITTED_ROOT)}\n`);
  } finally {
    removeExternalTempDirectory(tempRoot, "autostream-contracts-characterization-update-");
  }
}

function verifyCommittedCharacterization() {
  if (!existsSync(COMMITTED_ROOT)) {
    throw new Error(`${toRepositoryPath(COMMITTED_ROOT)} does not exist; run update first`);
  }
  const tempRoot = makeExternalTempDirectory("autostream-contracts-characterization-verify-");
  try {
    const snapshotRoot = path.join(tempRoot, "snapshot");
    captureSnapshot(snapshotRoot);
    compareDirectoriesExactly(COMMITTED_ROOT, snapshotRoot);
    process.stdout.write("contract characterization matches committed generated artifacts\n");
  } finally {
    removeExternalTempDirectory(tempRoot, "autostream-contracts-characterization-verify-");
  }
}

function captureSnapshot(outputRoot) {
  const resolvedOutput = path.resolve(outputRoot);
  if (resolvedOutput === REPO_ROOT || isWithin(resolvedOutput, REPO_ROOT)) {
    throw new Error(`snapshot output must be outside the repository: ${resolvedOutput}`);
  }
  if (existsSync(resolvedOutput) && readdirSync(resolvedOutput).length !== 0) {
    throw new Error(`snapshot output directory must be empty: ${resolvedOutput}`);
  }
  mkdirSync(resolvedOutput, { recursive: true });
  captureGoAndSchemaCharacterization(resolvedOutput);
  captureOpenAPICharacterization(resolvedOutput);
}

function captureGoAndSchemaCharacterization(outputRoot) {
  const result = runProcess(
    "go",
    [
      "test",
      "-mod=readonly",
      "-p",
      "1",
      "./pkg/contracts",
      "-run",
      "^Test(JSONSchema|PublicAPI|ZeroValueWire)Characterization$",
      "-count=1",
    ],
    {
      cwd: REPO_ROOT,
      env: {
        ...process.env,
        AUTOSTREAM_CHARACTERIZATION_OUTPUT: outputRoot,
        GOMAXPROCS: "2",
        GOTOOLCHAIN: "auto",
      },
    },
  );
  if (result.status !== 0) {
    throw commandFailure("Go/Schema characterization", result);
  }
  process.stdout.write(result.stdout);
  process.stderr.write(result.stderr);
}

function captureOpenAPICharacterization(outputRoot) {
  const actualEntrypoints = readdirSync(path.join(REPO_ROOT, "openapi"), { withFileTypes: true })
    .filter((entry) => entry.isFile() && /\.ya?ml$/i.test(entry.name))
    .map((entry) => `openapi/${entry.name}`)
    .sort();
  if (JSON.stringify(actualEntrypoints) !== JSON.stringify(API_ENTRYPOINTS)) {
    throw new Error(
      `OpenAPI entrypoint inventory drifted\nexpected=${JSON.stringify(API_ENTRYPOINTS)}\nactual=${JSON.stringify(actualEntrypoints)}`,
    );
  }

  const redoclyRoot = makeExternalTempDirectory("autostream-contracts-redocly-");
  const bundleRoot = path.join(redoclyRoot, "bundles");
  const cacheRoot = path.join(redoclyRoot, "npm-cache");
  mkdirSync(bundleRoot, { recursive: true });
  mkdirSync(cacheRoot, { recursive: true });
  const redoclyEnvironment = {
    ...process.env,
    FORCE_COLOR: "0",
    NO_COLOR: "1",
    REDOCLY_TELEMETRY: "off",
    npm_config_cache: cacheRoot,
    npm_config_update_notifier: "false",
  };

  try {
    const version = runRedocly(["--version"], redoclyEnvironment);
    if (version.status !== 0 || version.stdout.trim() !== REDOCLY_VERSION) {
      throw commandFailure(`Redocly version check (wanted ${REDOCLY_VERSION})`, version);
    }

    const sourceInventory = buildOpenAPISourceInventory(API_ENTRYPOINTS);
    writeCanonicalJSON(outputRoot, "openapi/source-inventory.json", sourceInventory);

    const fingerprints = { format_version: 1, redocly_version: REDOCLY_VERSION, apis: [] };
    const semanticInventory = { format_version: 1, redocly_version: REDOCLY_VERSION, apis: [] };
    const lintBaseline = { format_version: 1, redocly_version: REDOCLY_VERSION, apis: [] };

    for (const entrypoint of API_ENTRYPOINTS) {
      const apiName = path.basename(entrypoint, path.extname(entrypoint));
      const bundlePath = path.join(bundleRoot, `${apiName}.json`);
      const bundleResult = runRedocly(
        ["bundle", entrypoint, "--ext", "json", "--output", bundlePath],
        redoclyEnvironment,
      );
      if (bundleResult.status !== 0 || !existsSync(bundlePath)) {
        throw commandFailure(`Redocly bundle ${entrypoint}`, bundleResult);
      }
      process.stdout.write(bundleResult.stdout);
      process.stderr.write(bundleResult.stderr);

      const parsedBundle = JSON.parse(readFileSync(bundlePath, "utf8"));
      const normalizedBundle = normalizeBundle(parsedBundle);
      assertNoAbsolutePaths(normalizedBundle, `${entrypoint} normalized bundle`);
      const normalizedCompact = canonicalCompact(normalizedBundle);
      const normalizedSHA256 = sha256(normalizedCompact);
      const refLayoutIndependentSHA256 = sha256(
        canonicalCompact(buildRefLayoutIndependentProjection(normalizedBundle)),
      );
      const normalizedRelativePath = `openapi/normalized/${apiName}.json`;
      writeCanonicalJSON(outputRoot, normalizedRelativePath, normalizedBundle);

      const analysis = analyzeOpenAPI(entrypoint, normalizedBundle);
      analysis.normalized_bundle_sha256 = normalizedSHA256;
      analysis.ref_layout_independent_sha256 = refLayoutIndependentSHA256;
      semanticInventory.apis.push(analysis);
      fingerprints.apis.push({
        api: entrypoint,
        normalized_file: normalizedRelativePath,
        normalized_sha256: normalizedSHA256,
        ref_layout_independent_sha256: refLayoutIndependentSHA256,
      });

      const lintResult = runRedocly(
        ["lint", entrypoint, "--format", "json", "--max-problems", "10000"],
        redoclyEnvironment,
        true,
      );
      const lint = normalizeLintResult(entrypoint, lintResult);
      lintBaseline.apis.push(lint);
    }

    fingerprints.apis.sort((left, right) => compareText(left.api, right.api));
    semanticInventory.apis.sort((left, right) => compareText(left.api, right.api));
    lintBaseline.apis.sort((left, right) => compareText(left.api, right.api));
    writeCanonicalJSON(outputRoot, "openapi/fingerprints.json", fingerprints);
    writeCanonicalJSON(outputRoot, "openapi/semantic-inventory.json", semanticInventory);
    writeCanonicalJSON(outputRoot, "openapi/lint-baseline.json", lintBaseline);
    const control = semanticInventory.apis.find((api) => api.api === "openapi/control-api.yaml");
    if (!control) {
      throw new Error("control-api semantic inventory was not generated");
    }
    writeCanonicalJSON(outputRoot, "openapi/control-api-semantics.json", control);
  } finally {
    removeExternalTempDirectory(redoclyRoot, "autostream-contracts-redocly-");
  }
}

function runRedocly(args, environment, allowLintFailure = false) {
  let executable = "npx";
  let launcherArguments = [];
  if (process.platform === "win32") {
    const npxCLI = path.join(path.dirname(process.execPath), "node_modules", "npm", "bin", "npx-cli.js");
    if (!existsSync(npxCLI)) {
      throw new Error(`cannot locate the Node-bundled npx CLI at ${npxCLI}`);
    }
    executable = process.execPath;
    launcherArguments = [npxCLI];
  }
  const result = runProcess(
    executable,
    [...launcherArguments, "--yes", "--package", REDOCLY_PACKAGE, "redocly", ...args],
    { cwd: REPO_ROOT, env: environment },
  );
  if (!allowLintFailure && result.status !== 0) {
    return result;
  }
  return result;
}

function runProcess(executable, args, settings) {
  const result = spawnSync(executable, args, {
    cwd: settings.cwd,
    env: settings.env,
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
    windowsHide: true,
  });
  return {
    status: result.status ?? -1,
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
    error: result.error,
    command: [executable, ...args].join(" "),
  };
}

function commandFailure(label, result) {
  const errorPart = result.error ? `\n${result.error.message}` : "";
  return new Error(
    `${label} failed (exit ${result.status})\ncommand: ${result.command}${errorPart}\n${result.stdout}${result.stderr}`,
  );
}

function buildOpenAPISourceInventory(entrypoints) {
  const visited = new Set();
  const refs = [];
  const queue = entrypoints.map((entrypoint) => path.join(REPO_ROOT, entrypoint));
  while (queue.length > 0) {
    const current = path.resolve(queue.shift());
    assertRepositoryFile(current, "OpenAPI source/ref");
    const repositoryPath = toRepositoryPath(current);
    if (visited.has(repositoryPath)) {
      continue;
    }
    visited.add(repositoryPath);
    const raw = normalizeLineEndings(readFileSync(current, "utf8"));
    for (const ref of extractRefs(current, raw)) {
      const refPath = ref.value.split("#", 1)[0];
      if (refPath === "") {
        refs.push({ source: repositoryPath, value: ref.value, target: repositoryPath });
        continue;
      }
      if (/^[A-Za-z][A-Za-z0-9+.-]*:/.test(refPath) || refPath.startsWith("//")) {
        throw new Error(`external network OpenAPI ref is forbidden: ${repositoryPath} -> ${ref.value}`);
      }
      if (path.isAbsolute(refPath)) {
        throw new Error(`absolute OpenAPI ref is forbidden: ${repositoryPath} -> ${ref.value}`);
      }
      const target = path.resolve(path.dirname(current), refPath);
      assertRepositoryFile(target, `OpenAPI ref from ${repositoryPath}`);
      const targetRepositoryPath = toRepositoryPath(target);
      refs.push({ source: repositoryPath, value: ref.value, target: targetRepositoryPath });
      queue.push(target);
    }
  }
  const files = [...visited]
    .sort()
    .map((repositoryPath) => {
      const body = normalizeLineEndings(readFileSync(path.join(REPO_ROOT, repositoryPath), "utf8"));
      return { path: repositoryPath, normalized_source_sha256: sha256(body) };
    });
  refs.sort((left, right) =>
    compareText(`${left.source}\0${left.value}\0${left.target}`, `${right.source}\0${right.value}\0${right.target}`),
  );
  return {
    format_version: 1,
    entrypoint_count: entrypoints.length,
    entrypoints: [...entrypoints],
    source_file_count: files.length,
    source_files: files,
    refs,
    external_network_refs: [],
    missing_local_refs: [],
  };
}

function extractRefs(filePath, raw) {
  if (filePath.endsWith(".json")) {
    const refs = [];
    walkValue(JSON.parse(raw), "", (value, pointer) => {
      if (value && typeof value === "object" && !Array.isArray(value) && typeof value.$ref === "string") {
        refs.push({ pointer: `${pointer}/$ref`, value: value.$ref });
      }
    });
    return refs;
  }
  const refs = [];
  const expression = /\$ref\s*:\s*(?:"([^"]+)"|'([^']+)'|([^\s,}\]]+))/g;
  for (const match of raw.matchAll(expression)) {
    refs.push({ pointer: `offset:${match.index}`, value: match[1] ?? match[2] ?? match[3] });
  }
  return refs;
}

function normalizeBundle(value) {
  if (Array.isArray(value)) {
    return value.map(normalizeBundle);
  }
  if (value && typeof value === "object") {
    const normalized = {};
    for (const key of Object.keys(value).sort()) {
      if (VOLATILE_GENERATOR_KEYS.has(key)) {
        continue;
      }
      normalized[key] = normalizeBundle(value[key]);
    }
    return normalized;
  }
  return value;
}

function analyzeOpenAPI(entrypoint, root) {
  const methodCounts = Object.fromEntries(HTTP_METHODS.map((method) => [method, 0]));
  const responseStatuses = new Map();
  const contentTypes = new Set();
  const operationIds = new Map();
  const operations = [];
  const methodPaths = new Set();
  const duplicateMethodPaths = [];
  const rootHasSecurity = Object.hasOwn(root, "security");
  let requestBodyCount = 0;
  let explicitSecurityCount = 0;
  let inheritedSecurityCount = 0;
  let undefinedSecurityCount = 0;
  let inlineSchemaCount = 0;

  for (const apiPath of Object.keys(root.paths ?? {}).sort()) {
    const pathItem = dereferenceOnce(root, root.paths[apiPath]);
    for (const method of HTTP_METHODS) {
      if (!Object.hasOwn(pathItem ?? {}, method)) {
        continue;
      }
      const operation = dereferenceOnce(root, pathItem[method]);
      const operationKey = `${method.toUpperCase()} ${apiPath}`;
      if (methodPaths.has(operationKey)) {
        duplicateMethodPaths.push(operationKey);
      }
      methodPaths.add(operationKey);
      methodCounts[method] += 1;
      if (typeof operation.operationId === "string" && operation.operationId !== "") {
        const keys = operationIds.get(operation.operationId) ?? [];
        keys.push(operationKey);
        operationIds.set(operation.operationId, keys);
      }

      const hasExplicitSecurity = Object.hasOwn(operation, "security");
      let securitySource = "undefined";
      let effectiveSecurity = null;
      if (hasExplicitSecurity) {
        securitySource = "explicit";
        effectiveSecurity = normalizeBundle(operation.security);
        explicitSecurityCount += 1;
      } else if (rootHasSecurity) {
        securitySource = "inherited";
        effectiveSecurity = normalizeBundle(root.security);
        inheritedSecurityCount += 1;
      } else {
        undefinedSecurityCount += 1;
      }
      const noAuth = securityAllowsNoAuthentication(effectiveSecurity);
      let exposureClassification = "authenticated";
      if (securitySource === "undefined") {
        exposureClassification = "unknown_security";
      } else if (noAuth) {
        const classificationKey = `${path.basename(entrypoint)}|${operationKey}`;
        exposureClassification = NO_AUTH_CLASSIFICATION.get(classificationKey) ?? "unknown_unauthenticated";
        if (exposureClassification === "unknown_unauthenticated") {
          throw new Error(`classify unauthenticated operation before updating baseline: ${classificationKey}`);
        }
      }

      const requestBody = operation.requestBody ? dereferenceOnce(root, operation.requestBody) : null;
      const requestContent = characterizeContent(root, requestBody?.content ?? {});
      if (requestBody) {
        requestBodyCount += 1;
      }
      for (const item of requestContent) {
        contentTypes.add(item.content_type);
        inlineSchemaCount += item.schema?.kind === "inline" ? 1 : 0;
      }

      const responses = [];
      for (const status of Object.keys(operation.responses ?? {}).sort(compareStatusCodes)) {
        const response = dereferenceOnce(root, operation.responses[status]);
        responseStatuses.set(status, (responseStatuses.get(status) ?? 0) + 1);
        const responseContent = characterizeContent(root, response?.content ?? {});
        for (const item of responseContent) {
          contentTypes.add(item.content_type);
          inlineSchemaCount += item.schema?.kind === "inline" ? 1 : 0;
        }
        responses.push({
          status,
          content_types: responseContent.map((item) => item.content_type),
          schemas: responseContent.map((item) => item.schema).filter(Boolean),
        });
      }

      operations.push({
        method,
        path: apiPath,
        operation_id: operation.operationId ?? "",
        tags: [...(operation.tags ?? [])].sort(),
        security_source: securitySource,
        effective_security: effectiveSecurity,
        exposure_classification: exposureClassification,
        request_body_content_types: requestContent.map((item) => item.content_type),
        request_schemas: requestContent.map((item) => item.schema).filter(Boolean),
        responses,
        deprecated: operation.deprecated === true,
        summary_present: typeof operation.summary === "string" && operation.summary.trim() !== "",
      });
    }
  }

  operations.sort((left, right) => compareText(`${left.path}\0${left.method}`, `${right.path}\0${right.method}`));
  const duplicateOperationIds = [...operationIds.entries()]
    .filter(([, keys]) => keys.length > 1)
    .map(([operationId, keys]) => ({ operation_id: operationId, operations: [...keys].sort() }))
    .sort((left, right) => compareText(left.operation_id, right.operation_id));
  const unresolvedRefs = collectUnresolvedRefs(root);
  const noResponses = operations.filter((operation) => operation.responses.length === 0).map(operationIdentity);
  const noSuccessResponse = operations
    .filter((operation) => !operation.responses.some((response) => /^2(?:\d\d|XX)$/i.test(response.status)))
    .map(operationIdentity);
  const no4xxResponse = operations
    .filter((operation) => !operation.responses.some((response) => /^4(?:\d\d|XX)$/i.test(response.status)))
    .map(operationIdentity);
  const undefinedSecurity = operations
    .filter((operation) => operation.security_source === "undefined")
    .map(operationIdentity);
  const unauthenticatedOperations = operations
    .filter((operation) => operation.exposure_classification !== "authenticated" && operation.exposure_classification !== "unknown_security")
    .map((operation) => ({ ...operationIdentity(operation), classification: operation.exposure_classification }));
  const publicOperations = unauthenticatedOperations.filter((operation) =>
    ["intended_public", "conditional_bootstrap_public"].includes(operation.classification),
  );

  return {
    api: entrypoint,
    title: root.info?.title ?? "",
    openapi_version: root.openapi ?? "",
    bundle_success: true,
    path_count: Object.keys(root.paths ?? {}).length,
    operation_count: operations.length,
    method_counts: methodCounts,
    schema_count: Object.keys(root.components?.schemas ?? {}).length,
    security_scheme_count: Object.keys(root.components?.securitySchemes ?? {}).length,
    operation_id_count: [...operationIds.values()].reduce((total, keys) => total + keys.length, 0),
    duplicate_operation_ids: duplicateOperationIds,
    unresolved_ref_count: unresolvedRefs.length,
    unresolved_refs: unresolvedRefs,
    response_status_distribution: Object.fromEntries([...responseStatuses.entries()].sort(([left], [right]) => compareStatusCodes(left, right))),
    request_body_count: requestBodyCount,
    explicit_security_count: explicitSecurityCount,
    inherited_security_count: inheritedSecurityCount,
    undefined_security_count: undefinedSecurityCount,
    unauthenticated_operation_count: unauthenticatedOperations.length,
    public_operation_count: publicOperations.length,
    public_operations: publicOperations,
    unauthenticated_operations: unauthenticatedOperations,
    content_type_inventory: [...contentTypes].sort(),
    duplicate_method_paths: duplicateMethodPaths.sort(),
    operations_without_security_definition: undefinedSecurity,
    operations_without_responses: noResponses,
    operations_without_success_response: noSuccessResponse,
    operations_without_4xx_response: no4xxResponse,
    inline_schema_count: inlineSchemaCount,
    operations,
  };
}

function characterizeContent(root, content) {
  return Object.keys(content ?? {})
    .sort()
    .map((contentType) => ({
      content_type: contentType,
      schema: characterizeSchema(root, content[contentType]?.schema),
    }));
}

function characterizeSchema(root, schema) {
  if (!schema || typeof schema !== "object") {
    return null;
  }
  if (typeof schema.$ref === "string") {
    const resolved = resolveJSONReference(root, schema.$ref);
    return {
      kind: "ref",
      ref: schema.$ref,
      name: decodeReferenceName(schema.$ref),
      resolved_sha256: resolved === undefined ? "" : sha256(canonicalCompact(fullyResolveValue(root, resolved, new Set([schema.$ref])))),
    };
  }
  return {
    kind: "inline",
    name: typeof schema.title === "string" ? schema.title : "",
    type: schema.type ?? "",
    resolved_sha256: sha256(canonicalCompact(fullyResolveValue(root, schema, new Set()))),
  };
}

function dereferenceOnce(root, value) {
  if (!value || typeof value !== "object" || typeof value.$ref !== "string") {
    return value;
  }
  return resolveJSONReference(root, value.$ref) ?? value;
}

function resolveJSONReference(root, reference) {
  if (reference === "#") {
    return root;
  }
  if (!reference.startsWith("#/")) {
    return undefined;
  }
  let current = root;
  for (const rawToken of reference.slice(2).split("/")) {
    const token = decodeURIComponent(rawToken).replaceAll("~1", "/").replaceAll("~0", "~");
    if (!current || typeof current !== "object" || !Object.hasOwn(current, token)) {
      return undefined;
    }
    current = current[token];
  }
  return current;
}

function collectUnresolvedRefs(root) {
  const unresolved = [];
  walkValue(root, "", (value, pointer) => {
    if (!value || typeof value !== "object" || Array.isArray(value) || typeof value.$ref !== "string") {
      return;
    }
    if (resolveJSONReference(root, value.$ref) === undefined) {
      unresolved.push({ pointer: `${pointer}/$ref`, ref: value.$ref });
    }
  });
  unresolved.sort((left, right) => compareText(`${left.pointer}\0${left.ref}`, `${right.pointer}\0${right.ref}`));
  return unresolved;
}

function buildRefLayoutIndependentProjection(root) {
  const projection = {};
  for (const key of Object.keys(root).sort()) {
    if (key === "components") {
      continue;
    }
    projection[key] = fullyResolveValue(root, root[key], new Set());
  }
  const componentMultisets = {};
  for (const category of Object.keys(root.components ?? {}).sort()) {
    const values = root.components[category];
    if (!values || typeof values !== "object" || Array.isArray(values)) {
      componentMultisets[category] = fullyResolveValue(root, values, new Set());
      continue;
    }
    componentMultisets[category] = Object.values(values)
      .map((value) => sha256(canonicalCompact(fullyResolveValue(root, value, new Set()))))
      .sort();
  }
  projection.component_semantic_hashes = componentMultisets;
  return normalizeBundle(projection);
}

function fullyResolveValue(root, value, stack) {
  if (Array.isArray(value)) {
    return value.map((item) => fullyResolveValue(root, item, stack));
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  if (typeof value.$ref === "string") {
    const reference = value.$ref;
    const siblings = Object.fromEntries(Object.entries(value).filter(([key]) => key !== "$ref"));
    if (stack.has(reference)) {
      return { $recursive: true, ...fullyResolveValue(root, siblings, stack) };
    }
    const target = resolveJSONReference(root, reference);
    if (target === undefined) {
      return { $unresolved: true, ...fullyResolveValue(root, siblings, stack) };
    }
    const nextStack = new Set(stack);
    nextStack.add(reference);
    const resolved = fullyResolveValue(root, target, nextStack);
    const resolvedSiblings = fullyResolveValue(root, siblings, nextStack);
    if (resolved && typeof resolved === "object" && !Array.isArray(resolved)) {
      return normalizeBundle({ ...resolved, ...resolvedSiblings });
    }
    return normalizeBundle({ $resolved_value: resolved, ...resolvedSiblings });
  }
  const result = {};
  for (const key of Object.keys(value).sort()) {
    result[key] = fullyResolveValue(root, value[key], stack);
  }
  return result;
}

function normalizeLintResult(entrypoint, result) {
  let parsed;
  try {
    parsed = JSON.parse(result.stdout.trim() || "[]");
  } catch (error) {
    throw new Error(
      `cannot parse Redocly lint JSON for ${entrypoint}: ${error.message}\nstdout=${result.stdout}\nstderr=${result.stderr}`,
    );
  }
  const rawProblems = Array.isArray(parsed) ? parsed : parsed.problems ?? parsed.issues ?? [];
  const findings = rawProblems.map((problem) => {
    const locations = (problem.location ?? problem.locations ?? [])
      .map((location) => normalizeLintLocation(entrypoint, location))
      .sort((left, right) => compareText(`${left.source}\0${left.pointer}`, `${right.source}\0${right.pointer}`));
    const finding = {
      severity: String(problem.severity ?? "unknown").toLowerCase(),
      category: String(problem.category ?? "openapi_lint"),
      rule_id: String(problem.ruleId ?? problem.rule_id ?? problem.code ?? "unknown"),
      locations,
    };
    return { ...finding, fingerprint: sha256(canonicalCompact(finding)) };
  });
  findings.sort((left, right) =>
    compareText(
      `${left.severity}\0${left.rule_id}\0${left.fingerprint}`,
      `${right.severity}\0${right.rule_id}\0${right.fingerprint}`,
    ),
  );
  const errorCount = findings.filter((finding) => finding.severity === "error").length;
  const warningCount = findings.filter((finding) => finding.severity === "warn" || finding.severity === "warning").length;
  if (result.status !== 0 && errorCount === 0) {
    throw commandFailure(`Redocly lint ${entrypoint}`, result);
  }
  return {
    api: entrypoint,
    lint_exit_code: result.status,
    error_count: errorCount,
    warning_count: warningCount,
    finding_count: findings.length,
    findings,
  };
}

function normalizeLintLocation(entrypoint, location) {
  const rawSource = location?.source?.absoluteRef ?? location?.source?.ref ?? location?.source ?? entrypoint;
  let source = String(rawSource);
  if (source.startsWith("file://")) {
    try {
      source = fileURLToPath(source);
    } catch {
      source = entrypoint;
    }
  }
  if (path.isAbsolute(source)) {
    const resolved = path.resolve(source);
    source = isWithin(resolved, REPO_ROOT) || resolved === REPO_ROOT ? toRepositoryPath(resolved) : "$EXTERNAL";
  } else {
    source = source.replaceAll("\\", "/");
  }
  let pointer = location?.pointer ?? "";
  if (Array.isArray(pointer)) {
    pointer = `/${pointer.map(escapeJSONPointerToken).join("/")}`;
  }
  if (!pointer && Array.isArray(location?.reportOnKey)) {
    pointer = `/${location.reportOnKey.map(escapeJSONPointerToken).join("/")}`;
  }
  return { source, pointer: String(pointer) };
}

function securityAllowsNoAuthentication(security) {
  return Array.isArray(security) &&
    (security.length === 0 || security.some((requirement) => requirement && typeof requirement === "object" && Object.keys(requirement).length === 0));
}

function operationIdentity(operation) {
  return { method: operation.method, path: operation.path, operation_id: operation.operation_id };
}

function decodeReferenceName(reference) {
  const token = reference.split("/").at(-1) ?? "";
  return decodeURIComponent(token).replaceAll("~1", "/").replaceAll("~0", "~");
}

function compareStatusCodes(left, right) {
  if (left === "default") return 1;
  if (right === "default") return -1;
  return compareText(left, right);
}

function compareText(left, right) {
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
}

function walkValue(value, pointer, visitor) {
  visitor(value, pointer);
  if (Array.isArray(value)) {
    value.forEach((child, index) => walkValue(child, `${pointer}/${index}`, visitor));
    return;
  }
  if (value && typeof value === "object") {
    for (const key of Object.keys(value)) {
      walkValue(value[key], `${pointer}/${escapeJSONPointerToken(key)}`, visitor);
    }
  }
}

function escapeJSONPointerToken(value) {
  return String(value).replaceAll("~", "~0").replaceAll("/", "~1");
}

function writeCanonicalJSON(outputRoot, relativePath, value) {
  assertNoAbsolutePaths(value, relativePath);
  const target = path.resolve(outputRoot, relativePath);
  if (!isWithin(target, outputRoot)) {
    throw new Error(`unsafe generated path ${target}`);
  }
  mkdirSync(path.dirname(target), { recursive: true });
  writeFileSync(target, `${JSON.stringify(normalizeBundle(value), null, 2)}\n`, { encoding: "utf8", mode: 0o600 });
}

function assertNoAbsolutePaths(value, label) {
  const serialized = JSON.stringify(value);
  const variants = [REPO_ROOT, REPO_ROOT.replaceAll("\\", "/")];
  for (const variant of variants) {
    if (serialized.toLowerCase().includes(variant.toLowerCase())) {
      throw new Error(`${label} contains absolute repository path`);
    }
  }
  if (/\b[A-Za-z]:[\\/](?:Users|Windows|Program Files)\b/i.test(serialized)) {
    throw new Error(`${label} contains an absolute Windows path`);
  }
}

function canonicalCompact(value) {
  return JSON.stringify(normalizeBundle(value));
}

function sha256(value) {
  return createHash("sha256").update(value, "utf8").digest("hex");
}

function normalizeLineEndings(value) {
  return value.replaceAll("\r\n", "\n").replaceAll("\r", "\n");
}

function compareDirectoriesExactly(expectedRoot, actualRoot) {
  const expectedFiles = listFiles(expectedRoot);
  const actualFiles = listFiles(actualRoot);
  if (JSON.stringify(expectedFiles) !== JSON.stringify(actualFiles)) {
    throw new Error(
      `generated file inventory drifted\nexpected=${JSON.stringify(expectedFiles)}\nactual=${JSON.stringify(actualFiles)}`,
    );
  }
  const changed = expectedFiles.filter(
    (relativePath) =>
      normalizeLineEndings(readFileSync(path.join(expectedRoot, relativePath), "utf8")) !==
      normalizeLineEndings(readFileSync(path.join(actualRoot, relativePath), "utf8")),
  );
  if (changed.length > 0) {
    throw new Error(`characterization drifted in ${changed.join(", ")}`);
  }
}

function compareMoveOnlySnapshots(beforeRoot, afterRoot) {
  const before = loadMoveOnlyProjection(beforeRoot);
  const after = loadMoveOnlyProjection(afterRoot);
  const beforeCanonical = canonicalCompact(before);
  const afterCanonical = canonicalCompact(after);
  if (beforeCanonical !== afterCanonical) {
    throw new Error(
      `move-only semantic comparison failed\nbefore_sha256=${sha256(beforeCanonical)}\nafter_sha256=${sha256(afterCanonical)}`,
    );
  }
  process.stdout.write(`move-only semantic comparison passed: ${sha256(beforeCanonical)}\n`);
}

function loadMoveOnlyProjection(root) {
  const required = [
    "public-api.json",
    "struct-fields.json",
    "enum-constants.json",
    "zero-value-wire.json",
    "schemas.json",
    "openapi/fingerprints.json",
    "openapi/semantic-inventory.json",
  ];
  for (const relativePath of required) {
    if (!existsSync(path.join(root, relativePath))) {
      throw new Error(`snapshot is missing ${relativePath}: ${root}`);
    }
  }
  const fingerprints = readJSON(path.join(root, "openapi", "fingerprints.json"));
  const semantics = readJSON(path.join(root, "openapi", "semantic-inventory.json"));
  return {
    public_api: readJSON(path.join(root, "public-api.json")),
    struct_fields: readJSON(path.join(root, "struct-fields.json")),
    enum_constants: readJSON(path.join(root, "enum-constants.json")),
    zero_value_wire: readJSON(path.join(root, "zero-value-wire.json")),
    schemas: readJSON(path.join(root, "schemas.json")),
    openapi_ref_layout_independent: fingerprints.apis.map((api) => ({
      api: api.api,
      ref_layout_independent_sha256: api.ref_layout_independent_sha256,
    })),
    openapi_operation_semantics: semantics.apis.map(stripLayoutSensitiveOpenAPIFields),
  };
}

function stripLayoutSensitiveOpenAPIFields(value) {
  if (Array.isArray(value)) {
    return value.map(stripLayoutSensitiveOpenAPIFields);
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  const result = {};
  for (const key of Object.keys(value).sort()) {
    if (["ref", "name", "normalized_bundle_sha256"].includes(key)) {
      continue;
    }
    result[key] = stripLayoutSensitiveOpenAPIFields(value[key]);
  }
  return result;
}

function readJSON(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function listFiles(root) {
  const files = [];
  const visit = (current) => {
    for (const entry of readdirSync(current, { withFileTypes: true }).sort((left, right) => compareText(left.name, right.name))) {
      const child = path.join(current, entry.name);
      if (entry.isDirectory()) {
        visit(child);
      } else if (entry.isFile()) {
        files.push(path.relative(root, child).replaceAll("\\", "/"));
      }
    }
  };
  visit(root);
  return files.sort();
}

function assertRepositoryFile(filePath, label) {
  const resolved = path.resolve(filePath);
  if (!isWithin(resolved, REPO_ROOT) || !existsSync(resolved) || !statSync(resolved).isFile()) {
    throw new Error(`${label} is missing or outside the repository: ${resolved}`);
  }
}

function toRepositoryPath(filePath) {
  const resolved = path.resolve(filePath);
  if (resolved === REPO_ROOT) {
    return ".";
  }
  if (!isWithin(resolved, REPO_ROOT)) {
    throw new Error(`path is outside repository: ${resolved}`);
  }
  return path.relative(REPO_ROOT, resolved).replaceAll("\\", "/");
}

function isWithin(candidate, parent) {
  const relative = path.relative(path.resolve(parent), path.resolve(candidate));
  return relative !== "" && !relative.startsWith("..") && !path.isAbsolute(relative);
}

function makeExternalTempDirectory(prefix) {
  const root = mkdtempSync(path.join(realpathSync(tmpdir()), prefix));
  if (root === REPO_ROOT || isWithin(root, REPO_ROOT)) {
    throw new Error(`temporary directory must be outside repository: ${root}`);
  }
  return root;
}

function removeExternalTempDirectory(directory, expectedPrefix) {
  const resolved = path.resolve(directory);
  const tempRoot = realpathSync(tmpdir());
  if (!isWithin(resolved, tempRoot) || !path.basename(resolved).startsWith(expectedPrefix)) {
    throw new Error(`unsafe temporary cleanup target: ${resolved}`);
  }
  rmSync(resolved, { recursive: true, force: true });
}

function assertExactManagedRoot(directory) {
  if (path.resolve(directory) !== path.resolve(REPO_ROOT, "testdata", "characterization", "generated")) {
    throw new Error(`unsafe managed characterization root: ${directory}`);
  }
}
