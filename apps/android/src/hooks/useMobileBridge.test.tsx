// @vitest-environment jsdom
import { renderHook, waitFor } from "@testing-library/react";
import { invoke } from "@tauri-apps/api/core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentMessageTaskPayload, LoopWritePayload } from "../mobileTypes";
import { useMobileBridge } from "./useMobileBridge";

vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn(),
}));

const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

describe("useMobileBridge", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      return {
        error: "",
        payload: payloadForAction(request.action, request.params),
        status: "success",
      };
    });
  });

  it("sends the persisted HTTP API token with bridge requests", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: " 080906 ",
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      }),
    );

    const { result } = renderHook(() => useMobileBridge());

    await result.current.connectBridge();

    await waitFor(() => {
      expect(bridgeBusRequests().some((request) => request.action === "SESSIONS_LIST")).toBe(true);
    });
    expect(bridgeBusRequests().every((request) => request.apiToken === "080906")).toBe(true);
    await waitFor(() => {
      expect(storedSettings().lastSuccessfulConnection).toMatchObject({ apiToken: "080906" });
    });
  });

  it("loads a full session through paginated SESSION_GET pages", async () => {
    useHTTPSettings();
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      if (request.action !== "SESSION_GET") {
        return {
          error: "",
          payload: payloadForAction(request.action, request.params),
          status: "success",
        };
      }

      return {
        error: "",
        payload: request.params.before === 3
          ? sessionDetailPayload([
            { index: 0, role: "user", text: "first" },
            { index: 1, role: "assistant", text: "second" },
          ], false)
          : sessionDetailPayload([
            { index: 3, role: "assistant", text: "fourth" },
            { index: 2, role: "user", text: "third" },
          ], true),
        status: "success",
      };
    });

    const { result } = renderHook(() => useMobileBridge());

    const detail = await result.current.getFullSession("session-1");

    expect(bridgeBusRequests().filter((request) => request.action === "SESSION_GET").map((request) => request.params)).toEqual([
      { id: "session-1", limit: 200 },
      { before: 3, id: "session-1", limit: 200 },
    ]);
    expect(detail.messages.map((message) => message.index)).toEqual([0, 1, 2, 3]);
    expect(detail.page.has_more_before).toBe(false);
  });

  it("searches sessions through SESSIONS_SEARCH", async () => {
    useHTTPSettings();
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      if (request.action !== "SESSIONS_SEARCH") {
        return {
          error: "",
          payload: payloadForAction(request.action, request.params),
          status: "success",
        };
      }

      return {
        error: "",
        payload: [{
          created_at: "2026-01-01T00:00:00.000Z",
          id: "session-search",
          message_count: 2,
          title: "Search result",
          token_count: 12,
          updated_at: "2026-01-02T00:00:00.000Z",
        }],
        status: "success",
      };
    });

    const { result } = renderHook(() => useMobileBridge());

    const sessions = await result.current.searchSessions(" fulltext ");

    expect(lastBridgeBusRequest("SESSIONS_SEARCH")?.params).toEqual({ query: "fulltext" });
    expect(sessions).toHaveLength(1);
    expect(sessions[0]?.id).toBe("session-search");
  });

  it("records the last successful HTTP connection after bridge connect succeeds", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: " token ",
        autoConnectEnabled: false,
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      }),
    );

    const { result } = renderHook(() => useMobileBridge());

    await result.current.connectBridge();

    await waitFor(() => {
      expect(storedSettings().lastSuccessfulConnection).toMatchObject({
        apiToken: "token",
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      });
    });
  });

  it("keeps the previous successful connection when bridge connect fails", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: "new-token",
        autoConnectEnabled: false,
        bridgeUrl: "http://new.example:8080",
        connectionMode: "http",
        lastSuccessfulConnection: {
          apiToken: "old-token",
          bridgeUrl: "http://old.example:8080",
          connectionMode: "http",
        },
      }),
    );
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      if (request.action === "CONFIG_GET") {
        return {
          error: "connect failed",
          payload: {},
          status: "error",
        };
      }
      return {
        error: "",
        payload: payloadForAction(request.action, request.params),
        status: "success",
      };
    });

    const { result } = renderHook(() => useMobileBridge());

    await result.current.connectBridge();

    await waitFor(() => {
      expect(result.current.connectionStatus.tone).toBe("error");
    });
    expect(storedSettings().lastSuccessfulConnection).toMatchObject({
      apiToken: "old-token",
      bridgeUrl: "http://old.example:8080",
      connectionMode: "http",
    });
  });

  it("auto connects with the last successful HTTP connection on startup", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: "draft-token",
        autoConnectEnabled: true,
        bridgeUrl: "http://draft.example:8080",
        connectionMode: "http",
        lastSuccessfulConnection: {
          apiToken: "last-token",
          bridgeUrl: "http://last.example:8080",
          connectionMode: "http",
        },
      }),
    );

    const { result } = renderHook(() => useMobileBridge());

    await waitFor(() => {
      expect(result.current.connectionStatus.tone).toBe("success");
    });
    const requests = bridgeBusRequests();
    expect(requests.some((request) => request.action === "CONFIG_GET")).toBe(true);
    expect(requests.every((request) => request.baseUrl === "http://last.example:8080")).toBe(true);
    expect(requests.every((request) => request.apiToken === "last-token")).toBe(true);
    expect(result.current.settings.bridgeUrl).toBe("http://last.example:8080");
  });

  it("does not auto connect when auto connect is off", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        autoConnectEnabled: false,
        bridgeUrl: "http://last.example:8080",
        connectionMode: "http",
        lastSuccessfulConnection: {
          bridgeUrl: "http://last.example:8080",
          connectionMode: "http",
        },
      }),
    );

    renderHook(() => useMobileBridge());

    await waitFor(() => {
      expect(bridgeBusRequests()).toEqual([]);
    });
  });

  it("does not fail bridge connection when skill management is unsupported", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: "token",
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      }),
    );
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      if (request.action === "SKILL_LIST") {
        return {
          error: 'unsupported action "SKILL_LIST", expected one of: AGENT_SEND|CONFIG_GET',
          payload: {},
          status: "error",
        };
      }
      return {
        error: "",
        payload: payloadForAction(request.action, request.params),
        status: "success",
      };
    });

    try {
      const { result } = renderHook(() => useMobileBridge());

      await result.current.connectBridge();

      await waitFor(() => {
        expect(result.current.connectionStatus.tone).toBe("success");
      });
      await waitFor(() => {
        expect(result.current.skillListError).toBe("电脑端不支持技能管理");
      });
      expect(result.current.config).toEqual({});
    } finally {
      consoleError.mockRestore();
    }
  });

  it("requests the user task list after bridge connection succeeds", async () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: "token",
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      }),
    );

    const { result } = renderHook(() => useMobileBridge());

    await result.current.connectBridge();

    await waitFor(() => {
      expect(bridgeBusRequests().some((request) => request.action === "TASK_LIST")).toBe(true);
    });
    expect(bridgeBusRequests().find((request) => request.action === "TASK_LIST")?.params).toEqual({ scope: "user" });
  });

  it("refreshLoops requests user tasks and keeps only relay agent message loops", async () => {
    useHTTPSettings();
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      return {
        error: "",
        payload: request.action === "TASK_LIST"
          ? [
            loopTask({ id: "loop-1", enabled: true }),
            loopTask({ id: "not-loop", agent_mode: "single" }),
            { id: "workflow-1", task_kind: "workflow" },
          ]
          : payloadForAction(request.action, request.params),
        status: "success",
      };
    });
    const { result } = renderHook(() => useMobileBridge());

    await result.current.refreshLoops();

    expect(bridgeBusRequests().find((request) => request.action === "TASK_LIST")?.params).toEqual({ scope: "user" });
    await waitFor(() => {
      expect(result.current.taskList?.map((task) => task.id)).toEqual(["loop-1"]);
    });
  });

  it("createLoop sends a relay agent task create payload", async () => {
    useHTTPSettings();
    const { result } = renderHook(() => useMobileBridge());
    const input = loopWritePayload({ interval_seconds: 300 });

    await result.current.createLoop(input);

    expect(lastBridgeBusRequest("TASK_CREATE")?.params).toEqual({
      scope: "user",
      task_kind: "agent_message",
      agent_mode: "relay",
      message: "检查状态",
      relay: {
        stop_policy: "max_rounds",
        max_rounds: 4,
        execution_timeout_ms: 0,
      },
      interval_seconds: 300,
    });
  });

  it("updateLoop sends only cron_expr when switching to cron mode", async () => {
    useHTTPSettings();
    const { result } = renderHook(() => useMobileBridge());
    const input = loopWritePayload({ cron_expr: "*/5 * * * *" });

    await result.current.updateLoop("loop-1", input);

    const params = lastBridgeBusRequest("TASK_UPDATE")?.params;
    expect(params).toMatchObject({
      id: "loop-1",
      scope: "user",
      task_kind: "agent_message",
      agent_mode: "relay",
      message: "检查状态",
      relay: {
        stop_policy: "max_rounds",
        max_rounds: 4,
        execution_timeout_ms: 0,
      },
      cron_expr: "*/5 * * * *",
    });
    expect(params).not.toHaveProperty("interval_seconds");
  });

  it("setLoopEnabled, runLoopNow, and deleteLoop send the expected task actions", async () => {
    useHTTPSettings();
    const { result } = renderHook(() => useMobileBridge());

    await result.current.setLoopEnabled("loop-1", false);
    await result.current.runLoopNow("loop-1");
    await result.current.deleteLoop("loop-1");

    expect(lastBridgeBusRequest("TASK_UPDATE")?.params).toEqual({
      id: "loop-1",
      scope: "user",
      enabled: false,
    });
    expect(lastBridgeBusRequest("TASK_RUN_NOW")?.params).toEqual({
      id: "loop-1",
      scope: "user",
      start_only: true,
    });
    expect(lastBridgeBusRequest("TASK_DELETE")?.params).toEqual({
      id: "loop-1",
      scope: "user",
    });
  });
});

