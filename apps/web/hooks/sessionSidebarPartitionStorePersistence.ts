import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import {
  getSessionSidebarPartitions,
  putSessionSidebarPartitions,
} from '@/lib/api/sessions/api';
import { createClientTraceId } from '@/lib/api/trace';
import { toErrorMessage } from '@/lib/errors';
import {
  storesAreEqual,
  type SessionPartitionStoreV1,
} from '@/lib/sessionSidebarPartitions';
import {
  coercePartitionStore,
  hasLegacyPartitionData,
  readLegacyPartitionStore,
  removeLegacyPartitionStore,
} from './sessionSidebarPartitionPersistence';

const SESSION_PARTITIONS_TRACE_PREFIX = 'session-partitions';
const SESSION_PARTITIONS_MIGRATE_TRACE_PREFIX = 'session-partitions-migrate';

export interface PartitionStoreControls {
  storeRef: MutableRefObject<SessionPartitionStoreV1>;
  confirmedRef: MutableRefObject<SessionPartitionStoreV1>;
  replaceStore: (next: SessionPartitionStoreV1) => void;
}

export interface PartitionPersistOptions extends PartitionStoreControls {
  mountedRef: MutableRefObject<boolean>;
  requestFailedText: string;
  setPartitionError: Dispatch<SetStateAction<string>>;
}

export interface PartitionHydrationOptions extends PartitionPersistOptions {
  setLegacyMigrationAvailable: Dispatch<SetStateAction<boolean>>;
}

export interface LegacyMigrationOptions extends PartitionPersistOptions {
  setLegacyMigrationAvailable: Dispatch<SetStateAction<boolean>>;
  setLegacyMigrationRunning: Dispatch<SetStateAction<boolean>>;
}

interface PartitionPersistQueueOptions extends PartitionPersistOptions {
  pendingRef: MutableRefObject<SessionPartitionStoreV1 | null>;
  persistingRef: MutableRefObject<boolean>;
}

export async function flushQueuedPartitionPersist(options: PartitionPersistQueueOptions): Promise<void> {
  if (options.persistingRef.current) {
    return;
  }

  options.persistingRef.current = true;
  try {
    let next = takePendingPartitionStore(options.pendingRef);
    while (next && options.mountedRef.current) {
      await persistPartitionStore(options, next);
      next = takePendingPartitionStore(options.pendingRef);
    }
  } finally {
    options.persistingRef.current = false;
  }
}

export async function hydratePartitionStore(options: PartitionHydrationOptions): Promise<void> {
  try {
    const remote = coercePartitionStore(await getSessionSidebarPartitions());
    if (!options.mountedRef.current) {
      return;
    }

    options.confirmedRef.current = remote;
    options.replaceStore(remote);
    options.setLegacyMigrationAvailable(hasLegacyPartitionData(readLegacyPartitionStore()));
    options.setPartitionError('');
  } catch (error) {
    if (!options.mountedRef.current) {
      return;
    }
    options.replaceStore(options.confirmedRef.current);
    options.setPartitionError(toErrorMessage(error, options.requestFailedText));
  }
}

export async function migrateLegacyPartitionStore(options: LegacyMigrationOptions): Promise<void> {
  const legacy = readLegacyPartitionStore();
  if (!hasLegacyPartitionData(legacy)) {
    options.setLegacyMigrationAvailable(false);
    return;
  }

  options.setLegacyMigrationRunning(true);
  try {
    const migrated = coercePartitionStore(await putSessionSidebarPartitions(
      legacy,
      createClientTraceId(SESSION_PARTITIONS_MIGRATE_TRACE_PREFIX),
    ));
    commitLegacyPartitionMigration(options, migrated);
  } catch (error) {
    if (options.mountedRef.current) {
      options.setPartitionError(toErrorMessage(error, options.requestFailedText));
    }
  } finally {
    if (options.mountedRef.current) {
      options.setLegacyMigrationRunning(false);
    }
  }
}

function takePendingPartitionStore(
  pendingRef: MutableRefObject<SessionPartitionStoreV1 | null>,
): SessionPartitionStoreV1 | null {
  const next = pendingRef.current;
  pendingRef.current = null;
  return next;
}

async function persistPartitionStore(
  options: PartitionPersistQueueOptions,
  next: SessionPartitionStoreV1,
): Promise<void> {
  try {
    const saved = coercePartitionStore(await putSessionSidebarPartitions(
      next,
      createClientTraceId(SESSION_PARTITIONS_TRACE_PREFIX),
    ));
    commitPersistedPartitionStore(options, next, saved);
  } catch (error) {
    rollbackPartitionStore(options, error);
  }
}

function commitPersistedPartitionStore(
  options: PartitionPersistQueueOptions,
  requested: SessionPartitionStoreV1,
  saved: SessionPartitionStoreV1,
): void {
  if (!options.mountedRef.current) {
    return;
  }

  options.confirmedRef.current = saved;
  if (options.pendingRef.current === null && storesAreEqual(options.storeRef.current, requested)) {
    options.replaceStore(saved);
  }
  options.setPartitionError('');
}

function rollbackPartitionStore(options: PartitionPersistQueueOptions, error: unknown): void {
  if (!options.mountedRef.current) {
    return;
  }

  options.pendingRef.current = null;
  options.replaceStore(options.confirmedRef.current);
  options.setPartitionError(toErrorMessage(error, options.requestFailedText));
}

function commitLegacyPartitionMigration(
  options: LegacyMigrationOptions,
  migrated: SessionPartitionStoreV1,
): void {
  if (!options.mountedRef.current) {
    return;
  }

  options.confirmedRef.current = migrated;
  options.replaceStore(migrated);
  removeLegacyPartitionStore();
  options.setLegacyMigrationAvailable(false);
  options.setPartitionError('');
}
