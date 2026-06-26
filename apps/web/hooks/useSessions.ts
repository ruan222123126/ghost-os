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
import { deleteSession as deleteSessionRequest } from '@/lib/api/sessions/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SessionMetadata } from '@/lib/types';
import { SESSION_LOAD_COMMIT_NEW_ONLY, useLoadSessions, type LoadSessionsOptions } from './useSessionLoader';

interface UseSessionsOptions {
  autoRefresh?: boolean;
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

export function useSessions(options: UseSessionsOptions = {}): UseSessionsResult {
  const { autoRefresh = false } = options;
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

  useSessionAutoRefresh({ autoRefresh, loadSessions });

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
  loadSessions: (options?: LoadSessionsOptions) => Promise<void>;
}) {
  const { autoRefresh, loadSessions } = options;

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

    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  }, [autoRefresh, loadSessions]);
}
