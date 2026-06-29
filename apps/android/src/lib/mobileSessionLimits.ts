import type {
  MobileSessionView,
  SessionMetadata,
  StoredMobileConversation,
} from "../mobileTypes";

export const MOBILE_HOT_SESSION_VIEW_LIMIT = 8;
export const MOBILE_PERSISTED_CONVERSATION_LIMIT = 50;
export const MOBILE_PERSISTED_SESSION_PAGE_LIMIT = 100;

interface TrimMobileSessionViewsOptions {
  activeSessionId?: string;
  limit?: number;
  pinnedSessionIds?: Set<string>;
}

interface TrimStoredMobileConversationsOptions {
  activeSessionId?: string;
  bridgeSessionIds?: Set<string>;
  limit?: number;
  pinnedSessionIds?: Set<string>;
  runningSessionIds?: Set<string>;
}

export function trimMobileSessionViews(
  views: Record<string, MobileSessionView>,
  options: TrimMobileSessionViewsOptions = {},
): Record<string, MobileSessionView> {
  const limit = positiveLimit(options.limit, MOBILE_HOT_SESSION_VIEW_LIMIT);
  const entries = Object.entries(views);
  if (entries.length <= limit) {
    return views;
  }

  const activeSessionId = options.activeSessionId?.trim() || "";
  const protectedEntries = entries.filter(([, view]) => isProtectedSessionView(view, activeSessionId));
  const protectedIds = new Set(protectedEntries.map(([id]) => id));
  const candidates = entries
    .filter(([id]) => !protectedIds.has(id))
    .sort(([, left], [, right]) => compareSessionViewsForRetention(left, right, options.pinnedSessionIds));
  const retained = [
    ...protectedEntries,
    ...candidates.slice(0, Math.max(limit - protectedEntries.length, 0)),
  ];

  return Object.fromEntries(retained);
}

export function trimStoredMobileConversations(
  conversations: StoredMobileConversation[],
  options: TrimStoredMobileConversationsOptions = {},
): StoredMobileConversation[] {
  const limit = positiveLimit(options.limit, MOBILE_PERSISTED_CONVERSATION_LIMIT);
  if (conversations.length <= limit) {
    return conversations;
  }

  const protectedConversations: StoredMobileConversation[] = [];
  const candidates: StoredMobileConversation[] = [];
  for (const conversation of conversations) {
    if (isProtectedStoredConversation(conversation, options)) {
      protectedConversations.push(conversation);
      continue;
    }
    candidates.push(conversation);
  }

  return [
    ...protectedConversations,
    ...candidates
      .sort(compareStoredConversationsForRetention)
      .slice(0, Math.max(limit - protectedConversations.length, 0)),
  ].sort(compareStoredConversationsForRetention);
}

export function recentMobileBridgeSessions(
  sessions: SessionMetadata[],
  limit = MOBILE_PERSISTED_CONVERSATION_LIMIT,
): SessionMetadata[] {
  return [...sessions]
    .sort(compareSessionsForRetention)
    .slice(0, positiveLimit(limit, MOBILE_PERSISTED_CONVERSATION_LIMIT));
}

function isProtectedSessionView(view: MobileSessionView, activeSessionId: string): boolean {
  return view.id === activeSessionId || view.run.status === "running";
}

function isProtectedStoredConversation(
  conversation: StoredMobileConversation,
  options: TrimStoredMobileConversationsOptions,
): boolean {
  const id = conversation.id.trim();
  if (!id) {
    return false;
  }
  if (id === options.activeSessionId?.trim()) {
    return true;
  }
  if (options.pinnedSessionIds?.has(id) || options.runningSessionIds?.has(id)) {
    return true;
  }
  if (hasUnsyncedMessages(conversation)) {
    return true;
  }
  return !options.bridgeSessionIds?.has(id) && conversation.messages.length > 0;
}

function hasUnsyncedMessages(conversation: StoredMobileConversation): boolean {
  return typeof conversation.synced_message_count === "number"
    && conversation.messages.length > conversation.synced_message_count;
}

function compareSessionViewsForRetention(
  left: MobileSessionView,
  right: MobileSessionView,
  pinnedSessionIds?: Set<string>,
): number {
  const priority = sessionViewPriority(right, pinnedSessionIds) - sessionViewPriority(left, pinnedSessionIds);
  if (priority !== 0) {
    return priority;
  }
  return compareUpdatedAt(right.updatedAt, left.updatedAt) || left.title.localeCompare(right.title, "zh-Hans");
}

function sessionViewPriority(view: MobileSessionView, pinnedSessionIds?: Set<string>): number {
  if (view.unread) {
    return 2;
  }
  if (pinnedSessionIds?.has(view.id)) {
    return 1;
  }
  return 0;
}

function compareStoredConversationsForRetention(
  left: StoredMobileConversation,
  right: StoredMobileConversation,
): number {
  return compareUpdatedAt(right.updated_at, left.updated_at) || left.title.localeCompare(right.title, "zh-Hans");
}

function compareSessionsForRetention(left: SessionMetadata, right: SessionMetadata): number {
  return compareUpdatedAt(right.updated_at, left.updated_at) || left.title.localeCompare(right.title, "zh-Hans");
}

function compareUpdatedAt(left: string, right: string): number {
  return left.localeCompare(right);
}

function positiveLimit(value: number | undefined, fallback: number): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value <= 0) {
    return fallback;
  }
  return value;
}
