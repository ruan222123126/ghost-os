import { findNextThinkingAnchor, type ThinkingAnchor, type ThinkingAnchorKind } from '@/lib/chatThinkingAnchors';
import type { ChatMessage } from '@/lib/types';

interface ThinkingInsertion {
  anchorContent?: string;
  anchorId?: string;
  anchorKind?: ThinkingAnchorKind;
  anchorRef?: string;
  anchorToolCallId?: string;
  thinking: ChatMessage;
}

export interface PersistedThinkingMessage {
  id: string;
  content: string;
  anchorId?: string;
  anchorKind?: ThinkingAnchorKind;
  anchorRef?: string;
  anchorContent?: string;
  anchorToolCallId?: string;
}

export function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  const latestIDs = new Set(latest.map((message) => message.id));
  const preserved = previous.filter((message) => shouldPreserveMessage(message, latestIDs));
  const insertions = collectThinkingInsertions(previous, latestIDs);
  const inserted = insertThinkingBeforeAnchor(latest, insertions);
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
      anchorContent: normalizeOptionalText(message.anchorContent),
      anchorId: normalizeOptionalText(message.anchorId),
      anchorKind: message.anchorKind,
      anchorRef: normalizeOptionalText(message.anchorRef),
      anchorToolCallId: normalizeOptionalText(message.anchorToolCallId),
      thinking: {
        id: message.id,
        kind: 'thinking',
        content: message.content,
      },
    }))
    .filter(hasThinkingAnchor);
  return insertThinkingBeforeAnchor(latest, insertions).messages;
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
    const anchor = findNextThinkingAnchor(previous, index + 1);
    if (!anchor) {
      continue;
    }
    const insertion = buildThinkingInsertion(message, anchor);
    if (insertion) {
      insertions.push(insertion);
    }
  }
  return insertions;
}

function buildThinkingInsertion(thinking: ChatMessage, anchor: ThinkingAnchor): ThinkingInsertion | null {
  const insertion: ThinkingInsertion = {
    anchorContent: normalizeOptionalText(anchor.content),
    anchorId: normalizeOptionalText(anchor.id),
    anchorKind: anchor.kind,
    anchorRef: normalizeOptionalText(anchor.ref),
    anchorToolCallId: normalizeOptionalText(anchor.toolCallId),
    thinking,
  };
  return hasThinkingAnchor(insertion) ? insertion : null;
}

function hasThinkingAnchor(insertion: ThinkingInsertion): boolean {
  return Boolean(
    insertion.anchorId
      || insertion.anchorToolCallId
      || insertion.anchorContent,
  );
}
function insertThinkingBeforeAnchor(
  latest: ChatMessage[],
  insertions: ThinkingInsertion[],
): { messages: ChatMessage[]; thinkingIDs: Set<string> } {
  if (insertions.length === 0) {
    return { messages: latest, thinkingIDs: new Set() };
  }
  const lookups = buildAnchorLookups(latest);
  const resolvedIndexesByAnchorRef = new Map<string, number>();
  const usedAnchorIndexes = new Set<number>();
  const beforeAnchor = new Map<number, ChatMessage[]>();
  const insertedThinkingIDs = new Set<string>();

  for (const insertion of insertions) {
    const anchorIndex = resolveAnchorIndex({
      insertion,
      lookups,
      resolvedIndexesByAnchorRef,
      usedAnchorIndexes,
    });
    if (anchorIndex === null) {
      continue;
    }
    const before = beforeAnchor.get(anchorIndex) ?? [];
    before.push(insertion.thinking);
    beforeAnchor.set(anchorIndex, before);
    insertedThinkingIDs.add(insertion.thinking.id);
  }
  return {
    messages: composeMessagesWithThinking(latest, beforeAnchor),
    thinkingIDs: insertedThinkingIDs,
  };
}
interface AnchorLookups {
  indexesByContent: Map<string, number[]>;
  indexesByID: Map<string, number>;
  indexesByToolCallId: Map<string, number[]>;
}

