import type {
  AskHumanOption,
  SessionContentPart,
  SessionDetail,
  SessionMessage,
  SessionMessagePage,
  SessionMessageRole,
  SessionRuntimeSelection,
  SessionMetadata,
  ProviderType,
  SessionToolCall,
  SessionToolResult,
  SessionToolResultStatus,
  SessionTurnDraft,
  SessionTurnDraftPendingQuestion,
  SessionTurnDraftSegment,
  SessionTurnDraftStatus,
  SessionTurnDraftTool,
} from "../mobileTypes";
import {
  parseArray,
  parseOptionalString,
  requireArray,
  requireBoolean,
  requireInteger,
  requireNullableInteger,
  requireRecord,
  requireString,
} from "./payloadValidators";

const SESSION_MESSAGE_ROLES = new Set<SessionMessageRole>(["system", "internal", "user", "assistant", "tool"]);
const SESSION_TOOL_RESULT_STATUSES = new Set<SessionToolResultStatus>(["success", "error"]);
const SESSION_RUNTIME_SELECTION_RUNTIMES = new Set<SessionRuntimeSelection["runtime"]>(["ghost", "codex"]);
const SESSION_RUNTIME_SELECTION_PROVIDER_TYPES = new Set<ProviderType>(["openai", "anthropic", "custom", "codex"]);
const SESSION_RUNTIME_SELECTION_MODES = new Set<NonNullable<SessionRuntimeSelection["mode"]>>(["default", "plan"]);
const SESSION_TURN_DRAFT_STATUSES = new Set<SessionTurnDraftStatus>(["streaming", "awaiting_human", "error"]);

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
    turn_draft: parseOptionalSessionTurnDraft(
      object.turn_draft,
      "SESSION_GET payload.turn_draft",
    ),
    last_runtime_selection: parseOptionalSessionRuntimeSelection(
      object.last_runtime_selection,
      "SESSION_GET payload.last_runtime_selection",
    ),
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

function parseOptionalSessionRuntimeSelection(
  payload: unknown,
  path: string,
): SessionRuntimeSelection | null | undefined {
  if (payload === undefined) {
    return undefined;
  }
  if (payload === null) {
    return null;
  }
  const object = requireRecord(payload, path);
  const selection: SessionRuntimeSelection = {
    runtime: requireRuntimeSelectionRuntime(object.runtime, `${path}.runtime`),
  };
  if (object.provider !== undefined) {
    selection.provider = requireString(object.provider, `${path}.provider`);
  }
  if (object.provider_type !== undefined) {
    selection.provider_type = requireProviderType(object.provider_type, `${path}.provider_type`);
  }
  if (object.model !== undefined) {
    selection.model = requireString(object.model, `${path}.model`);
  }
  if (object.mode !== undefined) {
    selection.mode = requireRuntimeSelectionMode(object.mode, `${path}.mode`);
  }
  return selection;
}

function parseOptionalSessionTurnDraft(
  payload: unknown,
  path: string,
): SessionTurnDraft | null | undefined {
  if (payload === undefined) {
    return undefined;
  }
  if (payload === null) {
    return null;
  }

  const object = requireRecord(payload, path);
  const status = requireSessionTurnDraftStatus(object.status, `${path}.status`);
  const error = parseOptionalString(object.error, `${path}.error`);
  if (status === "error" && !error) {
    throw new Error(`${path}.error must be a string when status=error`);
  }

  return {
    trace_id: requireString(object.trace_id, `${path}.trace_id`),
    turn: requireInteger(object.turn, `${path}.turn`),
    status,
    error,
    pending_questions: parseArray(object.pending_questions, `${path}.pending_questions`, parseSessionTurnDraftPendingQuestion),
    assistant_segments: parseArray(object.assistant_segments, `${path}.assistant_segments`, parseSessionTurnDraftSegment),
    thinking_segments: parseArray(object.thinking_segments, `${path}.thinking_segments`, parseSessionTurnDraftSegment),
    tools: parseArray(object.tools, `${path}.tools`, parseSessionTurnDraftTool),
    item_order: parseArray(object.item_order, `${path}.item_order`, requireString),
  };
}

