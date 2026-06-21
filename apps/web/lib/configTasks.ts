import type {
  AgentMessageTaskPayload,
  BridgeConfig,
  TaskPayload,
  TaskRelayConfig,
  TaskRuntimeOverrides,
  TextTaskCreateRequest,
  TextTaskUpdateRequest,
  WorkflowTaskPayload,
} from '@/lib/types';

const MIN_INTERVAL_SECONDS = 1;
const DECIMAL_RADIX = 10;
const TOOL_ALLOWLIST_SEPARATOR_PATTERN = /[\n,]/;
const DEFAULT_INTERVAL_SECONDS = '300';
const DEFAULT_RELAY_MAX_ROUNDS = 20;
const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = 0;
const DEFAULT_RELAY_STOP_POLICY: TaskRelayStopPolicy = 'ai_decides';

export type TaskEditorMode = 'create' | 'edit';
export type TaskEditorType = 'text' | 'loop';
export type TaskScheduleMode = 'interval' | 'cron';
export type TaskRelayStopPolicy = TaskRelayConfig['stop_policy'];
export type TaskSettingsListItem = AgentMessageTaskPayload | WorkflowTaskPayload;

export interface TaskEditorState {
  taskType: TaskEditorType;
  message: string;
  scheduleMode: TaskScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
  sessionId: string;
  relayStopPolicy: TaskRelayStopPolicy;
  relayMaxRounds: string;
  relayExecutionTimeoutMS: string;
  runtimeOverridesEnabled: boolean;
  runtimePresetId: string;
  runtimeModel: string;
  runtimeToolAllowlist: string;
}

export const emptyTaskEditorState = createTaskEditorState(null);

