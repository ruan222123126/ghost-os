import type {
  AgentCompletionDeltaPayload,
  AgentDonePayload,
  AgentErrorPayload,
  AgentIterationSummaryItem,
  AgentRunStartedPayload,
  AgentSendResponse,
  AgentStopResponsePayload,
  AgentStreamEvent,
  AgentStreamMessagePayload,
  AgentToolCallFinishedPayload,
  AgentToolCallStartedPayload,
  AssistantSessionEndSignal,
  AskHumanOption,
} from '@/lib/types';
import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
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
const AGENT_STREAM_EVENT_KEYS = ['id', 'step_id', 'trace_id', 'session_id', 'turn', 'type', 'payload', 'at'] as const;
const AGENT_RUN_STARTED_KEYS = ['session_id'] as const;
const AGENT_COMPLETION_DELTA_KEYS = [
  'kind',
  'text',
  'tool_call_index',
  'tool_call_id',
  'tool_name',
  'arguments_fragment',
] as const;
const AGENT_TOOL_CALL_STARTED_KEYS = ['tool', 'tool_call_id'] as const;
const AGENT_TOOL_CALL_FINISHED_KEYS = ['tool', 'tool_call_id', 'status', 'error'] as const;
const AGENT_AWAITING_EVENT_KEYS = ['tool', 'tool_call_id', 'question_id', 'prompt', 'selection_mode', 'options'] as const;
const AGENT_STREAM_MESSAGE_KEYS = ['text', 'session_id'] as const;
const AGENT_DONE_KEYS = ['session_id', 'session_ended'] as const;
const AGENT_ERROR_KEYS = ['message', 'session_id'] as const;
const ITERATION_SUMMARY_KEYS = [
  'iteration',
  'did',
  'remaining',
  'completed',
  'trace_id',
  'recorded_at',
  'final_change_log',
] as const;
const AGENT_MODES = ['pro', 'prox'] as const;
const STOP_STATUSES = ['stopped', 'not_running'] as const;
const SESSION_END_SIGNALS = ['END_SESSION'] as const;
const AGENT_STREAM_EVENT_TYPES = [
  'run_started',
  'completion_delta',
  'tool_call_started',
  'tool_call_finished',
  'awaiting_human',
  'message',
  'done',
  'error',
] as const;
const COMPLETION_DELTA_KINDS = ['text', 'tool_call_start', 'tool_call_delta', 'tool_call_end'] as const;

export interface AgentAwaitingHumanStreamPayload {
  tool?: string;
  tool_call_id?: string;
  question_id: string;
  prompt: string;
  selection_mode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

function parseSessionEndSignal(value: unknown): AssistantSessionEndSignal | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = pickKnownKeys(expectRecord(value, 'agent response.session_end'), SESSION_END_KEYS);

  return {
    signal: expectStringEnum(record.signal, SESSION_END_SIGNALS, 'agent response.session_end.signal'),
    message: expectString(record.message, 'agent response.session_end.message'),
  };
}

function parseAwaitingHumanResponse(payload: unknown): AgentSendResponse {
  const record = pickKnownKeys(expectRecord(payload, 'agent response'), AGENT_AWAITING_KEYS);

  return {
    status: 'awaiting_human',
    session_id: expectString(record.session_id, 'agent response.session_id'),
    question_id: expectString(record.question_id, 'agent response.question_id'),
    prompt: expectString(record.prompt, 'agent response.prompt'),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, 'agent response.selection_mode'),
    options: parseOptionalAskHumanOptions(record.options, 'agent response.options'),
  };
}

function parseIterationSummaryItem(value: unknown, label: string): AgentIterationSummaryItem {
  const record = pickKnownKeys(expectRecord(value, label), ITERATION_SUMMARY_KEYS);

  return {
    iteration: expectNumber(record.iteration, `${label}.iteration`),
    did: expectString(record.did, `${label}.did`),
    remaining: expectString(record.remaining, `${label}.remaining`),
    completed: record.completed === undefined ? undefined : expectBoolean(record.completed, `${label}.completed`),
    trace_id: parseOptionalString(record.trace_id, `${label}.trace_id`),
    recorded_at: parseOptionalString(record.recorded_at, `${label}.recorded_at`),
    final_change_log: parseOptionalString(record.final_change_log, `${label}.final_change_log`),
  };
}

function parseIterationSummary(value: unknown): AgentIterationSummaryItem[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error('Invalid agent response.iteration_summary: expected array');
  }

  return value.map((entry, index) => parseIterationSummaryItem(entry, `agent response.iteration_summary[${index}]`));
}

function parseSuccessResponse(payload: unknown): AgentSendResponse {
  const record = pickKnownKeys(expectRecord(payload, 'agent response'), AGENT_SUCCESS_KEYS);

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
  const record = pickKnownKeys(expectRecord(payload, 'agent stop response'), AGENT_STOP_KEYS);

  return {
    status: expectStringEnum(record.status, STOP_STATUSES, 'agent stop response.status'),
    message: expectString(record.message, 'agent stop response.message'),
  };
}

