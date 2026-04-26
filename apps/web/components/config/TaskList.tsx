'use client';

import { useMemo, useState, type KeyboardEvent } from 'react';
import { listTaskLogs } from '@/lib/api/tasks/api';
import { TaskLogsModal } from '@/components/config/TaskLogsModal';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, TaskPayload, TaskRunLog, WorkflowTaskPayload } from '@/lib/types';

const TASK_SKELETON_COUNT = 3;

interface TaskListProps {
  tasks: TaskPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  onEditTextTask: (task: AgentMessageTaskPayload) => void;
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

interface TaskCardProps {
  task: TaskPayload;
  controlsDisabled: boolean;
  onEditTextTask: (task: AgentMessageTaskPayload) => void;
  onEditWorkflowTask: (task: WorkflowTaskPayload) => void;
  onOpenLogs: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function TaskList(props: TaskListProps) {
  const { copy } = useWebLocale();
  const { tasks, loading, controlsDisabled, onEditTextTask, onEditWorkflowTask, onSetEnabled, onRunNow, onDelete } = props;
  const orderedTasks = useMemo(() => prioritizeEnabledTasks(tasks), [tasks]);
  const [logsTaskID, setLogsTaskID] = useState('');
  const [logsData, setLogsData] = useState<TaskRunLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsError, setLogsError] = useState('');

  const openLogs = async (taskID: string): Promise<void> => {
    setLogsTaskID(taskID);
    setLogsLoading(true);
    setLogsError('');
    setLogsData([]);
    try {
      setLogsData(await listTaskLogs(taskID, 20));
    } catch (error) {
      setLogsError(toErrorMessage(error, copy.settings.tasksLogsEmpty));
    } finally {
      setLogsLoading(false);
    }
  };

  const closeLogs = () => {
    setLogsTaskID('');
    setLogsData([]);
    setLogsError('');
    setLogsLoading(false);
  };

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

  if (tasks.length === 0) {
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
          onOpenLogs={openLogs}
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
          onClose={closeLogs}
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
          <p className="truncate text-[12px] text-[#737373]">{formatRuntimeOverrides(task, copy)}</p>
        ) : null}
      </div>

      <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            ignorePromise(onOpenLogs(task.id));
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.tasksLogs}
        </button>
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            ignorePromise(onRunNow(task.id));
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.tasksRun}
        </button>
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            ignorePromise(onSetEnabled(task.id, !task.enabled));
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {toggleLabel}
        </button>
        <button
          type="button"
          disabled={controlsDisabled || !editable}
          onClick={(event) => {
            event.stopPropagation();
            if (editable) {
              handleTaskEdit(task, onEditTextTask, onEditWorkflowTask);
            }
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#F5F5F5] hover:text-[#111111] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={copy.settings.tasksEditAria(task.id)}
          title={editable ? copy.settings.tasksEditTitle : copy.settings.tasksEditUnavailableTitle}
        >
          <EditIcon />
        </button>
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            handleTaskDelete(task.id, onDelete, copy.settings.tasksDeleteConfirm(task.id));
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#FEF2F2] hover:text-[#DC2626] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={copy.settings.tasksDeleteAria(task.id)}
        >
          <TrashIcon />
        </button>
      </div>
    </article>
  );
}

function formatTaskKind(task: TaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  return task.task_kind === 'workflow' ? copy.settings.tasksWorkflowKind : copy.settings.tasksTextKind;
}

function formatPrimaryText(task: TaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.task_kind === 'agent_message') {
    return task.message;
  }

  const workflowTask = task as WorkflowTaskPayload;
  return copy.settings.tasksWorkflowWithSteps(workflowStepCount(workflowTask));
}

function formatSchedule(task: TaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.schedule_type === 'interval') {
    return copy.settings.tasksEverySeconds(task.interval_seconds ?? 0);
  }

  return copy.settings.tasksCron(task.cron_expr ?? '');
}

function formatSecondaryLine(task: TaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (task.task_kind === 'agent_message') {
    return copy.settings.tasksSessionLabel(task.session_id?.trim() ? task.session_id : copy.settings.tasksSessionNewEachRun);
  }

  return copy.settings.tasksSessionWorkflowManaged;
}

function formatRuntimeOverrides(task: AgentMessageTaskPayload, copy: ReturnType<typeof useWebLocale>['copy']): string {
  const overrides = task.runtime_overrides;
  if (!overrides) {
    return copy.settings.tasksRuntimeGlobalDefaults;
  }
  const parts: string[] = [];
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

function workflowStepCount(task: WorkflowTaskPayload): number {
  return task.workflow.nodes.filter((node) => node.type !== 'start' && node.type !== 'end').length;
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  task: TaskPayload,
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
  task: TaskPayload,
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
  if (!window.confirm(message)) {
    return;
  }

  ignorePromise(onDelete(id));
}

function prioritizeEnabledTasks(tasks: TaskPayload[]): TaskPayload[] {
  const enabledTasks: TaskPayload[] = [];
  const disabledTasks: TaskPayload[] = [];

  for (const task of tasks) {
    if (task.enabled) {
      enabledTasks.push(task);
      continue;
    }
    disabledTasks.push(task);
  }
  return [...enabledTasks, ...disabledTasks];
}

function EditIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="m4.6 13.9 8.9-8.9 1.8 1.8-8.9 8.9-2.4.6.6-2.4Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
      <path d="m12.7 5.8 1.8 1.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M4.6 5.5h10.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
      <path d="M7.3 5.5V4.2h5.4v1.3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
      <path d="m6.3 5.5.8 10h5.8l.8-10" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    </svg>
  );
}
