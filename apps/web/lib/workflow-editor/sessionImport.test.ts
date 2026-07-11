import { importWorkflowFromSessionTasks } from '@/lib/workflow-editor/sessionImport';
import type { TaskPayload } from '@/lib/types';

describe('lib/workflow-editor/sessionImport', () => {
  it('builds ordered agent chain from session text tasks', () => {
    const tasks: TaskPayload[] = [
      {
        id: 'text-2',
        task_kind: 'agent_message',
        message: 'second',
        session_id: 'session-a',
        schedule_type: 'interval',
        interval_seconds: 30,
        enabled: true,
        created_at: '2026-04-05T08:01:00Z',
        updated_at: '2026-04-05T08:01:00Z',
      },
      {
        id: 'text-empty',
        task_kind: 'agent_message',
        message: '   ',
        session_id: 'session-a',
        schedule_type: 'interval',
        interval_seconds: 30,
        enabled: true,
        created_at: '2026-04-05T08:00:30Z',
        updated_at: '2026-04-05T08:00:30Z',
      },
      {
        id: 'text-1',
        task_kind: 'agent_message',
        message: 'first',
        session_id: 'session-a',
        schedule_type: 'interval',
        interval_seconds: 30,
        enabled: true,
        created_at: '2026-04-05T08:00:00Z',
        updated_at: '2026-04-05T08:00:00Z',
      },
    ];

    const imported = importWorkflowFromSessionTasks(tasks, 'session-a');

    expect(imported.nodes.map((node) => node.type)).toEqual(['start', 'agent', 'agent', 'end']);
    expect(imported.nodes[1].agent?.message).toBe('first');
    expect(imported.nodes[2].agent?.message).toBe('second');
    expect(imported.edges).toHaveLength(3);
  });

  it('throws when session has no usable text task', () => {
    expect(() => importWorkflowFromSessionTasks([], 'session-missing')).toThrow(
      'no frontend text tasks found for this session id',
    );
  });
});
