// React hook that loads and mutates session collections via bridge APIs.

'use client';

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from 'react';
import { deleteSession as deleteSessionRequest, listSessions } from '@/lib/api/sessions/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SessionMetadata } from '@/lib/types';

const defaultRefreshIntervalMs = 3000;
const SESSION_LOAD_COMMIT_ALWAYS = 'always';
const SESSION_LOAD_COMMIT_NEW_ONLY = 'when-new-session';

interface LoadSessionsOptions {
  silent?: boolean;
  commitMode?: SessionLoadCommitMode;
}

interface UseSessionsOptions {
  autoRefresh?: boolean;
  refreshIntervalMs?: number;
}

interface UseSessionsResult {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  loadSessions: (options?: LoadSessionsOptions) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  createNewSession: () => void;
  setCurrentSessionId: (id: string) => void;
}

type SessionLoadCommitMode = typeof SESSION_LOAD_COMMIT_ALWAYS | typeof SESSION_LOAD_COMMIT_NEW_ONLY;

interface ResolvedLoadSessionsOptions {
  silent: boolean;
  commitMode: SessionLoadCommitMode;
}

export function useSessions(options: UseSessionsOptions = {}): UseSessionsResult {
  const { autoRefresh = false, refreshIntervalMs = defaultRefreshIntervalMs } = options;
  const { copy } = useWebLocale();
  const mountedRef = useMountedRef();
  const [sessions, setSessions] = useState<SessionMetadata[]>([]);
  const [currentSessionId, setCurrentSessionIdState] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const sessionsRef = useLatestRef(sessions);
  const errorRef = useLatestRef(error);
  const loadSessions = useLoadSessions({
    mountedRef,
    sessionsRef,
    errorRef,
    fallbackMessage: copy.system.genericRequestFailed,
    setLoading,
    setError,
    setSessions,
  });
  const deleteSession = useDeleteSession({
    fallbackMessage: copy.system.genericRequestFailed,
    setSessions,
    setCurrentSessionIdState,
    setError,
  });

  useSessionAutoRefresh({ autoRefresh, refreshIntervalMs, loadSessions });

  const setCurrentSessionId = useCallback((id: string) => {
    setCurrentSessionIdState(id.trim());
  }, []);

  const createNewSession = useCallback(() => {
    setCurrentSessionIdState('');
  }, []);

  useEffect(() => {
    ignorePromise(loadSessions());
  }, [loadSessions]);

  return {
    sessions,
    currentSessionId,
    loading,
    error,
    loadSessions,
    deleteSession,
    createNewSession,
    setCurrentSessionId,
  };
}

function useMountedRef(): MutableRefObject<boolean> {
  // Strict Mode runs effects twice (mount → cleanup → mount). Initialise to
  // `false` so the mount handler must set it back to `true` on every mount —
  // that way the assignment is not "dead code" and won't be auto-removed.
  const mountedRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  return mountedRef;
}

function useLatestRef<T>(value: T): MutableRefObject<T> {
  const ref = useRef(value);
  ref.current = value;
  return ref;
}

function useLoadSessions(options: {
  mountedRef: MutableRefObject<boolean>;
  sessionsRef: MutableRefObject<SessionMetadata[]>;
  errorRef: MutableRefObject<string>;
  fallbackMessage: string;
  setLoading: (loading: boolean) => void;
  setError: Dispatch<SetStateAction<string>>;
  setSessions: Dispatch<SetStateAction<SessionMetadata[]>>;
}) {
  const {
    mountedRef,
    sessionsRef,
    errorRef,
    fallbackMessage,
    setLoading,
    setError,
    setSessions,
  } = options;
  const activeLoadRef = useRef<Promise<void> | null>(null);
  const queuedLoadOptionsRef = useRef<ResolvedLoadSessionsOptions | null>(null);

  const runLoad = useCallback(async (loadOptions: ResolvedLoadSessionsOptions) => {
    const { silent, commitMode } = loadOptions;
    if (!silent) {
      setLoading(true);
      setError('');
    }

    try {
      const loaded = await listSessions();
      if (!mountedRef.current) {
        return;
      }
      if (shouldCommitSessionList(sessionsRef.current, loaded, commitMode)) {
        setSessions(loaded);
      }
      if (errorRef.current !== '') {
        setError('');
      }
    } catch (error) {
      if (!mountedRef.current) {
        return;
      }
      if (silent) {
        console.error('[useSessions] silent refresh failed', error);
        return;
      }
      setError(toErrorMessage(error, fallbackMessage));
    } finally {
      if (mountedRef.current && !silent) {
        setLoading(false);
      }
    }
  }, [errorRef, fallbackMessage, mountedRef, sessionsRef, setError, setLoading, setSessions]);

  return useCallback((loadOptions: LoadSessionsOptions = {}) => {
    queuedLoadOptionsRef.current = mergeQueuedLoadOptions(
      queuedLoadOptionsRef.current,
      resolveLoadSessionsOptions(loadOptions),
    );

    if (activeLoadRef.current) {
      return activeLoadRef.current;
    }

    const executeQueuedLoads = async () => {
      while (queuedLoadOptionsRef.current !== null) {
        const nextLoadOptions = queuedLoadOptionsRef.current;
        queuedLoadOptionsRef.current = null;
        await runLoad(nextLoadOptions);
      }
    };

    let activeLoad: Promise<void>;
    activeLoad = executeQueuedLoads().finally(() => {
      if (activeLoadRef.current === activeLoad) {
        activeLoadRef.current = null;
      }
    });
    activeLoadRef.current = activeLoad;
    return activeLoad;
  }, [runLoad]);
}

function useDeleteSession(options: {
  fallbackMessage: string;
  setSessions: Dispatch<SetStateAction<SessionMetadata[]>>;
  setCurrentSessionIdState: Dispatch<SetStateAction<string>>;
  setError: Dispatch<SetStateAction<string>>;
}) {
  const { fallbackMessage, setSessions, setCurrentSessionIdState, setError } = options;

  return useCallback(async (id: string) => {
    const trimmedID = id.trim();
    if (!trimmedID) {
      return;
    }

    setError('');
    try {
      await deleteSessionRequest(trimmedID);
      setSessions((previous) => previous.filter((session) => session.id !== trimmedID));
      setCurrentSessionIdState((previous) => (previous === trimmedID ? '' : previous));
    } catch (error) {
      setError(toErrorMessage(error, fallbackMessage));
    }
  }, [fallbackMessage, setCurrentSessionIdState, setError, setSessions]);
}

function useSessionAutoRefresh(options: {
  autoRefresh: boolean;
  refreshIntervalMs: number;
  loadSessions: (options?: LoadSessionsOptions) => Promise<void>;
}) {
  const { autoRefresh, refreshIntervalMs, loadSessions } = options;

  useEffect(() => {
    if (!autoRefresh || typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    const refresh = () => {
      ignorePromise(loadSessions({
        silent: true,
        commitMode: SESSION_LOAD_COMMIT_NEW_ONLY,
      }));
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refresh();
      }
    };

    const intervalID = window.setInterval(refresh, refreshIntervalMs);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => {
      window.clearInterval(intervalID);
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  }, [autoRefresh, loadSessions, refreshIntervalMs]);
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
