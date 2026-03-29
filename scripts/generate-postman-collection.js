const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const YAML = require("yaml");

const rootDir = path.resolve(__dirname, "..");
const specPath = path.join(rootDir, "TramplinAPI.yaml");
const outputDir = path.join(rootDir, "postman");
const collectionPath = path.join(outputDir, "Tramplin.postman_collection.json");
const environmentPath = path.join(outputDir, "Tramplin.local.postman_environment.json");

const spec = YAML.parse(fs.readFileSync(specPath, "utf8"));
const httpMethods = ["get", "post", "put", "patch", "delete", "options", "head"];
const rootSecurity = Array.isArray(spec.security) ? spec.security : [];
const exportedAt = new Date().toISOString();
const baseURLContext = buildBaseURLContext();
const seedVariables = {
  adminCuratorEmail: "seed.admin.curator@tramplin.local",
  curatorEmail: "seed.curator@tramplin.local",
  employerOwnerEmail: "seed.owner@tramplin.local",
  employerRecruiterEmail: "seed.recruiter@tramplin.local",
  applicantEmail: "seed.applicant.anna@tramplin.local",
  applicantSecondEmail: "seed.applicant.boris@tramplin.local",
  applicantThirdEmail: "seed.applicant.clara@tramplin.local",
  password: "password123",
  adminCuratorUserId: "11111111-1111-1111-1111-111111111201",
  curatorUserId: "11111111-1111-1111-1111-111111111202",
  employerOwnerUserId: "11111111-1111-1111-1111-111111111203",
  employerRecruiterUserId: "11111111-1111-1111-1111-111111111204",
  applicantUserId: "11111111-1111-1111-1111-111111111205",
  secondApplicantUserId: "11111111-1111-1111-1111-111111111206",
  thirdApplicantUserId: "11111111-1111-1111-1111-111111111207",
  companyId: "11111111-1111-1111-1111-111111111401",
  companySlug: "seed-labs",
  opportunityId: "11111111-1111-1111-1111-111111111601",
  eventOpportunityId: "11111111-1111-1111-1111-111111111602",
  draftOpportunityId: "11111111-1111-1111-1111-111111111603",
  opportunitySlug: "junior-go-backend-internship",
  eventOpportunitySlug: "spring-career-meetup",
  draftOpportunitySlug: "backend-mentor-circle",
  applicationId: "11111111-1111-1111-1111-111111111801",
  secondApplicationId: "11111111-1111-1111-1111-111111111802",
  connectionId: "11111111-1111-1111-1111-111111111901",
  campaignId: "11111111-1111-1111-1111-111111112101",
  verificationRequestId: "11111111-1111-1111-1111-111111112201",
  moderationCaseId: "11111111-1111-1111-1111-111111112301",
  companySocialLinkId: "11111111-1111-1111-1111-111111111431",
  applicantSocialLinkId: "11111111-1111-1111-1111-111111111701",
  companyMediaId: "11111111-1111-1111-1111-111111111433",
  mediaFileId: "11111111-1111-1111-1111-111111111301",
  tagId: "11111111-1111-1111-1111-111111111501",
  locationId: "11111111-1111-1111-1111-111111111101",
  membershipId: "11111111-1111-1111-1111-111111111411",
  linkId: "11111111-1111-1111-1111-111111111611",
  notificationId: "11111111-1111-1111-1111-111111112001",
  evidenceFileId: "11111111-1111-1111-1111-111111111303",
};

function main() {
  fs.mkdirSync(outputDir, { recursive: true });

  const collection = buildCollection();
  const environment = buildEnvironment();

  fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2) + "\n");
  fs.writeFileSync(environmentPath, JSON.stringify(environment, null, 2) + "\n");

  process.stdout.write(`Generated ${path.relative(rootDir, collectionPath)}\n`);
  process.stdout.write(`Generated ${path.relative(rootDir, environmentPath)}\n`);
}

