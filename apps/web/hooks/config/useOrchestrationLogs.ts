'use client';

import { useCallback, useState } from 'react';
import { listOrchestrationLogs } from '@/lib/api/orchestrations/api';
import { toErrorMessage } from '@/lib/errors';
import type { TaskRunLog } from '@/lib/types';

interface OrchestrationLogsState {
  entries: TaskRunLog[];
  error: string;
  loading: boolean;
  open: (taskID: string) => Promise<void>;
  close: () => void;
  taskID: string;
}

export function useOrchestrationLogs(emptyMessage: string): OrchestrationLogsState {
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