export function createTaskEditorState(config: BridgeConfig | null): TaskEditorState {
  return {
    taskType: 'text',
    message: '',
    scheduleMode: 'interval',
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    cronExpr: '',
    sessionId: '',
    relayStopPolicy: config?.relay_default_stop_policy ?? DEFAULT_RELAY_STOP_POLICY,
    relayMaxRounds: String(config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    relayExecutionTimeoutMS: String(
      config?.relay_default_execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
    runtimeOverridesEnabled: false,
    runtimePresetId: '',
    runtimeModel: '',
    runtimeToolAllowlist: '',
  };
}

export function editorStateFromTask(task: AgentMessageTaskPayload): TaskEditorState {
  const runtimeOverrides = task.runtime_overrides;
  const relay = task.relay;
  const taskType = task.agent_mode === 'relay' ? 'loop' : 'text';

  return {
    taskType,
    message: task.message,
    scheduleMode: task.schedule_type,
    intervalSeconds: task.interval_seconds === undefined ? DEFAULT_INTERVAL_SECONDS : String(task.interval_seconds),
    cronExpr: task.cron_expr ?? '',
    sessionId: taskType === 'text' ? task.session_id ?? '' : '',
    relayStopPolicy: relay?.stop_policy ?? DEFAULT_RELAY_STOP_POLICY,
    relayMaxRounds: String(relay?.max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    relayExecutionTimeoutMS: String(
      relay?.execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
    runtimeOverridesEnabled: taskType === 'text' && runtimeOverrides !== undefined,
    runtimePresetId: runtimeOverrides?.preset_id ?? '',
    runtimeModel: taskType === 'text' ? runtimeOverrides?.model ?? '' : '',
    runtimeToolAllowlist: taskType === 'text'
      ? (runtimeOverrides?.tool_allowlist ?? []).join(', ')
      : '',
  };
}

export function editorStateWithTaskType(
  editor: TaskEditorState,
  taskType: TaskEditorType,
  config: BridgeConfig | null,
): TaskEditorState {
  const defaults = createTaskEditorState(config);
  return {
    ...defaults,
    taskType,
    message: editor.message,
    scheduleMode: editor.scheduleMode,
    intervalSeconds: editor.intervalSeconds,
    cronExpr: editor.cronExpr,
  };
}

export function taskCreateRequestFromEditor(editor: TaskEditorState): TextTaskCreateRequest {
  if (editor.taskType === 'loop') {
    return loopCreateRequestFromEditor(editor);
  }

  return {
    task_kind: 'agent_message',
    ...textTaskMutationFieldsFromEditor(editor),
    ...textTaskAgentFields(),
    runtime_overrides: runtimeOverridesFromEditor(editor),
  };
}

export function taskUpdateRequestFromEditor(editor: TaskEditorState): TextTaskUpdateRequest {
  if (editor.taskType === 'loop') {
    return loopUpdateRequestFromEditor(editor);
  }

  return {
    task_kind: 'agent_message',
    ...textTaskMutationFieldsFromEditor(editor),
    ...textTaskAgentFields(),
    runtime_overrides: runtimeOverridesForUpdate(editor),
  };
}

export function isTaskSettingsListItem(task: TaskPayload): task is TaskSettingsListItem {
  return task.task_kind === 'agent_message' || task.task_kind === 'workflow';
}

export function filterTaskSettingsTasks(tasks: TaskPayload[]): TaskSettingsListItem[] {
  return tasks.filter(isTaskSettingsListItem);
}

function textTaskMutationFieldsFromEditor(editor: TaskEditorState): {
  message: string;
  session_id: string;
  interval_seconds?: number;
  cron_expr?: string;
} {
  return {
    ...commonTaskMutationFieldsFromEditor(editor),
    session_id: editor.sessionId.trim(),
  };
}

function loopCreateRequestFromEditor(editor: TaskEditorState): TextTaskCreateRequest {
  return {
    task_kind: 'agent_message',
    ...commonTaskMutationFieldsFromEditor(editor),
    ...loopTaskAgentFields(editor),
    runtime_overrides: runtimeOverridesFromLoopPreset(editor.runtimePresetId),
  };
}

function loopUpdateRequestFromEditor(editor: TaskEditorState): TextTaskUpdateRequest {
  return {
    task_kind: 'agent_message',
    ...commonTaskMutationFieldsFromEditor(editor),
    ...loopTaskAgentFields(editor),
    runtime_overrides: runtimeOverridesFromLoopPreset(editor.runtimePresetId) ?? {},
  };
}

function commonTaskMutationFieldsFromEditor(editor: TaskEditorState): {
  message: string;
  interval_seconds?: number;
  cron_expr?: string;
} {
  const message = editor.message.trim();
  if (message.length === 0) {
    throw new Error('message is required');
  }

  return {
    message,
    ...scheduleFieldsFromEditor(editor),
  };
}

function scheduleFieldsFromEditor(editor: TaskEditorState): {
  interval_seconds?: number;
  cron_expr?: string;
} {
  if (editor.scheduleMode === 'interval') {
    return {
      interval_seconds: parseIntervalSeconds(editor.intervalSeconds),
    };
  }

  const cronExpr = editor.cronExpr.trim();
  if (!cronExpr) {
    throw new Error('cron expression is required');
  }

  return {
    cron_expr: cronExpr,
  };
}

function textTaskAgentFields(): { agent_mode: 'single' } {
  return { agent_mode: 'single' };
}

function loopTaskAgentFields(editor: TaskEditorState): {
  agent_mode: 'relay';
  relay: TaskRelayConfig;
} {
  return {
    agent_mode: 'relay',
    relay: {
      stop_policy: editor.relayStopPolicy,
      max_rounds: parsePositiveInteger(editor.relayMaxRounds, 'relay max_rounds'),
      execution_timeout_ms: parseNonNegativeInteger(
        editor.relayExecutionTimeoutMS,
        'relay execution_timeout_ms',
      ),
    },
  };
}

function runtimeOverridesForUpdate(
  editor: TaskEditorState,
): TaskRuntimeOverrides | Record<string, never> {
  if (!editor.runtimeOverridesEnabled) {
    return {};
  }

  return runtimeOverridesFromEditor(editor) ?? {};
}

function runtimeOverridesFromEditor(editor: TaskEditorState): TaskRuntimeOverrides | undefined {
  if (!editor.runtimeOverridesEnabled) {
    return undefined;
  }
  const model = editor.runtimeModel.trim();
  const presetId = editor.runtimePresetId.trim();
  const toolAllowlist = parseToolAllowlist(editor.runtimeToolAllowlist);
  if (model === '' && presetId === '' && toolAllowlist.length === 0) {
    return undefined;
  }

  const presetSelected = presetId !== '';
  return {
    model: model === '' ? undefined : model,
    preset_id: presetSelected ? presetId : undefined,
    tool_allowlist_only: presetSelected ? true : undefined,
    tool_allowlist: runtimeToolAllowlistValue(toolAllowlist, presetSelected),
  };
}

function runtimeToolAllowlistValue(
  toolAllowlist: string[],
  presetSelected: boolean,
): string[] | undefined {
  if (toolAllowlist.length > 0) {
    return toolAllowlist;
  }
  if (presetSelected) {
    return [];
  }
  return undefined;
}

function runtimeOverridesFromLoopPreset(presetId: string): TaskRuntimeOverrides | undefined {
  const id = presetId.trim();
  if (id.length === 0) {
    return undefined;
  }
  return { preset_id: id };
}

function parseToolAllowlist(raw: string): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const chunk of raw.split(TOOL_ALLOWLIST_SEPARATOR_PATTERN)) {
    const name = chunk.trim();
    if (name === '' || seen.has(name)) {
      continue;
    }
    seen.add(name);
    out.push(name);
  }

  return out;
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
