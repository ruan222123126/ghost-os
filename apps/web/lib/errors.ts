export function toErrorMessage(error: unknown, fallback = 'request failed'): string {
  if (error instanceof Error) {
    const message = error.message.trim();
    if (message) {
      return message;
    }
  }
  return fallback;
}

export function ignorePromise<T>(promise: Promise<T>): void {
  promise.catch(() => undefined);
}
