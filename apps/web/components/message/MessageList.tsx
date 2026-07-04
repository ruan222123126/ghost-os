'use client';
import type { FC } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
  const renderedRows = useMemo(() => {
    return effectiveRows
      .map((row, index) => ({ index, row }))
      .reverse();
  }, [effectiveRows]);
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
    streamingRows,
    visibleCommittedMessages,
  });
  const showHistoryLoading = olderHistoryLoadingPaused || loadingOlderHistory;

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
      <div ref={scrollElementRef} className="messages ui-scroll is-reverse-flow" aria-live="polite">
        {trailingSpacerPx > 0 ? (
          <div
            className="messages-trailing-spacer"
            style={{ height: trailingSpacerPx }}
            aria-hidden="true"
          />
        ) : null}
        {renderedRows.map(({ index, row }) => {
          const hasTrailingTool = shouldPlaceAssistantCopyInline({
            currentRow: row,
            currentIndex: index,
            rowCount: effectiveRowCount,
            getRowAtIndex: (index) => effectiveRows[index],
          });

          return (
            <div
              key={row.key}
              data-index={index}
              className="messages-flow-row"
              ref={row.kind === 'message' && row.message.kind === 'user'
                ? registerMessageRow(row.message.id)
                : undefined}
            >
              {renderRow(row, {
                assistantMarkdownEnabled,
                hasTrailingTool,
                loading,
                thinkingStartedAtMs,
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
        <div ref={historySentinelRef} className="messages-history-sentinel" aria-hidden="true" />
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

function renderRow(
  row: MessageListRow,
  options: {
    assistantMarkdownEnabled: boolean;
    hasTrailingTool: boolean;
    loading: boolean;
    thinkingStartedAtMs: number | null;
    onAnswerQuestion: MessageListProps['onAnswerQuestion'];
    onCancelQuestion: MessageListProps['onCancelQuestion'];
    onToggleThinkingPanel: (messageId: string) => void;
    onToggleToolCard: (messageId: string) => void;
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
          thinkingStartedAtMs={options.thinkingStartedAtMs}
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
