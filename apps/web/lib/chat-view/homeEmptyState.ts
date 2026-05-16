import { filterCommittedMessagesForDisplay } from '@/components/message/messageVisibility';
import { shouldShowThinkingIndicator } from '@/components/message/thinkingState';
import { getOrderedStreamingRows } from '@/lib/chat-view/streamingRows';
import type { UseBridgeChatResult } from '@/hooks/chat/types';

interface HomeEmptyStateOptions {
  currentSessionId: string;
  chat: Pick<
    UseBridgeChatResult,
    | 'committedMessages'
    | 'pendingQuestions'
    | 'streamingAssistantSegments'
    | 'streamingThinkingSegments'
    | 'streamingItemOrder'
    | 'streamingTools'
    | 'loading'
    | 'historyLoading'
  >;
  showSystemPromptMessages: boolean;
}

export function shouldShowHomeEmptyState(options: HomeEmptyStateOptions): boolean {
  if (options.currentSessionId.trim()) {
    return false;
  }

  if (options.chat.historyLoading) {
    return false;
  }

  const visibleCommittedMessages = filterCommittedMessagesForDisplay(
    options.chat.committedMessages,
    options.chat.pendingQuestions,
    options.showSystemPromptMessages,
  );
  if (visibleCommittedMessages.length > 0) {
    return false;
  }

  const streamingRows = getOrderedStreamingRows(options.chat);
  if (streamingRows.length > 0) {
    return false;
  }

  return !shouldShowThinkingIndicator(options.chat);
}
