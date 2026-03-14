import type {
  AgentSendResponse,
  AgentStopResponsePayload,
  AssistantSessionEndSignal,
} from '@/lib/types';
import {
  ensureKnownKeys,
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalAskHumanOptions,
  parseOptionalSelectionMode,
  parseOptionalString,
} from '@/lib/api/shared';

const AGENT_SUCCESS_KEYS = [
  'message',
  'session_id',
  'session_ended',
  'mode',
  'iteration_count',
  'stopped_by',
  'final_change_log',
  'iteration_summary',
  'session_end',
] as const;
const AGENT_AWAITING_KEYS = [
  'status',
  'session_id',
  'question_id',
  'prompt',
  'selection_mode',
  'options',
 ] as const;
const AGENT_STOP_KEYS = ['status', 'message'] as const;
const SESSION_END_KEYS = ['signal', 'message'] as const;
const AGENT_MODES = ['pro', 'prox'] as const;
const STOP_STATUSES = ['stopped', 'not_running'] as const;
const SESSION_END_SIGNALS = ['END_SESSION'] as const;

function parseSessionEndSignal(value: unknown): AssistantSessionEndSignal | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, 'agent response.session_end');
  ensureKnownKeys(record, SESSION_END_KEYS, 'agent response.session_end');

  return {
    signal: expectStringEnum(record.signal, SESSION_END_SIGNALS, 'agent response.session_end.signal'),
    message: expectString(record.message, 'agent response.session_end.message'),
  };
}

function parseAwaitingHumanResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');
  ensureKnownKeys(record, AGENT_AWAITING_KEYS, 'agent response');

  return {
    status: 'awaiting_human',
    session_id: expectString(record.session_id, 'agent response.session_id'),
    question_id: expectString(record.question_id, 'agent response.question_id'),
    prompt: expectString(record.prompt, 'agent response.prompt'),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, 'agent response.selection_mode'),
    options: parseOptionalAskHumanOptions(record.options, 'agent response.options'),
  };
}

function parseIterationSummary(value: unknown): Record<string, unknown>[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error('Invalid agent response.iteration_summary: expected array');
  }

  return value.map((entry, index) => expectRecord(entry, `agent response.iteration_summary[${index}]`));
}

function parseSuccessResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');
  ensureKnownKeys(record, AGENT_SUCCESS_KEYS, 'agent response');

  return {
    message: expectString(record.message, 'agent response.message'),
    session_id: expectString(record.session_id, 'agent response.session_id'),
    session_ended: expectBoolean(record.session_ended, 'agent response.session_ended'),
    mode: record.mode === undefined ? undefined : expectStringEnum(record.mode, AGENT_MODES, 'agent response.mode'),
    iteration_count: record.iteration_count === undefined
      ? undefined
      : expectNumber(record.iteration_count, 'agent response.iteration_count'),
    stopped_by: parseOptionalString(record.stopped_by, 'agent response.stopped_by'),
    final_change_log: parseOptionalString(record.final_change_log, 'agent response.final_change_log'),
    iteration_summary: parseIterationSummary(record.iteration_summary),
    session_end: parseSessionEndSignal(record.session_end),
  };
}

export function parseAgentSendResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');
  if (record.status === 'awaiting_human') {
    return parseAwaitingHumanResponse(payload);
  }

  return parseSuccessResponse(payload);
}

export function parseAgentStopResponse(payload: unknown): AgentStopResponsePayload {
  const record = expectRecord(payload, 'agent stop response');
  ensureKnownKeys(record, AGENT_STOP_KEYS, 'agent stop response');

  return {
    status: expectStringEnum(record.status, STOP_STATUSES, 'agent stop response.status'),
    message: expectString(record.message, 'agent stop response.message'),
  };
}
