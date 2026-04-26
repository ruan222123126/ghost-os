import type { PersistedThinkingMessage } from '@/hooks/chat/chatHistoryMerge';
import { mergePersistedThinkingMessages } from '@/hooks/chat/chatHistoryMerge';
import type { ChatMessage } from '@/lib/types';

const CHAT_THINKING_STORAGE_KEY = 'ghostos:web:chat-thinking:v1';

interface PersistedThinkingStore {
  sessions: Record<string, PersistedThinkingMessage[]>;
}

export function mergeSessionMessagesWithPersistedThinking(
  sessionId: string,
  latest: ChatMessage[],
): ChatMessage[] {
  return mergePersistedThinkingMessages(latest, readPersistedThinkingForSession(sessionId));
}

export function persistSessionThinkingSnapshot(sessionId: string, messages: ChatMessage[]): void {
  const normalizedSessionId = sessionId.trim();
  if (!normalizedSessionId || typeof window === 'undefined') {
    return;
  }

  const store = readPersistedThinkingStore();
  store.sessions[normalizedSessionId] = collectPersistedThinkingMessages(messages);
  writePersistedThinkingStore(store);
}

export function readPersistedThinkingForSession(sessionId: string): PersistedThinkingMessage[] {
  const normalizedSessionId = sessionId.trim();
  if (!normalizedSessionId || typeof window === 'undefined') {
    return [];
  }

  const store = readPersistedThinkingStore();
  return store.sessions[normalizedSessionId] ?? [];
}

export function collectPersistedThinkingMessages(messages: ChatMessage[]): PersistedThinkingMessage[] {
  const persisted: PersistedThinkingMessage[] = [];

  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index];
    if (message.kind !== 'thinking') {
      continue;
    }
    if (!message.content.trim()) {
      continue;
    }

    const anchorAssistant = findAnchorAssistant(messages, index + 1);
    if (!anchorAssistant) {
      continue;
    }

    const entry: PersistedThinkingMessage = {
      id: message.id,
      content: message.content,
      assistantContent: optionalTrimmed(anchorAssistant.content),
      assistantId: isLocalOrStreamingMessage(anchorAssistant.id) ? undefined : anchorAssistant.id,
    };
    if (!entry.assistantId && !entry.assistantContent) {
      continue;
    }
    persisted.push(entry);
  }

  return dedupePersistedThinkingMessages(persisted);
}

function readPersistedThinkingStore(): PersistedThinkingStore {
  const raw = readPersistedThinkingStoreRaw();
  if (!raw) {
    return createEmptyPersistedThinkingStore();
  }
  try {
    return parsePersistedThinkingStore(raw);
  } catch (error) {
    console.error('[ChatThinking] failed to parse persisted thinking store', error);
    return createEmptyPersistedThinkingStore();
  }
}

function readPersistedThinkingStoreRaw(): string | null {
  try {
    return window.localStorage.getItem(CHAT_THINKING_STORAGE_KEY);
  } catch (error) {
    console.error('[ChatThinking] failed to read persisted thinking store', error);
    return null;
  }
}

function writePersistedThinkingStore(store: PersistedThinkingStore): void {
  try {
    window.localStorage.setItem(CHAT_THINKING_STORAGE_KEY, JSON.stringify(store));
  } catch (error) {
    console.error('[ChatThinking] failed to write persisted thinking store', error);
  }
}

function parsePersistedThinkingStore(raw: string): PersistedThinkingStore {
  const parsed: unknown = JSON.parse(raw);
  if (!isRecord(parsed)) {
    return createEmptyPersistedThinkingStore();
  }

  const sessions = isRecord(parsed.sessions) ? parsed.sessions : {};
  const normalizedSessions: Record<string, PersistedThinkingMessage[]> = {};

  for (const [sessionId, messages] of Object.entries(sessions)) {
    const normalizedSessionId = sessionId.trim();
    if (!normalizedSessionId || !Array.isArray(messages)) {
      continue;
    }
    const parsedMessages = messages
      .map((message) => parsePersistedThinkingMessage(message))
      .filter((message): message is PersistedThinkingMessage => message !== null);
    normalizedSessions[normalizedSessionId] = dedupePersistedThinkingMessages(parsedMessages);
  }

  return { sessions: normalizedSessions };
}

function parsePersistedThinkingMessage(value: unknown): PersistedThinkingMessage | null {
  if (!isRecord(value)) {
    return null;
  }

  const id = optionalTrimmed(value.id);
  const content = optionalTrimmed(value.content);
  if (!id || !content) {
    return null;
  }
  return {
    id,
    content,
    assistantId: optionalTrimmed(value.assistantId),
    assistantContent: optionalTrimmed(value.assistantContent),
  };
}

function dedupePersistedThinkingMessages(messages: PersistedThinkingMessage[]): PersistedThinkingMessage[] {
  const ids = new Set<string>();
  const deduped: PersistedThinkingMessage[] = [];

  for (const message of messages) {
    if (ids.has(message.id)) {
      continue;
    }
    ids.add(message.id);
    deduped.push(message);
  }

  return deduped;
}

function findAnchorAssistant(messages: ChatMessage[], startIndex: number): ChatMessage | null {
  for (let index = startIndex; index < messages.length; index += 1) {
    const message = messages[index];
    if (message.kind === 'assistant') {
      return message;
    }
    if (message.kind === 'user' || message.kind === 'question' || message.kind === 'pending_question') {
      return null;
    }
  }
  return null;
}

function createEmptyPersistedThinkingStore(): PersistedThinkingStore {
  return { sessions: {} };
}

function isLocalOrStreamingMessage(id: string): boolean {
  return id.startsWith('stream-') || id.startsWith('local:');
}

function optionalTrimmed(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
