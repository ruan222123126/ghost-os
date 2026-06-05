import type { ChatMessage, ToolChatMessage, UserChatMessage } from '@/lib/types';

interface EquivalentPair {
  previousIndex: number;
  latestIndex: number;
}

export function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  if (previous.length === 0) {
    return latest;
  }

  const pairs = buildEquivalentPairs(previous, latest);
  return mergeMessagesByPairs(previous, latest, pairs);
}

function buildEquivalentPairs(previous: ChatMessage[], latest: ChatMessage[]): EquivalentPair[] {
  const matrix = buildEquivalentMatrix(previous, latest);
  const pairs: EquivalentPair[] = [];
  let previousIndex = 0;
  let latestIndex = 0;

  while (previousIndex < previous.length && latestIndex < latest.length) {
    if (messagesEquivalent(previous[previousIndex], latest[latestIndex])) {
      pairs.push({ previousIndex, latestIndex });
      previousIndex += 1;
      latestIndex += 1;
      continue;
    }

    if (matrix[previousIndex + 1][latestIndex] >= matrix[previousIndex][latestIndex + 1]) {
      previousIndex += 1;
      continue;
    }
    latestIndex += 1;
  }

  return pairs;
}

function buildEquivalentMatrix(previous: ChatMessage[], latest: ChatMessage[]): number[][] {
  const matrix = Array.from(
    { length: previous.length + 1 },
    () => Array<number>(latest.length + 1).fill(0),
  );

  for (let previousIndex = previous.length - 1; previousIndex >= 0; previousIndex -= 1) {
    for (let latestIndex = latest.length - 1; latestIndex >= 0; latestIndex -= 1) {
      matrix[previousIndex][latestIndex] = messagesEquivalent(previous[previousIndex], latest[latestIndex])
        ? matrix[previousIndex + 1][latestIndex + 1] + 1
        : Math.max(matrix[previousIndex + 1][latestIndex], matrix[previousIndex][latestIndex + 1]);
    }
  }

  return matrix;
}

function mergeMessagesByPairs(
  previous: ChatMessage[],
  latest: ChatMessage[],
  pairs: EquivalentPair[],
): ChatMessage[] {
  const merged: ChatMessage[] = [];
  let previousIndex = 0;
  let latestIndex = 0;

  for (const pair of pairs) {
    appendPreservedPreviousMessages(merged, previous, previousIndex, pair.previousIndex);
    appendLatestMessages(merged, latest, latestIndex, pair.latestIndex);
    merged.push(latest[pair.latestIndex]);
    previousIndex = pair.previousIndex + 1;
    latestIndex = pair.latestIndex + 1;
  }

  appendPreservedPreviousMessages(merged, previous, previousIndex, previous.length);
  appendLatestMessages(merged, latest, latestIndex, latest.length);
  return merged;
}

function appendPreservedPreviousMessages(
  merged: ChatMessage[],
  previous: ChatMessage[],
  start: number,
  end: number,
): void {
  for (let index = start; index < end; index += 1) {
    const message = previous[index];
    if (shouldPreserveUnmatchedPreviousMessage(message)) {
      merged.push(message);
    }
  }
}

function appendLatestMessages(
  merged: ChatMessage[],
  latest: ChatMessage[],
  start: number,
  end: number,
): void {
  for (let index = start; index < end; index += 1) {
    merged.push(latest[index]);
  }
}

function shouldPreserveUnmatchedPreviousMessage(message: ChatMessage): boolean {
  if (!isEphemeralMessageID(message.id)) {
    return true;
  }

  return message.kind === 'user'
    || message.kind === 'assistant'
    || message.kind === 'tool';
}

function messagesEquivalent(previous: ChatMessage, latest: ChatMessage): boolean {
  if (previous.id === latest.id) {
    return true;
  }
  if (previous.kind !== latest.kind) {
    return false;
  }

  switch (previous.kind) {
    case 'user':
      return userMessagesEquivalent(previous, latest as UserChatMessage);
    case 'assistant':
    case 'thinking':
    case 'error':
    case 'event':
    case 'system':
      return normalizeText(previous.content) === normalizeText(latest.content);
    case 'tool':
      return toolMessagesEquivalent(previous, latest as ToolChatMessage);
    default:
      return false;
  }
}

function userMessagesEquivalent(previous: UserChatMessage, latest: UserChatMessage): boolean {
  return normalizeText(previous.content) === normalizeText(latest.content)
    && countImages(previous) === countImages(latest);
}

function toolMessagesEquivalent(previous: ToolChatMessage, latest: ToolChatMessage): boolean {
  const previousToolCallId = previous.toolCallId?.trim();
  const latestToolCallId = latest.toolCallId?.trim();
  if (previousToolCallId && latestToolCallId) {
    return previousToolCallId === latestToolCallId;
  }

  return normalizeText(previous.toolName) === normalizeText(latest.toolName)
    && normalizeText(previous.toolInput) === normalizeText(latest.toolInput)
    && normalizeText(previous.content) === normalizeText(latest.content);
}

function countImages(message: UserChatMessage): number {
  return message.images?.length ?? 0;
}

function normalizeText(value?: string): string {
  return value?.trim() ?? '';
}

function isEphemeralMessageID(id: string): boolean {
  return id.startsWith('local:') || id.startsWith('stream-');
}
