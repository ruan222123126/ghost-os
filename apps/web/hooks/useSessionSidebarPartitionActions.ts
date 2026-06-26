import { useCallback, type MutableRefObject } from 'react';
import {
  UNCLASSIFIED_PARTITION_ID,
  createSessionPartition,
  moveSessionToPartition,
  validatePartitionName,
  type PartitionNameValidationError,
  type SessionPartitionStoreV1,
} from '@/lib/sessionSidebarPartitions';
import {
  deleteSessionPartition,
  renameSessionPartition,
  validatePartitionRenameName,
} from '@/lib/sessionSidebarPartitionMutations';
import type { SessionMetadata } from '@/lib/types';
import { buildPartitionID } from './sessionSidebarPartitionPersistence';

interface UseSessionSidebarPartitionActionsOptions {
  sessions: SessionMetadata[];
  storeRef: MutableRefObject<SessionPartitionStoreV1>;
  unclassifiedName: string;
  applyOptimisticStore: (next: SessionPartitionStoreV1) => void;
  reportPartitionError: (error: unknown) => void;
}

interface AddPartitionInput {
  store: SessionPartitionStoreV1;
  name: string;
  unclassifiedName: string;
  applyOptimisticStore: (next: SessionPartitionStoreV1) => void;
}

interface RenamePartitionInput extends AddPartitionInput {
  partitionID: string;
}

interface DeletePartitionInput {
  store: SessionPartitionStoreV1;
  sessions: SessionMetadata[];
  partitionID: string;
  applyOptimisticStore: (next: SessionPartitionStoreV1) => void;
}

interface MoveSessionInput extends DeletePartitionInput {
  sessionID: string;
  index: number;
  reportPartitionError: (error: unknown) => void;
}

export interface AddPartitionResult {
  ok: boolean;
  error?: PartitionNameValidationError;
  createdName?: string;
}

type PartitionMutationError = PartitionNameValidationError | 'missing' | 'readonly';

export interface RenamePartitionResult {
  ok: boolean;
  error?: PartitionMutationError;
  renamedName?: string;
}

export interface DeletePartitionResult {
  ok: boolean;
  error?: 'missing' | 'readonly';
  deletedName?: string;
}

export interface SessionSidebarPartitionActions {
  addPartition: (name: string) => AddPartitionResult;
  renamePartition: (partitionID: string, name: string) => RenamePartitionResult;
  deletePartition: (partitionID: string) => DeletePartitionResult;
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
}

export function useSessionSidebarPartitionActions(
  options: UseSessionSidebarPartitionActionsOptions,
): SessionSidebarPartitionActions {
  const { applyOptimisticStore, reportPartitionError, sessions, storeRef, unclassifiedName } = options;

  const addPartition = useCallback((name: string) => {
    return addSessionSidebarPartition({
      applyOptimisticStore,
      name,
      store: storeRef.current,
      unclassifiedName,
    });
  }, [applyOptimisticStore, storeRef, unclassifiedName]);

  const renamePartition = useCallback((partitionID: string, name: string) => {
    return renameSessionSidebarPartition({
      applyOptimisticStore,
      name,
      partitionID,
      store: storeRef.current,
      unclassifiedName,
    });
  }, [applyOptimisticStore, storeRef, unclassifiedName]);

  const deletePartition = useCallback((partitionID: string) => {
    return deleteSessionSidebarPartition({
      applyOptimisticStore,
      partitionID,
      sessions,
      store: storeRef.current,
    });
  }, [applyOptimisticStore, sessions, storeRef]);

  const moveSession = useCallback((sessionID: string, partitionID: string, index: number) => {
    moveSessionSidebarPartition({
      applyOptimisticStore,
      index,
      partitionID,
      reportPartitionError,
      sessionID,
      sessions,
      store: storeRef.current,
    });
  }, [applyOptimisticStore, reportPartitionError, sessions, storeRef]);

  return {
    addPartition,
    renamePartition,
    deletePartition,
    moveSession,
  };
}

function addSessionSidebarPartition(input: AddPartitionInput): AddPartitionResult {
  const error = validatePartitionName(input.name, input.store, input.unclassifiedName);
  if (error) {
    return { ok: false, error };
  }

  const normalizedName = input.name.trim();
  input.applyOptimisticStore(createSessionPartition(input.store, normalizedName, buildPartitionID()));
  return { ok: true, createdName: normalizedName };
}

function renameSessionSidebarPartition(input: RenamePartitionInput): RenamePartitionResult {
  const normalizedID = input.partitionID.trim();
  if (!isWritablePartitionID(normalizedID)) {
    return { ok: false, error: 'readonly' };
  }

  const partition = input.store.partitions.find((item) => item.id === normalizedID);
  if (!partition) {
    return { ok: false, error: 'missing' };
  }

  const error = validatePartitionRenameName({
    value: input.name,
    store: input.store,
    partitionID: normalizedID,
    unclassifiedName: input.unclassifiedName,
  });
  if (error) {
    return { ok: false, error };
  }

  const normalizedName = input.name.trim();
  if (partition.name === normalizedName) {
    return { ok: true, renamedName: partition.name };
  }

  input.applyOptimisticStore(renameSessionPartition(input.store, normalizedID, normalizedName));
  return { ok: true, renamedName: normalizedName };
}

function deleteSessionSidebarPartition(input: DeletePartitionInput): DeletePartitionResult {
  const normalizedID = input.partitionID.trim();
  if (!isWritablePartitionID(normalizedID)) {
    return { ok: false, error: 'readonly' };
  }

  const partition = input.store.partitions.find((item) => item.id === normalizedID);
  if (!partition) {
    return { ok: false, error: 'missing' };
  }

  input.applyOptimisticStore(deleteSessionPartition({
    store: input.store,
    sessions: input.sessions,
    partitionID: normalizedID,
  }));
  return { ok: true, deletedName: partition.name };
}

function moveSessionSidebarPartition(input: MoveSessionInput): void {
  try {
    input.applyOptimisticStore(moveSessionToPartition({
      store: input.store,
      sessions: input.sessions,
      sessionID: input.sessionID,
      targetPartitionID: input.partitionID,
      targetIndex: input.index,
    }));
  } catch (error) {
    input.reportPartitionError(error);
  }
}

function isWritablePartitionID(partitionID: string): boolean {
  return partitionID !== '' && partitionID !== UNCLASSIFIED_PARTITION_ID;
}
