// @vitest-environment jsdom
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MobileSettingsPanel } from "./MobileSettingsPanel";
import type { StoredSettings } from "../mobileTypes";

describe("MobileSettingsPanel", () => {
  it("renders computer session persistence switch after auto connect and updates settings", () => {
    const settings = baseSettings();
    const onSettingsChange = vi.fn();

    render(
      <MobileSettingsPanel
        open
        settings={settings}
        config={undefined}
        computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
        connectionStatus={{ tone: "idle", text: "未连接" }}
        providerList={undefined}
        onActivateProvider={vi.fn()}
        onClose={vi.fn()}
        onConnect={vi.fn()}
        onCreateProvider={vi.fn()}
        onDeleteProvider={vi.fn()}
        onDeleteSkill={vi.fn()}
        onRefreshProviders={vi.fn()}
        onRefreshSkills={vi.fn()}
        onSettingsChange={onSettingsChange}
        onUpdateProvider={vi.fn()}
        onUpdateSkill={vi.fn()}
        skillList={undefined}
        skillListError=""
      />,
    );

    const switches = screen.getAllByRole("switch");
    expect(switches[0]?.textContent).toContain("是否自动连接");
    expect(switches[1]?.textContent).toContain("是否持久化电脑会话内容");

    fireEvent.click(switches[1]);

    const update = onSettingsChange.mock.calls[0]?.[0] as (current: StoredSettings) => StoredSettings;
    expect(update(settings)).toMatchObject({ persistComputerSessionsEnabled: true });
  });
});

function baseSettings(): StoredSettings {
  return {
    apiToken: "",
    autoConnectEnabled: false,
    bridgeUrl: "http://127.0.0.1:8080",
    connectionMode: "http",
    persistComputerSessionsEnabled: false,
  };
}
