const FALLBACK_FRAME_DELAY_MS = 16;

export interface StreamReplyCommitter {
  cancel: () => void;
  flush: () => void;
  enqueue: (commit: () => void) => void;
}

export function createStreamReplyCommitter(): StreamReplyCommitter {
  let cancelled = false;
  let pendingCommit: (() => void) | null = null;
  let scheduledHandle: number | null = null;
  let scheduledKind: "raf" | "timeout" | null = null;

  function schedule(): void {
    if (cancelled || scheduledHandle !== null || pendingCommit === null) {
      return;
    }

    if (typeof window === "undefined") {
      const commit = pendingCommit;
      pendingCommit = null;
      commit?.();
      return;
    }

    if (typeof window.requestAnimationFrame === "function") {
      scheduledKind = "raf";
      scheduledHandle = window.requestAnimationFrame(runScheduledCommit);
      return;
    }

    scheduledKind = "timeout";
    scheduledHandle = window.setTimeout(runScheduledCommit, FALLBACK_FRAME_DELAY_MS);
  }

  function runScheduledCommit(): void {
    scheduledHandle = null;
    scheduledKind = null;
    if (cancelled) {
      pendingCommit = null;
      return;
    }

    const commit = pendingCommit;
    pendingCommit = null;
    commit?.();
  }

  function cancel(): void {
    cancelled = true;
    pendingCommit = null;
    if (scheduledHandle === null || typeof window === "undefined") {
      return;
    }

    if (scheduledKind === "raf" && typeof window.cancelAnimationFrame === "function") {
      window.cancelAnimationFrame(scheduledHandle);
    } else {
      window.clearTimeout(scheduledHandle);
    }
    scheduledHandle = null;
    scheduledKind = null;
  }

  function flush(): void {
    if (cancelled) {
      return;
    }

    const commit = pendingCommit;
    pendingCommit = null;
    if (scheduledHandle !== null && typeof window !== "undefined") {
      if (scheduledKind === "raf" && typeof window.cancelAnimationFrame === "function") {
        window.cancelAnimationFrame(scheduledHandle);
      } else {
        window.clearTimeout(scheduledHandle);
      }
      scheduledHandle = null;
      scheduledKind = null;
    }

    commit?.();
  }

  function enqueue(commit: () => void): void {
    if (cancelled) {
      return;
    }

    pendingCommit = commit;
    if (scheduledHandle === null) {
      schedule();
    }
  }

  return {
    cancel,
    enqueue,
    flush,
  };
}

export function enqueueStreamReplyCommit(
  committers: Map<string, StreamReplyCommitter>,
  requestId: string,
  commit: () => void,
): void {
  const key = requestId.trim();
  if (!key) {
    commit();
    return;
  }

  const committer = committers.get(key) ?? createStreamReplyCommitter();
  committers.set(key, committer);
  committer.enqueue(commit);
}

export function cancelStreamReplyCommit(
  committers: Map<string, StreamReplyCommitter>,
  requestId: string,
): void {
  const key = requestId.trim();
  if (!key) {
    return;
  }

  committers.get(key)?.cancel();
  committers.delete(key);
}

export function flushStreamReplyCommit(
  committers: Map<string, StreamReplyCommitter>,
  requestId: string,
): void {
  const key = requestId.trim();
  if (!key) {
    return;
  }

  committers.get(key)?.flush();
  committers.delete(key);
}

export function cancelAllStreamReplyCommitters(committers: Map<string, StreamReplyCommitter>): void {
  for (const committer of committers.values()) {
    committer.cancel();
  }
  committers.clear();
}
