'use client';

import nextDynamic from 'next/dynamic';
import { useMemo, type KeyboardEvent } from 'react';
import { ConfigCardActions } from '@/components/config/ConfigCardActions';
import type { TaskLogsModalProps } from '@/components/config/TaskLogsModal';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, PresetPayload, TaskRunLog } from '@/lib/types';

const LOOP_SKELETON_COUNT = 3;
const TaskLogsModal = nextDynamic<TaskLogsModalProps>(
  () => import('@/components/config/TaskLogsModal').then((mod) => mod.TaskLogsModal),
  { ssr: false },
);

interface LoopTaskListProps {
  loops: AgentMessageTaskPayload[];
  presets: PresetPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  runningLoopID: string;
  onEdit: (task: AgentMessageTaskPayload) => void;
  onOpenLogs: (id: string) => Promise<void>;
  onRun: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  logsTaskID: string;
  logsData: TaskRunLog[];
  logsLoading: boolean;
  logsError: string;
  onRefreshLogs: (id: string) => Promise<TaskRunLog[]>;
  onStopRun: (run: TaskRunLog) => Promise<void>;
  stoppingRunId: string;
  onCloseLogs: () => void;
}

export function LoopTaskList(props: LoopTaskListProps) {
  const { copy } = useWebLocale();
  const orderedLoops = useMemo(() => prioritizeEnabledLoops(props.loops), [props.loops]);

  if (props.loading) {
    return <LoadingList />;
  }
  if (orderedLoops.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {copy.settings.loopEmpty}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {orderedLoops.map((loop) => (
        <LoopTaskCard
          key={loop.id}
          loop={loop}
          presets={props.presets}
          controlsDisabled={props.controlsDisabled}
          running={props.runningLoopID === loop.id}
          onEdit={props.onEdit}
          onOpenLogs={props.onOpenLogs}
          onRun={props.onRun}
          onSetEnabled={props.onSetEnabled}
          onDelete={props.onDelete}
        />
      ))}
      {props.logsTaskID ? (
        <TaskLogsModal
          taskID={props.logsTaskID}
          logs={props.logsData}
          loading={props.logsLoading}
          error={props.logsError}
          onRefreshLogs={props.onRefreshLogs}
          onStopRun={props.onStopRun}
          stoppingRunId={props.stoppingRunId}
          onClose={props.onCloseLogs}
        />
      ) : null}
    </div>
  );
}

function LoopTaskCard(props: {
  loop: AgentMessageTaskPayload;
  presets: PresetPayload[];
  controlsDisabled: boolean;
  running: boolean;
  onEdit: (task: AgentMessageTaskPayload) => void;
  onOpenLogs: (id: string) => Promise<void>;
  onRun: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}) {
  const { copy } = useWebLocale();
  const { loop } = props;
  const toggleLabel = loop.enabled ? copy.settings.tasksDisable : copy.settings.tasksEnable;

  return (
    <article
      className="group relative flex items-center justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]"
      onClick={() => props.onEdit(loop)}
      onKeyDown={(event) => handleCardKeyDown(event, loop, props.onEdit)}
      role="button"
      tabIndex={0}
    >
      <LoopCardContent loop={loop} presets={props.presets} />
      <LoopCardActions
        loop={loop}
        controlsDisabled={props.controlsDisabled}
        running={props.running}
        toggleLabel={toggleLabel}
        onEdit={props.onEdit}
        onOpenLogs={props.onOpenLogs}
        onRun={props.onRun}
        onSetEnabled={props.onSetEnabled}
        onDelete={props.onDelete}
      />
    </article>
  );
}

function LoopCardContent(props: {
  loop: AgentMessageTaskPayload;
  presets: PresetPayload[];
}) {
  const { copy } = useWebLocale();
  const { loop, presets } = props;

  return (
    <div className="min-w-0">
      <LoopCardBadges loop={loop} />
      <p className="mb-1 line-clamp-2 text-[14px] text-[#111111]">{loop.message}</p>
      <p className="text-[12px] text-[#737373]">{formatSchedule(loop, copy)}</p>
      <p className="truncate text-[12px] text-[#737373]">{formatStopPolicy(loop, copy)}</p>
      <p className="truncate text-[12px] text-[#737373]">{formatPreset(loop, presets, copy)}</p>
    </div>
  );
}

