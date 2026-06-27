// Normalized error helpers used across web hooks, API calls, and UI state.

export function toErrorMessage(error: unknown, fallback = 'request failed'): string {
  if (error instanceof Error) {
    const message = error.message.trim();
    if (message) {
      return message;
    }
  }
  return fallback;
}

export function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError';
}

export function isAgentRunCancellationMessage(message: string): boolean {
  const normalized = message.trim().toLowerCase();
  if (!normalized) {
    return false;
  }
  return normalized === 'agent run cancelled'
    || normalized === 'agent stream closed before terminal event'
    || normalized.endsWith(': agent run cancelled')
    || normalized === 'context canceled'
    || normalized.endsWith(': context canceled');
}

export function ignorePromise<T>(promise: Promise<T>): void {
  promise.catch(() => undefined);
}
