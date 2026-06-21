// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AgentMessageTaskPayload, ConfigPayload, LoopWritePayload } from "../mobileTypes";
import { MobileLoopSettings } from "./MobileLoopSettings";

describe("MobileLoopSettings", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("shows loading, error, and empty states", () => {
    const { rerender } = renderLoopSettings({ loops: undefined });
    expect(screen.getByText("循环加载中")).toBeTruthy();
    expect(screen.getByText("加载中")).toBeTruthy();

    rerender(<MobileLoopSettings {...propsWith({ loops: undefined, loadError: "循环列表加载失败：boom" })} />);
    expect(screen.getByRole("alert").textContent).toBe("循环列表加载失败：boom");
    expect(screen.getByText("循环未加载")).toBeTruthy();

    rerender(<MobileLoopSettings {...propsWith({ loops: [] })} />);
    expect(screen.getByText("暂无循环")).toBeTruthy();
    expect(screen.getByText("未配置循环")).toBeTruthy();
  });

  it("shows enabled statistics and loop card details", () => {
    renderLoopSettings({
      loops: [
        loopTask({ id: "loop-enabled", enabled: true, runtime_overrides: { preset_id: "preset-1" } }),
        loopTask({ id: "loop-disabled", enabled: false, last_error: "last failure" }),
      ],
    });

    expect(screen.getByText("1/2 已启用")).toBeTruthy();
    expect(screen.getByText("loop-enabled")).toBeTruthy();
    expect(screen.getByText("Preset preset-1")).toBeTruthy();
    expect(screen.getByText("last failure")).toBeTruthy();
  });

  it("runs card actions for edit, run, toggle, and delete", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const props = propsWith({ loops: [loopTask({ id: "loop-1", enabled: true })] });
    render(<MobileLoopSettings {...props} />);

    fireEvent.click(screen.getByRole("button", { name: "运行" }));
    await waitFor(() => {
      expect(props.onRunLoopNow).toHaveBeenCalledWith("loop-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "停用" }));
    await waitFor(() => {
      expect(props.onSetLoopEnabled).toHaveBeenCalledWith("loop-1", false);
    });

    fireEvent.click(screen.getByRole("button", { name: "删除循环 loop-1" }));
    await waitFor(() => {
      expect(props.onDeleteLoop).toHaveBeenCalledWith("loop-1");
    });
    expect(confirm).toHaveBeenCalledWith("删除循环 loop-1？");

    fireEvent.click(screen.getByRole("button", { name: "编辑循环 loop-1" }));
    expect(screen.getByText("编辑循环")).toBeTruthy();
  });

  it("validates an empty message", async () => {
    renderEditor();

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    expect(await screen.findByText("消息不能为空")).toBeTruthy();
  });

  it("validates interval_seconds", async () => {
    renderEditor();
    fireEvent.change(screen.getByLabelText("消息"), { target: { value: "检查状态" } });
    fireEvent.change(screen.getByLabelText("interval_seconds"), { target: { value: "0" } });

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    expect(await screen.findByText("interval_seconds 必须是正整数")).toBeTruthy();
  });

  it("validates cron_expr", async () => {
    renderEditor();
    fireEvent.change(screen.getByLabelText("消息"), { target: { value: "检查状态" } });
    fireEvent.click(screen.getByRole("button", { name: "Cron" }));

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    expect(await screen.findByText("cron_expr 不能为空")).toBeTruthy();
  });

  it("validates max_rounds", async () => {
    renderEditor();
    fireEvent.change(screen.getByLabelText("消息"), { target: { value: "检查状态" } });
    fireEvent.change(screen.getByLabelText("max_rounds"), { target: { value: "0" } });

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    expect(await screen.findByText("max_rounds 必须是正整数")).toBeTruthy();
  });

  it("validates execution_timeout_ms", async () => {
    renderEditor();
    fireEvent.change(screen.getByLabelText("消息"), { target: { value: "检查状态" } });
    fireEvent.change(screen.getByLabelText("execution_timeout_ms"), { target: { value: "-1" } });

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    expect(await screen.findByText("execution_timeout_ms 必须是非负整数")).toBeTruthy();
  });

  it("creates a loop and refreshes the list after success", async () => {
    const props = propsWith({ loops: [] });
    render(<MobileLoopSettings {...props} />);
    fireEvent.click(screen.getByRole("button", { name: "新增循环" }));
    fireEvent.change(screen.getByLabelText("消息"), { target: { value: " 检查状态 " } });

    fireEvent.click(screen.getByRole("button", { name: "创建" }));

    await waitFor(() => {
      expect(props.onCreateLoop).toHaveBeenCalledWith({
        message: "检查状态",
        relay: {
          stop_policy: "ai_decides",
          max_rounds: 8,
          execution_timeout_ms: 0,
        },
        interval_seconds: 300,
      });
    });
    expect(props.onRefreshLoops).toHaveBeenCalled();
  });
});

function renderEditor(): void {
  renderLoopSettings({ loops: [] });
  fireEvent.click(screen.getByRole("button", { name: "新增循环" }));
}

function renderLoopSettings(overrides: Partial<MobileLoopSettingsPropsForTest> = {}) {
  return render(<MobileLoopSettings {...propsWith(overrides)} />);
}

function propsWith(overrides: Partial<MobileLoopSettingsPropsForTest> = {}) {
  return {
    config: {
      relay_default_stop_policy: "ai_decides",
      relay_default_max_rounds: 8,
      relay_default_execution_timeout_ms: 0,
    } satisfies ConfigPayload,
    loadError: "",
    loops: [],
    runningLoopId: "",
    onCreateLoop: vi.fn(async () => true),
    onDeleteLoop: vi.fn(async () => true),
    onRefreshLoops: vi.fn(async () => true),
    onRunLoopNow: vi.fn(async () => true),
    onSetLoopEnabled: vi.fn(async () => true),
    onUpdateLoop: vi.fn(async () => true),
    ...overrides,
  };
}

type MobileLoopSettingsPropsForTest = {
  config: ConfigPayload | undefined;
  loadError: string;
  loops: AgentMessageTaskPayload[] | undefined;
  runningLoopId: string;
  onCreateLoop: (input: LoopWritePayload) => Promise<boolean>;
  onDeleteLoop: (id: string) => Promise<boolean>;
  onRefreshLoops: () => Promise<boolean>;
  onRunLoopNow: (id: string) => Promise<boolean>;
  onSetLoopEnabled: (id: string, enabled: boolean) => Promise<boolean>;
  onUpdateLoop: (id: string, input: LoopWritePayload) => Promise<boolean>;
};

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
