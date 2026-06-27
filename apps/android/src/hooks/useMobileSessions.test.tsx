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
    expect(result.current.postSendFocusRequest).toMatchObject({
      messageId: result.current.activeMessages[0]?.id,
      token: 1,
    });
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
      getFullSession: vi.fn(),
      getSession: vi.fn(),
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: false,
      sendAgentMessage,
      sessions: [],
      sessionsLoaded: false,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    });
    await act(async () => {
      await result.current.sendMessage("follow up");
    });

    expect(sendAgentMessage).toHaveBeenCalledTimes(1);
  });

  it("loads Bridge session detail and persists projected messages", async () => {
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(getSession).toHaveBeenCalledWith("session-1");
    expect(result.current.activeMessages).toEqual([
      expect.objectContaining({ role: "user", sessionId: "session-1", text: "loaded" }),
    ]);
    expect(loadStored()[0]).toMatchObject({
      id: "session-1",
      messages: [expect.objectContaining({ role: "user", sessionId: "session-1", text: "loaded" })],
      title: "Bridge title",
    });
  });

  it("restores assistant tool calls and tool results from Bridge session detail", async () => {
    const getSession = vi.fn(async (sessionId: string) => toolSessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(result.current.activeMessages).toEqual([
      expect.objectContaining({ role: "user", text: "run pwd" }),
      expect.objectContaining({
        role: "assistant",
        text: "我来查看。",
        tools: [
          {
            id: "session-1:1:tool-call:call-1",
            input: "{\n  \"cmd\": \"pwd\"\n}",
            output: "/repo",
            status: "success",
            toolCallId: "call-1",
            toolName: "bash_exec",
            traceId: "trace-1",
          },
        ],
      }),
    ]);
    expect(loadStored()[0]?.messages[1]).toMatchObject({
      role: "assistant",
      tools: [expect.objectContaining({ output: "/repo", status: "success", toolCallId: "call-1" })],
    });
  });

  it("persists final reply tools into local cache", async () => {
    const sendAgentMessage = vi.fn(async (options: SendOptions) => {
      const reply = agentReply("session-1", "done", [
        {
          id: "stream-tool:trace-1:call-1",
          input: "{\"cmd\":\"pwd\"}",
          output: "/repo",
          status: "success",
          toolCallId: "call-1",
          toolName: "bash_exec",
          traceId: "trace-1",
        },
      ]);
      options.onSessionId("session-1");
      options.onReply(reply);
      return { ok: true, reply, sessionId: "session-1" };
    });
    const { result } = renderMobileSessions({ sendAgentMessage });

    await act(async () => {
      await result.current.sendMessage("pwd");
    });

    expect(result.current.activeMessages[result.current.activeMessages.length - 1]).toMatchObject({
      role: "assistant",
      tools: [expect.objectContaining({ output: "/repo", status: "success", toolCallId: "call-1" })],
    });
    const storedMessages = loadStored()[0]?.messages ?? [];
    expect(storedMessages[storedMessages.length - 1]).toMatchObject({
      role: "assistant",
      tools: [expect.objectContaining({ output: "/repo", status: "success", toolCallId: "call-1" })],
    });
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
    expect(result.current.postSendFocusRequest).toBeNull();
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

  it("stops the active run with the run trace id and syncs the stopped session", async () => {
    let finishRun: (() => void) | undefined;
    let runTraceId = "";
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      runTraceId = options.traceId ?? "";
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        finishRun = () => {
          options.onStatus({
            tone: "error",
            text: "trace_id=agent-run-f35b5652-da4f-4f78-833c-a77154adba62 turn=7 complete_once: stream interrupted: context canceled",
          });
          resolve({ ok: false });
        };
      });
    });
    const stopAgentRun = vi.fn(async () => ({ ok: true, sessionId: "session-1", status: "stopped" as const }));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({ getSession, sendAgentMessage, stopAgentRun });

    act(() => {
      void result.current.sendMessage("run one");
    });
    await waitFor(() => expect(result.current.canStop).toBe(true));

    await act(async () => {
      await result.current.stopCurrentRun();
    });

    expect(stopAgentRun).toHaveBeenCalledWith({ sessionId: undefined, traceId: runTraceId });
    expect(getSession).toHaveBeenCalledWith("session-1");
    expect(result.current.canStop).toBe(false);
    expect(result.current.activeStatus).toEqual({ tone: "success", text: "已停止" });

    await act(async () => {
      finishRun?.();
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

  it("does not sync all Bridge sessions when computer session persistence is disabled", async () => {
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      persistComputerSessionsEnabled: false,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("未开启"));

    expect(getFullSession).not.toHaveBeenCalled();
  });

  it("syncs all Bridge sessions when computer session persistence is enabled", async () => {
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      persistComputerSessionsEnabled: true,
      sessions: [session("session-1", "One"), session("session-2", "Two")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 2 个"));

    expect(getFullSession).toHaveBeenCalledTimes(2);
    expect(loadStored().map((conversation) => conversation.id).sort()).toEqual(["session-1", "session-2"]);
    expect(loadStored()[0]).toMatchObject({ source_message_count: 1 });
  });

  it("skips fully synced Bridge sessions by updated_at and source_message_count", async () => {
    saveStored([
      {
        ...storedConversation("session-1", "Cached", [message("session-1", "user", "loaded")]),
        source_message_count: 1,
        updated_at: "2026-01-02T00:00:00.000Z",
      },
    ]);
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      persistComputerSessionsEnabled: true,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 1 个"));

    expect(getFullSession).not.toHaveBeenCalled();
  });

  it("reports sync failures without saving partial success", async () => {
    const getFullSession = vi.fn(async (sessionId: string) => {
      if (sessionId === "session-2") {
        throw new Error("SESSION_GET failed");
      }
      return sessionDetail(sessionId);
    });
    const { result } = renderMobileSessions({
      getFullSession,
      persistComputerSessionsEnabled: true,
      sessions: [session("session-1", "One"), session("session-2", "Two")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("同步失败：SESSION_GET failed"));

    expect(loadStored()).toEqual([]);
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
  requestId?: string;
  sessionId?: string;
  traceId?: string;
}

interface SendResult {
  ok: boolean;
  reply?: AgentPayload;
  sessionId?: string;
}

type UseMobileSessionsOptionsForTest = Parameters<typeof useMobileSessions>[0];

function renderMobileSessions(overrides: Partial<UseMobileSessionsOptionsForTest> = {}) {
  const defaultGetSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
  const defaultSendAgentMessage = vi.fn(async (options: SendOptions) => {
    options.onReply(agentReply(options.sessionId || "session-1", "reply"));
    return { ok: true, reply: agentReply(options.sessionId || "session-1", "reply"), sessionId: options.sessionId || "session-1" };
  });
  const defaultStopAgentRun = vi.fn(async () => ({ ok: true, status: "stopped" as const }));
  const initialProps: UseMobileSessionsOptionsForTest = {
    bridgeConnected: overrides.bridgeConnected ?? true,
    computerSessionSyncScope: overrides.computerSessionSyncScope,
    getFullSession: overrides.getFullSession ?? defaultGetSession,
    getSession: overrides.getSession ?? defaultGetSession,
    pinnedHistoryIds: overrides.pinnedHistoryIds ?? [],
    persistComputerSessionsEnabled: overrides.persistComputerSessionsEnabled ?? false,
    sendAgentMessage: overrides.sendAgentMessage ?? defaultSendAgentMessage,
    sessions: overrides.sessions ?? [],
    sessionsLoaded: overrides.sessionsLoaded ?? false,
    stopAgentRun: overrides.stopAgentRun ?? defaultStopAgentRun,
  };
  return renderHook((props: UseMobileSessionsOptionsForTest) => useMobileSessions(props), {
    initialProps,
  });
}

function agentReply(sessionId: string, text: string, tools?: AgentPayload["tools"]): AgentPayload {
  return {
    message: text,
    session_ended: false,
    session_id: sessionId,
    tools,
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

function toolSessionDetail(id: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    messages: [
      {
        index: 0,
        role: "user",
        text: "run pwd",
      },
      {
        index: 1,
        role: "assistant",
        text: "我来查看。",
        tool_calls: [
          {
            arguments: { cmd: "pwd" },
            id: "call-1",
            name: "bash_exec",
          },
        ],
      },
      {
        index: 2,
        role: "tool",
        text: "/repo",
        tool_call_id: "call-1",
        tool_result: {
          output: "/repo",
          status: "success",
          tool: "bash_exec",
          trace_id: "trace-1",
        },
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
