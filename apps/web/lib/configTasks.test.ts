import {
  editorStateFromTask,
  emptyTaskEditorState,
  filterTaskSettingsTasks,
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
} from '@/lib/configTasks';
import type { AgentMessageTaskPayload, TaskPayload, WorkflowTaskPayload } from '@/lib/types';

describe('lib/configTasks', () => {
  it('builds text task create payload', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Daily summary',
      sessionId: 'session-1',
      scheduleMode: 'interval' as const,
      intervalSeconds: '120',
    };

    const payload = taskCreateRequestFromEditor(editor);

    expect(payload).toEqual({
      task_kind: 'agent_message',
      message: 'Daily summary',
      session_id: 'session-1',
      interval_seconds: 120,
      agent_mode: 'single',
      runtime_overrides: undefined,
    });
  });

  it('builds update payload with empty runtime override object when disabled', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Weekly report',
      scheduleMode: 'cron' as const,
      cronExpr: '0 9 * * 1',
      runtimeOverridesEnabled: false,
    };

    const payload = taskUpdateRequestFromEditor(editor);

    expect(payload).toEqual({
      task_kind: 'agent_message',
      message: 'Weekly report',
      session_id: '',
      cron_expr: '0 9 * * 1',
      agent_mode: 'single',
      runtime_overrides: {},
    });
  });

  it('keeps text task payload in single-agent mode even if stale editor state says relay', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Long running task',
      agentMode: 'relay' as const,
      relayStopPolicy: 'max_rounds' as const,
      relayMaxRounds: '3',
      relayExecutionTimeoutMS: '0',
    };

    expect(taskCreateRequestFromEditor(editor)).toEqual({
      task_kind: 'agent_message',
      message: 'Long running task',
      session_id: '',
      interval_seconds: 300,
      agent_mode: 'single',
      runtime_overrides: undefined,
    });
  });

  it('builds text task payload with mounted preset and strict tools', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Preset task',
      runtimeOverridesEnabled: true,
      runtimePresetId: 'preset-1',
      runtimeToolAllowlist: 'script_exec, web_search',
    };

    expect(taskCreateRequestFromEditor(editor)).toMatchObject({
      runtime_overrides: {
        preset_id: 'preset-1',
        tool_allowlist_only: true,
        tool_allowlist: ['script_exec', 'web_search'],
      },
    });
  });

  it('keeps an empty strict tool allowlist when a preset has no tools', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'No-tool preset task',
      runtimeOverridesEnabled: true,
      runtimePresetId: 'preset-empty',
      runtimeToolAllowlist: '',
    };

    expect(taskCreateRequestFromEditor(editor)).toMatchObject({
      runtime_overrides: {
        preset_id: 'preset-empty',
        tool_allowlist_only: true,
        tool_allowlist: [],
      },
    });
  });

  it('hydrates mounted preset from an existing text task', () => {
    const editor = editorStateFromTask(createAgentTask({
      id: 'preset-task',
      runtime_overrides: {
        preset_id: 'preset-1',
        tool_allowlist_only: true,
        tool_allowlist: ['script_exec'],
      },
    }));

    expect(editor.runtimeOverridesEnabled).toBe(true);
    expect(editor.runtimePresetId).toBe('preset-1');
    expect(editor.runtimeToolAllowlist).toBe('script_exec');
  });

  it('filters relay tasks out of the normal task settings list', () => {
    const textTask = createAgentTask({ id: 'text-1', agent_mode: 'single' });
    const legacyTextTask = createAgentTask({ id: 'text-legacy' });
    const loopTask = createAgentTask({ id: 'loop-1', agent_mode: 'relay' });
    const workflowTask = createWorkflowTask();

    const tasks: TaskPayload[] = [loopTask, textTask, workflowTask, legacyTextTask];

    expect(filterTaskSettingsTasks(tasks).map((task) => task.id)).toEqual([
      'text-1',
      'workflow-1',
      'text-legacy',
    ]);
  });

  it('throws on invalid interval value', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Invalid interval',
      intervalSeconds: '0',
    };

    expect(() => taskCreateRequestFromEditor(editor)).toThrow(
      'interval seconds must be a positive integer',
    );
  });
});

function createAgentTask(input: {
  id: string;
  agent_mode?: AgentMessageTaskPayload['agent_mode'];
  runtime_overrides?: AgentMessageTaskPayload['runtime_overrides'];
}): AgentMessageTaskPayload {
  return {
    id: input.id,
    message: input.id,
    agent_mode: input.agent_mode,
    runtime_overrides: input.runtime_overrides,
    task_kind: 'agent_message',
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: true,
    created_at: '2026-05-10T00:00:00Z',
    updated_at: '2026-05-10T00:00:00Z',
  };
}

function createWorkflowTask(): WorkflowTaskPayload {
  return {
    id: 'workflow-1',
    task_kind: 'workflow',
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: true,
    created_at: '2026-05-10T00:00:00Z',
    updated_at: '2026-05-10T00:00:00Z',
    workflow: {
      nodes: [
        { id: 'start', type: 'start' },
        { id: 'end', type: 'end' },
      ],
      edges: [{ from_node_id: 'start', to_node_id: 'end' }],
    },
  };
}
