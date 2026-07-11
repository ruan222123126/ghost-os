import type { ChatMessage } from '@/lib/types';
import { shouldReleasePostSendAnchor } from './messageListScroll';

function buildUserMessage(id: string): ChatMessage {
  return { id, kind: 'user', content: 'question' };
}

function buildAssistantMessage(id: string): ChatMessage {
  return { id, kind: 'assistant', content: 'answer' };
}

describe('components/message/messageListScroll post-send anchor release', () => {
  it('releases the anchor once the turn stops loading', () => {
    expect(shouldReleasePostSendAnchor({
      anchorIndex: 0,
      loading: false,
      messages: [buildUserMessage('local:user:trace-1')],
    })).toBe(true);
  });

  it('releases the anchor once a committed reply appears after the local user message', () => {
    expect(shouldReleasePostSendAnchor({
      anchorIndex: 0,
      loading: true,
      messages: [
        buildUserMessage('local:user:trace-1'),
        buildAssistantMessage('assistant-1'),
      ],
    })).toBe(true);
  });

  it('keeps the anchor while the local user message is still the tail during loading', () => {
    expect(shouldReleasePostSendAnchor({
      anchorIndex: 0,
      loading: true,
      messages: [buildUserMessage('local:user:trace-1')],
    })).toBe(false);
  });
});
