import {
  emptyTaskEditorState,
  taskCreateRequestFromEditor,
} from '@/lib/configTasks';
import type { TaskPayload } from '@/lib/types';

describe('lib/configTasks', () => {
  it('builds workflow task from frontend text tasks in selected session', () => {
    const editor = {
      ...emptyTaskEditorState,
      taskKind: 'workflow' as const,
      workflowSessionId: 'session-1',
      scheduleMode: 'interval' as const,
      intervalSeconds: '120',
    };

    const tasks: TaskPayload[] = [
      {
        id: 'text-1',
        task_kind: 'agent_message',
        message: 'step one',
        session_id: 'session-1',
        schedule_type: 'interval',
        interval_seconds: 60,
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T07:00:00Z',
      },
      {
        id: 'text-2',
        task_kind: 'agent_message',
        message: 'step two',
        session_id: 'session-1',
        schedule_type: 'interval',
        interval_seconds: 60,
        enabled: true,
        created_at: '2026-04-05T07:01:00Z',
        updated_at: '2026-04-05T07:01:00Z',
      },
    ];

    const request = taskCreateRequestFromEditor(editor, tasks);

    expect(request).toEqual({
      task_kind: 'workflow',
      interval_seconds: 120,
      workflow: {
        nodes: [
          { id: 'start-node', type: 'start' },
          { id: 'agent-node-1', type: 'agent', agent: { message: 'step one' } },
          { id: 'agent-node-2', type: 'agent', agent: { message: 'step two' } },
          { id: 'end-node', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start-node', to_node_id: 'agent-node-1' },
          { from_node_id: 'agent-node-1', to_node_id: 'agent-node-2' },
          { from_node_id: 'agent-node-2', to_node_id: 'end-node' },
        ],
      },
    });
  });

  it('throws when selected session has no text tasks', () => {
    const editor = {
      ...emptyTaskEditorState,
      taskKind: 'workflow' as const,
      workflowSessionId: 'missing-session',
      scheduleMode: 'cron' as const,
      cronExpr: '*/5 * * * *',
    };

    expect(() => taskCreateRequestFromEditor(editor, [])).toThrow(
      'no frontend tasks found for this session id',
    );
  });
});
