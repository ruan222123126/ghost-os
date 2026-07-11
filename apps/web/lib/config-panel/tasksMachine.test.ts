import {
  createInitialTasksState,
  tasksReducer,
  upsertTask,
} from '@/lib/config-panel/tasksMachine';
import { createTaskEditorState } from '@/lib/configTasks';
import type { AgentMessageTaskPayload } from '@/lib/types';

describe('lib/config-panel/tasksMachine', () => {
  it('enters create mode with the provided editor state', () => {
    const editor = { ...createTaskEditorState(null), message: 'say hello' };
    const state = tasksReducer(createInitialTasksState(null), {
      type: 'enter_create',
      editor,
    });

    expect(state.view).toBe('editor');
    expect(state.editorMode).toBe('create');
    expect(state.editor.taskType).toBe('text');
    expect(state.editor.message).toBe('say hello');
    expect(state.success).toBe('');
  });

  it('enters edit mode from an agent message task', () => {
    const task = buildTask({ id: 'task-1', message: 'daily summary' });
    const state = tasksReducer(createInitialTasksState(null), {
      type: 'enter_edit',
      task,
    });

    expect(state.view).toBe('editor');
    expect(state.editorMode).toBe('edit');
    expect(state.editingTaskID).toBe('task-1');
    expect(state.editor.taskType).toBe('text');
    expect(state.editor.message).toBe('daily summary');
  });

  it('enters edit mode from a relay loop task', () => {
    const task = buildTask({ id: 'loop-1', agent_mode: 'relay', message: 'loop goal' });
    const state = tasksReducer(createInitialTasksState(null), {
      type: 'enter_edit',
      task,
    });

    expect(state.editorMode).toBe('edit');
    expect(state.editingTaskID).toBe('loop-1');
    expect(state.editor.taskType).toBe('loop');
    expect(state.editor.message).toBe('loop goal');
  });

  it('replaces editor state immutably', () => {
    const initial = tasksReducer(createInitialTasksState(null), {
      type: 'enter_create',
      editor: createTaskEditorState(null),
    });
    const replacement = { ...initial.editor, taskType: 'loop' as const, message: 'loop' };
    const state = tasksReducer(initial, {
      type: 'replace_editor',
      editor: replacement,
    });

    expect(state.editor).toEqual(replacement);
    expect(state.editor).not.toBe(initial.editor);
  });

  it('upserts and removes tasks immutably', () => {
    const first = buildTask({ id: 'task-1', message: 'first' });
    const second = buildTask({ id: 'task-2', message: 'second' });
    const updated = buildTask({ id: 'task-1', message: 'updated' });

    expect(upsertTask([first], second)).toEqual([second, first]);
    expect(upsertTask([first, second], updated)).toEqual([updated, second]);

    const loaded = tasksReducer(createInitialTasksState(null), {
      type: 'load_success',
      tasks: [first, second],
    });
    const state = tasksReducer(loaded, { type: 'remove_task', id: 'task-1' });

    expect(state.tasks).toEqual([second]);
  });

  it('tracks run lifecycle state', () => {
    const started = tasksReducer(createInitialTasksState(null), {
      type: 'run_start',
      id: 'task-1',
    });
    const apiDone = tasksReducer(started, {
      type: 'run_api_done',
      success: 'Run started successfully',
    });
    const finished = tasksReducer(apiDone, { type: 'run_finish' });

    expect(started.saving).toBe(true);
    expect(started.runningTaskID).toBe('task-1');
    expect(apiDone.saving).toBe(false);
    expect(apiDone.runningTaskID).toBe('task-1');
    expect(apiDone.success).toBe('Run started successfully');
    expect(finished.runningTaskID).toBe('');
  });
});

function buildTask(overrides: Partial<AgentMessageTaskPayload>): AgentMessageTaskPayload {
  return {
    id: 'task-1',
    message: 'message',
    task_kind: 'agent_message',
    schedule_type: 'interval',
    interval_seconds: 60,
    enabled: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}
