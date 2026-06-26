import {
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
  DEFAULT_RELAY_MAX_ROUNDS,
  DEFAULT_RELAY_STOP_POLICY,
} from '@/lib/configTaskDefaults';
import type {
  AgentMessageTaskPayload,
  BridgeConfig,
  TaskRelayConfig,
  TaskRuntimeOverrides,
} from '@/lib/types';

export type TaskEditorMode = 'create' | 'edit';
export type TaskEditorType = 'text' | 'loop';
export type TaskScheduleMode = 'interval' | 'cron';
export type TaskRelayStopPolicy = TaskRelayConfig['stop_policy'];

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

type TaskScheduleEditorFields = Pick<
  TaskEditorState,
  'scheduleMode' | 'intervalSeconds' | 'cronExpr'
>;
type TaskRelayEditorFields = Pick<
  TaskEditorState,
  'relayStopPolicy' | 'relayMaxRounds' | 'relayExecutionTimeoutMS'
>;
type TaskRuntimeEditorFields = Pick<
  TaskEditorState,
  'runtimeOverridesEnabled' | 'runtimePresetId' | 'runtimeModel' | 'runtimeToolAllowlist'
>;

export const emptyTaskEditorState = createTaskEditorState(null);

export function createTaskEditorState(config: BridgeConfig | null): TaskEditorState {
  return {
    taskType: 'text',
    message: '',
    scheduleMode: 'interval',
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    cronExpr: '',
    sessionId: '',
    relayStopPolicy: relayStopPolicyFromConfig(config),
    relayMaxRounds: relayMaxRoundsFromConfig(config),
    relayExecutionTimeoutMS: relayExecutionTimeoutMSFromConfig(config),
    runtimeOverridesEnabled: false,
    runtimePresetId: '',
    runtimeModel: '',
    runtimeToolAllowlist: '',
  };
}

export function editorStateFromTask(task: AgentMessageTaskPayload): TaskEditorState {
  const taskType = editorTaskTypeFromTask(task);

  return {
    taskType,
    message: task.message,
    ...scheduleEditorFieldsFromTask(task),
    sessionId: sessionIDFromTask(task, taskType),
    ...relayEditorFieldsFromTask(task.relay),
    ...runtimeEditorFieldsFromTask(task.runtime_overrides, taskType),
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

function editorTaskTypeFromTask(task: AgentMessageTaskPayload): TaskEditorType {
  return task.agent_mode === 'relay' ? 'loop' : 'text';
}

function scheduleEditorFieldsFromTask(task: AgentMessageTaskPayload): TaskScheduleEditorFields {
  return {
    scheduleMode: task.schedule_type,
    intervalSeconds: intervalSecondsFromTask(task),
    cronExpr: task.cron_expr ?? '',
  };
}

function intervalSecondsFromTask(task: AgentMessageTaskPayload): string {
  if (task.interval_seconds === undefined) {
    return DEFAULT_INTERVAL_SECONDS;
  }
  return String(task.interval_seconds);
}

function relayStopPolicyFromConfig(config: BridgeConfig | null): TaskRelayStopPolicy {
  return config?.relay_default_stop_policy ?? DEFAULT_RELAY_STOP_POLICY;
}

function relayMaxRoundsFromConfig(config: BridgeConfig | null): string {
  return String(config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS);
}

function relayExecutionTimeoutMSFromConfig(config: BridgeConfig | null): string {
  return String(
    config?.relay_default_execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
  );
}

function sessionIDFromTask(
  task: AgentMessageTaskPayload,
  taskType: TaskEditorType,
): string {
  if (taskType === 'loop') {
    return '';
  }
  return task.session_id ?? '';
}

function relayEditorFieldsFromTask(
  relay: TaskRelayConfig | undefined,
): TaskRelayEditorFields {
  return {
    relayStopPolicy: relayStopPolicyFromTaskRelay(relay),
    relayMaxRounds: relayMaxRoundsFromTaskRelay(relay),
    relayExecutionTimeoutMS: relayExecutionTimeoutMSFromTaskRelay(relay),
  };
}

function relayStopPolicyFromTaskRelay(relay: TaskRelayConfig | undefined): TaskRelayStopPolicy {
  return relay?.stop_policy ?? DEFAULT_RELAY_STOP_POLICY;
}

function relayMaxRoundsFromTaskRelay(relay: TaskRelayConfig | undefined): string {
  return String(relay?.max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS);
}

function relayExecutionTimeoutMSFromTaskRelay(relay: TaskRelayConfig | undefined): string {
  return String(relay?.execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS);
}

function runtimeEditorFieldsFromTask(
  runtimeOverrides: TaskRuntimeOverrides | undefined,
  taskType: TaskEditorType,
): TaskRuntimeEditorFields {
  if (taskType === 'loop') {
    return loopRuntimeEditorFields(runtimeOverrides);
  }
  return textRuntimeEditorFields(runtimeOverrides);
}

function textRuntimeEditorFields(
  runtimeOverrides: TaskRuntimeOverrides | undefined,
): TaskRuntimeEditorFields {
  return {
    runtimeOverridesEnabled: runtimeOverrides !== undefined,
    runtimePresetId: runtimePresetIDFromOverrides(runtimeOverrides),
    runtimeModel: runtimeOverrides?.model ?? '',
    runtimeToolAllowlist: (runtimeOverrides?.tool_allowlist ?? []).join(', '),
  };
}

function loopRuntimeEditorFields(
  runtimeOverrides: TaskRuntimeOverrides | undefined,
): TaskRuntimeEditorFields {
  return {
    runtimeOverridesEnabled: false,
    runtimePresetId: runtimePresetIDFromOverrides(runtimeOverrides),
    runtimeModel: '',
    runtimeToolAllowlist: '',
  };
}

function runtimePresetIDFromOverrides(
  runtimeOverrides: TaskRuntimeOverrides | undefined,
): string {
  return runtimeOverrides?.preset_id ?? '';
}
