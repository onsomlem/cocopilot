import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ListResourcesRequestSchema,
  ReadResourceRequestSchema,
  ListPromptsRequestSchema,
  GetPromptRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";

const DEFAULT_API_BASE = "http://localhost:8080";
const API_BASE = normalizeApiBase(process.env.COCO_API_BASE);
const API_KEY = process.env.COCO_API_KEY ?? "";
const API_HEADERS_RAW = process.env.COCO_API_HEADERS ?? "";
const LOG_PREFIX = "cocopilot-mcp";
const MAX_BODY_LOG = 2000;

type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue };

type ToolArgs = Record<string, JsonValue> | undefined;

function logInfo(message: string): void {
  console.error(`[${LOG_PREFIX}] ${message}`);
}

function logError(message: string): void {
  console.error(`[${LOG_PREFIX}] ${message}`);
}

function formatBody(text: string): string {
  const trimmed = text.trim();
  if (!trimmed) {
    return "<empty>";
  }
  if (trimmed.length > MAX_BODY_LOG) {
    return `${trimmed.slice(0, MAX_BODY_LOG)}...<truncated>`;
  }
  return trimmed;
}

function normalizeApiBase(raw: string | undefined): string {
  const value = (raw ?? "").trim();
  const candidate = value.length > 0 ? value : DEFAULT_API_BASE;

  let parsed: URL;
  try {
    parsed = new URL(candidate);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`COCO_API_BASE must be a valid URL, got "${candidate}": ${message}`);
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error(
      `COCO_API_BASE must start with http:// or https://, got "${candidate}"`
    );
  }

  if (!parsed.hostname) {
    throw new Error(`COCO_API_BASE must include a host, got "${candidate}"`);
  }

  const basePath = parsed.pathname === "/" ? "" : parsed.pathname.replace(/\/+$/, "");
  return `${parsed.origin}${basePath}`;
}

