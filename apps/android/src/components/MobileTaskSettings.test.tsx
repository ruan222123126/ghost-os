// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AgentMessageTaskPayload, TaskPayload, WorkflowTaskPayload } from "../mobileTypes";
import { MobileTaskSettings } from "./MobileTaskSettings";

describe("MobileTaskSettings", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("shows loading, error, and empty states with task copy", () => {
    const { rerender } = renderTaskSettings({ tasks: undefined });
    expect(screen.getByText("任务加载中")).toBeTruthy();
    expect(screen.getByText("加载中")).toBeTruthy();

    rerender(<MobileTaskSettings {...propsWith({ tasks: undefined, loadError: "任务列表加载失败：boom" })} />);
    expect(screen.getByRole("alert").textContent).toBe("任务列表加载失败：boom");
    expect(screen.getByText("任务未加载")).toBeTruthy();

    rerender(<MobileTaskSettings {...propsWith({ tasks: [] })} />);
    expect(screen.getByText("暂无任务")).toBeTruthy();
    expect(screen.getByText("未配置任务")).toBeTruthy();
  });

  it("shows text, loop, and workflow task cards", () => {
    const { container } = renderTaskSettings({
      tasks: [
        agentTask({ id: "text-1", agent_mode: "single", message: "普通文本任务" }),
        agentTask({ id: "loop-1", agent_mode: "relay", runtime_overrides: { preset_id: "preset-1" } }),
        workflowTask({ id: "workflow-1", enabled: false, last_error: "workflow failure" }),
      ],
    });

    expect(screen.getByText("2/3 已启用")).toBeTruthy();
    expect(screen.getByText("文本任务")).toBeTruthy();
    expect(screen.getByText("普通文本任务")).toBeTruthy();
    expect(screen.getByText("循环任务")).toBeTruthy();
    expect(screen.getByText("Preset preset-1")).toBeTruthy();
    expect(screen.getAllByText("工作流任务")).toHaveLength(2);
    expect(screen.getByText("工作流步骤：2")).toBeTruthy();
    expect(screen.getByText("由工作流编辑器管理")).toBeTruthy();
    expect(screen.getByText("workflow failure")).toBeTruthy();
    expect(container.textContent).not.toContain("text-1");
    expect(container.textContent).not.toContain("loop-1");
    expect(container.textContent).not.toContain("workflow-1");
  });

  it("allows workflow tasks to run, toggle, and delete", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const props = propsWith({ tasks: [workflowTask({ id: "workflow-1", enabled: false })] });
    render(<MobileTaskSettings {...props} />);

    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
    expect(screen.queryByRole("button", { name: "新增任务" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "运行" }));
    await waitFor(() => {
      expect(props.onRunTaskNow).toHaveBeenCalledWith("workflow-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "启用" }));
    await waitFor(() => {
      expect(props.onSetTaskEnabled).toHaveBeenCalledWith("workflow-1", true);
    });

    fireEvent.click(screen.getByRole("button", { name: "删除工作流任务" }));
    await waitFor(() => {
      expect(props.onDeleteTask).toHaveBeenCalledWith("workflow-1");
    });
    expect(confirm).toHaveBeenCalledWith("删除工作流任务？");
  });

  it("allows non-workflow tasks to run, toggle, and delete", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const props = propsWith({ tasks: [agentTask({ id: "loop-1", enabled: true })] });
    render(<MobileTaskSettings {...props} />);

    fireEvent.click(screen.getByRole("button", { name: "运行" }));
    await waitFor(() => {
      expect(props.onRunTaskNow).toHaveBeenCalledWith("loop-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "停用" }));
    await waitFor(() => {
      expect(props.onSetTaskEnabled).toHaveBeenCalledWith("loop-1", false);
    });

    fireEvent.click(screen.getByRole("button", { name: "删除循环任务" }));
    await waitFor(() => {
      expect(props.onDeleteTask).toHaveBeenCalledWith("loop-1");
    });
    expect(confirm).toHaveBeenCalledWith("删除循环任务？");
  });
});

function renderTaskSettings(overrides: Partial<MobileTaskSettingsPropsForTest> = {}) {
  return render(<MobileTaskSettings {...propsWith(overrides)} />);
}

function propsWith(overrides: Partial<MobileTaskSettingsPropsForTest> = {}) {
  return {
    loadError: "",
    runningTaskId: "",
    tasks: [],
    onDeleteTask: vi.fn(async () => true),
    onRefreshTasks: vi.fn(async () => true),
    onRunTaskNow: vi.fn(async () => true),
    onSetTaskEnabled: vi.fn(async () => true),
    ...overrides,
  };
}

interface MobileTaskSettingsPropsForTest {
  loadError: string;
  runningTaskId: string;
  tasks: TaskPayload[] | undefined;
  onDeleteTask: (id: string) => Promise<boolean>;
  onRefreshTasks: () => Promise<boolean>;
  onRunTaskNow: (id: string) => Promise<boolean>;
  onSetTaskEnabled: (id: string, enabled: boolean) => Promise<boolean>;
}

function agentTask(overrides: Partial<AgentMessageTaskPayload> = {}): AgentMessageTaskPayload {
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

function workflowTask(overrides: Partial<WorkflowTaskPayload> = {}): WorkflowTaskPayload {
  return {
    id: "workflow-1",
    task_kind: "workflow",
    workflow: {
      nodes: [
        { id: "start", type: "start" },
        { id: "agent", type: "agent" },
      ],
      edges: [],
    },
    schedule_type: "cron",
    cron_expr: "*/5 * * * *",
    enabled: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}
