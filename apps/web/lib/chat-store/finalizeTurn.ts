import {
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  STREAMING_ASSISTANT_ORDER_PREFIX,
  STREAMING_THINKING_ORDER_PREFIX,
  STREAMING_TOOL_ORDER_PREFIX,
} from '@/lib/chat-stream/streamState';
import { buildAssistantMessage, buildThinkingMessage } from '@/lib/chatMessages';
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
  const assistantSegmentContents = resolveAssistantSegmentContents(state, assistantText);
  const orderedMessages = buildOrderedStreamingCommitMessages(state, assistantSegmentContents);
  if (!assistantText.trim() || hasCommittedAssistantSegments(state, orderedMessages)) {
    return orderedMessages;
  }
  return [...orderedMessages, buildAssistantMessage(assistantText, assistantMessageId)];
}

function buildOrderedStreamingCommitMessages(
  state: ChatStateStore,
  assistantSegmentContents: Map<string, string>,
): ChatMessage[] {
  const context: StreamingCommitContext = {
    messages: [],
    seenMessageIDs: new Set<string>(),
    state,
  };
  for (const orderKey of state.streamingItemOrder) {
    appendStreamingCommitMessage(context, orderKey, assistantSegmentContents);
  }
  appendMissingThinkingMessages(context);
  appendMissingToolMessages(context);
  appendMissingAssistantMessages(context, assistantSegmentContents);
  return context.messages;
}

function appendStreamingCommitMessage(
  context: StreamingCommitContext,
  orderKey: string,
  assistantSegmentContents: Map<string, string>,
): void {
  if (orderKey.startsWith(STREAMING_ASSISTANT_ORDER_PREFIX)) {
    appendAssistantCommitMessage(context, orderKey, assistantSegmentContents);
    return;
  }
  if (orderKey.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    appendThinkingCommitMessage(context, orderKey);
    return;
  }
  if (orderKey.startsWith(STREAMING_TOOL_ORDER_PREFIX)) {
    appendToolCommitMessage(context, orderKey);
  }
}

function appendAssistantCommitMessage(
  context: StreamingCommitContext,
  orderKey: string,
  assistantSegmentContents: Map<string, string>,
): void {
  const segmentId = orderKey.slice(STREAMING_ASSISTANT_ORDER_PREFIX.length);
  appendAssistantSegment(context, segmentId, assistantSegmentContents);
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

function appendMissingAssistantMessages(
  context: StreamingCommitContext,
  assistantSegmentContents: Map<string, string>,
): void {
  for (const segmentId of context.state.streamingAssistantState.order) {
    appendAssistantSegment(context, segmentId, assistantSegmentContents);
  }
}

function appendAssistantSegment(
  context: StreamingCommitContext,
  segmentId: string,
  assistantSegmentContents: Map<string, string>,
): void {
  if (context.seenMessageIDs.has(segmentId)) {
    return;
  }
  const segment = context.state.streamingAssistantState.segmentsById[segmentId];
  const content = assistantSegmentContents.get(segmentId);
  if (!segment || content === undefined) {
    return;
  }
  context.messages.push(buildAssistantMessage(content, segment.id));
  context.seenMessageIDs.add(segmentId);
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

function resolveAssistantSegmentContents(
  state: ChatStateStore,
  assistantText: string,
): Map<string, string> {
  const contents = new Map<string, string>();
  for (const segmentId of state.streamingAssistantState.order) {
    const segment = state.streamingAssistantState.segmentsById[segmentId];
    if (segment) {
      contents.set(segmentId, segment.content);
    }
  }
  if (contents.size === 0) {
    return contents;
  }

  const orderedSegmentIDs = state.streamingAssistantState.order.filter((segmentId) => contents.has(segmentId));
  const mergedText = orderedSegmentIDs.map((segmentId) => contents.get(segmentId) || '').join('');

  if (orderedSegmentIDs.length === 1 && assistantText && assistantText !== mergedText) {
    contents.set(orderedSegmentIDs[0], assistantText);
    return contents;
  }

  if (assistantText.startsWith(mergedText) && assistantText !== mergedText) {
    const lastSegmentId = orderedSegmentIDs.at(-1);
    if (lastSegmentId) {
      contents.set(lastSegmentId, `${contents.get(lastSegmentId) || ''}${assistantText.slice(mergedText.length)}`);
    }
  }

  return contents;
}

function hasCommittedAssistantSegments(
  state: ChatStateStore,
  messages: ChatMessage[],
): boolean {
  if (state.streamingAssistantState.order.length === 0) {
    return false;
  }
  const assistantIDs = new Set(state.streamingAssistantState.order);
  return messages.some((message) => assistantIDs.has(message.id));
}
