import type { ChatMessage } from './types';

export type ThinkingAnchorKind = Exclude<ChatMessage['kind'], 'thinking' | 'error'>;

export interface ThinkingAnchor {
  ref: string;
  id?: string;
  kind: ThinkingAnchorKind;
  content?: string;
  toolCallId?: string;
}

export function findNextThinkingAnchor(messages: ChatMessage[], startIndex: number): ThinkingAnchor | null {
  for (let index = startIndex; index < messages.length; index += 1) {
    const message = messages[index];
    if (message.kind === 'thinking') {
      continue;
    }
    if (message.kind === 'user') {
      return null;
    }
    return buildThinkingAnchor(message);
  }
  return null;
}

export function buildThinkingAnchor(message: ChatMessage): ThinkingAnchor | null {
  if (!isThinkingAnchorKind(message.kind)) {
    return null;
  }

  const content = normalizeOptionalText(message.content);
  const toolCallId = message.kind === 'tool'
    ? normalizeOptionalText(message.toolCallId)
    : undefined;
  const id = isLocalOrStreamingMessage(message.id) ? undefined : message.id;
  if (!id && !toolCallId && !content) {
    return null;
  }

  return {
    ref: message.id,
    id,
    kind: message.kind,
    content,
    toolCallId,
  };
}

function isThinkingAnchorKind(kind: ChatMessage['kind']): kind is ThinkingAnchorKind {
  return kind !== 'thinking' && kind !== 'error';
}

function isLocalOrStreamingMessage(id: string): boolean {
  return id.startsWith('stream-') || id.startsWith('local:');
}

function normalizeOptionalText(value?: string): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}
