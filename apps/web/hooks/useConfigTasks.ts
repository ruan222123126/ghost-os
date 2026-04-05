'use client';

import { useCallback, useEffect, useState } from 'react';
import { createTask, deleteTask, listTasks, runTaskNow, updateTask } from '@/lib/api/tasks/api';
import type { TaskUpdateRequest } from '@/lib/envelope.generated';
import {
  type TaskEditorMode,
  type TaskEditorState,
  editorStateFromTask,
  emptyTaskEditorState,
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
} from '@/lib/configTasks';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { AgentMessageTaskPayload, TaskPayload } from '@/lib/types';

interface UseConfigTasksOptions {
  open: boolean;
}

interface UseConfigTasksResult {
  tasks: TaskPayload[];
  tasksLoading: boolean;
  taskSaving: boolean;
  taskError: string;
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  refreshTasks: () => Promise<void>;
  beginCreateTask: () => void;
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
  const { open } = options;
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
    setEditor(emptyTaskEditorState);
  }, []);

  const refreshTasks = useCallback(async () => {
    setTasksLoading(true);
    try {
      applyTaskList(await listTasks());
      setTaskError('');
    } catch (error) {
      setTaskError(toErrorMessage(error, 'failed to load tasks'));
    } finally {
      setTasksLoading(false);
    }
  }, [applyTaskList]);

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
      setTaskError(toErrorMessage(error, fallbackMessage));
      return null;
    } finally {
      setTaskSaving(false);
    }
  }, []);

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
      : () => createTask(taskCreateRequestFromEditor(editor, tasks));
    const task = await runMutation(action, 'failed to save task');
    if (!task) {
      return false;
    }

    upsertTask(task);
    resetEditor();
    return true;
  }, [editor, editorMode, editingTaskID, resetEditor, runMutation, tasks, upsertTask]);

  const setTaskEnabled = useCallback(async (id: string, enabled: boolean) => {
    const payload: Pick<TaskUpdateRequest, 'enabled'> = { enabled };
    const task = await runMutation(() => updateTask(id, payload), 'failed to update task');
    if (task) {
      upsertTask(task);
    }
  }, [runMutation, upsertTask]);

  const deleteTaskByID = useCallback(async (id: string) => {
    const result = await runMutation(() => deleteTask(id), 'failed to delete task');
    if (result === null) {
      return;
    }

    setTasks((state) => state.filter((task) => task.id !== id));
    if (editingTaskID === id) {
      resetEditor();
    }
  }, [editingTaskID, resetEditor, runMutation]);

  const runTaskNowByID = useCallback(async (id: string) => {
    const result = await runMutation(() => runTaskNow(id), 'failed to run task');
    if (result === null) {
      return;
    }
    await refreshTasks();
  }, [refreshTasks, runMutation]);

  const updateEditor = useCallback((patch: Partial<TaskEditorState>) => {
    setEditor((state) => ({ ...state, ...patch }));
  }, []);

  const editTask = useCallback((task: TaskPayload) => {
    if (!isAgentMessageTask(task)) {
      setTaskError('workflow editing is not available yet');
      return;
    }
    setEditorMode('edit');
    setEditingTaskID(task.id);
    setEditor(editorStateFromTask(task));
    setTaskError('');
  }, []);

  return {
    tasks,
    tasksLoading,
    taskSaving,
    taskError,
    editorMode,
    editor,
    refreshTasks,
    beginCreateTask: resetEditor,
    editTask,
    updateEditor,
    submitTask,
    setTaskEnabled,
    runTaskNowByID,
    deleteTaskByID,
    cancelEditing: resetEditor,
  };
}
