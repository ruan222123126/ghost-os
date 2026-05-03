'use client';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { FC } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import { EmptyState } from './EmptyState';
import { MessageRow } from './MessageRow';
import {
  estimateMessageRowSize,
  getMessageListRowAtIndex,
  getMessageListRowCount,
} from './messageListRows';
import { shouldPlaceAssistantCopyInline } from './messageCopyPlacement';
import { buildMessageListLayoutSignature } from './messageListScroll';
import { filterCommittedMessagesForDisplay } from './messageVisibility';
import { getOrderedStreamingRows, type StreamingMessageRow } from './streamingRows';
import { ThinkingIndicator } from './ThinkingIndicator';
import {
  getLatestStreamingThinkingId,
  hasAssistantStreamedVisibleText,
  hasStreamingThinkingText,
  shouldAutoCollapseThinkingPanel,
  shouldShowThinkingIndicator,
} from './thinkingState';
import type { MessageListProps, MessageListRow } from './types';
import { useMessageListScroll } from './useMessageListScroll';

const MESSAGE_LIST_OVERSCAN = 8;

export const MessageList: FC<MessageListProps> = ({
  committedMessages,
  showSystemPromptMessages,
  assistantMarkdownEnabled,
  toolCallCompactOutputEnabled,
  streamingAssistantSegments,
  streamingThinkingSegments,
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
  const { copy } = useWebLocale();
  const [openToolCards, setOpenToolCards] = useState<Record<string, boolean>>({});
  const [openThinkingPanels, setOpenThinkingPanels] = useState<Record<string, boolean>>({});
  const latestStreamingThinkingIdRef = useRef('');
  const previousHasAssistantTextRef = useRef(false);
  const thinkingAutoCollapsedRef = useRef(false);
  const visibleCommittedMessages = filterCommittedMessagesForDisplay(
    committedMessages,
    pendingQuestions,
    showSystemPromptMessages,
  );
  const streamingRows = getOrderedStreamingRows({
    pendingQuestions,
    streamingAssistantSegments,
    streamingThinkingSegments,
    streamingItemOrder,
    streamingTools,
  });
  const hasThinkingText = hasStreamingThinkingText(streamingThinkingSegments);
  const hasAssistantText = hasAssistantStreamedVisibleText(streamingAssistantSegments);
  const latestStreamingThinkingId = getLatestStreamingThinkingId(streamingThinkingSegments);
  const latestStreamingThinkingPanelOpen = latestStreamingThinkingId
    ? Boolean(openThinkingPanels[latestStreamingThinkingId])
    : false;
  const showThinkingIndicator = shouldShowThinkingIndicator({
    loading,
    streamingThinkingSegments,
    streamingAssistantSegments,
    streamingTools,
  });
  const messageRowSource = {
    committedMessages: visibleCommittedMessages,
    showThinkingIndicator,
    loadingOlderHistory,
    streamingRows,
  };
  const rowCount = getMessageListRowCount(
    visibleCommittedMessages,
    streamingRows,
    showThinkingIndicator,
    loadingOlderHistory,
  );
  const rowVirtualizer = useVirtualizer({
    count: rowCount,
    estimateSize: estimateMessageRowSize,
    getItemKey: (index) => getMessageListRowAtIndex(index, messageRowSource).key,
    getScrollElement: () => scrollElementRef.current,
    overscan: MESSAGE_LIST_OVERSCAN,
    useAnimationFrameWithResizeObserver: true,
  });
  const virtualItems = rowVirtualizer.getVirtualItems();
  const measureMessageRow = rowVirtualizer.measureElement;
  const layoutSignature = buildMessageListLayoutSignature({
    committedMessages: visibleCommittedMessages,
    latestStreamingThinkingId,
    latestStreamingThinkingPanelOpen,
    loadingOlderHistory,
    showThinkingIndicator,
    streamingRows,
  });
  const { scrollElementRef } = useMessageListScroll({
    rowVirtualizer,
    firstVirtualItemIndex: virtualItems[0]?.index ?? null,
    hasOlderHistory,
    layoutSignature,
    loadOlderHistory,
    loadingOlderHistory,
    visibleCommittedMessageCount: visibleCommittedMessages.length,
  });

  const handleToggleToolCard = useCallback((messageId: string) => {
    setOpenToolCards((previous) => ({
      ...previous,
      [messageId]: !previous[messageId],
    }));
  }, []);

  const handleToggleThinkingPanel = useCallback((messageId: string) => {
    setOpenThinkingPanels((previous) => ({
      ...previous,
      [messageId]: !previous[messageId],
    }));
  }, []);

  useEffect(() => {
    if (!latestStreamingThinkingId) {
      latestStreamingThinkingIdRef.current = '';
      previousHasAssistantTextRef.current = hasAssistantText;
      thinkingAutoCollapsedRef.current = false;
      return;
    }

    if (latestStreamingThinkingIdRef.current !== latestStreamingThinkingId) {
      setOpenThinkingPanels((previous) => ({
        ...previous,
        [latestStreamingThinkingId]: true,
      }));
      latestStreamingThinkingIdRef.current = latestStreamingThinkingId;
      thinkingAutoCollapsedRef.current = false;
    }

    if (shouldAutoCollapseThinkingPanel({
      hasThinkingText,
      hasAssistantText,
      hadAssistantText: previousHasAssistantTextRef.current,
      alreadyAutoCollapsed: thinkingAutoCollapsedRef.current,
    })) {
      setOpenThinkingPanels((previous) => ({
        ...previous,
        [latestStreamingThinkingId]: false,
      }));
      thinkingAutoCollapsedRef.current = true;
    }

    previousHasAssistantTextRef.current = hasAssistantText;
  }, [hasAssistantText, hasThinkingText, latestStreamingThinkingId]);

  if (rowCount === 0) {
    return <EmptyState />;
  }
  return (
    <div ref={scrollElementRef} className="messages ui-scroll" aria-live="polite">
      <div className="messages-viewport" style={{ height: rowVirtualizer.getTotalSize() }}>
        {virtualItems.map((virtualItem) => {
          const row = getMessageListRowAtIndex(virtualItem.index, messageRowSource);
          const hasTrailingTool = shouldPlaceAssistantCopyInline(
            row,
            virtualItem.index,
            rowCount,
            (index) => getMessageListRowAtIndex(index, messageRowSource),
          );

          return (
            <div
              key={row.key}
              data-index={virtualItem.index}
              ref={measureMessageRow}
              className="messages-virtual-row"
              style={{ transform: `translateY(${virtualItem.start}px)` }}
            >
              {renderRow(row, {
                copy,
                assistantMarkdownEnabled,
                hasTrailingTool,
                toolCallCompactOutputEnabled,
                loading,
                openToolCards,
                onAnswerQuestion,
                onCancelQuestion,
                onToggleThinkingPanel: handleToggleThinkingPanel,
                onToggleToolCard: handleToggleToolCard,
                openThinkingPanels,
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
    copy: ReturnType<typeof useWebLocale>['copy'];
    assistantMarkdownEnabled: boolean;
    hasTrailingTool: boolean;
    toolCallCompactOutputEnabled: boolean;
    loading: boolean;
    onAnswerQuestion: MessageListProps['onAnswerQuestion'];
    onCancelQuestion: MessageListProps['onCancelQuestion'];
    onToggleThinkingPanel: (messageId: string) => void;
    onToggleToolCard: (messageId: string) => void;
    openThinkingPanels: Record<string, boolean>;
    openToolCards: Record<string, boolean>;
  },
) {
  switch (row.kind) {
    case 'history_loading':
      return (
        <div className="message-row is-history-loading">
          <div className="message-note is-history-loading">{options.copy.chat.loadingOlderMessages}</div>
        </div>
      );
    case 'thinking_indicator':
      return <ThinkingIndicator />;
    case 'message':
      return (
        <MessageRow
          message={row.message}
          assistantMarkdownEnabled={options.assistantMarkdownEnabled}
          hasTrailingTool={options.hasTrailingTool}
          toolCallCompactOutputEnabled={options.toolCallCompactOutputEnabled}
          isToolCardOpen={Boolean(options.openToolCards[row.message.id])}
          isThinkingPanelOpen={Boolean(options.openThinkingPanels[row.message.id])}
          loading={options.loading}
          onAnswerQuestion={options.onAnswerQuestion}
          onCancelQuestion={options.onCancelQuestion}
          onToggleThinkingPanel={options.onToggleThinkingPanel}
          onToggleToolCard={options.onToggleToolCard}
        />
      );
    default:
      return null;
  }
}
