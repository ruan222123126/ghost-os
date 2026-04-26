import type { ChatMessage } from '@/lib/types';

interface ThinkingInsertion {
  assistantContent?: string;
  assistantId?: string;
  thinking: ChatMessage;
}

export interface PersistedThinkingMessage {
  id: string;
  content: string;
  assistantId?: string;
  assistantContent?: string;
}

export function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  const latestIDs = new Set(latest.map((message) => message.id));
  const preserved = previous.filter((message) => shouldPreserveMessage(message, latestIDs));
  const insertions = collectThinkingInsertions(previous, latestIDs);
  const inserted = insertThinkingBeforeAssistant(latest, insertions);
  const retainedPreserved = preserved.filter((message) => !inserted.thinkingIDs.has(message.id));
  return [...retainedPreserved, ...inserted.messages];
}

export function mergePersistedThinkingMessages(
  latest: ChatMessage[],
  persisted: PersistedThinkingMessage[],
): ChatMessage[] {
  if (persisted.length === 0) {
    return latest;
  }

  const latestIDs = new Set(latest.map((message) => message.id));
  const insertions = persisted
    .filter((message) => message.content.trim() && !latestIDs.has(message.id))
    .map((message): ThinkingInsertion => ({
      assistantContent: normalizeOptionalText(message.assistantContent),
      assistantId: normalizeOptionalText(message.assistantId),
      thinking: {
        id: message.id,
        kind: 'thinking',
        content: message.content,
      },
    }));
  return insertThinkingBeforeAssistant(latest, insertions).messages;
}

function shouldPreserveMessage(message: ChatMessage, latestIDs: Set<string>): boolean {
  if (latestIDs.has(message.id)) {
    return false;
  }
  if (message.kind !== 'thinking' && isLocalOrStreamingMessage(message.id)) {
    return false;
  }
  return true;
}

function isLocalOrStreamingMessage(id: string): boolean {
  return id.startsWith('stream-') || id.startsWith('local:');
}

function normalizeOptionalText(value?: string): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

function collectThinkingInsertions(previous: ChatMessage[], latestIDs: Set<string>): ThinkingInsertion[] {
  const insertions: ThinkingInsertion[] = [];

  for (let index = 0; index < previous.length; index += 1) {
    const message = previous[index];
    if (message.kind !== 'thinking' || latestIDs.has(message.id)) {
      continue;
    }
    const anchorAssistant = findAnchorAssistant(previous, index + 1);
    if (!anchorAssistant) {
      continue;
    }
    if (latestIDs.has(anchorAssistant.id)) {
      insertions.push({ assistantId: anchorAssistant.id, thinking: message });
      continue;
    }
    if (!isLocalOrStreamingMessage(anchorAssistant.id)) {
      continue;
    }
    const assistantContent = anchorAssistant.content.trim();
    if (!assistantContent) {
      continue;
    }
    insertions.push({ assistantContent, thinking: message });
  }

  return insertions;
}

function findAnchorAssistant(messages: ChatMessage[], startIndex: number): ChatMessage | null {
  for (let index = startIndex; index < messages.length; index += 1) {
    const message = messages[index];
    if (message.kind === 'assistant') {
      return message;
    }
    if (message.kind === 'user' || message.kind === 'pending_question' || message.kind === 'question') {
      return null;
    }
  }
  return null;
}

function insertThinkingBeforeAssistant(
  latest: ChatMessage[],
  insertions: ThinkingInsertion[],
): { messages: ChatMessage[]; thinkingIDs: Set<string> } {
  if (insertions.length === 0) {
    return { messages: latest, thinkingIDs: new Set() };
  }

  const latestIndexByID = new Map(latest.map((message, index) => [message.id, index]));
  const assistantIndexesByContent = buildAssistantIndexesByContent(latest);
  const usedAssistantIndexes = new Set<number>();
  const beforeAssistant = new Map<number, ChatMessage[]>();
  const insertedThinkingIDs = new Set<string>();

  for (const insertion of insertions) {
    const assistantIndex = resolveAssistantIndex({
      assistantIndexesByContent,
      insertion,
      latestIndexByID,
      usedAssistantIndexes,
    });
    if (assistantIndex === null) {
      continue;
    }
    const before = beforeAssistant.get(assistantIndex) ?? [];
    before.push(insertion.thinking);
    beforeAssistant.set(assistantIndex, before);
    insertedThinkingIDs.add(insertion.thinking.id);
  }

  return {
    messages: composeMessagesWithThinking(latest, beforeAssistant),
    thinkingIDs: insertedThinkingIDs,
  };
}

function buildAssistantIndexesByContent(messages: ChatMessage[]): Map<string, number[]> {
  const indexesByContent = new Map<string, number[]>();

  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index];
    if (message.kind !== 'assistant') {
      continue;
    }
    const content = message.content.trim();
    if (!content) {
      continue;
    }
    const indexes = indexesByContent.get(content) ?? [];
    indexes.push(index);
    indexesByContent.set(content, indexes);
  }

  return indexesByContent;
}

function resolveAssistantIndex(
  options: {
    assistantIndexesByContent: Map<string, number[]>;
    insertion: ThinkingInsertion;
    latestIndexByID: Map<string, number>;
    usedAssistantIndexes: Set<number>;
  },
): number | null {
  if (options.insertion.assistantId) {
    const index = options.latestIndexByID.get(options.insertion.assistantId);
    if (index !== undefined) {
      options.usedAssistantIndexes.add(index);
      return index;
    }
  }
  if (!options.insertion.assistantContent) {
    return null;
  }
  const candidates = options.assistantIndexesByContent.get(options.insertion.assistantContent);
  if (!candidates || candidates.length === 0) {
    return null;
  }
  for (const candidate of candidates) {
    if (options.usedAssistantIndexes.has(candidate)) {
      continue;
    }
    options.usedAssistantIndexes.add(candidate);
    return candidate;
  }
  return null;
}

function composeMessagesWithThinking(
  latest: ChatMessage[],
  beforeAssistant: Map<number, ChatMessage[]>,
): ChatMessage[] {
  const messages: ChatMessage[] = [];

  for (let index = 0; index < latest.length; index += 1) {
    const before = beforeAssistant.get(index);
    if (before) {
      messages.push(...before);
    }
    messages.push(latest[index]);
  }

  return messages;
}
