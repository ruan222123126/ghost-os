import type { ChatMessage } from '@/lib/types';
import type { MessageListRow } from '@/lib/chat-view/types';
import { shouldPlaceAssistantCopyInline } from './messageCopyPlacement';

describe('components/message/messageCopyPlacement', () => {
  it('returns true when an assistant row is followed by tool rows in the same turn', () => {
    const rows: MessageListRow[] = [
      buildMessageRow({ id: 'assistant-1', kind: 'assistant', content: 'draft answer' }),
      buildMessageRow({ id: 'thinking-1', kind: 'thinking', content: 'reasoning' }),
      buildMessageRow({ id: 'tool-1', kind: 'tool', content: 'tool output' }),
    ];

    expect(shouldPlaceAssistantCopyInline(rows[0], 0, rows.length, (index) => rows[index])).toBe(true);
  });

  it('returns false when the next visible message is a new assistant reply', () => {
    const rows: MessageListRow[] = [
      buildMessageRow({ id: 'assistant-1', kind: 'assistant', content: 'first segment' }),
      buildMessageRow({ id: 'assistant-2', kind: 'assistant', content: 'final answer' }),
      buildMessageRow({ id: 'tool-1', kind: 'tool', content: 'late tool' }),
    ];

    expect(shouldPlaceAssistantCopyInline(rows[0], 0, rows.length, (index) => rows[index])).toBe(false);
  });
});

function buildMessageRow(message: ChatMessage): MessageListRow {
  return {
    key: message.id,
    kind: 'message',
    message,
  };
}
