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
import { buildAssistantMessage } from '@/lib/chatMessages';
import {
  consumeToolTagStreamChunk,
  createToolTagStreamState,
  isPureToolTagDocument,
  stripToolTagCalls,
  type ToolTagStreamEvent,
  type ToolTagStreamUnit,
  type ToolTagStreamState,
} from '@/lib/toolTagText';
import type { AgentStreamEvent, PendingQuestionMessage } from '@/lib/types';
import type { ChatStateControls, StreamAgentRunInput, UseBridgeChatOptions } from './types';

const TOOL_PENDING_STATUS = 'pending';
const TOOL_RUNNING_STATUS = 'running';
const TOOL_SUCCESS_STATUS = 'success';
const TOOL_ERROR_STATUS = 'error';

interface StreamRuntimeState {
  assistantBuffer: string;
  assistantMessageId: string;
  pendingPreviewQueue: string[];
  previewToolArgs: Map<string, string>;
  previewToolCallSeqToID: Map<number, string>;
  previewToolIDByMessageId: Map<string, string>;
  sessionId: string;
  toolMessageIds: Map<string, string>;
  toolTagState: ToolTagStreamState;
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
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}

export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const {
    activeRunRef,
    appendCommittedMessages,
    appendStreamingAssistantText,
    clearStreamingAssistantText,
    clearStreamingState,
    currentSessionId,
    onSessionResolved,
    setActiveRun,
    syncRecentHistory,
    upsertPendingQuestion,
    upsertStreamingTool,
  } = options;

  const syncSession = useCallback(async (state: StreamRuntimeState, sessionId?: string) => {
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
    await syncRecentHistory(trimmedSessionId);
    clearStreamingState();
    state.assistantBuffer = '';
  }, [
    activeRunRef,
    clearStreamingState,
    currentSessionId,
    onSessionResolved,
    setActiveRun,
    syncRecentHistory,
  ]);

  const applyEvent = useCallback((state: StreamRuntimeState, event: AgentStreamEvent) => {
    syncActiveSession({
      activeRunRef,
      currentSessionId,
      event,
      onSessionResolved,
      setActiveRun,
      state,
    });
    applyEventState({
      appendCommittedMessages,
      appendStreamingAssistantText,
      clearStreamingAssistantText,
      event,
      state,
      upsertPendingQuestion,
      upsertStreamingTool,
    });
  }, [
    activeRunRef,
    appendCommittedMessages,
    appendStreamingAssistantText,
    clearStreamingAssistantText,
    currentSessionId,
    onSessionResolved,
    setActiveRun,
    upsertPendingQuestion,
    upsertStreamingTool,
  ]);

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
    await syncSession(state, result.sessionId || state.sessionId);
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
    await syncSession(state, result.sessionId || state.sessionId);
  }, [applyEvent, syncSession]);

  return {
    runAgentStream,
    runHumanStream,
  };
}

