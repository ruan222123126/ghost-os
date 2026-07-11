import { useCallback, useRef, type Dispatch, type MutableRefObject, type SetStateAction } from 'react';
import { listSessions } from '@/lib/api/sessions/api';
import { toErrorMessage } from '@/lib/errors';
import type { SessionMetadata } from '@/lib/types';

export const SESSION_LOAD_COMMIT_ALWAYS = 'always';
export const SESSION_LOAD_COMMIT_NEW_ONLY = 'when-new-session';

export interface LoadSessionsOptions {
  silent?: boolean;
  commitMode?: SessionLoadCommitMode;
}

type SessionLoadCommitMode = typeof SESSION_LOAD_COMMIT_ALWAYS | typeof SESSION_LOAD_COMMIT_NEW_ONLY;

interface ResolvedLoadSessionsOptions {
  silent: boolean;
  commitMode: SessionLoadCommitMode;
}

interface UseLoadSessionsOptions {
  mountedRef: MutableRefObject<boolean>;
  sessionsRef: MutableRefObject<SessionMetadata[]>;
  errorRef: MutableRefObject<string>;
  fallbackMessage: string;
  setLoading: (loading: boolean) => void;
  setError: Dispatch<SetStateAction<string>>;
  setSessions: Dispatch<SetStateAction<SessionMetadata[]>>;
}

interface QueuedSessionLoaderOptions {
  activeLoadRef: MutableRefObject<Promise<void> | null>;
  queuedLoadOptionsRef: MutableRefObject<ResolvedLoadSessionsOptions | null>;
  runLoad: (loadOptions: ResolvedLoadSessionsOptions) => Promise<void>;
}

interface SessionListRequestOptions extends UseLoadSessionsOptions {
  loadOptions: ResolvedLoadSessionsOptions;
}

export function useLoadSessions(options: UseLoadSessionsOptions) {
  const activeLoadRef = useRef<Promise<void> | null>(null);
  const queuedLoadOptionsRef = useRef<ResolvedLoadSessionsOptions | null>(null);
  const runLoad = useSessionListRequest(options);

  return useQueuedSessionLoader({
    activeLoadRef,
    queuedLoadOptionsRef,
    runLoad,
  });
}

function useSessionListRequest(options: UseLoadSessionsOptions) {
  const {
    errorRef,
    fallbackMessage,
    mountedRef,
    sessionsRef,
    setError,
    setLoading,
    setSessions,
  } = options;

  return useCallback(async (loadOptions: ResolvedLoadSessionsOptions) => {
    await loadSessionList({
      errorRef,
      fallbackMessage,
      loadOptions,
      mountedRef,
      sessionsRef,
      setError,
      setLoading,
      setSessions,
    });
  }, [errorRef, fallbackMessage, mountedRef, sessionsRef, setError, setLoading, setSessions]);
}

function useQueuedSessionLoader(options: QueuedSessionLoaderOptions) {
  const { activeLoadRef, queuedLoadOptionsRef, runLoad } = options;

  return useCallback((loadOptions: LoadSessionsOptions = {}) => {
    queueSessionLoad(queuedLoadOptionsRef, resolveLoadSessionsOptions(loadOptions));
    return activeLoadRef.current ?? startQueuedSessionLoads({
      activeLoadRef,
      queuedLoadOptionsRef,
      runLoad,
    });
  }, [activeLoadRef, queuedLoadOptionsRef, runLoad]);
}

async function loadSessionList(options: SessionListRequestOptions): Promise<void> {
  const { loadOptions } = options;
  beginSessionListRequest(loadOptions, options.setLoading, options.setError);

  try {
    const loaded = await listSessions();
    commitLoadedSessions(options, loaded);
  } catch (error) {
    handleSessionLoadError(options, error);
  } finally {
    finishSessionListRequest(loadOptions, options.mountedRef, options.setLoading);
  }
}

function beginSessionListRequest(
  loadOptions: ResolvedLoadSessionsOptions,
  setLoading: (loading: boolean) => void,
  setError: Dispatch<SetStateAction<string>>,
): void {
  if (loadOptions.silent) {
    return;
  }
  setLoading(true);
  setError('');
}

function commitLoadedSessions(options: SessionListRequestOptions, loaded: SessionMetadata[]): void {
  const { errorRef, loadOptions, mountedRef, sessionsRef, setError, setSessions } = options;
  if (!mountedRef.current) {
    return;
  }
  if (shouldCommitSessionList(sessionsRef.current, loaded, loadOptions.commitMode)) {
    setSessions(loaded);
  }
  if (errorRef.current !== '') {
    setError('');
  }
}

