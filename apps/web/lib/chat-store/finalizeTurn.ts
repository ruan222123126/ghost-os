import {
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  STREAMING_THINKING_ORDER_PREFIX,
  STREAMING_TOOL_ORDER_PREFIX,
} from '@/lib/chat-stream/streamState';
import { buildThinkingMessage } from '@/lib/chatMessages';
import type { ChatMessage, StreamingToolState } from '@/lib/types';
import type { ChatStateStore } from './reducer';

interface StreamingCommitContext {
  messages: ChatMessage[];
  seenMessageIDs: Set<string>;
  state: ChatStateStore;
}

export function finalizeStreamingTurnState(
  state: ChatStateStore,
  assistantMessageId: string,
  assistantText: string,
): ChatStateStore {
  const finalizedMessages = buildFinalizedTurnMessages(state, assistantMessageId, assistantText);
  return {
    ...state,
    committedMessages: finalizedMessages.length
      ? [...state.committedMessages, ...finalizedMessages]
      : state.committedMessages,
    streamingAssistantState: clearStreamingAssistantState(),
    streamingThinkingState: clearStreamingThinkingState(),
    streamingItemOrder: [],
    streamingToolState: clearStreamingToolState(),
  };
}

function buildFinalizedTurnMessages(
  state: ChatStateStore,
  assistantMessageId: string,
  assistantText: string,
): ChatMessage[] {
  const orderedMessages = buildOrderedStreamingCommitMessages(state);
  if (!assistantText.trim()) {
    return orderedMessages;
  }
  return [
    ...orderedMessages,
    {
      id: assistantMessageId,
      kind: 'assistant',
      content: assistantText,
    },
  ];
}

function buildOrderedStreamingCommitMessages(state: ChatStateStore): ChatMessage[] {
  const context: StreamingCommitContext = {
    messages: [],
    seenMessageIDs: new Set<string>(),
    state,
  };
  for (const orderKey of state.streamingItemOrder) {
    appendStreamingCommitMessage(context, orderKey);
  }
  appendMissingThinkingMessages(context);
  appendMissingToolMessages(context);
  return context.messages;
}

function appendStreamingCommitMessage(
  context: StreamingCommitContext,
  orderKey: string,
): void {
  if (orderKey.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    appendThinkingCommitMessage(context, orderKey);
    return;
  }
  if (orderKey.startsWith(STREAMING_TOOL_ORDER_PREFIX)) {
    appendToolCommitMessage(context, orderKey);
  }
}

function appendThinkingCommitMessage(
  context: StreamingCommitContext,
  orderKey: string,
): void {
  const segmentId = orderKey.slice(STREAMING_THINKING_ORDER_PREFIX.length);
  appendThinkingSegment(context, segmentId);
}

function appendToolCommitMessage(
  context: StreamingCommitContext,
  orderKey: string,
): void {
  const toolId = orderKey.slice(STREAMING_TOOL_ORDER_PREFIX.length);
  appendToolMessage(context, toolId);
}

function appendMissingThinkingMessages(context: StreamingCommitContext): void {
  for (const segmentId of context.state.streamingThinkingState.order) {
    appendThinkingSegment(context, segmentId);
  }
}

function appendMissingToolMessages(context: StreamingCommitContext): void {
  for (const toolId of context.state.streamingToolState.order) {
    appendToolMessage(context, toolId);
  }
}

function appendThinkingSegment(context: StreamingCommitContext, segmentId: string): void {
  if (context.seenMessageIDs.has(segmentId)) {
    return;
  }
  const segment = context.state.streamingThinkingState.segmentsById[segmentId];
  if (!segment) {
    return;
  }
  context.messages.push(buildThinkingMessage(segment.content, segment.id));
  context.seenMessageIDs.add(segmentId);
}

function appendToolMessage(context: StreamingCommitContext, toolId: string): void {
  if (context.seenMessageIDs.has(toolId)) {
    return;
  }
  const tool = context.state.streamingToolState.toolsById[toolId];
  if (!tool) {
    return;
  }
  context.messages.push(buildToolCommitMessage(tool));
  context.seenMessageIDs.add(toolId);
}

function buildToolCommitMessage(tool: StreamingToolState): ChatMessage {
  return {
    id: tool.id,
    kind: 'tool',
    content: tool.content,
    ...(tool.toolInput ? { toolInput: tool.toolInput } : {}),
    toolCallId: tool.toolCallId,
    toolName: tool.toolName,
    toolStatus: tool.toolStatus,
    traceId: tool.traceId,
  };
}
