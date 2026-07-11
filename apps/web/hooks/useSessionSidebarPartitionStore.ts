'use client';

import { useCallback, useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from 'react';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import {
  createInitialSessionPartitionStore,
  type SessionPartitionStoreV1,
} from '@/lib/sessionSidebarPartitions';
import {
  hasLegacyPartitionData,
  readLegacyPartitionStore,
  removeLegacyPartitionStore,
} from './sessionSidebarPartitionPersistence';
import {
  flushQueuedPartitionPersist,
  hydratePartitionStore,
  migrateLegacyPartitionStore,
  type PartitionHydrationOptions,
  type PartitionPersistOptions,
  type PartitionStoreControls,
} from './sessionSidebarPartitionStorePersistence';

interface UseSessionSidebarPartitionStoreOptions {
  requestFailedText: string;
}

interface UseSessionSidebarPartitionStoreResult {
  store: SessionPartitionStoreV1;
  storeRef: MutableRefObject<SessionPartitionStoreV1>;
  partitionError: string;
  legacyMigrationAvailable: boolean;
  legacyMigrationRunning: boolean;
  applyOptimisticStore: (next: SessionPartitionStoreV1) => void;
  reportPartitionError: (error: unknown) => void;
  runLegacyMigration: () => Promise<void>;
  discardLegacyMigration: () => void;
}

interface PartitionStoreState extends PartitionStoreControls {
  store: SessionPartitionStoreV1;
}

interface LegacyPartitionMigrationState {
  legacyMigrationAvailable: boolean;
  legacyMigrationRunning: boolean;
  setLegacyMigrationAvailable: Dispatch<SetStateAction<boolean>>;
  runLegacyMigration: () => Promise<void>;
  discardLegacyMigration: () => void;
}

export function useSessionSidebarPartitionStore(
  options: UseSessionSidebarPartitionStoreOptions,
): UseSessionSidebarPartitionStoreResult {
  const { requestFailedText } = options;
  const mountedRef = useMountedRef();
  const storeState = usePartitionStoreState();
  const [partitionError, setPartitionError] = useState('');
  const legacyMigration = useLegacyPartitionMigration({
    ...storeState,
    mountedRef,
    requestFailedText,
    setPartitionError,
  });
  const applyOptimisticStore = usePartitionPersistQueue({
    ...storeState,
    mountedRef,
    requestFailedText,
    setPartitionError,
  });
  const reportPartitionError = useCallback((error: unknown) => {
    setPartitionError(toErrorMessage(error, requestFailedText));
  }, [requestFailedText]);

  usePartitionHydration({
    ...storeState,
    mountedRef,
    requestFailedText,
    setLegacyMigrationAvailable: legacyMigration.setLegacyMigrationAvailable,
    setPartitionError,
  });

  return {
    store: storeState.store,
    storeRef: storeState.storeRef,
    partitionError,
    legacyMigrationAvailable: legacyMigration.legacyMigrationAvailable,
    legacyMigrationRunning: legacyMigration.legacyMigrationRunning,
    applyOptimisticStore,
    reportPartitionError,
    runLegacyMigration: legacyMigration.runLegacyMigration,
    discardLegacyMigration: legacyMigration.discardLegacyMigration,
  };
}

function useMountedRef(): MutableRefObject<boolean> {
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  return mountedRef;
}

function usePartitionStoreState(): PartitionStoreState {
  const [store, setStore] = useState(createInitialSessionPartitionStore);
  const storeRef = useRef(store);
  const confirmedRef = useRef(store);

  const replaceStore = useCallback((next: SessionPartitionStoreV1) => {
    storeRef.current = next;
    setStore(next);
  }, []);

  return {
    store,
    storeRef,
    confirmedRef,
    replaceStore,
  };
}

function usePartitionPersistQueue(options: PartitionPersistOptions): (next: SessionPartitionStoreV1) => void {
  const {
    confirmedRef,
    mountedRef,
    replaceStore,
    requestFailedText,
    setPartitionError,
    storeRef,
  } = options;
  const pendingRef = useRef<SessionPartitionStoreV1 | null>(null);
  const persistingRef = useRef(false);

  const flushPersistQueue = useCallback(async () => {
    await flushQueuedPartitionPersist({
      confirmedRef,
      mountedRef,
      pendingRef,
      persistingRef,
      replaceStore,
      requestFailedText,
      setPartitionError,
      storeRef,
    });
  }, [confirmedRef, mountedRef, replaceStore, requestFailedText, setPartitionError, storeRef]);

  return useCallback((next: SessionPartitionStoreV1) => {
    setPartitionError('');
    replaceStore(next);
    pendingRef.current = next;
    ignorePromise(flushPersistQueue());
  }, [flushPersistQueue, replaceStore, setPartitionError]);
}

function usePartitionHydration(options: PartitionHydrationOptions): void {
  const {
    confirmedRef,
    mountedRef,
    replaceStore,
    requestFailedText,
    setLegacyMigrationAvailable,
    setPartitionError,
    storeRef,
  } = options;

  const hydrateStore = useCallback(async () => {
    await hydratePartitionStore({
      confirmedRef,
      mountedRef,
      replaceStore,
      requestFailedText,
      setLegacyMigrationAvailable,
      setPartitionError,
      storeRef,
    });
  }, [
    confirmedRef,
    mountedRef,
    replaceStore,
    requestFailedText,
    setLegacyMigrationAvailable,
    setPartitionError,
    storeRef,
  ]);

  useEffect(() => {
    ignorePromise(hydrateStore());
  }, [hydrateStore]);
}

function useLegacyPartitionMigration(options: PartitionPersistOptions): LegacyPartitionMigrationState {
  const {
    confirmedRef,
    mountedRef,
    replaceStore,
    requestFailedText,
    setPartitionError,
    storeRef,
  } = options;
  const [legacyMigrationAvailable, setLegacyMigrationAvailable] = useState(() => {
    return hasLegacyPartitionData(readLegacyPartitionStore());
  });
  const [legacyMigrationRunning, setLegacyMigrationRunning] = useState(false);

  const runLegacyMigration = useCallback(async () => {
    await migrateLegacyPartitionStore({
      confirmedRef,
      mountedRef,
      replaceStore,
      requestFailedText,
      setLegacyMigrationAvailable,
      setLegacyMigrationRunning,
      setPartitionError,
      storeRef,
    });
  }, [
    confirmedRef,
    mountedRef,
    replaceStore,
    requestFailedText,
    setPartitionError,
    storeRef,
  ]);

  const discardLegacyMigration = useCallback(() => {
    removeLegacyPartitionStore();
    setLegacyMigrationAvailable(false);
  }, []);

  return {
    legacyMigrationAvailable,
    legacyMigrationRunning,
    setLegacyMigrationAvailable,
    runLegacyMigration,
    discardLegacyMigration,
  };
}
