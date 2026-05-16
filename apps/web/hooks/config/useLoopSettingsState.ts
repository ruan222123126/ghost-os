'use client';

import { useCallback, useMemo, useState } from 'react';
import { createTask, updateTask } from '@/lib/api/tasks/api';
import {
  createLoopEditorState,
  editorStateFromLoopTask,
  filterLoopTasks,
  loopCreateRequestFromEditor,
  loopUpdateRequestFromEditor,
  type LoopEditorMode,
  type LoopEditorState,
} from '@/lib/configLoops';
import { toErrorMessage } from '@/lib/errors';
import type { WebCopy } from '@/lib/i18n/messages';
import type { AgentMessageTaskPayload, BridgeConfig, PresetPayload, TaskPayload } from '@/lib/types';

interface LoopSettingsStateOptions {
  tasks: TaskPayload[];
  tasksLoading: boolean;
  taskSaving: boolean;
  taskError: string;
  presets: PresetPayload[];
  presetsLoading: boolean;
  presetError: string;
  config: BridgeConfig | null;
  copy: WebCopy;
  refreshTasks: () => Promise<void>;
  setTaskEnabled: (id: string, enabled: boolean) => Promise<void>;
  runTaskNowByID: (id: string) => Promise<void>;
  deleteTaskByID: (id: string) => Promise<void>;
}

export interface LoopSettingsState {
  loops: AgentMessageTaskPayload[];
  presets: PresetPayload[];
  loading: boolean;
  error: string;
  editorOpen: boolean;
  editorMode: LoopEditorMode;
  editor: LoopEditorState;
  submitting: boolean;
  runningLoopID: string;
  controlsDisabled: boolean;
  refresh: () => Promise<void>;
  startCreate: () => void;
  editLoop: (task: AgentMessageTaskPayload) => void;
  updateEditor: (patch: Partial<LoopEditorState>) => void;
  submitEditor: () => Promise<boolean>;
  cancelEditor: () => void;
  runByID: (id: string) => Promise<void>;
  setEnabledByID: (id: string, enabled: boolean) => Promise<void>;
  deleteByID: (id: string) => Promise<void>;
}

export function useLoopSettingsState(options: LoopSettingsStateOptions): LoopSettingsState {
  const data = useLoopSettingsData(options);
  const editor = useLoopEditorController(options);
  const runner = useLoopRunner({
    copy: options.copy,
    runTaskNowByID: options.runTaskNowByID,
  });
  const error = editor.error || runner.error || data.error;
  const loading = options.tasksLoading || options.presetsLoading;
  const controlsDisabled = loading || options.taskSaving || editor.submitting || runner.runningLoopID.length > 0;

  return {
    loops: data.loops,
    presets: options.presets,
    loading,
    error,
    editorOpen: editor.editorOpen,
    editorMode: editor.editorMode,
    editor: editor.editor,
    submitting: editor.submitting,
    runningLoopID: runner.runningLoopID,
    controlsDisabled,
    refresh: options.refreshTasks,
    startCreate: editor.startCreate,
    editLoop: editor.editLoop,
    updateEditor: editor.updateEditor,
    submitEditor: editor.submitEditor,
    cancelEditor: editor.resetEditor,
    runByID: runner.runByID,
    setEnabledByID: options.setTaskEnabled,
    deleteByID: options.deleteTaskByID,
  };
}

function useLoopSettingsData(options: LoopSettingsStateOptions) {
  const { presetError, taskError, tasks } = options;
  const loops = useMemo(() => filterLoopTasks(tasks), [tasks]);
  const error = taskError || presetError;

  return { loops, error };
}

function useLoopEditorController(options: LoopSettingsStateOptions) {
  const { config, copy, refreshTasks } = options;
  const [editorMode, setEditorMode] = useState<LoopEditorMode>('create');
  const [editorOpen, setEditorOpen] = useState(false);
  const [editor, setEditor] = useState<LoopEditorState>(() => createLoopEditorState(config));
  const [editingTaskID, setEditingTaskID] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const resetEditor = useCallback(() => {
    resetLoopEditorState(config, {
      setEditor,
      setEditingTaskID,
      setEditorMode,
      setEditorOpen,
    });
  }, [config]);

  const controlSetters = {
    config,
    setEditor,
    setEditingTaskID,
    setEditorMode,
    setEditorOpen,
    setError,
  };
  const submitEditor = useSubmitLoopEditor({
    copy,
    editor,
    editorMode,
    editingTaskID,
    refreshTasks,
    resetEditor,
    setError,
    setSubmitting,
  });

  return {
    editorMode,
    editorOpen,
    editor,
    submitting,
    error,
    resetEditor,
    startCreate: useStartLoopCreate(controlSetters),
    editLoop: useEditLoopTask(controlSetters),
    updateEditor: useCallback((patch: Partial<LoopEditorState>) => {
      setEditor((state) => ({ ...state, ...patch }));
    }, []),
    submitEditor,
  };
}

