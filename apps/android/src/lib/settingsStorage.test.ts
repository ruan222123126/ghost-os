// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from "vitest";
import { DEFAULT_BRIDGE_URL, loadSettings, saveSettings } from "./settingsStorage";
import type { StoredSettings } from "../mobileTypes";

const SETTINGS_STORAGE_KEY = "ghost-os-mobile.settings";

describe("settingsStorage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("defaults auto connect to off for existing settings", () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        bridgeUrl: "http://localhost:9000",
        connectionMode: "http",
      }),
    );

    expect(loadSettings()).toMatchObject({
      autoConnectEnabled: false,
      bridgeUrl: "http://localhost:9000",
      connectionMode: "http",
    });
  });

  it("persists auto connect and last successful HTTP connection", () => {
    const settings: StoredSettings = {
      apiToken: " token ",
      autoConnectEnabled: true,
      bridgeUrl: "http://localhost:9000",
      connectionMode: "http",
      lastSuccessfulConnection: {
        apiToken: " old-token ",
        bridgeUrl: "http://localhost:8000",
        connectionMode: "http",
      },
    };

    saveSettings(settings);

    expect(JSON.parse(window.localStorage.getItem(SETTINGS_STORAGE_KEY) || "{}")).toMatchObject({
      apiToken: "token",
      autoConnectEnabled: true,
      bridgeUrl: "http://localhost:9000",
      connectionMode: "http",
      lastSuccessfulConnection: {
        apiToken: "old-token",
        bridgeUrl: "http://localhost:8000",
        connectionMode: "http",
      },
    });
  });

  it("drops invalid last successful WebRTC connection snapshots", () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({
        autoConnectEnabled: true,
        bridgeUrl: DEFAULT_BRIDGE_URL,
        connectionMode: "webrtc",
        lastSuccessfulConnection: {
          bridgeUrl: DEFAULT_BRIDGE_URL,
          connectionMode: "webrtc",
        },
      }),
    );

    expect(loadSettings().lastSuccessfulConnection).toBeUndefined();
  });

  it("removes legacy sessionId when settings are saved", () => {
    window.localStorage.setItem(
      SETTINGS_STORAGE_KEY,
      JSON.stringify({ bridgeUrl: "http://localhost:9000", connectionMode: "http", sessionId: "old" }),
    );

    saveSettings(loadSettings());

    expect(window.localStorage.getItem(SETTINGS_STORAGE_KEY)).not.toContain("sessionId");
  });
});
