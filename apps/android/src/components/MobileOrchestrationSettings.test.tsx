// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { OrchestrationTaskPayload } from "../mobileTypes";
import { MobileOrchestrationSettings } from "./MobileOrchestrationSettings";

describe("MobileOrchestrationSettings", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("shows loading, error, and empty states", () => {
    const { rerender } = renderOrchestrationSettings({ orchestrations: undefined });
    expect(screen.getByText("编排加载中")).toBeTruthy();
    expect(screen.getByText("加载中")).toBeTruthy();

    rerender(
      <MobileOrchestrationSettings
        {...propsWith({ orchestrations: undefined, loadError: "编排列表加载失败：boom" })}
      />,
    );
    expect(screen.getByRole("alert").textContent).toBe("编排列表加载失败：boom");
    expect(screen.getByText("编排未加载")).toBeTruthy();

    rerender(<MobileOrchestrationSettings {...propsWith({ orchestrations: [] })} />);
    expect(screen.getByText("暂无编排")).toBeTruthy();
    expect(screen.getByText("未配置编排")).toBeTruthy();
  });

  it("shows orchestration cards with schedule, node counts, and errors", () => {
    const { container } = renderOrchestrationSettings({
      orchestrations: [
        orchestrationTask({ id: "orchestration-1", last_error: "orchestration failure" }),
        orchestrationTask({ id: "orchestration-2", enabled: false, name: "夜间检查" }),
      ],
    });

    expect(screen.getByText("1/2 已启用")).toBeTruthy();
    expect(screen.getAllByText("编排任务")).toHaveLength(2);
    expect(screen.getByText("客服编排")).toBeTruthy();
    expect(screen.getAllByText("每 300 秒")).toHaveLength(2);
    expect(screen.getAllByText("分组 1 / Agent 2")).toHaveLength(2);
    expect(screen.getAllByText("连接 2")).toHaveLength(2);
    expect(screen.getByText("orchestration failure")).toBeTruthy();
    expect(container.textContent).not.toContain("orchestration-1");
    expect(container.textContent).not.toContain("orchestration-2");
  });

  it("allows run, toggle, and delete", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const props = propsWith({ orchestrations: [orchestrationTask({ enabled: false })] });
    render(<MobileOrchestrationSettings {...props} />);

    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /新增/ })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "运行" }));
    await waitFor(() => {
      expect(props.onRunOrchestrationNow).toHaveBeenCalledWith("orchestration-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "启用" }));
    await waitFor(() => {
      expect(props.onSetOrchestrationEnabled).toHaveBeenCalledWith("orchestration-1", true);
    });

    fireEvent.click(screen.getByRole("button", { name: "删除编排 客服编排" }));
    await waitFor(() => {
      expect(props.onDeleteOrchestration).toHaveBeenCalledWith("orchestration-1");
    });
    expect(confirm).toHaveBeenCalledWith("删除编排 客服编排？");
  });
});

function renderOrchestrationSettings(overrides: Partial<MobileOrchestrationSettingsPropsForTest> = {}) {
  return render(<MobileOrchestrationSettings {...propsWith(overrides)} />);
}

function propsWith(overrides: Partial<MobileOrchestrationSettingsPropsForTest> = {}) {
  return {
    loadError: "",
    orchestrations: [],
    runningTaskId: "",
    onDeleteOrchestration: vi.fn(async () => true),
    onRefreshOrchestrations: vi.fn(async () => true),
    onRunOrchestrationNow: vi.fn(async () => true),
    onSetOrchestrationEnabled: vi.fn(async () => true),
    ...overrides,
  };
}

interface MobileOrchestrationSettingsPropsForTest {
  loadError: string;
  orchestrations: OrchestrationTaskPayload[] | undefined;
  runningTaskId: string;
  onDeleteOrchestration: (id: string) => Promise<boolean>;
  onRefreshOrchestrations: () => Promise<boolean>;
  onRunOrchestrationNow: (id: string) => Promise<boolean>;
  onSetOrchestrationEnabled: (id: string, enabled: boolean) => Promise<boolean>;
}

function orchestrationTask(overrides: Partial<OrchestrationTaskPayload> = {}): OrchestrationTaskPayload {
  return {
    id: "orchestration-1",
    name: "客服编排",
    task_kind: "orchestration",
    orchestration: {
      nodes: [
        {
          id: "group-1",
          type: "group",
          group: {
            title: "一线",
            shared_context: "",
            speaking_mode: "sequential",
            max_rounds: 4,
          },
        },
        {
          id: "agent-1",
          type: "agent",
          agent: {
            title: "分析",
            message: "分析问题",
          },
        },
        {
          id: "agent-2",
          type: "agent",
          agent: {
            title: "执行",
            message: "处理问题",
          },
        },
      ],
      edges: [
        { from_node_id: "group-1", to_node_id: "agent-1", kind: "member" },
        { from_node_id: "agent-1", to_node_id: "agent-2", kind: "control" },
      ],
    },
    schedule_type: "interval",
    interval_seconds: 300,
    enabled: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}
