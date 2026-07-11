// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MobileConnectionPanel } from "./MobileConnectionPanel";
import type { StoredSettings } from "../mobileTypes";

describe("MobileConnectionPanel", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders connection switches and updates persisted computer session settings", () => {
    const settings = baseSettings();
    const onSettingsChange = vi.fn();

    renderConnectionPanel({ settings, onSettingsChange });

    const switches = screen.getAllByRole("switch");
    expect(switches[0]?.textContent).toContain("是否自动连接");
    expect(switches[1]?.textContent).toContain("是否持久化电脑会话内容");
    expect(switches[2]?.textContent).toContain("跟随电脑模型");
    expect(switches[2]?.textContent).toContain("连接电脑后可开启");

    fireEvent.click(switches[1]);

    const update = onSettingsChange.mock.calls[0]?.[0] as (current: StoredSettings) => StoredSettings;
    expect(update(settings)).toMatchObject({ persistComputerSessionsEnabled: true });
  });

  it("shows model-following copy based on connection and enabled state", () => {
    renderConnectionPanel({
      connectionStatus: { tone: "success", text: "已连接" },
      settings: { ...baseSettings(), remoteExecutionEnabled: true },
    });

    expect(screen.getByRole("switch", { name: /跟随电脑模型/ }).textContent).toContain("使用电脑端当前激活模型");
  });

  it("opens connection detail from the connection button", () => {
    renderConnectionPanel();

    expect(screen.queryByLabelText("API Token")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: /连接/ }));

    expect(screen.getByLabelText("API Token")).toBeTruthy();
    expect(screen.getByRole("group", { name: "连接模式" })).toBeTruthy();
  });

  it("saves HTTP bridge URL and token drafts through settings updates", () => {
    const settings = baseSettings();
    const onSettingsChange = vi.fn();

    renderConnectionPanel({ settings, onSettingsChange });

    fireEvent.click(screen.getByRole("button", { name: /连接/ }));

    fireEvent.change(screen.getByDisplayValue(settings.bridgeUrl), {
      target: { value: "http://192.168.1.10:8080" },
    });
    fireEvent.change(screen.getByLabelText("API Token"), {
      target: { value: "token-1" },
    });

    const saveUrl = onSettingsChange.mock.calls[0]?.[0] as (current: StoredSettings) => StoredSettings;
    const saveToken = onSettingsChange.mock.calls[1]?.[0] as (current: StoredSettings) => StoredSettings;
    expect(saveUrl(settings)).toMatchObject({ bridgeUrl: "http://192.168.1.10:8080" });
    expect(saveToken(settings)).toMatchObject({ apiToken: "token-1" });
  });

  it("keeps backspace inside the bridge url field instead of bubbling to the surrounding page", () => {
    const onSettingsChange = vi.fn();
    const surroundingBackHandler = vi.fn();

    render(
      <div
        onKeyDown={(event) => {
          if (event.key === "Backspace") {
            surroundingBackHandler();
          }
        }}
      >
        <MobileConnectionPanel
          open
          settings={baseSettings()}
          computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
          connectionStatus={{ tone: "idle", text: "未连接" }}
          onClose={vi.fn()}
          onConnect={vi.fn()}
          onSettingsChange={onSettingsChange}
        />
      </div>,
    );

    fireEvent.click(screen.getByRole("button", { name: /连接/ }));

    const bridgeUrlInput = screen.getByDisplayValue("http://127.0.0.1:8080") as HTMLInputElement;
    bridgeUrlInput.focus();
    bridgeUrlInput.setSelectionRange(bridgeUrlInput.value.length, bridgeUrlInput.value.length);

    fireEvent.keyDown(bridgeUrlInput, { key: "Backspace" });

    expect(surroundingBackHandler).not.toHaveBeenCalled();
    expect(screen.getByDisplayValue("http://127.0.0.1:808")).toBeTruthy();
  });

  it("keeps backspace inside the pairing textarea instead of bubbling to the surrounding page", () => {
    const onSettingsChange = vi.fn();
    const surroundingBackHandler = vi.fn();

    render(
      <div
        onKeyDown={(event) => {
          if (event.key === "Backspace") {
            surroundingBackHandler();
          }
        }}
      >
        <MobileConnectionPanel
          open
          settings={{ ...baseSettings(), connectionMode: "webrtc" }}
          computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
          connectionStatus={{ tone: "idle", text: "未连接" }}
          onClose={vi.fn()}
          onConnect={vi.fn()}
          onSettingsChange={onSettingsChange}
        />
      </div>,
    );

    fireEvent.click(screen.getByRole("button", { name: /连接/ }));

    const pairingUriInput = screen.getByLabelText("WebRTC 配对 URI") as HTMLTextAreaElement;
    fireEvent.change(pairingUriInput, { target: { value: "ghost-os://mobile-pair?abc=123" } });
    pairingUriInput.focus();
    pairingUriInput.setSelectionRange(pairingUriInput.value.length, pairingUriInput.value.length);

    fireEvent.keyDown(pairingUriInput, { key: "Backspace" });

    expect(surroundingBackHandler).not.toHaveBeenCalled();
    expect(screen.getByDisplayValue("ghost-os://mobile-pair?abc=12")).toBeTruthy();
  });
});

function renderConnectionPanel(
  overrides: Partial<Parameters<typeof MobileConnectionPanel>[0]> = {},
) {
  return render(
    <MobileConnectionPanel
      open
      settings={baseSettings()}
      computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
      connectionStatus={{ tone: "idle", text: "未连接" }}
      onClose={vi.fn()}
      onConnect={vi.fn()}
      onSettingsChange={vi.fn()}
      {...overrides}
    />,
  );
}

function baseSettings(): StoredSettings {
  return {
    apiToken: "",
    autoConnectEnabled: false,
    bridgeUrl: "http://127.0.0.1:8080",
    connectionMode: "http",
    persistComputerSessionsEnabled: false,
    remoteExecutionEnabled: false,
  };
}
