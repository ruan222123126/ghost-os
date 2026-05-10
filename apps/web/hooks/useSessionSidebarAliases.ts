'use client';

import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useState } from 'react';
import {
  SESSION_ALIAS_STORAGE_KEY,
  createInitialSessionAliasStore,
  parseSessionAliasStore,
  sanitizeSessionAliasStore,
  setSessionAlias,
  stringifySessionAliasStore,
  type SessionAliasStoreV1,
} from '@/lib/sessionSidebarAliases';
import type { SessionMetadata } from '@/lib/types';

export type RenameSessionError = 'empty' | 'unknown';

interface UseSessionSidebarAliasesOptions {
  sessions: SessionMetadata[];
  resolveDefaultTitle: (session: SessionMetadata) => string;
}

export interface RenameSessionResult {
  ok: boolean;
  error?: RenameSessionError;
}

export interface UseSessionSidebarAliasesResult {
  resolveSessionTitle: (session: SessionMetadata) => string;
  resolveSessionTitleByID: (sessionID: string) => string;
  renameSession: (sessionID: string, title: string) => RenameSessionResult;
}

export function useSessionSidebarAliases(
  options: UseSessionSidebarAliasesOptions,
): UseSessionSidebarAliasesResult {
  const { sessions, resolveDefaultTitle } = options;
  const [store, setStore] = useAliasStore(sessions);
  const sessionsByID = useSessionMapByID(sessions);
  const resolveSessionTitle = useSessionTitleResolver(store.aliases, resolveDefaultTitle);
  const resolveSessionTitleByID = useSessionTitleByIDResolver(
    store.aliases,
    sessionsByID,
    resolveDefaultTitle,
  );
  const renameSession = useRenameSessionAlias(setStore, sessionsByID);

  return {
    resolveSessionTitle,
    resolveSessionTitleByID,
    renameSession,
  };
}

function useAliasStore(sessions: SessionMetadata[]) {
  const [store, setStore] = useState(createInitialSessionAliasStore);

  useEffect(() => {
    const loaded = readAliasStoreFromWindow();
    setStore((previous) => {
      if (stringifySessionAliasStore(previous) === stringifySessionAliasStore(loaded)) {
        return previous;
      }
      return loaded;
    });
  }, []);

  useEffect(() => {
    setStore((previous) => {
      const sanitized = sanitizeSessionAliasStore(previous, sessions);
      if (stringifySessionAliasStore(previous) === stringifySessionAliasStore(sanitized)) {
        return previous;
      }
      return sanitized;
    });
  }, [sessions]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    window.localStorage.setItem(SESSION_ALIAS_STORAGE_KEY, stringifySessionAliasStore(store));
  }, [store]);

  return [store, setStore] as const;
}

function readAliasStoreFromWindow(): SessionAliasStoreV1 {
  if (typeof window === 'undefined') {
    return createInitialSessionAliasStore();
  }

  const raw = window.localStorage.getItem(SESSION_ALIAS_STORAGE_KEY);
  if (!raw) {
    return createInitialSessionAliasStore();
  }

  try {
    return parseSessionAliasStore(raw);
  } catch (error) {
    console.error('[SessionSidebar] failed to parse alias store', error);
    return createInitialSessionAliasStore();
  }
}

function useSessionMapByID(sessions: SessionMetadata[]) {
  return useMemo(() => {
    return new Map(sessions.map((session) => [session.id, session]));
  }, [sessions]);
}

function useSessionTitleResolver(
  aliases: Record<string, string>,
  resolveDefaultTitle: UseSessionSidebarAliasesOptions['resolveDefaultTitle'],
) {
  return useCallback((session: SessionMetadata): string => {
    return resolveSessionTitleValue(aliases[session.id], session.title, resolveDefaultTitle(session));
  }, [aliases, resolveDefaultTitle]);
}

function useSessionTitleByIDResolver(
  aliases: Record<string, string>,
  sessionsByID: Map<string, SessionMetadata>,
  resolveDefaultTitle: UseSessionSidebarAliasesOptions['resolveDefaultTitle'],
) {
  return useCallback((sessionID: string): string => {
    const session = sessionsByID.get(sessionID);
    if (!session) {
      return '';
    }
    return resolveSessionTitleValue(aliases[session.id], session.title, resolveDefaultTitle(session));
  }, [aliases, resolveDefaultTitle, sessionsByID]);
}

export function resolveSessionTitleValue(
  alias: string | undefined,
  backendTitle: string,
  fallbackTitle: string,
): string {
  const normalizedAlias = alias?.trim();
  if (normalizedAlias) {
    return normalizedAlias;
  }
  const normalizedBackendTitle = backendTitle.trim();
  if (normalizedBackendTitle) {
    return normalizedBackendTitle;
  }
  return fallbackTitle;
}

function useRenameSessionAlias(
  setStore: Dispatch<SetStateAction<SessionAliasStoreV1>>,
  sessionsByID: Map<string, SessionMetadata>,
) {
  return useCallback((sessionID: string, title: string): RenameSessionResult => {
    const normalizedID = sessionID.trim();
    if (!normalizedID || !sessionsByID.has(normalizedID)) {
      return { ok: false, error: 'unknown' };
    }

    const normalizedTitle = title.trim();
    if (!normalizedTitle) {
      return { ok: false, error: 'empty' };
    }

    setStore((previous) => setSessionAlias(previous, normalizedID, normalizedTitle));
    return { ok: true };
  }, [sessionsByID, setStore]);
}
