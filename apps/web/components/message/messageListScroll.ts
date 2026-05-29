import type { ChatMessage } from '@/lib/types';
import type { StreamingMessageRow } from '@/lib/chat-view/streamingRows';

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

export interface VisibleMessageTailSnapshot {
  count: number;
  id: string | null;
}

interface PostSendAnchorLayoutOptions {
  anchorStartPx: number;
  containerHeightPx: number;
  realContentHeightPx: number;
}

interface PostSendOverflowDecisionOptions extends PostSendAnchorLayoutOptions {
  autoFollow: boolean;
  baselineContentHeightPx: number;
  hasVisibleAssistantText: boolean;
}

export interface PostSendOverflowDecision {
  overflowed: boolean;
  shouldScrollToBottom: boolean;
  trailingSpacerPx: number | null;
}

const POST_SEND_USER_ID_PREFIX = 'local:user:';

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

export function buildVisibleMessageTailSnapshot(
  messages: ChatMessage[],
): VisibleMessageTailSnapshot {
  return {
    count: messages.length,
    id: messages[messages.length - 1]?.id ?? null,
  };
}

export function getPostSendAnchorIndexFromVisibleMessages(options: {
  messages: ChatMessage[];
  previousTail: VisibleMessageTailSnapshot | null;
}): number | null {
  if (!options.previousTail) {
    return null;
  }
  if (options.messages.length <= options.previousTail.count) {
    return null;
  }

  const anchorIndex = options.messages.length - 1;
  const tail = options.messages[anchorIndex];
  if (!tail || tail.kind !== 'user') {
    return null;
  }
  if (!isPostSendCommittedUserMessageId(tail.id)) {
    return null;
  }
  return tail.id === options.previousTail.id ? null : anchorIndex;
}

export function isPostSendCommittedUserMessageId(messageId: string): boolean {
  return messageId.startsWith(POST_SEND_USER_ID_PREFIX);
}

export function computePostSendAnchorLayout(
  options: PostSendAnchorLayoutOptions,
) {
  const viewportBottomPx = options.anchorStartPx + options.containerHeightPx;
  const missingPx = viewportBottomPx - options.realContentHeightPx;

  return {
    anchorScrollTopPx: options.anchorStartPx,
    trailingSpacerPx: missingPx > 0 ? missingPx : 0,
    viewportBottomPx,
  };
}

export function hasPostSendRealContentOverflow(
  options: PostSendAnchorLayoutOptions,
): boolean {
  return options.realContentHeightPx > options.anchorStartPx + options.containerHeightPx;
}

export function resolvePostSendOverflowDecision(
  options: PostSendOverflowDecisionOptions,
): PostSendOverflowDecision {
  const overflowed = options.hasVisibleAssistantText
    && options.realContentHeightPx > options.baselineContentHeightPx
    && hasPostSendRealContentOverflow(options);
  if (!overflowed) {
    return {
      overflowed,
      shouldScrollToBottom: false,
      trailingSpacerPx: null,
    };
  }

  return {
    overflowed,
    shouldScrollToBottom: options.autoFollow,
    trailingSpacerPx: 0,
  };
}

export function hasVisibleAssistantTextAfterIndex(
  messages: ChatMessage[],
  anchorIndex: number | null,
): boolean {
  if (anchorIndex === null) {
    return false;
  }

  return messages
    .slice(anchorIndex + 1)
    .some((message) => message.kind === 'assistant' && message.content.trim().length > 0);
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
