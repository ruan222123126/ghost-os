import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { MessageListProjection } from '@/lib/chat-view/types';
import { MessageList } from './MessageList';
import { useMessageListScroll } from './useMessageListScroll';

jest.mock('./MessageRow', () => ({
  MessageRow: () => React.createElement('div', { className: 'message-row' }),
}));

const mockScrollToIndex = jest.fn();
let mockVisibleVirtualIndexes: number[] | null = null;

jest.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getTotalSize: () => count * 112,
    getVirtualItems: () => Array.from(
      mockVisibleVirtualIndexes ?? Array.from({ length: count }, (_, index) => index),
      (index) => ({
        index,
        key: `virtual-${index}`,
        start: index * 112,
      }),
    ),
    measureElement: jest.fn(),
    scrollToIndex: mockScrollToIndex,
  }),
}));

jest.mock('./useMessageListScroll', () => ({
  useMessageListScroll: jest.fn(),
}));

describe('components/message/MessageList', () => {
  const mockedUseMessageListScroll = useMessageListScroll as jest.MockedFunction<typeof useMessageListScroll>;

  beforeEach(() => {
    mockedUseMessageListScroll.mockReturnValue({
      historySentinelRef: React.createRef<HTMLDivElement>(),
      olderHistoryLoadingPaused: true,
      registerMessageRow: jest.fn(() => jest.fn()),
      scrollElementRef: React.createRef<HTMLDivElement>(),
      scrollToBottom: jest.fn(),
      showScrollToBottom: false,
      trailingSpacerPx: 0,
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
    mockVisibleVirtualIndexes = null;
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
            postSendFocusRequest: null,
            view: buildMessageListView(),
          }),
        ),
      );
      await Promise.resolve();
    });

    expect(renderer.root.findAllByProps({
      className: 'top-loading-bar messages-history-loading-overlay',
    })).toHaveLength(1);
    expect(renderer.root.findAllByProps({ className: 'messages-flow-row' })).toHaveLength(1);
  });

  it('does not let the virtual list override post-send focus when the requested user message is visible', async () => {
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          null,
          React.createElement(MessageList, {
            assistantMarkdownEnabled: false,
            hasOlderHistory: false,
            loadOlderHistory: jest.fn(async () => undefined),
            onAnswerQuestion: jest.fn(async () => undefined),
            onCancelQuestion: jest.fn(async () => undefined),
            postSendFocusRequest: { messageId: 'user-2', token: 1 },
            view: buildMessageListViewWithUsers(),
          }),
        ),
      );
      await Promise.resolve();
    });

    expect(mockScrollToIndex).not.toHaveBeenCalled();
  });

  it('smoothly scrolls the virtual list to render an offscreen post-send user message', async () => {
    mockVisibleVirtualIndexes = [0, 1];

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          null,
          React.createElement(MessageList, {
            assistantMarkdownEnabled: false,
            hasOlderHistory: false,
            loadOlderHistory: jest.fn(async () => undefined),
            onAnswerQuestion: jest.fn(async () => undefined),
            onCancelQuestion: jest.fn(async () => undefined),
            postSendFocusRequest: { messageId: 'user-2', token: 1 },
            view: buildMessageListViewWithUsers(),
          }),
        ),
      );
      await Promise.resolve();
    });

    expect(mockScrollToIndex).toHaveBeenCalledWith(2, {
      align: 'start',
      behavior: 'smooth',
    });
  });
});

function buildMessageListView(): MessageListProjection {
  const message = { id: 'message-1', kind: 'assistant' as const, content: 'hello' };

  return {
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
  };
}

function buildMessageListViewWithUsers(): MessageListProjection {
  const messages = [
    { id: 'assistant-1', kind: 'assistant' as const, content: 'previous answer' },
    { id: 'user-1', kind: 'user' as const, content: 'previous question' },
    { id: 'user-2', kind: 'user' as const, content: 'latest question' },
  ];

  return {
    hasAssistantText: false,
    hasThinkingText: false,
    latestStreamingThinkingId: null,
    loading: true,
    loadingOlderHistory: false,
    rowCount: messages.length,
    rows: messages.map((message) => ({ key: message.id, kind: 'message' as const, message })),
    shouldAutoCollapseLatestThinkingPanel: false,
    showThinkingIndicator: false,
    streamingRows: [],
    visibleCommittedMessages: messages,
  };
}