function resetLoopEditorState(
  config: BridgeConfig | null,
  setters: Pick<
    LoopEditorControlSetters,
    'setEditor' | 'setEditingTaskID' | 'setEditorMode' | 'setEditorOpen'
  >,
) {
  setters.setEditorMode('create');
  setters.setEditorOpen(false);
  setters.setEditingTaskID('');
  setters.setEditor(createLoopEditorState(config));
}

interface LoopEditorControlSetters {
  config: BridgeConfig | null;
  setEditorMode: (mode: LoopEditorMode) => void;
  setEditorOpen: (open: boolean) => void;
  setEditingTaskID: (id: string) => void;
  setEditor: (state: LoopEditorState) => void;
  setError: (message: string) => void;
}

function useStartLoopCreate(options: LoopEditorControlSetters) {
  const { config, setEditor, setEditingTaskID, setEditorMode, setEditorOpen, setError } = options;

  return useCallback(() => {
    setEditorMode('create');
    setEditorOpen(true);
    setEditingTaskID('');
    setEditor(createLoopEditorState(config));
    setError('');
  }, [config, setEditingTaskID, setEditor, setEditorMode, setEditorOpen, setError]);
}

function useEditLoopTask(options: LoopEditorControlSetters) {
  const { config, setEditor, setEditingTaskID, setEditorMode, setEditorOpen, setError } = options;

  return useCallback((task: AgentMessageTaskPayload) => {
    setEditorMode('edit');
    setEditorOpen(true);
    setEditingTaskID(task.id);
    setEditor(editorStateFromLoopTask(task, config));
    setError('');
  }, [config, setEditingTaskID, setEditor, setEditorMode, setEditorOpen, setError]);
}

function useSubmitLoopEditor(options: {
  copy: WebCopy;
  editor: LoopEditorState;
  editorMode: LoopEditorMode;
  editingTaskID: string;
  refreshTasks: () => Promise<void>;
  resetEditor: () => void;
  setError: (message: string) => void;
  setSubmitting: (submitting: boolean) => void;
}) {
  const { copy, editor, editorMode, editingTaskID, refreshTasks, resetEditor, setError, setSubmitting } = options;

  return useCallback(async (): Promise<boolean> => {
    setSubmitting(true);
    setError('');
    try {
      await submitLoop(editorMode, editingTaskID, editor);
      await refreshTasks();
      resetEditor();
      return true;
    } catch (error) {
      setError(localizeLoopError(toErrorMessage(error, copy.system.failedToSaveTask), copy));
      return false;
    } finally {
      setSubmitting(false);
    }
  }, [copy, editor, editorMode, editingTaskID, refreshTasks, resetEditor, setError, setSubmitting]);
}

function useLoopRunner(options: {
  copy: WebCopy;
  runTaskNowByID: (id: string) => Promise<void>;
}) {
  const { copy, runTaskNowByID } = options;
  const [runningLoopID, setRunningLoopID] = useState('');
  const [error, setError] = useState('');

  const runByID = useCallback(async (id: string) => {
    if (runningLoopID.length > 0) {
      return;
    }
    setRunningLoopID(id);
    setError('');
    try {
      await runTaskNowByID(id);
    } catch (error) {
      setError(toErrorMessage(error, copy.system.failedToRunTask));
    } finally {
      setRunningLoopID('');
    }
  }, [copy.system.failedToRunTask, runTaskNowByID, runningLoopID]);

  return { runByID, runningLoopID, error };
}

async function submitLoop(
  mode: LoopEditorMode,
  taskID: string,
  editor: LoopEditorState,
): Promise<void> {
  if (mode === 'edit') {
    await updateTask(taskID, loopUpdateRequestFromEditor(editor));
    return;
  }
  await createTask(loopCreateRequestFromEditor(editor));
}

function localizeLoopError(message: string, copy: WebCopy): string {
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
