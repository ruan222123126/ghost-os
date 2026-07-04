import { useVirtualizer } from '@tanstack/react-virtual';
import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { MessageListProjection } from '@/lib/chat-view/types';
import { MessageList } from './MessageList';
import { useMessageListScroll } from './useMessageListScroll';

jest.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: jest.fn(),
}));

jest.mock('./MessageRow', () => ({
  MessageRow: () => React.createElement('div', { className: 'message-row' }),
}));

jest.mock('./useMessageListScroll', () => ({
  useMessageListScroll: jest.fn(),
}));

describe('components/message/MessageList', () => {
  const mockedUseMessageListScroll = useMessageListScroll as jest.MockedFunction<typeof useMessageListScroll>;
  const mockedUseVirtualizer = useVirtualizer as jest.Mock;

  beforeEach(() => {
    mockedUseMessageListScroll.mockReturnValue({
      olderHistoryLoadingPaused: true,
      scrollElementRef: React.createRef<HTMLDivElement>(),
      scrollToBottom: jest.fn(),
      showScrollToBottom: false,
      trailingSpacerPx: 0,
    });
    mockedUseVirtualizer.mockImplementation((config: { count: number }) => ({
      getTotalSize: () => 96,
      getVirtualItems: () => config.count > 0 ? [
        {
          end: 96,
          index: 0,
          key: 'message-1',
          lane: 0,
          size: 96,
          start: 0,
        },
      ] : [],
      measureElement: jest.fn(),
      measurementsCache: [{ start: 0 }],
    }));
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('renders older-history loading as an overlay without adding a virtual row', async () => {
    let renderer!: TestRenderer.ReactTestRenderer;

    await act(async () => {
      renderer = TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          null,
          React.createElement(MessageList, {
            assistantMarkdownEnabled: false,
            hasOlderHistory: true,
            loadOlderHistory: jest.fn(async () => undefined),
            onAnswerQuestion: jest.fn(async () => undefined),
            onCancelQuestion: jest.fn(async () => undefined),
            view: buildMessageListView(),
          }),
        ),
      );
      await Promise.resolve();
    });

    expect(mockedUseVirtualizer).toHaveBeenLastCalledWith(expect.objectContaining({ count: 1 }));
    expect(renderer.root.findAllByProps({
      className: 'top-loading-bar messages-history-loading-overlay',
    })).toHaveLength(1);
  });
});

function buildMessageListView(): MessageListProjection {
  const message = { id: 'message-1', kind: 'assistant' as const, content: 'hello' };

  return {
    estimatedRowSize: 96,
    hasAssistantText: true,
    hasThinkingText: false,
    latestStreamingThinkingId: null,
    loading: false,
    loadingOlderHistory: false,
    rowCount: 1,
    rows: [{ key: message.id, kind: 'message', message }],
    shouldAutoCollapseLatestThinkingPanel: false,
    showThinkingIndicator: false,
    streamingRows: [],
    visibleCommittedMessages: [message],
    visibleMessagesForPostSendOverflow: [message],
  };
}
