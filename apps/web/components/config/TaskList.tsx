'use client';

import type { KeyboardEvent } from 'react';
import { ignorePromise } from '@/lib/errors';
import type { AgentMessageTaskPayload, TaskPayload } from '@/lib/types';

const TASK_SKELETON_COUNT = 3;
type WorkflowTaskPayload = Extract<TaskPayload, { task_kind: 'workflow' }>;

interface TaskListProps {
  tasks: TaskPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  onEdit: (task: TaskPayload) => void;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

interface TaskCardProps {
  task: TaskPayload;
  controlsDisabled: boolean;
  onEdit: (task: TaskPayload) => void;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onRunNow: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function TaskList(props: TaskListProps) {
  const { tasks, loading, controlsDisabled, onEdit, onSetEnabled, onRunNow, onDelete } = props;

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
        No tasks configured yet.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {tasks.map((task) => (
        <TaskCard
          key={task.id}
          task={task}
          controlsDisabled={controlsDisabled}
          onEdit={onEdit}
          onSetEnabled={onSetEnabled}
          onRunNow={onRunNow}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}

function TaskCard(props: TaskCardProps) {
  const { task, controlsDisabled, onEdit, onSetEnabled, onRunNow, onDelete } = props;
  const toggleLabel = task.enabled ? 'Disable' : 'Enable';
  const editable = task.task_kind === 'agent_message';

  return (
    <article
      className="group relative flex items-center justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]"
      onClick={() => {
        if (editable) {
          onEdit(task);
        }
      }}
      onKeyDown={(event) => handleCardKeyDown(event, task, editable, onEdit)}
      role={editable ? 'button' : undefined}
      tabIndex={editable ? 0 : -1}
    >
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2">
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {task.enabled ? 'Enabled' : 'Disabled'}
          </span>
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {formatTaskKind(task)}
          </span>
          <span className="font-mono text-[11px] text-[#737373]">{task.id}</span>
        </div>
        <p className="mb-1 line-clamp-2 text-[14px] text-[#111111]">{formatPrimaryText(task)}</p>
        <p className="text-[12px] text-[#737373]">{formatSchedule(task)}</p>
        <p className="truncate font-mono text-[12px] text-[#737373]">{formatSecondaryLine(task)}</p>
        {task.task_kind === 'agent_message' ? (
          <p className="truncate text-[12px] text-[#737373]">{formatRuntimeOverrides(task)}</p>
        ) : null}
      </div>

      <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            ignorePromise(onRunNow(task.id));
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          Run now
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
              onEdit(task);
            }
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#F5F5F5] hover:text-[#111111] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={`Edit task ${task.id}`}
          title={editable ? 'Edit task' : 'Workflow editor is not available yet'}
        >
          <EditIcon />
        </button>
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            handleTaskDelete(task.id, onDelete);
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#FEF2F2] hover:text-[#DC2626] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={`Delete task ${task.id}`}
        >
          <TrashIcon />
        </button>
      </div>
    </article>
  );
}

function formatTaskKind(task: TaskPayload): string {
  return task.task_kind === 'workflow' ? 'Workflow' : 'Text Task';
}

function formatPrimaryText(task: TaskPayload): string {
  if (task.task_kind === 'agent_message') {
    return task.message;
  }

  const workflowTask = task as WorkflowTaskPayload;
  return `Workflow with ${workflowAgentNodeCount(workflowTask)} agent steps`;
}

function formatSchedule(task: TaskPayload): string {
  if (task.schedule_type === 'interval') {
    return `Every ${task.interval_seconds ?? 0}s`;
  }

  return `Cron ${task.cron_expr ?? ''}`;
}

function formatSecondaryLine(task: TaskPayload): string {
  if (task.task_kind === 'agent_message') {
    return `Session: ${task.session_id?.trim() ? task.session_id : '(new each run)'}`;
  }

  return 'Session: workflow-internal (from selected session tasks)';
}

function formatRuntimeOverrides(task: AgentMessageTaskPayload): string {
  const overrides = task.runtime_overrides;
  if (!overrides) {
    return 'Runtime: global defaults';
  }
  const parts: string[] = [];
  if (overrides.model) {
    parts.push(`model=${overrides.model}`);
  }
  if (overrides.tool_allowlist?.length) {
    parts.push(`tools=${overrides.tool_allowlist.join(',')}`);
  }
  if (parts.length === 0) {
    return 'Runtime: global defaults';
  }

  return `Runtime: ${parts.join(' | ')}`;
}

function workflowAgentNodeCount(task: WorkflowTaskPayload): number {
  return task.workflow.nodes.filter((node) => node.type === 'agent').length;
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  task: TaskPayload,
  editable: boolean,
  onEdit: (task: TaskPayload) => void,
) {
  if (!editable) {
    return;
  }
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }

  event.preventDefault();
  onEdit(task);
}

function handleTaskDelete(id: string, onDelete: (id: string) => Promise<void>) {
  if (!window.confirm(`Delete task "${id}"?`)) {
    return;
  }

  ignorePromise(onDelete(id));
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
