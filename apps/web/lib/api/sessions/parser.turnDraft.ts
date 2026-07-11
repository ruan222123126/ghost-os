import type {
  SessionTurnDraft,
  SessionTurnDraftPendingQuestion,
  SessionTurnDraftSegment,
  SessionTurnDraftTool,
} from '@/lib/envelope.generated';
import {
  defineStringEnumValues,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalAskHumanOptions,
  parseOptionalSelectionMode,
  parseOptionalString,
} from '@/lib/api/shared';

const TURN_DRAFT_STATUSES = defineStringEnumValues<SessionTurnDraft['status']>({
  streaming: true,
  awaiting_human: true,
  error: true,
});

type TurnDraftArrayParser<T> = (value: unknown, label: string) => T;

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

function parseSessionTurnDraftPendingQuestion(
  value: unknown,
  label: string,
): SessionTurnDraftPendingQuestion {
  const record = expectRecord(value, label);
  return {
    question_id: expectString(record.question_id, `${label}.question_id`),
    prompt: expectString(record.prompt, `${label}.prompt`),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, `${label}.selection_mode`),
    options: parseOptionalAskHumanOptions(record.options, `${label}.options`),
  };
}

function parseTurnDraftStringArray(value: unknown, label: string): string[] {
  return parseTurnDraftArray(value, label, expectString);
}

function parseTurnDraftArray<T>(
  value: unknown,
  label: string,
  parser: TurnDraftArrayParser<T>,
): T[] {
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }
  return value.map((item, index) => parser(item, `${label}[${index}]`));
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
  const status = expectStringEnum(record.status, TURN_DRAFT_STATUSES, `${label}.status`);
  const error = parseOptionalString(record.error, `${label}.error`);
  if (status === 'error' && !error) {
    throw new Error(`Invalid ${label}.error: expected string when status=error`);
  }

  return {
    trace_id: expectString(record.trace_id, `${label}.trace_id`),
    turn: expectNumber(record.turn, `${label}.turn`),
    status,
    error,
    pending_questions: parseTurnDraftArray(
      record.pending_questions,
      `${label}.pending_questions`,
      parseSessionTurnDraftPendingQuestion,
    ),
    assistant_segments: parseTurnDraftArray(
      record.assistant_segments,
      `${label}.assistant_segments`,
      parseSessionTurnDraftSegment,
    ),
    thinking_segments: parseTurnDraftArray(
      record.thinking_segments,
      `${label}.thinking_segments`,
      parseSessionTurnDraftSegment,
    ),
    tools: parseTurnDraftArray(record.tools, `${label}.tools`, parseSessionTurnDraftTool),
    item_order: parseTurnDraftStringArray(record.item_order, `${label}.item_order`),
  };
}
