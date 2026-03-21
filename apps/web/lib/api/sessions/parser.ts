import type {
  SessionContentPart,
  SessionDetail,
  SessionFileContent,
  SessionHumanInteraction,
  SessionImageContent,
  SessionMessage,
  SessionMetadata,
  SessionToolCall,
  SessionToolResult,
} from '@/lib/types';
import {
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalAskHumanOptions,
  parseOptionalNumber,
  parseOptionalSelectionMode,
  parseOptionalString,
} from '@/lib/api/shared';

const SESSION_ROLES = ['system', 'internal', 'user', 'assistant', 'tool'] as const;
const TOOL_RESULT_STATUSES = ['success', 'error'] as const;
const SESSION_METADATA_KEYS = [
  'id',
  'created_at',
  'updated_at',
  'message_count',
  'token_count',
] as const;
const SESSION_DETAIL_KEYS = ['id', 'messages', 'created_at', 'updated_at', 'token_count'] as const;
const SESSION_MESSAGE_KEYS = [
  'role',
  'text',
  'content',
  'tool_calls',
  'tool_result',
  'human_interaction',
  'tool_call_id',
] as const;
const SESSION_CONTENT_PART_KEYS = ['type', 'text', 'image', 'file'] as const;
const SESSION_IMAGE_KEYS = ['path', 'url', 'mime_type', 'width', 'height', 'sha256', 'bytes'] as const;
const SESSION_FILE_KEYS = [
  'artifact_id',
  'name',
  'mime_type',
  'bytes',
  'sha256',
  'download_url',
  'source_path',
  'note',
] as const;
const SESSION_TOOL_CALL_KEYS = ['id', 'name', 'arguments'] as const;
const SESSION_TOOL_RESULT_KEYS = ['status', 'tool', 'trace_id', 'output', 'error'] as const;
const SESSION_HUMAN_INTERACTION_KEYS = [
  'question_id',
  'prompt',
  'selection_mode',
  'options',
  'answer',
] as const;

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

  const record = pickKnownKeys(expectRecord(value, label), SESSION_IMAGE_KEYS);

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

  const record = pickKnownKeys(expectRecord(value, label), SESSION_FILE_KEYS);

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
  const record = pickKnownKeys(expectRecord(value, label), SESSION_CONTENT_PART_KEYS);

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
  const record = pickKnownKeys(expectRecord(value, label), SESSION_TOOL_CALL_KEYS);

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

  const record = pickKnownKeys(expectRecord(value, label), SESSION_TOOL_RESULT_KEYS);

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

  const record = pickKnownKeys(expectRecord(value, label), SESSION_HUMAN_INTERACTION_KEYS);

  return {
    question_id: expectString(record.question_id, `${label}.question_id`),
    prompt: expectString(record.prompt, `${label}.prompt`),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, `${label}.selection_mode`),
    options: parseOptionalAskHumanOptions(record.options, `${label}.options`),
    answer: parseOptionalString(record.answer, `${label}.answer`),
  };
}

function parseSessionMessage(value: unknown, label: string): SessionMessage {
  const record = pickKnownKeys(expectRecord(value, label), SESSION_MESSAGE_KEYS);

  return {
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
  };
}

function parseSessionMetadata(value: unknown, label: string): SessionMetadata {
  const record = pickKnownKeys(expectRecord(value, label), SESSION_METADATA_KEYS);

  return {
    id: expectString(record.id, `${label}.id`),
    created_at: expectString(record.created_at, `${label}.created_at`),
    updated_at: expectString(record.updated_at, `${label}.updated_at`),
    message_count: expectNumber(record.message_count, `${label}.message_count`),
    token_count: expectNumber(record.token_count, `${label}.token_count`),
  };
}

export function parseSessionMetadataList(payload: unknown): SessionMetadata[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid sessions list: expected array');
  }

  return payload.map((session, index) => parseSessionMetadata(session, `sessions list[${index}]`));
}

export function parseSessionDetail(payload: unknown): SessionDetail {
  const record = pickKnownKeys(expectRecord(payload, 'session detail'), SESSION_DETAIL_KEYS);
  if (!Array.isArray(record.messages)) {
    throw new Error('Invalid session detail.messages: expected array');
  }

  return {
    id: expectString(record.id, 'session detail.id'),
    messages: record.messages.map((message, index) => {
      return parseSessionMessage(message, `session detail.messages[${index}]`);
    }),
    created_at: expectString(record.created_at, 'session detail.created_at'),
    updated_at: expectString(record.updated_at, 'session detail.updated_at'),
    token_count: expectNumber(record.token_count, 'session detail.token_count'),
  };
}
