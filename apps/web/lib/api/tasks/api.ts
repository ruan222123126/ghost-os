import type {
  TaskPatchRequest,
  TaskPayload,
  TaskRunLog,
  TextTaskCreateRequest,
  WorkflowTaskCreateRequest,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseTaskPayload, parseTaskPayloadList, parseTaskRunLogList } from '@/lib/api/tasks/parser';
import { parseTaskRunStopResponse } from '@/lib/api/tasks/stopParser';
import type { TaskRunStopRequest, TaskRunStopResponse } from '@/lib/taskRunStop';

const START_ONLY_QUERY = '?start_only=1';

type UserTaskCreateRequest = TextTaskCreateRequest | WorkflowTaskCreateRequest;

export async function listTasks(): Promise<TaskPayload[]> {
  return requestJSON('/api/tasks', {}, parseTaskPayloadList);
}

export async function getTask(id: string): Promise<TaskPayload> {
  return requestJSON(`/api/tasks/${encodeURIComponent(id)}`, {}, parseTaskPayload);
}

export async function createTask(input: UserTaskCreateRequest): Promise<TaskPayload> {
  return requestJSON('/api/tasks', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseTaskPayload);
}

export async function updateTask(id: string, input: TaskPatchRequest): Promise<TaskPayload> {
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
  await requestJSON(`/api/tasks/${encodeURIComponent(id)}/run${START_ONLY_QUERY}`, {
    method: 'POST',
  });
}

export async function stopTaskRun(id: string, runId: string): Promise<TaskRunStopResponse> {
  const body: TaskRunStopRequest = {
    run_id: runId.trim(),
  };
  return requestJSON(`/api/tasks/${encodeURIComponent(id)}/stop`, {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseTaskRunStopResponse);
}

export async function listTaskLogs(id: string, limit = 20): Promise<TaskRunLog[]> {
  const query = new URLSearchParams();
  query.set('limit', String(limit));
  return requestJSON(`/api/tasks/${encodeURIComponent(id)}/logs?${query.toString()}`, {}, parseTaskRunLogList);
}
