import type { ChatMessage } from '@/lib/types';

export function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  if (previous.length === 0) {
    return latest;
  }

  const latestIDs = new Set(latest.map((message) => message.id));
  const preserved = previous.filter((message) => shouldPreserveCommittedMessage(message, latestIDs));
  return [...preserved, ...latest];
}

function shouldPreserveCommittedMessage(message: ChatMessage, latestIDs: Set<string>): boolean {
  if (latestIDs.has(message.id)) {
    return false;
  }
  return !isEphemeralMessageID(message.id);
}

function isEphemeralMessageID(id: string): boolean {
  return id.startsWith('local:') || id.startsWith('stream-');
}