function buildCollection() {
  const itemsByTag = new Map();
  const tagOrder = new Map((spec.tags || []).map((tag, index) => [tag.name, index]));
  const variableNames = new Set(["baseUrl", "baseProtocol", "baseHost", "basePort"]);

  for (const [pathTemplate, pathItem] of Object.entries(spec.paths || {})) {
    for (const method of httpMethods) {
      const operation = pathItem[method];
      if (!operation) {
        continue;
      }

      const tag = (operation.tags && operation.tags[0]) || "Ungrouped";
      if (!itemsByTag.has(tag)) {
        itemsByTag.set(tag, []);
      }

      const requestItem = buildRequestItem(pathTemplate, method, pathItem, operation, variableNames);
      itemsByTag.get(tag).push(requestItem);
    }
  }

  const orderedTags = [...itemsByTag.keys()].sort((left, right) => {
    const leftIndex = tagOrder.has(left) ? tagOrder.get(left) : Number.MAX_SAFE_INTEGER;
    const rightIndex = tagOrder.has(right) ? tagOrder.get(right) : Number.MAX_SAFE_INTEGER;
    if (leftIndex !== rightIndex) {
      return leftIndex - rightIndex;
    }
    return left.localeCompare(right);
  });

  return {
    info: {
      _postman_id: crypto.randomUUID(),
      name: spec.info?.title || "OpenAPI Collection",
      description: buildCollectionDescription(),
      schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
      version: String(spec.info?.version || "1.0.0"),
    },
    item: orderedTags.map((tag) => ({
      name: tag,
      item: itemsByTag.get(tag),
    })),
    variable: buildCollectionVariables(variableNames),
  };
}

function buildEnvironment() {
  return {
    id: crypto.randomUUID(),
    name: "Tramplin Local",
    values: [
      envVar("baseUrl", chooseBaseURL()),
      envVar("baseProtocol", baseURLContext.protocol),
      envVar("baseHost", baseURLContext.host),
      envVar("basePort", baseURLContext.port),
      ...Object.entries(seedVariables).map(([key, value]) => envVar(key, value)),
    ],
    _postman_variable_scope: "environment",
    _postman_exported_at: exportedAt,
    _postman_exported_using: "Codex OpenAPI generator",
  };
}

function buildCollectionDescription() {
  const lines = [];
  if (spec.info?.summary) {
    lines.push(spec.info.summary);
  }
  if (spec.info?.description) {
    lines.push(spec.info.description.trim());
  }
  lines.push("Generated from TramplinAPI.yaml.");
  lines.push("Auth is cookie-based. Login once in Postman and reuse the same environment/session so cookies persist.");
  lines.push("For local manual testing, import the bundled `Tramplin Local` environment and run `make infra-start`, `make db-migrate`, `make seed-dev`, then `make run`.");
  return lines.filter(Boolean).join("\n\n");
}

function buildCollectionVariables(variableNames) {
  const variables = [envVar("baseUrl", chooseBaseURL())];
  for (const name of [...variableNames].sort()) {
    if (name === "baseUrl") {
      continue;
    }
    variables.push(envVar(name, placeholderForVariable(name)));
  }
  return variables;
}

function buildRequestItem(pathTemplate, method, pathItem, operation, variableNames) {
  const parameters = mergeParameters(pathItem.parameters || [], operation.parameters || []);
  const requestURL = buildRequestURL(pathTemplate, parameters, variableNames);
  const body = buildRequestBody(operation, operation.requestBody, variableNames);
  const headers = [];

  if (body && body.mode === "raw") {
    headers.push({ key: "Content-Type", value: "application/json" });
  }

  return {
    name: operation.summary || operation.operationId || `${method.toUpperCase()} ${pathTemplate}`,
    request: {
      method: method.toUpperCase(),
      header: headers,
      body,
      url: requestURL,
      description: buildOperationDescription(operation),
    },
    response: [],
  };
}

function buildOperationDescription(operation) {
  const parts = [];
  if (operation.description) {
    parts.push(operation.description.trim());
  } else if (operation.summary) {
    parts.push(operation.summary.trim());
  }

  if (operation.operationId) {
    parts.push(`operationId: \`${operation.operationId}\``);
  }

  parts.push(describeSecurity(operation.security));
  return parts.filter(Boolean).join("\n\n");
}

function describeSecurity(operationSecurity) {
  const effectiveSecurity = operationSecurity === undefined ? rootSecurity : operationSecurity;
  if (Array.isArray(effectiveSecurity) && effectiveSecurity.length === 0) {
    return "Auth: none.";
  }
  return "Auth: cookie-based session. Use `/auth/login` first; Postman should retain the auth cookies automatically.";
}

