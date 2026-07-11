'use client';

import { useEffect, useMemo, type MutableRefObject } from 'react';
import {
  buildSessionPartitionViews,
  sanitizeSessionPartitionStore,
  storesAreEqual,
  type SessionSearchMatcher,
  type SessionPartitionStoreV1,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';
import {
  useSessionSidebarPartitionActions,
  type SessionSidebarPartitionActions,
} from './useSessionSidebarPartitionActions';
import { useSessionSidebarPartitionStore } from './useSessionSidebarPartitionStore';

interface UseSessionSidebarPartitionsOptions {
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
  searchQuery: string;
  unclassifiedName: string;
  requestFailedText: string;
  matchesSearch?: SessionSearchMatcher;
}

export interface UseSessionSidebarPartitionsResult extends SessionSidebarPartitionActions {
  partitionViews: SessionPartitionView[];
  partitionError: string;
  legacyMigrationAvailable: boolean;
  legacyMigrationRunning: boolean;
  runLegacyMigration: () => Promise<void>;
  discardLegacyMigration: () => void;
}

export function useSessionSidebarPartitions(
  options: UseSessionSidebarPartitionsOptions,
): UseSessionSidebarPartitionsResult {
  const { sessions, sessionsLoaded, searchQuery, unclassifiedName, requestFailedText, matchesSearch } = options;
  const {
    store,
    storeRef,
    partitionError,
    legacyMigrationAvailable,
    legacyMigrationRunning,
    applyOptimisticStore,
    reportPartitionError,
    runLegacyMigration,
    discardLegacyMigration,
  } = useSessionSidebarPartitionStore({ requestFailedText });

  useLoadedSessionPartitionSanitizer({
    applyOptimisticStore,
    sessions,
    sessionsLoaded,
    storeRef,
  });

  const partitionViews = useMemo(() => {
    return buildSessionPartitionViews({
      sessions,
      store,
      searchQuery,
      unclassifiedName,
      matchesSearch,
    });
  }, [matchesSearch, searchQuery, sessions, store, unclassifiedName]);
  const actions = useSessionSidebarPartitionActions({
    applyOptimisticStore,
    reportPartitionError,
    sessions,
    storeRef,
    unclassifiedName,
  });

  return {
    partitionViews,
    partitionError,
    legacyMigrationAvailable,
    legacyMigrationRunning,
    ...actions,
    runLegacyMigration,
    discardLegacyMigration,
  };
}

function useLoadedSessionPartitionSanitizer(options: {
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
  storeRef: MutableRefObject<SessionPartitionStoreV1>;
  applyOptimisticStore: (next: SessionPartitionStoreV1) => void;
}): void {
  const { applyOptimisticStore, sessions, sessionsLoaded, storeRef } = options;

  useEffect(() => {
    if (!sessionsLoaded) {
      return;
    }
    const sanitized = sanitizeSessionPartitionStore(storeRef.current, sessions);
    if (!storesAreEqual(storeRef.current, sanitized)) {
      applyOptimisticStore(sanitized);
    }
  }, [applyOptimisticStore, sessions, sessionsLoaded, storeRef]);
}
