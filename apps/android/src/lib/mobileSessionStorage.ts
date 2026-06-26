import { invoke } from "@tauri-apps/api/core";
import type { MobileConversationMessage, MobileToolCard, MobileToolCardStatus, StoredMobileConversation } from "../mobileTypes";
import { hasTauriRuntime } from "./bridgeBus";

export const MOBILE_CONVERSATIONS_STORAGE_KEY = "ghost-os-mobile.conversations.v1";

interface ConversationUpsert {
  bridgeMessageCount?: number;
  id: string;
  title: string;
  messages: MobileConversationMessage[];
  createdAt?: string;
  preserveExistingTitle?: boolean;
  sourceMessageCount?: number;
  syncedMessageCount?: number;
  updatedAt?: string;
}

export function loadStoredMobileConversations(): StoredMobileConversation[] {
  return loadLocalStoredMobileConversations();
}

export async function loadPersistedMobileConversations(): Promise<StoredMobileConversation[]> {
  if (!hasTauriRuntime()) {
    return loadLocalStoredMobileConversations();
  }

  const persisted = normalizeConversationArray(await invoke<unknown>("mobile_conversations_load"));
  if (persisted.length > 0) {
    return persisted;
  }

  const legacy = loadLocalStoredMobileConversations();
  if (legacy.length > 0) {
    await saveTauriMobileConversations(legacy);
  }
  return legacy;
}

export function saveStoredMobileConversations(conversations: StoredMobileConversation[]): void {
  window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify(conversations));
}

export async function savePersistedMobileConversations(conversations: StoredMobileConversation[]): Promise<void> {
  if (hasTauriRuntime()) {
    await saveTauriMobileConversations(conversations);
    return;
  }
  saveStoredMobileConversations(conversations);
}

function loadLocalStoredMobileConversations(): StoredMobileConversation[] {
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

async function saveTauriMobileConversations(conversations: StoredMobileConversation[]): Promise<void> {
  await invoke("mobile_conversations_save", { conversations });
}

function normalizeConversationArray(value: unknown): StoredMobileConversation[] {
  if (!Array.isArray(value)) {
    throw new Error("mobile conversation storage must be an array");
  }
  return value.map(normalizeConversation).filter(isStoredConversation);
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
  const incomingTitle = upsert.title.trim();
  const existingTitle = existing?.title.trim();
  const next: StoredMobileConversation = {
    created_at: upsert.createdAt?.trim() || existing?.created_at || now,
    id,
    messages: upsert.messages,
    source_message_count: upsert.bridgeMessageCount ?? upsert.sourceMessageCount ?? existing?.source_message_count,
    synced_message_count: upsert.syncedMessageCount ?? existing?.synced_message_count,
    title: upsert.preserveExistingTitle ? existingTitle || incomingTitle || id : incomingTitle || existingTitle || id,
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
  const sourceMessageCount = asOptionalInteger(record.source_message_count);
  const syncedMessageCount = asOptionalInteger(record.synced_message_count);
  if (!id || !createdAt || !updatedAt) {
    return null;
  }

  return {
    created_at: createdAt,
    id,
    messages: normalizeMessages(record.messages, id),
    ...(sourceMessageCount === undefined ? {} : { source_message_count: sourceMessageCount }),
    ...(syncedMessageCount === undefined ? {} : { synced_message_count: syncedMessageCount }),
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
    tools: normalizeTools(record.tools),
  };
}

function normalizeTools(value: unknown): MobileToolCard[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }

  const tools = value.map(normalizeTool).filter(isMobileToolCard);
  return tools.length > 0 ? tools : undefined;
}

function normalizeTool(value: unknown): MobileToolCard | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, unknown>;
  const id = asTrimmedString(record.id);
  const status = asToolStatus(record.status);
  if (!id || !status) {
    return null;
  }

  return {
    id,
    error: asString(record.error),
    input: asString(record.input),
    output: asString(record.output),
    status,
    toolCallId: asString(record.toolCallId),
    toolName: asString(record.toolName),
    traceId: asString(record.traceId),
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

function isMobileToolCard(value: MobileToolCard | null): value is MobileToolCard {
  return value !== null;
}

function asTrimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function asOptionalInteger(value: unknown): number | undefined {
  return Number.isInteger(value) && typeof value === "number" && value >= 0 ? value : undefined;
}

function asToolStatus(value: unknown): MobileToolCardStatus | undefined {
  if (value === "pending" || value === "running" || value === "success" || value === "error") {
    return value;
  }
  return undefined;
}
