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
  });
});

interface BridgeBusRequest {
  action: string;
  apiToken?: string;
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
