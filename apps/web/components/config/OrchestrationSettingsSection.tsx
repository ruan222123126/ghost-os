'use client';

import { type KeyboardEvent, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { ConfigCardActions } from '@/components/config/ConfigCardActions';
import {
  OrchestrationLoadingList,
  OrchestrationStatusBanner,
} from '@/components/config/OrchestrationSettingsSectionParts';
import { TaskLogsModal } from '@/components/config/TaskLogsModal';
import { OrchestrationCreateForm } from '@/components/config/OrchestrationCreateForm';
import { useOrchestrationLogs } from '@/hooks/config/useOrchestrationLogs';
import { useOrchestrationSectionState } from '@/hooks/config/useOrchestrationSectionState';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { OrchestrationTaskPayload } from '@/lib/types';

export function OrchestrationSettingsSection() {
  const { copy } = useWebLocale();
  const router = useRouter();
  const machine = useOrchestrationSectionState();
  const logs = useOrchestrationLogs({
    empty: copy.settings.tasksLogsEmpty,
    stopFailed: copy.system.failedToStopTask,
  });
  const { state, actions } = machine;
  const { open: openLogs, taskID: logsTaskID } = logs;
  const handleRun = useCallback(async (taskID: string) => {
    await actions.runByID(taskID);
    if (logsTaskID === taskID) {
      await openLogs(taskID);
    }
  }, [actions, logsTaskID, openLogs]);

  if (state.view === 'create') {
    return (
      <OrchestrationCreateForm
        name={state.name}
        controlsDisabled={state.loading || state.submitting}
        saving={state.submitting}
        onChangeName={actions.setName}
        onSubmit={actions.submitCreate}
        onCancel={actions.cancelCreate}
      />
    );
  }

  return (
    <section data-testid="orchestration-settings-section">
      <SectionHeader
        title={copy.settings.orchestrationTitle}
        description={copy.settings.orchestrationDescription}
        actionLabel={copy.settings.orchestrationNew}
        onAction={actions.startCreate}
      />
      <OrchestrationStatusBanner
        message={state.error || state.success}
        tone={state.error ? 'error' : 'success'}
      />
      <OrchestrationList
        orchestrations={state.orchestrations}
        loading={state.loading}
        emptyText={copy.settings.orchestrationEmpty}
        onOpenEditor={(taskID) => router.push(`/orchestration/${encodeURIComponent(taskID)}`)}
        onRun={handleRun}
        runningOrchestrationID={state.runningOrchestrationID}
        onOpenLogs={openLogs}
        onToggleEnabled={actions.setEnabledByID}
        onDelete={actions.deleteByID}
        controlsDisabled={state.loading || state.submitting}
      />
      {logs.taskID ? (
        <TaskLogsModal
          taskID={logs.taskID}
          logs={logs.entries}
          loading={logs.loading}
          error={logs.error}
          onRefreshLogs={logs.refresh}
          onStopRun={logs.stopRun}
          stoppingRunId={logs.stoppingRunId}
          onClose={logs.close}
        />
      ) : null}
    </section>
  );
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
    return <OrchestrationLoadingList />;
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
            label: copy.settings.tasksRun,
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
