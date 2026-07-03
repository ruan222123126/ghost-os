import { invoke } from "@tauri-apps/api/core";
import type {
  MobileAssistantPart,
  MobileConversationMessage,
  MobileToolCard,
  MobileToolCardStatus,
  StoredMobileConversation,
} from "../mobileTypes";
import { hasTauriRuntime } from "./bridgeBus";

export const MOBILE_CONVERSATIONS_STORAGE_KEY = "ghost-os-mobile.conversations.v1";
export const MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY = "ghost-os-mobile.lastActiveSessionId.v1";

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

interface PersistedConversationUpsertOptions {
  limit?: number;
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

export async function loadPersistedMobileConversationIndex(): Promise<StoredMobileConversation[]> {
  if (!hasTauriRuntime()) {
    return loadLocalStoredMobileConversations();
  }

  const persisted = normalizeConversationArray(await invoke<unknown>("mobile_conversations_load_index"));
  if (persisted.length > 0) {
    return persisted;
  }

  const legacy = loadLocalStoredMobileConversations();
  if (legacy.length > 0) {
    await saveTauriMobileConversations(legacy);
  }
  return compactStoredMobileConversations(legacy);
}

export async function loadPersistedMobileConversation(
  sessionId: string,
): Promise<StoredMobileConversation | undefined> {
  const id = sessionId.trim();
  if (!id) {
    return undefined;
  }

  if (hasTauriRuntime()) {
    const conversation = await invoke<unknown>("mobile_conversation_get", { sessionId: id });
    return normalizeOptionalConversation(conversation);
  }

  return loadLocalStoredMobileConversations().find((conversation) => conversation.id === id);
}

export function saveStoredMobileConversations(conversations: StoredMobileConversation[]): void {
  window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify(conversations));
}

export function loadStoredLastActiveMobileSessionId(): string {
  return window.localStorage.getItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY)?.trim() || "";
}

export function saveStoredLastActiveMobileSessionId(sessionId: string | undefined): void {
  const trimmedSessionId = sessionId?.trim() || "";
  if (!trimmedSessionId) {
    window.localStorage.removeItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY);
    return;
  }
  window.localStorage.setItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY, trimmedSessionId);
}

export async function savePersistedMobileConversations(conversations: StoredMobileConversation[]): Promise<void> {
  if (hasTauriRuntime()) {
    await saveTauriMobileConversations(conversations);
    return;
  }
  saveStoredMobileConversations(conversations);
}

export async function upsertPersistedMobileConversations(
  conversations: StoredMobileConversation[],
  options: PersistedConversationUpsertOptions = {},
): Promise<StoredMobileConversation[]> {
  if (conversations.length === 0) {
    return loadPersistedMobileConversationIndex();
  }

  if (hasTauriRuntime()) {
    return normalizeConversationArray(
      await invoke<unknown>("mobile_conversations_upsert", {
        conversations,
        limit: options.limit,
      }),
    );
  }

  const current = loadLocalStoredMobileConversations();
  const nextById = new Map(current.map((conversation) => [conversation.id, conversation]));
  for (const conversation of conversations) {
    nextById.set(conversation.id, conversation);
  }
  const next = trimPersistedConversations([...nextById.values()], options.limit);
  saveStoredMobileConversations(next);
  return next;
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

function normalizeOptionalConversation(value: unknown): StoredMobileConversation | undefined {
  if (value === null || value === undefined) {
    return undefined;
  }
  return normalizeConversation(value) ?? undefined;
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
    parts: role === "assistant" ? normalizeAssistantParts(record.parts) : undefined,
    role,
    selectedSkill: normalizeSelectedSkill(record.selectedSkill),
    sessionId,
    text,
    thinking: asString(record.thinking),
    tools: normalizeTools(record.tools),
  };
}

function normalizeSelectedSkill(value: unknown): MobileConversationMessage["selectedSkill"] {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  const record = value as Record<string, unknown>;
  const id = asTrimmedString(record.id);
  const name = asTrimmedString(record.name);
  if (!id || !name) {
    return undefined;
  }
  return { id, name };
}

function normalizeAssistantParts(value: unknown): MobileAssistantPart[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }

  const parts = value.map(normalizeAssistantPart).filter(isMobileAssistantPart);
  return parts.length > 0 ? parts : undefined;
}

function normalizeAssistantPart(value: unknown): MobileAssistantPart | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, unknown>;
  const id = asTrimmedString(record.id);
  if (!id) {
    return null;
  }

  if (record.kind === "text") {
    const text = asString(record.text);
    if (text === undefined) {
      return null;
    }
    return {
      id,
      kind: "text",
      text,
    };
  }

  if (record.kind === "tool") {
    const tool = normalizeTool(record.tool);
    if (!tool) {
      return null;
    }
    return {
      id,
      kind: "tool",
      tool,
    };
  }

  return null;
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
    approvalDecision: asApprovalDecision(record.approvalDecision),
    id,
    approvalId: asString(record.approvalId),
    approvalInFlight: asBoolean(record.approvalInFlight),
    approvalKind: asString(record.approvalKind),
    approvalPayload: asRecord(record.approvalPayload),
    approvalPrompt: asString(record.approvalPrompt),
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

function trimPersistedConversations(
  conversations: StoredMobileConversation[],
  limit: number | undefined,
): StoredMobileConversation[] {
  if (typeof limit !== "number" || !Number.isInteger(limit) || limit <= 0) {
    return conversations.sort(compareConversationsByUpdatedAt);
  }
  return conversations.sort(compareConversationsByUpdatedAt).slice(0, limit);
}

function compactStoredMobileConversations(
  conversations: StoredMobileConversation[],
): StoredMobileConversation[] {
  return conversations.map((conversation) => ({
    ...conversation,
    messages: [],
  }));
}

function isStoredConversation(value: StoredMobileConversation | null): value is StoredMobileConversation {
  return value !== null;
}

function isMobileConversationMessage(value: MobileConversationMessage | null): value is MobileConversationMessage {
  return value !== null;
}

function isMobileAssistantPart(value: MobileAssistantPart | null): value is MobileAssistantPart {
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

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : undefined;
}

function asApprovalDecision(value: unknown): MobileToolCard["approvalDecision"] {
  return value === "approved"
    || value === "approved_for_session"
    || value === "denied"
    || value === "abort"
    ? value
    : undefined;
}

function asToolStatus(value: unknown): MobileToolCardStatus | undefined {
  if (value === "pending" || value === "running" || value === "success" || value === "error") {
    return value;
  }
  return undefined;
}
