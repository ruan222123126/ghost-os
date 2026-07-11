import { buildAssistantMessage } from '@/lib/chatMessages';
import type { ChatViewInput } from './types';
import { shouldShowHomeEmptyState } from './homeEmptyState';

describe('lib/chat-view/homeEmptyState', () => {
  it('shows the empty home state when no visible chat content exists', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: '',
      chat: buildChatState(),
      showSystemPromptMessages: true,
    })).toBe(true);
  });

  it('hides the empty home state after entering any session, even with no messages', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: 'session-1',
      chat: buildChatState(),
      showSystemPromptMessages: true,
    })).toBe(false);
  });

  it('hides the empty home state as soon as the first run starts loading', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: '',
      chat: buildChatState({ loading: true }),
      showSystemPromptMessages: true,
    })).toBe(false);
  });

  it('hides the empty home state when a visible committed message exists', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: '',
      chat: buildChatState({
        committedMessages: [buildAssistantMessage('done', 'assistant-1')],
      }),
      showSystemPromptMessages: true,
    })).toBe(false);
  });

  it('hides the empty home state when pending question input is active', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: '',
      chat: buildChatState({
        pendingQuestions: [{
          id: 'pending-1',
          kind: 'pending_question',
          content: 'Need approval',
          questionId: 'question-1',
          sessionId: 'session-1',
        }],
      }),
      showSystemPromptMessages: true,
    })).toBe(false);
  });

  it('hides the empty home state while history is loading', () => {
    expect(shouldShowHomeEmptyState({
      currentSessionId: '',
      chat: buildChatState({ historyLoading: true }),
      showSystemPromptMessages: true,
    })).toBe(false);
  });
});

function buildChatState(
  overrides: Partial<Omit<ChatViewInput, 'showSystemPromptMessages'> & {
    historyLoading: boolean;
  }> = {},
): Omit<ChatViewInput, 'showSystemPromptMessages'> & {
  historyLoading: boolean;
} {
  return {
    committedMessages: [],
    pendingQuestions: [],
    streamingAssistantSegments: [],
    streamingThinkingSegments: [],
    streamingItemOrder: [],
    streamingTools: [],
    loading: false,
    historyLoading: false,
    ...overrides,
  };
}
