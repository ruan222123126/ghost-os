// @vitest-environment jsdom
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useMobileSessions, mergeHistoryItems, reconcileStoredConversationsWithBridge } from "./useMobileSessions";
import type {
  AgentPayload,
  MobileConversationMessage,
  MobileSessionView,
  SessionDetail,
  SessionMetadata,
  StoredMobileConversation,
} from "../mobileTypes";

const STORAGE_KEY = "ghost-os-mobile.conversations.v1";

describe("useMobileSessions", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.useRealTimers();
  });

  it("creates and activates a Bridge session from the home page without sending session_id", async () => {
    const sendAgentMessage = vi.fn(async (options: SendOptions) => {
      expect(options.sessionId).toBeUndefined();
      options.onSessionId("session-1");
      options.onReply(agentReply("session-1", "reply"));
      options.onStatus({ tone: "success", text: "回复已返回" });
      return { ok: true, reply: agentReply("session-1", "reply"), sessionId: "session-1" };
    });
    const { result } = renderMobileSessions({ sendAgentMessage });

    await act(async () => {
      await result.current.sendMessage("first task");
    });

    expect(result.current.activeSessionId).toBe("session-1");
    expect(result.current.activeMessages.map((message) => message.role)).toEqual(["user", "assistant"]);
    expect(result.current.historyItems[0]).toMatchObject({ id: "session-1", title: "first task" });
    expect(loadStored()[0]?.title).toBe("first task");
  });

  it("selects a cached session and sends follow-up messages with its session_id", async () => {
    saveStored([storedConversation("session-1", "Cached title", [message("session-1", "user", "old")])]);
    const sendAgentMessage = vi.fn(async (options: SendOptions) => {
      expect(options.sessionId).toBe("session-1");
      options.onReply(agentReply("session-1", "continued"));
      return { ok: true, reply: agentReply("session-1", "continued"), sessionId: "session-1" };
    });
    const { result, rerender } = renderMobileSessions({ bridgeConnected: false, sendAgentMessage });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    expect(result.current.activeSessionId).toBe("session-1");
    expect(result.current.canSend).toBe(false);

    rerender({
      bridgeConnected: true,
      getSession: vi.fn(),
      pinnedHistoryIds: [],
      sendAgentMessage,
      sessions: [],
      sessionsLoaded: false,
    });
    await act(async () => {
      await result.current.sendMessage("follow up");
    });

    expect(sendAgentMessage).toHaveBeenCalledTimes(1);
  });

  it("clears active session when starting a new session", async () => {
    const { result } = renderMobileSessions({
      sendAgentMessage: vi.fn(async (options: SendOptions) => {
        options.onSessionId("session-1");
        return { ok: true, reply: agentReply("session-1", "reply"), sessionId: "session-1" };
      }),
    });

    await act(async () => {
      await result.current.sendMessage("first");
      result.current.startNewSession();
    });

    expect(result.current.activeSessionId).toBeUndefined();
    expect(result.current.activeMessages).toEqual([]);
    expect(result.current.hasConversation).toBe(false);
  });

  it("allows concurrent runs in different sessions but blocks duplicate sends in the same session", async () => {
    saveStored([
      storedConversation("session-1", "One", [message("session-1", "user", "one")]),
      storedConversation("session-2", "Two", [message("session-2", "user", "two")]),
    ]);
    const pendingRuns: Array<() => void> = [];
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        pendingRuns.push(() => resolve({ ok: true, reply: agentReply(options.sessionId || "", "done"), sessionId: options.sessionId }));
      });
    });
    const { result } = renderMobileSessions({ sendAgentMessage });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    act(() => {
      void result.current.sendMessage("run one");
    });
    await waitFor(() => expect(result.current.canSend).toBe(false));

    let duplicateAccepted = true;
    await act(async () => {
      duplicateAccepted = await result.current.sendMessage("duplicate");
    });
    expect(duplicateAccepted).toBe(false);
    expect(sendAgentMessage).toHaveBeenCalledTimes(1);

    await act(async () => {
      await result.current.selectSession("session-2");
    });
    act(() => {
      void result.current.sendMessage("run two");
    });
    await waitFor(() => expect(sendAgentMessage).toHaveBeenCalledTimes(2));

    await act(async () => {
      pendingRuns.forEach((finish) => finish());
    });
  });

  it("marks background completion unread on the matching session only", async () => {
    saveStored([
      storedConversation("session-1", "One", [message("session-1", "user", "one")]),
      storedConversation("session-2", "Two", [message("session-2", "user", "two")]),
    ]);
    let finishRun: (() => void) | undefined;
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        finishRun = () => {
          options.onStatus({ tone: "success", text: "回复已返回" });
          resolve({ ok: true, reply: agentReply("session-1", "done"), sessionId: "session-1" });
        };
      });
    });
    const { result } = renderMobileSessions({ sendAgentMessage });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    act(() => {
      void result.current.sendMessage("run one");
    });
    await waitFor(() => expect(result.current.canSend).toBe(false));
    await act(async () => {
      await result.current.selectSession("session-2");
      finishRun?.();
    });
    await waitFor(() => expect(result.current.historyItems.find((item) => item.id === "session-1")?.unread).toBe(true));
    expect(result.current.historyItems.find((item) => item.id === "session-2")?.unread).toBe(false);
  });
});

