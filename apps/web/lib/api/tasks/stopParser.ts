import type { TaskRunStopResponse } from '@/lib/taskRunStop';
import {
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalString,
  pickKnownKeys,
} from '@/lib/api/shared';
import { parseTaskRunLog } from './parser';

const TASK_RUN_STOP_KEYS = ['status', 'message', 'task_id', 'run_id', 'run'] as const;
const TASK_RUN_STOP_STATUSES = ['stopped', 'not_running'] as const;

export function parseTaskRunStopResponse(payload: unknown): TaskRunStopResponse {
  const record = pickKnownKeys(expectRecord(payload, 'task run stop response'), TASK_RUN_STOP_KEYS);
  return {
    status: expectStringEnum(record.status, TASK_RUN_STOP_STATUSES, 'task run stop response.status'),
    message: expectString(record.message, 'task run stop response.message'),
    task_id: expectString(record.task_id, 'task run stop response.task_id'),
    run_id: parseOptionalString(record.run_id, 'task run stop response.run_id'),
    run: record.run === undefined ? undefined : parseTaskRunLog(record.run),
  };
}
