// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

interface MockSendAgentMessageOptions {
  history: unknown[];
  message: string;
  onReply: (reply: unknown) => void;
  onSessionId: (sessionId: string) => void;
  onStatus: (status: unknown) => void;
  selectedSkill?: unknown;
}

interface MockMobileSessionsOptions {
  sendAgentMessage: (options: MockSendAgentMessageOptions) => Promise<{ ok: boolean }>;
}

interface MockHistoryItem {
  id: string;
  pinned: boolean;
  status?: "running" | "success" | "error";
  title: string;
  updatedAt: string;
}

const mocks = vi.hoisted(() => ({
  bridgeSendAgentMessage: vi.fn(),
  bridgeStopAgentRun: vi.fn(),
  useMobileSessions: vi.fn((options: MockMobileSessionsOptions) => ({
    activeMessages: [],
    activeReply: undefined,
    activeSessionId: undefined,
    activeStatus: { tone: "idle", text: "首页" },
    canSend: true,
    canStop: false,
    clearCurrentConversation: vi.fn(),
    computerSessionPersistStatus: { tone: "idle", text: "未开启" },
    hasConversation: false,
    hasOlderHistory: false,
    historyItems: [] as MockHistoryItem[],
    loadOlderHistory: vi.fn(),
    loadingOlderHistory: false,
    loadingSessionMessages: false,
    postSendFocusRequest: null,
    selectSession: vi.fn(async (_sessionId: string) => undefined) as (sessionId: string) => Promise<void>,
    sendMessage: async (message: string, selectedSkill?: unknown) => {
      const result = await options.sendAgentMessage({
        history: [],
        message,
        onReply: vi.fn(),
        onSessionId: vi.fn(),
        onStatus: vi.fn(),
        selectedSkill,
      });
      return result.ok;
    },
    startNewSession: vi.fn(),
    stopCurrentRun: vi.fn(),
  })),
}));

vi.mock("./hooks/useBodyScrollLock", () => ({
  useBodyScrollLock: vi.fn(),
}));

vi.mock("./hooks/useChatFeedScroll", () => ({
  useChatFeedScroll: () => ({
    handleScroll: vi.fn(),
    historySentinelRef: { current: null },
    registerUserMessageRow: vi.fn(() => vi.fn()),
    resetScrollDown: vi.fn(),
    scrollRef: { current: null },
    scrollToBottom: vi.fn(),
    showScrollDown: false,
    trailingSpacerPx: 0,
  }),
}));

vi.mock("./hooks/useMobileBridge", () => ({
  useMobileBridge: () => ({
    activateProvider: vi.fn(),
    appendSessionMessages: vi.fn(),
    approveExternalAgent: vi.fn(),
    bridgeUrl: "http://127.0.0.1:8080",
    config: {
      external_codex_permission_mode: "default",
      model: "deepseek-pro",
      project_root: "/tmp/ghost-os",
      provider: "DeepSeek",
    },
    connectBridge: vi.fn(),
    connectionStatus: { tone: "success", text: "HTTP fallback 已连接" },
    createProvider: vi.fn(),
    deleteOrchestration: vi.fn(),
    deleteProvider: vi.fn(),
    deleteSkill: vi.fn(),
    deleteTask: vi.fn(),
    getFullSession: vi.fn(),
    getSession: vi.fn(),
    host: { mobile: true, productName: "Ghost OS", target: "android", version: "test" },
    orchestrationList: undefined,
    orchestrationListError: "",
    providerList: { active_provider: "DeepSeek", providers: [] },
    refreshOrchestrations: vi.fn(),
    refreshProviders: vi.fn(),
    refreshSkills: vi.fn(),
    refreshTasks: vi.fn(),
    runTaskNow: vi.fn(),
    runningTaskId: "",
    searchSessions: vi.fn(),
    sendAgentMessage: mocks.bridgeSendAgentMessage,
    sessions: [],
    sessionsLoaded: true,
    setOrchestrationEnabled: vi.fn(),
    setSettings: vi.fn(),
    setTaskEnabled: vi.fn(),
    settings: {
      autoConnectEnabled: false,
      bridgeUrl: "http://127.0.0.1:8080",
      connectionMode: "http",
      persistComputerSessionsEnabled: false,
      remoteExecutionEnabled: false,
    },
    skillList: [],
    skillListError: "",
    status: { tone: "idle", text: "首页" },
    stopAgentRun: mocks.bridgeStopAgentRun,
    switchModel: vi.fn(),
    switchRuntimeSelection: vi.fn(),
    taskList: undefined,
    taskListError: "",
    updateExternalCodexPermissionMode: vi.fn(),
    updateProvider: vi.fn(),
    updateSkill: vi.fn(),
  }),
}));

