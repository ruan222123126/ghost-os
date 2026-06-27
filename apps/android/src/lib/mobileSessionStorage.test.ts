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

  it("preserves approval tool metadata when loading stored conversations", () => {
    window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify([storedConversationWithApprovalTool()]));

    expect(loadStoredMobileConversations()).toEqual([storedConversationWithApprovalTool()]);
  });

  it("preserves ordered assistant parts when loading stored conversations", () => {
    window.localStorage.setItem(MOBILE_CONVERSATIONS_STORAGE_KEY, JSON.stringify([storedConversationWithOrderedParts()]));

    expect(loadStoredMobileConversations()).toEqual([storedConversationWithOrderedParts()]);
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

function storedConversationWithApprovalTool() {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id: "session-1",
    messages: [
      {
        id: "session-1:assistant:1",
        role: "assistant",
        sessionId: "session-1",
        text: "",
        tools: [
          {
            approvalId: "approval-1",
            approvalKind: "exec",
            approvalPayload: { command: "go test ./..." },
            approvalPrompt: "Approve command execution",
            id: "tool-approval-1",
            input: "Approve command execution",
            status: "pending",
            toolCallId: "approval:approval-1",
            toolName: "codex_approval",
          },
        ],
      },
    ],
    title: "Session 1",
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}

function storedConversationWithOrderedParts() {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id: "session-2",
    messages: [
      {
        id: "session-2:assistant:1",
        parts: [
          {
            id: "text-1",
            kind: "text",
            text: "先开始绘图。",
          },
          {
            id: "tool-1",
            kind: "tool",
            tool: {
              id: "tool-1",
              output: "Generated 1 image(s).",
              status: "success",
              toolCallId: "call-draw",
              toolName: "screen_action",
            },
          },
          {
            id: "text-2",
            kind: "text",
            text: "图片已经生成。",
          },
        ],
        role: "assistant",
        sessionId: "session-2",
        text: "先开始绘图。图片已经生成。",
      },
    ],
    title: "Session 2",
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}
