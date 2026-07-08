'use client';
import type { FC } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { MessageListRow } from '@/lib/chat-view/types';
import { useWebLocale } from '@/lib/i18n/provider';
import { TopLoadingBar } from '@/components/TopLoadingBar';
import { MessageRow } from './MessageRow';
import { shouldPlaceAssistantCopyInline } from './messageCopyPlacement';
import { buildMessageListLayoutSignature } from './messageListScroll';
import { ThinkingIndicator } from './ThinkingIndicator';
import type { MessageListProps } from './types';
import { useMessageListScroll } from './useMessageListScroll';

export const MessageList: FC<MessageListProps> = ({
  view,
  assistantMarkdownEnabled,
  hasOlderHistory,
  loadOlderHistory,
  onAnswerQuestion,
  onCancelQuestion,
  postSendFocusRequest,
}) => {
  const { copy } = useWebLocale();
  const [openToolCards, setOpenToolCards] = useState<Record<string, boolean>>({});
  const [openThinkingPanels, setOpenThinkingPanels] = useState<Record<string, boolean>>({});
  const [expandedUserMessages, setExpandedUserMessages] = useState<Record<string, boolean>>({});
  const latestStreamingThinkingIdRef = useRef('');
  const previousHasAssistantTextRef = useRef(false);
  const thinkingAutoCollapsedRef = useRef(false);
  const {
    hasAssistantText,
    latestStreamingThinkingId,
    loading,
    loadingOlderHistory,
    rowCount,
    rows,
    showThinkingIndicator,
    shouldAutoCollapseLatestThinkingPanel,
    streamingRows,
    visibleCommittedMessages,
  } = view;
  const thinkingStartedAtMs = useThinkingStartedAtMs(loading);
  const latestStreamingThinkingPanelOpen = latestStreamingThinkingId
    ? Boolean(openThinkingPanels[latestStreamingThinkingId])
    : false;
  const effectiveRows = rows;
  const effectiveRowCount = effectiveRows.length;
  const layoutSignature = buildMessageListLayoutSignature({
    committedMessages: visibleCommittedMessages,
    latestStreamingThinkingId,
    latestStreamingThinkingPanelOpen,
    loadingOlderHistory,
    showThinkingIndicator,
    streamingRows,
  });
  const {
    historySentinelRef,
    olderHistoryLoadingPaused,
    registerMessageRow,
    scrollElementRef,
    scrollToBottom,
    showScrollToBottom,
    trailingSpacerPx,
  } = useMessageListScroll({
    hasOlderHistory,
    layoutSignature,
    loadOlderHistory,
    loadingOlderHistory,
    postSendFocusRequest,
    rowCount,
    visibleCommittedMessages,
  });
  const rowVirtualizer = useVirtualizer({
    count: effectiveRowCount,
    estimateSize: estimateMessageRowHeight,
    getItemKey: (index) => effectiveRows[index]?.key ?? index,
    getScrollElement: () => scrollElementRef.current,
    overscan: 8,
  });
  usePostSendVirtualAnchor({
    postSendFocusRequest,
    rowVirtualizer,
    rows: effectiveRows,
  });
  const showHistoryLoading = olderHistoryLoadingPaused || loadingOlderHistory;
  const virtualItems = rowVirtualizer.getVirtualItems();
  const virtualContentHeight = rowVirtualizer.getTotalSize() + trailingSpacerPx;

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

  const handleToggleUserMessage = useCallback((messageId: string) => {
    setExpandedUserMessages((previous) => ({
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

    if (
      shouldAutoCollapseLatestThinkingPanel
      && !previousHasAssistantTextRef.current
      && !thinkingAutoCollapsedRef.current
    ) {
      setOpenThinkingPanels((previous) => ({
        ...previous,
        [latestStreamingThinkingId]: false,
      }));
      thinkingAutoCollapsedRef.current = true;
    }

    previousHasAssistantTextRef.current = hasAssistantText;
  }, [hasAssistantText, latestStreamingThinkingId, shouldAutoCollapseLatestThinkingPanel]);

  if (effectiveRowCount === 0) {
    return (
      <div className="messages-shell">
        <div ref={scrollElementRef} className="messages is-empty" aria-live="polite" />
      </div>
    );
  }
  return (
    <div className="messages-shell">
      {showHistoryLoading ? (
        <TopLoadingBar className="messages-history-loading-overlay" label={copy.chat.loadingOlderMessages} />
      ) : null}
      <div ref={scrollElementRef} className="messages ui-scroll" aria-live="polite">
        <div
          className="messages-virtual-flow"
          style={{ height: virtualContentHeight }}
        >
          {virtualItems.map((virtualItem) => {
            const index = virtualItem.index;
            const row = effectiveRows[index];
            if (!row) {
              return null;
            }
            const hasTrailingTool = shouldPlaceAssistantCopyInline({
              currentRow: row,
              currentIndex: index,
              rowCount: effectiveRowCount,
              getRowAtIndex: (index) => effectiveRows[index],
            });
            const registerUserRow = row.kind === 'message' && row.message.kind === 'user'
              ? registerMessageRow(row.message.id)
              : undefined;

            return (
              <div
                key={row.key}
                data-index={index}
                data-virtual-index={virtualItem.index}
                className="messages-flow-row"
                ref={(node) => {
                  rowVirtualizer.measureElement(node);
                  registerUserRow?.(node);
                }}
                style={{
                  transform: `translateY(${virtualItem.start}px)`,
                }}
              >
                {renderRow(row, {
                  assistantMarkdownEnabled,
                  expandedUserMessages,
                  hasTrailingTool,
                  loading,
                  thinkingStartedAtMs,
                  openToolCards,
                  onAnswerQuestion,
                  onCancelQuestion,
                  onToggleThinkingPanel: handleToggleThinkingPanel,
                  onToggleToolCard: handleToggleToolCard,
                  onToggleUserMessage: handleToggleUserMessage,
                  openThinkingPanels,
                })}
              </div>
            );
          })}
          {trailingSpacerPx > 0 ? (
            <div
              className="messages-trailing-spacer"
              style={{
                height: trailingSpacerPx,
                transform: `translateY(${rowVirtualizer.getTotalSize()}px)`,
              }}
              aria-hidden="true"
            />
          ) : null}
          <div ref={historySentinelRef} className="messages-history-sentinel" aria-hidden="true" />
        </div>
      </div>
      {showScrollToBottom ? (
        <button
          type="button"
          className="messages-scroll-bottom-button"
          aria-label={copy.chat.scrollToBottom}
          onClick={scrollToBottom}
        >
          <span className="messages-scroll-bottom-icon" aria-hidden="true" />
        </button>
      ) : null}
    </div>
  );
};

function usePostSendVirtualAnchor(options: {
  postSendFocusRequest: MessageListProps['postSendFocusRequest'];
  rowVirtualizer: ReturnType<typeof useVirtualizer<HTMLDivElement, Element>>;
  rows: MessageListRow[];
}) {
  const handledTokenRef = useRef<number | null>(null);

  useLayoutEffect(() => {
    const request = options.postSendFocusRequest;
    if (!request || handledTokenRef.current === request.token) {
      return;
    }

    const rowIndex = options.rows.findIndex((row) => (
      row.kind === 'message'
      && row.message.kind === 'user'
      && row.message.id === request.messageId
    ));
    if (rowIndex < 0) {
      return;
    }

    handledTokenRef.current = request.token;
    options.rowVirtualizer.scrollToIndex(rowIndex, {
      align: 'start',
      behavior: 'auto',
    });
  }, [options.postSendFocusRequest, options.rowVirtualizer, options.rows]);
}

function estimateMessageRowHeight(): number {
  return 112;
}

function renderRow(
  row: MessageListRow,
  options: {
    assistantMarkdownEnabled: boolean;
    expandedUserMessages: Record<string, boolean>;
    hasTrailingTool: boolean;
    loading: boolean;
    thinkingStartedAtMs: number | null;
    onAnswerQuestion: MessageListProps['onAnswerQuestion'];
    onCancelQuestion: MessageListProps['onCancelQuestion'];
    onToggleThinkingPanel: (messageId: string) => void;
    onToggleToolCard: (messageId: string) => void;
    onToggleUserMessage: (messageId: string) => void;
    openThinkingPanels: Record<string, boolean>;
    openToolCards: Record<string, boolean>;
  },
) {
  switch (row.kind) {
    case 'thinking_indicator':
      return <ThinkingIndicator startedAtMs={options.thinkingStartedAtMs} />;
    case 'message':
      return (
        <MessageRow
          message={row.message}
          toolCard={row.toolCard}
          assistantMarkdownEnabled={options.assistantMarkdownEnabled}
          hasTrailingTool={options.hasTrailingTool}
          isToolCardOpen={Boolean(options.openToolCards[row.message.id])}
          isThinkingPanelOpen={Boolean(options.openThinkingPanels[row.message.id])}
          isUserMessageExpanded={Boolean(options.expandedUserMessages[row.message.id])}
          thinkingStartedAtMs={options.thinkingStartedAtMs}
          loading={options.loading}
          onAnswerQuestion={options.onAnswerQuestion}
          onCancelQuestion={options.onCancelQuestion}
          onToggleThinkingPanel={options.onToggleThinkingPanel}
          onToggleToolCard={options.onToggleToolCard}
          onToggleUserMessage={options.onToggleUserMessage}
        />
      );
    default:
      return null;
  }
}

function useThinkingStartedAtMs(loading: boolean): number | null {
  const [startedAtMs, setStartedAtMs] = useState<number | null>(null);
  useEffect(() => {
    if (!loading) {
      setStartedAtMs(null);
      return;
    }

    setStartedAtMs(Date.now());
  }, [loading]);

  return startedAtMs;
}
