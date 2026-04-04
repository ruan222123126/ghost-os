import {
  parseAgentAwaitingHumanStreamPayload,
  parseAgentCompletionDeltaPayload,
  parseAgentStreamMessagePayload,
  parseAgentToolCallFinishedPayload,
  parseAgentToolCallStartedPayload,
} from '@/lib/api/agent/parser';
import { buildAssistantMessage } from '@/lib/chatMessages';
import {
  consumeToolTagStreamChunk,
  type ToolTagStreamEvent,
  type ToolTagStreamUnit,
  stripToolTagCalls,
} from '@/lib/toolTagText';
import type { AgentStreamEvent, PendingQuestionMessage } from '@/lib/types';
import {
  TOOL_ERROR_STATUS,
  TOOL_PENDING_STATUS,
  TOOL_RUNNING_STATUS,
  TOOL_SUCCESS_STATUS,
  type EventApplyOptions,
  type StreamRuntimeState,
} from './chatStreamControllerTypes';

interface EventDispatchOptions extends EventApplyOptions {
  event: AgentStreamEvent;
}

export function applyEventState(options: EventDispatchOptions): void {
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

function applyCompletionDelta(options: EventDispatchOptions): void {
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

function applyToolStarted(options: EventDispatchOptions): void {
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

function applyToolFinished(options: EventDispatchOptions): void {
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

function applyAwaitingHuman(options: EventDispatchOptions): void {
  const payload = parseAgentAwaitingHumanStreamPayload(options.event.payload);
  const sessionId = options.state.sessionId.trim();
  if (!sessionId) {
    throw new Error('awaiting_human stream event is missing session_id');
  }

  options.upsertPendingQuestion(buildPendingQuestionMessage(options.state, payload, sessionId));
}

function applyMessage(options: EventDispatchOptions): void {
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

  options.appendCommittedMessages([buildAssistantMessage(visibleAssistantText, options.state.assistantMessageId)]);
  options.state.assistantBuffer = '';
}

function applyToolTagUnits(
  state: StreamRuntimeState,
  traceId: string,
  units: ToolTagStreamUnit[],
  appendStreamingAssistantText: EventApplyOptions['appendStreamingAssistantText'],
  upsertStreamingTool: EventApplyOptions['upsertStreamingTool'],
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
  appendStreamingAssistantText: EventApplyOptions['appendStreamingAssistantText'],
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
  upsertStreamingTool: EventApplyOptions['upsertStreamingTool'],
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

function resolveToolMessageId(state: StreamRuntimeState, event: AgentStreamEvent, toolCallId?: string): string {
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
