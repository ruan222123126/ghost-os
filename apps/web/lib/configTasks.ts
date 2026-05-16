import type {
  TaskUpdateRequest as SharedTaskUpdateRequest,
} from '@/lib/envelope.generated';
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
const DEFAULT_RELAY_MAX_ROUNDS = 20;
const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = 0;
const DEFAULT_RELAY_STOP_POLICY: TaskRelayStopPolicy = 'ai_decides';

type TaskUpdateRequest = Pick<
  SharedTaskUpdateRequest,
  | 'message'
  | 'session_id'
  | 'runtime_overrides'
  | 'agent_mode'
  | 'interval_seconds'
  | 'cron_expr'
  | 'task_kind'
> | TextTaskUpdateRequest;

export type TaskEditorMode = 'create' | 'edit';
export type TaskScheduleMode = 'interval' | 'cron';
export type TaskAgentMode = 'single' | 'relay';
export type TaskRelayStopPolicy = TaskRelayConfig['stop_policy'];
export type TaskSettingsListItem = AgentMessageTaskPayload | WorkflowTaskPayload;

export interface TaskEditorState {
  message: string;
  scheduleMode: TaskScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
  sessionId: string;
  agentMode: TaskAgentMode;
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
    message: '',
    scheduleMode: 'interval',
    intervalSeconds: '300',
    cronExpr: '',
    sessionId: '',
    agentMode: 'single',
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

  return {
    message: task.message,
    scheduleMode: task.schedule_type,
    intervalSeconds: task.interval_seconds === undefined ? '' : String(task.interval_seconds),
    cronExpr: task.cron_expr ?? '',
    sessionId: task.session_id ?? '',
    agentMode: 'single',
    relayStopPolicy: relay?.stop_policy ?? DEFAULT_RELAY_STOP_POLICY,
    relayMaxRounds: String(relay?.max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    relayExecutionTimeoutMS: String(
      relay?.execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
    runtimeOverridesEnabled: runtimeOverrides !== undefined,
    runtimePresetId: runtimeOverrides?.preset_id ?? '',
    runtimeModel: runtimeOverrides?.model ?? '',
    runtimeToolAllowlist: (runtimeOverrides?.tool_allowlist ?? []).join(', '),
  };
}

export function taskCreateRequestFromEditor(editor: TaskEditorState): TextTaskCreateRequest {
  const runtimeOverrides = runtimeOverridesFromEditor(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    ...textTaskAgentFields(),
    runtime_overrides: runtimeOverrides,
  };
}

export function taskUpdateRequestFromEditor(editor: TaskEditorState): TaskUpdateRequest {
  const runtimeOverrides = runtimeOverridesForUpdate(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    ...textTaskAgentFields(),
    runtime_overrides: runtimeOverrides,
  };
}

export function isTaskSettingsTextTask(task: TaskPayload): task is AgentMessageTaskPayload {
  return task.task_kind === 'agent_message' && task.agent_mode !== 'relay';
}

export function isTaskSettingsListItem(task: TaskPayload): task is TaskSettingsListItem {
  return isTaskSettingsTextTask(task) || task.task_kind === 'workflow';
}

export function filterTaskSettingsTasks(tasks: TaskPayload[]): TaskSettingsListItem[] {
  return tasks.filter(isTaskSettingsListItem);
}

function taskMutationFieldsFromEditor(editor: TaskEditorState): {
  message: string;
  session_id: string;
  interval_seconds?: number;
  cron_expr?: string;
} {
  const message = editor.message.trim();
  if (message.length === 0) {
    throw new Error('message is required');
  }

  return {
    message,
    session_id: editor.sessionId.trim(),
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