interface BridgeBusRequest {
  action: string;
  apiToken?: string;
  baseUrl: string;
  params: Record<string, unknown>;
}

function storedSettings(): Record<string, unknown> {
  return JSON.parse(window.localStorage.getItem(SETTINGS_STORAGE_KEY) || "{}") as Record<string, unknown>;
}

function bridgeBusRequests(): BridgeBusRequest[] {
  return vi
    .mocked(invoke)
    .mock.calls.filter(([command]) => command === "bridge_bus_request")
    .map(([, args]) => bridgeBusRequestFromArgs(args));
}

function lastBridgeBusRequest(action: string): BridgeBusRequest | undefined {
  const requests = bridgeBusRequests().filter((request) => request.action === action);
  return requests[requests.length - 1];
}

function bridgeBusRequestFromArgs(args: unknown): BridgeBusRequest {
  const record = args as { request?: BridgeBusRequest };
  if (!record.request) {
    throw new Error("bridge_bus_request args must include request");
  }
  return record.request;
}

function payloadForAction(action: string, params: Record<string, unknown> = {}): unknown {
  switch (action) {
    case "CONFIG_GET":
      return {};
    case "CONFIG_PROVIDERS_GET":
      return { active_provider: "", providers: [] };
    case "SESSIONS_LIST":
    case "SESSIONS_SEARCH":
    case "SKILL_LIST":
    case "TASK_LIST":
      return [];
    case "TASK_CREATE":
    case "TASK_UPDATE":
      return loopTaskFromParams(params);
    case "TASK_RUN_NOW":
    case "TASK_DELETE":
      return {};
    default:
      return {};
  }
}

