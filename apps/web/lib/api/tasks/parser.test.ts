import { parseTaskPayload, parseTaskPayloadList, parseTaskRunLogList } from './parser';
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
      agent_mode: 'relay',
      relay: {
        stop_policy: 'max_rounds',
        max_rounds: 4,
        execution_timeout_ms: 0,
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

  it('parses AI-decides relay payload with hard max rounds', () => {
    const parsed = parseTaskPayload({
      id: 'loop-ai',
      message: 'Watch inbox',
      agent_mode: 'relay',
      relay: {
        stop_policy: 'ai_decides',
        max_rounds: 4,
        execution_timeout_ms: 0,
      },
      task_kind: 'agent_message',
      schedule_type: 'interval',
      interval_seconds: 300,
      enabled: true,
      created_at: '2026-04-05T07:00:00Z',
      updated_at: '2026-04-05T07:00:00Z',
    });

    expect(parsed).toMatchObject({
      agent_mode: 'relay',
      relay: {
        stop_policy: 'ai_decides',
        max_rounds: 4,
        execution_timeout_ms: 0,
      },
    });
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
        preset_id: 'preset-a',
        tool_allowlist: ['script_exec'],
      },
    });

    expect(parsed).toMatchObject({
      runtime_overrides: {
        model: 'gpt-5.4',
        preset_id: 'preset-a',
        tool_allowlist: ['script_exec'],
      },
    });
  });

  it('parses workflow agent runtime_overrides', () => {
    const parsed = parseTaskPayload({
      id: 'workflow-agent-override',
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
            id: 'agent',
            type: 'agent',
            agent: {
              message: 'run agent',
              runtime_overrides: {
                provider_name: 'openai-main',
                model: 'gpt-5.4',
                system_prompt: 'Be concise',
                tool_allowlist_only: true,
                tool_allowlist: ['script_exec'],
                max_turns: 3,
              },
            },
          },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'agent' },
          { from_node_id: 'agent', to_node_id: 'end' },
        ],
      },
    });

    expect(parsed.task_kind).toBe('workflow');
    if (parsed.task_kind !== 'workflow') {
      throw new Error('Expected workflow task payload');
    }
    expect(parsed.workflow.nodes.find((node) => node.id === 'agent')).toMatchObject({
      agent: {
        message: 'run agent',
        runtime_overrides: {
          provider_name: 'openai-main',
          model: 'gpt-5.4',
          system_prompt: 'Be concise',
          tool_allowlist_only: true,
          tool_allowlist: ['script_exec'],
          max_turns: 3,
        },
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

  it('parses task run logs with node_results', () => {
    const logs = parseTaskRunLogList([
      {
        task_id: 'task-log-1',
        run_id: 'run-1',
        trace_id: 'trace-1',
        task_kind: 'workflow',
        scheduled_at: '2026-04-05T07:00:00Z',
        started_at: '2026-04-05T07:00:01Z',
        finished_at: '2026-04-05T07:00:02Z',
        status: 'success',
        run_cards: [
          {
            card_id: 'card-1',
            run_id: 'run-1',
            kind: 'workflow_agent',
            title: 'agent-node',
            node_id: 'agent-node',
            node_type: 'agent',
            iteration: 2,
            source_session_id: 'session-agent',
            started_at: '2026-04-05T07:00:01Z',
            status: 'success',
            finished_at: '2026-04-05T07:00:02Z',
            preview: 'done',
            final_text: 'done',
            source_events: [
              {
                id: 'evt-1',
                step_id: 'turn-0001-assistant',
                trace_id: 'trace-1',
                session_id: 'session-agent',
                turn: 1,
                type: 'completion_delta',
                payload: { kind: 'text', text: 'done' },
                at: '2026-04-05T07:00:01Z',
              },
            ],
          },
        ],
        node_results: [
          {
            node_id: 'start-node',
            node_type: 'start',
            status: 'success',
            started_at: '2026-04-05T07:00:01Z',
            finished_at: '2026-04-05T07:00:01Z',
            completed_seq: 1,
            output: { next_node_id: 'tool-node' },
          },
        ],
      },
    ]);

    expect(logs).toHaveLength(1);
    expect(logs[0].node_results?.[0]).toMatchObject({
      node_id: 'start-node',
      completed_seq: 1,
    });
    expect(logs[0].run_cards?.[0]).toMatchObject({
      card_id: 'card-1',
      iteration: 2,
      final_text: 'done',
      source_events: [
        expect.objectContaining({
          id: 'evt-1',
          type: 'completion_delta',
        }),
      ],
    });
  });

  it('accepts incomplete task run status', () => {
    const logs = parseTaskRunLogList([
      {
        task_id: 'task-log-incomplete',
        run_id: 'run-incomplete',
        trace_id: 'trace-incomplete',
        scheduled_at: '2026-04-05T07:00:00Z',
        status: 'incomplete',
      },
    ]);

    expect(logs[0].status).toBe('incomplete');
  });

  it('accepts running task run status', () => {
    const logs = parseTaskRunLogList([
      {
        task_id: 'task-log-running',
        run_id: 'run-running',
        trace_id: 'trace-running',
        scheduled_at: '2026-04-05T07:00:00Z',
        started_at: '2026-04-05T07:00:01Z',
        status: 'running',
      },
    ]);

    expect(logs[0].status).toBe('running');
  });

  it('accepts task run log when node_results is missing', () => {
    const logs = parseTaskRunLogList([
      {
        task_id: 'task-log-2',
        run_id: 'run-2',
        trace_id: 'trace-2',
        scheduled_at: '2026-04-05T07:00:00Z',
        status: 'success',
      },
    ]);

    expect(logs).toHaveLength(1);
    expect(logs[0].node_results).toBeUndefined();
  });

  it('accepts task run log when node_results is null', () => {
    const logs = parseTaskRunLogList([
      {
        task_id: 'task-log-3',
        run_id: 'run-3',
        trace_id: 'trace-3',
        scheduled_at: '2026-04-05T07:00:00Z',
        status: 'success',
        node_results: null,
      },
    ]);

    expect(logs).toHaveLength(1);
    expect(logs[0].node_results).toBeUndefined();
  });

  it('throws when task run log node_results is non-array object', () => {
    expect(() => {
      parseTaskRunLogList([
        {
          task_id: 'task-log-4',
          run_id: 'run-4',
          trace_id: 'trace-4',
          scheduled_at: '2026-04-05T07:00:00Z',
          status: 'success',
          node_results: {},
        },
      ]);
    }).toThrow('node_results');
  });
});
