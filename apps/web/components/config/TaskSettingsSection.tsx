'use client';

import { useMemo, useState } from 'react';
import { TaskEditorForm } from '@/components/config/TaskEditorForm';
import { TaskList } from '@/components/config/TaskList';
import { ignorePromise } from '@/lib/errors';
import type { TaskEditorMode, TaskEditorState } from '@/lib/configTasks';
import type { TaskPayload } from '@/lib/types';

interface TaskSettingsSectionProps {
  tasks: TaskPayload[];
  loading: boolean;
  saving: boolean;
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  onRefresh: () => Promise<void>;
  onBeginCreate: () => void;
  onEdit: (task: TaskPayload) => void;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSubmit: () => Promise<boolean>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCancelEditing: () => void;
}

export function TaskSettingsSection(props: TaskSettingsSectionProps) {
  const {
    tasks,
    loading,
    saving,
    editorMode,
    editor,
    onRefresh,
    onBeginCreate,
    onEdit,
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

  const handleBeginCreate = () => {
    onBeginCreate();
    setEditorOpen(true);
  };

  const handleEdit = (task: TaskPayload) => {
    onEdit(task);
    setEditorOpen(task.task_kind === 'agent_message');
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
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">Tasks</h1>
          <p className="text-[14px] text-[#737373]">Manage scheduled tasks and workflow orchestrations.</p>
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
            Refresh
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={handleBeginCreate}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Add
          </button>
        </div>
      </header>

      <TaskList
        tasks={tasks}
        loading={loading}
        controlsDisabled={controlsDisabled}
        onEdit={handleEdit}
        onSetEnabled={onSetEnabled}
        onRunNow={onRunNow}
        onDelete={onDelete}
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
