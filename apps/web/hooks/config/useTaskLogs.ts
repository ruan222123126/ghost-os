'use client';

import { listTaskLogs, stopTaskRun } from '@/lib/api/tasks/api';
import { useRunLogs, type RunLogsMessages, type RunLogsState } from './useRunLogs';

export function useTaskLogs(messages: RunLogsMessages): RunLogsState {
  return useRunLogs({
    messages,
    listLogs: listTaskLogs,
    stopRun: stopTaskRun,
  });
}