function buildAnchorLookups(messages: ChatMessage[]): AnchorLookups {
  const indexesByContent = new Map<string, number[]>();
  const indexesByToolCallId = new Map<string, number[]>();
  const indexesByID = new Map(messages.map((message, index) => [message.id, index]));
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index];
    const content = normalizeOptionalText(message.content);
    if (content) {
      const key = buildAnchorContentKey(message.kind, content);
      const indexes = indexesByContent.get(key) ?? [];
      indexes.push(index);
      indexesByContent.set(key, indexes);
    }
    if (message.kind !== 'tool') {
      continue;
    }
    const toolCallId = normalizeOptionalText(message.toolCallId);
    if (!toolCallId) {
      continue;
    }
    const indexes = indexesByToolCallId.get(toolCallId) ?? [];
    indexes.push(index);
    indexesByToolCallId.set(toolCallId, indexes);
  }
  return {
    indexesByContent,
    indexesByID,
    indexesByToolCallId,
  };
}
function resolveAnchorIndex(
  options: {
    insertion: ThinkingInsertion;
    lookups: AnchorLookups;
    resolvedIndexesByAnchorRef: Map<string, number>;
    usedAnchorIndexes: Set<number>;
  },
): number | null {
  const anchorRef = buildInsertionAnchorRef(options.insertion);
  if (anchorRef) {
    const cached = options.resolvedIndexesByAnchorRef.get(anchorRef);
    if (cached !== undefined) {
      return cached;
    }
  }
  const resolved = resolveNewAnchorIndex(options.insertion, options.lookups, options.usedAnchorIndexes);
  if (resolved === null) {
    return null;
  }
  if (anchorRef) {
    options.resolvedIndexesByAnchorRef.set(anchorRef, resolved);
  }
  return resolved;
}
function buildInsertionAnchorRef(insertion: ThinkingInsertion): string | undefined {
  return insertion.anchorRef
    || insertion.anchorId
    || (insertion.anchorToolCallId ? `tool:${insertion.anchorToolCallId}` : undefined)
    || (insertion.anchorKind && insertion.anchorContent
      ? buildAnchorContentKey(insertion.anchorKind, insertion.anchorContent)
      : undefined);
}
function resolveNewAnchorIndex(
  insertion: ThinkingInsertion,
  lookups: AnchorLookups,
  usedAnchorIndexes: Set<number>,
): number | null {
  const idMatch = takeAnchorIndex(insertion.anchorId, lookups.indexesByID, usedAnchorIndexes);
  if (idMatch !== null) {
    return idMatch;
  }
  const toolMatch = takeAnchorCandidate(
    insertion.anchorToolCallId,
    lookups.indexesByToolCallId,
    usedAnchorIndexes,
  );
  if (toolMatch !== null) {
    return toolMatch;
  }
  if (!insertion.anchorKind || !insertion.anchorContent) {
    return null;
  }
  return takeAnchorCandidate(
    buildAnchorContentKey(insertion.anchorKind, insertion.anchorContent),
    lookups.indexesByContent,
    usedAnchorIndexes,
  );
}
function takeAnchorIndex(
  key: string | undefined,
  indexesByID: Map<string, number>,
  usedAnchorIndexes: Set<number>,
): number | null {
  if (!key) {
    return null;
  }
  const index = indexesByID.get(key);
  if (index === undefined || usedAnchorIndexes.has(index)) {
    return null;
  }
  usedAnchorIndexes.add(index);
  return index;
}
function takeAnchorCandidate(
  key: string | undefined,
  indexesByKey: Map<string, number[]>,
  usedAnchorIndexes: Set<number>,
): number | null {
  if (!key) {
    return null;
  }
  const candidates = indexesByKey.get(key);
  if (!candidates || candidates.length === 0) {
    return null;
  }
  for (const candidate of candidates) {
    if (usedAnchorIndexes.has(candidate)) {
      continue;
    }
    usedAnchorIndexes.add(candidate);
    return candidate;
  }
  return null;
}
function buildAnchorContentKey(kind: string, content: string): string {
  return `${kind}:${content}`;
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
