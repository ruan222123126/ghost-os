'use client';

import { useCallback, useState } from 'react';
import { listTaskLogs } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import type { TaskRunLog } from '@/lib/types';

interface TaskLogsState {
  entries: TaskRunLog[];
  error: string;
  loading: boolean;
  open: (taskID: string) => Promise<void>;
  close: () => void;
  taskID: string;
}

export function useTaskLogs(emptyMessage: string): TaskLogsState {
  const [taskID, setTaskID] = useState('');
  const [entries, setEntries] = useState<TaskRunLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const open = useCallback(async (nextTaskID: string) => {
    setTaskID(nextTaskID);
    setLoading(true);
    setError('');
    setEntries([]);
    try {
      setEntries(await listTaskLogs(nextTaskID, 20));
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
    setLoading(false);
  }, []);

  return { taskID, entries, loading, error, open, close };
}

