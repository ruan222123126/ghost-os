import type { ChatMessage } from '@/lib/types';
import type { StreamingMessageRow } from './streamingRows';
import type { MessageListRow } from './types';

export function getMessageListRowCount(
  committedMessages: ChatMessage[],
  streamingRows: StreamingMessageRow[],
  showThinkingIndicator: boolean,
  loadingOlderHistory: boolean,
): number {
  return committedMessages.length
    + streamingRows.length
    + (showThinkingIndicator ? 1 : 0)
    + (loadingOlderHistory ? 1 : 0);
}

export function getMessageListRowAtIndex(
  index: number,
  options: {
    committedMessages: ChatMessage[];
    showThinkingIndicator: boolean;
    loadingOlderHistory: boolean;
    streamingRows: StreamingMessageRow[];
  },
): MessageListRow {
  let cursor = index;

  if (options.loadingOlderHistory) {
    if (cursor === 0) {
      return { key: 'history-loading', kind: 'history_loading' };
    }
    cursor -= 1;
  }

  if (cursor < options.committedMessages.length) {
    return {
      key: options.committedMessages[cursor].id,
      kind: 'message',
      message: options.committedMessages[cursor],
    };
  }
  cursor -= options.committedMessages.length;

  if (cursor < options.streamingRows.length) {
    return {
      key: options.streamingRows[cursor].key,
      kind: 'message',
      message: options.streamingRows[cursor].message,
    };
  }
  cursor -= options.streamingRows.length;

  if (options.showThinkingIndicator && cursor === 0) {
    return {
      key: 'thinking-indicator',
      kind: 'thinking_indicator',
    };
  }

  throw new Error(`message row index out of range: ${index}`);
}

export function estimateMessageRowSize(): number {
  return 96;
}
