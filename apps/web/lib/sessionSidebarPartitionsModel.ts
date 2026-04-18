import type { SessionMetadata } from '@/lib/types';

export const SESSION_PARTITION_STORAGE_KEY = 'ghost.web.session.partitions.v1';
export const UNCLASSIFIED_PARTITION_ID = '__unclassified__';
export const SESSION_PARTITION_VERSION = 1 as const;

export type PartitionNameValidationError = 'empty' | 'duplicate';

export interface SessionPartition {
  id: string;
  name: string;
}

export interface SessionPartitionStoreV1 {
  version: 1;
  partitions: SessionPartition[];
  assignments: Record<string, string>;
  orders: Record<string, string[]>;
}

export interface SessionPartitionView {
  id: string;
  name: string;
  sessions: SessionMetadata[];
}

export interface BuildSessionPartitionViewsInput {
  sessions: SessionMetadata[];
  store: SessionPartitionStoreV1;
  searchQuery: string;
  unclassifiedName: string;
}