function handleSessionLoadError(options: SessionListRequestOptions, error: unknown): void {
  const { fallbackMessage, loadOptions, mountedRef, setError } = options;
  if (!mountedRef.current) {
    return;
  }
  if (loadOptions.silent) {
    console.error('[useSessions] silent refresh failed', error);
    return;
  }
  setError(toErrorMessage(error, fallbackMessage));
}

function finishSessionListRequest(
  loadOptions: ResolvedLoadSessionsOptions,
  mountedRef: MutableRefObject<boolean>,
  setLoading: (loading: boolean) => void,
): void {
  if (mountedRef.current && !loadOptions.silent) {
    setLoading(false);
  }
}

function queueSessionLoad(
  queuedLoadOptionsRef: MutableRefObject<ResolvedLoadSessionsOptions | null>,
  next: ResolvedLoadSessionsOptions,
): void {
  queuedLoadOptionsRef.current = mergeQueuedLoadOptions(queuedLoadOptionsRef.current, next);
}

function startQueuedSessionLoads(options: QueuedSessionLoaderOptions): Promise<void> {
  const activeLoad = executeQueuedSessionLoads(options).finally(() => {
    clearActiveSessionLoad(options.activeLoadRef, activeLoad);
  });
  options.activeLoadRef.current = activeLoad;
  return activeLoad;
}

async function executeQueuedSessionLoads(options: QueuedSessionLoaderOptions): Promise<void> {
  let nextLoadOptions = takeQueuedSessionLoad(options.queuedLoadOptionsRef);
  while (nextLoadOptions) {
    await options.runLoad(nextLoadOptions);
    nextLoadOptions = takeQueuedSessionLoad(options.queuedLoadOptionsRef);
  }
}

function takeQueuedSessionLoad(
  queuedLoadOptionsRef: MutableRefObject<ResolvedLoadSessionsOptions | null>,
): ResolvedLoadSessionsOptions | null {
  const nextLoadOptions = queuedLoadOptionsRef.current;
  queuedLoadOptionsRef.current = null;
  return nextLoadOptions;
}

function clearActiveSessionLoad(
  activeLoadRef: MutableRefObject<Promise<void> | null>,
  activeLoad: Promise<void>,
): void {
  if (activeLoadRef.current === activeLoad) {
    activeLoadRef.current = null;
  }
}

function sessionListsEqual(left: SessionMetadata[], right: SessionMetadata[]): boolean {
  if (left.length !== right.length) {
    return false;
  }

  for (let index = 0; index < left.length; index += 1) {
    if (!sessionMetadataEqual(left[index], right[index])) {
      return false;
    }
  }
  return true;
}

function shouldCommitSessionList(
  current: SessionMetadata[],
  next: SessionMetadata[],
  commitMode: SessionLoadCommitMode,
): boolean {
  if (commitMode === SESSION_LOAD_COMMIT_NEW_ONLY) {
    return hasNewSessionID(current, next);
  }
  return !sessionListsEqual(current, next);
}

function sessionMetadataEqual(left: SessionMetadata, right: SessionMetadata): boolean {
  return left.id === right.id
    && left.title === right.title
    && left.created_at === right.created_at
    && left.updated_at === right.updated_at
    && left.message_count === right.message_count
    && left.token_count === right.token_count;
}

function hasNewSessionID(current: SessionMetadata[], next: SessionMetadata[]): boolean {
  if (next.length === 0) {
    return false;
  }

  const currentIDs = new Set(current.map((session) => session.id));
  for (const session of next) {
    if (!currentIDs.has(session.id)) {
      return true;
    }
  }
  return false;
}

function resolveLoadSessionsOptions(loadOptions: LoadSessionsOptions): ResolvedLoadSessionsOptions {
  return {
    silent: loadOptions.silent ?? false,
    commitMode: loadOptions.commitMode ?? SESSION_LOAD_COMMIT_ALWAYS,
  };
}

function mergeQueuedLoadOptions(
  current: ResolvedLoadSessionsOptions | null,
  next: ResolvedLoadSessionsOptions,
): ResolvedLoadSessionsOptions {
  if (!current) {
    return next;
  }
  return {
    silent: current.silent && next.silent,
    commitMode: current.commitMode === SESSION_LOAD_COMMIT_ALWAYS || next.commitMode === SESSION_LOAD_COMMIT_ALWAYS
      ? SESSION_LOAD_COMMIT_ALWAYS
      : SESSION_LOAD_COMMIT_NEW_ONLY,
  };
}
