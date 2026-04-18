import { parseTaskPayload, parseTaskPayloadList } from './parser';
import type { AgentMessageTaskPayload } from '@/lib/types';

describe('lib/api/tasks/parser', () => {
  it('parses task list payloads', () => {
    const expected: AgentMessageTaskPayload = {
      id: 'task-1',
      message: 'Daily sync',
      session_id: 'session-1',
      runtime_overrides: {
        model: 'gpt-5.4',
        tool_allowlist: ['script_exec', 'web_search'],
      },
      task_kind: 'agent_message',
      schedule_type: 'interval',
      interval_seconds: 300,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T07:00:00Z',
    };

    const tasks = parseTaskPayloadList([
      expected,
      {
        id: 'workflow-1',
        task_kind: 'workflow',
        schedule_type: 'cron',
        cron_expr: '*/5 * * * *',
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T07:00:00Z',
        workflow: {
          nodes: [
            { id: 'start', type: 'start' },
            { id: 'end', type: 'end' },
          ],
          edges: [{ from_node_id: 'start', to_node_id: 'end' }],
        },
      },
    ]);

    expect(tasks[0]).toEqual(expected);
    expect(tasks).toHaveLength(2);
  });

  it('rejects unknown fields on task payloads', () => {
    const expected: AgentMessageTaskPayload = {
      id: 'task-2',
      message: 'Weekly report',
      task_kind: 'agent_message',
      schedule_type: 'cron',
      cron_expr: '0 9 * * 1',
      enabled: false,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T08:00:00Z',
      last_error: 'session not found',
    };

    expect(() => {
      parseTaskPayload({
        ...expected,
        future_field: 'ignored',
        action_params: {
          extra: 'ignored',
        },
      });
    }).toThrow('Invalid payload: unexpected field "future_field"');
  });

  it('parses workflow start inputs with typed defaults', () => {
    const parsed = parseTaskPayload({
      id: 'workflow-with-inputs',
      task_kind: 'workflow',
      schedule_type: 'interval',
      interval_seconds: 60,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T08:00:00Z',
      workflow: {
        nodes: [
          {
            id: 'start',
            type: 'start',
            start: {
              inputs: [
                {
                  name: 'title',
                  type: 'string',
                  required: true,
                  default: 'Weekly digest',
                  description: 'Briefing title',
                },
                {
                  name: 'filters',
                  type: 'object',
                  default: { importance: 'high' },
                },
              ],
            },
          },
          { id: 'end', type: 'end' },
        ],
        edges: [{ from_node_id: 'start', to_node_id: 'end' }],
      },
    });

    expect(parsed.task_kind).toBe('workflow');
    if (parsed.task_kind !== 'workflow') {
      throw new Error('Expected workflow task payload');
    }
    expect(parsed.workflow.nodes[0]).toMatchObject({
      id: 'start',
      type: 'start',
      start: {
        inputs: [
          {
            name: 'title',
            type: 'string',
            required: true,
            default: 'Weekly digest',
            description: 'Briefing title',
          },
          {
            name: 'filters',
            type: 'object',
            default: { importance: 'high' },
          },
        ],
      },
    });
  });

  it('parses workflow if/loop node payloads', () => {
    const parsed = parseTaskPayload({
      id: 'workflow-if-loop',
      task_kind: 'workflow',
      schedule_type: 'interval',
      interval_seconds: 60,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T08:00:00Z',
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          {
            id: 'if-1',
            type: 'if',
            if: {
              source_node_id: 'tool-1',
              operator: 'contains',
              value: 'ok',
              true_node_id: 'loop-1',
              false_node_id: 'end',
            },
          },
          {
            id: 'loop-1',
            type: 'loop',
            loop: {
              max_iterations: 3,
              body_node_id: 'tool-1',
              exit_node_id: 'end',
            },
          },
          { id: 'tool-1', type: 'tool', tool: { tool_name: 'script_exec', arguments: { command: 'pwd' } } },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'if-1' },
          { from_node_id: 'if-1', to_node_id: 'loop-1' },
          { from_node_id: 'if-1', to_node_id: 'end' },
          { from_node_id: 'loop-1', to_node_id: 'tool-1' },
          { from_node_id: 'loop-1', to_node_id: 'end' },
          { from_node_id: 'tool-1', to_node_id: 'loop-1' },
        ],
      },
    });

    expect(parsed.task_kind).toBe('workflow');
    if (parsed.task_kind !== 'workflow') {
      throw new Error('Expected workflow task payload');
    }
    expect(parsed.workflow.nodes.find((node) => node.type === 'if')).toMatchObject({
      id: 'if-1',
      if: {
        operator: 'contains',
        true_node_id: 'loop-1',
        false_node_id: 'end',
      },
    });
    expect(parsed.workflow.nodes.find((node) => node.type === 'loop')).toMatchObject({
      id: 'loop-1',
      loop: {
        max_iterations: 3,
        body_node_id: 'tool-1',
        exit_node_id: 'end',
      },
    });
  });

  it('parses runtime_overrides as optional object', () => {
    const parsed = parseTaskPayload({
      id: 'task-override',
      message: 'Run with overrides',
      task_kind: 'agent_message',
      schedule_type: 'interval',
      interval_seconds: 60,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T08:00:00Z',
      runtime_overrides: {
        model: 'gpt-5.4',
        tool_allowlist: ['script_exec'],
      },
    });

    expect(parsed).toMatchObject({
      runtime_overrides: {
        model: 'gpt-5.4',
        tool_allowlist: ['script_exec'],
      },
    });
  });

  it('throws when required fields are missing', () => {
    expect(() => {
      parseTaskPayload({
        id: 'task-3',
        task_kind: 'agent_message',
        schedule_type: 'interval',
        interval_seconds: 60,
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T08:00:00Z',
      });
    }).toThrow('task.message');
  });

  it('throws when workflow start default does not match declared type', () => {
    expect(() => {
      parseTaskPayload({
        id: 'workflow-invalid-default',
        task_kind: 'workflow',
        schedule_type: 'interval',
        interval_seconds: 60,
        enabled: true,
        created_at: '2026-04-05T07:00:00Z',
        updated_at: '2026-04-05T08:00:00Z',
        workflow: {
          nodes: [
            {
              id: 'start',
              type: 'start',
              start: {
                inputs: [
                  {
                    name: 'retry_count',
                    type: 'number',
                    default: '3',
                  },
                ],
              },
            },
            { id: 'end', type: 'end' },
          ],
          edges: [{ from_node_id: 'start', to_node_id: 'end' }],
        },
      });
    }).toThrow('task.workflow.nodes[0].start.inputs[0].default');
  });
});
