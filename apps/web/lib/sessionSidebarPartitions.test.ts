import {
  buildSessionPartitionViews,
  createInitialSessionPartitionStore,
  createSessionPartition,
  moveSessionToPartition,
  sanitizeSessionPartitionStore,
  validatePartitionName,
} from './sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

describe('lib/sessionSidebarPartitions', () => {
  it('puts all sessions into unclassified by default', () => {
    const store = createInitialSessionPartitionStore();
    const views = buildSessionPartitionViews({
      sessions: sampleSessions(),
      store,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(views).toHaveLength(1);
    expect(views[0].name).toBe('Unclassified');
    expect(views[0].sessions.map((session) => session.id)).toEqual(['session-1', 'session-2', 'session-3']);
  });

  it('validates partition names with non-empty and no duplicates', () => {
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');

    expect(validatePartitionName('', store, 'Unclassified')).toBe('empty');
    expect(validatePartitionName('  work  ', store, 'Unclassified')).toBe('duplicate');
    expect(validatePartitionName(' unclassified ', store, 'Unclassified')).toBe('duplicate');
    expect(validatePartitionName('Personal', store, 'Unclassified')).toBeUndefined();
  });

  it('moves sessions with insertion index across partitions', () => {
    const sessions = sampleSessions();
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');
    store = createSessionPartition(store, 'Personal', 'partition-personal');

    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-1',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-2',
      targetPartitionID: 'partition-work',
      targetIndex: 1,
    });
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-3',
      targetPartitionID: 'partition-personal',
      targetIndex: 0,
    });
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-2',
      targetPartitionID: 'partition-personal',
      targetIndex: 1,
    });

    const views = buildSessionPartitionViews({
      sessions,
      store,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(findPartition(views, 'Work')?.sessions.map((session) => session.id)).toEqual(['session-1']);
    expect(findPartition(views, 'Personal')?.sessions.map((session) => session.id)).toEqual(['session-3', 'session-2']);
  });

  it('keeps partitions but hides empty groups during search', () => {
    const sessions = sampleSessions();
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');

    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-2',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });

    const views = buildSessionPartitionViews({
      sessions,
      store,
      searchQuery: 'session-2',
      unclassifiedName: 'Unclassified',
    });

    expect(views).toHaveLength(1);
    expect(views[0].name).toBe('Work');
    expect(views[0].sessions.map((session) => session.id)).toEqual(['session-2']);
  });

  it('prunes deleted sessions from assignments and order records', () => {
    const sessions = sampleSessions();
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-2',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });

    const sanitized = sanitizeSessionPartitionStore(store, sessions.slice(0, 2));

    expect(sanitized.assignments).toEqual({ 'session-2': 'partition-work' });
    expect(sanitized.orders['partition-work']).toEqual(['session-2']);
    expect(sanitized.orders['__unclassified__']).toEqual(['session-1']);
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

function findPartition(
  views: ReturnType<typeof buildSessionPartitionViews>,
  name: string,
) {
  return views.find((view) => view.name === name);
}
