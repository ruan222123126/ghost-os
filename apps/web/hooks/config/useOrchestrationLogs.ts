'use client';

import { listOrchestrationLogs, stopOrchestrationRun } from '@/lib/api/orchestrations/api';
import { useRunLogs, type RunLogsMessages, type RunLogsState } from './useRunLogs';

export function useOrchestrationLogs(messages: RunLogsMessages): RunLogsState {
  return useRunLogs({
    messages,
    listLogs: listOrchestrationLogs,
    stopRun: stopOrchestrationRun,
  });
}
