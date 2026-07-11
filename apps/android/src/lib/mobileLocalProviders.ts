import { invoke } from "@tauri-apps/api/core";
import type {
  ProviderConfigInputPayload,
  ProviderConfigPayload,
  ProviderListPayload,
  ProviderSyncRecordPayload,
} from "../mobileTypes";
import { hasTauriRuntime } from "./bridgeBus";

const LOCAL_PROVIDERS_STORAGE_KEY = "ghost-os-mobile.local-providers.v1";

interface LocalProviderState {
  records: ProviderConfigPayload[];
}

export interface LocalProviderExportPayload extends ProviderConfigInputPayload {
  provider_id: string;
  updated_at: string;
}

type ProviderRecordInput = Partial<ProviderConfigInputPayload & ProviderConfigPayload> & Record<string, unknown>;

function nowISO(): string {
  return new Date().toISOString();
}

export async function loadLocalProviderList(): Promise<ProviderListPayload> {
  if (hasTauriRuntime()) {
    return normalizeProviderList(await invoke<unknown>("mobile_local_provider_list"));
  }
  return normalizeProviderList(readLocalProviderState());
}

export async function createOrUpdateLocalProvider(
  provider: ProviderConfigInputPayload,
): Promise<ProviderListPayload> {
  if (hasTauriRuntime()) {
    return normalizeProviderList(
      await invoke<unknown>("mobile_local_provider_upsert", { provider: providerInputForTauri(provider) }),
    );
  }

  const state = readLocalProviderState();
  const normalized = normalizeProviderRecord(provider, state.records);
  const nextRecords = [normalized, ...state.records.filter((item) => item.provider_id !== normalized.provider_id)];
  writeLocalProviderState({ records: nextRecords });
  return normalizeProviderList({ records: nextRecords });
}

export async function deleteLocalProvider(providerId: string): Promise<ProviderListPayload> {
  const trimmedId = providerId.trim();
  if (!trimmedId) {
    return loadLocalProviderList();
  }

  if (hasTauriRuntime()) {
    return normalizeProviderList(await invoke<unknown>("mobile_local_provider_delete", { providerId: trimmedId }));
  }

  const state = readLocalProviderState();
  const timestamp = nowISO();
  const nextRecords = state.records.map((record) =>
    record.provider_id === trimmedId
      ? {
          ...record,
          deleted_at: timestamp,
          sync_state: "local" as const,
          updated_at: timestamp,
        }
      : record,
  );
  writeLocalProviderState({ records: nextRecords });
  return normalizeProviderList({ records: nextRecords });
}

export async function exportLocalProvider(providerId: string): Promise<LocalProviderExportPayload> {
  const trimmed = providerId.trim();
  if (!trimmed) {
    throw new Error("provider_id is required");
  }
  if (hasTauriRuntime()) {
    return normalizeLocalProviderExport(await invoke<unknown>("mobile_local_provider_export", { providerId: trimmed }));
  }
  const record = readLocalProviderState().records.find((provider) => provider.provider_id === trimmed);
  if (!record) {
    throw new Error(`local provider not found: ${trimmed}`);
  }
  return {
    base_url: record.base_url,
    deleted_at: record.deleted_at,
    model_context_window_tokens: record.model_context_window_tokens,
    model_response_reserve_tokens: record.model_response_reserve_tokens,
    models: record.models,
    name: record.name,
    provider_id: record.provider_id,
    response_reserve_tokens: record.response_reserve_tokens,
    type: record.type,
    updated_at: record.updated_at,
  };
}

function readLocalProviderState(): LocalProviderState {
  const raw = window.localStorage.getItem(LOCAL_PROVIDERS_STORAGE_KEY);
  if (!raw) {
    return { records: [] };
  }

  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return { records: [] };
    }
    const records = Array.isArray((parsed as { records?: unknown[] }).records)
      ? normalizeProviderRecords((parsed as { records: unknown[] }).records)
      : [];
    return { records };
  } catch {
    return { records: [] };
  }
}

function writeLocalProviderState(state: LocalProviderState): void {
  window.localStorage.setItem(LOCAL_PROVIDERS_STORAGE_KEY, JSON.stringify(state));
}

function normalizeProviderList(value: unknown): ProviderListPayload {
  const record = objectRecord(value);
  const syncRecords = record
    ? (record.provider_sync_records ?? record.providerSyncRecords)
    : undefined;
  const providers = record?.providers;
  const records = isProviderListPayload(value) || providers || syncRecords
    ? normalizeProviderRecords(syncRecords ?? providers)
    : record && "records" in record
      ? normalizeProviderRecords(record.records)
      : [];

  return {
    active_provider: stringField(record, "active_provider", "activeProvider"),
    active_provider_id: optionalStringField(record, "active_provider_id", "activeProviderId"),
    provider_sync_records: records.map(providerToSyncRecord),
    providers: records.filter((record) => !record.deleted_at),
  };
}

function normalizeProviderRecords(value: unknown): ProviderConfigPayload[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const current = value
    .map((item) => normalizeProviderRecord(item, []))
    .filter((item) => Boolean(item.provider_id && item.name));
  return current;
}

