import { useState } from "react";
import { Play, Power, RefreshCw, Trash2 } from "lucide-react";
import type { AgentMessageTaskPayload, TaskPayload, WorkflowTaskPayload } from "../mobileTypes";
import { formatSchedule } from "../lib/taskSchedule";
import "./MobileTaskSettings.css";

interface MobileTaskSettingsProps {
  loadError: string;
  runningTaskId: string;
  tasks: TaskPayload[] | undefined;
  onDeleteTask: (id: string) => Promise<boolean>;
  onRefreshTasks: () => Promise<boolean>;
  onRunTaskNow: (id: string) => Promise<boolean>;
  onSetTaskEnabled: (id: string, enabled: boolean) => Promise<boolean>;
}

export function MobileTaskSettings(props: MobileTaskSettingsProps) {
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  const tasks = prioritizeEnabledTasks(props.tasks ?? []);
  const enabledCount = tasks.filter((task) => task.enabled).length;
  const error = actionError || props.loadError;

  return (
    <section className="mobile-settings-loop-shell" aria-labelledby="mobile-settings-task-title">
      <div className="mobile-settings-loop-title-row">
        <div>
          <div className="mobile-settings-connection-title" id="mobile-settings-task-title">
            任务
          </div>
          <p className="mobile-settings-loop-summary">{taskSummary(props.tasks, enabledCount)}</p>
        </div>
        <div className="mobile-settings-loop-title-actions">
          <button
            type="button"
            aria-label="刷新任务"
            disabled={busy}
            onClick={() => void runTaskAction(setBusy, setActionError, props.onRefreshTasks, "任务刷新失败")}
          >
            <RefreshCw className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
        </div>
      </div>

      {error ? <p className="mobile-settings-error is-card" role="alert">{error}</p> : null}

      {props.tasks === undefined ? (
        <div className="mobile-settings-loop-empty-card">{error ? "任务未加载" : "任务加载中"}</div>
      ) : tasks.length === 0 ? (
        <div className="mobile-settings-loop-empty-card">暂无任务</div>
      ) : (
        <div className="mobile-settings-loop-list">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              busy={busy}
              task={task}
              running={props.runningTaskId === task.id}
              onDelete={(target) => void deleteTask(target, setBusy, setActionError, props.onDeleteTask)}
              onRun={(target) =>
                void runTaskAction(setBusy, setActionError, () => props.onRunTaskNow(target.id), "任务启动失败")}
              onToggle={(target) =>
                void runTaskAction(
                  setBusy,
                  setActionError,
                  () => props.onSetTaskEnabled(target.id, !target.enabled),
                  "任务状态保存失败",
                )}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function TaskCard(props: {
  busy: boolean;
  running: boolean;
  task: TaskPayload;
  onDelete: (task: TaskPayload) => void;
  onRun: (task: TaskPayload) => void;
  onToggle: (task: TaskPayload) => void;
}) {
  const { busy, running, task } = props;
  const workflowTask = isWorkflowTask(task);

  return (
    <article className={`mobile-settings-loop-card ${task.enabled ? "is-enabled" : ""}`}>
      <div className="mobile-settings-loop-card-main">
        <div className="mobile-settings-loop-badges">
          <span>{task.enabled ? "已启用" : "已停用"}</span>
          <span>{taskTypeLabel(task)}</span>
        </div>
        <p className="mobile-settings-loop-message">{primaryTaskText(task)}</p>
        <p className="mobile-settings-loop-meta">{formatSchedule(task)}</p>
        {workflowTask ? (
          <>
            <p className="mobile-settings-loop-meta">工作流步骤：{task.workflow.nodes.length}</p>
            <p className="mobile-settings-loop-meta">由工作流编辑器管理</p>
          </>
        ) : (
          <AgentTaskMeta task={task} />
        )}
        {task.last_error ? <p className="mobile-settings-loop-last-error">{task.last_error}</p> : null}
      </div>
      <div className="mobile-settings-loop-actions">
        <button type="button" disabled={busy || running} onClick={() => props.onRun(task)}>
          <Play className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {running ? "运行中" : "运行"}
        </button>
        <button type="button" disabled={busy} onClick={() => props.onToggle(task)}>
          <Power className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {task.enabled ? "停用" : "启用"}
        </button>
        <button type="button" aria-label={`删除${taskTypeLabel(task)}`} disabled={busy} onClick={() => props.onDelete(task)}>
          <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>
    </article>
  );
}

function AgentTaskMeta(props: { task: AgentMessageTaskPayload }) {
  return (
    <>
      <p className="mobile-settings-loop-meta">{formatAgentMode(props.task)}</p>
      {props.task.relay ? <p className="mobile-settings-loop-meta">{formatStopPolicy(props.task)}</p> : null}
      {props.task.runtime_overrides?.preset_id ? (
        <p className="mobile-settings-loop-meta">Preset {props.task.runtime_overrides.preset_id}</p>
      ) : null}
    </>
  );
}

async function runTaskAction(
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  action: () => Promise<boolean>,
  message: string,
): Promise<void> {
  setError("");
  setBusy(true);
  try {
    const ok = await action();
    if (!ok) {
      setError(message);
    }
  } finally {
    setBusy(false);
  }
}

async function deleteTask(
  task: TaskPayload,
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  onDeleteTask: (id: string) => Promise<boolean>,
): Promise<void> {
  if (!window.confirm(`删除${taskTypeLabel(task)}？`)) {
    return;
  }
  await runTaskAction(setBusy, setError, () => onDeleteTask(task.id), "任务删除失败");
}

function isWorkflowTask(task: TaskPayload): task is WorkflowTaskPayload {
  return task.task_kind === "workflow";
}

function taskSummary(tasks: TaskPayload[] | undefined, enabledCount: number): string {
  if (!tasks) {
    return "加载中";
  }
  if (tasks.length === 0) {
    return "未配置任务";
  }
  return `${enabledCount}/${tasks.length} 已启用`;
}

function prioritizeEnabledTasks(tasks: TaskPayload[]): TaskPayload[] {
  return [...tasks].sort((left, right) => {
    if (left.enabled !== right.enabled) {
      return left.enabled ? -1 : 1;
    }
    return left.id.localeCompare(right.id);
  });
}

function taskTypeLabel(task: TaskPayload): string {
  if (isWorkflowTask(task)) {
    return "工作流任务";
  }
  return task.agent_mode === "relay" ? "循环任务" : "文本任务";
}

function primaryTaskText(task: TaskPayload): string {
  if (isWorkflowTask(task)) {
    return "工作流任务";
  }
  return messageSummary(task.message);
}

function messageSummary(message: string): string {
  return message.trim().replace(/\s+/g, " ") || "空消息";
}

function formatAgentMode(task: AgentMessageTaskPayload): string {
  return task.agent_mode === "relay" ? "模式：循环" : "模式：文本";
}

function formatStopPolicy(task: AgentMessageTaskPayload): string {
  const maxRounds = task.relay?.max_rounds ?? 0;
  if (task.relay?.stop_policy === "max_rounds") {
    return `停止：最多 ${maxRounds} 轮`;
  }
  return `停止：AI 决定，最多 ${maxRounds} 轮`;
}
