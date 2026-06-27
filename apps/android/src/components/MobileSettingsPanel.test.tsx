// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MobileSettingsPanel } from "./MobileSettingsPanel";
import type { StoredSettings } from "../mobileTypes";

describe("MobileSettingsPanel", () => {
  afterEach(() => {
    cleanup();
  });

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
        localProviderList={undefined}
        providerList={undefined}
        onActivateProvider={vi.fn()}
        onClose={vi.fn()}
        onConnect={vi.fn()}
        onCreateProvider={vi.fn()}
        onDeleteOrchestration={vi.fn()}
        onDeleteProvider={vi.fn()}
        onDeleteSkill={vi.fn()}
        onDeleteTask={vi.fn()}
        onRefreshOrchestrations={vi.fn()}
        onRefreshProviders={vi.fn()}
        onRefreshSkills={vi.fn()}
        onRefreshTasks={vi.fn()}
        onRunTaskNow={vi.fn()}
        onSettingsChange={onSettingsChange}
        onSetOrchestrationEnabled={vi.fn()}
        onSetTaskEnabled={vi.fn()}
        onUpdateProvider={vi.fn()}
        onUpdateSkill={vi.fn()}
        onUpdateCodexPermission={vi.fn()}
        orchestrationList={undefined}
        orchestrationListError=""
        runningTaskId=""
        skillList={undefined}
        skillListError=""
        taskList={undefined}
        taskListError=""
      />,
    );

    const switches = screen.getAllByRole("switch");
    expect(switches[0]?.textContent).toContain("是否自动连接");
    expect(switches[1]?.textContent).toContain("是否持久化电脑会话内容");
    expect(switches[2]?.textContent).toContain("跟随电脑模型");
    expect(switches[2]?.textContent).toContain("连接电脑后可开启");
    expect(screen.getByRole("button", { name: /任务/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /编排/ })).toHaveProperty("disabled", true);

    fireEvent.click(switches[1]);

    const update = onSettingsChange.mock.calls[0]?.[0] as (current: StoredSettings) => StoredSettings;
    expect(update(settings)).toMatchObject({ persistComputerSessionsEnabled: true });
  });

  it("opens orchestration settings when bridge is connected", () => {
    render(
      <MobileSettingsPanel
        open
        settings={baseSettings()}
        config={undefined}
        computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
        connectionStatus={{ tone: "success", text: "已连接" }}
        localProviderList={undefined}
        providerList={undefined}
        onActivateProvider={vi.fn()}
        onClose={vi.fn()}
        onConnect={vi.fn()}
        onCreateProvider={vi.fn()}
        onDeleteOrchestration={vi.fn()}
        onDeleteProvider={vi.fn()}
        onDeleteSkill={vi.fn()}
        onDeleteTask={vi.fn()}
        onRefreshOrchestrations={vi.fn()}
        onRefreshProviders={vi.fn()}
        onRefreshSkills={vi.fn()}
        onRefreshTasks={vi.fn()}
        onRunTaskNow={vi.fn()}
        onSettingsChange={vi.fn()}
        onSetOrchestrationEnabled={vi.fn()}
        onSetTaskEnabled={vi.fn()}
        onUpdateProvider={vi.fn()}
        onUpdateSkill={vi.fn()}
        onUpdateCodexPermission={vi.fn()}
        orchestrationList={[]}
        orchestrationListError=""
        runningTaskId=""
        skillList={undefined}
        skillListError=""
        taskList={undefined}
        taskListError=""
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /编排/ }));

    expect(screen.getByRole("heading", { name: "编排" })).toBeTruthy();
    expect(screen.getByText("暂无编排")).toBeTruthy();
  });

  it("shows model-following copy based on connection and enabled state", () => {
    render(
      <MobileSettingsPanel
        open
        settings={{ ...baseSettings(), remoteExecutionEnabled: true }}
        config={{ model: "gpt-4o", provider: "Remote OpenAI" }}
        computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
        connectionStatus={{ tone: "success", text: "已连接" }}
        localProviderList={undefined}
        providerList={undefined}
        onActivateProvider={vi.fn()}
        onClose={vi.fn()}
        onConnect={vi.fn()}
        onCreateProvider={vi.fn()}
        onDeleteOrchestration={vi.fn()}
        onDeleteProvider={vi.fn()}
        onDeleteSkill={vi.fn()}
        onDeleteTask={vi.fn()}
        onRefreshOrchestrations={vi.fn()}
        onRefreshProviders={vi.fn()}
        onRefreshSkills={vi.fn()}
        onRefreshTasks={vi.fn()}
        onRunTaskNow={vi.fn()}
        onSettingsChange={vi.fn()}
        onSetOrchestrationEnabled={vi.fn()}
        onSetTaskEnabled={vi.fn()}
        onUpdateProvider={vi.fn()}
        onUpdateSkill={vi.fn()}
        onUpdateCodexPermission={vi.fn()}
        orchestrationList={undefined}
        orchestrationListError=""
        runningTaskId=""
        skillList={undefined}
        skillListError=""
        taskList={undefined}
        taskListError=""
      />,
    );

    expect(screen.getByRole("switch", { name: /跟随电脑模型/ }).textContent).toContain("使用电脑端当前激活模型");
  });

  it("opens providers with the merged provider list", () => {
    render(
      <MobileSettingsPanel
        open
        settings={baseSettings()}
        config={{ model: "gpt-4o-mini", provider: "Phone OpenAI" }}
        computerSessionPersistStatus={{ tone: "idle", text: "未开启" }}
        connectionStatus={{ tone: "success", text: "已连接" }}
        localProviderList={undefined}
        providerList={{
          active_provider: "Remote OpenAI",
          providers: [{
            api_key_set: true,
            base_url: "https://api.openai.com/v1",
            models: ["gpt-4o"],
            name: "Remote OpenAI",
            provider_id: "remote-provider-1",
            type: "openai",
            updated_at: "2026-01-01T00:00:00.000Z",
          }],
        }}
        onActivateProvider={vi.fn()}
        onClose={vi.fn()}
        onConnect={vi.fn()}
        onCreateProvider={vi.fn()}
        onDeleteOrchestration={vi.fn()}
        onDeleteProvider={vi.fn()}
        onDeleteSkill={vi.fn()}
        onDeleteTask={vi.fn()}
        onRefreshOrchestrations={vi.fn()}
        onRefreshProviders={vi.fn()}
        onRefreshSkills={vi.fn()}
        onRefreshTasks={vi.fn()}
        onRunTaskNow={vi.fn()}
        onSettingsChange={vi.fn()}
        onSetOrchestrationEnabled={vi.fn()}
        onSetTaskEnabled={vi.fn()}
        onUpdateProvider={vi.fn()}
        onUpdateSkill={vi.fn()}
        onUpdateCodexPermission={vi.fn()}
        orchestrationList={[]}
        orchestrationListError=""
        runningTaskId=""
        skillList={undefined}
        skillListError=""
        taskList={undefined}
        taskListError=""
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /供应商/ }));

    expect(screen.getByText("Remote OpenAI")).toBeTruthy();
  });
});

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