export function parseAgentStreamEvent(payload: unknown): AgentStreamEvent {
  const record = pickKnownKeys(expectRecord(payload, 'agent stream event'), AGENT_STREAM_EVENT_KEYS);

  return {
    id: expectString(record.id, 'agent stream event.id'),
    step_id: expectString(record.step_id, 'agent stream event.step_id'),
    trace_id: expectString(record.trace_id, 'agent stream event.trace_id'),
    session_id: parseOptionalString(record.session_id, 'agent stream event.session_id'),
    turn: expectNumber(record.turn, 'agent stream event.turn'),
    type: expectStringEnum(record.type, AGENT_STREAM_EVENT_TYPES, 'agent stream event.type'),
    payload: expectRecord(record.payload, 'agent stream event.payload'),
    at: parseOptionalString(record.at, 'agent stream event.at'),
  };
}

export function parseAgentRunStartedPayload(payload: unknown): AgentRunStartedPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent run_started payload'), AGENT_RUN_STARTED_KEYS);

  return {
    session_id: parseOptionalString(record.session_id, 'agent run_started payload.session_id'),
  };
}

export function parseAgentCompletionDeltaPayload(payload: unknown): AgentCompletionDeltaPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent completion_delta payload'), AGENT_COMPLETION_DELTA_KEYS);

  return {
    kind: expectStringEnum(record.kind, COMPLETION_DELTA_KINDS, 'agent completion_delta payload.kind'),
    text: parseOptionalString(record.text, 'agent completion_delta payload.text'),
    tool_call_index: record.tool_call_index === undefined
      ? undefined
      : expectNumber(record.tool_call_index, 'agent completion_delta payload.tool_call_index'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent completion_delta payload.tool_call_id'),
    tool_name: parseOptionalString(record.tool_name, 'agent completion_delta payload.tool_name'),
    arguments_fragment: parseOptionalString(record.arguments_fragment, 'agent completion_delta payload.arguments_fragment'),
  };
}

export function parseAgentToolCallStartedPayload(payload: unknown): AgentToolCallStartedPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent tool_call_started payload'), AGENT_TOOL_CALL_STARTED_KEYS);

  return {
    tool: parseOptionalString(record.tool, 'agent tool_call_started payload.tool'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent tool_call_started payload.tool_call_id'),
  };
}

export function parseAgentToolCallFinishedPayload(payload: unknown): AgentToolCallFinishedPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent tool_call_finished payload'), AGENT_TOOL_CALL_FINISHED_KEYS);

  return {
    tool: parseOptionalString(record.tool, 'agent tool_call_finished payload.tool'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent tool_call_finished payload.tool_call_id'),
    status: parseOptionalString(record.status, 'agent tool_call_finished payload.status'),
    error: parseOptionalString(record.error, 'agent tool_call_finished payload.error'),
  };
}

export function parseAgentAwaitingHumanStreamPayload(payload: unknown): AgentAwaitingHumanStreamPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent awaiting_human payload'), AGENT_AWAITING_EVENT_KEYS);

  return {
    tool: parseOptionalString(record.tool, 'agent awaiting_human payload.tool'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent awaiting_human payload.tool_call_id'),
    question_id: expectString(record.question_id, 'agent awaiting_human payload.question_id'),
    prompt: expectString(record.prompt, 'agent awaiting_human payload.prompt'),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, 'agent awaiting_human payload.selection_mode'),
    options: parseOptionalAskHumanOptions(record.options, 'agent awaiting_human payload.options'),
  };
}

export function parseAgentStreamMessagePayload(payload: unknown): AgentStreamMessagePayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent message payload'), AGENT_STREAM_MESSAGE_KEYS);

  return {
    text: expectString(record.text, 'agent message payload.text'),
    session_id: parseOptionalString(record.session_id, 'agent message payload.session_id'),
  };
}

export function parseAgentDonePayload(payload: unknown): AgentDonePayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent done payload'), AGENT_DONE_KEYS);

  return {
    session_id: parseOptionalString(record.session_id, 'agent done payload.session_id'),
    session_ended: record.session_ended === undefined
      ? undefined
      : expectBoolean(record.session_ended, 'agent done payload.session_ended'),
  };
}

export function parseAgentErrorPayload(payload: unknown): AgentErrorPayload {
  const record = pickKnownKeys(expectRecord(payload, 'agent error payload'), AGENT_ERROR_KEYS);

  return {
    message: expectString(record.message, 'agent error payload.message'),
    session_id: parseOptionalString(record.session_id, 'agent error payload.session_id'),
  };
}
