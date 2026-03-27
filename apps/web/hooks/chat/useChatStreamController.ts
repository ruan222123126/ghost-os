import { useCallback } from 'react';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import {
  parseAgentAwaitingHumanStreamPayload,
  parseAgentCompletionDeltaPayload,
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
  parseAgentStreamMessagePayload,
  parseAgentToolCallFinishedPayload,
  parseAgentToolCallStartedPayload,
} from '@/lib/api/agent/parser';
import {
  appendAssistantText,
  replaceAssistantText,
  suppressGraphQLAssistantToolText,
  upsertPendingQuestion,
  upsertToolMessage,
} from '@/lib/chatStream';
import { isGraphQLToolDocument } from '@/lib/graphqlToolText';
import type { AgentStreamEvent } from '@/lib/types';
import type { ChatStateControls, StreamAgentRunInput, UseBridgeChatOptions } from './types';

const TOOL_RUNNING_STATUS = 'running';
const TOOL_SUCCESS_STATUS = 'success';
const TOOL_ERROR_STATUS = 'error';

interface StreamRuntimeState {
  assistantMessageId: string;
  sessionId: string;
  toolMessageIds: Map<string, string>;
  traceId: string;
}
interface StreamHumanRunOptions {
  answer: string;
  cancelled?: boolean;
  questionId: string;
  sessionId: string;
  signal?: AbortSignal;
  traceId: string;
}

interface UseChatStreamControllerOptions {
  activeRunRef: ChatStateControls['activeRunRef'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  hydrateSessionHistory: (sessionId: string) => Promise<void>;
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setMessages: ChatStateControls['setMessages'];
}
export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const { activeRunRef, currentSessionId, hydrateSessionHistory, onSessionResolved, setActiveRun, setMessages } = options;

  const syncSession = useCallback(async (sessionId?: string) => {
    const trimmedSessionId = sessionId?.trim();
    if (!trimmedSessionId) {
      return;
    }

    const currentRun = activeRunRef.current;
    if (currentRun && currentRun.sessionId !== trimmedSessionId) {
      setActiveRun({ ...currentRun, sessionId: trimmedSessionId });
    }
    if (trimmedSessionId !== currentSessionId) {
      onSessionResolved?.(trimmedSessionId);
    }
    await hydrateSessionHistory(trimmedSessionId);
  }, [activeRunRef, currentSessionId, hydrateSessionHistory, onSessionResolved, setActiveRun]);

  const applyEvent = useCallback((state: StreamRuntimeState, event: AgentStreamEvent) => {
    syncActiveSession({
      activeRunRef,
      currentSessionId,
      event,
      onSessionResolved,
      setActiveRun,
      state,
    });
    applyEventMessages({ event, setMessages, state });
  }, [activeRunRef, currentSessionId, onSessionResolved, setActiveRun, setMessages]);

  const runAgentStream = useCallback(async (run: StreamAgentRunInput) => {
    const state = createStreamRuntimeState(run.traceId, run.sessionId);
    const result = await streamMessage({
      images: run.images,
      message: run.message,
      onEvent: async (event) => {
        applyEvent(state, event);
      },
      sessionId: run.sessionId,
      signal: run.signal,
      traceId: run.traceId,
    });
    await syncSession(result.sessionId || state.sessionId);
  }, [applyEvent, syncSession]);

  const runHumanStream = useCallback(async (run: StreamHumanRunOptions) => {
    const state = createStreamRuntimeState(run.traceId, run.sessionId);
    const result = await streamHumanResponse({
      answer: run.answer,
      cancelled: run.cancelled,
      onEvent: async (event) => {
        applyEvent(state, event);
      },
      questionId: run.questionId,
      sessionId: run.sessionId,
      signal: run.signal,
      traceId: run.traceId,
    });
    await syncSession(result.sessionId || state.sessionId);
  }, [applyEvent, syncSession]);

