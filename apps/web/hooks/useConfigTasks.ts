'use client';

import { useCallback, useEffect, useState } from 'react';
import { createTask, deleteTask, listTasks, runTaskNow, updateTask } from '@/lib/api/tasks/api';
import type { TaskUpdateRequest } from '@/lib/envelope.generated';
import {
  type TaskEditorMode,
  type TaskEditorState,
  createTaskEditorState,
  editorStateFromTask,
  emptyTaskEditorState,
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
} from '@/lib/configTasks';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, BridgeConfig, TaskPayload } from '@/lib/types';

interface UseConfigTasksOptions {
  open: boolean;
  config: BridgeConfig | null;
}

interface UseConfigTasksResult {
  tasks: TaskPayload[];
  tasksLoading: boolean;
  taskSaving: boolean;
  taskError: string;
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  refreshTasks: () => Promise<void>;
  beginCreateTextTask: () => void;
  editTask: (task: TaskPayload) => void;
  updateEditor: (patch: Partial<TaskEditorState>) => void;
  submitTask: () => Promise<boolean>;
  setTaskEnabled: (id: string, enabled: boolean) => Promise<void>;
  runTaskNowByID: (id: string) => Promise<void>;
  deleteTaskByID: (id: string) => Promise<void>;
  cancelEditing: () => void;
}

function isAgentMessageTask(task: TaskPayload): task is AgentMessageTaskPayload {
  return task.task_kind === 'agent_message';
}

export function useConfigTasks(options: UseConfigTasksOptions): UseConfigTasksResult {
  const { copy } = useWebLocale();
  const { open, config } = options;
  const [tasks, setTasks] = useState<TaskPayload[]>([]);
  const [tasksLoading, setTasksLoading] = useState(false);
  const [taskSaving, setTaskSaving] = useState(false);
  const [taskError, setTaskError] = useState('');
  const [editorMode, setEditorMode] = useState<TaskEditorMode>('create');
  const [editingTaskID, setEditingTaskID] = useState('');
  const [editor, setEditor] = useState<TaskEditorState>(emptyTaskEditorState);

  const applyTaskList = useCallback((payload: TaskPayload[]) => {
    setTasks(payload);
  }, []);

  const resetEditor = useCallback((mode: TaskEditorMode = 'create') => {
    setEditorMode(mode);
    setEditingTaskID('');
    setEditor(createTaskEditorState(config));
  }, [config]);

  const refreshTasks = useCallback(async () => {
    setTasksLoading(true);
    try {
      applyTaskList(await listTasks());
      setTaskError('');
    } catch (error) {
      setTaskError(toErrorMessage(error, copy.system.failedToLoadTasks));
    } finally {
      setTasksLoading(false);
    }
  }, [applyTaskList, copy.system.failedToLoadTasks]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshTasks());
    }
  }, [open, refreshTasks]);

  const runMutation = useCallback(async <T,>(
    action: () => Promise<T>,
    fallbackMessage: string,
  ): Promise<T | null> => {
    setTaskSaving(true);
    setTaskError('');
    try {
      return await action();
    } catch (error) {
      const message = toErrorMessage(error, fallbackMessage);
      setTaskError(localizeTaskEditorError(message, copy));
      return null;
    } finally {
      setTaskSaving(false);
    }
  }, [copy]);

  const upsertTask = useCallback((task: TaskPayload) => {
    setTasks((state) => {
      const index = state.findIndex((item) => item.id === task.id);
      if (index === -1) {
        return [task, ...state];
      }
      const next = [...state];
      next[index] = task;
      return next;
    });
  }, []);

  const submitTask = useCallback(async (): Promise<boolean> => {
    const action = editorMode === 'edit'
      ? () => updateTask(editingTaskID, taskUpdateRequestFromEditor(editor))
      : () => createTask(taskCreateRequestFromEditor(editor));
    const task = await runMutation(action, copy.system.failedToSaveTask);
    if (!task) {
      return false;
    }

    upsertTask(task);
    resetEditor();
    return true;
  }, [copy.system.failedToSaveTask, editor, editorMode, editingTaskID, resetEditor, runMutation, upsertTask]);

  const setTaskEnabled = useCallback(async (id: string, enabled: boolean) => {
    const payload: Pick<TaskUpdateRequest, 'enabled'> = { enabled };
    const task = await runMutation(() => updateTask(id, payload), copy.system.failedToUpdateTask);
    if (task) {
      upsertTask(task);
    }
  }, [copy.system.failedToUpdateTask, runMutation, upsertTask]);

  const deleteTaskByID = useCallback(async (id: string) => {
    const result = await runMutation(() => deleteTask(id), copy.system.failedToDeleteTask);
    if (result === null) {
      return;
    }

    setTasks((state) => state.filter((task) => task.id !== id));
    if (editingTaskID === id) {
      resetEditor();
    }
  }, [copy.system.failedToDeleteTask, editingTaskID, resetEditor, runMutation]);

  const runTaskNowByID = useCallback(async (id: string) => {
    const result = await runMutation(() => runTaskNow(id), copy.system.failedToRunTask);
    if (result === null) {
      return;
    }
  }, [copy.system.failedToRunTask, runMutation]);

  const updateEditor = useCallback((patch: Partial<TaskEditorState>) => {
    setEditor((state) => ({ ...state, ...patch }));
  }, []);

  const editTask = useCallback((task: TaskPayload) => {
    if (!isAgentMessageTask(task)) {
      setTaskError(copy.system.textTaskEditorOnlySupportsAgentMessage);
      return;
    }
    setEditorMode('edit');
    setEditingTaskID(task.id);
    setEditor(editorStateFromTask(task));
    setTaskError('');
  }, [copy.system.textTaskEditorOnlySupportsAgentMessage]);

  return {
    tasks,
    tasksLoading,
    taskSaving,
    taskError,
    editorMode,
    editor,
    refreshTasks,
    beginCreateTextTask: resetEditor,
    editTask,
    updateEditor,
    submitTask,
    setTaskEnabled,
    runTaskNowByID,
    deleteTaskByID,
    cancelEditing: resetEditor,
  };
}

function localizeTaskEditorError(
  message: string,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (message === 'interval seconds must be a positive integer') {
    return copy.system.intervalSecondsPositiveInteger;
  }
  if (message === 'relay max_rounds must be a positive integer' || message === 'relay max_rounds must be > 0') {
    return copy.system.relayMaxRoundsPositiveInteger;
  }
  if (message === 'relay execution_timeout_ms must be a non-negative integer') {
    return copy.system.relayTimeoutNonNegativeInteger;
  }
  if (message === 'message is required') {
    return copy.system.messageRequired;
  }
  if (message === 'cron expression is required') {
    return copy.system.cronExpressionRequired;
  }
  return message;
}