function createStreamRuntimeState(traceId: string, sessionId?: string): StreamRuntimeState {
  const trimmedTraceId = traceId.trim();
  return {
    assistantBuffer: '',
    assistantMessageId: `stream-assistant:${trimmedTraceId}`,
    pendingPreviewQueue: [],
    previewToolArgs: new Map(),
    previewToolCallSeqToID: new Map(),
    previewToolIDByMessageId: new Map(),
    sessionId: sessionId?.trim() || '',
    toolMessageIds: new Map(),
    toolTagState: createToolTagStreamState(),
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

function applyEventState(options: {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
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
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}): void {
  const payload = parseAgentCompletionDeltaPayload(options.event.payload);
  const text = payload.text;
  if (payload.kind !== 'text' || !text) {
    return;
  }

  const consumed = consumeToolTagStreamChunk(options.state.toolTagState, text);
  applyToolTagUnits(
    options.state,
    options.event.trace_id,
    consumed.units,
    options.appendStreamingAssistantText,
    options.upsertStreamingTool,
  );
}

function applyToolStarted(options: {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}): void {
  const payload = parseAgentToolCallStartedPayload(options.event.payload);
  const messageId = resolveToolMessageId(options.state, options.event, payload.tool_call_id);
  const argsPreview = options.state.previewToolArgs.get(messageId) || '';
  const previewToolName = options.state.previewToolIDByMessageId.get(messageId);

  options.upsertStreamingTool({
    id: messageId,
    content: argsPreview || `${payload.tool || 'Tool'} running`,
    toolCallId: payload.tool_call_id,
    toolName: payload.tool || formatToolIDName(previewToolName),
    toolStatus: TOOL_RUNNING_STATUS,
    traceId: options.event.trace_id,
  });
}

function applyToolFinished(options: {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}): void {
  const payload = parseAgentToolCallFinishedPayload(options.event.payload);
  const toolStatus = payload.status?.trim() || (payload.error ? TOOL_ERROR_STATUS : TOOL_SUCCESS_STATUS);
  const messageId = resolveToolMessageId(options.state, options.event, payload.tool_call_id);
  const argsPreview = options.state.previewToolArgs.get(messageId) || '';
  const previewToolName = options.state.previewToolIDByMessageId.get(messageId);

  options.upsertStreamingTool({
    id: messageId,
    content: payload.error?.trim() || argsPreview || `${payload.tool || 'Tool'} finished`,
    toolCallId: payload.tool_call_id,
    toolName: payload.tool || formatToolIDName(previewToolName),
    toolStatus,
    traceId: options.event.trace_id,
  });
}

function applyAwaitingHuman(options: {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}): void {
  const payload = parseAgentAwaitingHumanStreamPayload(options.event.payload);
  const sessionId = options.state.sessionId.trim();
  if (!sessionId) {
    throw new Error('awaiting_human stream event is missing session_id');
  }

  options.upsertPendingQuestion(buildPendingQuestionMessage(options.state, payload, sessionId));
}

function applyMessage(options: {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'];
  clearStreamingAssistantText: ChatStateControls['clearStreamingAssistantText'];
  event: AgentStreamEvent;
  state: StreamRuntimeState;
  upsertPendingQuestion: ChatStateControls['upsertPendingQuestion'];
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'];
}): void {
  const payload = parseAgentStreamMessagePayload(options.event.payload);

  const finalized = consumeToolTagStreamChunk(options.state.toolTagState, '', true);
  applyToolTagUnits(
    options.state,
    options.event.trace_id,
    finalized.units,
    options.appendStreamingAssistantText,
    options.upsertStreamingTool,
  );

  options.clearStreamingAssistantText();

  if (!payload.text.trim()) {
    options.state.assistantBuffer = '';
    return;
  }

  const visibleAssistantText = stripToolTagCalls(payload.text);
  if (!visibleAssistantText.trim()) {
    options.state.assistantBuffer = '';
    return;
  }

  options.appendCommittedMessages([
    buildAssistantMessage(visibleAssistantText, options.state.assistantMessageId),
  ]);
  options.state.assistantBuffer = '';
}

function applyToolTagUnits(
  state: StreamRuntimeState,
  traceId: string,
  units: ToolTagStreamUnit[],
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'],
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'],
): void {
  for (const unit of units) {
    if (unit.type === 'text') {
      appendStreamingText(state, unit.text, appendStreamingAssistantText);
      continue;
    }
    applyToolTagEvent(state, traceId, unit, upsertStreamingTool);
  }
}

function appendStreamingText(
  state: StreamRuntimeState,
  text: string,
  appendStreamingAssistantText: ChatStateControls['appendStreamingAssistantText'],
): void {
  if (!text) {
    return;
  }
  state.assistantBuffer = `${state.assistantBuffer}${text}`;
  appendStreamingAssistantText(text);
}

function applyToolTagEvent(
  state: StreamRuntimeState,
  traceId: string,
  event: ToolTagStreamEvent,
  upsertStreamingTool: ChatStateControls['upsertStreamingTool'],
): void {
  if (event.type === 'tool_open') {
    const messageId = ensurePreviewMessageID(state, event.callSeq, event.toolId);
    const args = state.previewToolArgs.get(messageId) || '';
    upsertStreamingTool({
      id: messageId,
      content: args,
      toolName: formatToolIDName(event.toolId),
      toolStatus: TOOL_PENDING_STATUS,
      traceId,
    });
    return;
  }

  if (event.type === 'tool_args') {
    const messageId = state.previewToolCallSeqToID.get(event.callSeq);
    if (!messageId) {
      return;
    }
    const current = state.previewToolArgs.get(messageId) || '';
    const next = `${current}${event.argsDelta}`;
    state.previewToolArgs.set(messageId, next);
    upsertStreamingTool({
      id: messageId,
      content: next,
      toolName: formatToolIDName(state.previewToolIDByMessageId.get(messageId)),
      toolStatus: TOOL_PENDING_STATUS,
      traceId,
    });
    return;
  }

  const messageId = ensurePreviewMessageID(state, event.callSeq, event.toolId);
  state.previewToolArgs.set(messageId, event.argsText);
  if (!state.pendingPreviewQueue.includes(messageId)) {
    state.pendingPreviewQueue.push(messageId);
  }
  upsertStreamingTool({
    id: messageId,
    content: event.argsText,
    toolName: formatToolIDName(event.toolId),
    toolStatus: TOOL_PENDING_STATUS,
    traceId,
  });
}

function ensurePreviewMessageID(state: StreamRuntimeState, callSeq: number, toolId: string): string {
  const existing = state.previewToolCallSeqToID.get(callSeq);
  if (existing) {
    return existing;
  }
  const messageId = `stream-tag-tool:${state.traceId}:${callSeq}`;
  state.previewToolCallSeqToID.set(callSeq, messageId);
  state.previewToolIDByMessageId.set(messageId, toolId);
  if (!state.previewToolArgs.has(messageId)) {
    state.previewToolArgs.set(messageId, '');
  }
  return messageId;
}

function formatToolIDName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  if (!trimmed) {
    return undefined;
  }
  return `tool#${trimmed}`;
}

function buildPendingQuestionMessage(
  state: StreamRuntimeState,
  payload: ReturnType<typeof parseAgentAwaitingHumanStreamPayload>,
  sessionId: string,
): PendingQuestionMessage {
  return {
    id: `stream-question:${state.traceId}:${payload.question_id}`,
    kind: 'pending_question',
    content: payload.prompt,
    options: payload.options,
    questionId: payload.question_id,
    selectionMode: payload.selection_mode,
    sessionId,
  };
}

function resolveToolMessageId(
  state: StreamRuntimeState,
  event: AgentStreamEvent,
  toolCallId?: string,
): string {
  const trimmedToolCallID = toolCallId?.trim();
  if (trimmedToolCallID) {
    const existing = state.toolMessageIds.get(trimmedToolCallID);
    if (existing) {
      return existing;
    }

    if (state.pendingPreviewQueue.length > 0) {
      const previewMessageID = state.pendingPreviewQueue.shift();
      if (previewMessageID) {
        state.toolMessageIds.set(trimmedToolCallID, previewMessageID);
        return previewMessageID;
      }
    }

    const generated = `stream-tool:${state.traceId}:${trimmedToolCallID}`;
    state.toolMessageIds.set(trimmedToolCallID, generated);
    return generated;
  }

  const fallbackKey = event.step_id.trim() || event.id.trim();
  const existingFallback = state.toolMessageIds.get(fallbackKey);
  if (existingFallback) {
    return existingFallback;
  }

  const generatedFallback = `stream-tool:${state.traceId}:${fallbackKey}`;
  state.toolMessageIds.set(fallbackKey, generatedFallback);
  return generatedFallback;
}

export { isPureToolTagDocument };
