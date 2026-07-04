import type { ChatMessage } from '@/lib/types';
import { filterCommittedMessagesForDisplay } from './messageVisibility';
import { getOrderedStreamingRows } from './streamingRows';
import {
  canAutoCollapseLatestThinkingPanel,
  getLatestStreamingThinkingId,
  hasAssistantStreamedVisibleText,
  hasStreamingThinkingText,
  shouldShowThinkingIndicator,
} from './thinkingState';
import type {
  ChatViewProjection,
  ChatViewInput,
  MessageListMessageRow,
  MessageListProjection,
  MessageListProjectionInput,
  MessageListRow,
  StreamingMessageRow,
  ToolCardViewModelOptions,
} from './types';
import { buildToolCardViewModel } from './tool-details/viewModel';

export function buildChatViewProjection(input: ChatViewInput): ChatViewProjection {
  const visibleCommittedMessages = filterCommittedMessagesForDisplay(
    input.committedMessages,
    input.pendingQuestions,
    input.showSystemPromptMessages,
  );
  const streamingRows = getOrderedStreamingRows(input);
  const showThinkingIndicator = shouldShowThinkingIndicator(input);
  const hasThinkingText = hasStreamingThinkingText(input.streamingThinkingSegments);
  const hasAssistantText = hasAssistantStreamedVisibleText(input.streamingAssistantSegments);

  return {
    visibleCommittedMessages,
    streamingRows,
    showThinkingIndicator,
    latestStreamingThinkingId: getLatestStreamingThinkingId(input.streamingThinkingSegments),
    hasThinkingText,
    hasAssistantText,
    shouldAutoCollapseLatestThinkingPanel: canAutoCollapseLatestThinkingPanel({
      hasThinkingText,
      hasAssistantText,
    }),
  };
}

export function buildMessageListProjection(input: MessageListProjectionInput): MessageListProjection {
  const projection = buildChatViewProjection(input);
  const rows = buildMessageListRows({
    committedMessages: projection.visibleCommittedMessages,
    showThinkingIndicator: projection.showThinkingIndicator,
    streamingRows: projection.streamingRows,
    toolCard: input.toolCard,
  });

  return {
    ...projection,
    rows,
    rowCount: rows.length,
    loading: input.loading,
    loadingOlderHistory: input.loadingOlderHistory,
  };
}

export function buildMessageListRows(options: {
  committedMessages: ChatMessage[];
  showThinkingIndicator: boolean;
  streamingRows: StreamingMessageRow[];
  toolCard?: ToolCardViewModelOptions;
}): MessageListRow[] {
  const rows: MessageListRow[] = [];

  for (const message of options.committedMessages) {
    rows.push(buildMessageRow(message.id, message, options.toolCard));
  }

  for (const row of options.streamingRows) {
    rows.push(buildMessageRow(row.key, row.message, options.toolCard));
  }

  if (options.showThinkingIndicator) {
    rows.push({
      key: 'thinking-indicator',
      kind: 'thinking_indicator',
    });
  }

  return rows;
}

function buildMessageRow(
  key: string,
  message: ChatMessage,
  toolCardOptions?: ToolCardViewModelOptions,
): MessageListMessageRow {
  if (message.kind !== 'tool') {
    return {
      key,
      kind: 'message',
      message,
    };
  }

  return {
    key,
    kind: 'message',
    message,
    toolCard: buildToolCardViewModel(message, toolCardOptions),
  };
}
