import { requestJSON } from '@/lib/api/client';
import { parseTaskPayload, parseTaskPayloadList, parseTaskRunLogList } from '@/lib/api/tasks/parser';
import { parseTaskRunStopResponse } from '@/lib/api/tasks/stopParser';
import type {
  OrchestrationTaskCreateRequest,
  OrchestrationTaskPayload,
  TaskRunLog,
  TaskUpdateRequest,
} from '@/lib/types';
import type { TaskRunStopRequest, TaskRunStopResponse } from '@/lib/taskRunStop';

const START_ONLY_QUERY = '?start_only=1';

function parseOrchestrationTaskPayload(value: unknown): OrchestrationTaskPayload {
  const payload = parseTaskPayload(value);
  if (payload.task_kind !== 'orchestration') {
    throw new Error('task is not an orchestration');
  }
  return payload;
}

function parseOrchestrationTaskPayloadList(value: unknown): OrchestrationTaskPayload[] {
  return parseTaskPayloadList(value).map((payload) => {
    if (payload.task_kind !== 'orchestration') {
      throw new Error('tasks list contains a non-orchestration payload');
    }
    return payload;
  });
}

export async function listOrchestrations(): Promise<OrchestrationTaskPayload[]> {
  return requestJSON('/api/orchestrations', {}, parseOrchestrationTaskPayloadList);
}

export async function getOrchestration(id: string): Promise<OrchestrationTaskPayload> {
  return requestJSON(`/api/orchestrations/${encodeURIComponent(id)}`, {}, parseOrchestrationTaskPayload);
}

export async function createOrchestration(input: OrchestrationTaskCreateRequest): Promise<OrchestrationTaskPayload> {
  return requestJSON('/api/orchestrations', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseOrchestrationTaskPayload);
}

export async function updateOrchestration(
  id: string,
  input: Pick<TaskUpdateRequest, 'name' | 'task_kind' | 'orchestration' | 'interval_seconds' | 'cron_expr' | 'enabled'>,
): Promise<OrchestrationTaskPayload> {
  return requestJSON(`/api/orchestrations/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  }, parseOrchestrationTaskPayload);
}

export async function deleteOrchestration(id: string): Promise<void> {
  await requestJSON(`/api/orchestrations/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function runOrchestrationNow(id: string): Promise<void> {
  await requestJSON(`/api/orchestrations/${encodeURIComponent(id)}/run${START_ONLY_QUERY}`, { method: 'POST' });
}

export async function stopOrchestrationRun(id: string, runId: string): Promise<TaskRunStopResponse> {
  const body: TaskRunStopRequest = {
    run_id: runId.trim(),
  };
  return requestJSON(`/api/orchestrations/${encodeURIComponent(id)}/stop`, {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseTaskRunStopResponse);
}

export async function listOrchestrationLogs(id: string, limit = 20): Promise<TaskRunLog[]> {
  const query = new URLSearchParams();
  query.set('limit', String(limit));
  return requestJSON(`/api/orchestrations/${encodeURIComponent(id)}/logs?${query.toString()}`, {}, parseTaskRunLogList);
}
