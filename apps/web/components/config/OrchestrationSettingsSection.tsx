'use client';

import { type KeyboardEvent, useCallback, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ConfigCardActions } from '@/components/config/ConfigCardActions';
import { TaskLogsModal } from '@/components/config/TaskLogsModal';
import { OrchestrationCreateForm } from '@/components/config/OrchestrationCreateForm';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { listOrchestrationLogs } from '@/lib/api/orchestrations/api';
import type { OrchestrationTaskPayload, TaskRunLog } from '@/lib/types';
import { useOrchestrationSectionState } from './useOrchestrationSectionState';

export function OrchestrationSettingsSection() {
  const { copy } = useWebLocale();
  const router = useRouter();
  const state = useOrchestrationSectionState(copy);
  const logs = useOrchestrationLogs(copy.settings.tasksLogsEmpty);
  const handleRun = useCallback(async (taskID: string) => {
    await state.runByID(taskID);
    if (logs.taskID === taskID) {
      await logs.open(taskID);
    }
  }, [logs.open, logs.taskID, state.runByID]);

  if (state.creating) {
    return (
      <OrchestrationCreateForm
        name={state.name}
        controlsDisabled={state.controlsDisabled}
        saving={state.submitting}
        onChangeName={state.setName}
        onSubmit={state.submitCreate}
        onCancel={state.cancelCreate}
      />
    );
  }

  return (
    <section data-testid="orchestration-settings-section">
      <SectionHeader
        title={copy.settings.orchestrationTitle}
        description={copy.settings.orchestrationDescription}
        actionLabel={copy.settings.orchestrationNew}
        onAction={state.startCreate}
      />
      {state.error ? <div className="mb-4 rounded-[16px] border border-red-200 bg-red-50 px-6 py-4 text-[13px] text-red-700">{state.error}</div> : null}
      <OrchestrationList
        orchestrations={state.orchestrations}
        loading={state.loading}
        emptyText={copy.settings.orchestrationEmpty}
        onOpenEditor={(taskID) => router.push(`/orchestration/${encodeURIComponent(taskID)}`)}
        onRun={handleRun}
        runningOrchestrationID={state.runningOrchestrationID}
        onOpenLogs={logs.open}
        onToggleEnabled={state.setEnabledByID}
        onDelete={state.deleteByID}
        controlsDisabled={state.controlsDisabled}
      />
      {logs.taskID ? (
        <TaskLogsModal
          taskID={logs.taskID}
          logs={logs.entries}
          loading={logs.loading}
          error={logs.error}
          onClose={logs.close}
        />
      ) : null}
    </section>
  );
}

function useOrchestrationLogs(emptyMessage: string) {
  const [taskID, setTaskID] = useState('');
  const [entries, setEntries] = useState<TaskRunLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const open = useCallback(async (nextTaskID: string) => {
    setTaskID(nextTaskID);
    setEntries([]);
    setError('');
    setLoading(true);
    try {
      setEntries(await listOrchestrationLogs(nextTaskID, 20));
    } catch (nextError) {
      setError(toErrorMessage(nextError, emptyMessage));
    } finally {
      setLoading(false);
    }
  }, [emptyMessage]);

  const close = useCallback(() => {
    setTaskID('');
    setEntries([]);
    setError('');
  }, []);

  return { taskID, entries, loading, error, open, close };
}

function SectionHeader(props: {
  title: string;
  description: string;
  actionLabel: string;
  onAction: () => void;
}) {
  return (
    <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{props.title}</h1>
        <p className="text-[14px] text-[#737373]">{props.description}</p>
      </div>
      <button
        type="button"
        onClick={props.onAction}
        className="inline-flex items-center rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333]"
      >
        {props.actionLabel}
      </button>
    </header>
  );
}

function OrchestrationList(props: {
  orchestrations: OrchestrationTaskPayload[];
  loading: boolean;
  emptyText: string;
  controlsDisabled: boolean;
  onOpenEditor: (taskID: string) => void;
  onRun: (taskID: string) => Promise<void>;
  runningOrchestrationID: string;
  onOpenLogs: (taskID: string) => Promise<void>;
  onToggleEnabled: (taskID: string, enabled: boolean) => Promise<void>;
  onDelete: (taskID: string) => Promise<void>;
}) {
  const { copy } = useWebLocale();
  const orderedOrchestrations = useMemo(
    () => prioritizeEnabledOrchestrations(props.orchestrations),
    [props.orchestrations],
  );

  if (props.loading) {
    return <LoadingList />;
  }
  if (orderedOrchestrations.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {props.emptyText}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {orderedOrchestrations.map((task) => (
        <OrchestrationCard
          key={task.id}
          task={task}
          controlsDisabled={props.controlsDisabled}
          onOpenEditor={props.onOpenEditor}
          onRun={props.onRun}
          running={props.runningOrchestrationID === task.id}
          onOpenLogs={props.onOpenLogs}
          onToggleEnabled={props.onToggleEnabled}
          onDelete={props.onDelete}
          statusLabel={task.enabled ? copy.settings.enabled : copy.settings.disabled}
          toggleLabel={task.enabled ? copy.settings.tasksDisable : copy.settings.tasksEnable}
        />
      ))}
    </div>
  );
}

