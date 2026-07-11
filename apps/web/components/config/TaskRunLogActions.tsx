'use client';

import { type MouseEvent } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskRunLog } from '@/lib/types';

interface TaskRunLogActionsProps {
  log: TaskRunLog;
  onOpenLiveViewer: (run: TaskRunLog) => Promise<void>;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stopping: boolean;
}

export function TaskRunLogActions(props: TaskRunLogActionsProps) {
  const { log, onOpenLiveViewer, onStopRun, stopping } = props;

  return (
    <span
      className="flex items-center gap-2"
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
      }}
    >
      {log.status === 'running' && onStopRun ? (
        <TaskRunStopButton log={log} onStopRun={onStopRun} stopping={stopping} />
      ) : null}
      <TaskRunLiveViewButton log={log} onOpenLiveViewer={onOpenLiveViewer} />
    </span>
  );
}

function TaskRunStopButton(props: {
  log: TaskRunLog;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stopping: boolean;
}) {
  const { copy } = useWebLocale();
  const { log, onStopRun, stopping } = props;

  const stopRun = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!onStopRun || stopping) {
      return;
    }
    void onStopRun(log);
  };

  return (
    <button
      type="button"
      disabled={!onStopRun || stopping}
      onClick={stopRun}
      className={[
        'shrink-0 rounded-full border px-3 py-1 text-[11px] font-medium transition-colors',
        stopping
          ? 'cursor-not-allowed border-[#FCA5A5] bg-[#FEF2F2] text-[#B91C1C]'
          : 'border-[#FCA5A5] bg-white text-[#B91C1C] hover:bg-[#FEF2F2]',
      ].join(' ')}
    >
      {stopping ? copy.settings.tasksLogsStoppingRun : copy.settings.tasksLogsStopRun}
    </button>
  );
}

function TaskRunLiveViewButton(props: {
  log: TaskRunLog;
  onOpenLiveViewer: (run: TaskRunLog) => Promise<void>;
}) {
  const { copy } = useWebLocale();
  const { log, onOpenLiveViewer } = props;
  const sessionID = log.session_id_output?.trim() ?? '';
  const unavailable = sessionID.length === 0;

  const openViewer = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (unavailable) {
      return;
    }
    await onOpenLiveViewer(log);
  };

  return (
    <span title={unavailable ? copy.settings.tasksLogsLiveViewUnavailable : copy.settings.tasksLogsLiveView}>
      <button
        type="button"
        disabled={unavailable}
        onClick={openViewer}
        className={[
          'shrink-0 rounded-full border px-3 py-1 text-[11px] font-medium transition-colors',
          unavailable
            ? 'cursor-not-allowed border-[#E5E5E5] bg-[#F5F5F5] text-[#A3A3A3]'
            : 'border-[#111111] bg-[#111111] text-white hover:bg-[#404040]',
        ].join(' ')}
      >
        {copy.settings.tasksLogsLiveView}
      </button>
    </span>
  );
}
