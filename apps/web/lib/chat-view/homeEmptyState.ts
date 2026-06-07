import { buildChatViewProjection } from './messageRows';
import type { ChatViewInput } from './types';

interface HomeEmptyStateOptions {
  currentSessionId: string;
  chat: Omit<ChatViewInput, 'showSystemPromptMessages'> & {
    historyLoading: boolean;
  };
  showSystemPromptMessages: boolean;
}

export function shouldShowHomeEmptyState(options: HomeEmptyStateOptions): boolean {
  if (options.currentSessionId.trim()) {
    return false;
  }

  if (options.chat.historyLoading) {
    return false;
  }

  const projection = buildChatViewProjection({
    ...options.chat,
    showSystemPromptMessages: options.showSystemPromptMessages,
  });
  if (projection.visibleCommittedMessages.length > 0) {
    return false;
  }

  if (projection.streamingRows.length > 0) {
    return false;
  }

  return !projection.showThinkingIndicator;
}
