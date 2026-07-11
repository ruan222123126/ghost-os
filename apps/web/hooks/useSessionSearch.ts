'use client';

import { useEffect, useState, type Dispatch, type SetStateAction } from 'react';
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

interface SessionSearchEffectOptions extends UseSessionSearchOptions {
  setError: Dispatch<SetStateAction<string>>;
  setLoading: Dispatch<SetStateAction<boolean>>;
  setResults: Dispatch<SetStateAction<SessionMetadata[]>>;
}

interface SessionSearchRequestOptions extends SessionSearchEffectOptions {
  isCancelled: () => boolean;
}

export function useSessionSearch(options: UseSessionSearchOptions): UseSessionSearchResult {
  const { open, query, fallbackMessage } = options;
  const [results, setResults] = useState<SessionMetadata[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useSessionSearchEffect({
    fallbackMessage,
    open,
    query,
    setError,
    setLoading,
    setResults,
  });

  return { results, loading, error };
}

function useSessionSearchEffect(options: SessionSearchEffectOptions): void {
  const { fallbackMessage, open, query, setError, setLoading, setResults } = options;

  useEffect(() => {
    if (!open) {
      resetSessionSearchState(setResults, setLoading, setError);
      return;
    }

    beginSessionSearch(setResults, setLoading, setError);
    return scheduleSessionSearch({
      fallbackMessage,
      open,
      query,
      setError,
      setLoading,
      setResults,
    });
  }, [fallbackMessage, open, query, setError, setLoading, setResults]);
}

function scheduleSessionSearch(options: SessionSearchEffectOptions): () => void {
  let cancelled = false;
  const timerID = window.setTimeout(() => {
    void executeSessionSearch({
      ...options,
      isCancelled: () => cancelled,
    });
  }, SESSION_SEARCH_DEBOUNCE_MS);

  return () => {
    cancelled = true;
    window.clearTimeout(timerID);
  };
}

async function executeSessionSearch(options: SessionSearchRequestOptions): Promise<void> {
  try {
    const nextResults = await searchSessions(options.query);
    commitSessionSearchResults(options, nextResults);
  } catch (requestError) {
    commitSessionSearchError(options, requestError);
  } finally {
    finishSessionSearch(options);
  }
}

function resetSessionSearchState(
  setResults: Dispatch<SetStateAction<SessionMetadata[]>>,
  setLoading: Dispatch<SetStateAction<boolean>>,
  setError: Dispatch<SetStateAction<string>>,
): void {
  setResults([]);
  setLoading(false);
  setError('');
}

function beginSessionSearch(
  setResults: Dispatch<SetStateAction<SessionMetadata[]>>,
  setLoading: Dispatch<SetStateAction<boolean>>,
  setError: Dispatch<SetStateAction<string>>,
): void {
  setResults([]);
  setLoading(true);
  setError('');
}

function commitSessionSearchResults(options: SessionSearchRequestOptions, nextResults: SessionMetadata[]): void {
  if (options.isCancelled()) {
    return;
  }
  options.setResults(nextResults);
}

function commitSessionSearchError(options: SessionSearchRequestOptions, requestError: unknown): void {
  if (options.isCancelled()) {
    return;
  }
  options.setResults([]);
  options.setError(toErrorMessage(requestError, options.fallbackMessage));
}

function finishSessionSearch(options: SessionSearchRequestOptions): void {
  if (!options.isCancelled()) {
    options.setLoading(false);
  }
}