function buildRequestURL(pathTemplate, parameters, variableNames) {
  const convertedPath = pathTemplate.replace(/\{([^}]+)\}/g, (_, name) => {
    const variableName = variableNameForPathParam(pathTemplate, name);
    variableNames.add(variableName);
    return `{{${variableName}}}`;
  });

  const query = parameters
    .filter((parameter) => parameter.in === "query")
    .map((parameter) => ({
      key: parameter.name,
      value: sampleScalar(parameter.schema, variableNames, parameter.name),
      disabled: !parameter.required,
      description: parameter.description || undefined,
    }));

  const requiredQuery = query
    .filter((parameter) => !parameter.disabled)
    .map((parameter) => `${encodeURIComponent(parameter.key)}=${encodeURIComponent(parameter.value)}`)
    .join("&");

  const raw = requiredQuery ? `{{baseUrl}}${convertedPath}?${requiredQuery}` : `{{baseUrl}}${convertedPath}`;
  const pathSegments = [
    ...baseURLContext.pathSegments,
    ...convertedPath.replace(/^\//, "").split("/").filter(Boolean),
  ];

  return {
    raw,
    protocol: "{{baseProtocol}}",
    host: ["{{baseHost}}"],
    port: "{{basePort}}",
    path: pathSegments,
    query,
  };
}

function buildRequestBody(operation, requestBody, variableNames) {
  const resolvedRequestBody = dereference(requestBody);
  if (!resolvedRequestBody || !resolvedRequestBody.content) {
    return undefined;
  }

  const jsonContent = resolvedRequestBody.content["application/json"];
  if (!jsonContent) {
    return undefined;
  }

  const schema = dereference(jsonContent.schema);
  if (!schema) {
    return undefined;
  }

  const example = sampleSchema(schema, variableNames);
  applyRequestBodyHeuristics(operation, example);
  return {
    mode: "raw",
    raw: JSON.stringify(example, null, 2),
    options: {
      raw: {
        language: "json",
      },
    },
  };
}

function mergeParameters(pathParameters, operationParameters) {
  const merged = new Map();
  for (const parameter of [...pathParameters, ...operationParameters]) {
    const resolved = dereference(parameter);
    if (!resolved) {
      continue;
    }
    merged.set(`${resolved.in}:${resolved.name}`, resolved);
  }
  return [...merged.values()];
}

function dereference(value) {
  if (!value || !value.$ref) {
    return value;
  }

  let current = spec;
  for (const segment of value.$ref.replace(/^#\//, "").split("/")) {
    const unescaped = segment.replace(/~1/g, "/").replace(/~0/g, "~");
    current = current?.[unescaped];
  }
  return current || null;
}

function sampleSchema(schema, variableNames, state = new Set()) {
  if (!schema) {
    return null;
  }

  if (schema.$ref) {
    if (state.has(schema.$ref)) {
      return null;
    }
    state.add(schema.$ref);
    const result = sampleSchema(dereference(schema), variableNames, state);
    state.delete(schema.$ref);
    return result;
  }

  if (schema.example !== undefined) {
    return schema.example;
  }
  if (schema.default !== undefined) {
    return schema.default;
  }
  if (schema.const !== undefined) {
    return schema.const;
  }
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    return schema.enum.find((value) => value !== null) ?? schema.enum[0];
  }

  if (Array.isArray(schema.oneOf) && schema.oneOf.length > 0) {
    return sampleSchema(selectMeaningfulVariant(schema.oneOf), variableNames, state);
  }
  if (Array.isArray(schema.anyOf) && schema.anyOf.length > 0) {
    return sampleSchema(selectMeaningfulVariant(schema.anyOf), variableNames, state);
  }
  if (Array.isArray(schema.allOf) && schema.allOf.length > 0) {
    const mergedObject = {};
    let hasObjectPart = false;
    for (const part of schema.allOf) {
      const sample = sampleSchema(part, variableNames, state);
      if (sample && typeof sample === "object" && !Array.isArray(sample)) {
        Object.assign(mergedObject, sample);
        hasObjectPart = true;
      }
    }
    if (hasObjectPart) {
      return mergedObject;
    }
    return sampleSchema(schema.allOf[0], variableNames, state);
  }

  const type = normalizeSchemaType(schema.type);
  switch (type) {
    case "object":
      return sampleObjectSchema(schema, variableNames, state);
    case "array":
      return [sampleSchema(dereference(schema.items), variableNames, state)];
    case "integer":
      return typeof schema.minimum === "number" ? schema.minimum : 1;
    case "number":
      return typeof schema.minimum === "number" ? schema.minimum : 1.5;
    case "boolean":
      return true;
    case "string":
    default:
      return sampleString(schema, variableNames);
  }
}