vi.mock("./hooks/useMobileSessions", () => ({
  useMobileSessions: mocks.useMobileSessions,
}));

describe("App Codex mode routing", () => {
  beforeEach(() => {
    mocks.bridgeSendAgentMessage.mockResolvedValue({ mode: "remote", ok: true });
    mocks.bridgeStopAgentRun.mockResolvedValue({ ok: true, status: "stopped" });
    mocks.useMobileSessions.mockClear();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("routes composer plan mode through Codex instead of Ghost plan", async () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "添加内容" }));
    fireEvent.click(screen.getByRole("menuitem", { name: "功能" }));
    fireEvent.click(screen.getByRole("button", { name: "plan" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "请制定执行计划" } });
    fireEvent.click(screen.getByRole("button", { name: "发送任务" }));

    await waitFor(() => {
      expect(mocks.bridgeSendAgentMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          agentRuntime: "codex",
          codexModel: "gpt-5.5",
          mode: "plan",
        }),
      );
    });
  });

  it("shows a persistent completion card and opens the completed session", async () => {
    const selectSession = vi.fn(async (_sessionId: string) => undefined);
    mocks.useMobileSessions.mockImplementation((options: MockMobileSessionsOptions) =>
      mockMobileSessions(options, {
        historyItems: [historyItem("session-1", "设计复盘", "running")],
        selectSession,
      })
    );

    const { rerender } = render(<App />);

    expect(screen.queryByText("设计复盘会话已完成")).toBeNull();

    mocks.useMobileSessions.mockImplementation((options: MockMobileSessionsOptions) =>
      mockMobileSessions(options, {
        historyItems: [historyItem("session-1", "设计复盘", "success")],
        selectSession,
      })
    );
    rerender(<App />);

    const card = await screen.findByText("设计复盘会话已完成");
    fireEvent.click(card);

    expect(selectSession).toHaveBeenCalledWith("session-1");
    await waitFor(() => {
      expect(screen.queryByText("设计复盘会话已完成")).toBeNull();
    });
  });
});

function mockMobileSessions(
  options: MockMobileSessionsOptions,
  overrides: {
    historyItems?: MockHistoryItem[];
    selectSession?: (sessionId: string) => Promise<void>;
  } = {},
) {
  return {
    activeMessages: [],
    activeReply: undefined,
    activeSessionId: undefined,
    activeStatus: { tone: "idle", text: "首页" },
    canSend: true,
    canStop: false,
    clearCurrentConversation: vi.fn(),
    computerSessionPersistStatus: { tone: "idle", text: "未开启" },
    hasConversation: false,
    hasOlderHistory: false,
    historyItems: overrides.historyItems ?? [],
    loadOlderHistory: vi.fn(),
    loadingOlderHistory: false,
    loadingSessionMessages: false,
    postSendFocusRequest: null,
    selectSession: overrides.selectSession ?? (vi.fn(async (_sessionId: string) => undefined) as (sessionId: string) => Promise<void>),
    sendMessage: async (message: string, selectedSkill?: unknown) => {
      const result = await options.sendAgentMessage({
        history: [],
        message,
        onReply: vi.fn(),
        onSessionId: vi.fn(),
        onStatus: vi.fn(),
        selectedSkill,
      });
      return result.ok;
    },
    startNewSession: vi.fn(),
    stopCurrentRun: vi.fn(),
  };
}

function historyItem(id: string, title: string, status: MockHistoryItem["status"]): MockHistoryItem {
  return {
    id,
    pinned: false,
    status,
    title,
    updatedAt: "2026-07-09T00:00:00.000Z",
  };
}
