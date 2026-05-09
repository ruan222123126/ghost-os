import type { ChatMessage, ToolChatMessage } from '@/lib/types';
import {
  buildMessageListLayoutSignature,
  isMessageListNearBottom,
  MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX,
  shouldAdjustScrollPositionOnItemSizeChange,
} from './messageListScroll';
import type { StreamingMessageRow } from '@/lib/chat-view/streamingRows';

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
  });
});
