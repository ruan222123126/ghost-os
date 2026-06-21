import { useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { Pencil, Play, Plus, Power, RefreshCw, Trash2 } from "lucide-react";
import type {
  AgentMessageTaskPayload,
  ConfigPayload,
  LoopWritePayload,
  TaskRelayStopPolicy,
  TaskScheduleType,
} from "../mobileTypes";
import "./MobileLoopSettings.css";

interface MobileLoopSettingsProps {
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
}

interface LoopEditorState {
  message: string;
  scheduleMode: TaskScheduleType;
  intervalSeconds: string;
  cronExpr: string;
  stopPolicy: TaskRelayStopPolicy;
  maxRounds: string;
  executionTimeoutMS: string;
}

type LoopEditorMode = "create" | "edit";
type LoopView = "list" | "editor";

const DEFAULT_INTERVAL_SECONDS = "300";
const DEFAULT_RELAY_MAX_ROUNDS = "20";
const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = "0";

export function MobileLoopSettings(props: MobileLoopSettingsProps) {
  const [view, setView] = useState<LoopView>("list");
  const [editorMode, setEditorMode] = useState<LoopEditorMode>("create");
  const [editingLoopId, setEditingLoopId] = useState("");
  const [editor, setEditor] = useState<LoopEditorState>(() => createLoopEditorState(props.config));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function startCreate(): void {
    setEditor(createLoopEditorState(props.config));
    setEditorMode("create");
    setEditingLoopId("");
    setError("");
    setView("editor");
  }

  function startEdit(loop: AgentMessageTaskPayload): void {
    setEditor(editorStateFromLoop(loop, props.config));
    setEditorMode("edit");
    setEditingLoopId(loop.id);
    setError("");
    setView("editor");
  }

  async function submitLoop(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError("");
    setBusy(true);
    try {
      const input = loopWritePayloadFromEditor(editor);
      const ok = editorMode === "edit"
        ? await props.onUpdateLoop(editingLoopId, input)
        : await props.onCreateLoop(input);
      if (!ok) {
        setError("循环保存失败");
        return;
      }
      setView("list");
      await props.onRefreshLoops();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : String(caught));
    } finally {
      setBusy(false);
    }
  }

  if (view === "editor") {
    return (
      <LoopEditorForm
        busy={busy}
        editor={editor}
        editorMode={editorMode}
        error={error}
        onCancel={() => setView("list")}
        onChange={(patch) => setEditor((current) => ({ ...current, ...patch }))}
        onSubmit={submitLoop}
      />
    );
  }

  return (
    <LoopListView
      busy={busy}
      error={error || props.loadError}
      loops={props.loops}
      runningLoopId={props.runningLoopId}
      onCreate={startCreate}
      onDeleteLoop={(loop) => void deleteLoop(loop, setBusy, setError, props.onDeleteLoop)}
      onEdit={startEdit}
      onRefreshLoops={() => void runLoopAction(setBusy, setError, props.onRefreshLoops, "循环刷新失败")}
      onRunLoopNow={(loop) => void runLoopAction(setBusy, setError, () => props.onRunLoopNow(loop.id), "循环启动失败")}
      onToggleLoop={(loop) =>
        void runLoopAction(
          setBusy,
          setError,
          () => props.onSetLoopEnabled(loop.id, !loop.enabled),
          "循环状态保存失败",
        )}
    />
  );
}

interface LoopListViewProps {
  busy: boolean;
  error: string;
  loops: AgentMessageTaskPayload[] | undefined;
  runningLoopId: string;
  onCreate: () => void;
  onDeleteLoop: (loop: AgentMessageTaskPayload) => void;
  onEdit: (loop: AgentMessageTaskPayload) => void;
  onRefreshLoops: () => void;
  onRunLoopNow: (loop: AgentMessageTaskPayload) => void;
  onToggleLoop: (loop: AgentMessageTaskPayload) => void;
}

