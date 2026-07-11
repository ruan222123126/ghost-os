import { useState } from "react";
import { Play, Power, RefreshCw, Trash2 } from "lucide-react";
import type { OrchestrationTaskPayload } from "../mobileTypes";
import "./MobileTaskSettings.css";

interface MobileOrchestrationSettingsProps {
  loadError: string;
  orchestrations: OrchestrationTaskPayload[] | undefined;
  runningTaskId: string;
  onDeleteOrchestration: (id: string) => Promise<boolean>;
  onRefreshOrchestrations: () => Promise<boolean>;
  onRunOrchestrationNow: (id: string) => Promise<boolean>;
  onSetOrchestrationEnabled: (id: string, enabled: boolean) => Promise<boolean>;
}

export function MobileOrchestrationSettings(props: MobileOrchestrationSettingsProps) {
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  const orchestrations = prioritizeEnabledOrchestrations(props.orchestrations ?? []);
  const enabledCount = orchestrations.filter((task) => task.enabled).length;
  const error = actionError || props.loadError;

  return (
    <section className="mobile-settings-loop-shell" aria-labelledby="mobile-settings-orchestration-title">
      <div className="mobile-settings-loop-title-row">
        <div>
          <div className="mobile-settings-connection-title" id="mobile-settings-orchestration-title">
            编排
          </div>
          <p className="mobile-settings-loop-summary">{orchestrationSummary(props.orchestrations, enabledCount)}</p>
        </div>
        <div className="mobile-settings-loop-title-actions">
          <button
            type="button"
            aria-label="刷新编排"
            disabled={busy}
            onClick={() =>
              void runOrchestrationAction(
                setBusy,
                setActionError,
                props.onRefreshOrchestrations,
                "编排刷新失败",
              )}
          >
            <RefreshCw className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
        </div>
      </div>

      {error ? <p className="mobile-settings-error is-card" role="alert">{error}</p> : null}

      {props.orchestrations === undefined ? (
        <div className="mobile-settings-loop-empty-card">{error ? "编排未加载" : "编排加载中"}</div>
      ) : orchestrations.length === 0 ? (
        <div className="mobile-settings-loop-empty-card">暂无编排</div>
      ) : (
        <div className="mobile-settings-loop-list">
          {orchestrations.map((orchestration) => (
            <OrchestrationCard
              key={orchestration.id}
              busy={busy}
              orchestration={orchestration}
              running={props.runningTaskId === orchestration.id}
              onDelete={(target) =>
                void deleteOrchestration(target, setBusy, setActionError, props.onDeleteOrchestration)}
              onRun={(target) =>
                void runOrchestrationAction(
                  setBusy,
                  setActionError,
                  () => props.onRunOrchestrationNow(target.id),
                  "编排启动失败",
                )}
              onToggle={(target) =>
                void runOrchestrationAction(
                  setBusy,
                  setActionError,
                  () => props.onSetOrchestrationEnabled(target.id, !target.enabled),
                  "编排状态保存失败",
                )}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function OrchestrationCard(props: {
  busy: boolean;
  orchestration: OrchestrationTaskPayload;
  running: boolean;
  onDelete: (task: OrchestrationTaskPayload) => void;
  onRun: (task: OrchestrationTaskPayload) => void;
  onToggle: (task: OrchestrationTaskPayload) => void;
}) {
  const { busy, orchestration, running } = props;
  const nodeCounts = orchestrationNodeCounts(orchestration);

  return (
    <article className={`mobile-settings-loop-card ${orchestration.enabled ? "is-enabled" : ""}`}>
      <div className="mobile-settings-loop-card-main">
        <div className="mobile-settings-loop-badges">
          <span>{orchestration.enabled ? "已启用" : "已停用"}</span>
          <span>编排任务</span>
        </div>
        <p className="mobile-settings-loop-message">{orchestration.name}</p>
        <p className="mobile-settings-loop-meta">{formatSchedule(orchestration)}</p>
        <p className="mobile-settings-loop-meta">
          分组 {nodeCounts.groups} / Agent {nodeCounts.agents}
        </p>
        <p className="mobile-settings-loop-meta">连接 {orchestration.orchestration.edges.length}</p>
        {orchestration.last_error ? <p className="mobile-settings-loop-last-error">{orchestration.last_error}</p> : null}
      </div>
      <div className="mobile-settings-loop-actions">
        <button type="button" disabled={busy || running} onClick={() => props.onRun(orchestration)}>
          <Play className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {running ? "运行中" : "运行"}
        </button>
        <button type="button" disabled={busy} onClick={() => props.onToggle(orchestration)}>
          <Power className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {orchestration.enabled ? "停用" : "启用"}
        </button>
        <button
          type="button"
          aria-label={`删除编排 ${orchestration.name}`}
          disabled={busy}
          onClick={() => props.onDelete(orchestration)}
        >
          <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>
    </article>
  );
}

async function runOrchestrationAction(
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

async function deleteOrchestration(
  orchestration: OrchestrationTaskPayload,
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  onDeleteOrchestration: (id: string) => Promise<boolean>,
): Promise<void> {
  if (!window.confirm(`删除编排 ${orchestration.name}？`)) {
    return;
  }
  await runOrchestrationAction(setBusy, setError, () => onDeleteOrchestration(orchestration.id), "编排删除失败");
}

function orchestrationSummary(
  orchestrations: OrchestrationTaskPayload[] | undefined,
  enabledCount: number,
): string {
  if (!orchestrations) {
    return "加载中";
  }
  if (orchestrations.length === 0) {
    return "未配置编排";
  }
  return `${enabledCount}/${orchestrations.length} 已启用`;
}

function prioritizeEnabledOrchestrations(orchestrations: OrchestrationTaskPayload[]): OrchestrationTaskPayload[] {
  return [...orchestrations].sort((left, right) => {
    if (left.enabled !== right.enabled) {
      return left.enabled ? -1 : 1;
    }
    if (left.name !== right.name) {
      return left.name.localeCompare(right.name);
    }
    return left.id.localeCompare(right.id);
  });
}

function orchestrationNodeCounts(orchestration: OrchestrationTaskPayload): { agents: number; groups: number } {
  return orchestration.orchestration.nodes.reduce(
    (counts, node) => ({
      agents: counts.agents + (node.type === "agent" ? 1 : 0),
      groups: counts.groups + (node.type === "group" ? 1 : 0),
    }),
    { agents: 0, groups: 0 },
  );
}

function formatSchedule(task: OrchestrationTaskPayload): string {
  if (task.schedule_type === "interval") {
    return `每 ${task.interval_seconds ?? 0} 秒`;
  }
  return `Cron ${task.cron_expr ?? ""}`;
}
