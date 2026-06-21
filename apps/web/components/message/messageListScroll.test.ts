import type { ChatMessage, ToolChatMessage } from '@/lib/types';
import {
  buildVisibleMessageTailSnapshot,
  buildMessageListLayoutSignature,
  computePostSendAnchorLayout,
  getPostSendLockedScrollTop,
  getPostSendAnchorIndexFromVisibleMessages,
  hasVisibleContentAfterIndex,
  hasPostSendRealContentOverflow,
  isMessageListNearBottom,
  isPostSendCommittedUserMessageId,
  MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX,
  resolveMessageListAutoFollow,
  resolvePostSendOverflowDecision,
  shouldAdjustScrollPositionOnItemSizeChange,
  shouldClearPostSendProgrammaticScrollTarget,
} from './messageListScroll';
import type { StreamingMessageRow } from '@/lib/chat-view/types';

function buildStreamingRow(message: ChatMessage): StreamingMessageRow {
  return {
    key: message.id,
    message,
  };
}

function buildToolMessage(overrides?: Partial<ToolChatMessage>): ToolChatMessage {
  return {
    id: 'tool-1',
    kind: 'tool',
    content: 'tool output',
    toolName: 'bash_exec',
    toolStatus: 'running',
    ...overrides,
  };
}

function resolveOverflow(overrides?: {
  autoFollow?: boolean;
  hasVisibleContent?: boolean;
  realContentHeightPx?: number;
}) {
  return resolvePostSendOverflowDecision({
    anchorStartPx: 300,
    autoFollow: overrides?.autoFollow ?? true,
    baselineContentHeightPx: 520,
    containerHeightPx: 200,
    hasVisibleContent: overrides?.hasVisibleContent ?? true,
    realContentHeightPx: overrides?.realContentHeightPx ?? 560,
  });
}

