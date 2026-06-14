'use client';

import { useVirtualizer } from '@tanstack/react-virtual';
import { useEffect, useRef, useState } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { formatTaskRunStatus, formatTaskRunTimestamp } from '@/lib/taskRunDisplay';
import type { TaskRunLog } from '@/lib/types';
import { LiveRunViewerModal, type LiveRunViewerTarget } from './LiveRunViewerModal';
import { TaskRunLogActions } from './TaskRunLogActions';

const LIVE_VIEW_RUN_REFRESH_MS = 2000;
const TASK_LOG_INITIAL_VIEWPORT_PX = 640;
const TASK_LOG_ROW_ESTIMATE_PX = 120;
const TASK_LOG_ROW_OVERSCAN = 6;

export interface TaskLogsModalProps {
  taskID: string;
  logs: TaskRunLog[];
  loading: boolean;
  error: string;
  onRefreshLogs?: (taskID: string) => Promise<TaskRunLog[]>;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stoppingRunId?: string;
  onClose: () => void;
}

export function TaskLogsModal(props: TaskLogsModalProps) {
  const { copy } = useWebLocale();
  const {
    taskID,
    logs,
    loading,
    error,
    onClose,
    onRefreshLogs,
    onStopRun,
    stoppingRunId,
  } = props;
  const [viewerTarget, setViewerTarget] = useState<LiveRunViewerTarget | null>(null);
  const [viewerError, setViewerError] = useState('');

  useEffect(() => {
    if (!viewerTarget) {
      return;
    }
    const latest = findRunLog(logs, viewerTarget.run.run_id);
    if (!latest || !shouldReplaceViewerRun(viewerTarget.run, latest)) {
      return;
    }
    setViewerTarget({ run: latest });
  }, [logs, viewerTarget]);

  useEffect(() => {
    if (!viewerTarget || !onRefreshLogs || !isRunningRun(viewerTarget.run)) {
      return undefined;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const refreshOpenRun = async () => {
      try {
        const refreshed = await onRefreshLogs(taskID);
        if (cancelled) {
          return;
        }
        const latest = findRunLog(refreshed, viewerTarget.run.run_id);
        if (!latest) {
          return;
        }
        setViewerError('');
        setViewerTarget({ run: latest });
        if (isRunningRun(latest)) {
          timer = setTimeout(refreshOpenRun, LIVE_VIEW_RUN_REFRESH_MS);
        }
      } catch (nextError) {
        if (!cancelled) {
          setViewerError(toErrorMessage(nextError));
          timer = setTimeout(refreshOpenRun, LIVE_VIEW_RUN_REFRESH_MS);
        }
      }
    };

    timer = setTimeout(refreshOpenRun, LIVE_VIEW_RUN_REFRESH_MS);
    return () => {
      cancelled = true;
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [onRefreshLogs, taskID, viewerTarget]);

  const openLiveViewer = async (run: TaskRunLog) => {
    setViewerError('');
    try {
      const latest = await resolveViewerRun(taskID, run, onRefreshLogs);
      setViewerTarget({ run: latest });
    } catch (nextError) {
      setViewerError(toErrorMessage(nextError));
    }
  };

  return (
    <>
      <div className="fixed inset-0 z-[80] flex items-center justify-center p-4" role="dialog" aria-modal="true">
        <button type="button" className="absolute inset-0 bg-black/35 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
        <section className="relative z-[81] flex h-[78vh] w-full max-w-[900px] flex-col overflow-hidden rounded-[18px] border border-[#E5E5E5] bg-white shadow-2xl">
          <TaskLogsModalHeader taskID={taskID} onClose={onClose} />
          <TaskLogsModalBody
            logs={logs}
            loading={loading}
            error={error || viewerError}
            onOpenLiveViewer={openLiveViewer}
            onStopRun={onStopRun}
            stoppingRunId={stoppingRunId}
          />
        </section>
      </div>
      {viewerTarget ? (
        <LiveRunViewerModal
          run={viewerTarget.run}
          onClose={() => setViewerTarget(null)}
        />
      ) : null}
    </>
  );
}

function findRunLog(logs: TaskRunLog[], runID: string): TaskRunLog | undefined {
  return logs.find((entry) => entry.run_id === runID);
}

function isRunningRun(run: TaskRunLog): boolean {
  return run.status === 'running';
}

function shouldReplaceViewerRun(current: TaskRunLog, next: TaskRunLog): boolean {
  if (current === next) {
    return false;
  }
  const currentRank = runStatusRank(current);
  const nextRank = runStatusRank(next);
  if (nextRank < currentRank) {
    return false;
  }
  if (nextRank > currentRank) {
    return true;
  }
  if (newerTimestamp(next.finished_at, current.finished_at)) {
    return true;
  }
  if (currentRank === 2 && nextRank === 2) {
    return false;
  }
  return current.status !== next.status
    || current.response_preview !== next.response_preview
    || current.error !== next.error
    || current.run_cards !== next.run_cards;
}

function runStatusRank(run: TaskRunLog): number {
  if (run.status === 'running') {
    return 0;
  }
  if (run.status === 'awaiting_human') {
    return 1;
  }
  return 2;
}

function newerTimestamp(next?: string, current?: string): boolean {
  const nextTime = Date.parse(next ?? '');
  if (!Number.isFinite(nextTime)) {
    return false;
  }
  const currentTime = Date.parse(current ?? '');
  return !Number.isFinite(currentTime) || nextTime >= currentTime;
}

function TaskLogsModalHeader(props: { taskID: string; onClose: () => void }) {
  const { copy } = useWebLocale();
  const { taskID, onClose } = props;
  return (
    <header className="flex items-center justify-between border-b border-[#E5E5E5] px-5 py-4">
      <h2 className="text-[15px] font-semibold text-[#111111]">{copy.settings.tasksLogsTitle(taskID)}</h2>
      <CloseButton
        onClick={onClose}
        className="shrink-0"
        aria-label={copy.settings.tasksLogsClose}
      />
    </header>
  );
}

function TaskLogsModalBody(props: {
  logs: TaskRunLog[];
  loading: boolean;
  error: string;
  onOpenLiveViewer: (run: TaskRunLog) => Promise<void>;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stoppingRunId?: string;
}) {
  const { copy } = useWebLocale();
  const { logs, loading, error, onOpenLiveViewer, onStopRun, stoppingRunId } = props;

  if (loading) {
    return <div className="p-5 text-[13px] text-[#737373]">{copy.settings.tasksLogsLoading}</div>;
  }
  if (error) {
    return <div className="m-5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-3 py-2 text-[12px] text-[#B91C1C]">{error}</div>;
  }
  if (logs.length === 0) {
    return <div className="p-5 text-[13px] text-[#737373]">{copy.settings.tasksLogsEmpty}</div>;
  }
  return (
    <TaskRunLogVirtualList
      logs={logs}
      onOpenLiveViewer={onOpenLiveViewer}
      onStopRun={onStopRun}
      stoppingRunId={stoppingRunId}
    />
  );
}

function TaskRunLogVirtualList(props: {
  logs: TaskRunLog[];
  onOpenLiveViewer: (run: TaskRunLog) => Promise<void>;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stoppingRunId?: string;
}) {
  const { logs, onOpenLiveViewer, onStopRun, stoppingRunId } = props;
  const scrollElementRef = useRef<HTMLDivElement>(null);
  const rowVirtualizer = useVirtualizer({
    count: logs.length,
    estimateSize: () => TASK_LOG_ROW_ESTIMATE_PX,
    getItemKey: (index) => logs[index]?.run_id ?? index,
    getScrollElement: () => scrollElementRef.current,
    initialRect: { height: TASK_LOG_INITIAL_VIEWPORT_PX, width: 0 },
    overscan: TASK_LOG_ROW_OVERSCAN,
    useAnimationFrameWithResizeObserver: true,
  });
  const virtualItems = rowVirtualizer.getVirtualItems();

  return (
    <div ref={scrollElementRef} className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
      <div className="relative w-full" style={{ height: rowVirtualizer.getTotalSize() }}>
        {virtualItems.map((virtualItem) => {
          const log = logs[virtualItem.index];
          if (!log) {
            return null;
          }
          return (
            <div
              key={virtualItem.key}
              data-index={virtualItem.index}
              ref={rowVirtualizer.measureElement}
              className="absolute left-0 top-0 w-full pb-3"
              style={{ transform: `translateY(${virtualItem.start}px)` }}
            >
              <TaskRunLogCard
                log={log}
                onOpenLiveViewer={onOpenLiveViewer}
                onStopRun={onStopRun}
                stopping={stoppingRunId === log.run_id}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}

function TaskRunLogCard(props: {
  log: TaskRunLog;
  onOpenLiveViewer: (run: TaskRunLog) => Promise<void>;
  onStopRun?: (run: TaskRunLog) => Promise<void>;
  stopping: boolean;
}) {
  const { locale } = useWebLocale();
  const { log, onOpenLiveViewer, onStopRun, stopping } = props;

  return (
    <details className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-3">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-3 text-[12px] text-[#111111]">
        <span className="min-w-0">
          <span className="font-semibold">{log.run_id}</span>
          <span className="mx-2 text-[#737373]">{formatTaskRunStatus(log.status, locale)}</span>
          <span className="text-[#737373]">{formatTaskRunTimestamp(log.started_at ?? log.scheduled_at)}</span>
        </span>
        <TaskRunLogActions
          log={log}
          onOpenLiveViewer={onOpenLiveViewer}
          onStopRun={onStopRun}
          stopping={stopping}
        />
      </summary>
      <div className="mt-3 space-y-2 text-[12px] text-[#111111]">
        <PreviewBlock preview={log.response_preview} />
        {log.error ? <p className="text-[#B91C1C]">error: {log.error}</p> : null}
      </div>
    </details>
  );
}

async function resolveViewerRun(
  taskID: string,
  run: TaskRunLog,
  onRefreshLogs?: (taskID: string) => Promise<TaskRunLog[]>,
): Promise<TaskRunLog> {
  if (!onRefreshLogs) {
    return run;
  }

  const refreshed = await onRefreshLogs(taskID);
  const matched = refreshed.find((entry) => entry.run_id === run.run_id);
  if (!matched) {
    throw new Error(`run log ${run.run_id} is no longer available`);
  }
  return matched;
}

function PreviewBlock(props: { preview?: string }) {
  const preview = props.preview?.trim();
  if (!preview) {
    return null;
  }
  return (
    <div className="space-y-1">
      <p className="font-semibold">preview:</p>
      <p className="whitespace-pre-wrap break-words">{withTrailingEllipsis(preview)}</p>
    </div>
  );
}

function withTrailingEllipsis(value: string): string {
  if (value.endsWith('……') || value.endsWith('...') || value.endsWith('…')) {
    return value;
  }
  return `${value}……`;
}