function LoopCardBadges(props: { loop: AgentMessageTaskPayload }) {
  const { copy } = useWebLocale();
  const { loop } = props;

  return (
    <div className="mb-2 flex items-center gap-2">
      <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
        {loop.enabled ? copy.settings.enabled : copy.settings.disabled}
      </span>
      <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
        {copy.settings.loopKind}
      </span>
      <span className="font-mono text-[11px] text-[#737373]">{loop.id}</span>
    </div>
  );
}

function LoopCardActions(props: {
  loop: AgentMessageTaskPayload;
  controlsDisabled: boolean;
  running: boolean;
  toggleLabel: string;
  onEdit: (task: AgentMessageTaskPayload) => void;
  onOpenLogs: (id: string) => Promise<void>;
  onRun: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}) {
  const { copy } = useWebLocale();

  return (
    <ConfigCardActions
      pillActions={buildLoopCardActions({
        ...props,
        logsLabel: copy.settings.tasksLogs,
        runLabel: props.running ? copy.settings.tasksRunning : copy.settings.tasksRun,
      })}
      editLabel={copy.settings.loopEditAria(props.loop.id)}
      editTitle={copy.settings.loopEditTitle}
      editDisabled={props.controlsDisabled}
      onEdit={() => props.onEdit(props.loop)}
      deleteLabel={copy.settings.loopDeleteAria(props.loop.id)}
      deleteDisabled={props.controlsDisabled}
      onDelete={() => handleDelete(props.loop.id, props.onDelete, copy.settings.loopDeleteConfirm(props.loop.id))}
    />
  );
}

function buildLoopCardActions(options: {
  loop: AgentMessageTaskPayload;
  controlsDisabled: boolean;
  running: boolean;
  logsLabel: string;
  runLabel: string;
  toggleLabel: string;
  onOpenLogs: (id: string) => Promise<void>;
  onRun: (id: string) => Promise<void>;
  onSetEnabled: (id: string, enabled: boolean) => Promise<void>;
}) {
  return [
    {
      key: 'logs',
      label: options.logsLabel,
      disabled: options.controlsDisabled,
      onClick: () => ignorePromise(options.onOpenLogs(options.loop.id)),
    },
    {
      key: 'run',
      label: options.runLabel,
      disabled: options.controlsDisabled || options.running,
      onClick: () => ignorePromise(options.onRun(options.loop.id)),
    },
    {
      key: 'toggle',
      label: options.toggleLabel,
      disabled: options.controlsDisabled,
      onClick: () => ignorePromise(options.onSetEnabled(options.loop.id, !options.loop.enabled)),
    },
  ];
}

function formatSchedule(
  loop: AgentMessageTaskPayload,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (loop.schedule_type === 'interval') {
    return copy.settings.tasksEverySeconds(loop.interval_seconds ?? 0);
  }
  return copy.settings.tasksCron(loop.cron_expr ?? '');
}

function formatStopPolicy(
  loop: AgentMessageTaskPayload,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  const policy = loop.relay?.stop_policy ?? 'ai_decides';
  if (policy === 'max_rounds') {
    return copy.settings.loopCardMaxRounds(loop.relay?.max_rounds ?? 0);
  }
  return copy.settings.loopCardAIDecides(loop.relay?.max_rounds ?? 0);
}

function formatPreset(
  loop: AgentMessageTaskPayload,
  presets: PresetPayload[],
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  const presetID = loop.runtime_overrides?.preset_id?.trim();
  if (!presetID) {
    return copy.settings.loopCardPresetNone;
  }
  const preset = presets.find((item) => item.id === presetID);
  return copy.settings.loopCardPreset(preset?.name ?? presetID);
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  loop: AgentMessageTaskPayload,
  onEdit: (task: AgentMessageTaskPayload) => void,
) {
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }
  event.preventDefault();
  onEdit(loop);
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
      {Array.from({ length: LOOP_SKELETON_COUNT }).map((_, index) => (
        <div
          key={`loop-skeleton-${index}`}
          className="h-[130px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
        />
      ))}
    </div>
  );
}

function prioritizeEnabledLoops(tasks: AgentMessageTaskPayload[]): AgentMessageTaskPayload[] {
  const enabledTasks: AgentMessageTaskPayload[] = [];
  const disabledTasks: AgentMessageTaskPayload[] = [];

  for (const task of tasks) {
    if (task.enabled) {
      enabledTasks.push(task);
      continue;
    }
    disabledTasks.push(task);
  }
  return [...enabledTasks, ...disabledTasks];
}
