import type { MobileConversationMessage, StoredMobileConversation } from "../mobileTypes";

const MOBILE_CONVERSATIONS_STORAGE_KEY = "ghost-os-mobile.conversations.v1";

interface ConversationUpsert {
  id: string;
  title: string;
  messages: MobileConversationMessage[];
  createdAt?: string;
  updatedAt?: string;
}

export function loadStoredMobileConversations(): StoredMobileConversation[] {
  const raw = window.localStorage.getItem(MOBILE_CONVERSATIONS_STORAGE_KEY);
  if (!raw) {
    return [];
  }

  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      throw new Error("mobile conversation storage must be an array");
    }
    return parsed.map(normalizeConversation).filter(isStoredConversation);
  } catch (error) {
    console.error("[mobileSessionStorage] load conversations failed", error);
    return [];
  }
}

export function saveStoredMobileConversations(conversations: StoredMobileConversation[]): void {
  window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify(conversations));
}

export function upsertStoredMobileConversation(
  conversations: StoredMobileConversation[],
  upsert: ConversationUpsert,
): StoredMobileConversation[] {
  const id = upsert.id.trim();
  if (!id) {
    return conversations;
  }

  const now = new Date().toISOString();
  const existing = conversations.find((conversation) => conversation.id === id);
  const next: StoredMobileConversation = {
    created_at: upsert.createdAt?.trim() || existing?.created_at || now,
    id,
    messages: upsert.messages,
    title: existing?.title.trim() || upsert.title.trim() || id,
    updated_at: upsert.updatedAt?.trim() || now,
  };

  return [next, ...conversations.filter((conversation) => conversation.id !== id)].sort(compareConversationsByUpdatedAt);
}

export function appendStoredMobileMessages(
  conversation: StoredMobileConversation | undefined,
  messages: MobileConversationMessage[],
): MobileConversationMessage[] {
  const existing = conversation?.messages ?? [];
  const byID = new Map(existing.map((message) => [message.id, message]));
  for (const message of messages) {
    byID.set(message.id, message);
  }
  return [...byID.values()];
}

function normalizeConversation(value: unknown): StoredMobileConversation | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const record = value as Record<string, unknown>;
  const id = asTrimmedString(record.id);
  const title = asTrimmedString(record.title);
  const createdAt = asTrimmedString(record.created_at);
  const updatedAt = asTrimmedString(record.updated_at);
  if (!id || !createdAt || !updatedAt) {
    return null;
  }

  return {
    created_at: createdAt,
    id,
    messages: normalizeMessages(record.messages, id),
    title: title || id,
    updated_at: updatedAt,
  };
}

function normalizeMessages(value: unknown, sessionId: string): MobileConversationMessage[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => normalizeMessage(item, sessionId))
    .filter(isMobileConversationMessage);
}

function normalizeMessage(value: unknown, sessionId: string): MobileConversationMessage | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const record = value as Record<string, unknown>;
  const id = asTrimmedString(record.id);
  const role = record.role === "user" || record.role === "assistant" ? record.role : undefined;
  const text = asString(record.text);
  if (!id || !role || text === undefined) {
    return null;
  }

  return {
    id,
    role,
    sessionId,
    text,
    thinking: asString(record.thinking),
  };
}

function compareConversationsByUpdatedAt(a: StoredMobileConversation, b: StoredMobileConversation): number {
  const updatedOrder = b.updated_at.localeCompare(a.updated_at);
  if (updatedOrder !== 0) {
    return updatedOrder;
  }
  return a.title.localeCompare(b.title, "zh-Hans");
}

function isStoredConversation(value: StoredMobileConversation | null): value is StoredMobileConversation {
  return value !== null;
}

function isMobileConversationMessage(value: MobileConversationMessage | null): value is MobileConversationMessage {
  return value !== null;
}

function asTrimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}
