import {
  parseAgentAwaitingHumanStreamPayload,
  parseAgentCompletionDeltaPayload,
  parseAgentStreamMessagePayload,
  parseAgentToolCallFinishedPayload,
  parseAgentToolCallStartedPayload,
} from '@/lib/api/agent/parser';
import { consumeToolTagStreamChunk, stripToolTagCalls } from '@/lib/toolTagText';
import type { AgentStreamEvent, PendingQuestionMessage } from '@/lib/types';
import type { ChatRuntimeAction } from './actions';
import {
  appendRuntimeThinkingDelta,
  clearRuntimeThinking,
  markRuntimeThinkingBoundary,
  type ChatRuntimeState,
} from './runtimeState';
import {
  projectToolCallDelta,
  projectToolCallEndDelta,
  projectToolCallStartDelta,
  projectToolFinishedEvent,
  projectToolStartedEvent,
} from './toolCallProjection';
import { projectToolTagUnits } from './toolTagProjection';

interface ProjectAgentEventOptions {
  event: AgentStreamEvent;
  runtime: ChatRuntimeState;
}

const MIN_OVERLAP_DELTA_TRIM_LENGTH = 8;

export function projectAgentEvent(options: ProjectAgentEventOptions): ChatRuntimeAction[] {
  switch (options.event.type) {
    case 'completion_delta':
      return projectCompletionDelta(options);
    case 'tool_call_started':
      return projectToolStarted(options);
    case 'tool_call_finished':
      return projectToolFinished(options);
    case 'awaiting_human':
      return projectAwaitingHuman(options);
    case 'message':
      return projectMessage(options);
    case 'done':
      clearRuntimeThinking(options.runtime);
      return [{ type: 'clear_streaming_thinking_text' }];
    case 'error':
      return [];
    default:
      return [];
  }
}

function projectCompletionDelta({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentCompletionDeltaPayload(event.payload);
  switch (payload.kind) {
    case 'thinking':
      return projectThinkingDelta(runtime, payload.thinking);
    case 'text':
      return projectTextDelta(runtime, event.trace_id, payload.text);
    case 'tool_call_start':
      return projectToolCallStartDelta(runtime, event.trace_id, payload);
    case 'tool_call_delta':
      return projectToolCallDelta(runtime, event.trace_id, payload);
    case 'tool_call_end':
      return projectToolCallEndDelta(runtime, event.trace_id, payload);
    default:
      return [];
  }
}

function projectToolStarted({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentToolCallStartedPayload(event.payload);
  return projectToolStartedEvent(runtime, event, payload);
}

function projectToolFinished({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentToolCallFinishedPayload(event.payload);
  return projectToolFinishedEvent(runtime, event, payload);
}

function projectThinkingDelta(runtime: ChatRuntimeState, thinking?: string): ChatRuntimeAction[] {
  if (!thinking) {
    return [];
  }
  appendRuntimeThinkingDelta(runtime, thinking);
  return [{ type: 'append_streaming_thinking_text', text: thinking }];
}

function projectTextDelta(runtime: ChatRuntimeState, traceId: string, text?: string): ChatRuntimeAction[] {
  const delta = normalizeAssistantTextDelta(runtime.assistantRawBuffer, text);
  if (!delta) {
    return [];
  }
  runtime.assistantRawBuffer = `${runtime.assistantRawBuffer}${delta}`;
  const consumed = consumeToolTagStreamChunk(runtime.toolTagState, delta);
  return projectToolTagUnits(runtime, traceId, consumed.units);
}

function normalizeAssistantTextDelta(assistantBuffer: string, text?: string): string {
  if (!text) {
    return '';
  }
  if (!assistantBuffer || !text.startsWith(assistantBuffer)) {
    return trimOverlappingTextDelta(assistantBuffer, text);
  }
  return text.slice(assistantBuffer.length);
}

function trimOverlappingTextDelta(assistantBuffer: string, text: string): string {
  const overlapLength = longestSuffixPrefixLength(assistantBuffer, text);
  if (overlapLength < MIN_OVERLAP_DELTA_TRIM_LENGTH) {
    return text;
  }
  return text.slice(overlapLength);
}

function longestSuffixPrefixLength(left: string, right: string): number {
  const maxLength = Math.min(left.length, right.length);
  for (let length = maxLength; length > 0; length -= 1) {
    if (left.endsWith(right.slice(0, length))) {
      return length;
    }
  }
  return 0;
}

function projectAwaitingHuman({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentAwaitingHumanStreamPayload(event.payload);
  const sessionId = runtime.sessionId.trim();
  if (!sessionId) {
    throw new Error('awaiting_human stream event is missing session_id');
  }

  markRuntimeThinkingBoundary(runtime);
  return [
    {
      type: 'mark_streaming_thinking_boundary',
    },
    {
      type: 'upsert_pending_question',
      question: buildPendingQuestionMessage(runtime, payload, sessionId),
    },
  ];
}

function projectMessage({ event, runtime }: ProjectAgentEventOptions): ChatRuntimeAction[] {
  const payload = parseAgentStreamMessagePayload(event.payload);
  const finalized = consumeToolTagStreamChunk(runtime.toolTagState, '', true);
  const actions = projectToolTagUnits(runtime, event.trace_id, finalized.units);
  actions.push({
    type: 'finalize_streaming_turn',
    assistantMessageId: runtime.assistantMessageId,
    assistantText: resolveVisibleAssistantText(payload.text, runtime.assistantBuffer),
  });
  runtime.assistantBuffer = '';
  runtime.assistantRawBuffer = '';
  clearRuntimeThinking(runtime);
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

function resolveVisibleAssistantText(messageText: string, assistantBuffer: string): string {
  const finalizedText = stripToolTagCalls(messageText);
  if (finalizedText.trim()) {
    return finalizedText;
  }
  return assistantBuffer;
}