function parseHeaderJson(raw: string): Record<string, string> {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Invalid COCO_API_HEADERS JSON: ${message}`);
  }

  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error("COCO_API_HEADERS must be a JSON object of string headers");
  }

  const headers: Record<string, string> = {};
  for (const [key, value] of Object.entries(parsed)) {
    if (typeof value !== "string") {
      throw new Error(`COCO_API_HEADERS value for ${key} must be a string`);
    }
    headers[key] = value;
  }

  return headers;
}

function getAuthHeaders(): Record<string, string> {
  if (API_HEADERS_RAW.trim().length > 0) {
    return parseHeaderJson(API_HEADERS_RAW);
  }

  if (API_KEY.trim().length > 0) {
    return {
      Authorization: `Bearer ${API_KEY}`,
      "X-API-Key": API_KEY,
    };
  }

  return {};
}

async function fetchJson(path: string, options: RequestInit = {}): Promise<JsonValue> {
  const url = `${API_BASE}${path}`;
  const method = options.method ?? "GET";
  logInfo(`HTTP ${method} ${path}`);

  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        "content-type": "application/json",
        ...getAuthHeaders(),
        ...(options.headers ?? {}),
      },
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const fullMessage = `HTTP request failed for ${path}: ${message}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  let text = "";
  try {
    text = await response.text();
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const fullMessage = `Failed reading response body from ${path}: ${message}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  if (!response.ok) {
    const body = formatBody(text);
    const fullMessage = `HTTP ${response.status} ${response.statusText} for ${path}: ${body}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  if (!text) {
    logInfo(`HTTP ${response.status} ${response.statusText} ${path} (empty)`);
    return null;
  }

  try {
    const parsed = JSON.parse(text) as JsonValue;
    logInfo(`HTTP ${response.status} ${response.statusText} ${path}`);
    return parsed;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const body = formatBody(text);
    const fullMessage = `Failed to parse JSON from ${path} (HTTP ${response.status} ${response.statusText}): ${message}. Body: ${body}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }
}

async function fetchText(path: string, options: RequestInit = {}): Promise<string> {
  const url = `${API_BASE}${path}`;
  const method = options.method ?? "GET";
  logInfo(`HTTP ${method} ${path}`);

  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        ...getAuthHeaders(),
        ...(options.headers ?? {}),
      },
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const fullMessage = `HTTP request failed for ${path}: ${message}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  let text = "";
  try {
    text = await response.text();
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const fullMessage = `Failed reading response body from ${path}: ${message}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  if (!response.ok) {
    const body = formatBody(text);
    const fullMessage = `HTTP ${response.status} ${response.statusText} for ${path}: ${body}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }

  logInfo(`HTTP ${response.status} ${response.statusText} ${path}`);
  return text;
}

const server = new Server(
  {
    name: "cocopilot-mcp",
    version: "0.1.0",
  },
  {
    capabilities: {
      tools: {},
      resources: {},
      prompts: {},
    },
  }
);

const tools = [
  {
    name: "coco.project.list",
    description: "List projects from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        limit: { type: "integer" },
        offset: { type: "integer" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.create",
    description: "Create a project in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string" },
        description: { type: "string" },
        status: { type: "string" },
      },
      required: ["name"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.update",
    description: "Update a project in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        name: { type: "string" },
        description: { type: "string" },
        status: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.get",
    description: "Fetch a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.delete",
    description: "Delete a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.config.get",
    description: "Fetch server configuration from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {},
      additionalProperties: false,
    },
  },
  {
    name: "coco.version.get",
    description: "Fetch server version info from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {},
      additionalProperties: false,
    },
  },
  {
    name: "coco.health.get",
    description: "Fetch server health info from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {},
      additionalProperties: false,
    },
  },
  {
    name: "coco.agent.list",
    description: "List agents from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        status: { type: "string" },
        since: { type: "string" },
        limit: { type: "integer" },
        offset: { type: "integer" },
        sort: { type: "string" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "coco.agent.get",
    description: "Fetch agent details from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        agent_id: { type: "string" },
      },
      required: ["agent_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.agent.delete",
    description: "Delete an agent from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        agent_id: { type: "string" },
      },
      required: ["agent_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.tasks.list",
    description: "List tasks for a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        status: { type: "string" },
        type: { type: "string" },
        tag: { type: "string" },
        q: { type: "string" },
        limit: { type: "integer" },
        offset: { type: "integer" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.memory.query",
    description: "Query project memory from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        scope: { type: "string" },
        key: { type: "string" },
        q: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.memory.put",
    description: "Store a project memory item in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        scope: { type: "string" },
        key: { type: "string" },
        value: { type: "object", additionalProperties: true },
        tags: { type: "array", items: { type: "string" } },
      },
      required: ["project_id", "scope", "key", "value"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.audit.list",
    description: "List audit events for a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        type: { type: "string" },
        since: { type: "string" },
        limit: { type: "integer" },
        offset: { type: "integer" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.policy.list",
    description: "List policies for a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        enabled: { type: "boolean" },
        limit: { type: "integer" },
        offset: { type: "integer" },
        sort: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.policy.get",
    description: "Fetch a policy in a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        policy_id: { type: "string" },
      },
      required: ["project_id", "policy_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.policy.create",
    description: "Create a policy in a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        name: { type: "string" },
        description: { type: "string" },
        rules: { type: "array", items: { type: "object" } },
        enabled: { type: "boolean" },
      },
      required: ["project_id", "name"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.policy.update",
    description: "Update a policy in a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        policy_id: { type: "string" },
        name: { type: "string" },
        description: { type: "string" },
        rules: { type: "array", items: { type: "object" } },
        enabled: { type: "boolean" },
      },
      required: ["project_id", "policy_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.policy.delete",
    description: "Delete a policy in a project from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        policy_id: { type: "string" },
      },
      required: ["project_id", "policy_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.events.replay",
    description: "Replay project events since a given event id.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        since_id: { type: "string" },
        limit: { type: "integer" },
      },
      required: ["project_id", "since_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.automation.rules",
    description: "List automation rules for a project.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.automation.simulate",
    description: "Simulate automation for a project event.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        event: {
          type: "object",
          properties: {
            kind: { type: "string" },
            entity_id: { type: "string" },
            payload: { type: "object", additionalProperties: true },
          },
          required: ["kind"],
          additionalProperties: false,
        },
      },
      required: ["project_id", "event"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.project.automation.replay",
    description: "Replay automation for task.completed events since an event id.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        since_event_id: { type: "string" },
        limit: { type: "integer" },
      },
      required: ["project_id", "since_event_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.context_pack.create",
    description: "Create a context pack for a task in a project.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        task_id: { type: ["integer", "string"] },
        query: { type: "string" },
        budget: {
          type: "object",
          properties: {
            max_files: { type: "integer" },
            max_bytes: { type: "integer" },
            max_snippets: { type: "integer" },
          },
          additionalProperties: false,
        },
      },
      required: ["project_id", "task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.context_pack.get",
    description: "Fetch a context pack by id from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        pack_id: { type: ["integer", "string"] },
      },
      required: ["pack_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.create",
    description: "Create a task in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        instructions: { type: "string" },
        title: { type: "string" },
        project_id: { type: "string" },
        type: { type: "string" },
        priority: { type: "string" },
        tags: { type: "array", items: { type: "string" } },
        parent_task_id: { type: "integer" },
      },
      required: ["instructions"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.list",
    description: "List tasks from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        status: { type: "string" },
        limit: { type: "integer" },
        offset: { type: "integer" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.complete",
    description: "Complete a task in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        status: { type: "string" },
        output: { type: "string" },
        message: { type: "string" },
        result: {
          type: "object",
          additionalProperties: true,
        },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.update",
    description: "Update task fields in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        instructions: { type: "string" },
        status: { type: "string" },
        project_id: { type: "string" },
        parent_task_id: { type: "integer" },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.delete",
    description: "Delete a task in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.dependencies.list",
    description: "List task dependencies in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.dependencies.create",
    description: "Create a task dependency in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        depends_on_task_id: { type: ["integer", "string"] },
      },
      required: ["task_id", "depends_on_task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.dependencies.delete",
    description: "Delete a task dependency in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        depends_on_task_id: { type: ["integer", "string"] },
      },
      required: ["task_id", "depends_on_task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.lease.create",
    description: "Create a lease for a task claim in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        agent_id: { type: "string" },
        mode: { type: "string" },
      },
      required: ["task_id", "agent_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.lease.heartbeat",
    description: "Renew a lease in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        lease_id: { type: "string" },
      },
      required: ["lease_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.lease.release",
    description: "Release a lease in the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        lease_id: { type: "string" },
        reason: { type: "string" },
      },
      required: ["lease_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.events.list",
    description: "List events from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        type: { type: "string" },
        since: { type: "string" },
        task_id: { type: ["integer", "string"] },
        project_id: { type: "string" },
        limit: { type: "integer" },
        offset: { type: "integer" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "coco.run.get",
    description: "Fetch run details from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        run_id: { type: "string" },
      },
      required: ["run_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.run.steps",
    description: "Append a run step to the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        run_id: { type: "string" },
        name: { type: "string" },
        status: { type: "string" },
        details: {
          type: "object",
          additionalProperties: true,
        },
      },
      required: ["run_id", "name", "status"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.run.logs",
    description: "Append a run log entry to the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        run_id: { type: "string" },
        stream: { type: "string" },
        chunk: { type: "string" },
        ts: { type: "string" },
      },
      required: ["run_id", "stream", "chunk"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.run.artifacts",
    description: "Append a run artifact to the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        run_id: { type: "string" },
        name: { type: "string" },
        kind: { type: "string" },
        uri: { type: "string" },
        metadata: {
          type: "object",
          additionalProperties: true,
        },
      },
      required: ["run_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.claim",
    description: "Claim the next available task from the Cocopilot API (v2 POST /api/v2/projects/{project_id}/tasks/claim-next).",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string", description: "Project ID (defaults to 'default')" },
        agent_id: { type: "string", description: "Agent ID (defaults to 'mcp-agent')" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.claim.v2",
    description: "Claim a task by id from the Cocopilot API (v2 POST /api/v2/tasks/{taskId}/claim).",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.save",
    description: "Complete a task with output via the Cocopilot API (v2 POST /api/v2/tasks/{task_id}/complete).",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        message: { type: "string" },
      },
      required: ["task_id", "message"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.task.get",
    description: "Fetch a single task by ID from the Cocopilot API.",
    inputSchema: {
      type: "object",
      properties: {
        task_id: { type: ["integer", "string"] },
        project_id: { type: "string" },
      },
      required: ["task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.context_pack.build",
    description: "Build a context pack for a task in a project (alias for coco.context_pack.create).",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        task_id: { type: ["integer", "string"] },
        query: { type: "string" },
        budget: {
          type: "object",
          properties: {
            max_files: { type: "integer" },
            max_bytes: { type: "integer" },
            max_snippets: { type: "integer" },
          },
          additionalProperties: false,
        },
      },
      required: ["project_id", "task_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.memory.get",
    description: "Query project memory from the Cocopilot API (alias for coco.project.memory.query).",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        scope: { type: "string" },
        key: { type: "string" },
        q: { type: "string" },
      },
      required: ["project_id"],
      additionalProperties: false,
    },
  },
  {
    name: "coco.memory.set",
    description: "Store a project memory item in the Cocopilot API (alias for coco.project.memory.put).",
    inputSchema: {
      type: "object",
      properties: {
        project_id: { type: "string" },
        scope: { type: "string" },
        key: { type: "string" },
        value: { type: "object", additionalProperties: true },
        tags: { type: "array", items: { type: "string" } },
      },
      required: ["project_id", "scope", "key", "value"],
      additionalProperties: false,
    },
  },
];

server.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools,
}));

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const name = request.params.name;
  const args = request.params.arguments as ToolArgs;

  logInfo(`Tool request: ${name}`);
  try {

  if (name === "coco.project.list") {
    const query = new URLSearchParams();
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const result = await fetchJson(`/api/v2/projects${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.create") {
    if (typeof args?.name !== "string" || !args.name) {
      throw new Error("name is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {
      name: args.name,
    };

    if (typeof args?.description === "string") {
      payload.description = args.description;
    }
    if (typeof args?.status === "string") {
      payload.status = args.status;
    }

    const result = await fetchJson("/api/v2/projects", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.update") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {};

    if (typeof args?.name === "string") {
      payload.name = args.name;
    }
    if (typeof args?.description === "string") {
      payload.description = args.description;
    }
    if (typeof args?.status === "string") {
      payload.status = args.status;
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.get") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.delete") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.config.get") {
    const result = await fetchJson("/api/v2/config");
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.version.get") {
    const result = await fetchJson("/api/v2/version");
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.health.get") {
    const result = await fetchJson("/api/v2/health");
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.agent.list") {
    const query = new URLSearchParams();
    if (typeof args?.status === "string") {
      query.set("status", args.status);
    }
    if (typeof args?.since === "string") {
      query.set("since", args.since);
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }
    if (typeof args?.sort === "string") {
      query.set("sort", args.sort);
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const result = await fetchJson(`/api/v2/agents${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.agent.get") {
    if (typeof args?.agent_id !== "string" || !args.agent_id) {
      throw new Error("agent_id is required and must be a string.");
    }

    const agentId = encodeURIComponent(args.agent_id);
    const result = await fetchJson(`/api/v2/agents/${agentId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.agent.delete") {
    if (typeof args?.agent_id !== "string" || !args.agent_id) {
      throw new Error("agent_id is required and must be a string.");
    }

    const agentId = encodeURIComponent(args.agent_id);
    const result = await fetchJson(`/api/v2/agents/${agentId}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.tasks.list") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    if (typeof args?.status === "string") {
      query.set("status", args.status);
    }
    if (typeof args?.type === "string") {
      query.set("type", args.type);
    }
    if (typeof args?.tag === "string") {
      query.set("tag", args.tag);
    }
    if (typeof args?.q === "string") {
      query.set("q", args.q);
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/tasks${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.memory.query") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    if (typeof args?.scope === "string") {
      query.set("scope", args.scope);
    }
    if (typeof args?.key === "string") {
      query.set("key", args.key);
    }
    if (typeof args?.q === "string") {
      query.set("q", args.q);
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/memory${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.memory.put") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.scope !== "string" || !args.scope.trim()) {
      throw new Error("scope is required and must be a string.");
    }
    if (typeof args?.key !== "string" || !args.key.trim()) {
      throw new Error("key is required and must be a string.");
    }
    if (!args?.value || typeof args.value !== "object" || Array.isArray(args.value)) {
      throw new Error("value is required and must be an object.");
    }

    const payload: Record<string, JsonValue> = {
      scope: args.scope,
      key: args.key,
      value: args.value as JsonValue,
    };

    if (Array.isArray(args?.tags)) {
      const tags: string[] = [];
      for (const tag of args.tags) {
        if (typeof tag !== "string") {
          throw new Error("tags must be an array of strings.");
        }
        if (tag.trim().length > 0) {
          tags.push(tag);
        }
      }
      if (tags.length > 0) {
        payload.source_refs = tags;
      }
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/memory`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.audit.list") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    if (typeof args?.type === "string") {
      query.set("type", args.type);
    }
    if (typeof args?.since === "string") {
      query.set("since", args.since);
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/audit${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.policy.list") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    if (typeof args?.enabled === "boolean") {
      query.set("enabled", args.enabled ? "true" : "false");
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }
    if (typeof args?.sort === "string" && args.sort.length > 0) {
      query.set("sort", args.sort);
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/policies${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.policy.get") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.policy_id !== "string" || !args.policy_id) {
      throw new Error("policy_id is required and must be a string.");
    }

    const projectId = encodeURIComponent(args.project_id);
    const policyId = encodeURIComponent(args.policy_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/policies/${policyId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.policy.create") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.name !== "string" || !args.name) {
      throw new Error("name is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {
      name: args.name,
    };

    if (typeof args?.description === "string") {
      payload.description = args.description;
    }
    if (Array.isArray(args?.rules)) {
      payload.rules = args.rules as JsonValue;
    }
    if (typeof args?.enabled === "boolean") {
      payload.enabled = args.enabled;
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/policies`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.policy.update") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.policy_id !== "string" || !args.policy_id) {
      throw new Error("policy_id is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {};

    if (typeof args?.name === "string") {
      payload.name = args.name;
    }
    if (typeof args?.description === "string") {
      payload.description = args.description;
    }
    if (Array.isArray(args?.rules)) {
      payload.rules = args.rules as JsonValue;
    }
    if (typeof args?.enabled === "boolean") {
      payload.enabled = args.enabled;
    }

    const projectId = encodeURIComponent(args.project_id);
    const policyId = encodeURIComponent(args.policy_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/policies/${policyId}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.policy.delete") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.policy_id !== "string" || !args.policy_id) {
      throw new Error("policy_id is required and must be a string.");
    }

    const projectId = encodeURIComponent(args.project_id);
    const policyId = encodeURIComponent(args.policy_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/policies/${policyId}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.automation.rules") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/automation/rules`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.automation.simulate") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (!args?.event || typeof args.event !== "object" || Array.isArray(args.event)) {
      throw new Error("event is required and must be an object.");
    }
    if (typeof args.event.kind !== "string" || !args.event.kind) {
      throw new Error("event.kind is required and must be a string.");
    }

    const event: Record<string, JsonValue> = {
      kind: args.event.kind,
    };

    if (typeof args.event.entity_id === "string") {
      event.entity_id = args.event.entity_id;
    }
    if (
      args.event.payload &&
      typeof args.event.payload === "object" &&
      !Array.isArray(args.event.payload)
    ) {
      event.payload = args.event.payload as JsonValue;
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/automation/simulate`, {
      method: "POST",
      body: JSON.stringify({ event }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.automation.replay") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.since_event_id !== "string" || !args.since_event_id) {
      throw new Error("since_event_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    query.set("since_event_id", args.since_event_id);
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }

    const projectId = encodeURIComponent(args.project_id);
    const suffix = `?${query.toString()}`;
    const result = await fetchJson(`/api/v2/projects/${projectId}/automation/replay${suffix}`, {
      method: "POST",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.project.events.replay") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.since_id !== "string" || !args.since_id) {
      throw new Error("since_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    query.set("since_id", args.since_id);
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }

    const projectId = encodeURIComponent(args.project_id);
    const suffix = `?${query.toString()}`;
    const result = await fetchJson(`/api/v2/projects/${projectId}/events/replay${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.context_pack.create") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const payload: Record<string, JsonValue> = {
      task_id: args.task_id,
    };

    if (typeof args?.query === "string") {
      payload.query = args.query;
    }
    if (args?.budget && typeof args.budget === "object" && !Array.isArray(args.budget)) {
      payload.budget = args.budget as JsonValue;
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/context-packs`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.context_pack.get") {
    if (
      typeof args?.pack_id !== "number" &&
      (typeof args?.pack_id !== "string" || args.pack_id.length === 0)
    ) {
      throw new Error("pack_id is required and must be a number or string.");
    }

    const packId = encodeURIComponent(String(args.pack_id));
    const result = await fetchJson(`/api/v2/context-packs/${packId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.create") {
    const payload = args ?? {};
    const result = await fetchJson("/api/v2/tasks", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.list") {
    const query = new URLSearchParams();
    if (typeof args?.project_id === "string") {
      query.set("project_id", args.project_id);
    }
    if (typeof args?.status === "string") {
      query.set("status", args.status);
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const result = await fetchJson(`/api/v2/tasks${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.complete") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const payload: Record<string, JsonValue> = {};

    if (typeof args?.status === "string") {
      payload.status = args.status;
    }
    if (typeof args?.output === "string") {
      payload.output = args.output;
    }
    if (typeof args?.message === "string") {
      payload.message = args.message;
    }
    if (args?.result && typeof args.result === "object" && !Array.isArray(args.result)) {
      payload.result = args.result as JsonValue;
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}/complete`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.update") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const payload: Record<string, JsonValue> = {};

    if (typeof args?.instructions === "string") {
      payload.instructions = args.instructions;
    }
    if (typeof args?.status === "string") {
      payload.status = args.status;
    }
    if (typeof args?.project_id === "string") {
      payload.project_id = args.project_id;
    }
    if (typeof args?.parent_task_id === "number") {
      payload.parent_task_id = args.parent_task_id;
    }

    if (Object.keys(payload).length === 0) {
      throw new Error("At least one field is required.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.delete") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}`, {
      method: "DELETE",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.dependencies.list") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}/dependencies`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.dependencies.create") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }
    if (
      typeof args?.depends_on_task_id !== "number" &&
      (typeof args?.depends_on_task_id !== "string" || args.depends_on_task_id.length === 0)
    ) {
      throw new Error("depends_on_task_id is required and must be a number or string.");
    }

    const payload: Record<string, JsonValue> = {
      depends_on_task_id: args.depends_on_task_id,
    };

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}/dependencies`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.dependencies.delete") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }
    if (
      typeof args?.depends_on_task_id !== "number" &&
      (typeof args?.depends_on_task_id !== "string" || args.depends_on_task_id.length === 0)
    ) {
      throw new Error("depends_on_task_id is required and must be a number or string.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const dependsOnTaskId = encodeURIComponent(String(args.depends_on_task_id));
    const result = await fetchJson(
      `/api/v2/tasks/${taskId}/dependencies/${dependsOnTaskId}`,
      {
        method: "DELETE",
      }
    );
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.lease.create") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }
    if (typeof args?.agent_id !== "string" || !args.agent_id) {
      throw new Error("agent_id is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {
      task_id: args.task_id,
      agent_id: args.agent_id,
    };

    if (typeof args?.mode === "string" && args.mode.length > 0) {
      payload.mode = args.mode;
    }

    const result = await fetchJson("/api/v2/leases", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.lease.heartbeat") {
    if (typeof args?.lease_id !== "string" || !args.lease_id) {
      throw new Error("lease_id is required and must be a string.");
    }

    const leaseId = encodeURIComponent(args.lease_id);
    const result = await fetchJson(`/api/v2/leases/${leaseId}/heartbeat`, {
      method: "POST",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.lease.release") {
    if (typeof args?.lease_id !== "string" || !args.lease_id) {
      throw new Error("lease_id is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {};
    if (typeof args?.reason === "string" && args.reason.length > 0) {
      payload.reason = args.reason;
    }

    const leaseId = encodeURIComponent(args.lease_id);
    const result = await fetchJson(`/api/v2/leases/${leaseId}/release`, {
      method: "POST",
      body: Object.keys(payload).length ? JSON.stringify(payload) : undefined,
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.events.list") {
    const query = new URLSearchParams();
    if (typeof args?.type === "string") {
      query.set("type", args.type);
    }
    if (typeof args?.since === "string") {
      query.set("since", args.since);
    }
    if (typeof args?.task_id === "number") {
      query.set("task_id", String(args.task_id));
    }
    if (typeof args?.task_id === "string" && args.task_id.length > 0) {
      query.set("task_id", args.task_id);
    }
    if (typeof args?.project_id === "string") {
      query.set("project_id", args.project_id);
    }
    if (typeof args?.limit === "number") {
      query.set("limit", String(args.limit));
    }
    if (typeof args?.offset === "number") {
      query.set("offset", String(args.offset));
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const result = await fetchJson(`/api/v2/events${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.run.get") {
    if (typeof args?.run_id !== "string" || !args.run_id) {
      throw new Error("run_id is required and must be a string.");
    }

    const runId = encodeURIComponent(args.run_id);
    const result = await fetchJson(`/api/v2/runs/${runId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.run.steps") {
    if (typeof args?.run_id !== "string" || !args.run_id) {
      throw new Error("run_id is required and must be a string.");
    }
    if (typeof args?.name !== "string" || !args.name) {
      throw new Error("name is required and must be a string.");
    }
    if (typeof args?.status !== "string" || !args.status) {
      throw new Error("status is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {
      name: args.name,
      status: args.status,
    };

    if (args?.details && typeof args.details === "object" && !Array.isArray(args.details)) {
      payload.details = args.details as JsonValue;
    }

    const runId = encodeURIComponent(args.run_id);
    const result = await fetchJson(`/api/v2/runs/${runId}/steps`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.run.logs") {
    if (typeof args?.run_id !== "string" || !args.run_id) {
      throw new Error("run_id is required and must be a string.");
    }
    if (typeof args?.stream !== "string" || !args.stream) {
      throw new Error("stream is required and must be a string.");
    }
    if (typeof args?.chunk !== "string" || !args.chunk) {
      throw new Error("chunk is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {
      stream: args.stream,
      chunk: args.chunk,
    };

    if (typeof args?.ts === "string" && args.ts.length > 0) {
      payload.ts = args.ts;
    }

    const runId = encodeURIComponent(args.run_id);
    const result = await fetchText(`/api/v2/runs/${runId}/logs`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
      },
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: result || "OK" }],
    };
  }

  if (name === "coco.run.artifacts") {
    if (typeof args?.run_id !== "string" || !args.run_id) {
      throw new Error("run_id is required and must be a string.");
    }

    const payload: Record<string, JsonValue> = {};

    if (typeof args?.name === "string" && args.name.length > 0) {
      payload.name = args.name;
    }
    if (typeof args?.kind === "string" && args.kind.length > 0) {
      payload.kind = args.kind;
    }
    if (typeof args?.uri === "string" && args.uri.length > 0) {
      payload.uri = args.uri;
    }
    if (args?.metadata && typeof args.metadata === "object" && !Array.isArray(args.metadata)) {
      payload.metadata = args.metadata as JsonValue;
    }

    const runId = encodeURIComponent(args.run_id);
    const result = await fetchJson(`/api/v2/runs/${runId}/artifacts`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.claim") {
    const projectId = encodeURIComponent(String(args?.project_id || "default"));
    const agentId = String(args?.agent_id || "mcp-agent");
    const result = await fetchJson(`/api/v2/projects/${projectId}/tasks/claim-next`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ agent_id: agentId }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.claim.v2") {
    if (args?.task_id === undefined || args.task_id === null || args.task_id === "") {
      throw new Error("task_id is required.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}/claim`, {
      method: "POST",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.save") {
    if (args?.task_id === undefined || args.task_id === null || args.task_id === "") {
      throw new Error("task_id is required.");
    }
    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}/complete`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        output: typeof args?.message === "string" ? args.message : "",
        result: { summary: typeof args?.message === "string" ? args.message : "" },
      }),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.task.get") {
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const taskId = encodeURIComponent(String(args.task_id));
    const result = await fetchJson(`/api/v2/tasks/${taskId}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.context_pack.build") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (
      typeof args?.task_id !== "number" &&
      (typeof args?.task_id !== "string" || args.task_id.length === 0)
    ) {
      throw new Error("task_id is required and must be a number or string.");
    }

    const payload: Record<string, JsonValue> = {
      task_id: args.task_id,
    };

    if (typeof args?.query === "string") {
      payload.query = args.query;
    }
    if (args?.budget && typeof args.budget === "object" && !Array.isArray(args.budget)) {
      payload.budget = args.budget as JsonValue;
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/context-packs`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.memory.get") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }

    const query = new URLSearchParams();
    if (typeof args?.scope === "string") {
      query.set("scope", args.scope);
    }
    if (typeof args?.key === "string") {
      query.set("key", args.key);
    }
    if (typeof args?.q === "string") {
      query.set("q", args.q);
    }

    const suffix = query.toString() ? `?${query.toString()}` : "";
    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/memory${suffix}`);
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  if (name === "coco.memory.set") {
    if (typeof args?.project_id !== "string" || !args.project_id) {
      throw new Error("project_id is required and must be a string.");
    }
    if (typeof args?.scope !== "string" || !args.scope.trim()) {
      throw new Error("scope is required and must be a string.");
    }
    if (typeof args?.key !== "string" || !args.key.trim()) {
      throw new Error("key is required and must be a string.");
    }
    if (!args?.value || typeof args.value !== "object" || Array.isArray(args.value)) {
      throw new Error("value is required and must be an object.");
    }

    const payload: Record<string, JsonValue> = {
      scope: args.scope,
      key: args.key,
      value: args.value as JsonValue,
    };

    if (Array.isArray(args?.tags)) {
      const tags: string[] = [];
      for (const tag of args.tags) {
        if (typeof tag !== "string") {
          throw new Error("tags must be an array of strings.");
        }
        if (tag.trim().length > 0) {
          tags.push(tag);
        }
      }
      if (tags.length > 0) {
        payload.source_refs = tags;
      }
    }

    const projectId = encodeURIComponent(args.project_id);
    const result = await fetchJson(`/api/v2/projects/${projectId}/memory`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    return {
      content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
    };
  }

  throw new Error(`Unknown tool: ${name}`);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const fullMessage = `Tool ${name} failed: ${message}`;
    logError(fullMessage);
    throw new Error(fullMessage);
  }
});

// --- MCP Resources ---
server.setRequestHandler(ListResourcesRequestSchema, async () => ({
  resources: [
    { uri: "coco://projects", name: "Projects List", description: "List all projects" },
    { uri: "coco://health", name: "Server Health", description: "Server health status" },
    { uri: "coco://config", name: "Server Config", description: "Server configuration" },
  ],
}));

server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  const uri = request.params.uri;
  if (uri === "coco://projects") {
    const resp = await fetchJson("/api/v2/projects");
    return { contents: [{ uri, mimeType: "application/json", text: JSON.stringify(resp) }] };
  }
  if (uri === "coco://health") {
    const resp = await fetchJson("/api/v2/health");
    return { contents: [{ uri, mimeType: "application/json", text: JSON.stringify(resp) }] };
  }
  if (uri === "coco://config") {
    const resp = await fetchJson("/api/v2/config");
    return { contents: [{ uri, mimeType: "application/json", text: JSON.stringify(resp) }] };
  }
  throw new Error(`Unknown resource: ${uri}`);
});

// --- MCP Prompts ---
server.setRequestHandler(ListPromptsRequestSchema, async () => ({
  prompts: [
    {
      name: "create-task",
      description: "Create a new task with instructions",
      arguments: [
        { name: "project_id", description: "Project ID", required: true },
        { name: "instructions", description: "Task instructions", required: true },
      ],
    },
    {
      name: "review-task",
      description: "Review and complete a task",
      arguments: [
        { name: "task_id", description: "Task ID to review", required: true },
      ],
    },
  ],
}));

server.setRequestHandler(GetPromptRequestSchema, async (request) => {
  const { name, arguments: promptArgs } = request.params;
  if (name === "create-task") {
    return {
      messages: [
        {
          role: "user",
          content: {
            type: "text",
            text: `Create a new task in project ${promptArgs?.project_id} with these instructions: ${promptArgs?.instructions}`,
          },
        },
      ],
    };
  }
  if (name === "review-task") {
    return {
      messages: [
        {
          role: "user",
          content: {
            type: "text",
            text: `Review task ${promptArgs?.task_id}, check its current status, and provide completion output if ready.`,
          },
        },
      ],
    };
  }
  throw new Error(`Unknown prompt: ${name}`);
});

const transport = new StdioServerTransport();
await server.connect(transport);
console.error("cocopilot-mcp server running");
