import type { SessionMessage } from '@/lib/types';
import type { LiveTaskRunCard } from '@/lib/taskRunViewerCards';

export function sessionMessagesForCard(
  card: LiveTaskRunCard,
  messages: SessionMessage[],
): SessionMessage[] {
  if (isPositiveInteger(card.round)) {
    const roundMessages = sessionMessagesForUserTurn(messages, card.round);
    if (roundMessages.length > 0) {
      return roundMessages;
    }
  }
  if (hasMultipleUserTurns(messages)) {
    return [];
  }
  return messages;
}

function sessionMessagesForUserTurn(
  messages: SessionMessage[],
  targetTurn: number,
): SessionMessage[] {
  let currentTurn = 0;
  const selected: SessionMessage[] = [];
  for (const message of messages) {
    if (message.role === 'user') {
      currentTurn += 1;
    }
    if (currentTurn === targetTurn) {
      selected.push(message);
      continue;
    }
    if (currentTurn > targetTurn) {
      break;
    }
  }
  return selected;
}

function hasMultipleUserTurns(messages: SessionMessage[]): boolean {
  let count = 0;
  for (const message of messages) {
    if (message.role !== 'user') {
      continue;
    }
    count += 1;
    if (count > 1) {
      return true;
    }
  }
  return false;
}

function isPositiveInteger(value: unknown): value is number {
  return Number.isInteger(value) && Number(value) > 0;
}
