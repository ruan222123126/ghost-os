import type { ChatMessage } from '@/lib/types';
import type { StreamingMessageRow } from '@/lib/chat-view/types';

export const MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX = 120;
const POST_SEND_SCROLL_LOCK_EPSILON_PX = 1;

interface ScrollMetrics {
  scrollHeight: number;
  clientHeight: number;
  scrollTop: number;
}

export interface PostSendFollowTrackingState {
  mode: 'idle' | 'anchoring' | 'waiting_overflow';
  controlledScrollTopPx: number | null;
}

interface MessageListLayoutSignatureOptions {
  committedMessages: ChatMessage[];
  streamingRows: StreamingMessageRow[];
  showProcessingTimer: boolean;
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
  hasVisibleContent: boolean;
}

export interface PostSendOverflowDecision {
  overflowed: boolean;
  shouldScrollToBottom: boolean;
  trailingSpacerPx: number;
}

const POST_SEND_USER_ID_PREFIX = 'local:user:';

export function isMessageListNearBottom(metrics: ScrollMetrics): boolean {
  const distance = metrics.scrollHeight - metrics.clientHeight - metrics.scrollTop;
  return distance <= MESSAGE_LIST_BOTTOM_FOLLOW_THRESHOLD_PX;
}

export function resolveMessageListAutoFollow(
  metrics: ScrollMetrics,
  tracking: PostSendFollowTrackingState,
): boolean {
  if (isPostSendFocusLocked(tracking)) {
    if (tracking.controlledScrollTopPx === null) {
      return false;
    }
    return metrics.scrollTop + POST_SEND_SCROLL_LOCK_EPSILON_PX >= tracking.controlledScrollTopPx;
  }

  return isMessageListNearBottom(metrics);
}

export function shouldAdjustScrollPositionOnItemSizeChange(
  autoFollow: boolean,
  trackingMode: PostSendFollowTrackingState['mode'] = 'idle',
): boolean {
  return autoFollow && trackingMode === 'idle';
}

export function buildMessageListLayoutSignature(
  options: MessageListLayoutSignatureOptions,
): string {
  return [
    `history:${options.loadingOlderHistory ? 1 : 0}`,
    `processing:${options.showProcessingTimer ? 1 : 0}`,
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

export function shouldReleasePostSendAnchor(options: {
  anchorIndex: number | null;
  loading: boolean;
  messages: ChatMessage[];
}): boolean {
  if (options.anchorIndex === null) {
    return false;
  }

  const anchoredMessage = options.messages[options.anchorIndex];
  if (!anchoredMessage || anchoredMessage.kind !== 'user') {
    return true;
  }
  if (!isPostSendCommittedUserMessageId(anchoredMessage.id)) {
    return true;
  }
  if (!options.loading) {
    return true;
  }

  return options.messages[options.messages.length - 1]?.id !== anchoredMessage.id;
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

export function getPostSendLockedScrollTop(
  metrics: ScrollMetrics,
  tracking: PostSendFollowTrackingState,
): number | null {
  if (!isPostSendFocusLocked(tracking)) {
    return null;
  }
  if (tracking.controlledScrollTopPx === null) {
    return null;
  }
  if (metrics.scrollTop <= tracking.controlledScrollTopPx + POST_SEND_SCROLL_LOCK_EPSILON_PX) {
    return null;
  }

  return tracking.controlledScrollTopPx;
}

export function hasPostSendRealContentOverflow(
  options: PostSendAnchorLayoutOptions,
): boolean {
  return options.realContentHeightPx > options.anchorStartPx + options.containerHeightPx;
}

export function resolvePostSendOverflowDecision(
  options: PostSendOverflowDecisionOptions,
): PostSendOverflowDecision {
  const overflowLayout = computePostSendAnchorLayout(options);
  const overflowed = options.hasVisibleContent
    && options.realContentHeightPx > options.baselineContentHeightPx
    && hasPostSendRealContentOverflow(options);
  if (!overflowed) {
    return {
      overflowed,
      shouldScrollToBottom: false,
      trailingSpacerPx: overflowLayout.trailingSpacerPx,
    };
  }

  return {
    overflowed,
    shouldScrollToBottom: options.autoFollow,
    trailingSpacerPx: 0,
  };
}

export function hasVisibleContentAfterIndex(
  messages: ChatMessage[],
  anchorIndex: number | null,
): boolean {
  if (anchorIndex === null) {
    return false;
  }

  return messages
    .slice(anchorIndex + 1)
    .some(hasVisibleMessageContent);
}

function hasVisibleMessageContent(message: ChatMessage): boolean {
  switch (message.kind) {
    case 'user':
      return false;
    case 'tool':
      return hasNonBlankText(message.content)
        || hasNonBlankText(message.toolInput)
        || hasNonBlankText(message.toolName)
        || (message.images?.length ?? 0) > 0
        || (message.attachments?.length ?? 0) > 0;
    default:
      return hasNonBlankText(message.content);
  }
}

function hasNonBlankText(value: string | undefined): boolean {
  return (value?.trim().length ?? 0) > 0;
}

function isPostSendFocusLocked(tracking: PostSendFollowTrackingState): boolean {
  return tracking.mode === 'anchoring' || tracking.mode === 'waiting_overflow';
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
    case 'thinking':
      return [message.id, message.kind, message.content.length, message.inProgress ? 1 : 0].join(':');
    default:
      return [message.id, message.kind, message.content.length].join(':');
  }
}
