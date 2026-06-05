import type { ChatMessage } from '@/lib/types';
import { mergeLatestCommittedMessages } from './chatHistoryMerge';

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
  it('drops local and streaming placeholders once server history arrives', () => {
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
      'session:assistant:2',
    ]);
  });

  it('preserves older paged committed messages outside the latest window', () => {
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
      'session:assistant:1',
      'session:user:2',
      'session:assistant:2',
    ]);
  });

  it('keeps older committed tool loops but removes current streaming artifacts', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildToolMessage('session:tool:old', 'cat README.md', 'call-old'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-3', 'question'),
      buildThinkingMessage('stream-thinking:trace-3:1', 'before tool'),
      buildToolMessage('stream-tool:trace-3:call-1', 'ls -la', 'call-1'),
      buildAssistantMessage('stream-assistant:trace-3', 'answer'),
    ];

    const latest = [
      buildUserMessage('session:user:3', 'question'),
      buildToolMessage('session:tool:3', 'ls -la', 'call-1'),
      buildAssistantMessage('session:assistant:3', 'answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:tool:old',
      'session:assistant:old',
      'session:user:3',
      'session:tool:3',
      'session:assistant:3',
    ]);
  });

  it('keeps the current assistant reply visible when synced history has only caught up to the user message', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-4', 'new question'),
      buildThinkingMessage('stream-thinking:trace-4', 'thinking'),
      buildAssistantMessage('stream-assistant:trace-4', 'new answer'),
    ];

    const latest = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('session:user:4', 'new question'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:assistant:old',
      'session:user:4',
      'stream-assistant:trace-4',
    ]);
  });

  it('keeps the unfinished current turn visible when synced history has not caught up yet', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-5', 'new question'),
      buildThinkingMessage('stream-thinking:trace-5', 'thinking'),
      buildToolMessage('stream-tool:trace-5:call-5', 'ls -la', 'call-5'),
      buildAssistantMessage('stream-assistant:trace-5', 'new answer'),
    ];

    const latest = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:assistant:old',
      'local:user:trace-5',
      'stream-tool:trace-5:call-5',
      'stream-assistant:trace-5',
    ]);
  });
});
