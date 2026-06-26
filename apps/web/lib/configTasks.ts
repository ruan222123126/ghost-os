import type { AgentMessageTaskPayload, TaskPayload, WorkflowTaskPayload } from '@/lib/types';

export {
  createTaskEditorState,
  editorStateFromTask,
  editorStateWithTaskType,
  emptyTaskEditorState,
} from '@/lib/configTaskEditorState';
export type {
  TaskEditorMode,
  TaskEditorState,
  TaskEditorType,
  TaskRelayStopPolicy,
  TaskScheduleMode,
} from '@/lib/configTaskEditorState';
export {
  taskCreateRequestFromEditor,
  taskUpdateRequestFromEditor,
} from '@/lib/configTaskRequests';

export type TaskSettingsListItem = AgentMessageTaskPayload | WorkflowTaskPayload;

export function isTaskSettingsListItem(task: TaskPayload): task is TaskSettingsListItem {
  return task.task_kind === 'agent_message' || task.task_kind === 'workflow';
}

export function filterTaskSettingsTasks(tasks: TaskPayload[]): TaskSettingsListItem[] {
  return tasks.filter(isTaskSettingsListItem);
}
