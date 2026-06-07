'use client';

import { useEffect, useState } from 'react';
import { searchSessions } from '@/lib/api/sessions/api';
import { toErrorMessage } from '@/lib/errors';
import type { SessionMetadata } from '@/lib/types';

const SESSION_SEARCH_DEBOUNCE_MS = 150;

interface UseSessionSearchOptions {
  open: boolean;
  query: string;
  fallbackMessage: string;
}

interface UseSessionSearchResult {
  results: SessionMetadata[];
  loading: boolean;
  error: string;
}

export function useSessionSearch(options: UseSessionSearchOptions): UseSessionSearchResult {
  const { open, query, fallbackMessage } = options;
  const [results, setResults] = useState<SessionMetadata[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) {
      setResults([]);
      setLoading(false);
      setError('');
      return;
    }

    let cancelled = false;
    setResults([]);
    setLoading(true);
    setError('');

    const timerID = window.setTimeout(() => {
      searchSessions(query)
        .then((nextResults) => {
          if (cancelled) {
            return;
          }
          setResults(nextResults);
        })
        .catch((requestError: unknown) => {
          if (cancelled) {
            return;
          }
          setResults([]);
          setError(toErrorMessage(requestError, fallbackMessage));
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false);
          }
        });
    }, SESSION_SEARCH_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      window.clearTimeout(timerID);
    };
  }, [fallbackMessage, open, query]);

  return { results, loading, error };
}
