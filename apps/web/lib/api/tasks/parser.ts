import type { TaskPayload, TaskRunLog, TaskRunNodeResult } from '@/lib/types';
import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalBoolean,
  parseOptionalNumber,
  parseOptionalRecord,
  parseOptionalStringArray,
  parseOptionalString,
} from '@/lib/api/shared';
import { parseOrchestrationDefinition } from '@/lib/api/tasks/orchestrationParser';
import { parseWorkflowDefinition } from '@/lib/api/tasks/workflowParser';

const TASK_KINDS = ['agent_message', 'workflow', 'orchestration'] as const;
const SCHEDULE_TYPES = ['interval', 'cron'] as const;
const TASK_AGENT_MODES = ['single', 'relay'] as const;
const TASK_RELAY_STOP_POLICIES = ['ai_decides', 'max_rounds'] as const;
const TASK_PAYLOAD_KEYS = [
  'id',
  'name',
  'message',
  'session_id',
  'runtime_overrides',
  'agent_mode',
  'relay',
  'task_kind',
  'workflow',
  'orchestration',
  'schedule_type',
  'interval_seconds',
  'cron_expr',
  'enabled',
  'created_at',
  'updated_at',
  'last_run_at',
  'next_run_at',
  'last_error',
] as const;
const TASK_RUNTIME_OVERRIDE_KEYS = [
  'provider_name',
  'model',
  'system_prompt',
  'preset_id',
  'tool_allowlist',
  'tool_allowlist_only',
  'max_turns',
] as const;
const TASK_RELAY_CONFIG_KEYS = [
  'stop_policy',
  'max_rounds',
  'execution_timeout_ms',
] as const;
const TASK_RUN_STATUS = ['running', 'success', 'incomplete', 'cancelled', 'error', 'skipped', 'awaiting_human'] as const;
const TASK_RUN_LOG_KEYS = [
  'task_id',
  'run_id',
  'trace_id',
  'task_kind',
  'action',
  'scheduled_at',
  'started_at',
  'finished_at',
  'status',
  'session_id_input',
  'session_id_output',
  'response_preview',
  'node_results',
  'error',
] as const;
const TASK_RUN_NODE_RESULT_KEYS = [
  'node_id',
  'node_type',
  'status',
  'started_at',
  'finished_at',
  'completed_seq',
  'branch_id',
  'input',
  'output',
  'preview',
  'error',
] as const;

interface ParsedTaskBase {
  id: string;
  task_kind: 'agent_message' | 'workflow' | 'orchestration';
  schedule_type: 'interval' | 'cron';
  interval_seconds?: number;
  cron_expr?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_run_at?: string;
  next_run_at?: string;
  last_error?: string;
}

interface ParsedTaskRuntimeOverrides {
  provider_name?: string;
  model?: string;
  system_prompt?: string;
  preset_id?: string;
  tool_allowlist?: string[];
  tool_allowlist_only?: boolean;
  max_turns?: number;
}

interface ParsedTaskRelayConfig {
  stop_policy: 'ai_decides' | 'max_rounds';
  max_rounds: number;
  execution_timeout_ms?: number;
}

function parseTaskRuntimeOverrides(value: unknown, label: string): ParsedTaskRuntimeOverrides | undefined {
  const record = parseOptionalRecord(value, label);
  if (record === undefined) {
    return undefined;
  }
  const picked = pickKnownKeys(record, TASK_RUNTIME_OVERRIDE_KEYS);
  const parsed: ParsedTaskRuntimeOverrides = {
    provider_name: parseOptionalString(picked.provider_name, `${label}.provider_name`),
    model: parseOptionalString(picked.model, `${label}.model`),
    system_prompt: parseOptionalString(picked.system_prompt, `${label}.system_prompt`),
    preset_id: parseOptionalString(picked.preset_id, `${label}.preset_id`),
    tool_allowlist: parseOptionalStringArray(picked.tool_allowlist, `${label}.tool_allowlist`),
    tool_allowlist_only: parseOptionalBoolean(picked.tool_allowlist_only, `${label}.tool_allowlist_only`),
    max_turns: parseOptionalNumber(picked.max_turns, `${label}.max_turns`),
  };
  if (
    !parsed.provider_name
    && !parsed.model
    && !parsed.system_prompt
    && !parsed.preset_id
    && (!parsed.tool_allowlist || parsed.tool_allowlist.length === 0)
    && parsed.tool_allowlist_only === undefined
    && parsed.max_turns === undefined
  ) {
    return undefined;
  }
  return parsed;
}

function parseTaskRelayConfig(value: unknown, label: string): ParsedTaskRelayConfig | undefined {
  const record = parseOptionalRecord(value, label);
  if (record === undefined) {
    return undefined;
  }
  const picked = pickKnownKeys(record, TASK_RELAY_CONFIG_KEYS);
  const stopPolicy = expectStringEnum(picked.stop_policy, TASK_RELAY_STOP_POLICIES, `${label}.stop_policy`);

  return {
    stop_policy: stopPolicy,
    max_rounds: expectNumber(picked.max_rounds, `${label}.max_rounds`),
    execution_timeout_ms: parseOptionalNumber(
      picked.execution_timeout_ms,
      `${label}.execution_timeout_ms`,
    ),
  };
}

