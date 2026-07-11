import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import {
  getSessionSidebarPartitions,
  putSessionSidebarPartitions,
} from '@/lib/api/sessions/api';
import {
  UNCLASSIFIED_PARTITION_ID,
  createInitialSessionPartitionStore,
  stringifySessionPartitionStore,
} from '@/lib/sessionSidebarPartitions';
import type {
  SessionMetadata,
  SessionSidebarPartitionState,
} from '@/lib/types';
import { useSessionSidebarPartitions } from './useSessionSidebarPartitions';

jest.mock('@/lib/api/sessions/api', () => ({
  getSessionSidebarPartitions: jest.fn(),
  putSessionSidebarPartitions: jest.fn(),
}));

const mockedGetSessionSidebarPartitions = getSessionSidebarPartitions as jest.MockedFunction<
  typeof getSessionSidebarPartitions
>;
const mockedPutSessionSidebarPartitions = putSessionSidebarPartitions as jest.MockedFunction<
  typeof putSessionSidebarPartitions
>;

describe('hooks/useSessionSidebarPartitions', () => {
  beforeEach(() => {
    jest.resetAllMocks();
    installWindow(buildLocalStorageMock());
  });

  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
  });

  it('hydrates partition views from the backend state', async () => {
    mockedGetSessionSidebarPartitions.mockResolvedValue({
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-2': 'work' },
    });

    let latestState: HookState | null = null;
    await renderHook((state) => {
      latestState = state;
    });

    expect(mockedGetSessionSidebarPartitions).toHaveBeenCalledTimes(1);
    expect(findPartitionSessions(latestState!, 'Work')).toEqual(['session-2']);
    expect(findPartitionSessions(latestState!, 'Unclassified')).toEqual(['session-1']);
  });

  it('keeps legacy localStorage pending until the user explicitly migrates it', async () => {
    const legacyState = {
      ...createInitialSessionPartitionStore(),
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    };
    const localStorageMock = buildLocalStorageMock({
      'ghost.web.session.partitions.v1': stringifySessionPartitionStore(legacyState),
    });
    installWindow(localStorageMock);

    mockedGetSessionSidebarPartitions.mockResolvedValue(createInitialSessionPartitionStore());
    mockedPutSessionSidebarPartitions.mockResolvedValue(legacyState);

    let latestState: HookState | null = null;
    await renderHook((state) => {
      latestState = state;
    });

    expect(mockedPutSessionSidebarPartitions).not.toHaveBeenCalled();
    expect(latestState!.legacyMigrationAvailable).toBe(true);

    await act(async () => {
      await latestState!.runLegacyMigration();
    });

    expect(mockedPutSessionSidebarPartitions).toHaveBeenCalledTimes(1);
    expect(mockedPutSessionSidebarPartitions).toHaveBeenCalledWith(
      expect.objectContaining({
        partitions: [{ id: 'work', name: 'Work' }],
        assignments: { 'session-1': 'work' },
      }),
      expect.stringMatching(/^session-partitions-migrate-/),
    );
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('ghost.web.session.partitions.v1');
    expect(findPartitionSessions(latestState!, 'Work')).toEqual(['session-1']);
    expect(latestState!.legacyMigrationAvailable).toBe(false);
  });

  it('allows discarding legacy localStorage explicitly', async () => {
    const legacyState = {
      ...createInitialSessionPartitionStore(),
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    };
    const localStorageMock = buildLocalStorageMock({
      'ghost.web.session.partitions.v1': stringifySessionPartitionStore(legacyState),
    });
    installWindow(localStorageMock);

    mockedGetSessionSidebarPartitions.mockResolvedValue(createInitialSessionPartitionStore());

    let latestState: HookState | null = null;
    await renderHook((state) => {
      latestState = state;
    });

    act(() => {
      latestState!.discardLegacyMigration();
    });

    expect(localStorageMock.removeItem).toHaveBeenCalledWith('ghost.web.session.partitions.v1');
    expect(latestState!.legacyMigrationAvailable).toBe(false);
  });

  it('keeps only the last queued drag state when earlier save responses finish later', async () => {
    const initialState: SessionSidebarPartitionState = {
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: {},
    };
    const firstSave = createDeferred<SessionSidebarPartitionState>();
    const secondSave = createDeferred<SessionSidebarPartitionState>();
    mockedGetSessionSidebarPartitions.mockResolvedValue(initialState);
    mockedPutSessionSidebarPartitions
      .mockReturnValueOnce(firstSave.promise)
      .mockReturnValueOnce(secondSave.promise);

    let latestState: HookState | null = null;
    await renderHook((state) => {
      latestState = state;
    });

    act(() => {
      latestState!.moveSession('session-1', 'work', 0);
    });
    expect(mockedPutSessionSidebarPartitions).toHaveBeenCalledTimes(1);
    expect(findPartitionSessions(latestState!, 'Work')).toEqual(['session-1']);

    act(() => {
      latestState!.moveSession('session-1', UNCLASSIFIED_PARTITION_ID, 0);
    });
    expect(mockedPutSessionSidebarPartitions).toHaveBeenCalledTimes(1);
    expect(findPartitionSessions(latestState!, 'Work')).toEqual([]);

    await act(async () => {
      firstSave.resolve({
        version: 1,
        partitions: [{ id: 'work', name: 'Work' }],
        assignments: { 'session-1': 'work' },
      });
      await firstSave.promise;
      await Promise.resolve();
    });

    expect(mockedPutSessionSidebarPartitions).toHaveBeenCalledTimes(2);
    expect(mockedPutSessionSidebarPartitions.mock.calls[1][0]).toEqual({
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: {},
    });
    expect(findPartitionSessions(latestState!, 'Work')).toEqual([]);

    await act(async () => {
      secondSave.resolve(initialState);
      await secondSave.promise;
      await Promise.resolve();
    });

    expect(findPartitionSessions(latestState!, 'Work')).toEqual([]);
    expect(findPartitionSessions(latestState!, 'Unclassified')).toEqual(['session-1', 'session-2']);
    expect(latestState!.partitionError).toBe('');
  });

  it('rolls back optimistic changes and exposes the save error', async () => {
    mockedGetSessionSidebarPartitions.mockResolvedValue({
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: {},
    });
    mockedPutSessionSidebarPartitions.mockRejectedValue(new Error('save denied'));

    let latestState: HookState | null = null;
    await renderHook((state) => {
      latestState = state;
    });

    act(() => {
      latestState!.moveSession('session-1', 'work', 0);
    });
    expect(findPartitionSessions(latestState!, 'Work')).toEqual(['session-1']);

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(findPartitionSessions(latestState!, 'Work')).toEqual([]);
    expect(findPartitionSessions(latestState!, 'Unclassified')).toEqual(['session-1', 'session-2']);
    expect(latestState!.partitionError).toBe('save denied');
  });
});

