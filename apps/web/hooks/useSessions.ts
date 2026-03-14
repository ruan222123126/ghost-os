// React hook that loads and mutates session collections via bridge APIs.

'use client';

import { useCallback, useEffect, useState } from 'react';
import { deleteSession as deleteSessionRequest, listSessions } from '@/lib/api/sessions/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { SessionMetadata } from '@/lib/types';

interface UseSessionsResult {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  loadSessions: () => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  createNewSession: () => void;
  setCurrentSessionId: (id: string) => void;
}

export function useSessions(): UseSessionsResult {
  const [sessions, setSessions] = useState<SessionMetadata[]>([]);
  const [currentSessionId, setCurrentSessionIdState] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const loaded = await listSessions();
      setSessions(loaded);
    } catch (error) {
      setError(toErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  const setCurrentSessionId = useCallback((id: string) => {
    setCurrentSessionIdState(id.trim());
  }, []);

  const createNewSession = useCallback(() => {
    setCurrentSessionIdState('');
  }, []);

  const deleteSession = useCallback(async (id: string) => {
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
      setError(toErrorMessage(error));
    }
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
