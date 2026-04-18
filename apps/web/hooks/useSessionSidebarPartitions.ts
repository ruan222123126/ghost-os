'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  SESSION_PARTITION_STORAGE_KEY,
  UNCLASSIFIED_PARTITION_ID,
  buildSessionPartitionViews,
  createInitialSessionPartitionStore,
  createSessionPartition,
  moveSessionToPartition,
  parseSessionPartitionStore,
  sanitizeSessionPartitionStore,
  storesAreEqual,
  stringifySessionPartitionStore,
  validatePartitionName,
  type PartitionNameValidationError,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitions';
import {
  deleteSessionPartition,
  renameSessionPartition,
  validatePartitionRenameName,
} from '@/lib/sessionSidebarPartitionMutations';
import type { SessionMetadata } from '@/lib/types';

interface UseSessionSidebarPartitionsOptions {
  sessions: SessionMetadata[];
  searchQuery: string;
  unclassifiedName: string;
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
  addPartition: (name: string) => AddPartitionResult;
  renamePartition: (partitionID: string, name: string) => RenamePartitionResult;
  deletePartition: (partitionID: string) => DeletePartitionResult;
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
}

export function useSessionSidebarPartitions(
  options: UseSessionSidebarPartitionsOptions,
): UseSessionSidebarPartitionsResult {
  const { sessions, searchQuery, unclassifiedName } = options;
  const [store, setStore] = useState(createInitialSessionPartitionStore);
  const [hasHydratedStore, setHasHydratedStore] = useState(false);

  useEffect(() => {
    const loaded = readStoredPartitionStore();
    setStore((previous) => {
      if (storesAreEqual(previous, loaded)) {
        return previous;
      }
      return loaded;
    });
    setHasHydratedStore(true);
  }, []);

  useEffect(() => {
    setStore((previous) => {
      const sanitized = sanitizeSessionPartitionStore(previous, sessions);
      if (storesAreEqual(previous, sanitized)) {
        return previous;
      }
      return sanitized;
    });
  }, [sessions]);

  useEffect(() => {
    if (!canPersistPartitionStore(hasHydratedStore)) {
      return;
    }
    writeStoredPartitionStore(store);
  }, [hasHydratedStore, store]);

  const partitionViews = useMemo(() => {
    return buildSessionPartitionViews({
      sessions,
      store,
      searchQuery,
      unclassifiedName,
    });
  }, [searchQuery, sessions, store, unclassifiedName]);

  const addPartition = useCallback((name: string): AddPartitionResult => {
    const error = validatePartitionName(name, store, unclassifiedName);
    if (error) {
      return { ok: false, error };
    }

    const normalizedName = name.trim();
    const next = createSessionPartition(store, normalizedName, buildPartitionID());
    setStore(next);
    return { ok: true, createdName: normalizedName };
  }, [store, unclassifiedName]);

  const renamePartition = useCallback((partitionID: string, name: string): RenamePartitionResult => {
    const normalizedID = partitionID.trim();
    if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
      return { ok: false, error: 'readonly' };
    }

    const partition = store.partitions.find((item) => item.id === normalizedID);
    if (!partition) {
      return { ok: false, error: 'missing' };
    }

    const error = validatePartitionRenameName({
      value: name,
      store,
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

    const next = renameSessionPartition(store, normalizedID, normalizedName);
    setStore(next);
    return { ok: true, renamedName: normalizedName };
  }, [store, unclassifiedName]);

  const deletePartition = useCallback((partitionID: string): DeletePartitionResult => {
    const normalizedID = partitionID.trim();
    if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
      return { ok: false, error: 'readonly' };
    }

    const partition = store.partitions.find((item) => item.id === normalizedID);
    if (!partition) {
      return { ok: false, error: 'missing' };
    }

    const next = deleteSessionPartition({
      store,
      sessions,
      partitionID: normalizedID,
    });
    setStore(next);
    return { ok: true, deletedName: partition.name };
  }, [sessions, store]);

  const moveSession = useCallback((sessionID: string, partitionID: string, index: number) => {
    setStore((previous) => {
      return moveSessionToPartition({
        store: previous,
        sessions,
        sessionID,
        targetPartitionID: partitionID,
        targetIndex: index,
      });
    });
  }, [sessions]);

  return {
    partitionViews,
    addPartition,
    renamePartition,
    deletePartition,
    moveSession,
  };
}

function readStoredPartitionStore(): ReturnType<typeof createInitialSessionPartitionStore> {
  if (typeof window === 'undefined') {
    return createInitialSessionPartitionStore();
  }

  const raw = readStoredPartitionRaw();
  if (!raw) {
    return createInitialSessionPartitionStore();
  }

  return parseStoredPartitionStore(raw);
}

function readStoredPartitionRaw(): string | null {
  try {
    return window.localStorage.getItem(SESSION_PARTITION_STORAGE_KEY);
  } catch (error) {
    console.error('[SessionSidebar] failed to read partition store', error);
    return null;
  }
}

function writeStoredPartitionStore(
  store: ReturnType<typeof createInitialSessionPartitionStore>,
) {
  try {
    window.localStorage.setItem(
      SESSION_PARTITION_STORAGE_KEY,
      stringifySessionPartitionStore(store),
    );
  } catch (error) {
    console.error('[SessionSidebar] failed to write partition store', error);
  }
}

function parseStoredPartitionStore(raw: string) {
  try {
    return parseSessionPartitionStore(raw);
  } catch (error) {
    console.error('[SessionSidebar] failed to parse partition store', error);
    return createInitialSessionPartitionStore();
  }
}

function buildPartitionID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `partition:${crypto.randomUUID()}`;
  }
  return `partition:${Date.now()}:${Math.random().toString(16).slice(2)}`;
}

function canPersistPartitionStore(hasHydratedStore: boolean): boolean {
  return hasHydratedStore && typeof window !== 'undefined';
}