function sampleObjectSchema(schema, variableNames, state) {
  const properties = schema.properties || {};
  const required = new Set(schema.required || []);
  const result = {};

  for (const [name, propertySchema] of Object.entries(properties)) {
    if (required.has(name)) {
      result[name] = sampleSchema(dereference(propertySchema), variableNames, state);
    }
  }

  if (Object.keys(result).length === 0) {
    const firstProperty = Object.entries(properties)[0];
    if (firstProperty) {
      result[firstProperty[0]] = sampleSchema(dereference(firstProperty[1]), variableNames, state);
    }
  }

  applySchemaHeuristics(schema, result, variableNames, state);
  return result;
}

function applySchemaHeuristics(schema, result, variableNames, state) {
  const properties = schema.properties || {};

  if (properties.type && !result.type) {
    result.type = sampleSchema(dereference(properties.type), variableNames, state);
  }

  if (result.type && properties.vacancyDetails && properties.mentorProgramDetails && properties.eventDetails) {
    if (result.type === "internship" || result.type === "vacancy") {
      result.vacancyDetails = sampleSchema(dereference(properties.vacancyDetails), variableNames, state);
      delete result.mentorProgramDetails;
      delete result.eventDetails;
    } else if (result.type === "mentor_program") {
      result.mentorProgramDetails = sampleSchema(dereference(properties.mentorProgramDetails), variableNames, state);
      delete result.vacancyDetails;
      delete result.eventDetails;
    } else if (result.type === "event") {
      result.eventDetails = sampleSchema(dereference(properties.eventDetails), variableNames, state);
      delete result.vacancyDetails;
      delete result.mentorProgramDetails;
    }
  }

  if (properties.audienceType && properties.applicantUserIds && properties.opportunityId) {
    if (!result.audienceType) {
      result.audienceType = sampleSchema(dereference(properties.audienceType), variableNames, state);
    }
    if (result.audienceType === "all_applicants_of_opportunity") {
      result.opportunityId = sampleSchema(dereference(properties.opportunityId), variableNames, state);
      delete result.applicantUserIds;
    } else {
      result.applicantUserIds = [sampleSchema({ type: "string", format: "uuid" }, variableNames, state)];
      delete result.opportunityId;
    }
  }

  if (properties.method && properties.evidence && !result.evidence) {
    if (!result.method) {
      result.method = sampleSchema(dereference(properties.method), variableNames, state);
    }

    if (result.method === "official_website") {
      result.evidence = [
        {
          evidenceType: "website_link",
          value: "https://example.com/verify",
        },
      ];
    } else if (result.method === "corporate_email") {
      result.evidence = [
        {
          evidenceType: "corporate_email",
          value: "hr@example.com",
        },
      ];
    } else if (result.method === "inn") {
      variableNames.add("evidenceFileId");
      result.evidence = [
        {
          evidenceType: "inn_doc",
          evidenceFileId: "{{evidenceFileId}}",
        },
      ];
    } else {
      result.evidence = [];
    }
  }
}

function applyRequestBodyHeuristics(operation, example) {
  if (!example || typeof example !== "object" || Array.isArray(example)) {
    return;
  }

  switch (operation.operationId) {
    case "login":
      example.email = "{{employerOwnerEmail}}";
      example.password = "{{password}}";
      break;
    case "registerApplicant":
      example.email = "new.applicant.{{$timestamp}}@example.com";
      example.password = "{{password}}";
      example.displayName = "New Applicant";
      example.firstName = "New";
      example.lastName = "Applicant";
      break;
    case "registerEmployer":
      example.email = "new.employer.{{$timestamp}}@example.com";
      example.password = "{{password}}";
      example.displayName = "New Employer";
      example.fullName = "New Employer";
      break;
    case "createApplication":
      example.opportunityId = "{{opportunityId}}";
      break;
    case "saveOpportunity":
      example.opportunityId = "{{opportunityId}}";
      break;
    case "saveCompany":
      example.companyId = "{{companyId}}";
      break;
    case "createConnection":
      example.targetApplicantUserId = "{{secondApplicantUserId}}";
      break;
    case "createOpportunityRecommendation":
      example.recipientUserId = "{{secondApplicantUserId}}";
      example.opportunityId = "{{opportunityId}}";
      break;
    case "createCompanyMembership":
      example.employerEmail = "{{employerRecruiterEmail}}";
      break;
    case "createNotificationCampaign":
      example.companyId = "{{companyId}}";
      example.audienceType = "single_applicant";
      example.applicantUserIds = ["{{secondApplicantUserId}}"];
      delete example.opportunityId;
      break;
    default:
      break;
  }
}

