import type { TaskPayload } from '@/lib/types';
import {
  expectBoolean,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalNumber,
  parseOptionalRecord,
  parseOptionalStringArray,
  parseOptionalString,
} from '@/lib/api/shared';
import { parseWorkflowDefinition } from '@/lib/api/tasks/workflowParser';

const TASK_KINDS = ['agent_message', 'workflow'] as const;
const SCHEDULE_TYPES = ['interval', 'cron'] as const;
const TASK_PAYLOAD_KEYS = [
  'id',
  'message',
  'session_id',
  'runtime_overrides',
  'task_kind',
  'workflow',
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
const TASK_RUNTIME_OVERRIDE_KEYS = ['model', 'tool_allowlist'] as const;

interface ParsedTaskBase {
  id: string;
  task_kind: 'agent_message' | 'workflow';
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
  model?: string;
  tool_allowlist?: string[];
}

function parseTaskRuntimeOverrides(value: unknown, label: string): ParsedTaskRuntimeOverrides | undefined {
  const record = parseOptionalRecord(value, label);
  if (record === undefined) {
    return undefined;
  }
  const picked = pickKnownKeys(record, TASK_RUNTIME_OVERRIDE_KEYS);
  const parsed: ParsedTaskRuntimeOverrides = {
    model: parseOptionalString(picked.model, `${label}.model`),
    tool_allowlist: parseOptionalStringArray(picked.tool_allowlist, `${label}.tool_allowlist`),
  };
  if (!parsed.model && (!parsed.tool_allowlist || parsed.tool_allowlist.length === 0)) {
    return undefined;
  }
  return parsed;
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
    } as TaskPayload;
  }
  return {
    ...base,
    workflow: parseWorkflowDefinition(record.workflow, `${label}.workflow`),
  } as TaskPayload;
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
