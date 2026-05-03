import {
  createEmptyWorkflowDraft,
  type WorkflowCanvasDraft,
} from '@/lib/workflow-editor';

const ORCHESTRATION_INDEX_KEY = 'ghost.orchestration.index.v1';
const ORCHESTRATION_ITEM_KEY_PREFIX = 'ghost.orchestration.item.v1';
const EMPTY_NAME_ERROR = 'orchestration name is required';

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

export interface OrchestrationSummary {
  id: string;
  name: string;
  createdAt: number;
  updatedAt: number;
  stepCount: number;
}

export interface OrchestrationRecord {
  id: string;
  name: string;
  createdAt: number;
  updatedAt: number;
  draft: WorkflowCanvasDraft;
}

interface CreateOrchestrationInput {
  name: string;
}

interface PersistedOrchestrationRecord {
  id: string;
  name: string;
  createdAt: number;
  updatedAt: number;
  draft: WorkflowCanvasDraft;
}

export function createOrchestration(
  input: CreateOrchestrationInput,
  storage = getBrowserStorage(),
): OrchestrationRecord {
  const name = normalizeOrchestrationName(input.name);
  const now = Date.now();
  const record: PersistedOrchestrationRecord = {
    id: globalThis.crypto.randomUUID(),
    name,
    createdAt: now,
    updatedAt: now,
    draft: cloneDraft(createEmptyWorkflowDraft('edit')),
  };

  writeRecord(storage, record);
  writeIndex(storage, [record.id, ...readIndex(storage)]);
  return toPublicRecord(record);
}

export function getOrchestrationByID(
  id: string,
  storage = getBrowserStorage(),
): OrchestrationRecord | null {
  const record = readRecord(storage, id);
  return record ? toPublicRecord(record) : null;
}

export function listOrchestrationSummaries(
  storage = getBrowserStorage(),
): OrchestrationSummary[] {
  const records = readAllRecords(storage);
  return records
    .sort((left, right) => right.updatedAt - left.updatedAt)
    .map((record) => ({
      id: record.id,
      name: record.name,
      createdAt: record.createdAt,
      updatedAt: record.updatedAt,
      stepCount: workflowStepCount(record.draft),
    }));
}

export function saveOrchestrationDraft(
  id: string,
  draft: WorkflowCanvasDraft,
  storage = getBrowserStorage(),
): OrchestrationRecord {
  const record = requireRecord(storage, id);
  const nextRecord: PersistedOrchestrationRecord = {
    ...record,
    updatedAt: Date.now(),
    draft: cloneDraft({
      ...draft,
      mode: 'edit',
    }),
  };

  writeRecord(storage, nextRecord);
  return toPublicRecord(nextRecord);
}

function getBrowserStorage(): Storage {
  if (typeof window === 'undefined') {
    throw new Error('orchestration storage is only available in the browser');
  }

  return window.localStorage;
}

function normalizeOrchestrationName(raw: string): string {
  const name = raw.trim();
  if (name.length === 0) {
    throw new Error(EMPTY_NAME_ERROR);
  }

  return name;
}

function readAllRecords(storage: StorageLike): PersistedOrchestrationRecord[] {
  const records: PersistedOrchestrationRecord[] = [];
  for (const id of readIndex(storage)) {
    const record = readRecord(storage, id);
    if (record) {
      records.push(record);
    }
  }
  return records;
}

function readIndex(storage: StorageLike): string[] {
  const raw = storage.getItem(ORCHESTRATION_INDEX_KEY);
  if (!raw) {
    return [];
  }

  const parsed = parseJSON(raw, ORCHESTRATION_INDEX_KEY);
  if (!Array.isArray(parsed) || parsed.some((item) => typeof item !== 'string')) {
    throw new Error('invalid orchestration index payload');
  }

  return [...new Set(parsed)];
}

function writeIndex(storage: StorageLike, ids: string[]) {
  storage.setItem(ORCHESTRATION_INDEX_KEY, JSON.stringify(ids));
}

function readRecord(storage: StorageLike, id: string): PersistedOrchestrationRecord | null {
  const raw = storage.getItem(storageKey(id));
  if (!raw) {
    return null;
  }

  const parsed = parseJSON(raw, storageKey(id));
  return validateRecord(parsed);
}

function requireRecord(storage: StorageLike, id: string): PersistedOrchestrationRecord {
  const record = readRecord(storage, id);
  if (!record) {
    throw new Error(`orchestration "${id}" not found`);
  }
  return record;
}

function writeRecord(storage: StorageLike, record: PersistedOrchestrationRecord) {
  storage.setItem(storageKey(record.id), JSON.stringify(record));
}

function validateRecord(value: unknown): PersistedOrchestrationRecord {
  if (!value || typeof value !== 'object') {
    throw new Error('invalid orchestration record payload');
  }

  const record = value as PersistedOrchestrationRecord;
  if (
    typeof record.id !== 'string'
    || typeof record.name !== 'string'
    || typeof record.createdAt !== 'number'
    || typeof record.updatedAt !== 'number'
    || !record.draft
  ) {
    throw new Error('invalid orchestration record payload');
  }

  return {
    ...record,
    draft: cloneDraft(record.draft),
  };
}

function toPublicRecord(record: PersistedOrchestrationRecord): OrchestrationRecord {
  return {
    id: record.id,
    name: record.name,
    createdAt: record.createdAt,
    updatedAt: record.updatedAt,
    draft: cloneDraft(record.draft),
  };
}

function workflowStepCount(draft: WorkflowCanvasDraft): number {
  return draft.nodes.filter((node) => node.type !== 'start' && node.type !== 'end').length;
}

function storageKey(id: string): string {
  return `${ORCHESTRATION_ITEM_KEY_PREFIX}:${id}`;
}

function cloneDraft(draft: WorkflowCanvasDraft): WorkflowCanvasDraft {
  return parseJSON(JSON.stringify(draft), 'workflow draft clone') as WorkflowCanvasDraft;
}

function parseJSON(raw: string, label: string): unknown {
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new Error(`failed to parse ${label}`);
  }
}
