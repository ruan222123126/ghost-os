import { chatMessagesEquivalent } from '@/lib/chatMessageEquivalence';
import type { ChatMessage, SessionDetail } from '@/lib/types';

interface EquivalentPair {
  previousIndex: number;
  latestIndex: number;
}

interface HistoryPageStateTarget {
  setHasOlderHistory: (sessionId: string, value: boolean) => void;
  setNextHistoryBefore: (sessionId: string, value: number | null) => void;
}

interface MessageRange {
  messages: ChatMessage[];
  start: number;
  end: number;
}

export function applyHistoryPageState(
  detail: SessionDetail,
  sessionId: string,
  target: HistoryPageStateTarget,
): void {
  target.setHasOlderHistory(sessionId, detail.page.has_more_before);
  target.setNextHistoryBefore(sessionId, detail.page.next_before ?? null);
}

export function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  if (previous.length === 0) {
    return latest;
  }

  const pairs = buildEquivalentPairs(previous, latest);
  return mergeMessagesByPairs(previous, latest, pairs);
}

export function prependUniqueCommittedMessages(previous: ChatMessage[], older: ChatMessage[]): ChatMessage[] {
  if (older.length === 0) {
    return previous;
  }

  const olderIDs = new Set(older.map((message) => message.id));
  return [...older, ...previous.filter((message) => !olderIDs.has(message.id))];
}

function buildEquivalentPairs(previous: ChatMessage[], latest: ChatMessage[]): EquivalentPair[] {
  const matrix = buildEquivalentMatrix(previous, latest);
  const pairs: EquivalentPair[] = [];
  let previousIndex = 0;
  let latestIndex = 0;

  while (previousIndex < previous.length && latestIndex < latest.length) {
    if (chatMessagesEquivalent(previous[previousIndex], latest[latestIndex])) {
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
      matrix[previousIndex][latestIndex] = chatMessagesEquivalent(previous[previousIndex], latest[latestIndex])
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
    appendPreservedPreviousMessages(merged, {
      messages: previous,
      start: previousIndex,
      end: pair.previousIndex,
    });
    appendLatestMessages(merged, {
      messages: latest,
      start: latestIndex,
      end: pair.latestIndex,
    });
    merged.push(latest[pair.latestIndex]);
    previousIndex = pair.previousIndex + 1;
    latestIndex = pair.latestIndex + 1;
  }

  appendPreservedPreviousMessages(merged, {
    messages: previous,
    start: previousIndex,
    end: previous.length,
  });
  appendLatestMessages(merged, {
    messages: latest,
    start: latestIndex,
    end: latest.length,
  });
  return merged;
}

function appendPreservedPreviousMessages(merged: ChatMessage[], range: MessageRange): void {
  for (let index = range.start; index < range.end; index += 1) {
    const message = range.messages[index];
    if (shouldPreserveUnmatchedPreviousMessage(message)) {
      merged.push(message);
    }
  }
}

function appendLatestMessages(merged: ChatMessage[], range: MessageRange): void {
  for (let index = range.start; index < range.end; index += 1) {
    merged.push(range.messages[index]);
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

function isEphemeralMessageID(id: string): boolean {
  return id.startsWith('local:') || id.startsWith('stream-');
}
