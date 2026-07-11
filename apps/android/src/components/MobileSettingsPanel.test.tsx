// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MobileSettingsPanel } from "./MobileSettingsPanel";

describe("MobileSettingsPanel", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders management settings without connection controls", () => {
    renderSettingsPanel();

    expect(screen.getByRole("heading", { name: "设置" })).toBeTruthy();
    expect(screen.getByRole("button", { name: /供应商/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /任务/ })).toBeTruthy();
    expect(screen.queryByRole("switch", { name: /是否自动连接/ })).toBeNull();
    expect(screen.queryByRole("switch", { name: /跟随电脑模型/ })).toBeNull();
  });

  it("opens orchestration settings when bridge is connected", () => {
    renderSettingsPanel({
      connectionStatus: { tone: "success", text: "已连接" },
      orchestrationList: [],
    });

    fireEvent.click(screen.getByRole("button", { name: /编排/ }));

    expect(screen.getByRole("heading", { name: "编排" })).toBeTruthy();
    expect(screen.getByText("暂无编排")).toBeTruthy();
  });

  it("opens providers with the merged provider list", () => {
    renderSettingsPanel({
      config: { model: "gpt-4o-mini", provider: "Phone OpenAI" },
      connectionStatus: { tone: "success", text: "已连接" },
      orchestrationList: [],
      providerList: {
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
      },
    });

    fireEvent.click(screen.getByRole("button", { name: /供应商/ }));

    expect(screen.getByText("Remote OpenAI")).toBeTruthy();
  });
});

function renderSettingsPanel(
  overrides: Partial<Parameters<typeof MobileSettingsPanel>[0]> = {},
) {
  return render(
    <MobileSettingsPanel
      open
      config={undefined}
      connectionStatus={{ tone: "idle", text: "未连接" }}
      providerList={undefined}
      onActivateProvider={vi.fn()}
      onClose={vi.fn()}
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
      {...overrides}
    />,
  );
}
