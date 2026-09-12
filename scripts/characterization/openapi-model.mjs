
import { createHash } from "node:crypto";
import path from "node:path";

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

export function normalizeBundle(value) {
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

export function analyzeOpenAPI(entrypoint, root) {
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

export function buildRefLayoutIndependentProjection(root) {
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

export function compareText(left, right) {
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
}

export function walkValue(value, pointer, visitor) {
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

export function escapeJSONPointerToken(value) {
  return String(value).replaceAll("~", "~0").replaceAll("/", "~1");
}

export function canonicalCompact(value) {
  return JSON.stringify(normalizeBundle(value));
}

export function sha256(value) {
  return createHash("sha256").update(value, "utf8").digest("hex");
}