function sessionDetailPayload(messages: Array<Record<string, unknown>>, hasMoreBefore: boolean): Record<string, unknown> {
  return {
    created_at: "2026-01-01T00:00:00.000Z",
    id: "session-1",
    message_count: 4,
    messages,
    page: {
      has_more_before: hasMoreBefore,
      limit: 200,
      next_before: hasMoreBefore ? 3 : null,
    },
    title: "Session 1",
    token_count: 10,
    updated_at: "2026-01-02T00:00:00.000Z",
  };
}

function useHTTPSettings(): void {
  window.localStorage.setItem(
    SETTINGS_STORAGE_KEY,
    JSON.stringify({
      apiToken: "token",
      bridgeUrl: "http://100.80.12.34:8080",
      connectionMode: "http",
    }),
  );
}

function loopWritePayload(schedule: Pick<LoopWritePayload, "cron_expr" | "interval_seconds">): LoopWritePayload {
  return {
    message: "检查状态",
    relay: {
      stop_policy: "max_rounds",
      max_rounds: 4,
      execution_timeout_ms: 0,
    },
    ...schedule,
  };
}

function loopTask(overrides: Partial<AgentMessageTaskPayload> = {}): AgentMessageTaskPayload {
  return {
    id: "loop-1",
    message: "检查状态",
    agent_mode: "relay",
    relay: {
      stop_policy: "max_rounds",
      max_rounds: 4,
      execution_timeout_ms: 0,
    },
    task_kind: "agent_message",
    schedule_type: "interval",
    interval_seconds: 300,
    enabled: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function loopTaskFromParams(params: Record<string, unknown>): AgentMessageTaskPayload {
  const cronExpr = typeof params.cron_expr === "string" ? params.cron_expr : undefined;
  const intervalSeconds = typeof params.interval_seconds === "number" ? params.interval_seconds : undefined;
  return loopTask({
    id: typeof params.id === "string" ? params.id : "loop-created",
    message: typeof params.message === "string" ? params.message : "检查状态",
    relay: params.relay as AgentMessageTaskPayload["relay"],
    schedule_type: cronExpr ? "cron" : "interval",
    cron_expr: cronExpr,
    interval_seconds: cronExpr ? undefined : intervalSeconds ?? 300,
    enabled: typeof params.enabled === "boolean" ? params.enabled : true,
  });
}
