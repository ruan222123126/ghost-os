import type {
  SessionContentPart,
  SessionDetail,
  SessionFileContent,
  SessionHumanInteraction,
  SessionImageContent,
  SessionMessage,
  SessionMessagePage,
  SessionMetadata,
  SessionSourceAssignment,
  SessionSourceResolution,
  SessionSidebarPartition,
  SessionSidebarPartitionState,
  SessionToolCall,
  SessionToolResult,
} from '@/lib/types';
import {
  defineStringEnumValues,
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalAskHumanOptions,
  parseOptionalBoolean,
  parseOptionalNumber,
  parseOptionalSelectionMode,
  parseOptionalString,
} from '@/lib/api/shared';
import { parseOptionalSessionTurnDraft } from './parser.turnDraft';

const SESSION_ROLES = defineStringEnumValues<SessionMessage['role']>({
  system: true,
  internal: true,
  user: true,
  assistant: true,
  tool: true,
});
const TOOL_RESULT_STATUSES = defineStringEnumValues<SessionToolResult['status']>({
  success: true,
  error: true,
});
const SESSION_SOURCE_KINDS = defineStringEnumValues<SessionSourceAssignment['kind']>({
  workflow: true,
  orchestration: true,
  loop: true,
  task: true,
});

function parseSessionImageContent(
  value: unknown,
  label: string,
): SessionImageContent | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, label);

  return {
    path: parseOptionalString(record.path, `${label}.path`),
    url: parseOptionalString(record.url, `${label}.url`),
    mime_type: parseOptionalString(record.mime_type, `${label}.mime_type`),
    width: parseOptionalNumber(record.width, `${label}.width`),
    height: parseOptionalNumber(record.height, `${label}.height`),
    sha256: parseOptionalString(record.sha256, `${label}.sha256`),
    bytes: parseOptionalNumber(record.bytes, `${label}.bytes`),
  };
}

function parseSessionFileContent(
  value: unknown,
  label: string,
): SessionFileContent | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, label);

  return {
    artifact_id: expectString(record.artifact_id, `${label}.artifact_id`),
    name: expectString(record.name, `${label}.name`),
    mime_type: parseOptionalString(record.mime_type, `${label}.mime_type`),
    bytes: parseOptionalNumber(record.bytes, `${label}.bytes`),
    sha256: parseOptionalString(record.sha256, `${label}.sha256`),
    download_url: expectString(record.download_url, `${label}.download_url`),
    source_path: parseOptionalString(record.source_path, `${label}.source_path`),
    note: parseOptionalString(record.note, `${label}.note`),
  };
}

function parseSessionContentPart(value: unknown, label: string): SessionContentPart {
  const record = expectRecord(value, label);

  return {
    type: expectString(record.type, `${label}.type`),
    text: parseOptionalString(record.text, `${label}.text`),
    image: parseSessionImageContent(record.image, `${label}.image`),
    file: parseSessionFileContent(record.file, `${label}.file`),
  };
}

function parseOptionalSessionContent(
  value: unknown,
  label: string,
): SessionContentPart[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }

  return value.map((part, index) => parseSessionContentPart(part, `${label}[${index}]`));
}

function parseSessionToolCall(value: unknown, label: string): SessionToolCall {
  const record = expectRecord(value, label);

  return {
    id: expectString(record.id, `${label}.id`),
    name: expectString(record.name, `${label}.name`),
    arguments: expectRecord(record.arguments, `${label}.arguments`),
  };
}

function parseOptionalSessionToolCalls(
  value: unknown,
  label: string,
): SessionToolCall[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }

  return value.map((call, index) => parseSessionToolCall(call, `${label}[${index}]`));
}

function parseOptionalSessionToolResult(
  value: unknown,
  label: string,
): SessionToolResult | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, label);

  return {
    status: expectStringEnum(record.status, TOOL_RESULT_STATUSES, `${label}.status`),
    tool: expectString(record.tool, `${label}.tool`),
    trace_id: parseOptionalString(record.trace_id, `${label}.trace_id`),
    output: parseOptionalString(record.output, `${label}.output`),
    error: parseOptionalString(record.error, `${label}.error`),
  };
}

function parseOptionalSessionHumanInteraction(
  value: unknown,
  label: string,
): SessionHumanInteraction | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, label);

  return {
    question_id: expectString(record.question_id, `${label}.question_id`),
    prompt: expectString(record.prompt, `${label}.prompt`),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, `${label}.selection_mode`),
    options: parseOptionalAskHumanOptions(record.options, `${label}.options`),
    answer: parseOptionalString(record.answer, `${label}.answer`),
  };
}

function parseSessionMessage(value: unknown, label: string): SessionMessage {
  const record = expectRecord(value, label);

  return {
    index: expectNumber(record.index, `${label}.index`),
    role: expectStringEnum(record.role, SESSION_ROLES, `${label}.role`),
    text: parseOptionalString(record.text, `${label}.text`),
    content: parseOptionalSessionContent(record.content, `${label}.content`),
    tool_calls: parseOptionalSessionToolCalls(record.tool_calls, `${label}.tool_calls`),
    tool_result: parseOptionalSessionToolResult(record.tool_result, `${label}.tool_result`),
    human_interaction: parseOptionalSessionHumanInteraction(
      record.human_interaction,
      `${label}.human_interaction`,
    ),
    tool_call_id: parseOptionalString(record.tool_call_id, `${label}.tool_call_id`),
    in_progress: parseOptionalBoolean(record.in_progress, `${label}.in_progress`),
    thinking: parseOptionalString(record.thinking, `${label}.thinking`),
  };
}

