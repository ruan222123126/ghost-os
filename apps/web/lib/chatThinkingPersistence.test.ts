import type { ChatMessage } from '@/lib/types';
import {
  collectPersistedThinkingMessages,
  mergeSessionMessagesWithPersistedThinking,
  persistSessionThinkingSnapshot,
} from './chatThinkingPersistence';

interface LocalStorageMock {
  clear: () => void;
  getItem: (key: string) => string | null;
  removeItem: (key: string) => void;
  setItem: (key: string, value: string) => void;
}

describe('lib/chatThinkingPersistence', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: {
        localStorage: buildLocalStorageMock(),
      },
    });
  });

  afterEach(() => {
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: undefined,
    });
  });

  it('collects thinking with assistant content anchor for streaming assistant ids', () => {
    const collected = collectPersistedThinkingMessages([
      buildUserMessage('session:user:1', 'question'),
      buildThinkingMessage('stream-thinking:1', 'analyzing...'),
      buildAssistantMessage('stream-assistant:1', 'answer'),
    ]);

    expect(collected).toEqual([{
      id: 'stream-thinking:1',
      content: 'analyzing...',
      assistantContent: 'answer',
      assistantId: undefined,
    }]);
  });

  it('persists and restores thinking for session history hydration', () => {
    const sessionId = 'session-1';
    persistSessionThinkingSnapshot(sessionId, [
      buildUserMessage('session:user:1', 'question'),
      buildThinkingMessage('stream-thinking:1', 'analyzing...'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ]);

    const merged = mergeSessionMessagesWithPersistedThinking(sessionId, [
      buildUserMessage('session:user:1', 'question'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ]);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'stream-thinking:1',
      'session:assistant:1',
    ]);
  });

  it('returns latest untouched when persisted store is invalid', () => {
    const errorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
    const localStorageMock = (window as { localStorage: LocalStorageMock }).localStorage;
    localStorageMock.setItem('ghostos:web:chat-thinking:v1', '{invalid');

    const latest = [
      buildUserMessage('session:user:1', 'question'),
      buildAssistantMessage('session:assistant:1', 'answer'),
    ];
    const merged = mergeSessionMessagesWithPersistedThinking('session-1', latest);

    expect(merged).toEqual(latest);
    errorSpy.mockRestore();
  });
});

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

function buildThinkingMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'thinking',
    content,
  };
}

function buildLocalStorageMock(): LocalStorageMock {
  const store = new Map<string, string>();
  return {
    clear: () => {
      store.clear();
    },
    getItem: (key: string) => store.get(key) ?? null,
    removeItem: (key: string) => {
      store.delete(key);
    },
    setItem: (key: string, value: string) => {
      store.set(key, value);
    },
  };
}
