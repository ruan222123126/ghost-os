'use client';

import {
  LEGACY_SESSION_PARTITION_STORAGE_KEY,
  SESSION_PARTITION_VERSION,
  createInitialSessionPartitionStore,
  parseSessionPartitionStore,
  type SessionPartitionStoreV1,
} from '@/lib/sessionSidebarPartitions';
import type { SessionSidebarPartitionState } from '@/lib/types';

export function coercePartitionStore(state: SessionSidebarPartitionState): SessionPartitionStoreV1 {
  return {
    version: SESSION_PARTITION_VERSION,
    partitions: [...state.partitions],
    assignments: { ...state.assignments },
  };
}

export function isEmptyPartitionStore(store: SessionPartitionStoreV1): boolean {
  return store.partitions.length === 0 && Object.keys(store.assignments).length === 0;
}

export function hasLegacyPartitionData(store: SessionPartitionStoreV1): boolean {
  return !isEmptyPartitionStore(store);
}

export function readLegacyPartitionStore(): SessionPartitionStoreV1 {
  if (typeof window === 'undefined') {
    return createInitialSessionPartitionStore();
  }

  try {
    const raw = window.localStorage.getItem(LEGACY_SESSION_PARTITION_STORAGE_KEY);
    if (!raw) {
      return createInitialSessionPartitionStore();
    }
    return parseSessionPartitionStore(raw);
  } catch (error) {
    console.error('[SessionSidebar] failed to read legacy partition store', error);
    return createInitialSessionPartitionStore();
  }
}

export function removeLegacyPartitionStore() {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.removeItem(LEGACY_SESSION_PARTITION_STORAGE_KEY);
  } catch (error) {
    console.error('[SessionSidebar] failed to remove legacy partition store', error);
  }
}

export function buildPartitionID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `partition:${crypto.randomUUID()}`;
  }
  return `partition:${Date.now()}:${Math.random().toString(16).slice(2)}`;
}
