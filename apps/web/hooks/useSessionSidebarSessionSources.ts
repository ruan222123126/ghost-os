'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { listOrchestrations, listOrchestrationLogs } from '@/lib/api/orchestrations/api';
import { listTaskLogs, listTasks } from '@/lib/api/tasks/api';
import { isLoopTask } from '@/lib/configLoops';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import {
  collectSessionSourceResolution,
  type SessionSourceAssignments,
} from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata, TaskPayload, TaskRunLog } from '@/lib/types';

const TASK_NAME_PREVIEW_LENGTH = 48;
const defaultRefreshIntervalMs = 3000;

interface UseSessionSidebarSessionSourcesOptions {
  enabled: boolean;
  sessions: SessionMetadata[];
  requestFailedText: string;
  refreshIntervalMs?: number;
}

interface UseSessionSidebarSessionSourcesResult {
  assignments: SessionSourceAssignments;
  hiddenSessionIDs: string[];
  error: string;
}

export function useSessionSidebarSessionSources(
  options: UseSessionSidebarSessionSourcesOptions,
): UseSessionSidebarSessionSourcesResult {
  const {
    enabled,
    sessions,
    requestFailedText,
    refreshIntervalMs = defaultRefreshIntervalMs,
  } = options;
  const [assignments, setAssignments] = useState<SessionSourceAssignments>({});
  const [hiddenSessionIDs, setHiddenSessionIDs] = useState<string[]>([]);
  const [error, setError] = useState('');
  const sessionIDKey = useMemo(() => {
    return sessions.map((session) => session.id).join('\u0000');
  }, [sessions]);
  const knownSessionIDs = useMemo(() => {
    return sessionIDKey.length === 0 ? new Set<string>() : new Set(sessionIDKey.split('\u0000'));
  }, [sessionIDKey]);

  const reload = useCallback(async () => {
    if (!enabled) {
      setAssignments({});
      setHiddenSessionIDs([]);
      setError('');
      return;
    }

    try {
      const sources = await loadAllRunLogSources();
      const resolution = collectSessionSourceResolution(sources.runs, {
        sourceNamesByID: sources.sourceNamesByID,
        loopTaskIDs: sources.loopTaskIDs,
      });
      setAssignments(filterKnownAssignments(resolution.assignments, knownSessionIDs));
      setHiddenSessionIDs(filterKnownSessionIDs(resolution.hiddenSessionIDs, knownSessionIDs));
      setError('');
    } catch (error) {
      setAssignments({});
      setHiddenSessionIDs([]);
      setError(toErrorMessage(error, requestFailedText));
    }
  }, [enabled, knownSessionIDs, requestFailedText]);

  useEffect(() => {
    ignorePromise(reload());
  }, [reload]);

  useEffect(() => {
    if (!enabled || typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    const refresh = () => {
      ignorePromise(reload());
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refresh();
      }
    };

    const intervalID = window.setInterval(refresh, refreshIntervalMs);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => {
      window.clearInterval(intervalID);
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  }, [enabled, refreshIntervalMs, reload]);

  return { assignments, hiddenSessionIDs, error };
}

async function loadAllRunLogSources(): Promise<{
  runs: TaskRunLog[];
  sourceNamesByID: Record<string, string>;
  loopTaskIDs: string[];
}> {
  const [tasks, orchestrations] = await Promise.all([listTasks(), listOrchestrations()]);
  const taskLogs = await Promise.all(tasks.map((task) => listTaskLogs(task.id, 0)));
  const orchestrationLogs = await Promise.all(orchestrations.map((task) => listOrchestrationLogs(task.id, 0)));
  const runs = [
    ...tasks.flatMap((task, index) => taskLogs[index].map((run) => ({ ...run, task_kind: run.task_kind ?? task.task_kind }))),
    ...orchestrations.flatMap((task, index) => orchestrationLogs[index].map((run) => ({ ...run, task_kind: run.task_kind ?? task.task_kind }))),
  ];
  return {
    runs,
    sourceNamesByID: buildSourceNamesByID([...tasks, ...orchestrations]),
    loopTaskIDs: tasks.filter(isLoopTask).map((task) => task.id),
  };
}

function buildSourceNamesByID(tasks: TaskPayload[]): Record<string, string> {
  const names: Record<string, string> = {};
  for (const task of tasks) {
    names[task.id] = sourceNameFromTask(task);
  }
  return names;
}

function sourceNameFromTask(task: TaskPayload): string {
  if (task.task_kind === 'orchestration') {
    return task.name.trim() || task.id;
  }
  if (task.task_kind === 'agent_message') {
    return previewTaskMessage(task.message) || task.id;
  }
  return task.id;
}

function previewTaskMessage(message: string): string {
  const firstLine = message.trim().split('\n')[0]?.trim() ?? '';
  if (firstLine.length <= TASK_NAME_PREVIEW_LENGTH) {
    return firstLine;
  }
  return `${firstLine.slice(0, TASK_NAME_PREVIEW_LENGTH)}...`;
}

function filterKnownAssignments(
  assignments: SessionSourceAssignments,
  knownSessionIDs: ReadonlySet<string>,
): SessionSourceAssignments {
  const filtered: SessionSourceAssignments = {};
  for (const [sessionID, source] of Object.entries(assignments)) {
    if (knownSessionIDs.has(sessionID)) {
      filtered[sessionID] = source;
    }
  }
  return filtered;
}

function filterKnownSessionIDs(
  candidateIDs: string[],
  knownSessionIDs: ReadonlySet<string>,
): string[] {
  return candidateIDs.filter((sessionID) => knownSessionIDs.has(sessionID));
}
