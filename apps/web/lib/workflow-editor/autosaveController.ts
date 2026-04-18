export type AutosavePhase = 'idle' | 'saving' | 'saved' | 'error' | 'blocked';

export interface AutosaveState {
  phase: AutosavePhase;
  updatedAt: number;
  message?: string;
}

export interface AutosaveSnapshot<TPayload> {
  fingerprint: string;
  payload: TPayload;
}

interface AutosaveControllerOptions<TPayload> {
  debounceMs: number;
  retryMs: number;
  persist: (snapshot: AutosaveSnapshot<TPayload>) => Promise<void>;
  onStateChange: (state: AutosaveState) => void;
}

export interface AutosaveController<TPayload> {
  schedule: (snapshot: AutosaveSnapshot<TPayload>) => void;
  flush: (snapshot: AutosaveSnapshot<TPayload>, options?: AutosaveFlushOptions) => Promise<void>;
  markBlocked: (message: string) => void;
  markSaved: (fingerprint: string) => void;
  dispose: () => void;
}

export interface AutosaveFlushOptions {
  force?: boolean;
}

export function createAutosaveController<TPayload>(
  options: AutosaveControllerOptions<TPayload>,
): AutosaveController<TPayload> {
  let disposed = false;
  let running = false;
  let draining: Promise<void> | undefined;
  let state: AutosaveState | undefined;
  let pending: AutosaveSnapshot<TPayload> | undefined;
  let lastSavedFingerprint = '';
  let forcedFingerprint = '';
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;
  let retryTimer: ReturnType<typeof setTimeout> | undefined;

  const setState = (phase: AutosavePhase, message?: string) => {
    if (disposed) {
      return;
    }
    if (state && state.phase === phase && state.message === message) {
      return;
    }
    state = {
      phase,
      message,
      updatedAt: Date.now(),
    };
    options.onStateChange(state);
  };

  const clearDebounceTimer = () => {
    if (!debounceTimer) {
      return;
    }
    clearTimeout(debounceTimer);
    debounceTimer = undefined;
  };

  const clearRetryTimer = () => {
    if (!retryTimer) {
      return;
    }
    clearTimeout(retryTimer);
    retryTimer = undefined;
  };

  const scheduleRetry = () => {
    clearRetryTimer();
    retryTimer = setTimeout(() => {
      retryTimer = undefined;
      void drainQueue(false);
    }, options.retryMs);
  };

  const enqueueSnapshot = (snapshot: AutosaveSnapshot<TPayload>) => {
    pending = snapshot;
  };

  const shouldSkipSnapshot = (snapshot: AutosaveSnapshot<TPayload>) => {
    if (snapshot.fingerprint === forcedFingerprint) {
      return false;
    }
    return snapshot.fingerprint === lastSavedFingerprint;
  };

  const runDrainLoop = async (rethrowOnError: boolean): Promise<void> => {
    while (!disposed && pending) {
      const snapshot = pending;
      pending = undefined;
      if (shouldSkipSnapshot(snapshot)) {
        continue;
      }

      setState('saving', 'Saving...');
      try {
        await options.persist(snapshot);
      } catch (error) {
        enqueueSnapshot(snapshot);
        setState('error', messageFromError(error));
        scheduleRetry();
        if (rethrowOnError) {
          throw error;
        }
        return;
      }

      lastSavedFingerprint = snapshot.fingerprint;
      if (forcedFingerprint === snapshot.fingerprint) {
        forcedFingerprint = '';
      }
      setState('saved', 'Saved just now');
    }
  };

  const drainQueue = (rethrowOnError: boolean): Promise<void> => {
    if (disposed) {
      return Promise.resolve();
    }
    if (draining) {
      if (!rethrowOnError) {
        return draining;
      }
      return draining.then(() => drainQueue(true));
    }
    running = true;
    draining = runDrainLoop(rethrowOnError).finally(() => {
      running = false;
      draining = undefined;
    });
    return draining;
  };

  const schedule = (snapshot: AutosaveSnapshot<TPayload>) => {
    if (disposed) {
      return;
    }
    clearRetryTimer();
    enqueueSnapshot(snapshot);
    if (shouldSkipSnapshot(snapshot) && !running) {
      return;
    }
    if (running) {
      return;
    }
    clearDebounceTimer();
    debounceTimer = setTimeout(() => {
      debounceTimer = undefined;
      void drainQueue(false);
    }, options.debounceMs);
  };

  const flush = async (snapshot: AutosaveSnapshot<TPayload>, flushOptions?: AutosaveFlushOptions) => {
    if (disposed) {
      return;
    }
    clearDebounceTimer();
    clearRetryTimer();
    if (flushOptions?.force) {
      forcedFingerprint = snapshot.fingerprint;
    }
    enqueueSnapshot(snapshot);
    if (shouldSkipSnapshot(snapshot) && !running) {
      return;
    }
    await drainQueue(true);
  };

  const markBlocked = (message: string) => {
    if (disposed) {
      return;
    }
    setState('blocked', message);
  };

  const markSaved = (fingerprint: string) => {
    lastSavedFingerprint = fingerprint;
    if (forcedFingerprint === fingerprint) {
      forcedFingerprint = '';
    }
    if (pending?.fingerprint === fingerprint) {
      pending = undefined;
    }
  };

  const dispose = () => {
    disposed = true;
    pending = undefined;
    forcedFingerprint = '';
    clearDebounceTimer();
    clearRetryTimer();
  };

  return {
    schedule,
    flush,
    markBlocked,
    markSaved,
    dispose,
  };
}

function messageFromError(error: unknown): string {
  if (error instanceof Error) {
    const message = error.message.trim();
    if (message) {
      return message;
    }
  }
  return 'failed to save';
}
