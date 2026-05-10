'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { listOrchestrations, listOrchestrationLogs } from '@/lib/api/orchestrations/api';
import { listTaskLogs, listTasks } from '@/lib/api/tasks/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import {
  collectSessionSourceAssignments,
  type SessionSourceAssignments,
} from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata, TaskPayload, TaskRunLog } from '@/lib/types';

const TASK_NAME_PREVIEW_LENGTH = 48;

interface UseSessionSidebarSessionSourcesOptions {
  enabled: boolean;
  sessions: SessionMetadata[];
  requestFailedText: string;
}

interface UseSessionSidebarSessionSourcesResult {
  assignments: SessionSourceAssignments;
  error: string;
}

export function useSessionSidebarSessionSources(
  options: UseSessionSidebarSessionSourcesOptions,
): UseSessionSidebarSessionSourcesResult {
  const { enabled, sessions, requestFailedText } = options;
  const [assignments, setAssignments] = useState<SessionSourceAssignments>({});
  const [error, setError] = useState('');
  const sessionIDs = useMemo(() => sessions.map((session) => session.id), [sessions]);

  const reload = useCallback(async () => {
    if (!enabled || sessionIDs.length === 0) {
      setAssignments({});
      setError('');
      return;
    }

    try {
      const sources = await loadAllRunLogSources();
      const assignments = collectSessionSourceAssignments(sources.runs, {
        sourceNamesByID: sources.sourceNamesByID,
      });
      setAssignments(filterKnownAssignments(assignments, sessionIDs));
      setError('');
    } catch (error) {
      setAssignments({});
      setError(toErrorMessage(error, requestFailedText));
    }
  }, [enabled, requestFailedText, sessionIDs]);

  useEffect(() => {
    ignorePromise(reload());
  }, [reload]);

  return { assignments, error };
}

async function loadAllRunLogSources(): Promise<{
  runs: TaskRunLog[];
  sourceNamesByID: Record<string, string>;
}> {
  const [tasks, orchestrations] = await Promise.all([listTasks(), listOrchestrations()]);
  const taskLogs = await Promise.all(tasks.map((task) => listTaskLogs(task.id, 0)));
  const orchestrationLogs = await Promise.all(orchestrations.map((task) => listOrchestrationLogs(task.id, 0)));
  const runs = [
    ...tasks.flatMap((task, index) => taskLogs[index].map((run) => ({ ...run, task_kind: run.task_kind ?? task.task_kind }))),
    ...orchestrations.flatMap((task, index) => orchestrationLogs[index].map((run) => ({ ...run, task_kind: run.task_kind ?? task.task_kind }))),
  ];
  return { runs, sourceNamesByID: buildSourceNamesByID([...tasks, ...orchestrations]) };
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
  sessionIDs: string[],
): SessionSourceAssignments {
  const known = new Set(sessionIDs);
  const filtered: SessionSourceAssignments = {};
  for (const [sessionID, source] of Object.entries(assignments)) {
    if (known.has(sessionID)) {
      filtered[sessionID] = source;
    }
  }
  return filtered;
}
