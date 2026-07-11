import type {
  SessionContentPart,
  SessionDetail,
  SessionMessage,
  SessionMessagePage,
  SessionMessageRole,
  SessionMetadata,
  SessionToolCall,
  SessionToolResult,
  SessionToolResultStatus,
} from "../mobileTypes";

const SESSION_MESSAGE_ROLES = new Set<SessionMessageRole>(["system", "internal", "user", "assistant", "tool"]);
const SESSION_TOOL_RESULT_STATUSES = new Set<SessionToolResultStatus>(["success", "error"]);

export function parseSessionMetadataList(payload: unknown): SessionMetadata[] {
  if (!Array.isArray(payload)) {
    throw new Error("SESSIONS_LIST payload must be an array");
  }
  return payload.map((item, index) => parseSessionMetadata(item, `SESSIONS_LIST payload[${index}]`));
}

export function parseSessionDetail(payload: unknown): SessionDetail {
  const metadata = parseSessionMetadata(payload, "SESSION_GET payload");
  const object = requireRecord(payload, "SESSION_GET payload");
  return {
    ...metadata,
    messages: requireArray(object.messages, "SESSION_GET payload.messages").map((item, index) =>
      parseSessionMessage(item, `SESSION_GET payload.messages[${index}]`),
    ),
    page: parseSessionMessagePage(object.page, "SESSION_GET payload.page"),
  };
}

function parseSessionMetadata(payload: unknown, path: string): SessionMetadata {
  const object = requireRecord(payload, path);
  return {
    id: requireString(object.id, `${path}.id`),
    title: requireString(object.title, `${path}.title`),
    created_at: requireString(object.created_at, `${path}.created_at`),
    updated_at: requireString(object.updated_at, `${path}.updated_at`),
    message_count: requireInteger(object.message_count, `${path}.message_count`),
    token_count: requireInteger(object.token_count, `${path}.token_count`),
  };
}

function parseSessionMessage(payload: unknown, path: string): SessionMessage {
  const object = requireRecord(payload, path);
  const message: SessionMessage = {
    index: requireInteger(object.index, `${path}.index`),
    role: requireSessionMessageRole(object.role, `${path}.role`),
  };

  if (object.text !== undefined) {
    message.text = requireString(object.text, `${path}.text`);
  }
  if (object.content !== undefined) {
    message.content = requireArray(object.content, `${path}.content`).map((item, index) =>
      parseSessionContentPart(item, `${path}.content[${index}]`),
    );
  }
  if (object.tool_calls !== undefined) {
    message.tool_calls = requireArray(object.tool_calls, `${path}.tool_calls`).map((item, index) =>
      parseSessionToolCall(item, `${path}.tool_calls[${index}]`),
    );
  }
  if (object.tool_result !== undefined) {
    message.tool_result = parseNullableSessionToolResult(object.tool_result, `${path}.tool_result`);
  }
  if (object.tool_call_id !== undefined) {
    message.tool_call_id = requireString(object.tool_call_id, `${path}.tool_call_id`);
  }
  if (object.in_progress !== undefined) {
    message.in_progress = requireBoolean(object.in_progress, `${path}.in_progress`);
  }
  if (object.thinking !== undefined) {
    message.thinking = requireString(object.thinking, `${path}.thinking`);
  }
  return message;
}

function parseSessionContentPart(payload: unknown, path: string): SessionContentPart {
  const object = requireRecord(payload, path);
  const part: SessionContentPart = {
    type: requireString(object.type, `${path}.type`),
  };
  if (object.text !== undefined) {
    part.text = requireString(object.text, `${path}.text`);
  }
  return part;
}

function parseSessionToolCall(payload: unknown, path: string): SessionToolCall {
  const object = requireRecord(payload, path);
  return {
    arguments: requireRecord(object.arguments, `${path}.arguments`),
    id: requireString(object.id, `${path}.id`),
    name: requireString(object.name, `${path}.name`),
  };
}

function parseNullableSessionToolResult(payload: unknown, path: string): SessionToolResult | null {
  if (payload === null) {
    return null;
  }
  const object = requireRecord(payload, path);
  const result: SessionToolResult = {
    status: requireSessionToolResultStatus(object.status, `${path}.status`),
    tool: requireString(object.tool, `${path}.tool`),
  };
  if (object.trace_id !== undefined) {
    result.trace_id = requireString(object.trace_id, `${path}.trace_id`);
  }
  if (object.output !== undefined) {
    result.output = requireString(object.output, `${path}.output`);
  }
  if (object.error !== undefined) {
    result.error = requireString(object.error, `${path}.error`);
  }
  return result;
}

function parseSessionMessagePage(payload: unknown, path: string): SessionMessagePage {
  const object = requireRecord(payload, path);
  const page: SessionMessagePage = {
    limit: requireInteger(object.limit, `${path}.limit`),
    has_more_before: requireBoolean(object.has_more_before, `${path}.has_more_before`),
  };

  if (object.before !== undefined) {
    page.before = requireNullableInteger(object.before, `${path}.before`);
  }
  if (object.start_index !== undefined) {
    page.start_index = requireNullableInteger(object.start_index, `${path}.start_index`);
  }
  if (object.end_index !== undefined) {
    page.end_index = requireNullableInteger(object.end_index, `${path}.end_index`);
  }
  if (object.next_before !== undefined) {
    page.next_before = requireNullableInteger(object.next_before, `${path}.next_before`);
  }
  return page;
}

function requireRecord(value: unknown, path: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${path} must be an object`);
  }
  return value as Record<string, unknown>;
}

function requireArray(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`);
  }
  return value;
}

function requireString(value: unknown, path: string): string {
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string`);
  }
  return value;
}

function requireInteger(value: unknown, path: string): number {
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw new Error(`${path} must be an integer`);
  }
  return value;
}

function requireNullableInteger(value: unknown, path: string): number | null {
  if (value === null) {
    return null;
  }
  return requireInteger(value, path);
}

function requireBoolean(value: unknown, path: string): boolean {
  if (typeof value !== "boolean") {
    throw new Error(`${path} must be a boolean`);
  }
  return value;
}

function requireSessionMessageRole(value: unknown, path: string): SessionMessageRole {
  if (typeof value !== "string" || !SESSION_MESSAGE_ROLES.has(value as SessionMessageRole)) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_MESSAGE_ROLES).join(", ")}`);
  }
  return value as SessionMessageRole;
}

function requireSessionToolResultStatus(value: unknown, path: string): SessionToolResultStatus {
  if (typeof value !== "string" || !SESSION_TOOL_RESULT_STATUSES.has(value as SessionToolResultStatus)) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_TOOL_RESULT_STATUSES).join(", ")}`);
  }
  return value as SessionToolResultStatus;
}
