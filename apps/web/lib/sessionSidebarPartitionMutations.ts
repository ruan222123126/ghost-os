import type { PartitionNameValidationError, SessionPartitionStoreV1 } from '@/lib/sessionSidebarPartitionsModel';
import { UNCLASSIFIED_PARTITION_ID } from '@/lib/sessionSidebarPartitionsModel';
import { sanitizeSessionPartitionStore } from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

interface DeleteSessionPartitionInput {
  store: SessionPartitionStoreV1;
  sessions: SessionMetadata[];
  partitionID: string;
}

interface ValidatePartitionRenameNameInput {
  value: string;
  store: SessionPartitionStoreV1;
  partitionID: string;
  unclassifiedName: string;
}

export function validatePartitionRenameName(
  input: ValidatePartitionRenameNameInput,
): PartitionNameValidationError | undefined {
  const partition = findPartitionByID(input.store, input.partitionID);
  if (!partition) {
    throw new Error(`Partition not found: ${input.partitionID}`);
  }

  const normalizedName = normalizeName(input.value);
  if (!normalizedName) {
    return 'empty';
  }

  const lowerName = normalizedName.toLowerCase();
  if (lowerName === normalizeName(input.unclassifiedName).toLowerCase()) {
    return 'duplicate';
  }

  if (lowerName === normalizeName(partition.name).toLowerCase()) {
    return undefined;
  }

  const duplicated = input.store.partitions.some((item) => {
    if (item.id === partition.id) {
      return false;
    }
    return normalizeName(item.name).toLowerCase() === lowerName;
  });
  return duplicated ? 'duplicate' : undefined;
}

export function renameSessionPartition(
  store: SessionPartitionStoreV1,
  partitionID: string,
  name: string,
): SessionPartitionStoreV1 {
  const normalizedID = partitionID.trim();
  const normalizedName = normalizeName(name);
  if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
    throw new Error(`Invalid partition id: ${partitionID}`);
  }
  if (!normalizedName) {
    throw new Error('Partition name cannot be empty');
  }

  let found = false;
  const partitions = store.partitions.map((partition) => {
    if (partition.id !== normalizedID) {
      return partition;
    }
    found = true;
    return { ...partition, name: normalizedName };
  });
  if (!found) {
    throw new Error(`Partition not found: ${partitionID}`);
  }

  return {
    ...store,
    partitions,
  };
}

export function deleteSessionPartition(input: DeleteSessionPartitionInput): SessionPartitionStoreV1 {
  const normalizedID = input.partitionID.trim();
  if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
    throw new Error(`Invalid partition id: ${input.partitionID}`);
  }

  const sanitized = sanitizeSessionPartitionStore(input.store, input.sessions);
  const exists = sanitized.partitions.some((partition) => partition.id === normalizedID);
  if (!exists) {
    throw new Error(`Partition not found: ${input.partitionID}`);
  }

  return {
    ...sanitized,
    partitions: sanitized.partitions.filter((partition) => partition.id !== normalizedID),
    assignments: removePartitionAssignments(sanitized.assignments, normalizedID),
  };
}

function findPartitionByID(store: SessionPartitionStoreV1, partitionID: string) {
  const normalizedID = partitionID.trim();
  if (!normalizedID || normalizedID === UNCLASSIFIED_PARTITION_ID) {
    return undefined;
  }
  return store.partitions.find((partition) => partition.id === normalizedID);
}

function removePartitionAssignments(assignments: Record<string, string>, partitionID: string): Record<string, string> {
  const next: Record<string, string> = {};
  for (const [sessionID, assignedPartitionID] of Object.entries(assignments)) {
    if (assignedPartitionID === partitionID) {
      continue;
    }
    next[sessionID] = assignedPartitionID;
  }
  return next;
}

function normalizeName(value: string): string {
  return value.trim();
}
