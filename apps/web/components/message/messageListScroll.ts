import type { ChatMessage } from '@/lib/types';
import type { StreamingMessageRow } from './streamingRows';

export const MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX = 120;

interface ScrollMetrics {
  scrollHeight: number;
  clientHeight: number;
  scrollTop: number;
}

interface MessageListLayoutSignatureOptions {
  committedMessages: ChatMessage[];
  streamingRows: StreamingMessageRow[];
  showThinkingIndicator: boolean;
  loadingOlderHistory: boolean;
  latestStreamingThinkingId: string | null;
  latestStreamingThinkingPanelOpen: boolean;
}

export function isMessageListNearBottom(metrics: ScrollMetrics): boolean {
  const distance = metrics.scrollHeight - metrics.clientHeight - metrics.scrollTop;
  return distance <= MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX;
}

export function shouldAdjustScrollPositionOnItemSizeChange(autoFollow: boolean): boolean {
  return autoFollow;
}

export function buildMessageListLayoutSignature(
  options: MessageListLayoutSignatureOptions,
): string {
  return [
    `history:${options.loadingOlderHistory ? 1 : 0}`,
    `indicator:${options.showThinkingIndicator ? 1 : 0}`,
    `committed:${options.committedMessages.map(buildMessageSignature).join(',')}`,
    `streaming:${options.streamingRows.map((row) => buildMessageSignature(row.message)).join(',')}`,
    `thinking_panel:${buildThinkingPanelSignature(
      options.latestStreamingThinkingId,
      options.latestStreamingThinkingPanelOpen,
    )}`,
  ].join('|');
}

function buildThinkingPanelSignature(
  latestStreamingThinkingId: string | null,
  latestStreamingThinkingPanelOpen: boolean,
): string {
  if (!latestStreamingThinkingId) {
    return 'none';
  }

  return `${latestStreamingThinkingId}:${latestStreamingThinkingPanelOpen ? 1 : 0}`;
}

function buildMessageSignature(message: ChatMessage): string {
  switch (message.kind) {
    case 'tool':
      return [
        message.id,
        message.kind,
        message.content.length,
        message.toolStatus ?? '',
        message.toolInput?.length ?? 0,
      ].join(':');
    case 'question':
    case 'pending_question':
      return [
        message.id,
        message.kind,
        message.content.length,
        message.questionId ?? '',
        message.selectionMode ?? '',
        message.options?.length ?? 0,
      ].join(':');
    default:
      return [message.id, message.kind, message.content.length].join(':');
  }
}
