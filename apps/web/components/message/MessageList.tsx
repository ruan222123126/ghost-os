'use client';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { FC } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { ChatMessage, PendingQuestionMessage } from '@/lib/types';
import { EmptyState } from './EmptyState';
import { MessageRow } from './MessageRow';
import { getOrderedStreamingRows, type StreamingMessageRow } from './streamingRows';
import { ThinkingIndicator } from './ThinkingIndicator';
import { shouldShowThinkingIndicator } from './thinkingState';
import type { MessageListProps, MessageListRow } from './types';
const BOTTOM_FOLLOW_THRESHOLD_PX = 120;
const LOAD_OLDER_TRIGGER_ROWS = 5;
const MESSAGE_LIST_OVERSCAN = 8;

export const MessageList: FC<MessageListProps> = ({
  committedMessages,
  streamingAssistantSegments,
  streamingItemOrder,
  streamingTools,
  pendingQuestions,
  loading,
  loadingOlderHistory,
  hasOlderHistory,
  loadOlderHistory,
  onAnswerQuestion,
  onCancelQuestion,
}) => {
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const prependAnchorRef = useRef<{ scrollHeight: number; scrollTop: number } | null>(null);
  const shouldAutoFollowRef = useRef(true);
  const olderLoadPendingRef = useRef(false);
  const [openToolCards, setOpenToolCards] = useState<Record<string, boolean>>({});
  const visibleCommittedMessages = getVisibleCommittedMessages(committedMessages, pendingQuestions);
  const streamingRows = getOrderedStreamingRows({
    pendingQuestions,
    streamingAssistantSegments,
    streamingItemOrder,
    streamingTools,
  });
  const showThinking = shouldShowThinkingIndicator({ loading, streamingAssistantSegments, streamingTools });
  const rowCount = getRowCount(
    visibleCommittedMessages,
    streamingRows,
    showThinking,
    loadingOlderHistory,
  );
  const rowVirtualizer = useVirtualizer({
    count: rowCount,
    estimateSize: estimateMessageRowSize,
    getItemKey: (index) => getRowAtIndex(index, {
      committedMessages: visibleCommittedMessages,
      showThinking,
      loadingOlderHistory,
      streamingRows,
    }).key,
    getScrollElement: () => scrollElementRef.current,
    overscan: MESSAGE_LIST_OVERSCAN,
  });
  const virtualItems = rowVirtualizer.getVirtualItems();

  const handleToggleToolCard = useCallback((messageId: string) => {
    setOpenToolCards((previous) => ({
      ...previous,
      [messageId]: !previous[messageId],
    }));
  }, []);

  const handleLoadOlderHistory = useCallback(async () => {
    if (!hasOlderHistory || loadingOlderHistory || olderLoadPendingRef.current) {
      return;
    }

    const container = scrollElementRef.current;
    if (container) {
      prependAnchorRef.current = {
        scrollHeight: container.scrollHeight,
        scrollTop: container.scrollTop,
      };
    }

    olderLoadPendingRef.current = true;
    try {
      await loadOlderHistory();
    } finally {
      olderLoadPendingRef.current = false;
    }
  }, [hasOlderHistory, loadOlderHistory, loadingOlderHistory]);

  useEffect(() => {
    const container = scrollElementRef.current;
    if (!container) {
      return;
    }

    const handleScroll = () => {
      shouldAutoFollowRef.current = isNearBottom(container);
    };

    handleScroll();
    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => {
      container.removeEventListener('scroll', handleScroll);
    };
  }, []);

  useEffect(() => {
    if (!hasOlderHistory || loadingOlderHistory || virtualItems.length === 0) {
      return;
    }
    if (virtualItems[0].index > LOAD_OLDER_TRIGGER_ROWS) {
      return;
    }

    void handleLoadOlderHistory();
  }, [handleLoadOlderHistory, hasOlderHistory, loadingOlderHistory, virtualItems]);

  useLayoutEffect(() => {
    const anchor = prependAnchorRef.current;
    const container = scrollElementRef.current;
    if (!anchor || !container) {
      return;
    }

    container.scrollTop = anchor.scrollTop + (container.scrollHeight - anchor.scrollHeight);
    prependAnchorRef.current = null;
  }, [loadingOlderHistory, visibleCommittedMessages.length]);

  useLayoutEffect(() => {
    const container = scrollElementRef.current;
    if (!container || prependAnchorRef.current) {
      return;
    }
    if (!shouldAutoFollowRef.current && !loading) {
      return;
    }

    container.scrollTop = container.scrollHeight;
  }, [
    loading,
    streamingRows.length,
    visibleCommittedMessages.length,
  ]);

  if (rowCount === 0) {
    return <EmptyState />;
  }
  return (
    <div ref={scrollElementRef} className="messages ui-scroll" aria-live="polite">
      <div className="messages-viewport" style={{ height: rowVirtualizer.getTotalSize() }}>
        {virtualItems.map((virtualItem) => {
          const row = getRowAtIndex(virtualItem.index, {
            committedMessages: visibleCommittedMessages,
            showThinking,
            loadingOlderHistory,
            streamingRows,
          });

          return (
            <div
              key={row.key}
              data-index={virtualItem.index}
              ref={(node) => {
                rowVirtualizer.measureElement(node);
              }}
              className="messages-virtual-row"
              style={{ transform: `translateY(${virtualItem.start}px)` }}
            >
              {renderRow(row, {
                loading,
                openToolCards,
                onAnswerQuestion,
                onCancelQuestion,
                onToggleToolCard: handleToggleToolCard,
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
};

function renderRow(
  row: MessageListRow,
  options: {
    loading: boolean;
    onAnswerQuestion: MessageListProps['onAnswerQuestion'];
    onCancelQuestion: MessageListProps['onCancelQuestion'];
    onToggleToolCard: (messageId: string) => void;
    openToolCards: Record<string, boolean>;
  },
) {
  switch (row.kind) {
    case 'history_loading':
      return (
        <div className="message-row is-history-loading">
          <div className="message-note is-history-loading">Loading older messages...</div>
        </div>
      );
    case 'thinking':
      return <ThinkingIndicator />;
    case 'message':
      return (
        <MessageRow
          message={row.message}
          isToolCardOpen={Boolean(options.openToolCards[row.message.id])}
          loading={options.loading}
          onAnswerQuestion={options.onAnswerQuestion}
          onCancelQuestion={options.onCancelQuestion}
          onToggleToolCard={options.onToggleToolCard}
        />
      );
    default:
      return null;
  }
}

function getRowCount(
  committedMessages: ChatMessage[],
  streamingRows: StreamingMessageRow[],
  showThinking: boolean,
  loadingOlderHistory: boolean,
): number {
  return committedMessages.length
    + streamingRows.length
    + (showThinking ? 1 : 0)
    + (loadingOlderHistory ? 1 : 0);
}

function getRowAtIndex(
  index: number,
  options: {
    committedMessages: ChatMessage[];
    showThinking: boolean;
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

  if (options.showThinking && cursor === 0) {
    return { key: 'thinking', kind: 'thinking' };
  }

  throw new Error(`message row index out of range: ${index}`);
}

function getVisibleCommittedMessages(
  committedMessages: ChatMessage[],
  pendingQuestions: PendingQuestionMessage[],
): ChatMessage[] {
  if (pendingQuestions.length === 0) {
    return committedMessages;
  }

  const pendingQuestionIDs = new Set(
    pendingQuestions.map((question) => question.questionId),
  );

  return committedMessages.filter((message) => {
    return message.kind !== 'question'
      || !message.questionId
      || !pendingQuestionIDs.has(message.questionId);
  });
}

function estimateMessageRowSize(): number {
  return 96;
}

function isNearBottom(element: HTMLElement): boolean {
  const distance = element.scrollHeight - element.clientHeight - element.scrollTop;
  return distance <= BOTTOM_FOLLOW_THRESHOLD_PX;
}
