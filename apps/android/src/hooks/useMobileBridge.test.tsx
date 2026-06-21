// @vitest-environment jsdom
import { renderHook, waitFor } from "@testing-library/react";
import { invoke } from "@tauri-apps/api/core";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
        payload: payloadForAction(request.action),
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
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        apiToken: "token",
        bridgeUrl: "http://100.80.12.34:8080",
        connectionMode: "http",
      }),
    );
    vi.mocked(invoke).mockImplementation(async (command: string, args?: unknown) => {
      if (command !== "bridge_bus_request") {
        return {};
      }

      const request = bridgeBusRequestFromArgs(args);
      if (request.action !== "SESSION_GET") {
        return {
          error: "",
          payload: payloadForAction(request.action),
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
        payload: payloadForAction(request.action),
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
        payload: payloadForAction(request.action),
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

function bridgeBusRequestFromArgs(args: unknown): BridgeBusRequest {
  const record = args as { request?: BridgeBusRequest };
  if (!record.request) {
    throw new Error("bridge_bus_request args must include request");
  }
  return record.request;
}

function payloadForAction(action: string): unknown {
  switch (action) {
    case "CONFIG_GET":
      return {};
    case "CONFIG_PROVIDERS_GET":
      return { active_provider: "", providers: [] };
    case "SESSIONS_LIST":
    case "SKILL_LIST":
      return [];
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
