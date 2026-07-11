import type { AgentRuntimeType } from '@/lib/types';

export interface ActiveAgentRun {
  abortController?: AbortController;
  runtime?: AgentRuntimeType;
  sessionId: string;
  traceId: string;
}
