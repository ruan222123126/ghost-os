import type {
  SessionMetadata,
  SessionSidebarPartition as SharedSessionSidebarPartition,
  SessionSidebarPartitionState as SharedSessionSidebarPartitionState,
} from '@/lib/types';

export const LEGACY_SESSION_PARTITION_STORAGE_KEY = 'ghost.web.session.partitions.v1';
export const UNCLASSIFIED_PARTITION_ID = '__unclassified__';
export const SESSION_PARTITION_VERSION = 1 as const;

export type PartitionNameValidationError = 'empty' | 'duplicate';

export type SessionPartition = SharedSessionSidebarPartition;
export type SessionPartitionStoreV1 = Omit<SharedSessionSidebarPartitionState, 'version'> & { version: 1 };

export interface SessionPartitionView {
  id: string;
  name: string;
  readOnly?: boolean;
  sessions: SessionMetadata[];
  childPartitions?: SessionPartitionView[];
}

export interface BuildSessionPartitionViewsInput {
  sessions: SessionMetadata[];
  store: SessionPartitionStoreV1;
  searchQuery: string;
  unclassifiedName: string;
}
