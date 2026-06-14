'use client';

import nextDynamic from 'next/dynamic';
import { useMemo, type KeyboardEvent } from 'react';
import { ConfigCardActions } from '@/components/config/ConfigCardActions';
import type { TaskLogsModalProps } from '@/components/config/TaskLogsModal';
import { filterTaskSettingsTasks, type TaskSettingsListItem } from '@/lib/configTasks';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, TaskPayload, TaskRunLog, WorkflowTaskPayload } from '@/lib/types';

const TASK_SKELETON_COUNT = 3;
const TaskLogsModal = nextDynamic<TaskLogsModalProps>(
  () => import('@/components/config/TaskLogsModal').then((mod) => mod.TaskLogsModal),
  { ssr: false },
);

interface TaskListProps {
  tasks: TaskPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  onEditTextTask: (task: AgentMessageTaskPayload) => void;
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<unknown>;
  onDelete: (id: string) => Promise<void>;
  logsTaskID: string;
  logsData: TaskRunLog[];
  logsLoading: boolean;
  logsError: string;
  onOpenLogs: (id: string) => Promise<void>;
  onRefreshLogs: (id: string) => Promise<TaskRunLog[]>;
  onStopRun: (run: TaskRunLog) => Promise<void>;
  stoppingRunId: string;
  onCloseLogs: () => void;
}

interface TaskCardProps {
  task: TaskSettingsListItem;
  controlsDisabled: boolean;
  onEditTextTask: (task: AgentMessageTaskPayload) => void;
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void;
  onOpenLogs: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<unknown>;
  onDelete: (id: string) => Promise<void>;
}

export function TaskList(props: TaskListProps) {
  const { copy } = useWebLocale();
  const {
    tasks,
    loading,
    controlsDisabled,
    onEditTextTask,
    onEditWorkflowTask,
    onSetEnabled,
    onRunNow,
    onDelete,
    logsTaskID,
    logsData,
    logsLoading,
    logsError,
    onOpenLogs,
    onRefreshLogs,
    onStopRun,
    stoppingRunId,
    onCloseLogs,
  } = props;
  const visibleTasks = useMemo(() => filterTaskSettingsTasks(tasks), [tasks]);
  const orderedTasks = useMemo(() => prioritizeEnabledTasks(visibleTasks), [visibleTasks]);

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-3">
        {Array.from({ length: TASK_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`task-skeleton-${index}`}
            className="h-[130px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
          />
        ))}
      </div>
    );
  }

  if (visibleTasks.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {copy.settings.tasksNoItems}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {orderedTasks.map((task) => (
        <TaskCard
          key={task.id}
          task={task}
          controlsDisabled={controlsDisabled}
          onEditTextTask={onEditTextTask}
          onEditWorkflowTask={onEditWorkflowTask}
          onOpenLogs={onOpenLogs}
          onSetEnabled={onSetEnabled}
          onRunNow={onRunNow}
          onDelete={onDelete}
        />
      ))}
      {logsTaskID ? (
        <TaskLogsModal
          taskID={logsTaskID}
          logs={logsData}
          loading={logsLoading}
          error={logsError}
          onRefreshLogs={onRefreshLogs}
          onStopRun={onStopRun}
          stoppingRunId={stoppingRunId}
          onClose={onCloseLogs}
        />
      ) : null}
    </div>
  );
}

function TaskCard(props: TaskCardProps) {
  const { copy } = useWebLocale();
  const { task, controlsDisabled, onEditTextTask, onEditWorkflowTask, onOpenLogs, onSetEnabled, onRunNow, onDelete } = props;
  const toggleLabel = task.enabled ? copy.settings.tasksDisable : copy.settings.tasksEnable;
  const editable = task.task_kind === 'agent_message' || task.task_kind === 'workflow';

  return (
    <article
      className="group relative flex items-center justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]"
      onClick={() => {
        if (editable) {
          handleTaskEdit(task, onEditTextTask, onEditWorkflowTask);
        }
      }}
      onKeyDown={(event) => handleCardKeyDown(event, task, editable, onEditTextTask, onEditWorkflowTask)}
      role={editable ? 'button' : undefined}
      tabIndex={editable ? 0 : -1}
    >
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2">
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {task.enabled ? copy.settings.enabled : copy.settings.disabled}
          </span>
          <span className="inline-flex whitespace-nowrap rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {formatTaskKind(task, copy)}
          </span>
          <span className="font-mono text-[11px] text-[#737373]">{task.id}</span>
        </div>
        <p className="mb-1 line-clamp-2 text-[14px] text-[#111111]">{formatPrimaryText(task, copy)}</p>
        <p className="text-[12px] text-[#737373]">{formatSchedule(task, copy)}</p>
        <p className="truncate font-mono text-[12px] text-[#737373]">{formatSecondaryLine(task, copy)}</p>
        {task.task_kind === 'agent_message' ? (
          <>
            <p className="truncate text-[12px] text-[#737373]">{formatAgentMode(copy)}</p>
            <p className="truncate text-[12px] text-[#737373]">{formatRuntimeOverrides(task, copy)}</p>
          </>
        ) : null}
      </div>

      <ConfigCardActions
        pillActions={[
          {
            key: 'logs',
            label: copy.settings.tasksLogs,
            disabled: controlsDisabled,
            onClick: () => ignorePromise(onOpenLogs(task.id)),
          },
          {
            key: 'run',
            label: copy.settings.tasksRun,
            disabled: controlsDisabled,
            onClick: () => ignorePromise(onRunNow(task.id)),
          },
          {
            key: 'toggle',
            label: toggleLabel,
            disabled: controlsDisabled,
            onClick: () => ignorePromise(onSetEnabled(task.id, !task.enabled)),
          },
        ]}
        editLabel={copy.settings.tasksEditAria(task.id)}
        editTitle={editable ? copy.settings.tasksEditTitle : copy.settings.tasksEditUnavailableTitle}
        editDisabled={controlsDisabled || !editable}
        onEdit={() => {
          if (editable) {
            handleTaskEdit(task, onEditTextTask, onEditWorkflowTask);
          }
        }}
        deleteLabel={copy.settings.tasksDeleteAria(task.id)}
        deleteDisabled={controlsDisabled}
        onDelete={() => handleTaskDelete(task.id, onDelete, copy.settings.tasksDeleteConfirm(task.id))}
      />
    </article>
  );
}

