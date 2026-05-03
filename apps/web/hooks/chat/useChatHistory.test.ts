import type { ChatMessage } from '@/lib/types';
import { mergeLatestCommittedMessages, mergePersistedThinkingMessages } from './chatHistoryMerge';

function buildUserMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'user',
    content,
  };
}

function buildAssistantMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'assistant',
    content,
  };
}

function buildToolMessage(id: string, content: string, toolCallId?: string): ChatMessage {
  return {
    id,
    kind: 'tool',
    content,
    toolCallId,
  };
}

function buildThinkingMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'thinking',
    content,
  };
}

describe('hooks/chat/useChatHistory mergeLatestCommittedMessages', () => {
  it('keeps streaming thinking directly above the matched assistant response', () => {
    const previous = [
      buildUserMessage('session:user:1', 'older question'),
      buildUserMessage('local:user:trace-1', 'current question'),
      buildThinkingMessage('stream-thinking:trace-1', 'analyzing...'),
      buildAssistantMessage('stream-assistant:trace-1', 'final answer'),
    ];

    const latest = [
      buildUserMessage('session:user:1', 'older question'),
      buildUserMessage('session:user:2', 'current question'),
      buildAssistantMessage('session:assistant:2', 'final answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'session:user:2',
      'stream-thinking:trace-1',
      'session:assistant:2',
    ]);
  });

  it('preserves historical thinking above already-synced assistant and current one', () => {
    const previous = [
      buildUserMessage('session:user:1', 'question 1'),
      buildThinkingMessage('stream-thinking:trace-1', 'reasoning 1'),
      buildAssistantMessage('session:assistant:1', 'answer 1'),
      buildUserMessage('local:user:trace-2', 'question 2'),
      buildThinkingMessage('stream-thinking:trace-2', 'reasoning 2'),
      buildAssistantMessage('stream-assistant:trace-2', 'answer 2'),
    ];

    const latest = [
      buildUserMessage('session:user:1', 'question 1'),
      buildAssistantMessage('session:assistant:1', 'answer 1'),
      buildUserMessage('session:user:2', 'question 2'),
      buildAssistantMessage('session:assistant:2', 'answer 2'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'stream-thinking:trace-1',
      'session:assistant:1',
      'session:user:2',
      'stream-thinking:trace-2',
      'session:assistant:2',
    ]);
  });

  it('keeps thinking interleaved with tool calls when recent history sync completes', () => {
    const previous = [
      buildUserMessage('local:user:trace-3', 'question'),
      buildThinkingMessage('stream-thinking:trace-3:1', 'before tool'),
      buildToolMessage('stream-tool:trace-3:call-1', 'ls -la', 'call-1'),
      buildThinkingMessage('stream-thinking:trace-3:2', 'after tool'),
      buildAssistantMessage('stream-assistant:trace-3', 'answer'),
    ];

    const latest = [
      buildUserMessage('session:user:3', 'question'),
      buildToolMessage('session:tool:3', 'ls -la', 'call-1'),
      buildAssistantMessage('session:assistant:3', 'answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:3',
      'stream-thinking:trace-3:1',
      'session:tool:3',
      'stream-thinking:trace-3:2',
      'session:assistant:3',
    ]);
  });

  it('keeps unmatched old thinking in preserved history order', () => {
    const previous = [
      buildUserMessage('session:user:old', 'older question'),
      buildThinkingMessage('stream-thinking:old', 'older reasoning'),
      buildAssistantMessage('session:assistant:old', 'older answer'),
      buildUserMessage('session:user:1', 'new question'),
    ];

    const latest = [
      buildUserMessage('session:user:1', 'new question'),
      buildAssistantMessage('session:assistant:1', 'new answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'stream-thinking:old',
      'session:assistant:old',
      'session:user:1',
      'session:assistant:1',
    ]);
  });
});

describe('hooks/chat/useChatHistory mergePersistedThinkingMessages', () => {
  it('inserts persisted thinking before matched assistant by id', () => {
    const latest = [
      buildUserMessage('session:user:1', 'question'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ];

    const merged = mergePersistedThinkingMessages(latest, [{
      anchorId: 'session:assistant:1',
      anchorKind: 'assistant',
      id: 'stream-thinking:trace-1',
      content: 'analyzing...',
    }]);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'stream-thinking:trace-1',
      'session:assistant:1',
    ]);
  });

  it('inserts persisted thinking before matched assistant by content', () => {
    const latest = [
      buildUserMessage('session:user:1', 'question'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ];

    const merged = mergePersistedThinkingMessages(latest, [{
      anchorContent: 'answer',
      anchorKind: 'assistant',
      id: 'stream-thinking:trace-1',
      content: 'analyzing...',
    }]);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'stream-thinking:trace-1',
      'session:assistant:1',
    ]);
  });

  it('inserts persisted thinking before matched tool by tool call id', () => {
    const latest = [
      buildUserMessage('session:user:1', 'question'),
      buildToolMessage('session:tool:1', 'ls -la', 'call-1'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ];

    const merged = mergePersistedThinkingMessages(latest, [{
      anchorContent: 'ls -la',
      anchorKind: 'tool',
      anchorRef: 'stream-tool:1',
      anchorToolCallId: 'call-1',
      id: 'stream-thinking:trace-1',
      content: 'analyzing...',
    }]);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'stream-thinking:trace-1',
      'session:tool:1',
      'session:assistant:1',
    ]);
  });
});
