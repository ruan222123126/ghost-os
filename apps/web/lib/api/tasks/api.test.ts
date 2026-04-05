import { createTask, deleteTask, getTask, listTasks, runTaskNow, updateTask } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import type { AgentMessageTaskPayload, TextTaskCreateRequest, TextTaskUpdateRequest } from '@/lib/types';

describe('lib/api/tasks/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('listTasks calls GET /api/tasks', async () => {
    const expected: AgentMessageTaskPayload = {
      id: 'task-1',
      message: 'Daily summary',
      task_kind: 'agent_message',
      schedule_type: 'interval',
      interval_seconds: 300,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T07:00:00Z',
    };

    mockFetchJSON({
      status: 'success',
      payload: [expected],
      error: '',
    });

    const tasks = await listTasks();

    expect(tasks).toEqual([expected]);
    expect(fetchMock).toHaveBeenCalledWith('/api/tasks', expect.any(Object));
  });

  it('getTask calls GET /api/tasks/:id', async () => {
    const expected: AgentMessageTaskPayload = {
      id: 'task-get-1',
      message: 'Load task details',
      task_kind: 'agent_message',
      schedule_type: 'interval',
      interval_seconds: 120,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T07:00:00Z',
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const task = await getTask('task-get-1');

    expect(task).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/tasks/task-get-1', expect.any(Object));
  });

  it('createTask calls POST /api/tasks with task body', async () => {
    const input: TextTaskCreateRequest = {
      task_kind: 'agent_message',
      message: 'Create customer follow-up',
      interval_seconds: 600,
      session_id: '',
    };

    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'task-2',
        message: 'Create customer follow-up',
        task_kind: 'agent_message',
        schedule_type: 'interval',
        interval_seconds: 600,
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T07:00:00Z',
      },
      error: '',
    });

    await createTask(input);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tasks',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      }),
    );
  });

  it('updateTask calls PATCH /api/tasks/:id', async () => {
    const input: TextTaskUpdateRequest = {
      message: 'Updated briefing',
      cron_expr: '*/10 * * * *',
      session_id: '',
      task_kind: 'agent_message',
    };

    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'task-3',
        message: 'Updated briefing',
        task_kind: 'agent_message',
        schedule_type: 'cron',
        cron_expr: '*/10 * * * *',
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T08:00:00Z',
      },
      error: '',
    });

    await updateTask('task-3', input);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tasks/task-3',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    );
  });

  it('deleteTask calls DELETE /api/tasks/:id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { id: 'task-4', deleted: true },
      error: '',
    });

    await deleteTask('task-4');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tasks/task-4',
      expect.objectContaining({
        method: 'DELETE',
      }),
    );
  });

  it('runTaskNow calls POST /api/tasks/:id/run', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        task: {
          id: 'task-5',
        },
        run: {
          run_id: 'run-1',
        },
      },
      error: '',
    });

    await runTaskNow('task-5');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tasks/task-5/run',
      expect.objectContaining({
        method: 'POST',
      }),
    );
  });
});
