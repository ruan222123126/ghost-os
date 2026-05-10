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
  it('puts all sessions into unclassified by default and sorts by recent activity', () => {
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

  it('keeps partition order and sorts partition sessions by recent activity', () => {
    const sessions = sampleSessions();
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');
    store = createSessionPartition(store, 'Personal', 'partition-personal');

    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-3',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });
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
      targetPartitionID: 'partition-personal',
      targetIndex: 0,
    });

    const views = buildSessionPartitionViews({
      sessions,
      store,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(views.map((view) => view.name)).toEqual(['Unclassified', 'Work', 'Personal']);
    expect(findPartition(views, 'Work')?.sessions.map((session) => session.id)).toEqual(['session-1', 'session-3']);
    expect(findPartition(views, 'Personal')?.sessions.map((session) => session.id)).toEqual(['session-2']);
  });

  it('keeps recent-activity priority when drag insertion conflicts with timestamp order', () => {
    const sessions = [
      createSession('session-new', '2026-04-13T00:00:00Z'),
      createSession('session-old', '2026-04-11T00:00:00Z'),
    ];
    let store = createInitialSessionPartitionStore();
    store = createSessionPartition(store, 'Work', 'partition-work');

    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-new',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });
    store = moveSessionToPartition({
      store,
      sessions,
      sessionID: 'session-old',
      targetPartitionID: 'partition-work',
      targetIndex: 0,
    });

    const views = buildSessionPartitionViews({
      sessions,
      store,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(findPartition(views, 'Work')?.sessions.map((session) => session.id)).toEqual(['session-new', 'session-old']);
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

  it('prunes deleted sessions from assignments', () => {
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
  });

  it('falls back to created_at and treats invalid timestamps as oldest', () => {
    const store = createInitialSessionPartitionStore();
    const sessions = [
      createSession('from-created', 'invalid-updated', '2026-04-13T00:00:00Z'),
      createSession('from-updated', '2026-04-12T00:00:00Z', '2026-04-01T00:00:00Z'),
      createSession('older-updated', '2026-04-10T00:00:00Z', '2026-04-15T00:00:00Z'),
      createSession('invalid-both', 'invalid-updated', 'invalid-created'),
    ];

    const views = buildSessionPartitionViews({
      sessions,
      store,
      searchQuery: '',
      unclassifiedName: 'Unclassified',
    });

    expect(views[0].sessions.map((session) => session.id)).toEqual([
      'from-created',
      'from-updated',
      'older-updated',
      'invalid-both',
    ]);
  });
});

function sampleSessions(): SessionMetadata[] {
  return [
    createSession('session-1', '2026-04-13T00:00:00Z'),
    createSession('session-2', '2026-04-12T00:00:00Z'),
    createSession('session-3', '2026-04-11T00:00:00Z'),
  ];
}

function createSession(
  id: string,
  updatedAt = '2026-04-12T00:00:00Z',
  createdAt = '2026-04-12T00:00:00Z',
): SessionMetadata {
  return {
    id,
    title: '',
    created_at: createdAt,
    updated_at: updatedAt,
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
