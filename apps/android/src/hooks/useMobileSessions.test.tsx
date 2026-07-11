// @vitest-environment jsdom
import { invoke } from "@tauri-apps/api/core";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useMobileSessions, mergeHistoryItems, reconcileStoredConversationsWithBridge } from "./useMobileSessions";
import {
  MOBILE_HOT_SESSION_VIEW_LIMIT,
  MOBILE_PERSISTED_CONVERSATION_LIMIT,
} from "../lib/mobileSessionLimits";
import { MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY } from "../lib/mobileSessionStorage";
import { buildAgentMessageWithSelectedSkill } from "../lib/selectedSkillMessage";
import type {
  AgentPayload,
  ChatSelectedSkill,
  MobileConversationMessage,
  MobileSessionView,
  SessionDetail,
  SessionMetadata,
  StoredMobileConversation,
} from "../mobileTypes";

const STORAGE_KEY = "ghost-os-mobile.conversations.v1";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

describe("useMobileSessions", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    Reflect.deleteProperty(window, "__TAURI_INTERNALS__");
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
      messageId: expect.stringContaining("pending:user:"),
      token: 1,
    });
    expect(result.current.historyItems[0]).toMatchObject({ id: "session-1", title: "first task" });
    expect(loadStored()[0]?.title).toBe("first task");
    expect(window.localStorage.getItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY)).toBe("session-1");
  });

  it("sends selected skill metadata and keeps the selected skill on the local user message", async () => {
    const selectedSkill = { id: "skill_release", name: "release_flow" } satisfies ChatSelectedSkill;
    const sendAgentMessage = vi.fn(async (options: SendOptions) => {
      expect(options.message).toBe("");
      expect(options.selectedSkill).toEqual(selectedSkill);
      options.onSessionId("session-1");
      options.onReply(agentReply("session-1", "reply"));
      return { ok: true, reply: agentReply("session-1", "reply"), sessionId: "session-1" };
    });
    const { result } = renderMobileSessions({ sendAgentMessage });

    await act(async () => {
      await result.current.sendMessage("", selectedSkill);
    });

    expect(sendAgentMessage).toHaveBeenCalledTimes(1);
    expect(result.current.activeMessages[0]).toMatchObject({
      role: "user",
      selectedSkill,
      text: "",
    });
    expect(loadStored()[0]?.messages[0]).toMatchObject({
      role: "user",
      selectedSkill,
      text: "",
    });
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
    expect(window.localStorage.getItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY)).toBe("session-1");

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
    expect(window.localStorage.getItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY)).toBe("session-1");
  });

  it("auto-selects the last active session after Bridge sessions load", async () => {
    window.localStorage.setItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY, "session-1");
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.activeMessages[0]?.text).toBe("loaded"));

    expect(getSession).toHaveBeenCalledWith("session-1");
    expect(result.current.activeMessages[0]).toMatchObject({
      role: "user",
      sessionId: "session-1",
      text: "loaded",
    });
  });

  it("waits for Bridge sessions to load before restoring the last active session", async () => {
    window.localStorage.setItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY, "session-1");
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result, rerender } = renderMobileSessions({
      getSession,
      sessions: [],
      sessionsLoaded: false,
    });

    expect(result.current.activeSessionId).toBeUndefined();
    expect(getSession).not.toHaveBeenCalled();

    rerender({
      bridgeConnected: true,
      getFullSession: vi.fn(async (sessionId: string) => sessionDetail(sessionId)),
      getSession,
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: false,
      sendAgentMessage: vi.fn(async () => ({ ok: false })),
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    });

    await waitFor(() => expect(result.current.activeSessionId).toBe("session-1"));
    expect(getSession).toHaveBeenCalledTimes(1);
  });

  it("clears the last active session when starting a new session", async () => {
    window.localStorage.setItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY, "session-1");
    const { result } = renderMobileSessions({
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.activeSessionId).toBe("session-1"));
    act(() => {
      result.current.startNewSession();
    });

    expect(result.current.activeSessionId).toBeUndefined();
    expect(window.localStorage.getItem(MOBILE_LAST_ACTIVE_SESSION_STORAGE_KEY)).toBeNull();
  });

  it("shows a loading state and waits for the latest Bridge page when selecting cached history", async () => {
    saveStored([storedConversation("session-1", "Cached title", [
      message("session-1", "user", "cached older"),
      message("session-1", "assistant", "cached latest"),
    ])]);
    let resolveDetail: ((detail: SessionDetail) => void) | undefined;
    const getSession = vi.fn((_sessionId: string) =>
      new Promise<SessionDetail>((resolve) => {
        resolveDetail = resolve;
      }),
    );
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    act(() => {
      void result.current.selectSession("session-1");
    });

    await waitFor(() => expect(result.current.loadingSessionMessages).toBe(true));
    expect(result.current.activeMessages).toEqual([]);

    await act(async () => {
      resolveDetail?.(sessionDetailWithMessages("session-1", [
        { index: 19, role: "user", text: "latest page only" },
      ], {
        hasMoreBefore: true,
        messageCount: 20,
        nextBefore: 19,
      }));
    });

    await waitFor(() => expect(result.current.loadingSessionMessages).toBe(false));
    expect(result.current.activeMessages.map((item) => item.text)).toEqual(["latest page only"]);
    expect(result.current.hasOlderHistory).toBe(true);
  });

  it("limits hot-loaded session detail views on mobile", async () => {
    const getSession = vi.fn(async (sessionId: string) => {
      const index = Number(sessionId.replace("session-", ""));
      return {
        ...sessionDetail(sessionId),
        updated_at: `2026-02-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`,
      };
    });
    const { result } = renderMobileSessions({
      getSession,
      sessions: [],
      sessionsLoaded: true,
    });

    for (let index = 0; index < MOBILE_HOT_SESSION_VIEW_LIMIT + 2; index += 1) {
      await act(async () => {
        await result.current.selectSession(`session-${index}`);
      });
    }

    const hotHistoryIds = result.current.historyItems.map((item) => item.id);
    expect(hotHistoryIds).toHaveLength(MOBILE_HOT_SESSION_VIEW_LIMIT);
    expect(hotHistoryIds).toContain(`session-${MOBILE_HOT_SESSION_VIEW_LIMIT + 1}`);
    expect(hotHistoryIds).not.toContain("session-0");
  });

  it("loads older Bridge session history pages before current messages", async () => {
    const getSession = vi.fn(async (sessionId: string, options?: { before?: number }) => {
      if (options?.before === 1) {
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "older" },
        ], {
          hasMoreBefore: false,
          messageCount: 2,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 1, role: "user", text: "latest" },
      ], {
        hasMoreBefore: true,
        messageCount: 2,
        nextBefore: 1,
      });
    });
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(result.current.activeMessages.map((message) => message.text)).toEqual(["latest"]);
    expect(result.current.hasOlderHistory).toBe(true);

    await act(async () => {
      await result.current.loadOlderHistory();
    });

    expect(getSession).toHaveBeenLastCalledWith("session-1", { before: 1 });
    expect(result.current.activeMessages.map((message) => message.text)).toEqual(["older", "latest"]);
    expect(result.current.hasOlderHistory).toBe(false);
    expect(result.current.loadingOlderHistory).toBe(false);
    expect(loadStored()[0]).toMatchObject({
      messages: [
        expect.objectContaining({ text: "older" }),
        expect.objectContaining({ text: "latest" }),
      ],
      source_message_count: 2,
      synced_message_count: 2,
    });
  });

  it("notifies runtime selection after loading Bridge session detail", async () => {
    const detail: SessionDetail = {
      ...sessionDetail("session-1"),
      last_runtime_selection: {
        runtime: "ghost",
        provider: "openai-main",
        provider_type: "openai",
        model: "gpt-5.4",
        mode: "plan",
      },
    };
    const getSession = vi.fn(async () => detail);
    const onSessionRuntimeSelection = vi.fn();
    const { result } = renderMobileSessions({
      getSession,
      onSessionRuntimeSelection,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(onSessionRuntimeSelection).toHaveBeenCalledWith(detail.last_runtime_selection);
  });

  it("hydrates a web-started Codex turn draft as running and polls until it completes", async () => {
    let getSessionCalls = 0;
    let resolvePoll: ((detail: SessionDetail) => void) | undefined;
    const getSession = vi.fn((sessionId: string) => {
      getSessionCalls += 1;
      if (getSessionCalls === 1) {
        return Promise.resolve(codexDraftSessionDetail(sessionId));
      }
      return new Promise<SessionDetail>((resolve) => {
        resolvePoll = resolve;
      });
    });
    const onSessionRuntimeSelection = vi.fn();
    const { result } = renderMobileSessions({
      getSession,
      onSessionRuntimeSelection,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(result.current.activeStatus).toEqual({ tone: "loading", text: "等待用户输入" });
    expect(result.current.canSend).toBe(false);
    expect(result.current.historyItems.find((item) => item.id === "session-1")?.status).toBe("running");
    expect(result.current.activeReply).toMatchObject({
      message: "partial answer",
      parts: [
        { kind: "text", text: "partial answer" },
        {
          kind: "tool",
          tool: expect.objectContaining({
            approvalId: "approval-1",
            status: "pending",
            toolName: "codex_approval",
          }),
        },
      ],
      thinking: "thinking",
    });
    expect(onSessionRuntimeSelection).toHaveBeenCalledWith(
      expect.objectContaining({ runtime: "codex", model: "gpt-5-codex" }),
    );

    await waitFor(() => expect(getSession).toHaveBeenCalledTimes(2));
    await act(async () => {
      resolvePoll?.(completedCodexSessionDetail("session-1"));
    });

    await waitFor(() => expect(result.current.activeStatus).toEqual({ tone: "success", text: "回复已返回" }));
    expect(result.current.activeMessages.map((item) => item.text)).toEqual(["run codex", "done from bridge"]);
    expect(result.current.activeReply).toBeUndefined();
  });

  it("reloads an active Bridge-owned session when the Bridge session list version advances", async () => {
    let getSessionCalls = 0;
    let resolveExternalPoll: ((detail: SessionDetail) => void) | undefined;
    const getSession = vi.fn((sessionId: string) => {
      getSessionCalls += 1;
      if (getSessionCalls === 1) {
        return Promise.resolve(sessionDetail(sessionId));
      }
      if (getSessionCalls === 2) {
        return Promise.resolve(codexDraftSessionDetail(sessionId));
      }
      return new Promise<SessionDetail>((resolve) => {
        resolveExternalPoll = resolve;
      });
    });
    const baseProps: UseMobileSessionsOptionsForTest = {
      bridgeConnected: true,
      getFullSession: vi.fn(async (sessionId: string) => sessionDetail(sessionId)),
      getSession,
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: false,
      sendAgentMessage: vi.fn(async () => ({ ok: false })),
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    };
    const { result, rerender } = renderHook((props: UseMobileSessionsOptionsForTest) => useMobileSessions(props), {
      initialProps: baseProps,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    expect(result.current.activeStatus).toEqual({ tone: "success", text: "历史会话已加载" });

    rerender({
      ...baseProps,
      sessions: [sessionWithUpdatedAt("session-1", "Bridge title", "2026-01-03T00:00:00.000Z")],
    });

    await waitFor(() => expect(result.current.activeStatus).toEqual({ tone: "loading", text: "等待用户输入" }));
    await waitFor(() => expect(getSession).toHaveBeenCalledTimes(3));

    await act(async () => {
      resolveExternalPoll?.(completedCodexSessionDetail("session-1"));
    });
  });

  it("restores selected skill metadata from wrapped Bridge user messages", async () => {
    const getSession = vi.fn(async (sessionId: string) => selectedSkillSessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getSession,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });

    expect(result.current.activeMessages[0]).toMatchObject({
      role: "user",
      selectedSkill: {
        id: "skill_release",
        name: "release_flow",
      },
      text: "发布版本",
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

  it("preserves a background running session stream while reloading its Bridge snapshot", async () => {
    saveStored([
      storedConversation("session-1", "One", [message("session-1", "user", "one")]),
      storedConversation("session-2", "Two", [message("session-2", "user", "two")]),
    ]);
    let session1Loads = 0;
    let resolveSession1Reload: ((detail: SessionDetail) => void) | undefined;
    let finishRun: (() => void) | undefined;
    const getSession = vi.fn(async (sessionId: string) => {
      if (sessionId === "session-1") {
        session1Loads += 1;
        if (session1Loads > 1) {
          return new Promise<SessionDetail>((resolve) => {
            resolveSession1Reload = resolve;
          });
        }
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "one" },
        ], {
          hasMoreBefore: false,
          messageCount: 1,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 0, role: "user", text: "two" },
      ], {
        hasMoreBefore: false,
        messageCount: 1,
        nextBefore: null,
      });
    });
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        finishRun = () => {
          options.onReply(agentReply("session-1", "done"));
          options.onStatus({ tone: "success", text: "回复已返回" });
          resolve({ mode: "remote", ok: true, reply: agentReply("session-1", "done"), sessionId: "session-1" });
        };
      });
    });
    const { result } = renderMobileSessions({ getSession, sendAgentMessage });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    act(() => {
      void result.current.sendMessage("run one");
    });
    await waitFor(() => expect(result.current.canSend).toBe(false));

    await act(async () => {
      await result.current.selectSession("session-2");
    });
    act(() => {
      const streamOptions = sendAgentMessage.mock.calls[0]?.[0];
      streamOptions?.onReply(agentReply("session-1", "partial"));
    });
    act(() => {
      void result.current.selectSession("session-1");
    });

    await waitFor(() => expect(result.current.activeSessionId).toBe("session-1"));
    expect(result.current.activeReply?.message).toBe("partial");
    expect(result.current.activeStatus).toEqual({ tone: "loading", text: "运行中" });
    expect(result.current.activeMessages.map((item) => item.text)).toEqual(["one", "run one"]);
    expect(getSession.mock.calls.filter(([sessionId]) => sessionId === "session-1")).toHaveLength(2);

    await act(async () => {
      resolveSession1Reload?.(runningDraftSessionDetail("session-1", "partial from bridge"));
    });
    await waitFor(() => expect(result.current.activeReply?.message).toBe("partial from bridge"));

    await act(async () => {
      finishRun?.();
    });
  });

  it("polls a mobile-started running session after Bridge reconnects", async () => {
    saveStored([
      storedConversation("session-1", "One", [message("session-1", "user", "one")]),
    ]);
    let getSessionCalls = 0;
    let finishStream: ((result: SendResult) => void) | undefined;
    const getSession = vi.fn(async (sessionId: string) => {
      getSessionCalls += 1;
      if (getSessionCalls === 1) {
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "one" },
        ], {
          hasMoreBefore: false,
          messageCount: 1,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 0, role: "user", text: "one" },
        { index: 1, role: "user", text: "run one" },
        { index: 2, role: "assistant", text: "done after reconnect" },
      ], {
        hasMoreBefore: false,
        messageCount: 3,
        nextBefore: null,
      });
    });
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        finishStream = resolve;
      });
    });
    const baseProps: UseMobileSessionsOptionsForTest = {
      bridgeConnected: true,
      getFullSession: vi.fn(async (sessionId: string) => sessionDetail(sessionId)),
      getSession,
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: false,
      sendAgentMessage,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    };
    const { result, rerender } = renderHook((props: UseMobileSessionsOptionsForTest) => useMobileSessions(props), {
      initialProps: baseProps,
    });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    act(() => {
      void result.current.sendMessage("run one");
    });
    await waitFor(() => expect(result.current.activeStatus).toEqual({ tone: "loading", text: "运行中" }));

    await act(async () => {
      rerender({
        ...baseProps,
        bridgeConnected: false,
        sessions: [],
        sessionsLoaded: false,
      });
      await Promise.resolve();
    });
    await act(async () => {
      rerender(baseProps);
      await Promise.resolve();
    });

    await waitFor(() => expect(getSession).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(result.current.activeStatus).toEqual({ tone: "success", text: "回复已返回" }));
    expect(result.current.activeMessages.map((item) => item.text)).toEqual([
      "one",
      "run one",
      "done after reconnect",
    ]);

    await act(async () => {
      finishStream?.({ ok: false });
    });
  });

  it("polls a running session after its HTTP stream is interrupted", async () => {
    let getSessionCalls = 0;
    const getSession = vi.fn(async (sessionId: string) => {
      getSessionCalls += 1;
      if (getSessionCalls === 1) {
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "one" },
        ], {
          hasMoreBefore: false,
          messageCount: 1,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 0, role: "user", text: "one" },
        { index: 1, role: "user", text: "run one" },
        { index: 2, role: "assistant", text: "done after stream interruption" },
      ], {
        hasMoreBefore: false,
        messageCount: 3,
        nextBefore: null,
      });
    });
    const sendAgentMessage = vi.fn(async (options: SendOptions): Promise<SendResult> => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return {
        ok: false,
        sessionId: options.sessionId,
        streamInterrupted: true,
      };
    });
    const { result } = renderMobileSessions({ getSession, sendAgentMessage });

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    await act(async () => {
      await result.current.sendMessage("run one");
    });

    await waitFor(() => expect(getSession).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(result.current.activeStatus).toEqual({ tone: "success", text: "回复已返回" }));
    expect(result.current.activeMessages.map((item) => item.text)).toEqual([
      "one",
      "run one",
      "done after stream interruption",
    ]);
  });

  it("ignores inactive session completion until the user selects it again", async () => {
    saveStored([
      storedConversation("session-1", "One", [message("session-1", "user", "one")]),
      storedConversation("session-2", "Two", [message("session-2", "user", "two")]),
    ]);
    let completed = false;
    let finishRun: (() => void) | undefined;
    const getSession = vi.fn(async (sessionId: string) => {
      if (sessionId === "session-1" && completed) {
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "one" },
          { index: 1, role: "user", text: "run one" },
          { index: 2, role: "assistant", text: "done from bridge" },
        ], {
          hasMoreBefore: false,
          messageCount: 3,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 0, role: "user", text: sessionId === "session-1" ? "one" : "two" },
      ], {
        hasMoreBefore: false,
        messageCount: 1,
        nextBefore: null,
      });
    });
    const sendAgentMessage = vi.fn((options: SendOptions) => {
      options.onStatus({ tone: "loading", text: "运行中" });
      return new Promise<SendResult>((resolve) => {
        finishRun = () => {
          completed = true;
          options.onStatus({ tone: "success", text: "回复已返回" });
          resolve({ mode: "remote", ok: true, reply: agentReply("session-1", "done"), sessionId: "session-1" });
        };
      });
    });
    const { result } = renderMobileSessions({ getSession, sendAgentMessage });

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
    expect(result.current.historyItems.find((item) => item.id === "session-1")?.unread).toBe(false);
    expect(result.current.historyItems.find((item) => item.id === "session-2")?.unread).toBe(false);
    expect(getSession.mock.calls.filter(([sessionId]) => sessionId === "session-1")).toHaveLength(1);

    await act(async () => {
      await result.current.selectSession("session-1");
    });
    expect(getSession.mock.calls.filter(([sessionId]) => sessionId === "session-1")).toHaveLength(2);
    expect(result.current.activeMessages.map((item) => item.text)).toEqual(["one", "run one", "done from bridge"]);
  });

  it("keeps tracking a newly resolved session when the user switches before session id arrives", async () => {
    saveStored([
      storedConversation("session-2", "Two", [message("session-2", "user", "two")]),
    ]);
    let completed = false;
    let emitSessionId: (() => void) | undefined;
    let finishRun: (() => void) | undefined;
    const getSession = vi.fn(async (sessionId: string) => {
      if (sessionId === "session-new" && completed) {
        return sessionDetailWithMessages(sessionId, [
          { index: 0, role: "user", text: "new run" },
          { index: 1, role: "assistant", text: "done from bridge" },
        ], {
          hasMoreBefore: false,
          messageCount: 2,
          nextBefore: null,
        });
      }
      return sessionDetailWithMessages(sessionId, [
        { index: 0, role: "user", text: sessionId === "session-2" ? "two" : "new run" },
      ], {
        hasMoreBefore: false,
        messageCount: 1,
        nextBefore: null,
      });
    });
    const sendAgentMessage = vi.fn((options: SendOptions) =>
      new Promise<SendResult>((resolve) => {
        emitSessionId = () => {
          options.onSessionId("session-new");
        };
        finishRun = () => {
          completed = true;
          options.onStatus({ tone: "success", text: "回复已返回" });
          resolve({ mode: "remote", ok: true, reply: agentReply("session-new", "done"), sessionId: "session-new" });
        };
      }));
    const { result } = renderMobileSessions({ getSession, sendAgentMessage });

    act(() => {
      void result.current.sendMessage("new run");
    });
    await waitFor(() => expect(result.current.canSend).toBe(false));
    await act(async () => {
      await result.current.selectSession("session-2");
    });
    expect(result.current.activeSessionId).toBe("session-2");

    act(() => {
      emitSessionId?.();
    });
    expect(result.current.activeSessionId).toBe("session-2");
    expect(result.current.historyItems.find((item) => item.id === "session-new")?.status).toBe("running");
    expect(result.current.liveSessionRuns).toEqual([
      expect.objectContaining({
        sessionId: "session-new",
        status: "running",
      }),
    ]);

    await act(async () => {
      finishRun?.();
    });
    expect(result.current.activeSessionId).toBe("session-2");
    expect(result.current.historyItems.find((item) => item.id === "session-new")?.status).toBe("success");
    expect(result.current.liveSessionRuns).toEqual([
      expect.objectContaining({
        sessionId: "session-new",
        status: "success",
      }),
    ]);
    expect(getSession.mock.calls.filter(([sessionId]) => sessionId === "session-new")).toHaveLength(0);

    await act(async () => {
      await result.current.selectSession("session-new");
    });
    expect(getSession.mock.calls.filter(([sessionId]) => sessionId === "session-new")).toHaveLength(1);
    expect(result.current.activeMessages.map((item) => item.text)).toEqual(["new run", "done from bridge"]);
  });

  it("does not sync all Bridge sessions when computer session persistence is disabled", async () => {
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      getSession,
      persistComputerSessionsEnabled: false,
      sessions: [session("session-1", "Bridge title")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("未开启"));

    expect(getFullSession).not.toHaveBeenCalled();
    expect(getSession).not.toHaveBeenCalled();
  });

  it("syncs complete Bridge sessions when computer session persistence is enabled", async () => {
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      getSession,
      persistComputerSessionsEnabled: true,
      sessions: [session("session-1", "One"), session("session-2", "Two")],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 2 个"));

    expect(getFullSession).toHaveBeenCalledTimes(2);
    expect(getSession).not.toHaveBeenCalled();
    expect(loadStored().map((conversation) => conversation.id).sort()).toEqual(["session-1", "session-2"]);
    expect(loadStored()[0]).toMatchObject({ source_message_count: 1 });
  });

  it("persists computer sessions in batches and reloads compacted Tauri history on demand", async () => {
    Reflect.set(window, "__TAURI_INTERNALS__", {});
    const persistedById = new Map<string, StoredMobileConversation>();
    const upsertBatches: string[][] = [];
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command === "mobile_conversations_load_index") {
        return [];
      }
      if (command === "mobile_conversations_upsert") {
        const conversations = tauriConversationsArg(args);
        upsertBatches.push(conversations.map((conversation) => conversation.id));
        for (const conversation of conversations) {
          persistedById.set(conversation.id, conversation);
        }
        return [...persistedById.values()].map((conversation) => ({ ...conversation, messages: [] }));
      }
      if (command === "mobile_conversation_get") {
        return persistedById.get(tauriSessionIdArg(args));
      }
      return undefined;
    });

    const sessions = Array.from({ length: 7 }, (_, index) =>
      sessionWithUpdatedAt(`session-${index}`, `Session ${index}`, `2026-01-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`),
    );
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const baseProps: UseMobileSessionsOptionsForTest = {
      bridgeConnected: true,
      getFullSession,
      getSession,
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: true,
      sendAgentMessage: vi.fn(async () => ({ ok: false })),
      sessions,
      sessionsLoaded: true,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    };
    const { result, rerender } = renderHook((props: UseMobileSessionsOptionsForTest) => useMobileSessions(props), {
      initialProps: baseProps,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 7 个"));

    expect(upsertBatches.map((batch) => batch.length)).toEqual([3, 3, 1]);
    expect(getFullSession).toHaveBeenCalledTimes(7);
    expect(getSession).not.toHaveBeenCalled();
    expect([...persistedById.values()].every((conversation) => conversation.messages.length > 0)).toBe(true);

    rerender({
      ...baseProps,
      bridgeConnected: false,
      persistComputerSessionsEnabled: false,
      sessions: [],
      sessionsLoaded: false,
    });
    await act(async () => {
      await result.current.selectSession("session-6");
    });

    expect(invoke).toHaveBeenCalledWith("mobile_conversation_get", { sessionId: "session-6" });
    expect(result.current.activeMessages.map((message) => message.text)).toEqual(["loaded"]);
  });

  it("does not restart computer session persistence when refreshed sessions keep the same content", async () => {
    const pendingSessionLoads: Array<{
      resolve: (detail: SessionDetail) => void;
      sessionId: string;
    }> = [];
    const sessions = Array.from({ length: 4 }, (_, index) =>
      sessionWithUpdatedAt(`session-${index}`, `Session ${index}`, `2026-01-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`),
    );
    const getFullSession = vi.fn((sessionId: string) =>
      new Promise<SessionDetail>((resolve) => {
        pendingSessionLoads.push({ resolve, sessionId });
      }),
    );
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const baseProps: UseMobileSessionsOptionsForTest = {
      bridgeConnected: true,
      getFullSession,
      getSession,
      pinnedHistoryIds: [],
      persistComputerSessionsEnabled: true,
      sendAgentMessage: vi.fn(async () => ({ ok: false })),
      sessions,
      sessionsLoaded: true,
      stopAgentRun: vi.fn(async () => ({ ok: true, status: "stopped" as const })),
    };
    const { result, rerender } = renderHook((props: UseMobileSessionsOptionsForTest) => useMobileSessions(props), {
      initialProps: baseProps,
    });

    await waitFor(() => expect(getFullSession).toHaveBeenCalledTimes(1));

    rerender({
      ...baseProps,
      sessions: sessions.map((item) => ({ ...item })),
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(getFullSession).toHaveBeenCalledTimes(1);

    for (const [index] of sessions.entries()) {
      const pending = pendingSessionLoads[index];
      expect(pending).toBeDefined();
      await act(async () => {
        pending.resolve(sessionDetail(pending.sessionId));
        await Promise.resolve();
      });
      if (index < sessions.length - 1) {
        await waitFor(() => expect(getFullSession).toHaveBeenCalledTimes(index + 2));
      }
    }

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 4 个"));
    expect(getFullSession).toHaveBeenCalledTimes(4);
    expect(getSession).not.toHaveBeenCalled();
  });

  it("limits computer session persistence to the most recent Bridge sessions", async () => {
    const sessions = Array.from({ length: MOBILE_PERSISTED_CONVERSATION_LIMIT + 2 }, (_, index) =>
      sessionWithUpdatedAt(`session-${index}`, `Session ${index}`, `2026-01-${String(index + 1).padStart(2, "0")}T00:00:00.000Z`),
    );
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      getSession,
      persistComputerSessionsEnabled: true,
      sessions,
      sessionsLoaded: true,
    });

    await waitFor(() =>
      expect(result.current.computerSessionPersistStatus.text).toBe(
        `已同步 ${MOBILE_PERSISTED_CONVERSATION_LIMIT}/${sessions.length} 个最近会话`,
      ),
    );

    expect(getFullSession).toHaveBeenCalledTimes(MOBILE_PERSISTED_CONVERSATION_LIMIT);
    expect(getSession).not.toHaveBeenCalled();
    expect(loadStored()).toHaveLength(MOBILE_PERSISTED_CONVERSATION_LIMIT);
    expect(loadStored().map((conversation) => conversation.id)).not.toContain("session-0");
    expect(loadStored().map((conversation) => conversation.id)).not.toContain("session-1");
  });

  it("skips current Bridge sessions even when projection reduces the stored message count", async () => {
    saveStored([
      {
        ...storedConversation("session-1", "Cached", [
          message("session-1", "user", "run pwd"),
          message("session-1", "assistant", "done"),
        ]),
        source_snapshot_complete: true,
        source_message_count: 3,
        synced_message_count: 2,
        updated_at: "2026-01-02T00:00:00.000Z",
      },
    ]);
    const getFullSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const getSession = vi.fn(async (sessionId: string) => sessionDetail(sessionId));
    const { result } = renderMobileSessions({
      getFullSession,
      getSession,
      persistComputerSessionsEnabled: true,
      sessions: [{ ...session("session-1", "Bridge title"), message_count: 3 }],
      sessionsLoaded: true,
    });

    await waitFor(() => expect(result.current.computerSessionPersistStatus.text).toBe("已同步 1 个"));

    expect(getFullSession).not.toHaveBeenCalled();
    expect(getSession).not.toHaveBeenCalled();
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
    expect(getFullSession).toHaveBeenCalledTimes(2);
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
  selectedSkill?: ChatSelectedSkill;
  sessionId?: string;
  shouldStreamRealtime?: (sessionId: string) => boolean;
  traceId?: string;
}

interface SendResult {
  mode?: "local" | "remote";
  ok: boolean;
  reply?: AgentPayload;
  sessionId?: string;
  streamInterrupted?: boolean;
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
    onSessionRuntimeSelection: overrides.onSessionRuntimeSelection,
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

function sessionWithUpdatedAt(id: string, title: string, updatedAt: string): SessionMetadata {
  return {
    ...session(id, title),
    updated_at: updatedAt,
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

function sessionDetailWithMessages(
  id: string,
  messages: SessionDetail["messages"],
  options: {
    hasMoreBefore: boolean;
    messageCount: number;
    nextBefore: number | null;
  },
): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    message_count: options.messageCount,
    messages,
    page: {
      has_more_before: options.hasMoreBefore,
      limit: 100,
      next_before: options.nextBefore,
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

function selectedSkillSessionDetail(id: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    messages: [
      {
        index: 0,
        role: "user",
        text: buildAgentMessageWithSelectedSkill({
          message: "发布版本",
          selectedSkill: {
            id: "skill_release",
            name: "release_flow",
          },
        }),
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
  };
}

function codexDraftSessionDetail(id: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    last_runtime_selection: {
      runtime: "codex",
      provider: "codex",
      provider_type: "codex",
      model: "gpt-5-codex",
      mode: "default",
    },
    messages: [
      {
        index: 0,
        role: "user",
        text: "run codex",
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
    turn_draft: {
      trace_id: "trace-codex",
      turn: 2,
      status: "awaiting_human",
      pending_questions: [
        {
          question_id: "approval-1",
          prompt: "Approve command execution",
          selection_mode: "single",
        },
      ],
      assistant_segments: [
        {
          id: "stream-segment:assistant:1",
          content: "partial answer",
        },
      ],
      thinking_segments: [
        {
          id: "stream-segment:thinking:1",
          content: "thinking",
        },
      ],
      tools: [],
      item_order: [
        "assistant:stream-segment:assistant:1",
        "question:approval-1",
      ],
    },
  };
}

function runningDraftSessionDetail(id: string, partial: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    message_count: 2,
    messages: [
      {
        index: 0,
        role: "user",
        text: "one",
      },
      {
        index: 1,
        role: "user",
        text: "run one",
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
    turn_draft: {
      trace_id: "trace-running",
      turn: 2,
      status: "streaming",
      pending_questions: [],
      assistant_segments: [
        {
          id: "stream-segment:assistant:1",
          content: partial,
        },
      ],
      thinking_segments: [],
      tools: [],
      item_order: [
        "assistant:stream-segment:assistant:1",
      ],
    },
  };
}

function completedCodexSessionDetail(id: string): SessionDetail {
  return {
    ...session(id, `Bridge ${id}`),
    last_runtime_selection: {
      runtime: "codex",
      provider: "codex",
      provider_type: "codex",
      model: "gpt-5-codex",
      mode: "default",
    },
    message_count: 2,
    messages: [
      {
        index: 0,
        role: "user",
        text: "run codex",
      },
      {
        index: 1,
        role: "assistant",
        text: "done from bridge",
      },
    ],
    page: {
      has_more_before: false,
      limit: 100,
    },
    turn_draft: null,
  };
}

function saveStored(conversations: StoredMobileConversation[]): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
}

function loadStored(): StoredMobileConversation[] {
  return JSON.parse(window.localStorage.getItem(STORAGE_KEY) || "[]") as StoredMobileConversation[];
}

function tauriConversationsArg(args: unknown): StoredMobileConversation[] {
  if (!args || typeof args !== "object" || Array.isArray(args)) {
    return [];
  }
  const conversations = (args as { conversations?: unknown }).conversations;
  return Array.isArray(conversations) ? conversations as StoredMobileConversation[] : [];
}

function tauriSessionIdArg(args: unknown): string {
  if (!args || typeof args !== "object" || Array.isArray(args)) {
    return "";
  }
  return String((args as { sessionId?: unknown }).sessionId ?? "");
}
