import type {
  AgentCompletionDeltaPayload,
  AgentDonePayload,
  AgentErrorPayload,
  AgentAwaitingHumanStreamPayload,
  AgentRunStartedPayload,
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  AgentSendSuccessResponse,
  AgentStopResponsePayload,
  AgentStreamEvent,
  AgentStreamMessagePayload,
  AgentToolCallFinishedPayload,
  AgentToolCallStartedPayload,
  AssistantSessionEndSignal,
  ExternalAgentResponse,
} from '@/lib/types';
import {
  defineStringEnumValues,
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalAskHumanOptions,
  parseOptionalNumber,
  parseOptionalSelectionMode,
  parseOptionalString,
} from '@/lib/api/shared';

const AWAITING_HUMAN_STATUSES = defineStringEnumValues<AgentSendAwaitingHumanResponse['status']>({
  awaiting_human: true,
});
const AGENT_MODES = defineStringEnumValues<NonNullable<AgentSendSuccessResponse['mode']>>({
  plan: true,
});
const STOP_STATUSES = defineStringEnumValues<AgentStopResponsePayload['status']>({
  stopped: true,
  not_running: true,
});
const SESSION_END_SIGNALS = defineStringEnumValues<AssistantSessionEndSignal['signal']>({
  END_SESSION: true,
});
const AGENT_STREAM_EVENT_TYPES = defineStringEnumValues<AgentStreamEvent['type']>({
  run_started: true,
  completion_delta: true,
  tool_call_started: true,
  tool_call_finished: true,
  awaiting_human: true,
  message: true,
  done: true,
  error: true,
});
const COMPLETION_DELTA_KINDS = defineStringEnumValues<AgentCompletionDeltaPayload['kind']>({
  text: true,
  thinking: true,
  tool_call_start: true,
  tool_call_delta: true,
  tool_call_end: true,
});

function parseSessionEndSignal(value: unknown): AssistantSessionEndSignal | null | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }

  const record = expectRecord(value, 'agent response.session_end');

  return {
    signal: expectStringEnum(record.signal, SESSION_END_SIGNALS, 'agent response.session_end.signal'),
    message: expectString(record.message, 'agent response.session_end.message'),
  };
}

function parseAwaitingHumanResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');

  return {
    status: expectStringEnum(record.status, AWAITING_HUMAN_STATUSES, 'agent response.status'),
    session_id: expectString(record.session_id, 'agent response.session_id'),
    question_id: expectString(record.question_id, 'agent response.question_id'),
    prompt: expectString(record.prompt, 'agent response.prompt'),
    selection_mode: parseOptionalSelectionMode(record.selection_mode, 'agent response.selection_mode'),
    options: parseOptionalAskHumanOptions(record.options, 'agent response.options'),
  };
}

function parseSuccessResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');

  return {
    message: expectString(record.message, 'agent response.message'),
    session_id: expectString(record.session_id, 'agent response.session_id'),
    session_ended: expectBoolean(record.session_ended, 'agent response.session_ended'),
    mode: record.mode === undefined ? undefined : expectStringEnum(record.mode, AGENT_MODES, 'agent response.mode'),
    session_end: parseSessionEndSignal(record.session_end),
  };
}

export function parseAgentSendResponse(payload: unknown): AgentSendResponse {
  const record = expectRecord(payload, 'agent response');
  if (record.status === 'awaiting_human') {
    return parseAwaitingHumanResponse(record);
  }

  return parseSuccessResponse(record);
}

export function parseAgentStopResponse(payload: unknown): AgentStopResponsePayload {
  const record = expectRecord(payload, 'agent stop response');

  return {
    status: expectStringEnum(record.status, STOP_STATUSES, 'agent stop response.status'),
    message: expectString(record.message, 'agent stop response.message'),
    session_id: parseOptionalString(record.session_id, 'agent stop response.session_id'),
  };
}

