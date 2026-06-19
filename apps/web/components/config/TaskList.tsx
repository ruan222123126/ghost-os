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
      <div className="settings-card-list">
        {Array.from({ length: TASK_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`task-skeleton-${index}`}
            className="settings-card-skeleton animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (visibleTasks.length === 0) {
    return (
      <div className="settings-list-empty">
        {copy.settings.tasksNoItems}
      </div>
    );
  }

  return (
    <div className="settings-card-list">
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
      className={`settings-config-card${editable ? ' is-clickable' : ''}`}
      onClick={() => {
        if (editable) {
          handleTaskEdit(task, onEditTextTask, onEditWorkflowTask);
        }
      }}
      onKeyDown={(event) => handleCardKeyDown(event, task, editable, onEditTextTask, onEditWorkflowTask)}
      role={editable ? 'button' : undefined}
      tabIndex={editable ? 0 : -1}
    >
      <div className="settings-config-card-main">
        <div className="settings-card-badges">
          <span className="settings-card-badge">
            {task.enabled ? copy.settings.enabled : copy.settings.disabled}
          </span>
          <span className="settings-card-badge">
            {formatTaskKind(task, copy)}
          </span>
          <span className="settings-card-id">{task.id}</span>
        </div>
        <p className="settings-card-title line-clamp-2">{formatPrimaryText(task, copy)}</p>
        <p className="settings-card-meta">{formatSchedule(task, copy)}</p>
        <p className="settings-card-meta is-mono truncate">{formatSecondaryLine(task, copy)}</p>
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
