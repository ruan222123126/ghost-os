import type { ChatMessage, PendingQuestionMessage } from '@/lib/types';
import { filterCommittedMessagesForDisplay } from './messageVisibility';

describe('lib/chat-view/messageVisibility', () => {
  it('hides only real system prompt messages when disabled', () => {
    const messages: ChatMessage[] = [
      { id: 'system-1', kind: 'system', content: 'prompt', sourceRole: 'system' },
      { id: 'internal-1', kind: 'system', content: 'internal', sourceRole: 'internal' },
      { id: 'assistant-1', kind: 'assistant', content: 'done' },
    ];

    expect(filterCommittedMessagesForDisplay(messages, [], false).map((message) => message.id)).toEqual([
      'internal-1',
      'assistant-1',
    ]);
  });

  it('still removes answered question cards that are replaced by pending inputs', () => {
    const messages: ChatMessage[] = [
      { id: 'question-1', kind: 'question', content: 'continue?', questionId: 'q-1' },
      { id: 'assistant-1', kind: 'assistant', content: 'waiting' },
    ];
    const pendingQuestions: PendingQuestionMessage[] = [
      { id: 'pending-1', kind: 'pending_question', content: 'continue?', questionId: 'q-1', sessionId: 'session-1' },
    ];

    expect(filterCommittedMessagesForDisplay(messages, pendingQuestions, true).map((message) => message.id)).toEqual([
      'assistant-1',
    ]);
  });
});
