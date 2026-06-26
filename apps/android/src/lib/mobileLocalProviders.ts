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
    return normalizeProviderList(await invoke<unknown>("mobile_local_provider_upsert", { provider }));
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
    return invoke<LocalProviderExportPayload>("mobile_local_provider_export", { providerId: trimmed });
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
  const records = isProviderListPayload(value)
    ? normalizeProviderRecords(value.provider_sync_records ?? value.providers)
    : value && typeof value === "object" && !Array.isArray(value) && "records" in value
      ? normalizeProviderRecords((value as { records?: unknown[] }).records ?? [])
      : [];

  return {
    active_provider: "",
    provider_sync_records: records.map(providerToSyncRecord),
    providers: records.filter((record) => !record.deleted_at),
  };
}

function normalizeProviderRecords(value: unknown): ProviderConfigPayload[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const current = value
    .map((item) => normalizeProviderRecord(item as Partial<ProviderConfigInputPayload & ProviderConfigPayload>, []))
    .filter((item) => Boolean(item.provider_id && item.name));
  return current;
}

function normalizeProviderRecord(
  input: Partial<ProviderConfigInputPayload & ProviderConfigPayload>,
  existing: ProviderConfigPayload[],
): ProviderConfigPayload {
  const providerID = input.provider_id?.trim()
    || findExistingProviderID(existing, input.name?.trim() || "")
    || crypto.randomUUID();
  const deletedAt = input.deleted_at?.trim() || undefined;
  const updatedAt = input.updated_at?.trim() || nowISO();
  return {
    api_key_set: input.api_key?.trim() ? true : input.api_key_set === true,
    base_url: input.base_url?.trim() || "",
    context_window_tokens: input.context_window_tokens,
    deleted_at: deletedAt,
    model_context_window_tokens: input.model_context_window_tokens,
    model_response_reserve_tokens: input.model_response_reserve_tokens,
    models: input.models?.map((item) => item.trim()).filter(Boolean),
    name: input.name?.trim() || "",
    provider_id: providerID,
    response_reserve_tokens: input.response_reserve_tokens,
    sync_state: input.sync_state ?? "local",
    type: input.type ?? "openai",
    updated_at: updatedAt,
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
