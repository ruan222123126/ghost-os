import {
  type TaskEditorMode,
  type TaskEditorState,
  createTaskEditorState,
  editorStateFromTask,
} from '@/lib/configTasks';
import type { WebCopy } from '@/lib/i18n/messages';
import type { AgentMessageTaskPayload, BridgeConfig, TaskPayload } from '@/lib/types';

export type TasksView = 'list' | 'editor';

export interface TasksState {
  view: TasksView;
  tasks: TaskPayload[];
  loading: boolean;
  saving: boolean;
  error: string;
  success: string;
  runningTaskID: string;
  editorMode: TaskEditorMode;
  editingTaskID: string;
  editor: TaskEditorState;
}

export type TasksAction =
  | { type: 'load_start' }
  | { type: 'load_success'; tasks: TaskPayload[] }
  | { type: 'load_error'; error: string }
  | { type: 'enter_create'; editor: TaskEditorState }
  | { type: 'enter_edit'; task: AgentMessageTaskPayload }
  | { type: 'exit_editor'; editor: TaskEditorState }
  | { type: 'patch_editor'; patch: Partial<TaskEditorState> }
  | { type: 'replace_editor'; editor: TaskEditorState }
  | { type: 'mutate_start' }
  | { type: 'mutate_error'; error: string }
  | { type: 'set_saving'; saving: boolean }
  | { type: 'set_error'; error: string }
  | { type: 'set_success'; success: string }
  | { type: 'upsert_task'; task: TaskPayload }
  | { type: 'remove_task'; id: string }
  | { type: 'run_start'; id: string }
  | { type: 'run_api_done'; success?: string }
  | { type: 'run_error'; error: string }
  | { type: 'run_finish' };

type TasksActionReducer = (state: TasksState, action: TasksAction) => TasksState | null;

const TASK_ACTION_REDUCERS: TasksActionReducer[] = [
  reduceLoadAction,
  reduceEditorAction,
  reduceMutationAction,
  reduceTaskListAction,
  reduceRunAction,
];

export function createInitialTasksState(config: BridgeConfig | null): TasksState {
  return {
    view: 'list',
    tasks: [],
    loading: false,
    saving: false,
    error: '',
    success: '',
    runningTaskID: '',
    editorMode: 'create',
    editingTaskID: '',
    editor: createTaskEditorState(config),
  };
}

export function tasksReducer(state: TasksState, action: TasksAction): TasksState {
  for (const reduceAction of TASK_ACTION_REDUCERS) {
    const nextState = reduceAction(state, action);
    if (nextState) {
      return nextState;
    }
  }
  return state;
}

function reduceLoadAction(state: TasksState, action: TasksAction): TasksState | null {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return { ...state, loading: false, tasks: action.tasks, error: '' };
    case 'load_error':
      return { ...state, loading: false, error: action.error, success: '' };
    default:
      return null;
  }
}

function reduceEditorAction(state: TasksState, action: TasksAction): TasksState | null {
  switch (action.type) {
    case 'enter_create':
      return enterCreateTask(state, action.editor);
    case 'enter_edit':
      return enterEditTask(state, action.task);
    case 'exit_editor':
      return resetTaskEditor(state, action.editor);
    case 'patch_editor':
      return { ...state, editor: { ...state.editor, ...action.patch } };
    case 'replace_editor':
      return { ...state, editor: action.editor };
    default:
      return null;
  }
}

function reduceMutationAction(state: TasksState, action: TasksAction): TasksState | null {
  switch (action.type) {
    case 'mutate_start':
      return { ...state, saving: true, error: '', success: '' };
    case 'mutate_error':
      return { ...state, saving: false, error: action.error, success: '' };
    case 'set_saving':
      return { ...state, saving: action.saving };
    case 'set_error':
      return { ...state, error: action.error, success: '' };
    case 'set_success':
      return { ...state, error: '', success: action.success };
    default:
      return null;
  }
}

function reduceTaskListAction(state: TasksState, action: TasksAction): TasksState | null {
  switch (action.type) {
    case 'upsert_task':
      return { ...state, tasks: upsertTask(state.tasks, action.task) };
    case 'remove_task':
      return { ...state, tasks: removeTask(state.tasks, action.id) };
    default:
      return null;
  }
}

function reduceRunAction(state: TasksState, action: TasksAction): TasksState | null {
  switch (action.type) {
    case 'run_start':
      return runStart(state, action.id);
    case 'run_api_done':
      return runAPIDone(state, action.success);
    case 'run_error':
      return { ...state, saving: false, runningTaskID: '', error: action.error, success: '' };
    case 'run_finish':
      return { ...state, saving: false, runningTaskID: '' };
    default:
      return null;
  }
}

export function upsertTask(list: TaskPayload[], task: TaskPayload): TaskPayload[] {
  const index = list.findIndex((item) => item.id === task.id);
  if (index === -1) {
    return [task, ...list];
  }
  return list.map((item) => item.id === task.id ? task : item);
}

export function localizeTaskEditorError(message: string, copy: WebCopy): string {
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

function enterCreateTask(state: TasksState, editor: TaskEditorState): TasksState {
  return {
    ...state,
    view: 'editor',
    editorMode: 'create',
    editingTaskID: '',
    editor,
    error: '',
    success: '',
  };
}

function enterEditTask(
  state: TasksState,
  task: AgentMessageTaskPayload,
): TasksState {
  return {
    ...state,
    view: 'editor',
    editorMode: 'edit',
    editingTaskID: task.id,
    editor: editorStateFromTask(task),
    error: '',
    success: '',
  };
}

function resetTaskEditor(
  state: TasksState,
  editor: TaskEditorState,
): TasksState {
  return {
    ...state,
    view: 'list',
    editorMode: 'create',
    editingTaskID: '',
    editor,
  };
}

function removeTask(list: TaskPayload[], id: string): TaskPayload[] {
  return list.filter((task) => task.id !== id);
}

function runStart(state: TasksState, id: string): TasksState {
  return {
    ...state,
    saving: true,
    runningTaskID: id,
    error: '',
    success: '',
  };
}

function runAPIDone(state: TasksState, success = ''): TasksState {
  return {
    ...state,
    saving: false,
    error: '',
    success,
  };
}