describe("mobile session history projection", () => {
  it("uses Bridge sessions after list load, with Bridge titles overriding local titles", () => {
    const items = mergeHistoryItems({
      bridgeConnected: true,
      pinned: new Set(),
      sessionViews: {},
      sessions: [session("live", "Bridge title"), session("new", "New title")],
      sessionsLoaded: true,
      storedConversations: [
        storedConversation("live", "Local title", []),
        storedConversation("stale", "Stale", []),
      ],
    });

    expect(items.map((item) => item.id)).toEqual(["live", "new"]);
    expect(items[0]?.title).toBe("Bridge title");
  });

  it("prunes local cache entries missing from the Bridge session list", () => {
    const next = reconcileStoredConversationsWithBridge(
      [storedConversation("live", "Local title", []), storedConversation("stale", "Stale", [])],
      [session("live", "Bridge title")],
    );

    expect(next).toHaveLength(1);
    expect(next[0]).toMatchObject({ id: "live", title: "Bridge title" });
  });

  it("keeps local cached history while offline and exposes run status from session views", () => {
    const view: MobileSessionView = {
      bridgeOwned: false,
      id: "session-1",
      messages: [],
      run: { sessionEnded: false, status: "running", statusText: "运行中" },
      title: "Running",
      unread: false,
      updatedAt: "2026-01-01T00:00:00.000Z",
    };
    const items = mergeHistoryItems({
      bridgeConnected: false,
      pinned: new Set(["session-1"]),
      sessionViews: { "session-1": view },
      sessions: [],
      sessionsLoaded: false,
      storedConversations: [storedConversation("session-1", "Cached", [])],
    });

    expect(items[0]).toMatchObject({ id: "session-1", pinned: true, status: "running", title: "Running" });
  });

  it("hides local-only session views after Bridge session list has loaded", () => {
    const staleView: MobileSessionView = {
      bridgeOwned: false,
      id: "stale",
      messages: [message("stale", "user", "cached")],
      run: { sessionEnded: false, status: "success", statusText: "历史会话已加载" },
      title: "Stale local",
      unread: false,
      updatedAt: "2026-01-01T00:00:00.000Z",
    };

    const items = mergeHistoryItems({
      bridgeConnected: true,
      pinned: new Set(),
      sessionViews: { stale: staleView },
      sessions: [session("live", "Live")],
      sessionsLoaded: true,
      storedConversations: [storedConversation("stale", "Stale", [])],
    });

    expect(items.map((item) => item.id)).toEqual(["live"]);
  });
});

describe("settings session migration", () => {
  it("ignores legacy persisted sessionId on restart", async () => {
    const { loadSettings, saveSettings } = await import("../lib/settingsStorage");
    window.localStorage.setItem(
      "ghost-os-mobile.settings",
      JSON.stringify({ bridgeUrl: "http://localhost:9000", connectionMode: "http", sessionId: "old" }),
    );

    const settings = loadSettings();
    expect("sessionId" in settings).toBe(false);
    saveSettings(settings);
    expect(window.localStorage.getItem("ghost-os-mobile.settings")).not.toContain("sessionId");
  });
});

interface SendOptions {
  message: string;
  onReply: (reply: AgentPayload) => void;
  onSessionId: (sessionId: string) => void;
  onStatus: (status: { tone: "idle" | "loading" | "success" | "error"; text: string }) => void;
  sessionId?: string;
}

interface SendResult {
  ok: boolean;
  reply?: AgentPayload;
  sessionId?: string;
}

function renderMobileSessions(overrides: Partial<Parameters<typeof useMobileSessions>[0]> = {}) {
  const initialProps = {
    bridgeConnected: true,
    getSession: vi.fn(async (sessionId: string) => sessionDetail(sessionId)),
    pinnedHistoryIds: [],
    sendAgentMessage: vi.fn(async (options: SendOptions) => {
      options.onReply(agentReply(options.sessionId || "session-1", "reply"));
      return { ok: true, reply: agentReply(options.sessionId || "session-1", "reply"), sessionId: options.sessionId || "session-1" };
    }),
    sessions: [],
    sessionsLoaded: false,
    ...overrides,
  };
  return renderHook((props: Parameters<typeof useMobileSessions>[0]) => useMobileSessions(props), {
    initialProps,
  });
}

function agentReply(sessionId: string, text: string): AgentPayload {
  return {
    message: text,
    session_ended: false,
    session_id: sessionId,
  };
}

function message(sessionId: string, role: "user" | "assistant", text: string): MobileConversationMessage {
  return {
    id: `${sessionId}:${role}:${text}`,
    role,
    sessionId,
    text,
  };
}

function storedConversation(
  id: string,
  title: string,
  messages: MobileConversationMessage[],
): StoredMobileConversation {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id,
    messages,
    title,
    updated_at: "2026-01-01T00:00:00.000Z",
  };
}

function session(id: string, title: string): SessionMetadata {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id,
    message_count: 1,
    title,
    token_count: 1,
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}

function sessionDetail(id: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    messages: [
      {
        index: 0,
        role: "user",
        text: "loaded",
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
  };
}

function saveStored(conversations: StoredMobileConversation[]): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
}

function loadStored(): StoredMobileConversation[] {
  return JSON.parse(window.localStorage.getItem(STORAGE_KEY) || "[]") as StoredMobileConversation[];
}
