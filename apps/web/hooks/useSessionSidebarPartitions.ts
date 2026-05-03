'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  getSessionSidebarPartitions,
  putSessionSidebarPartitions,
} from '@/lib/api/sessions/api';
import { createClientTraceId } from '@/lib/api/trace';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import {
  UNCLASSIFIED_PARTITION_ID,
  buildSessionPartitionViews,
  createInitialSessionPartitionStore,
  createSessionPartition,
  moveSessionToPartition,
  sanitizeSessionPartitionStore,
  storesAreEqual,
  validatePartitionName,
  type PartitionNameValidationError,
  type SessionPartitionStoreV1,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitions';
import {
  deleteSessionPartition,
  renameSessionPartition,
  validatePartitionRenameName,
} from '@/lib/sessionSidebarPartitionMutations';
import type { SessionMetadata } from '@/lib/types';
import {
  buildPartitionID,
  coercePartitionStore,
  hasLegacyPartitionData,
  isEmptyPartitionStore,
  readLegacyPartitionStore,
  removeLegacyPartitionStore,
} from './sessionSidebarPartitionPersistence';

interface UseSessionSidebarPartitionsOptions {
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
  searchQuery: string;
  unclassifiedName: string;
  requestFailedText: string;
}

interface AddPartitionResult {
  ok: boolean;
  error?: PartitionNameValidationError;
  createdName?: string;
}

type PartitionMutationError = PartitionNameValidationError | 'missing' | 'readonly';

interface RenamePartitionResult {
  ok: boolean;
  error?: PartitionMutationError;
  renamedName?: string;
}

interface DeletePartitionResult {
  ok: boolean;
  error?: 'missing' | 'readonly';
  deletedName?: string;
}

interface UseSessionSidebarPartitionsResult {
  partitionViews: SessionPartitionView[];
  partitionError: string;
  addPartition: (name: string) => AddPartitionResult;
  renamePartition: (partitionID: string, name: string) => RenamePartitionResult;
  deletePartition: (partitionID: string) => DeletePartitionResult;
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
}

export function useSessionSidebarPartitions(
  options: UseSessionSidebarPartitionsOptions,
): UseSessionSidebarPartitionsResult {
  const { sessions, sessionsLoaded, searchQuery, unclassifiedName, requestFailedText } = options;
  const [store, setStore] = useState(createInitialSessionPartitionStore);
  const [partitionError, setPartitionError] = useState('');
  const storeRef = useRef(store);
  const confirmedRef = useRef(store);
  const pendingRef = useRef<SessionPartitionStoreV1 | null>(null);
  const persistingRef = useRef(false);
  const mountedRef = useRef(true);

  const replaceStore = useCallback((next: SessionPartitionStoreV1) => {
    storeRef.current = next;
    setStore(next);
  }, []);

  const flushPersistQueue = useCallback(async () => {
    if (persistingRef.current) {
      return;
    }

    const next = pendingRef.current;
    if (!next) {
      return;
    }
    pendingRef.current = null;
    persistingRef.current = true;

    try {
      const saved = coercePartitionStore(await putSessionSidebarPartitions(
        next,
        createClientTraceId('session-partitions'),
      ));
      if (!mountedRef.current) {
        return;
      }

      confirmedRef.current = saved;
      if (pendingRef.current === null && storesAreEqual(storeRef.current, next)) {
        replaceStore(saved);
      }
      setPartitionError('');
    } catch (error) {
      if (!mountedRef.current) {
        return;
      }
      pendingRef.current = null;
      replaceStore(confirmedRef.current);
      setPartitionError(toErrorMessage(error, requestFailedText));
    } finally {
      persistingRef.current = false;
      if (mountedRef.current && pendingRef.current) {
        ignorePromise(flushPersistQueue());
      }
    }
  }, [replaceStore, requestFailedText]);

  const enqueuePersist = useCallback((next: SessionPartitionStoreV1) => {
    pendingRef.current = next;
    ignorePromise(flushPersistQueue());
  }, [flushPersistQueue]);

  const applyOptimisticStore = useCallback((next: SessionPartitionStoreV1) => {
    setPartitionError('');
    replaceStore(next);
    enqueuePersist(next);
  }, [enqueuePersist, replaceStore]);

  const hydrateStore = useCallback(async () => {
    try {
      const remote = coercePartitionStore(await getSessionSidebarPartitions());
      if (!mountedRef.current) {
        return;
      }

      confirmedRef.current = remote;
      replaceStore(remote);
      setPartitionError('');

      const legacy = readLegacyPartitionStore();
      if (!isEmptyPartitionStore(remote) || !hasLegacyPartitionData(legacy)) {
        return;
      }

      const migrated = coercePartitionStore(await putSessionSidebarPartitions(
        legacy,
        createClientTraceId('session-partitions-migrate'),
      ));
      if (!mountedRef.current) {
        return;
      }

      confirmedRef.current = migrated;
      replaceStore(migrated);
      removeLegacyPartitionStore();
      setPartitionError('');
    } catch (error) {
      if (!mountedRef.current) {
        return;
      }
      replaceStore(confirmedRef.current);
      setPartitionError(toErrorMessage(error, requestFailedText));
    }
  }, [replaceStore, requestFailedText]);

  useEffect(() => {
    ignorePromise(hydrateStore());
  }, [hydrateStore]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (!sessionsLoaded) {
      return;
    }
    const sanitized = sanitizeSessionPartitionStore(storeRef.current, sessions);
    if (storesAreEqual(storeRef.current, sanitized)) {
      return;
    }
    applyOptimisticStore(sanitized);
  }, [applyOptimisticStore, sessions, sessionsLoaded]);

  const partitionViews = useMemo(() => {
    return buildSessionPartitionViews({
      sessions,
      store,
      searchQuery,
      unclassifiedName,
    });
  }, [searchQuery, sessions, store, unclassifiedName]);

  const addPartition = useCallback((name: string): AddPartitionResult => {
    const current = storeRef.current;
    const error = validatePartitionName(name, current, unclassifiedName);
    if (error) {
      return { ok: false, error };
    }

    const normalizedName = name.trim();
    applyOptimisticStore(createSessionPartition(current, normalizedName, buildPartitionID()));
    return { ok: true, createdName: normalizedName };
  }, [applyOptimisticStore, unclassifiedName]);

  const renamePartition = useCallback((partitionID: string, name: string): RenamePartitionResult => {
    const normalizedID = partitionID.trim();
    if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
      return { ok: false, error: 'readonly' };
    }

    const current = storeRef.current;
    const partition = current.partitions.find((item) => item.id === normalizedID);
    if (!partition) {
      return { ok: false, error: 'missing' };
    }

    const error = validatePartitionRenameName({
      value: name,
      store: current,
      partitionID: normalizedID,
      unclassifiedName,
    });
    if (error) {
      return { ok: false, error };
    }

    const normalizedName = name.trim();
    if (partition.name === normalizedName) {
      return { ok: true, renamedName: partition.name };
    }

    applyOptimisticStore(renameSessionPartition(current, normalizedID, normalizedName));
    return { ok: true, renamedName: normalizedName };
  }, [applyOptimisticStore, unclassifiedName]);

  const deletePartition = useCallback((partitionID: string): DeletePartitionResult => {
    const normalizedID = partitionID.trim();
    if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
      return { ok: false, error: 'readonly' };
    }

    const current = storeRef.current;
    const partition = current.partitions.find((item) => item.id === normalizedID);
    if (!partition) {
      return { ok: false, error: 'missing' };
    }

    applyOptimisticStore(deleteSessionPartition({
      store: current,
      sessions,
      partitionID: normalizedID,
    }));
    return { ok: true, deletedName: partition.name };
  }, [applyOptimisticStore, sessions]);

  const moveSession = useCallback((sessionID: string, partitionID: string, index: number) => {
    try {
      applyOptimisticStore(moveSessionToPartition({
        store: storeRef.current,
        sessions,
        sessionID,
        targetPartitionID: partitionID,
        targetIndex: index,
      }));
    } catch (error) {
      setPartitionError(toErrorMessage(error, requestFailedText));
    }
  }, [applyOptimisticStore, requestFailedText, sessions]);

  return {
    partitionViews,
    partitionError,
    addPartition,
    renamePartition,
    deletePartition,
    moveSession,
  };
}
