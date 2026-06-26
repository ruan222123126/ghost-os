// @vitest-environment jsdom
import { invoke } from "@tauri-apps/api/core";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createOrUpdateLocalProvider, exportLocalProvider, loadLocalProviderList } from "./mobileLocalProviders";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

describe("mobileLocalProviders", () => {
  beforeEach(() => {
    window.localStorage.clear();
    setTauriRuntime(true);
    vi.clearAllMocks();
  });

  afterEach(() => {
    setTauriRuntime(false);
  });

  it("normalizes camelCase Tauri provider payloads without losing sync fields", async () => {
    vi.mocked(invoke).mockResolvedValue(camelProviderListPayload());

    const payload = await loadLocalProviderList();

    expect(payload.active_provider).toBe("Remote OpenAI");
    expect(payload.active_provider_id).toBe("provider-1");
    expect(payload.providers[0]).toMatchObject({
      api_key_set: true,
      base_url: "https://api.openai.com/v1",
      name: "Remote OpenAI",
      provider_id: "provider-1",
      type: "openai",
      updated_at: "2026-01-01T00:00:00.000Z",
    });
    expect(payload.provider_sync_records?.[0]).toMatchObject({
      api_key_set: true,
      provider_id: "provider-1",
    });
  });

  it("sends camelCase upsert payloads to the Tauri command", async () => {
    vi.mocked(invoke).mockResolvedValue(camelProviderListPayload());

    await createOrUpdateLocalProvider({
      api_key: "secret",
      base_url: "https://example.com/v1",
      context_window_tokens: 128000,
      models: ["gpt-4o"],
      name: "Remote OpenAI",
      provider_id: "provider-1",
      type: "openai",
    });

    expect(vi.mocked(invoke).mock.calls[0]).toEqual([
      "mobile_local_provider_upsert",
      {
        provider: expect.objectContaining({
          apiKey: "secret",
          baseUrl: "https://example.com/v1",
          contextWindowTokens: 128000,
          providerId: "provider-1",
          type: "openai",
        }),
      },
    ]);
  });

  it("normalizes camelCase Tauri export payloads", async () => {
    vi.mocked(invoke).mockResolvedValue({
      apiKey: "secret",
      baseUrl: "https://api.openai.com/v1",
      models: ["gpt-4o"],
      name: "Remote OpenAI",
      providerId: "provider-1",
      type: "openai",
      updatedAt: "2026-01-01T00:00:00.000Z",
    });

    await expect(exportLocalProvider("provider-1")).resolves.toMatchObject({
      api_key: "secret",
      base_url: "https://api.openai.com/v1",
      name: "Remote OpenAI",
      provider_id: "provider-1",
      type: "openai",
      updated_at: "2026-01-01T00:00:00.000Z",
    });
  });
});

function camelProviderListPayload(): Record<string, unknown> {
  return {
    activeProvider: "Remote OpenAI",
    activeProviderId: "provider-1",
    providers: [camelProviderPayload()],
    providerSyncRecords: [camelProviderPayload()],
  };
}

function camelProviderPayload(): Record<string, unknown> {
  return {
    apiKeySet: true,
    baseUrl: "https://api.openai.com/v1",
    models: ["gpt-4o"],
    name: "Remote OpenAI",
    providerId: "provider-1",
    type: "openai",
    updatedAt: "2026-01-01T00:00:00.000Z",
  };
}

function setTauriRuntime(enabled: boolean): void {
  const runtime = window as Window & { __TAURI_INTERNALS__?: unknown };
  if (enabled) {
    runtime.__TAURI_INTERNALS__ = {};
    return;
  }
  delete runtime.__TAURI_INTERNALS__;
}
