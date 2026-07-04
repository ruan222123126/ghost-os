import type { ChatMessage, ToolChatMessage } from '@/lib/types';
import type { StreamingMessageRow } from '@/lib/chat-view/types';
import {
  buildMessageListLayoutSignature,
  isMessageListNearBottom,
  MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX,
  resolveMessageListAutoFollow,
} from './messageListScroll';

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

describe('components/message/messageListScroll', () => {
  it('treats reverse-flow scroll positions near zero as bottom-following', () => {
    expect(isMessageListNearBottom({ scrollTop: 0 })).toBe(true);
    expect(isMessageListNearBottom({ scrollTop: -MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX })).toBe(true);
    expect(isMessageListNearBottom({ scrollTop: MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX })).toBe(true);
  });

  it('treats large reverse-flow offsets as away from the bottom', () => {
    expect(isMessageListNearBottom({
      scrollTop: -(MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX + 1),
    })).toBe(false);
  });

  it('resolves auto-follow from the reverse-flow bottom threshold', () => {
    expect(resolveMessageListAutoFollow({ scrollTop: 0 })).toBe(true);
    expect(resolveMessageListAutoFollow({ scrollTop: -240 })).toBe(false);
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

  it('changes the layout signature when history loading or pending rows change', () => {
    const base = buildMessageListLayoutSignature({
      committedMessages: [{ id: 'assistant-1', kind: 'assistant', content: 'done' }],
      latestStreamingThinkingId: null,
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: false,
      showThinkingIndicator: false,
      streamingRows: [],
    });
    const historyLoading = buildMessageListLayoutSignature({
      committedMessages: [{ id: 'assistant-1', kind: 'assistant', content: 'done' }],
      latestStreamingThinkingId: null,
      latestStreamingThinkingPanelOpen: false,
      loadingOlderHistory: true,
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

    expect(historyLoading).not.toBe(base);
    expect(questionChanged).not.toBe(base);
  });
});