function sampleString(schema, variableNames) {
  if (schema.format === "uuid") {
    return "11111111-1111-1111-1111-111111111111";
  }
  if (schema.format === "email") {
    return "user@example.com";
  }
  if (schema.format === "date-time") {
    return "2026-01-01T10:00:00Z";
  }
  if (schema.format === "uri") {
    return "https://example.com/resource";
  }
  if (schema.format === "password") {
    return "password123";
  }
  if (schema.format === "date") {
    return "2026-01-01";
  }
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    return schema.enum[0];
  }
  return schema.title || "string";
}

function sampleScalar(schema, variableNames, name) {
  const resolved = dereference(schema) || {};
  if (name) {
    variableNames.add(name);
  }

  if (resolved.type === "array" && resolved.items) {
    return String(sampleSchema(resolved.items, variableNames));
  }

  const sample = sampleSchema(resolved, variableNames);
  if (sample === null || sample === undefined) {
    return placeholderForVariable(name);
  }
  if (typeof sample === "object") {
    return JSON.stringify(sample);
  }
  return String(sample);
}

function selectMeaningfulVariant(variants) {
  for (const variant of variants) {
    const resolved = dereference(variant);
    if (!resolved) {
      continue;
    }
    const type = normalizeSchemaType(resolved.type);
    if (type && type !== "null") {
      return resolved;
    }
    if (resolved.properties || resolved.allOf || resolved.oneOf || resolved.anyOf) {
      return resolved;
    }
  }
  return dereference(variants[0]);
}

function normalizeSchemaType(type) {
  if (Array.isArray(type)) {
    return type.find((value) => value !== "null") || type[0];
  }
  return type;
}

function chooseBaseURL() {
  const servers = spec.servers || [];
  const local = servers.find((server) => /localhost|127\.0\.0\.1/.test(server.url));
  if (local) {
    return local.url.replace("localhost", "127.0.0.1");
  }
  return servers[0]?.url || "http://127.0.0.1:8080/v1";
}

function buildBaseURLContext() {
  const parsed = new URL(chooseBaseURL());
  const defaultPort = parsed.protocol === "https:" ? "443" : "80";

  return {
    protocol: parsed.protocol.replace(/:$/, ""),
    host: parsed.hostname,
    port: parsed.port || defaultPort,
    pathSegments: parsed.pathname.split("/").filter(Boolean),
  };
}

function variableNameForPathParam(pathTemplate, name) {
  if (name === "slug") {
    if (pathTemplate.includes("/opportunities/")) {
      return "opportunitySlug";
    }
    if (pathTemplate.includes("/companies/")) {
      return "companySlug";
    }
  }
  if (name === "userId" && (pathTemplate.startsWith("/applicants/") || pathTemplate.startsWith("/employer/applicants/"))) {
    return "applicantUserId";
  }
  if (name === "mediaFileId") {
    return "mediaFileId";
  }
  if (name === "companyId") {
    return "companyId";
  }
  if (name === "opportunityId") {
    return "opportunityId";
  }
  if (name === "applicationId") {
    return "applicationId";
  }
  if (name === "membershipId") {
    return "membershipId";
  }
  if (name === "connectionId") {
    return "connectionId";
  }
  if (name === "campaignId") {
    return "campaignId";
  }
  if (name === "verificationRequestId") {
    return "verificationRequestId";
  }
  if (name === "moderationCaseId") {
    return "moderationCaseId";
  }
  if (name === "notificationId") {
    return "notificationId";
  }
  if (name === "tagId") {
    return "tagId";
  }
  if (name === "locationId") {
    return "locationId";
  }
  if (name === "linkId") {
    if (pathTemplate.includes("/me/applicant/social-links/")) {
      return "applicantSocialLinkId";
    }
    if (pathTemplate.includes("/companies/") && pathTemplate.includes("/social-links/")) {
      return "companySocialLinkId";
    }
    return "linkId";
  }
  return name;
}

function placeholderForVariable(name) {
  if (!name) {
    return "value";
  }
  if (seedVariables[name]) {
    return seedVariables[name];
  }
  if (/id$/i.test(name)) {
    return "11111111-1111-1111-1111-111111111111";
  }
  if (/slug/i.test(name)) {
    return "example-slug";
  }
  if (/pageSize/i.test(name)) {
    return "20";
  }
  if (/page/i.test(name)) {
    return "1";
  }
  if (/q/i.test(name)) {
    return "search";
  }
  if (/email/i.test(name)) {
    return "user@example.com";
  }
  return name;
}

function envVar(key, value) {
  return {
    key,
    value,
    enabled: true,
  };
}

main();
