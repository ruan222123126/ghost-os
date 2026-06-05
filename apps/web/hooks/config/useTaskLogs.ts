'use client';

import { useCallback, useState } from 'react';
import { listTaskLogs, stopTaskRun } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import type { TaskRunLog } from '@/lib/types';

interface TaskLogsMessages {
  empty: string;
  stopFailed: string;
}

interface TaskLogsState {
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

export function useTaskLogs(messages: TaskLogsMessages): TaskLogsState {
  const [taskID, setTaskID] = useState('');
  const [entries, setEntries] = useState<TaskRunLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [stoppingRunId, setStoppingRunId] = useState('');

  const load = useCallback(async (
    nextTaskID: string,
    resetEntries: boolean,
  ): Promise<TaskRunLog[]> => {
    setTaskID(nextTaskID);
    setLoading(true);
    setError('');
    if (resetEntries) {
      setEntries([]);
    }

    try {
      const nextEntries = await listTaskLogs(nextTaskID, 20);
      setEntries(nextEntries);
      return nextEntries;
    } catch (nextError) {
      setError(toErrorMessage(nextError, messages.empty));
      throw nextError;
    } finally {
      setLoading(false);
    }
  }, [messages.empty]);

  const open = useCallback(async (nextTaskID: string) => {
    await load(nextTaskID, true);
  }, [load]);

  const refresh = useCallback(async (nextTaskID?: string): Promise<TaskRunLog[]> => {
    const resolvedTaskID = (nextTaskID ?? taskID).trim();
    if (!resolvedTaskID) {
      return [];
    }
    return load(resolvedTaskID, false);
  }, [load, taskID]);

  const stopRun = useCallback(async (run: TaskRunLog) => {
    if (!taskID) {
      return;
    }
    setStoppingRunId(run.run_id);
    setError('');
    try {
      await stopTaskRun(taskID, run.run_id);
    } catch (nextError) {
      setError(toErrorMessage(nextError, messages.stopFailed));
    } finally {
      await load(taskID, false).catch(() => undefined);
      setStoppingRunId('');
    }
  }, [load, messages.stopFailed, taskID]);

  const close = useCallback(() => {
    setTaskID('');
    setEntries([]);
    setError('');
    setLoading(false);
    setStoppingRunId('');
  }, []);

  return { taskID, entries, loading, error, open, refresh, stopRun, stoppingRunId, close };
}
