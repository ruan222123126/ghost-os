'use client';

import { useMemo, useState } from 'react';
import { TaskEditorForm } from '@/components/config/TaskEditorForm';
import { TaskList } from '@/components/config/TaskList';
import { useTaskLogs } from '@/hooks/config/useTaskLogs';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskEditorMode, TaskEditorState } from '@/lib/configTasks';
import type { AgentMessageTaskPayload, TaskPayload, WorkflowTaskPayload } from '@/lib/types';

interface TaskSettingsSectionProps {
  tasks: TaskPayload[];
  loading: boolean;
  saving: boolean;
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  onRefresh: () => Promise<void>;
  onBeginCreateTextTask: () => void;
  onEditTextTask: (task: AgentMessageTaskPayload) => void;
  onOpenWorkflowCreate: () => void;
  onOpenWorkflowEdit: (task: WorkflowTaskPayload) => void;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSubmit: () => Promise<boolean>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCancelEditing: () => void;
}

export function TaskSettingsSection(props: TaskSettingsSectionProps) {
  const { copy } = useWebLocale();
  const {
    tasks,
    loading,
    saving,
    editorMode,
    editor,
    onRefresh,
    onBeginCreateTextTask,
    onEditTextTask,
    onOpenWorkflowCreate,
    onOpenWorkflowEdit,
    onChangeEditor,
    onSubmit,
    onSetEnabled,
    onRunNow,
    onDelete,
    onCancelEditing,
  } = props;
  const [editorOpen, setEditorOpen] = useState(false);
  const controlsDisabled = loading || saving;
  const sessionOptions = useMemo(() => buildSessionOptions(tasks), [tasks]);
  const logs = useTaskLogs(copy.settings.tasksLogsEmpty);

  const handleBeginCreateTextTask = () => {
    onBeginCreateTextTask();
    setEditorOpen(true);
  };

  const handleEditTextTask = (task: AgentMessageTaskPayload) => {
    onEditTextTask(task);
    setEditorOpen(true);
  };

  const handleEditWorkflowTask = (task: WorkflowTaskPayload) => {
    onOpenWorkflowEdit(task);
  };

  const handleCancel = () => {
    onCancelEditing();
    setEditorOpen(false);
  };

  const handleSubmit = async () => {
    if (await onSubmit()) {
      setEditorOpen(false);
    }
  };

  if (editorOpen) {
    return (
      <TaskEditorForm
        editorMode={editorMode}
        editor={editor}
        sessionOptions={sessionOptions}
        controlsDisabled={controlsDisabled}
        saving={saving}
        onChangeEditor={onChangeEditor}
        onSubmit={handleSubmit}
        onCancelEditing={handleCancel}
      />
    );
  }

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.tasksTitle}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.tasksDescription}</p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onRefresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.refresh}
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={handleBeginCreateTextTask}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.tasksNewText}
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={onOpenWorkflowCreate}
            className="rounded-full border border-[#111111] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.tasksNewWorkflow}
          </button>
        </div>
      </header>

      <TaskList
        tasks={tasks}
        loading={loading}
        controlsDisabled={controlsDisabled}
        onEditTextTask={handleEditTextTask}
        onEditWorkflowTask={handleEditWorkflowTask}
        onSetEnabled={onSetEnabled}
        onRunNow={onRunNow}
        onDelete={onDelete}
        logsTaskID={logs.taskID}
        logsData={logs.entries}
        logsLoading={logs.loading}
        logsError={logs.error}
        onOpenLogs={logs.open}
        onCloseLogs={logs.close}
      />
    </section>
  );
}

function buildSessionOptions(tasks: TaskPayload[]): string[] {
  const options = new Set<string>();
  for (const task of tasks) {
    if (task.task_kind !== 'agent_message') {
      continue;
    }
    const sessionID = (task.session_id ?? '').trim();
    if (sessionID.length > 0) {
      options.add(sessionID);
    }
  }

  return [...options].sort((left, right) => left.localeCompare(right));
}
