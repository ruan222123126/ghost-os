import {
  emptyTaskEditorState,
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
} from '@/lib/configTasks';

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

  it('builds relay task payload with max-round stop policy', () => {
    const editor = {
      ...emptyTaskEditorState,
      message: 'Long running task',
      agentMode: 'relay' as const,
      relayStopPolicy: 'max_rounds' as const,
      relayMaxRounds: '3',
      relayExecutionTimeoutMS: '0',
    };

    expect(taskCreateRequestFromEditor(editor)).toMatchObject({
      agent_mode: 'relay',
      relay: {
        stop_policy: 'max_rounds',
        max_rounds: 3,
        execution_timeout_ms: 0,
      },
    });
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
