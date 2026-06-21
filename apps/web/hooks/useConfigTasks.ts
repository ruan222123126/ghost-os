'use client';

import { useCallback, useEffect, useMemo, useReducer, type Dispatch } from 'react';
import { useTasksApi } from '@/hooks/config-panel/useTasksApi';
import {
  createInitialTasksState,
  localizeTaskEditorError,
  tasksReducer,
  type TasksAction,
  type TasksState,
} from '@/lib/config-panel/tasksMachine';
import {
  createTaskEditorState,
  editorStateWithTaskType,
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
  type TaskEditorType,
} from '@/lib/configTasks';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskPatchRequest } from '@/lib/types';
import type { AgentMessageTaskPayload, BridgeConfig, TaskPayload } from '@/lib/types';

interface UseConfigTasksOptions {
  open: boolean;
  config: BridgeConfig | null;
}

interface RunTaskOptions {
  reportSuccess?: boolean;
}

interface UseConfigTasksActions {
  refresh: () => Promise<boolean>;
  startCreateTask: () => void;
  startEditTextTask: (task: TaskPayload) => void;
  cancelEditing: () => void;
  updateEditor: (patch: Partial<TasksState['editor']>) => void;
  selectTaskType: (taskType: TaskEditorType) => void;
  submit: () => Promise<boolean>;
  setEnabled: (id: string, enabled: boolean) => Promise<void>;
  runNow: (id: string, options?: RunTaskOptions) => Promise<boolean>;
  delete: (id: string) => Promise<void>;
}

interface UseConfigTasksResult {
  state: TasksState;
  actions: UseConfigTasksActions;
}

type TasksApi = ReturnType<typeof useTasksApi>;

function isAgentMessageTask(task: TaskPayload): task is AgentMessageTaskPayload {
  return task.task_kind === 'agent_message';
}

export function useConfigTasks(options: UseConfigTasksOptions): UseConfigTasksResult {
  const { copy } = useWebLocale();
  const api = useTasksApi();
  const [state, dispatch] = useReducer(tasksReducer, options.config, createInitialTasksState);
  const createEditor = useCallback(() => createTaskEditorState(options.config), [options.config]);
  const refresh = useRefreshTasks(api, copy, dispatch);
  const submit = useSubmitTask(api, copy, state, createEditor, dispatch);
  const setEnabled = useSetTaskEnabled(api, copy, dispatch);
  const runNow = useRunTaskNow(api, copy, state, refresh, dispatch);
  const deleteByID = useDeleteTask(api, copy, state, createEditor, dispatch);
  const selectTaskType = useSelectTaskType(options.config, state, dispatch);
  const actions = useTaskActionBundle(
    dispatch,
    createEditor,
    copy,
    refresh,
    submit,
    setEnabled,
    runNow,
    deleteByID,
    selectTaskType,
  );

  useEffect(() => {
    if (options.open) {
      ignorePromise(refresh());
    }
  }, [options.open, refresh]);

  return { state, actions };
}

function useRefreshTasks(
  api: TasksApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback(async (): Promise<boolean> => {
    dispatch({ type: 'load_start' });
    try {
      dispatch({ type: 'load_success', tasks: await api.listTasks() });
      return true;
    } catch (error) {
      dispatch({
        type: 'load_error',
        error: taskErrorMessage(error, copy.system.failedToLoadTasks, copy),
      });
      return false;
    }
  }, [api, copy, dispatch]);
}

function useSubmitTask(
  api: TasksApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: TasksState,
  createEditor: () => TasksState['editor'],
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback(async (): Promise<boolean> => {
    dispatch({ type: 'mutate_start' });
    try {
      const task = state.editorMode === 'edit'
        ? await api.updateTask(state.editingTaskID, taskUpdateRequestFromEditor(state.editor))
        : await api.createTask(taskCreateRequestFromEditor(state.editor));
      dispatch({ type: 'upsert_task', task });
      dispatch({ type: 'set_saving', saving: false });
      dispatch({ type: 'exit_editor', editor: createEditor() });
      return true;
    } catch (error) {
      dispatchTaskError(dispatch, error, copy.system.failedToSaveTask, copy);
      return false;
    }
  }, [api, copy, createEditor, dispatch, state.editingTaskID, state.editor, state.editorMode]);
}

