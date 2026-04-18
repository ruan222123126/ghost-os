import {
  buildSessionPartitionViews,
  createInitialSessionPartitionStore,
  createSessionPartition,
  moveSessionToPartition,
} from './sessionSidebarPartitions';
import {
  deleteSessionPartition,
  renameSessionPartition,
  validatePartitionRenameName,
} from './sessionSidebarPartitionMutations';
import type { SessionMetadata } from '@/lib/types';

describe('lib/sessionSidebarPartitionMutations', () => {
  it('validates rename input and allows keeping the same name', () => {
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');

    expect(validatePartitionRenameName({
      value: '',
      store,
      partitionID: 'partition-work',
      unclassifiedName: 'Unclassified',
    })).toBe('empty');
    expect(validatePartitionRenameName({
      value: ' Unclassified ',
      store,
      partitionID: 'partition-work',
      unclassifiedName: 'Unclassified',
    })).toBe('duplicate');
    expect(validatePartitionRenameName({
      value: ' work ',
      store,
      partitionID: 'partition-work',
      unclassifiedName: 'Unclassified',
    })).toBeUndefined();
  });

  it('renames target partition only', () => {
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');
    store = createSessionPartition(store, 'Personal', 'partition-personal');

    const renamed = renameSessionPartition(store, 'partition-work', 'Focus');

    expect(renamed.partitions).toEqual([
      { id: 'partition-work', name: 'Focus' },
      { id: 'partition-personal', name: 'Personal' },
    ]);
  });

  it('deletes partition and moves assigned sessions back to unclassified', () => {
    const sessions = sampleSessions();
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');
    store = createSessionPartition(store, 'Personal', 'partition-personal');
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-2',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-3',
      targetPartitionID: 'partition-personal',
      targetIndex: 0,
    });

    const next = deleteSessionPartition({
      store,
      sessions,
      partitionID: 'partition-work',
    });
    const views = buildSessionPartitionViews({
      sessions,
      store: next,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(next.partitions.map((partition) => partition.id)).toEqual(['partition-personal']);
    expect(next.assignments).toEqual({ 'session-3': 'partition-personal' });
    expect(views[0].name).toBe('Unclassified');
    expect(views[0].sessions.map((session) => session.id)).toContain('session-2');
  });
});

function sampleSessions(): SessionMetadata[] {
  return [
    createSession('session-1'),
    createSession('session-2'),
    createSession('session-3'),
  ];
}

function createSession(id: string): SessionMetadata {
  return {
    id,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z',
    message_count: 0,
    token_count: 0,
  };
}