function normalizeProviderRecord(
  value: unknown,
  existing: ProviderConfigPayload[],
): ProviderConfigPayload {
  const input = providerRecordInput(value);
  const name = stringField(input, "name");
  const providerID = stringField(input, "provider_id", "providerId")
    || findExistingProviderID(existing, name)
    || crypto.randomUUID();
  const deletedAt = optionalStringField(input, "deleted_at", "deletedAt");
  const updatedAt = stringField(input, "updated_at", "updatedAt") || nowISO();
  return {
    api_key_set: Boolean(stringField(input, "api_key", "apiKey")) || booleanField(input, "api_key_set", "apiKeySet"),
    base_url: stringField(input, "base_url", "baseUrl"),
    context_window_tokens: optionalNumberField(input, "context_window_tokens", "contextWindowTokens"),
    deleted_at: deletedAt,
    model_context_window_tokens: optionalNumberRecordField(input, "model_context_window_tokens", "modelContextWindowTokens"),
    model_response_reserve_tokens: optionalNumberRecordField(
      input,
      "model_response_reserve_tokens",
      "modelResponseReserveTokens",
    ),
    models: stringArrayField(input, "models"),
    name,
    provider_id: providerID,
    response_reserve_tokens: optionalNumberField(input, "response_reserve_tokens", "responseReserveTokens"),
    sync_state: syncStateField(input),
    type: providerTypeField(input),
    updated_at: updatedAt,
  };
}

function normalizeLocalProviderExport(value: unknown): LocalProviderExportPayload {
  const provider = normalizeProviderRecord(value, []);
  const input = providerRecordInput(value);
  const apiKey = stringField(input, "api_key", "apiKey");
  const contextWindowTokens = optionalNumberField(input, "context_window_tokens", "contextWindowTokens");
  return {
    base_url: provider.base_url,
    deleted_at: provider.deleted_at,
    model_context_window_tokens: provider.model_context_window_tokens,
    model_response_reserve_tokens: provider.model_response_reserve_tokens,
    models: provider.models,
    name: provider.name,
    provider_id: provider.provider_id,
    response_reserve_tokens: provider.response_reserve_tokens,
    type: provider.type,
    updated_at: provider.updated_at,
    ...(contextWindowTokens === undefined ? {} : { context_window_tokens: contextWindowTokens }),
    ...(apiKey ? { api_key: apiKey } : {}),
  };
}

function providerToSyncRecord(provider: ProviderConfigPayload): ProviderSyncRecordPayload {
  return {
    api_key_set: provider.api_key_set,
    base_url: provider.base_url,
    context_window_tokens: provider.context_window_tokens,
    deleted_at: provider.deleted_at,
    model_context_window_tokens: provider.model_context_window_tokens,
    model_response_reserve_tokens: provider.model_response_reserve_tokens,
    models: provider.models,
    name: provider.name,
    provider_id: provider.provider_id,
    response_reserve_tokens: provider.response_reserve_tokens,
    type: provider.type,
    updated_at: provider.updated_at,
  };
}

function findExistingProviderID(records: ProviderConfigPayload[], name: string): string | undefined {
  const normalizedName = name.trim().toLowerCase();
  if (!normalizedName) {
    return undefined;
  }
  return records.find((record) => record.name.trim().toLowerCase() === normalizedName)?.provider_id;
}

function isProviderListPayload(value: unknown): value is ProviderListPayload {
  return Boolean(value && typeof value === "object" && !Array.isArray(value) && "providers" in value);
}

function providerInputForTauri(provider: ProviderConfigInputPayload): Record<string, unknown> {
  return {
    apiKey: provider.api_key,
    baseUrl: provider.base_url,
    contextWindowTokens: provider.context_window_tokens,
    deletedAt: provider.deleted_at,
    modelContextWindowTokens: provider.model_context_window_tokens,
    modelResponseReserveTokens: provider.model_response_reserve_tokens,
    models: provider.models,
    name: provider.name,
    providerId: provider.provider_id,
    responseReserveTokens: provider.response_reserve_tokens,
    type: provider.type,
    updatedAt: provider.updated_at,
  };
}

function providerRecordInput(value: unknown): ProviderRecordInput {
  return (objectRecord(value) ?? {}) as ProviderRecordInput;
}

function objectRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function stringField(record: Record<string, unknown> | undefined, ...keys: string[]): string {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed) {
        return trimmed;
      }
    }
  }
  return "";
}

function optionalStringField(record: Record<string, unknown> | undefined, ...keys: string[]): string | undefined {
  return stringField(record, ...keys) || undefined;
}

function booleanField(record: Record<string, unknown>, ...keys: string[]): boolean {
  return keys.some((key) => record[key] === true);
}

function optionalNumberField(record: Record<string, unknown>, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "number" && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function stringArrayField(record: Record<string, unknown>, key: string): string[] | undefined {
  const value = record[key];
  if (!Array.isArray(value)) {
    return undefined;
  }
  return value.map((item) => (typeof item === "string" ? item.trim() : "")).filter(Boolean);
}

function optionalNumberRecordField(record: Record<string, unknown>, ...keys: string[]): Record<string, number> | undefined {
  for (const key of keys) {
    const value = objectRecord(record[key]);
    if (!value) {
      continue;
    }
    const entries = Object.entries(value).filter((entry): entry is [string, number] =>
      typeof entry[1] === "number" && Number.isFinite(entry[1]),
    );
    if (entries.length > 0) {
      return Object.fromEntries(entries);
    }
  }
  return undefined;
}

function providerTypeField(record: Record<string, unknown>): ProviderConfigPayload["type"] {
  const value = stringField(record, "type", "provider_type", "providerType");
  if (value === "anthropic" || value === "codex" || value === "custom" || value === "openai") {
    return value;
  }
  return "openai";
}

function syncStateField(record: Record<string, unknown>): ProviderConfigPayload["sync_state"] {
  const value = stringField(record, "sync_state", "syncState");
  return value === "synced" ? "synced" : "local";
}
