import type {
  AgentMessageTaskPayload,
  BridgeConfig,
  TaskPayload,
  TaskRelayConfig,
  TaskRuntimeOverrides,
  TextTaskCreateRequest,
  TextTaskUpdateRequest,
} from '@/lib/types';

const DEFAULT_INTERVAL_SECONDS = '300';
const DEFAULT_RELAY_MAX_ROUNDS = '20';
const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = '0';
const DECIMAL_RADIX = 10;
const MIN_INTERVAL_SECONDS = 1;

export type LoopEditorMode = 'create' | 'edit';
export type LoopScheduleMode = 'interval' | 'cron';
export type LoopStopPolicy = TaskRelayConfig['stop_policy'];

export interface LoopEditorState {
  message: string;
  scheduleMode: LoopScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
  stopPolicy: LoopStopPolicy;
  maxRounds: string;
  executionTimeoutMS: string;
  presetId: string;
}

export function createLoopEditorState(config: BridgeConfig | null): LoopEditorState {
  return {
    message: '',
    scheduleMode: 'interval',
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    cronExpr: '',
    stopPolicy: config?.relay_default_stop_policy ?? 'ai_decides',
    maxRounds: String(config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    executionTimeoutMS: String(
      config?.relay_default_execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
    presetId: '',
  };
}

export function editorStateFromLoopTask(
  task: AgentMessageTaskPayload,
  config: BridgeConfig | null,
): LoopEditorState {
  const presetId = task.runtime_overrides?.preset_id ?? '';
  return {
    message: task.message,
    scheduleMode: task.schedule_type,
    intervalSeconds: task.interval_seconds === undefined ? DEFAULT_INTERVAL_SECONDS : String(task.interval_seconds),
    cronExpr: task.cron_expr ?? '',
    stopPolicy: task.relay?.stop_policy ?? config?.relay_default_stop_policy ?? 'ai_decides',
    maxRounds: String(task.relay?.max_rounds ?? config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    executionTimeoutMS: String(
      task.relay?.execution_timeout_ms
        ?? config?.relay_default_execution_timeout_ms
        ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
    presetId,
  };
}

export function isLoopTask(task: TaskPayload): task is AgentMessageTaskPayload {
  return task.task_kind === 'agent_message' && task.agent_mode === 'relay';
}

export function filterLoopTasks(tasks: TaskPayload[]): AgentMessageTaskPayload[] {
  return tasks.filter(isLoopTask);
}

export function loopCreateRequestFromEditor(editor: LoopEditorState): TextTaskCreateRequest {
  return {
    task_kind: 'agent_message',
    message: parseMessage(editor.message),
    agent_mode: 'relay',
    relay: relayConfigFromEditor(editor),
    runtime_overrides: runtimeOverridesFromPreset(editor.presetId),
    ...scheduleFieldsFromEditor(editor),
  };
}

export function loopUpdateRequestFromEditor(editor: LoopEditorState): TextTaskUpdateRequest {
  return {
    task_kind: 'agent_message',
    message: parseMessage(editor.message),
    agent_mode: 'relay',
    relay: relayConfigFromEditor(editor),
    runtime_overrides: runtimeOverridesForUpdate(editor),
    ...scheduleFieldsFromEditor(editor),
  };
}

function parseMessage(raw: string): string {
  const message = raw.trim();
  if (message.length === 0) {
    throw new Error('message is required');
  }
  return message;
}

function scheduleFieldsFromEditor(editor: LoopEditorState): {
  interval_seconds?: number;
  cron_expr?: string;
} {
  if (editor.scheduleMode === 'interval') {
    return { interval_seconds: parseIntervalSeconds(editor.intervalSeconds) };
  }

  const cronExpr = editor.cronExpr.trim();
  if (cronExpr.length === 0) {
    throw new Error('cron expression is required');
  }
  return { cron_expr: cronExpr };
}

function relayConfigFromEditor(editor: LoopEditorState): TaskRelayConfig {
  return {
    stop_policy: editor.stopPolicy,
    max_rounds: parsePositiveInteger(editor.maxRounds, 'relay max_rounds'),
    execution_timeout_ms: parseNonNegativeInteger(
      editor.executionTimeoutMS,
      'relay execution_timeout_ms',
    ),
  };
}

function runtimeOverridesForUpdate(
  editor: LoopEditorState,
): TaskRuntimeOverrides | Record<string, never> | undefined {
  return runtimeOverridesFromPreset(editor.presetId) ?? {};
}

function runtimeOverridesFromPreset(presetId: string): TaskRuntimeOverrides | undefined {
  const id = presetId.trim();
  if (id.length === 0) {
    return undefined;
  }
  return { preset_id: id };
}

function parseIntervalSeconds(value: string): number {
  const parsed = Number.parseInt(value.trim(), DECIMAL_RADIX);
  if (!Number.isFinite(parsed) || parsed < MIN_INTERVAL_SECONDS) {
    throw new Error('interval seconds must be a positive integer');
  }
  return parsed;
}

function parsePositiveInteger(raw: string, fieldName: string): number {
  const trimmed = raw.trim();
  if (!/^[1-9]\d*$/.test(trimmed)) {
    throw new Error(`${fieldName} must be a positive integer`);
  }
  return Number(trimmed);
}

function parseNonNegativeInteger(raw: string, fieldName: string): number {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    throw new Error(`${fieldName} must be a non-negative integer`);
  }
  return Number(trimmed);
}