function parseSessionTurnDraftSegment(payload: unknown, path: string): SessionTurnDraftSegment {
  const object = requireRecord(payload, path);
  return {
    id: requireString(object.id, `${path}.id`),
    content: requireString(object.content, `${path}.content`),
  };
}

function parseSessionTurnDraftTool(payload: unknown, path: string): SessionTurnDraftTool {
  const object = requireRecord(payload, path);
  return {
    id: requireString(object.id, `${path}.id`),
    content: requireString(object.content, `${path}.content`),
    tool_input: parseOptionalString(object.tool_input, `${path}.tool_input`),
    tool_name: parseOptionalString(object.tool_name, `${path}.tool_name`),
    tool_status: parseOptionalString(object.tool_status, `${path}.tool_status`),
    tool_call_id: parseOptionalString(object.tool_call_id, `${path}.tool_call_id`),
    trace_id: parseOptionalString(object.trace_id, `${path}.trace_id`),
  };
}

function parseSessionTurnDraftPendingQuestion(
  payload: unknown,
  path: string,
): SessionTurnDraftPendingQuestion {
  const object = requireRecord(payload, path);
  return {
    question_id: requireString(object.question_id, `${path}.question_id`),
    prompt: requireString(object.prompt, `${path}.prompt`),
    selection_mode: parseOptionalSelectionMode(object.selection_mode, `${path}.selection_mode`),
    options: object.options === undefined
      ? undefined
      : parseArray(object.options, `${path}.options`, parseAskHumanOption),
  };
}

function parseAskHumanOption(payload: unknown, path: string): AskHumanOption {
  const object = requireRecord(payload, path);
  const option: AskHumanOption = {
    label: requireString(object.label, `${path}.label`),
  };
  if (object.allow_custom !== undefined) {
    option.allow_custom = requireBoolean(object.allow_custom, `${path}.allow_custom`);
  }
  return option;
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

function requireRuntimeSelectionRuntime(value: unknown, path: string): SessionRuntimeSelection["runtime"] {
  if (typeof value !== "string" || !SESSION_RUNTIME_SELECTION_RUNTIMES.has(value as SessionRuntimeSelection["runtime"])) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_RUNTIME_SELECTION_RUNTIMES).join(", ")}`);
  }
  return value as SessionRuntimeSelection["runtime"];
}

function requireProviderType(value: unknown, path: string): ProviderType {
  if (typeof value !== "string" || !SESSION_RUNTIME_SELECTION_PROVIDER_TYPES.has(value as ProviderType)) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_RUNTIME_SELECTION_PROVIDER_TYPES).join(", ")}`);
  }
  return value as ProviderType;
}

function requireRuntimeSelectionMode(value: unknown, path: string): NonNullable<SessionRuntimeSelection["mode"]> {
  if (typeof value !== "string" || !SESSION_RUNTIME_SELECTION_MODES.has(value as NonNullable<SessionRuntimeSelection["mode"]>)) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_RUNTIME_SELECTION_MODES).join(", ")}`);
  }
  return value as NonNullable<SessionRuntimeSelection["mode"]>;
}

function parseOptionalSelectionMode(
  value: unknown,
  path: string,
): SessionTurnDraftPendingQuestion["selection_mode"] {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string" || (value !== "single" && value !== "multiple")) {
    throw new Error(`${path} must be "single" or "multiple"`);
  }
  return value;
}

function requireSessionTurnDraftStatus(value: unknown, path: string): SessionTurnDraftStatus {
  if (typeof value !== "string" || !SESSION_TURN_DRAFT_STATUSES.has(value as SessionTurnDraftStatus)) {
    throw new Error(`${path} must be one of ${Array.from(SESSION_TURN_DRAFT_STATUSES).join(", ")}`);
  }
  return value as SessionTurnDraftStatus;
}
