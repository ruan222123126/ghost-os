'use client';

import { useMemo } from 'react';
import { TaskEditorForm } from '@/components/config/TaskEditorForm';
import { TaskList } from '@/components/config/TaskList';
import { useTaskLogs } from '@/hooks/config/useTaskLogs';
import { filterTaskSettingsTasks } from '@/lib/configTasks';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { useConfigTasks } from '@/hooks/useConfigTasks';
import type { PresetPayload, TaskPayload, WorkflowTaskPayload } from '@/lib/types';

interface TaskSettingsSectionProps {
  machine: ReturnType<typeof useConfigTasks>;
  presets: PresetPayload[];
  onOpenWorkflowCreate: () => void;
  onOpenWorkflowEdit: (task: WorkflowTaskPayload) => void;
}

export function TaskSettingsSection(props: TaskSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { machine, presets, onOpenWorkflowCreate, onOpenWorkflowEdit } = props;
  const { state, actions } = machine;
  const controlsDisabled = state.loading || state.saving;
  const visibleTasks = useMemo(() => filterTaskSettingsTasks(state.tasks), [state.tasks]);
  const sessionOptions = useMemo(() => buildSessionOptions(visibleTasks), [visibleTasks]);
  const logs = useTaskLogs({
    empty: copy.settings.tasksLogsEmpty,
    stopFailed: copy.system.failedToStopTask,
  });

  if (state.view === 'editor') {
    return (
      <TaskEditorForm
        editorMode={state.editorMode}
        editor={state.editor}
        presets={presets}
        sessionOptions={sessionOptions}
        controlsDisabled={controlsDisabled}
        saving={state.saving}
        onChangeEditor={actions.updateEditor}
        onSelectTaskType={actions.selectTaskType}
        onSubmit={async () => {
          await actions.submit();
        }}
        onCancelEditing={actions.cancelEditing}
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
              ignorePromise(actions.refresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.refresh}
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={actions.startCreateTask}
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
        tasks={visibleTasks}
        presets={presets}
        loading={state.loading}
        controlsDisabled={controlsDisabled}
        onEditTextTask={actions.startEditTextTask}
        onEditWorkflowTask={onOpenWorkflowEdit}
        onSetEnabled={actions.setEnabled}
        onRunNow={async (id) => {
          await actions.runNow(id, { reportSuccess: true });
        }}
        onDelete={actions.delete}
        logsTaskID={logs.taskID}
        logsData={logs.entries}
        logsLoading={logs.loading}
        logsError={logs.error}
        onOpenLogs={logs.open}
        onRefreshLogs={logs.refresh}
        onStopRun={logs.stopRun}
        stoppingRunId={logs.stoppingRunId}
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
