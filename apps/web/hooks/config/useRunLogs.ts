'use client';

import { useCallback, useMemo, useState } from 'react';
import { toErrorMessage } from '@/lib/errors';
import type { TaskRunLog } from '@/lib/types';

const RUN_LOG_LIMIT = 20;

export interface RunLogsMessages {
  empty: string;
  stopFailed: string;
}

export interface RunLogsState {
  entries: TaskRunLog[];
  error: string;
  loading: boolean;
  open: (taskID: string) => Promise<void>;
  refresh: (taskID?: string) => Promise<TaskRunLog[]>;
  stopRun: (run: TaskRunLog) => Promise<void>;
  stoppingRunId: string;
  close: () => void;
  taskID: string;
}

interface RunLogsOptions {
  messages: RunLogsMessages;
  listLogs: (taskID: string, limit: number) => Promise<TaskRunLog[]>;
  stopRun: (taskID: string, runID: string) => Promise<unknown>;
}

interface RunLogSetters {
  setTaskID: (value: string) => void;
  setEntries: (value: TaskRunLog[]) => void;
  setLoading: (value: boolean) => void;
  setError: (value: string) => void;
  setStoppingRunId: (value: string) => void;
}

export function useRunLogs(options: RunLogsOptions): RunLogsState {
  const { messages, listLogs, stopRun: stopRunAction } = options;
  const [taskID, setTaskID] = useState('');
  const [entries, setEntries] = useState<TaskRunLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [stoppingRunId, setStoppingRunId] = useState('');
  const setters = useMemo(function buildRunLogSetters() {
    return {
      setTaskID,
      setEntries,
      setLoading,
      setError,
      setStoppingRunId,
    };
  }, []);

  const load = useCallback(function load(nextTaskID: string, resetEntries: boolean) {
    return loadRunLogs({ nextTaskID, resetEntries, listLogs, messages, setters });
  }, [listLogs, messages, setters]);

  const open = useCallback(async function open(nextTaskID: string) {
    await load(nextTaskID, true);
  }, [load]);

  const refresh = useCallback(async function refresh(nextTaskID?: string): Promise<TaskRunLog[]> {
    const resolvedTaskID = (nextTaskID ?? taskID).trim();
    if (!resolvedTaskID) {
      return [];
    }
    return load(resolvedTaskID, false);
  }, [load, taskID]);

  const stopRun = useCallback(function stopRun(run: TaskRunLog) {
    return stopRunLog({ taskID, run, load, messages, setters, stopRunAction });
  }, [load, messages, setters, stopRunAction, taskID]);

  const close = useCallback(function close() {
    resetRunLogs(setters);
  }, [setters]);

  return { taskID, entries, loading, error, open, refresh, stopRun, stoppingRunId, close };
}

async function loadRunLogs(options: {
  nextTaskID: string;
  resetEntries: boolean;
  listLogs: RunLogsOptions['listLogs'];
  messages: RunLogsMessages;
  setters: RunLogSetters;
}): Promise<TaskRunLog[]> {
  const { nextTaskID, resetEntries, listLogs, messages, setters } = options;
  setters.setTaskID(nextTaskID);
  setters.setLoading(true);
  setters.setError('');
  if (resetEntries) {
    setters.setEntries([]);
  }

  try {
    const nextEntries = await listLogs(nextTaskID, RUN_LOG_LIMIT);
    setters.setEntries(nextEntries);
    return nextEntries;
  } catch (nextError) {
    setters.setError(toErrorMessage(nextError, messages.empty));
    throw nextError;
  } finally {
    setters.setLoading(false);
  }
}

async function stopRunLog(options: {
  taskID: string;
  run: TaskRunLog;
  load: (taskID: string, resetEntries: boolean) => Promise<TaskRunLog[]>;
  messages: RunLogsMessages;
  setters: RunLogSetters;
  stopRunAction: RunLogsOptions['stopRun'];
}): Promise<void> {
  const { taskID, run, load, messages, setters, stopRunAction } = options;
  if (!taskID) {
    return;
  }

  setters.setStoppingRunId(run.run_id);
  setters.setError('');
  try {
    await stopRunAction(taskID, run.run_id);
  } catch (nextError) {
    setters.setError(toErrorMessage(nextError, messages.stopFailed));
  } finally {
    await load(taskID, false).catch(() => undefined);
    setters.setStoppingRunId('');
  }
}

function resetRunLogs(setters: RunLogSetters): void {
  setters.setTaskID('');
  setters.setEntries([]);
  setters.setError('');
  setters.setLoading(false);
  setters.setStoppingRunId('');
}