export function parseExternalAgentResponse(payload: unknown): ExternalAgentResponse {
  const record = expectRecord(payload, 'external agent response');

  return {
    status: expectString(record.status, 'external agent response.status'),
    provider: parseOptionalString(record.provider, 'external agent response.provider'),
    session_id: parseOptionalString(record.session_id, 'external agent response.session_id'),
    thread_id: parseOptionalString(record.thread_id, 'external agent response.thread_id'),
    turn_id: parseOptionalString(record.turn_id, 'external agent response.turn_id'),
    permission_mode: parseOptionalString(record.permission_mode, 'external agent response.permission_mode'),
  };
}

export function parseAgentStreamEvent(payload: unknown): AgentStreamEvent {
  const record = expectRecord(payload, 'agent stream event');

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
  const record = expectRecord(payload, 'agent run_started payload');

  return {
    session_id: parseOptionalString(record.session_id, 'agent run_started payload.session_id'),
  };
}

export function parseAgentCompletionDeltaPayload(payload: unknown): AgentCompletionDeltaPayload {
  const record = expectRecord(payload, 'agent completion_delta payload');

  return {
    kind: expectStringEnum(record.kind, COMPLETION_DELTA_KINDS, 'agent completion_delta payload.kind'),
    text: parseOptionalString(record.text, 'agent completion_delta payload.text'),
    thinking: parseOptionalString(record.thinking, 'agent completion_delta payload.thinking'),
    tool_call_index: record.tool_call_index === undefined
      ? undefined
      : expectNumber(record.tool_call_index, 'agent completion_delta payload.tool_call_index'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent completion_delta payload.tool_call_id'),
    tool_name: parseOptionalString(record.tool_name, 'agent completion_delta payload.tool_name'),
    arguments_fragment: parseOptionalString(record.arguments_fragment, 'agent completion_delta payload.arguments_fragment'),
  };
}

export function parseAgentToolCallStartedPayload(payload: unknown): AgentToolCallStartedPayload {
  const record = expectRecord(payload, 'agent tool_call_started payload');

  return {
    tool: parseOptionalString(record.tool, 'agent tool_call_started payload.tool'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent tool_call_started payload.tool_call_id'),
    arguments_json: parseOptionalString(record.arguments_json, 'agent tool_call_started payload.arguments_json'),
  };
}

export function parseAgentToolCallFinishedPayload(payload: unknown): AgentToolCallFinishedPayload {
  const record = expectRecord(payload, 'agent tool_call_finished payload');

  return {
    tool: parseOptionalString(record.tool, 'agent tool_call_finished payload.tool'),
    tool_call_id: parseOptionalString(record.tool_call_id, 'agent tool_call_finished payload.tool_call_id'),
    status: parseOptionalString(record.status, 'agent tool_call_finished payload.status'),
    error: parseOptionalString(record.error, 'agent tool_call_finished payload.error'),
    output: parseOptionalString(record.output, 'agent tool_call_finished payload.output'),
  };
}

export function parseAgentAwaitingHumanStreamPayload(payload: unknown): AgentAwaitingHumanStreamPayload {
  const record = expectRecord(payload, 'agent awaiting_human payload');

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
  const record = expectRecord(payload, 'agent message payload');

  return {
    text: expectString(record.text, 'agent message payload.text'),
    session_id: parseOptionalString(record.session_id, 'agent message payload.session_id'),
  };
}

export function parseAgentDonePayload(payload: unknown): AgentDonePayload {
  const record = expectRecord(payload, 'agent done payload');

  return {
    session_id: parseOptionalString(record.session_id, 'agent done payload.session_id'),
    session_ended: record.session_ended === undefined
      ? undefined
      : expectBoolean(record.session_ended, 'agent done payload.session_ended'),
  };
}

export function parseAgentErrorPayload(payload: unknown): AgentErrorPayload {
  const record = expectRecord(payload, 'agent error payload');

  return {
    message: expectString(record.message, 'agent error payload.message'),
    session_id: parseOptionalString(record.session_id, 'agent error payload.session_id'),
    code: parseOptionalNumber(record.code, 'agent error payload.code'),
  };
}
