import {
  SESSION_PARTITION_VERSION,
  UNCLASSIFIED_PARTITION_ID,
  type SessionPartition,
  type SessionPartitionStoreV1,
} from '@/lib/sessionSidebarPartitionsModel';

export function createInitialSessionPartitionStore(): SessionPartitionStoreV1 {
  return {
    version: SESSION_PARTITION_VERSION,
    partitions: [],
    assignments: {},
    orders: {},
  };
}

export function stringifySessionPartitionStore(store: SessionPartitionStoreV1): string {
  return JSON.stringify(store);
}

export function parseSessionPartitionStore(raw: string): SessionPartitionStoreV1 {
  const payload = parsePartitionStorePayload(raw);
  if (payload.version !== SESSION_PARTITION_VERSION) {
    throw new Error(`Invalid session partition version: ${payload.version}`);
  }

  return {
    version: SESSION_PARTITION_VERSION,
    partitions: payload.partitions,
    assignments: payload.assignments,
    orders: payload.orders,
  };
}

export function storesAreEqual(a: SessionPartitionStoreV1, b: SessionPartitionStoreV1): boolean {
  return stringifySessionPartitionStore(a) === stringifySessionPartitionStore(b);
}

interface ParsedPartitionStore {
  version: number;
  partitions: SessionPartition[];
  assignments: Record<string, string>;
  orders: Record<string, string[]>;
}

function parsePartitionStorePayload(raw: string): ParsedPartitionStore {
  let decoded: unknown;
  try {
    decoded = JSON.parse(raw);
  } catch (error) {
    throw new Error(`Invalid session partition payload JSON: ${String(error)}`);
  }

  const root = expectRecord(decoded, 'session partition payload');
  const version = expectNumber(root.version, 'session partition payload.version');
  const partitions = parsePartitions(root.partitions);
  const assignments = parseAssignments(root.assignments);
  const orders = parseOrders(root.orders);

  return { version, partitions, assignments, orders };
}

function parsePartitions(value: unknown): SessionPartition[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid session partition payload.partitions: expected array');
  }

  return value.map((item, index) => {
    const label = `session partition payload.partitions[${index}]`;
    const record = expectRecord(item, label);
    const id = expectString(record.id, `${label}.id`).trim();
    const name = expectString(record.name, `${label}.name`).trim();
    if (!id || !name || id === UNCLASSIFIED_PARTITION_ID) {
      throw new Error(`Invalid ${label}`);
    }
    return { id, name };
  });
}

function parseAssignments(value: unknown): Record<string, string> {
  const record = expectRecord(value, 'session partition payload.assignments');
  const parsed: Record<string, string> = {};

  for (const [sessionID, partitionID] of Object.entries(record)) {
    const key = sessionID.trim();
    const val = expectString(partitionID, `session partition payload.assignments.${sessionID}`).trim();
    if (!key || !val) {
      continue;
    }
    parsed[key] = val;
  }

  return parsed;
}

function parseOrders(value: unknown): Record<string, string[]> {
  const record = expectRecord(value, 'session partition payload.orders');
  const parsed: Record<string, string[]> = {};

  for (const [key, rawIDs] of Object.entries(record)) {
    if (!Array.isArray(rawIDs)) {
      throw new Error(`Invalid session partition payload.orders.${key}: expected array`);
    }

    const partitionID = key.trim();
    if (!partitionID) {
      continue;
    }

    parsed[partitionID] = rawIDs
      .map((item, index) => expectString(item, `session partition payload.orders.${key}[${index}]`).trim())
      .filter((id) => id.length > 0);
  }

  return parsed;
}

function expectRecord(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected object`);
  }
  return value as Record<string, unknown>;
}

function expectString(value: unknown, label: string): string {
  if (typeof value !== 'string') {
    throw new Error(`Invalid ${label}: expected string`);
  }
  return value;
}

function expectNumber(value: unknown, label: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Invalid ${label}: expected number`);
  }
  return value;
}
