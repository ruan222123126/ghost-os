import {
  parseAgentAwaitingHumanStreamPayload,
  parseAgentCompletionDeltaPayload,
  parseAgentStreamMessagePayload,
  parseAgentToolCallFinishedPayload,
  parseAgentToolCallStartedPayload,
} from '@/lib/api/agent/parser';
import { buildAssistantMessage, buildThinkingMessage } from '@/lib/chatMessages';
import { consumeToolTagStreamChunk, stripToolTagCalls } from '@/lib/toolTagText';
import type { AgentStreamEvent, PendingQuestionMessage } from '@/lib/types';
import type { ChatRuntimeAction } from './actions';
import {
  TOOL_ERROR_STATUS,
  TOOL_RUNNING_STATUS,
  TOOL_SUCCESS_STATUS,
} from './constants';
import type { ChatRuntimeState } from './runtimeState';
import { projectToolTagUnits } from './toolTagProjection';

interface ProjectAgentEventOptions {
  event: AgentStreamEvent;
  runtime: ChatRuntimeState;
}

export function projectAgentEvent(options: ProjectAgentEventOptions): ChatRuntimeAction[] {
  switch (options.event.type) {
    case 'completion_delta':
      return projectCompletionDelta(options);
    case 'tool_call_started':
      return [projectToolStarted(options)];
    case 'tool_call_finished':
      return [projectToolFinished(options)];
    case 'awaiting_human':
      return [projectAwaitingHuman(options)];
    case 'message':
      return projectMessage(options);
    case 'done':
    case 'error':
      options.runtime.thinkingBuffer = '';
      return [{ type: 'clear_streaming_thinking_text' }];
    default:
      return [];
  }
}

function projectCompletionDelta({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentCompletionDeltaPayload(event.payload);
  if (payload.kind === 'thinking') {
    if (!payload.thinking) {
      return [];
    }
    runtime.thinkingBuffer = `${runtime.thinkingBuffer}${payload.thinking}`;
    return [{ type: 'append_streaming_thinking_text', text: payload.thinking }];
  }
  if (payload.kind !== 'text' || !payload.text) {
    return [];
  }

  const consumed = consumeToolTagStreamChunk(runtime.toolTagState, payload.text);
  return projectToolTagUnits(runtime, event.trace_id, consumed.units);
}

function projectToolStarted({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction {
  const payload = parseAgentToolCallStartedPayload(event.payload);
  const messageId = resolveToolMessageID(runtime, event, payload.tool_call_id);
  const argsPreview = runtime.previewToolArgs.get(messageId) || '';
  const previewToolName = runtime.previewToolIDByMessageId.get(messageId);
  return {
    type: 'upsert_streaming_tool',
    tool: {
      id: messageId,
      content: argsPreview || `${payload.tool || 'Tool'} running`,
      toolCallId: payload.tool_call_id,
      toolName: payload.tool || formatToolIDName(previewToolName),
      toolStatus: TOOL_RUNNING_STATUS,
      traceId: event.trace_id,
    },
  };
}

function projectToolFinished({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction {
  const payload = parseAgentToolCallFinishedPayload(event.payload);
  const toolStatus = payload.status?.trim() || (payload.error ? TOOL_ERROR_STATUS : TOOL_SUCCESS_STATUS);
  const messageId = resolveToolMessageID(runtime, event, payload.tool_call_id);
  const argsPreview = runtime.previewToolArgs.get(messageId) || '';
  const previewToolName = runtime.previewToolIDByMessageId.get(messageId);
  return {
    type: 'upsert_streaming_tool',
    tool: {
      id: messageId,
      content: payload.error?.trim() || argsPreview || `${payload.tool || 'Tool'} finished`,
      toolCallId: payload.tool_call_id,
      toolName: payload.tool || formatToolIDName(previewToolName),
      toolStatus,
      traceId: event.trace_id,
    },
  };
}

function projectAwaitingHuman({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction {
  const payload = parseAgentAwaitingHumanStreamPayload(event.payload);
  const sessionId = runtime.sessionId.trim();
  if (!sessionId) {
    throw new Error('awaiting_human stream event is missing session_id');
  }

  return {
    type: 'upsert_pending_question',
    question: buildPendingQuestionMessage(runtime, payload, sessionId),
  };
}

function projectMessage({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentStreamMessagePayload(event.payload);
  const finalized = consumeToolTagStreamChunk(runtime.toolTagState, '', true);
  const actions = projectToolTagUnits(runtime, event.trace_id, finalized.units);
  actions.push({ type: 'clear_streaming_assistant_text' });
  actions.push({ type: 'clear_streaming_thinking_text' });
  const thinkingText = runtime.thinkingBuffer.trim();
  runtime.thinkingBuffer = '';
  const committedMessages = thinkingText
    ? [buildThinkingMessage(thinkingText, runtime.thinkingMessageId)]
    : [];

  if (!payload.text.trim()) {
    runtime.assistantBuffer = '';
    if (committedMessages.length > 0) {
      actions.push({
        type: 'append_committed_messages',
        messages: committedMessages,
      });
    }
    return actions;
  }

  const visibleAssistantText = stripToolTagCalls(payload.text);
  if (!visibleAssistantText.trim()) {
    runtime.assistantBuffer = '';
    if (committedMessages.length > 0) {
      actions.push({
        type: 'append_committed_messages',
        messages: committedMessages,
      });
    }
    return actions;
  }

  committedMessages.push(buildAssistantMessage(visibleAssistantText, runtime.assistantMessageId));
  actions.push({
    type: 'append_committed_messages',
    messages: committedMessages,
  });
  runtime.assistantBuffer = '';
  return actions;
}

function buildPendingQuestionMessage(
  runtime: ChatRuntimeState,
  payload: ReturnType<typeof parseAgentAwaitingHumanStreamPayload>,
  sessionId: string,
): PendingQuestionMessage {
  return {
    id: `stream-question:${runtime.traceId}:${payload.question_id}`,
    kind: 'pending_question',
    content: payload.prompt,
    options: payload.options,
    questionId: payload.question_id,
    selectionMode: payload.selection_mode,
    sessionId,
  };
}

function resolveToolMessageID(runtime: ChatRuntimeState, event: AgentStreamEvent, toolCallId?: string): string {
  const trimmedToolCallID = toolCallId?.trim();
  if (trimmedToolCallID) {
    return resolveToolMessageIDWithCallID(runtime, trimmedToolCallID);
  }

  return resolveToolMessageIDWithFallback(runtime, event.step_id.trim() || event.id.trim());
}

function resolveToolMessageIDWithCallID(runtime: ChatRuntimeState, toolCallId: string): string {
  const existing = runtime.toolMessageIds.get(toolCallId);
  if (existing) {
    return existing;
  }

  if (runtime.pendingPreviewQueue.length > 0) {
    const previewMessageID = runtime.pendingPreviewQueue.shift();
    if (previewMessageID) {
      runtime.toolMessageIds.set(toolCallId, previewMessageID);
      return previewMessageID;
    }
  }

  const generated = `stream-tool:${runtime.traceId}:${toolCallId}`;
  runtime.toolMessageIds.set(toolCallId, generated);
  return generated;
}

function resolveToolMessageIDWithFallback(runtime: ChatRuntimeState, key: string): string {
  const existing = runtime.toolMessageIds.get(key);
  if (existing) {
    return existing;
  }

  const generated = `stream-tool:${runtime.traceId}:${key}`;
  runtime.toolMessageIds.set(key, generated);
  return generated;
}

function formatToolIDName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  if (!trimmed) {
    return undefined;
  }
  return `tool#${trimmed}`;
}
