import {
  createLoopEditorState,
  filterLoopTasks,
  loopCreateRequestFromEditor,
  loopUpdateRequestFromEditor,
} from '@/lib/configLoops';
import type { AgentMessageTaskPayload, TaskPayload } from '@/lib/types';

describe('lib/configLoops', () => {
  it('filters relay agent tasks as loop tasks', () => {
    const loopTask = createAgentTask({ id: 'loop-1', agent_mode: 'relay' });
    const tasks: TaskPayload[] = [
      createAgentTask({ id: 'single-1', agent_mode: 'single' }),
      loopTask,
    ];

    expect(filterLoopTasks(tasks)).toEqual([loopTask]);
  });

  it('builds create payload with max rounds and preset', () => {
    const editor = {
      ...createLoopEditorState(null),
      message: 'Finish the migration',
      stopPolicy: 'max_rounds' as const,
      maxRounds: '4',
      presetId: 'preset-1',
    };

    expect(loopCreateRequestFromEditor(editor)).toEqual({
      task_kind: 'agent_message',
      message: 'Finish the migration',
      interval_seconds: 300,
      agent_mode: 'relay',
      relay: {
        stop_policy: 'max_rounds',
        max_rounds: 4,
        execution_timeout_ms: 0,
      },
      runtime_overrides: {
        preset_id: 'preset-1',
      },
    });
  });

  it('includes hard max rounds and omits preset for AI-decides create payload', () => {
    const editor = {
      ...createLoopEditorState(null),
      message: 'Watch inbox',
      stopPolicy: 'ai_decides' as const,
      maxRounds: '6',
      presetId: '',
    };

    expect(loopCreateRequestFromEditor(editor)).toEqual({
      task_kind: 'agent_message',
      message: 'Watch inbox',
      interval_seconds: 300,
      agent_mode: 'relay',
      relay: {
        stop_policy: 'ai_decides',
        max_rounds: 6,
        execution_timeout_ms: 0,
      },
      runtime_overrides: undefined,
    });
  });

  it('clears runtime overrides on update when preset is empty', () => {
    const editor = {
      ...createLoopEditorState(null),
      message: 'Update task',
      presetId: '',
    };

    expect(loopUpdateRequestFromEditor(editor)).toMatchObject({
      runtime_overrides: {},
    });
  });

  it('rejects invalid max rounds', () => {
    const editor = {
      ...createLoopEditorState(null),
      message: 'Invalid loop',
      stopPolicy: 'max_rounds' as const,
      maxRounds: '0',
    };

    expect(() => loopCreateRequestFromEditor(editor)).toThrow('relay max_rounds');
  });
});

function createAgentTask(input: {
  id: string;
  agent_mode?: AgentMessageTaskPayload['agent_mode'];
}): AgentMessageTaskPayload {
  return {
    id: input.id,
    message: input.id,
    agent_mode: input.agent_mode,
    task_kind: 'agent_message',
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: true,
    created_at: '2026-05-10T00:00:00Z',
    updated_at: '2026-05-10T00:00:00Z',
  };
}
