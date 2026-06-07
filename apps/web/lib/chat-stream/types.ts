export interface ActiveAgentRun {
  abortController?: AbortController;
  sessionId: string;
  traceId: string;
}