function parseTaskBase(record: Record<string, unknown>, label: string): ParsedTaskBase {
  const parsed: ParsedTaskBase = {
    id: expectString(record.id, `${label}.id`),
    task_kind: expectStringEnum(record.task_kind, TASK_KINDS, `${label}.task_kind`),
    schedule_type: expectStringEnum(record.schedule_type, SCHEDULE_TYPES, `${label}.schedule_type`),
    interval_seconds: parseOptionalNumber(record.interval_seconds, `${label}.interval_seconds`),
    cron_expr: parseOptionalString(record.cron_expr, `${label}.cron_expr`),
    enabled: expectBoolean(record.enabled, `${label}.enabled`),
    created_at: expectString(record.created_at, `${label}.created_at`),
    updated_at: expectString(record.updated_at, `${label}.updated_at`),
    last_run_at: parseOptionalString(record.last_run_at, `${label}.last_run_at`),
    next_run_at: parseOptionalString(record.next_run_at, `${label}.next_run_at`),
    last_error: parseOptionalString(record.last_error, `${label}.last_error`),
  };
  if (parsed.schedule_type === 'interval' && parsed.interval_seconds === undefined) {
    throw new Error(`Invalid ${label}.interval_seconds: expected number for interval schedule`);
  }
  if (parsed.schedule_type === 'cron' && !parsed.cron_expr) {
    throw new Error(`Invalid ${label}.cron_expr: expected non-empty string for cron schedule`);
  }
  return parsed;
}

function parseTaskPayloadWithLabel(value: unknown, label: string): TaskPayload {
  const record = pickKnownKeys(expectRecord(value, label), TASK_PAYLOAD_KEYS);
  const base = parseTaskBase(record, label);

  if (base.task_kind === 'agent_message') {
    return {
      ...base,
      message: expectString(record.message, `${label}.message`),
      session_id: parseOptionalString(record.session_id, `${label}.session_id`),
      runtime_overrides: parseTaskRuntimeOverrides(record.runtime_overrides, `${label}.runtime_overrides`),
      agent_mode: parseOptionalAgentMode(record.agent_mode, `${label}.agent_mode`),
      relay: parseTaskRelayConfig(record.relay, `${label}.relay`),
    } as TaskPayload;
  }
  if (base.task_kind === 'orchestration') {
    return {
      ...base,
      name: expectString(record.name, `${label}.name`),
      orchestration: parseOrchestrationDefinition(record.orchestration, `${label}.orchestration`),
    } as TaskPayload;
  }
  return {
    ...base,
    workflow: parseWorkflowDefinition(record.workflow, `${label}.workflow`),
  } as TaskPayload;
}

function parseOptionalAgentMode(value: unknown, label: string): 'single' | 'relay' | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  return expectStringEnum(value, TASK_AGENT_MODES, label);
}

export function parseTaskPayload(payload: unknown): TaskPayload {
  return parseTaskPayloadWithLabel(payload, 'task');
}

export function parseTaskPayloadList(payload: unknown): TaskPayload[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid tasks list: expected array');
  }
  return payload.map((task, index) => parseTaskPayloadWithLabel(task, `tasks list[${index}]`));
}

function parseTaskRunNodeResult(value: unknown, label: string): TaskRunNodeResult {
  const record = pickKnownKeys(expectRecord(value, label), TASK_RUN_NODE_RESULT_KEYS);
  return {
    node_id: expectString(record.node_id, `${label}.node_id`),
    node_type: expectString(record.node_type, `${label}.node_type`),
    status: expectString(record.status, `${label}.status`),
    started_at: parseOptionalString(record.started_at, `${label}.started_at`),
    finished_at: parseOptionalString(record.finished_at, `${label}.finished_at`),
    completed_seq: parseOptionalNumber(record.completed_seq, `${label}.completed_seq`),
    branch_id: parseOptionalString(record.branch_id, `${label}.branch_id`),
    input: record.input,
    output: record.output,
    preview: parseOptionalString(record.preview, `${label}.preview`),
    error: parseOptionalString(record.error, `${label}.error`),
  };
}

function parseTaskRunNodeResultList(value: unknown, label: string): TaskRunNodeResult[] | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }
  return value.map((item, index) => parseTaskRunNodeResult(item, `${label}[${index}]`));
}

function parseTaskRunLogWithLabel(value: unknown, label: string): TaskRunLog {
  const record = pickKnownKeys(expectRecord(value, label), TASK_RUN_LOG_KEYS);
  const rawTaskKind = parseOptionalString(record.task_kind, `${label}.task_kind`);
  if (
    rawTaskKind !== undefined
    && rawTaskKind !== 'agent_message'
    && rawTaskKind !== 'workflow'
    && rawTaskKind !== 'orchestration'
  ) {
    throw new Error(`Invalid ${label}.task_kind: unexpected value "${rawTaskKind}"`);
  }
  return {
    task_id: expectString(record.task_id, `${label}.task_id`),
    run_id: expectString(record.run_id, `${label}.run_id`),
    trace_id: expectString(record.trace_id, `${label}.trace_id`),
    task_kind: rawTaskKind,
    action: parseOptionalString(record.action, `${label}.action`),
    scheduled_at: expectString(record.scheduled_at, `${label}.scheduled_at`),
    started_at: parseOptionalString(record.started_at, `${label}.started_at`),
    finished_at: parseOptionalString(record.finished_at, `${label}.finished_at`),
    status: expectStringEnum(record.status, TASK_RUN_STATUS, `${label}.status`),
    session_id_input: parseOptionalString(record.session_id_input, `${label}.session_id_input`),
    session_id_output: parseOptionalString(record.session_id_output, `${label}.session_id_output`),
    response_preview: parseOptionalString(record.response_preview, `${label}.response_preview`),
    node_results: parseTaskRunNodeResultList(record.node_results, `${label}.node_results`),
    error: parseOptionalString(record.error, `${label}.error`),
  };
}

export function parseTaskRunLog(payload: unknown): TaskRunLog {
  return parseTaskRunLogWithLabel(payload, 'task run log');
}

export function parseTaskRunLogList(payload: unknown): TaskRunLog[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid task run logs list: expected array');
  }
  return payload.map((item, index) => parseTaskRunLogWithLabel(item, `task run logs[${index}]`));
}
