'use client';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { FC } from 'react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { MessageListRow } from '@/lib/chat-view/types';
import { useWebLocale } from '@/lib/i18n/provider';
import { MessageRow } from './MessageRow';
import { shouldPlaceAssistantCopyInline } from './messageCopyPlacement';
import {
  buildMessageListLayoutSignature,
  buildVisibleMessageTailSnapshot,
  getPostSendAnchorIndexFromVisibleMessages,
  hasVisibleContentAfterIndex,
  shouldReleasePostSendAnchor,
} from './messageListScroll';
import { ThinkingIndicator } from './ThinkingIndicator';
import type { MessageListProps } from './types';
import { useMessageListScroll } from './useMessageListScroll';

const MESSAGE_LIST_OVERSCAN = 8;

export const MessageList: FC<MessageListProps> = ({
  view,
  assistantMarkdownEnabled,
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
  const {
    estimatedRowSize,
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
    visibleMessagesForPostSendOverflow,
  } = view;
  const thinkingStartedAtMs = useThinkingStartedAtMs(loading);
  const visibleMessageTailRef = useRef<ReturnType<typeof buildVisibleMessageTailSnapshot> | null>(
    visibleCommittedMessages.length === 0 ? buildVisibleMessageTailSnapshot(visibleCommittedMessages) : null,
  );
  const [postSendAnchorIndex, setPostSendAnchorIndex] = useState<number | null>(null);
  const [postSendToken, setPostSendToken] = useState(0);
  const latestStreamingThinkingPanelOpen = latestStreamingThinkingId
    ? Boolean(openThinkingPanels[latestStreamingThinkingId])
    : false;
  const rowVirtualizer = useVirtualizer({
    count: rowCount,
    estimateSize: () => estimatedRowSize,
    getItemKey: (index) => rows[index].key,
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
  const postSendHasVisibleContent = hasVisibleContentAfterIndex(
    visibleMessagesForPostSendOverflow,
    postSendAnchorIndex,
  );
  const { scrollElementRef, trailingSpacerPx } = useMessageListScroll({
    rowVirtualizer,
    firstVirtualItemIndex: virtualItems[0]?.index ?? null,
    hasOlderHistory,
    layoutSignature,
    loadOlderHistory,
    loadingOlderHistory,
    postSendAnchorIndex,
    postSendHasVisibleContent,
    postSendToken,
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

  useLayoutEffect(() => {
    const previousTail = visibleMessageTailRef.current;
    const anchorIndex = getPostSendAnchorIndexFromVisibleMessages({
      messages: visibleCommittedMessages,
      previousTail,
    });
    visibleMessageTailRef.current = buildVisibleMessageTailSnapshot(visibleCommittedMessages);
    if (anchorIndex === null) {
      if (shouldReleasePostSendAnchor({
        anchorIndex: postSendAnchorIndex,
        loading,
        messages: visibleCommittedMessages,
      })) {
        setPostSendAnchorIndex(null);
      }
      return;
    }

    setPostSendAnchorIndex(anchorIndex);
    setPostSendToken((token) => token + 1);
  }, [loading, postSendAnchorIndex, visibleCommittedMessages]);

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

  if (rowCount === 0) {
    return <div ref={scrollElementRef} className="messages is-empty" aria-live="polite" />;
  }
  return (
    <div ref={scrollElementRef} className="messages ui-scroll" aria-live="polite">
      <div
        className="messages-viewport"
        style={{ height: rowVirtualizer.getTotalSize() + trailingSpacerPx }}
      >
        {virtualItems.map((virtualItem) => {
          const row = rows[virtualItem.index];
          const hasTrailingTool = shouldPlaceAssistantCopyInline(
            row,
            virtualItem.index,
            rowCount,
            (index) => rows[index],
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
    case 'history_loading':
      return (
        <div className="message-row is-history-loading">
          <div className="message-note is-history-loading">{options.copy.chat.loadingOlderMessages}</div>
        </div>
      );
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
