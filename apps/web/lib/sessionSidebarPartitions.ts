import type { SessionMetadata } from '@/lib/types';
import {
  LEGACY_SESSION_PARTITION_STORAGE_KEY,
  UNCLASSIFIED_PARTITION_ID,
  type BuildSessionPartitionViewsInput,
  type PartitionNameValidationError,
  type SessionPartition,
  type SessionPartitionStoreV1,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitionsModel';
import { sortSessionIDsByRecentActivity } from '@/lib/sessionSidebarSessionSort';

export {
  LEGACY_SESSION_PARTITION_STORAGE_KEY,
  SESSION_PARTITION_VERSION,
  UNCLASSIFIED_PARTITION_ID,
  type BuildSessionPartitionViewsInput,
  type PartitionNameValidationError,
  type SessionPartition,
  type SessionPartitionStoreV1,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitionsModel';

export {
  createInitialSessionPartitionStore,
  parseSessionPartitionStore,
  stringifySessionPartitionStore,
  storesAreEqual,
} from '@/lib/sessionSidebarPartitionsCodec';

export function validatePartitionName(
  value: string,
  store: SessionPartitionStoreV1,
  unclassifiedName: string,
): PartitionNameValidationError | undefined {
  const normalizedName = normalizeName(value);
  if (!normalizedName) {
    return 'empty';
  }

  const lowerName = normalizedName.toLowerCase();
  if (lowerName === normalizeName(unclassifiedName).toLowerCase()) {
    return 'duplicate';
  }

  const duplicated = store.partitions.some((partition) => {
    return normalizeName(partition.name).toLowerCase() === lowerName;
  });
  return duplicated ? 'duplicate' : undefined;
}

export function createSessionPartition(
  store: SessionPartitionStoreV1,
  name: string,
  partitionID: string,
): SessionPartitionStoreV1 {
  const normalizedName = normalizeName(name);
  const normalizedID = partitionID.trim();
  if (!normalizedName) {
    throw new Error('Partition name cannot be empty');
  }
  if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
    throw new Error(`Invalid partition id: ${partitionID}`);
  }
  if (store.partitions.some((partition) => partition.id === normalizedID)) {
    throw new Error(`Partition already exists: ${partitionID}`);
  }

  return {
    ...store,
    partitions: [...store.partitions, { id: normalizedID, name: normalizedName }],
  };
}

export function sanitizeSessionPartitionStore(
  store: SessionPartitionStoreV1,
  sessions: SessionMetadata[],
): SessionPartitionStoreV1 {
  const knownSessionIDs = new Set(sessions.map((session) => session.id));
  const partitions = dedupePartitions(store.partitions);
  const partitionIDs = buildPartitionIDSet(partitions);

  return {
    version: store.version,
    partitions,
    assignments: pruneAssignments(store.assignments, knownSessionIDs, partitionIDs),
  };
}

export function buildSessionPartitionViews(input: BuildSessionPartitionViewsInput): SessionPartitionView[] {
  const sanitized = sanitizeSessionPartitionStore(input.store, input.sessions);
  const partitionMetas = [
    { id: UNCLASSIFIED_PARTITION_ID, name: input.unclassifiedName },
    ...sanitized.partitions,
  ];

  const byID = new Map(input.sessions.map((session) => [session.id, session]));
  const grouped = buildOrderedSessionIDsByPartition(input.sessions, sanitized, partitionMetas.map((meta) => meta.id));
  const query = input.searchQuery.trim().toLowerCase();
  const hideEmpty = query.length > 0;

  return partitionMetas
    .map((meta) => ({
      id: meta.id,
      name: meta.name,
      sessions: (grouped[meta.id] ?? [])
        .map((sessionID) => byID.get(sessionID))
        .filter((session): session is SessionMetadata => Boolean(session))
        .filter((session) => session.id.toLowerCase().includes(query)),
    }))
    .filter((view) => !hideEmpty || view.sessions.length > 0);
}

export function moveSessionToPartition(input: {
  store: SessionPartitionStoreV1;
  sessions: SessionMetadata[];
  sessionID: string;
  targetPartitionID: string;
  targetIndex: number;
}): SessionPartitionStoreV1 {
  const { store, sessions, sessionID, targetPartitionID } = input;
  const sanitized = sanitizeSessionPartitionStore(store, sessions);
  const knownSessionIDs = new Set(sessions.map((session) => session.id));
  if (!knownSessionIDs.has(sessionID)) {
    throw new Error(`Unknown session id: ${sessionID}`);
  }

  const partitionIDs = [UNCLASSIFIED_PARTITION_ID, ...sanitized.partitions.map((partition) => partition.id)];
  if (!partitionIDs.includes(targetPartitionID)) {
    throw new Error(`Unknown partition id: ${targetPartitionID}`);
  }

  const assignments = { ...sanitized.assignments };
  if (targetPartitionID === UNCLASSIFIED_PARTITION_ID) {
    delete assignments[sessionID];
  } else {
    assignments[sessionID] = targetPartitionID;
  }

  return {
    ...sanitized,
    assignments,
  };
}

function buildPartitionIDSet(partitions: SessionPartition[]): Set<string> {
  const ids = new Set<string>([UNCLASSIFIED_PARTITION_ID]);
  for (const partition of partitions) {
    ids.add(partition.id);
  }
  return ids;
}

function pruneAssignments(
  assignments: Record<string, string>,
  knownSessionIDs: Set<string>,
  partitionIDs: Set<string>,
): Record<string, string> {
  const next: Record<string, string> = {};
  for (const [sessionID, partitionID] of Object.entries(assignments)) {
    if (!knownSessionIDs.has(sessionID)) {
      continue;
    }
    if (!partitionIDs.has(partitionID) || partitionID === UNCLASSIFIED_PARTITION_ID) {
      continue;
    }
    next[sessionID] = partitionID;
  }
  return next;
}

function buildOrderedSessionIDsByPartition(
  sessions: SessionMetadata[],
  store: SessionPartitionStoreV1,
  partitionIDs: string[],
): Record<string, string[]> {
  const sessionByID = new Map(sessions.map((session) => [session.id, session]));
  const grouped = Object.fromEntries(partitionIDs.map((partitionID) => [partitionID, [] as string[]]));

  for (const session of sessions) {
    const assigned = store.assignments[session.id];
    const target = partitionIDs.includes(assigned) ? assigned : UNCLASSIFIED_PARTITION_ID;
    grouped[target].push(session.id);
  }

  for (const partitionID of partitionIDs) {
    grouped[partitionID] = sortSessionIDsByRecentActivity(grouped[partitionID], sessionByID);
  }

  return grouped;
}

function dedupePartitions(partitions: SessionPartition[]): SessionPartition[] {
  const seen = new Set<string>();
  const deduped: SessionPartition[] = [];

  for (const partition of partitions) {
    const id = partition.id.trim();
    const name = normalizeName(partition.name);
    if (!id || !name || id === UNCLASSIFIED_PARTITION_ID || seen.has(id)) {
      continue;
    }
    seen.add(id);
    deduped.push({ id, name });
  }

  return deduped;
}

function normalizeName(value: string): string {
  return value.trim();
}
