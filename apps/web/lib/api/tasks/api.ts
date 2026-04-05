import type {
  TaskCreateRequest as SharedTaskCreateRequest,
  TaskUpdateRequest as SharedTaskUpdateRequest,
} from '@/lib/envelope.generated';
import type { TaskPayload, TextTaskCreateRequest } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseTaskPayload, parseTaskPayloadList } from '@/lib/api/tasks/parser';

type WorkflowTaskCreateRequest = Extract<SharedTaskCreateRequest, { task_kind: 'workflow' }>;
type TaskCreateRequest = WorkflowTaskCreateRequest | TextTaskCreateRequest;
type TaskUpdateRequest = Pick<
  SharedTaskUpdateRequest,
  'message' | 'session_id' | 'runtime_overrides' | 'interval_seconds' | 'cron_expr' | 'enabled' | 'task_kind' | 'workflow'
>;

export async function listTasks(): Promise<TaskPayload[]> {
  return requestJSON('/api/tasks', {}, parseTaskPayloadList);
}

export async function getTask(id: string): Promise<TaskPayload> {
  return requestJSON(`/api/tasks/${encodeURIComponent(id)}`, {}, parseTaskPayload);
}

export async function createTask(input: TaskCreateRequest): Promise<TaskPayload> {
  return requestJSON('/api/tasks', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseTaskPayload);
}

export async function updateTask(id: string, input: TaskUpdateRequest): Promise<TaskPayload> {
  return requestJSON(`/api/tasks/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  }, parseTaskPayload);
}

export async function deleteTask(id: string): Promise<void> {
  await requestJSON(`/api/tasks/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function runTaskNow(id: string): Promise<void> {
  await requestJSON(`/api/tasks/${encodeURIComponent(id)}/run`, {
    method: 'POST',
  });
}