function OrchestrationCard(props: {
  task: OrchestrationTaskPayload;
  controlsDisabled: boolean;
  running: boolean;
  statusLabel: string;
  toggleLabel: string;
  onOpenEditor: (taskID: string) => void;
  onRun: (taskID: string) => Promise<void>;
  onOpenLogs: (taskID: string) => Promise<void>;
  onToggleEnabled: (taskID: string, enabled: boolean) => Promise<void>;
  onDelete: (taskID: string) => Promise<void>;
}) {
  const { copy } = useWebLocale();
  const groupCount = props.task.orchestration.nodes.filter((node) => node.type === 'group').length;

  return (
    <article
      className="group relative flex items-center justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]"
      onClick={() => props.onOpenEditor(props.task.id)}
      onKeyDown={(event) => handleCardKeyDown(event, props.task.id, props.onOpenEditor)}
      role="button"
      tabIndex={0}
    >
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2">
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {props.statusLabel}
          </span>
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {copy.settings.tabOrchestration}
          </span>
          <span className="font-mono text-[11px] text-[#737373]">{props.task.id}</span>
        </div>
        <p className="mb-1 line-clamp-2 text-[14px] text-[#111111]">{props.task.name}</p>
        <p className="text-[12px] text-[#737373]">{copy.settings.orchestrationSteps(groupCount)}</p>
        <p className="truncate text-[12px] text-[#737373]">{formatSchedule(props.task, copy)}</p>
      </div>
      <ConfigCardActions
        pillActions={[
          {
            key: 'logs',
            label: copy.settings.tasksLogs,
            disabled: props.controlsDisabled,
            onClick: () => ignorePromise(props.onOpenLogs(props.task.id)),
          },
          {
            key: 'run',
            label: props.running ? copy.settings.tasksRunning : copy.settings.tasksRun,
            disabled: props.controlsDisabled || props.running,
            onClick: () => ignorePromise(props.onRun(props.task.id)),
          },
          {
            key: 'toggle',
            label: props.toggleLabel,
            disabled: props.controlsDisabled,
            onClick: () => ignorePromise(props.onToggleEnabled(props.task.id, !props.task.enabled)),
          },
        ]}
        editLabel={copy.settings.orchestrationEditAria(props.task.id)}
        editTitle={copy.settings.orchestrationOpenEditor}
        editDisabled={props.controlsDisabled}
        onEdit={() => props.onOpenEditor(props.task.id)}
        deleteLabel={copy.settings.orchestrationDeleteAria(props.task.id)}
        deleteDisabled={props.controlsDisabled}
        onDelete={() => handleDelete(props.task.id, props.onDelete, copy.settings.orchestrationDeleteConfirm(props.task.id))}
      />
    </article>
  );
}

function formatSchedule(
  task: OrchestrationTaskPayload,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (task.schedule_type === 'interval') {
    return copy.settings.tasksEverySeconds(task.interval_seconds ?? 0);
  }

  return copy.settings.tasksCron(task.cron_expr ?? '');
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  taskID: string,
  onOpenEditor: (taskID: string) => void,
) {
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }

  event.preventDefault();
  onOpenEditor(taskID);
}

function handleDelete(
  id: string,
  onDelete: (id: string) => Promise<void>,
  message: string,
) {
  if (!globalThis.confirm(message)) {
    return;
  }

  ignorePromise(onDelete(id));
}

function LoadingList() {
  return (
    <div className="grid grid-cols-1 gap-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <div
          key={`orchestration-skeleton-${index}`}
          className="h-[130px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
        />
      ))}
    </div>
  );
}

function prioritizeEnabledOrchestrations(tasks: OrchestrationTaskPayload[]): OrchestrationTaskPayload[] {
  const enabledTasks: OrchestrationTaskPayload[] = [];
  const disabledTasks: OrchestrationTaskPayload[] = [];

  for (const task of tasks) {
    if (task.enabled) {
      enabledTasks.push(task);
      continue;
    }
    disabledTasks.push(task);
  }

  return [...enabledTasks, ...disabledTasks];
}