function formatTaskKind(task: TaskSettingsListItem, copy: ReturnType<typeof useWebLocale>['copy']): string {
  return task.task_kind === 'workflow' ? copy.settings.tasksWorkflowKind : copy.settings.tasksTextKind;
}

function formatPrimaryText(task: TaskSettingsListItem, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.task_kind === 'agent_message') {
    return task.message;
  }

  return copy.settings.tasksWorkflowWithSteps(workflowStepCount(task));
}

function formatSchedule(task: TaskSettingsListItem, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.schedule_type === 'interval') {
    return copy.settings.tasksEverySeconds(task.interval_seconds ?? 0);
  }

  return copy.settings.tasksCron(task.cron_expr ?? '');
}

function formatSecondaryLine(task: TaskSettingsListItem, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.task_kind === 'agent_message') {
    return copy.settings.tasksSessionLabel(task.session_id?.trim() ? copy.settings.tasksSessionConfigured : copy.settings.tasksSessionNewEachRun);
  }

  return copy.settings.tasksSessionWorkflowManaged;
}

function formatRuntimeOverrides(task: AgentMessageTaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  const overrides = task.runtime_overrides;
  if (!overrides) {
    return copy.settings.tasksRuntimeGlobalDefaults;
  }
  const parts: string[] = [];
  if (overrides.preset_id) {
    parts.push(`preset=${overrides.preset_id}`);
  }
  if (overrides.model) {
    parts.push(`model=${overrides.model}`);
  }
  if (overrides.tool_allowlist?.length) {
    parts.push(`tools=${overrides.tool_allowlist.join(',')}`);
  }
  if (parts.length === 0) {
    return copy.settings.tasksRuntimeGlobalDefaults;
  }

  return copy.settings.tasksRuntimeLabel(parts.join(' | '));
}

function formatAgentMode(copy: ReturnType<typeof useWebLocale>['copy']): string {
  return copy.settings.tasksAgentModeSingle;
}

function workflowStepCount(task: WorkflowTaskPayload): number {
  return task.workflow.nodes.filter((node) => node.type !== 'start' && node.type !== 'end').length;
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  task: TaskSettingsListItem,
  editable: boolean,
  onEditTextTask: (task: AgentMessageTaskPayload) => void,
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void,
) {
  if (!editable) {
    return;
  }
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }

  event.preventDefault();
  handleTaskEdit(task, onEditTextTask, onEditWorkflowTask);
}

function handleTaskEdit(
  task: TaskSettingsListItem,
  onEditTextTask: (task: AgentMessageTaskPayload) => void,
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void,
) {
  if (task.task_kind === 'agent_message') {
    onEditTextTask(task);
    return;
  }
  if (task.task_kind === 'workflow') {
    onEditWorkflowTask(task);
  }
}

function handleTaskDelete(id: string, onDelete: (id: string) => Promise<void>, message: string) {
  if (!globalThis.confirm(message)) {
    return;
  }

  ignorePromise(onDelete(id));
}

function prioritizeEnabledTasks(tasks: TaskSettingsListItem[]): TaskSettingsListItem[] {
  const enabledTasks: TaskSettingsListItem[] = [];
  const disabledTasks: TaskSettingsListItem[] = [];

  for (const task of tasks) {
    if (task.enabled) {
      enabledTasks.push(task);
      continue;
    }
    disabledTasks.push(task);
  }
  return [...enabledTasks, ...disabledTasks];
}