function parseNullableNumber(value: unknown, label: string): number | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }
  return expectNumber(value, label);
}

function parseSessionPage(value: unknown, label: string): SessionMessagePage {
  const record = expectRecord(value, label);

  return {
    limit: expectNumber(record.limit, `${label}.limit`),
    before: parseNullableNumber(record.before, `${label}.before`),
    start_index: parseNullableNumber(record.start_index, `${label}.start_index`),
    end_index: parseNullableNumber(record.end_index, `${label}.end_index`),
    has_more_before: expectBoolean(record.has_more_before, `${label}.has_more_before`),
    next_before: parseNullableNumber(record.next_before, `${label}.next_before`),
  };
}

function parseSessionMetadata(value: unknown, label: string): SessionMetadata {
  const record = expectRecord(value, label);

  return {
    id: expectString(record.id, `${label}.id`),
    title: expectString(record.title, `${label}.title`),
    created_at: expectString(record.created_at, `${label}.created_at`),
    updated_at: expectString(record.updated_at, `${label}.updated_at`),
    message_count: expectNumber(record.message_count, `${label}.message_count`),
    token_count: expectNumber(record.token_count, `${label}.token_count`),
  };
}

function parseSessionSidebarPartition(
  value: unknown,
  label: string,
): SessionSidebarPartition {
  const record = expectRecord(value, label);
  return { id: expectString(record.id, `${label}.id`), name: expectString(record.name, `${label}.name`) };
}

function parseSessionSourceAssignment(value: unknown, label: string): SessionSourceAssignment {
  const record = expectRecord(value, label);
  return {
    kind: expectStringEnum(record.kind, SESSION_SOURCE_KINDS, `${label}.kind`),
    owner_id: expectString(record.owner_id, `${label}.owner_id`),
    owner_name: expectString(record.owner_name, `${label}.owner_name`),
  };
}

export function parseSessionMetadataList(payload: unknown): SessionMetadata[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid sessions list: expected array');
  }

  return payload.map((session, index) => parseSessionMetadata(session, `sessions list[${index}]`));
}

export function parseSessionSidebarPartitionState(payload: unknown): SessionSidebarPartitionState {
  const record = expectRecord(payload, 'session sidebar partition state');
  const version = expectNumber(record.version, 'session sidebar partition state.version');
  if (version !== 1) {
    throw new Error(`Invalid session sidebar partition state.version: expected 1, got ${version}`);
  }
  if (!Array.isArray(record.partitions)) {
    throw new Error('Invalid session sidebar partition state.partitions: expected array');
  }

  const assignmentsRecord = expectRecord(record.assignments, 'session sidebar partition state.assignments');
  const assignments: Record<string, string> = {};
  for (const [sessionID, partitionID] of Object.entries(assignmentsRecord)) {
    assignments[sessionID] = expectString(partitionID, `session sidebar partition state.assignments.${sessionID}`);
  }

  return {
    version: 1,
    partitions: record.partitions.map((partition, index) => parseSessionSidebarPartition(partition, `session sidebar partition state.partitions[${index}]`)),
    assignments,
  };
}

export function parseSessionSourceResolution(payload: unknown): SessionSourceResolution {
  const record = expectRecord(payload, 'session source resolution');
  const assignmentsRecord = expectRecord(record.assignments, 'session source resolution.assignments');
  const assignments: Record<string, SessionSourceAssignment> = {};
  for (const [sessionID, assignment] of Object.entries(assignmentsRecord)) {
    assignments[sessionID] = parseSessionSourceAssignment(
      assignment,
      `session source resolution.assignments.${sessionID}`,
    );
  }
  if (!Array.isArray(record.hidden_session_ids)) {
    throw new Error('Invalid session source resolution.hidden_session_ids: expected array');
  }
  return {
    assignments,
    hidden_session_ids: record.hidden_session_ids.map((sessionID, index) => {
      return expectString(sessionID, `session source resolution.hidden_session_ids[${index}]`);
    }),
  };
}

export function parseSessionDetail(payload: unknown): SessionDetail {
  const record = expectRecord(payload, 'session detail');
  if (!Array.isArray(record.messages)) {
    throw new Error('Invalid session detail.messages: expected array');
  }

  return {
    id: expectString(record.id, 'session detail.id'),
    title: expectString(record.title, 'session detail.title'),
    messages: record.messages.map((message, index) => parseSessionMessage(message, `session detail.messages[${index}]`)),
    created_at: expectString(record.created_at, 'session detail.created_at'),
    updated_at: expectString(record.updated_at, 'session detail.updated_at'),
    message_count: expectNumber(record.message_count, 'session detail.message_count'),
    page: parseSessionPage(record.page, 'session detail.page'),
    token_count: expectNumber(record.token_count, 'session detail.token_count'),
    turn_draft: parseOptionalSessionTurnDraft(record.turn_draft, 'session detail.turn_draft'),
  };
}