function useSetTaskEnabled(
  api: TasksApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback(async (id: string, enabled: boolean): Promise<void> => {
    dispatch({ type: 'mutate_start' });
    try {
      const payload: Pick<TaskPatchRequest, 'enabled'> = { enabled };
      dispatch({ type: 'upsert_task', task: await api.updateTask(id, payload) });
      dispatch({ type: 'set_saving', saving: false });
    } catch (error) {
      dispatchTaskError(dispatch, error, copy.system.failedToUpdateTask, copy);
    }
  }, [api, copy, dispatch]);
}

function useRunTaskNow(
  api: TasksApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: TasksState,
  refresh: () => Promise<boolean>,
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback(async (id: string, options?: RunTaskOptions): Promise<boolean> => {
    if (state.runningTaskID) {
      return false;
    }
    dispatch({ type: 'run_start', id });
    try {
      await api.runTaskNow(id);
      dispatch({ type: 'run_api_done', success: options?.reportSuccess ? copy.settings.tasksRunStarted : '' });
      await refresh();
      return true;
    } catch (error) {
      dispatchTaskError(dispatch, error, copy.system.failedToRunTask, copy, 'run_error');
      return false;
    } finally {
      dispatch({ type: 'run_finish' });
    }
  }, [api, copy, dispatch, refresh, state.runningTaskID]);
}

function useDeleteTask(
  api: TasksApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: TasksState,
  createEditor: () => TasksState['editor'],
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback(async (id: string): Promise<void> => {
    dispatch({ type: 'mutate_start' });
    try {
      await api.deleteTask(id);
      dispatch({ type: 'remove_task', id });
      if (state.editingTaskID === id) {
        dispatch({ type: 'exit_editor', editor: createEditor() });
      }
      dispatch({ type: 'set_saving', saving: false });
    } catch (error) {
      dispatchTaskError(dispatch, error, copy.system.failedToDeleteTask, copy);
    }
  }, [api, copy, createEditor, dispatch, state.editingTaskID]);
}

function useSelectTaskType(
  config: BridgeConfig | null,
  state: TasksState,
  dispatch: Dispatch<TasksAction>,
) {
  return useCallback((taskType: TaskEditorType): void => {
    if (state.editorMode !== 'create' || state.editor.taskType === taskType) {
      return;
    }
    dispatch({
      type: 'replace_editor',
      editor: editorStateWithTaskType(state.editor, taskType, config),
    });
  }, [config, dispatch, state.editor, state.editorMode]);
}

function useTaskActionBundle(
  dispatch: Dispatch<TasksAction>,
  createEditor: () => TasksState['editor'],
  copy: ReturnType<typeof useWebLocale>['copy'],
  refresh: UseConfigTasksActions['refresh'],
  submit: UseConfigTasksActions['submit'],
  setEnabled: UseConfigTasksActions['setEnabled'],
  runNow: UseConfigTasksActions['runNow'],
  deleteByID: UseConfigTasksActions['delete'],
  selectTaskType: UseConfigTasksActions['selectTaskType'],
): UseConfigTasksActions {
  return useMemo(() => ({
    refresh,
    submit,
    setEnabled,
    runNow,
    delete: deleteByID,
    selectTaskType,
    startCreateTask: () => dispatch({ type: 'enter_create', editor: createEditor() }),
    startEditTextTask: (task) => startEditTextTask(
      task,
      dispatch,
      copy.system.textTaskEditorOnlySupportsAgentMessage,
    ),
    cancelEditing: () => dispatch({ type: 'exit_editor', editor: createEditor() }),
    updateEditor: (patch) => dispatch({ type: 'patch_editor', patch }),
  }), [
    copy.system.textTaskEditorOnlySupportsAgentMessage,
    createEditor,
    deleteByID,
    dispatch,
    refresh,
    runNow,
    selectTaskType,
    setEnabled,
    submit,
  ]);
}

function startEditTextTask(
  task: TaskPayload,
  dispatch: Dispatch<TasksAction>,
  unsupportedMessage: string,
) {
  if (!isAgentMessageTask(task)) {
    dispatch({
      type: 'set_error',
      error: unsupportedMessage,
    });
    return;
  }
  dispatch({ type: 'enter_edit', task });
}

function dispatchTaskError(
  dispatch: Dispatch<TasksAction>,
  error: unknown,
  fallback: string,
  copy: ReturnType<typeof useWebLocale>['copy'],
  actionType: 'mutate_error' | 'run_error' = 'mutate_error',
) {
  dispatch({ type: actionType, error: taskErrorMessage(error, fallback, copy) });
}

function taskErrorMessage(
  error: unknown,
  fallback: string,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  return localizeTaskEditorError(toErrorMessage(error, fallback), copy);
}
