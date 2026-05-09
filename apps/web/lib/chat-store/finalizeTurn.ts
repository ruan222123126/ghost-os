import {
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  STREAMING_THINKING_ORDER_PREFIX,
  STREAMING_TOOL_ORDER_PREFIX,
} from '@/lib/chat-stream/streamState';
import { buildThinkingMessage } from '@/lib/chatMessages';
import type { ChatMessage } from '@/lib/types';
import type { ChatStateStore } from './reducer';

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
  const messages: ChatMessage[] = [];
  const seenMessageIDs = new Set<string>();
  for (const orderKey of state.streamingItemOrder) {
    appendStreamingCommitMessage(messages, seenMessageIDs, state, orderKey);
  }
  appendMissingThinkingMessages(messages, seenMessageIDs, state);
  appendMissingToolMessages(messages, seenMessageIDs, state);
  return messages;
}

function appendStreamingCommitMessage(
  messages: ChatMessage[],
  seenMessageIDs: Set<string>,
  state: ChatStateStore,
  orderKey: string,
): void {
  if (orderKey.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    appendThinkingCommitMessage(messages, seenMessageIDs, state, orderKey);
    return;
  }
  if (orderKey.startsWith(STREAMING_TOOL_ORDER_PREFIX)) {
    appendToolCommitMessage(messages, seenMessageIDs, state, orderKey);
  }
}

function appendThinkingCommitMessage(
  messages: ChatMessage[],
  seenMessageIDs: Set<string>,
  state: ChatStateStore,
  orderKey: string,
): void {
  const segmentId = orderKey.slice(STREAMING_THINKING_ORDER_PREFIX.length);
  const segment = state.streamingThinkingState.segmentsById[segmentId];
  if (!segment || seenMessageIDs.has(segmentId)) {
    return;
  }
  messages.push(buildThinkingMessage(segment.content, segment.id));
  seenMessageIDs.add(segmentId);
}

function appendToolCommitMessage(
  messages: ChatMessage[],
  seenMessageIDs: Set<string>,
  state: ChatStateStore,
  orderKey: string,
): void {
  const toolId = orderKey.slice(STREAMING_TOOL_ORDER_PREFIX.length);
  const tool = state.streamingToolState.toolsById[toolId];
  if (!tool || seenMessageIDs.has(toolId)) {
    return;
  }
  messages.push({
    id: tool.id,
    kind: 'tool',
    content: tool.content,
    ...(tool.toolInput ? { toolInput: tool.toolInput } : {}),
    toolCallId: tool.toolCallId,
    toolName: tool.toolName,
    toolStatus: tool.toolStatus,
    traceId: tool.traceId,
  });
  seenMessageIDs.add(toolId);
}

function appendMissingThinkingMessages(
  messages: ChatMessage[],
  seenMessageIDs: Set<string>,
  state: ChatStateStore,
): void {
  for (const segmentId of state.streamingThinkingState.order) {
    if (seenMessageIDs.has(segmentId)) {
      continue;
    }
    const segment = state.streamingThinkingState.segmentsById[segmentId];
    if (!segment) {
      continue;
    }
    messages.push(buildThinkingMessage(segment.content, segment.id));
    seenMessageIDs.add(segmentId);
  }
}

function appendMissingToolMessages(
  messages: ChatMessage[],
  seenMessageIDs: Set<string>,
  state: ChatStateStore,
): void {
  for (const toolId of state.streamingToolState.order) {
    if (seenMessageIDs.has(toolId)) {
      continue;
    }
    const tool = state.streamingToolState.toolsById[toolId];
    if (!tool) {
      continue;
    }
    messages.push({
      id: tool.id,
      kind: 'tool',
      content: tool.content,
      ...(tool.toolInput ? { toolInput: tool.toolInput } : {}),
      toolCallId: tool.toolCallId,
      toolName: tool.toolName,
      toolStatus: tool.toolStatus,
      traceId: tool.traceId,
    });
    seenMessageIDs.add(toolId);
  }
}