describe('components/message/messageListScroll', () => {
  it('treats users outside the bottom threshold as not auto-following', () => {
    expect(isMessageListNearBottom({
      scrollHeight: 1000,
      clientHeight: 400,
      scrollTop: 1000 - 400 - MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX - 1,
    })).toBe(false);
  });

  it('resumes auto-follow when the user returns within the bottom threshold', () => {
    expect(isMessageListNearBottom({
      scrollHeight: 1000,
      clientHeight: 400,
      scrollTop: 1000 - 400 - MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX,
    })).toBe(true);
  });

  it('keeps auto-follow while the post-send anchor lock is still controlling scroll', () => {
    expect(resolveMessageListAutoFollow(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 540,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
      },
    )).toBe(true);
  });

  it('keeps auto-follow while a post-send scroll is still moving toward the anchor', () => {
    expect(resolveMessageListAutoFollow(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 240,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
        programmaticScrollTargetPx: 540,
      },
    )).toBe(true);
  });

  it('clears the post-send target once the scroll reaches the anchor', () => {
    expect(shouldClearPostSendProgrammaticScrollTarget(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 540,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
        programmaticScrollTargetPx: 540,
      },
    )).toBe(true);
  });

  it('keeps the post-send lock when the user scrolls down into temporary spacer', () => {
    expect(resolveMessageListAutoFollow(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 560,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
      },
    )).toBe(true);
  });

  it('releases auto-follow once the user drags away from the post-send anchor lock', () => {
    expect(resolveMessageListAutoFollow(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 500,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
      },
    )).toBe(false);
  });

  it('stops clamping once the post-send lock is cleared', () => {
    expect(getPostSendLockedScrollTop(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 560,
      },
      {
        mode: 'idle',
        controlledScrollTopPx: null,
      },
    )).toBeNull();
  });

  it('changes the layout signature when streaming text or tool state changes', () => {
    const base = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: true,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [
        buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking' }),
        buildStreamingRow({ id: 'assistant-1', kind: 'assistant', content: 'answer' }),
        buildStreamingRow(buildToolMessage()),
      ],
    });

    const thinkingChanged = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: true,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [
        buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking more' }),
        buildStreamingRow({ id: 'assistant-1', kind: 'assistant', content: 'answer' }),
        buildStreamingRow(buildToolMessage()),
      ],
    });
    const assistantChanged = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: true,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [
        buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking' }),
        buildStreamingRow({ id: 'assistant-1', kind: 'assistant', content: 'answer extended' }),
        buildStreamingRow(buildToolMessage()),
      ],
    });
    const toolChanged = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: true,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [
        buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking' }),
        buildStreamingRow({ id: 'assistant-1', kind: 'assistant', content: 'answer' }),
        buildStreamingRow(buildToolMessage({ content: 'tool output expanded', toolStatus: 'success' })),
      ],
    });

    expect(thinkingChanged).not.toBe(base);
    expect(assistantChanged).not.toBe(base);
    expect(toolChanged).not.toBe(base);
  });

  it('changes the layout signature when the streaming thinking panel auto-collapses', () => {
    const expanded = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: true,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking' })],
    });
    const collapsed = buildMessageListLayoutSignature({
      committedMessages: [],
      latestStreamingThinkingId: 'thinking-1',
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [buildStreamingRow({ id: 'thinking-1', kind: 'thinking', content: 'thinking' })],
    });

    expect(collapsed).not.toBe(expanded);
  });

  it('changes the layout signature when pending questions or committed history change', () => {
    const base = buildMessageListLayoutSignature({
      committedMessages: [{ id: 'assistant-1', kind: 'assistant', content: 'done' }],
      latestStreamingThinkingId: null,
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [],
    });
    const questionChanged = buildMessageListLayoutSignature({
      committedMessages: [{ id: 'assistant-1', kind: 'assistant', content: 'done' }],
      latestStreamingThinkingId: null,
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [
        buildStreamingRow({
          id: 'pending-1',
          kind: 'pending_question',
          content: 'continue?',
          questionId: 'q-1',
          sessionId: 'session-1',
        }),
      ],
    });
    const historyChanged = buildMessageListLayoutSignature({
      committedMessages: [
        { id: 'assistant-1', kind: 'assistant', content: 'done' },
        { id: 'assistant-2', kind: 'assistant', content: 'done again' },
      ],
      latestStreamingThinkingId: null,
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [],
    });

    expect(questionChanged).not.toBe(base);
    expect(historyChanged).not.toBe(base);
  });

  it('disables virtualizer scroll adjustment when auto-follow is off', () => {
    expect(shouldAdjustScrollPositionOnItemSizeChange(false)).toBe(false);
    expect(shouldAdjustScrollPositionOnItemSizeChange(true)).toBe(true);
    expect(shouldAdjustScrollPositionOnItemSizeChange(true, 'anchoring')).toBe(false);
    expect(shouldAdjustScrollPositionOnItemSizeChange(true, 'waiting_overflow')).toBe(false);
  });

  it('detects only appended local user messages as post-send anchors', () => {
    const previousMessages: ChatMessage[] = [
      { id: 'assistant-1', kind: 'assistant', content: 'previous' },
    ];
    const previousTail = buildVisibleMessageTailSnapshot(previousMessages);

    expect(getPostSendAnchorIndexFromVisibleMessages({
      messages: [
        ...previousMessages,
        { id: 'local:user:trace-1', kind: 'user', content: 'next' },
      ],
      previousTail,
    })).toBe(1);
    expect(getPostSendAnchorIndexFromVisibleMessages({
      messages: [
        ...previousMessages,
        { id: 'local:answer:trace-1:q-1', kind: 'user', content: 'answer' },
      ],
      previousTail,
    })).toBeNull();
    expect(getPostSendAnchorIndexFromVisibleMessages({
      messages: previousMessages,
      previousTail,
    })).toBeNull();
  });

  it('does not trigger post-send focus on initial visible history', () => {
    expect(getPostSendAnchorIndexFromVisibleMessages({
      messages: [{ id: 'local:user:trace-1', kind: 'user', content: 'already there' }],
      previousTail: null,
    })).toBeNull();
  });

  it('keeps enough temporary spacer to align the sent user message at the top', () => {
    expect(computePostSendAnchorLayout({
      anchorStartPx: 540,
      containerHeightPx: 400,
      realContentHeightPx: 700,
    })).toEqual({
      anchorScrollTopPx: 540,
      trailingSpacerPx: 240,
      viewportBottomPx: 940,
    });
  });

  it('does not add post-send spacer when the real content already covers the anchor viewport', () => {
    expect(computePostSendAnchorLayout({
      anchorStartPx: 240,
      containerHeightPx: 400,
      realContentHeightPx: 900,
    }).trailingSpacerPx).toBe(0);
  });

  it('waits for visible assistant text before treating real content growth as overflow', () => {
    expect(resolveOverflow({ hasVisibleContent: false })).toEqual({
      overflowed: false,
      shouldScrollToBottom: false,
      trailingSpacerPx: 0,
    });
    expect(resolveOverflow()).toEqual({
      overflowed: true,
      shouldScrollToBottom: true,
      trailingSpacerPx: 0,
    });
  });

  it('reports overflow without forcing bottom follow after manual upward scroll', () => {
    expect(resolveOverflow({ autoFollow: false })).toEqual({
      overflowed: true,
      shouldScrollToBottom: false,
      trailingSpacerPx: 0,
    });
  });

  it('can disable bottom follow entirely even when the content overflows', () => {
    expect(resolveOverflow({ autoFollow: false })).toMatchObject({
      overflowed: true,
      shouldScrollToBottom: false,
    });
  });

  it('keeps spacer when the failed turn has no visible assistant text', () => {
    expect(resolveOverflow({
      hasVisibleContent: false,
      realContentHeightPx: 520,
    }).trailingSpacerPx).toBe(0);
  });

  it('finds visible content after the post-send anchor', () => {
    const messages: ChatMessage[] = [
      { id: 'local:user:trace-1', kind: 'user', content: 'question' },
      { id: 'thinking-1', kind: 'thinking', content: 'thinking' },
      { id: 'assistant-1', kind: 'assistant', content: 'answer' },
    ];

    expect(hasVisibleContentAfterIndex(messages, 0)).toBe(true);
    expect(hasVisibleContentAfterIndex(messages, 2)).toBe(false);
  });

  it('treats streamed tool cards as visible post-send content', () => {
    const messages: ChatMessage[] = [
      { id: 'local:user:trace-1', kind: 'user', content: 'question' },
      {
        id: 'tool-1',
        kind: 'tool',
        content: '',
        toolName: 'bash_exec',
        toolInput: 'ls',
      },
    ];

    expect(hasVisibleContentAfterIndex(messages, 0)).toBe(true);
  });

  it('shrinks the temporary spacer as real content fills the anchored viewport', () => {
    expect(resolvePostSendOverflowDecision({
      anchorStartPx: 300,
      autoFollow: true,
      baselineContentHeightPx: 500,
      containerHeightPx: 260,
      hasVisibleContent: true,
      realContentHeightPx: 540,
    })).toEqual({
      overflowed: false,
      shouldScrollToBottom: false,
      trailingSpacerPx: 20,
    });

    expect(resolvePostSendOverflowDecision({
      anchorStartPx: 300,
      autoFollow: true,
      baselineContentHeightPx: 520,
      containerHeightPx: 240,
      hasVisibleContent: true,
      realContentHeightPx: 541,
    })).toEqual({
      overflowed: true,
      shouldScrollToBottom: true,
      trailingSpacerPx: 0,
    });
  });

  it('only clamps downward scroll into the temporary post-send spacer', () => {
    expect(getPostSendLockedScrollTop(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 560,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
      },
    )).toBe(540);

    expect(getPostSendLockedScrollTop(
      {
        scrollHeight: 1200,
        clientHeight: 400,
        scrollTop: 500,
      },
      {
        mode: 'waiting_overflow',
        controlledScrollTopPx: 540,
      },
    )).toBeNull();
  });

  it('uses anchor-relative real content overflow instead of total page overflow', () => {
    expect(hasPostSendRealContentOverflow({
      anchorStartPx: 700,
      containerHeightPx: 400,
      realContentHeightPx: 1000,
    })).toBe(false);
    expect(hasPostSendRealContentOverflow({
      anchorStartPx: 700,
      containerHeightPx: 400,
      realContentHeightPx: 1101,
    })).toBe(true);
  });

  it('classifies only local user ids as normal send ids', () => {
    expect(isPostSendCommittedUserMessageId('local:user:trace-1')).toBe(true);
    expect(isPostSendCommittedUserMessageId('local:answer:trace-1:q-1')).toBe(false);
  });
});