function LoopListView(props: LoopListViewProps) {
  const loops = prioritizeEnabledLoops(props.loops ?? []);
  const enabledCount = loops.filter((loop) => loop.enabled).length;

  return (
    <section className="mobile-settings-loop-shell" aria-labelledby="mobile-settings-loop-title">
      <div className="mobile-settings-loop-title-row">
        <div>
          <div className="mobile-settings-connection-title" id="mobile-settings-loop-title">
            循环
          </div>
          <p className="mobile-settings-loop-summary">{loopSummary(props.loops, enabledCount)}</p>
        </div>
        <div className="mobile-settings-loop-title-actions">
          <button type="button" aria-label="刷新循环" disabled={props.busy} onClick={props.onRefreshLoops}>
            <RefreshCw className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
          <button type="button" aria-label="新增循环" disabled={props.busy} onClick={props.onCreate}>
            <Plus className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          </button>
        </div>
      </div>

      {props.error ? <p className="mobile-settings-error is-card" role="alert">{props.error}</p> : null}

      {props.loops === undefined ? (
        <div className="mobile-settings-loop-empty-card">{props.error ? "循环未加载" : "循环加载中"}</div>
      ) : loops.length === 0 ? (
        <div className="mobile-settings-loop-empty-card">暂无循环</div>
      ) : (
        <div className="mobile-settings-loop-list">
          {loops.map((loop) => (
            <LoopCard
              key={loop.id}
              busy={props.busy}
              loop={loop}
              running={props.runningLoopId === loop.id}
              onDelete={props.onDeleteLoop}
              onEdit={props.onEdit}
              onRun={props.onRunLoopNow}
              onToggle={props.onToggleLoop}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function LoopCard(props: {
  busy: boolean;
  loop: AgentMessageTaskPayload;
  running: boolean;
  onDelete: (loop: AgentMessageTaskPayload) => void;
  onEdit: (loop: AgentMessageTaskPayload) => void;
  onRun: (loop: AgentMessageTaskPayload) => void;
  onToggle: (loop: AgentMessageTaskPayload) => void;
}) {
  const { busy, loop, running } = props;
  return (
    <article className={`mobile-settings-loop-card ${loop.enabled ? "is-enabled" : ""}`}>
      <div className="mobile-settings-loop-card-main">
        <div className="mobile-settings-loop-badges">
          <span>{loop.enabled ? "已启用" : "已停用"}</span>
          <code>{loop.id}</code>
        </div>
        <p className="mobile-settings-loop-message">{messageSummary(loop.message)}</p>
        <p className="mobile-settings-loop-meta">{formatSchedule(loop)}</p>
        <p className="mobile-settings-loop-meta">{formatStopPolicy(loop)}</p>
        {loop.runtime_overrides?.preset_id ? (
          <p className="mobile-settings-loop-meta">Preset {loop.runtime_overrides.preset_id}</p>
        ) : null}
        {loop.last_error ? <p className="mobile-settings-loop-last-error">{loop.last_error}</p> : null}
      </div>
      <div className="mobile-settings-loop-actions">
        <button type="button" disabled={busy || running} onClick={() => props.onRun(loop)}>
          <Play className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {running ? "运行中" : "运行"}
        </button>
        <button type="button" disabled={busy} onClick={() => props.onToggle(loop)}>
          <Power className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
          {loop.enabled ? "停用" : "启用"}
        </button>
        <button type="button" aria-label={`编辑循环 ${loop.id}`} disabled={busy} onClick={() => props.onEdit(loop)}>
          <Pencil className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
        <button type="button" aria-label={`删除循环 ${loop.id}`} disabled={busy} onClick={() => props.onDelete(loop)}>
          <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.7} />
        </button>
      </div>
    </article>
  );
}

interface LoopEditorFormProps {
  busy: boolean;
  editor: LoopEditorState;
  editorMode: LoopEditorMode;
  error: string;
  onCancel: () => void;
  onChange: (patch: Partial<LoopEditorState>) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
}

function LoopEditorForm(props: LoopEditorFormProps) {
  const submitLabel = props.busy ? "保存中" : props.editorMode === "edit" ? "保存" : "创建";
  const cronMode = props.editor.scheduleMode === "cron";

  return (
    <form className="mobile-settings-loop-form" noValidate onSubmit={(event) => void props.onSubmit(event)}>
      <div className="mobile-settings-connection-title">{props.editorMode === "edit" ? "编辑循环" : "新增循环"}</div>
      {props.error ? <p className="mobile-settings-error is-card" role="alert">{props.error}</p> : null}

      <LoopField label="消息">
        <textarea
          value={props.editor.message}
          disabled={props.busy}
          rows={5}
          aria-label="消息"
          onChange={(event) => props.onChange({ message: event.target.value })}
        />
      </LoopField>

      <LoopSegmentedField label="调度模式">
        <SegmentedButton
          active={!cronMode}
          disabled={props.busy}
          label="间隔"
          onClick={() => props.onChange({ scheduleMode: "interval" })}
        />
        <SegmentedButton
          active={cronMode}
          disabled={props.busy}
          label="Cron"
          onClick={() => props.onChange({ scheduleMode: "cron" })}
        />
      </LoopSegmentedField>

      {cronMode ? (
        <LoopField label="cron_expr">
          <input
            value={props.editor.cronExpr}
            disabled={props.busy}
            aria-label="cron_expr"
            onChange={(event) => props.onChange({ cronExpr: event.target.value })}
          />
        </LoopField>
      ) : (
        <LoopField label="interval_seconds">
          <input
            value={props.editor.intervalSeconds}
            disabled={props.busy}
            type="number"
            min="1"
            step="1"
            aria-label="interval_seconds"
            onChange={(event) => props.onChange({ intervalSeconds: event.target.value })}
          />
        </LoopField>
      )}

      <LoopSegmentedField label="停止策略">
        <SegmentedButton
          active={props.editor.stopPolicy === "ai_decides"}
          disabled={props.busy}
          label="AI 决定"
          onClick={() => props.onChange({ stopPolicy: "ai_decides" })}
        />
        <SegmentedButton
          active={props.editor.stopPolicy === "max_rounds"}
          disabled={props.busy}
          label="最大轮数"
          onClick={() => props.onChange({ stopPolicy: "max_rounds" })}
        />
      </LoopSegmentedField>

      <LoopField label="max_rounds">
        <input
          value={props.editor.maxRounds}
          disabled={props.busy}
          type="number"
          min="1"
          step="1"
          aria-label="max_rounds"
          onChange={(event) => props.onChange({ maxRounds: event.target.value })}
        />
      </LoopField>

      <LoopField label="execution_timeout_ms">
        <input
          value={props.editor.executionTimeoutMS}
          disabled={props.busy}
          type="number"
          min="0"
          step="1"
          aria-label="execution_timeout_ms"
          onChange={(event) => props.onChange({ executionTimeoutMS: event.target.value })}
        />
      </LoopField>

      <div className="mobile-settings-loop-form-actions">
        <button type="button" disabled={props.busy} onClick={props.onCancel}>
          取消
        </button>
        <button type="submit" disabled={props.busy}>
          {submitLabel}
        </button>
      </div>
    </form>
  );
}

function LoopField(props: { children: ReactNode; label: string }) {
  return (
    <label className="mobile-settings-loop-field">
      <span>{props.label}</span>
      {props.children}
    </label>
  );
}

function LoopSegmentedField(props: { children: ReactNode; label: string }) {
  return (
    <div className="mobile-settings-loop-field">
      <span>{props.label}</span>
      <div className="mobile-settings-loop-segmented" role="group" aria-label={props.label}>
        {props.children}
      </div>
    </div>
  );
}

function SegmentedButton(props: {
  active: boolean;
  disabled: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className={props.active ? "is-active" : ""}
      type="button"
      disabled={props.disabled}
      aria-pressed={props.active}
      onClick={props.onClick}
    >
      {props.label}
    </button>
  );
}

async function runLoopAction(
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

async function deleteLoop(
  loop: AgentMessageTaskPayload,
  setBusy: (busy: boolean) => void,
  setError: (error: string) => void,
  onDeleteLoop: (id: string) => Promise<boolean>,
): Promise<void> {
  if (!window.confirm(`删除循环 ${loop.id}？`)) {
    return;
  }
  await runLoopAction(setBusy, setError, () => onDeleteLoop(loop.id), "循环删除失败");
}

function createLoopEditorState(config: ConfigPayload | undefined): LoopEditorState {
  return {
    message: "",
    scheduleMode: "interval",
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    cronExpr: "",
    stopPolicy: config?.relay_default_stop_policy ?? "ai_decides",
    maxRounds: String(config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    executionTimeoutMS: String(
      config?.relay_default_execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
  };
}

function editorStateFromLoop(
  loop: AgentMessageTaskPayload,
  config: ConfigPayload | undefined,
): LoopEditorState {
  return {
    message: loop.message,
    scheduleMode: loop.schedule_type,
    intervalSeconds: loop.interval_seconds === undefined ? DEFAULT_INTERVAL_SECONDS : String(loop.interval_seconds),
    cronExpr: loop.cron_expr ?? "",
    stopPolicy: loop.relay?.stop_policy ?? config?.relay_default_stop_policy ?? "ai_decides",
    maxRounds: String(loop.relay?.max_rounds ?? config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    executionTimeoutMS: String(
      loop.relay?.execution_timeout_ms
        ?? config?.relay_default_execution_timeout_ms
        ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
  };
}

function loopWritePayloadFromEditor(editor: LoopEditorState): LoopWritePayload {
  return {
    message: parseMessage(editor.message),
    relay: {
      stop_policy: editor.stopPolicy,
      max_rounds: parsePositiveInteger(editor.maxRounds, "max_rounds"),
      execution_timeout_ms: parseNonNegativeInteger(editor.executionTimeoutMS, "execution_timeout_ms"),
    },
    ...scheduleFieldsFromEditor(editor),
  };
}

function scheduleFieldsFromEditor(editor: LoopEditorState): Pick<LoopWritePayload, "cron_expr" | "interval_seconds"> {
  if (editor.scheduleMode === "interval") {
    return { interval_seconds: parsePositiveInteger(editor.intervalSeconds, "interval_seconds") };
  }

  const cronExpr = editor.cronExpr.trim();
  if (!cronExpr) {
    throw new Error("cron_expr 不能为空");
  }
  return { cron_expr: cronExpr };
}

function parseMessage(raw: string): string {
  const message = raw.trim();
  if (!message) {
    throw new Error("消息不能为空");
  }
  return message;
}

function parsePositiveInteger(raw: string, fieldName: string): number {
  const value = raw.trim();
  if (!/^[1-9]\d*$/.test(value)) {
    throw new Error(`${fieldName} 必须是正整数`);
  }
  return Number(value);
}

function parseNonNegativeInteger(raw: string, fieldName: string): number {
  const value = raw.trim();
  if (!/^\d+$/.test(value)) {
    throw new Error(`${fieldName} 必须是非负整数`);
  }
  return Number(value);
}

function loopSummary(loops: AgentMessageTaskPayload[] | undefined, enabledCount: number): string {
  if (!loops) {
    return "加载中";
  }
  if (loops.length === 0) {
    return "未配置循环";
  }
  return `${enabledCount}/${loops.length} 已启用`;
}

function prioritizeEnabledLoops(loops: AgentMessageTaskPayload[]): AgentMessageTaskPayload[] {
  return [...loops].sort((left, right) => {
    if (left.enabled !== right.enabled) {
      return left.enabled ? -1 : 1;
    }
    return left.id.localeCompare(right.id);
  });
}

function messageSummary(message: string): string {
  return message.trim().replace(/\s+/g, " ") || "空消息";
}

function formatSchedule(loop: AgentMessageTaskPayload): string {
  if (loop.schedule_type === "interval") {
    return `每 ${loop.interval_seconds ?? 0} 秒`;
  }
  return `Cron ${loop.cron_expr ?? ""}`;
}

function formatStopPolicy(loop: AgentMessageTaskPayload): string {
  const maxRounds = loop.relay?.max_rounds ?? 0;
  if (loop.relay?.stop_policy === "max_rounds") {
    return `停止：最多 ${maxRounds} 轮`;
  }
  return `停止：AI 决定，最多 ${maxRounds} 轮`;
}