function HookProbe(props: { onRender: (state: HookState) => void }) {
  const state = useSessionSidebarPartitions({
    sessions: sampleSessions(),
    sessionsLoaded: true,
    searchQuery: '',
    unclassifiedName: 'Unclassified',
    requestFailedText: 'request failed',
  });
  props.onRender(state);
  return null;
}

async function renderHook(onRender: (state: HookState) => void) {
  await act(async () => {
    TestRenderer.create(React.createElement(HookProbe, { onRender }));
    await Promise.resolve();
    await Promise.resolve();
  });
}

interface HookState {
  partitionViews: Array<{
    id: string;
    name: string;
    sessions: SessionMetadata[];
  }>;
  partitionError: string;
  legacyMigrationAvailable: boolean;
  legacyMigrationRunning: boolean;
  addPartition: (name: string) => { ok: boolean };
  renamePartition: (partitionID: string, name: string) => { ok: boolean };
  deletePartition: (partitionID: string) => { ok: boolean };
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
  runLegacyMigration: () => Promise<void>;
  discardLegacyMigration: () => void;
}

function sampleSessions(): SessionMetadata[] {
  return [
    createSession('session-1', '2026-04-12T00:00:00Z'),
    createSession('session-2', '2026-04-11T00:00:00Z'),
  ];
}

function createSession(id: string, updatedAt: string): SessionMetadata {
  return {
    id,
    title: '',
    created_at: updatedAt,
    updated_at: updatedAt,
    message_count: 0,
    token_count: 0,
  };
}

function findPartitionSessions(state: HookState, name: string): string[] {
  return state.partitionViews.find((partition) => partition.name === name)?.sessions.map((session) => session.id) ?? [];
}

function installWindow(localStorage: LocalStorageMock) {
  (globalThis as { window?: unknown }).window = {
    localStorage,
  };
}

type LocalStorageMock = Storage & {
  removeItem: jest.Mock<void, [string]>;
};

function buildLocalStorageMock(
  initial: Record<string, string> = {},
): LocalStorageMock {
  const store = { ...initial };
  return {
    getItem: jest.fn((key: string) => store[key] ?? null),
    setItem: jest.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: jest.fn((key: string) => {
      delete store[key];
    }),
    clear: jest.fn(() => {
      Object.keys(store).forEach((key) => {
        delete store[key];
      });
    }),
    key: jest.fn(() => null),
    get length() {
      return Object.keys(store).length;
    },
  };
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;

  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}