  return {
    runAgentStream,
    runHumanStream,
  };
}
function createStreamRuntimeState(traceId: string, sessionId?: string): StreamRuntimeState {
  const trimmedTraceId = traceId.trim();
  return {
    assistantMessageId: `stream-assistant:${trimmedTraceId}`,
    sessionId: sessionId?.trim() || '',
    toolMessageIds: new Map(),
    traceId: trimmedTraceId,
  };
}
function syncActiveSession(options: {
  activeRunRef: ChatStateControls['activeRunRef'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  event: AgentStreamEvent;
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  state: StreamRuntimeState;
}): void {
  const sessionId = resolveEventSessionId(options.event);
  if (!sessionId || sessionId === options.state.sessionId) {
    return;
  }

  options.state.sessionId = sessionId;
  const currentRun = options.activeRunRef.current;
  if (currentRun && currentRun.sessionId !== sessionId) {
    options.setActiveRun({ ...currentRun, sessionId });
  }
  if (sessionId !== options.currentSessionId) {
    options.onSessionResolved?.(sessionId);
  }
}
function resolveEventSessionId(event: AgentStreamEvent): string {
  if (event.session_id?.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case 'run_started':
      return parseAgentRunStartedPayload(event.payload).session_id?.trim() || '';
    case 'message':
      return parseAgentStreamMessagePayload(event.payload).session_id?.trim() || '';
    case 'done':
      return parseAgentDonePayload(event.payload).session_id?.trim() || '';
    case 'error':
      return parseAgentErrorPayload(event.payload).session_id?.trim() || '';
    default:
      return '';
  }
}
function applyEventMessages(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  switch (options.event.type) {
    case 'completion_delta':
      applyCompletionDelta(options);
      return;
    case 'tool_call_started':
      applyToolStarted(options);
      return;
    case 'tool_call_finished':
      applyToolFinished(options);
      return;
    case 'awaiting_human':
      applyAwaitingHuman(options);
      return;
    case 'message':
      applyMessage(options);
      return;
    default:
      return;
  }
}
function applyCompletionDelta(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  const payload = parseAgentCompletionDeltaPayload(options.event.payload);
  const text = payload.text;
  if (payload.kind !== 'text' || !text) {
    return;
  }

  options.setMessages((previous) => appendAssistantText(previous, {
    messageId: options.state.assistantMessageId,
    text,
  }));
}
function applyToolStarted(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  const payload = parseAgentToolCallStartedPayload(options.event.payload);
  const messageId = resolveToolMessageId(options.state, options.event, payload.tool_call_id);

  options.setMessages((previous) => upsertToolMessage(suppressGraphQLAssistantToolText(previous, options.state.assistantMessageId), {
    messageId,
    content: `${payload.tool || 'Tool'} running`,
    toolCallId: payload.tool_call_id,
    toolName: payload.tool,
    toolStatus: TOOL_RUNNING_STATUS,
    traceId: options.event.trace_id,
  }));
}
function applyToolFinished(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  const payload = parseAgentToolCallFinishedPayload(options.event.payload);
  const messageId = resolveToolMessageId(options.state, options.event, payload.tool_call_id);
  const toolStatus = payload.status?.trim() || (payload.error ? TOOL_ERROR_STATUS : TOOL_SUCCESS_STATUS);
  const content = payload.error?.trim() || `${payload.tool || 'Tool'} finished`;

  options.setMessages((previous) => upsertToolMessage(previous, {
    messageId,
    content,
    toolCallId: payload.tool_call_id,
    toolName: payload.tool,
    toolStatus,
    traceId: options.event.trace_id,
  }));
}
function applyAwaitingHuman(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  const payload = parseAgentAwaitingHumanStreamPayload(options.event.payload);
  const sessionId = options.state.sessionId.trim();
  if (!sessionId) {
    throw new Error('awaiting_human stream event is missing session_id');
  }

  options.setMessages((previous) => upsertPendingQuestion(previous, {
    messageId: `stream-question:${options.state.traceId}:${payload.question_id}`,
    content: payload.prompt,
    options: payload.options,
    questionId: payload.question_id,
    selectionMode: payload.selection_mode,
    sessionId,
  }));
}
function applyMessage(options: {
  event: AgentStreamEvent;
  setMessages: ChatStateControls['setMessages'];
  state: StreamRuntimeState;
}): void {
  const payload = parseAgentStreamMessagePayload(options.event.payload);
  if (options.state.toolMessageIds.size > 0 && isGraphQLToolDocument(payload.text)) {
    options.setMessages((previous) => suppressGraphQLAssistantToolText(previous, options.state.assistantMessageId));
    return;
  }
  options.setMessages((previous) => replaceAssistantText(previous, {
    messageId: options.state.assistantMessageId,
    text: payload.text,
  }));
}
function resolveToolMessageId(
  state: StreamRuntimeState,
  event: AgentStreamEvent,
  toolCallId?: string,
): string {
  const toolKey = toolCallId?.trim() || event.step_id.trim() || event.id.trim();
  const existing = state.toolMessageIds.get(toolKey);
  if (existing) {
    return existing;
  }

  const messageId = `stream-tool:${state.traceId}:${toolKey}`;
  state.toolMessageIds.set(toolKey, messageId);
  return messageId;
}
