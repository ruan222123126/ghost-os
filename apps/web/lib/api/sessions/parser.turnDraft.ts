import type {
  SessionTurnDraft,
  SessionTurnDraftSegment,
  SessionTurnDraftTool,
} from '@/lib/types';
import {
  expectNumber,
  expectRecord,
  expectString,
  parseOptionalString,
} from '@/lib/api/shared';

function parseSessionTurnDraftSegment(value: unknown, label: string): SessionTurnDraftSegment {
  const record = expectRecord(value, label);
  return {
    id: expectString(record.id, `${label}.id`),
    content: expectString(record.content, `${label}.content`),
  };
}

function parseSessionTurnDraftTool(value: unknown, label: string): SessionTurnDraftTool {
  const record = expectRecord(value, label);
  return {
    id: expectString(record.id, `${label}.id`),
    content: expectString(record.content, `${label}.content`),
    tool_input: parseOptionalString(record.tool_input, `${label}.tool_input`),
    tool_name: parseOptionalString(record.tool_name, `${label}.tool_name`),
    tool_status: parseOptionalString(record.tool_status, `${label}.tool_status`),
    tool_call_id: parseOptionalString(record.tool_call_id, `${label}.tool_call_id`),
    trace_id: parseOptionalString(record.trace_id, `${label}.trace_id`),
  };
}

function parseTurnDraftStringArray(value: unknown, label: string): string[] {
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }
  return value.map((item, index) => expectString(item, `${label}[${index}]`));
}

export function parseOptionalSessionTurnDraft(
  value: unknown,
  label: string,
): SessionTurnDraft | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, label);
  if (!Array.isArray(record.assistant_segments)) {
    throw new Error(`Invalid ${label}.assistant_segments: expected array`);
  }
  if (!Array.isArray(record.thinking_segments)) {
    throw new Error(`Invalid ${label}.thinking_segments: expected array`);
  }
  if (!Array.isArray(record.tools)) {
    throw new Error(`Invalid ${label}.tools: expected array`);
  }

  return {
    trace_id: expectString(record.trace_id, `${label}.trace_id`),
    turn: expectNumber(record.turn, `${label}.turn`),
    assistant_segments: record.assistant_segments.map((item, index) => {
      return parseSessionTurnDraftSegment(item, `${label}.assistant_segments[${index}]`);
    }),
    thinking_segments: record.thinking_segments.map((item, index) => {
      return parseSessionTurnDraftSegment(item, `${label}.thinking_segments[${index}]`);
    }),
    tools: record.tools.map((item, index) => {
      return parseSessionTurnDraftTool(item, `${label}.tools[${index}]`);
    }),
    item_order: parseTurnDraftStringArray(record.item_order, `${label}.item_order`),
  };
}
