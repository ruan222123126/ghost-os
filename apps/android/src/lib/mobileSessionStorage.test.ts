// @vitest-environment jsdom
import { invoke } from "@tauri-apps/api/core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  loadPersistedMobileConversations,
  loadStoredMobileConversations,
  MOBILE_CONVERSATIONS_STORAGE_KEY,
} from "./mobileSessionStorage";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

describe("mobileSessionStorage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    Reflect.deleteProperty(window, "__TAURI_INTERNALS__");
  });

  it("uses localStorage outside Tauri", () => {
    window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify([storedConversation()]));

    expect(loadStoredMobileConversations()).toEqual([storedConversation()]);
  });

  it("migrates legacy localStorage conversations when the Tauri file is empty", async () => {
    Reflect.set(window, "__TAURI_INTERNALS__", {});
    window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify([storedConversation()]));
    vi.mocked(invoke).mockImplementation(async (command: string) => {
      if (command === "mobile_conversations_load") {
        return [];
      }
      return undefined;
    });

    const conversations = await loadPersistedMobileConversations();

    expect(conversations).toEqual([storedConversation()]);
    expect(invoke).toHaveBeenCalledWith("mobile_conversations_save", { conversations: [storedConversation()] });
    expect(window.localStorage.getItem(MOBILE_CONVERSATIONS_STORAGE_KEY)).toBeTruthy();
  });
});

function storedConversation() {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id: "session-1",
    messages: [],
    title: "Session 1",
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}
