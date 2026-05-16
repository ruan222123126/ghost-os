import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import {
  deleteSession as deleteSessionRequest,
  listSessions,
} from '@/lib/api/sessions/api';
import type { SessionMetadata } from '@/lib/types';
import { useSessions } from './useSessions';

jest.mock('@/lib/api/sessions/api', () => ({
  deleteSession: jest.fn(),
  listSessions: jest.fn(),
}));

jest.mock('@/lib/i18n/provider', () => ({
  useWebLocale: () => ({
    copy: {
      system: {
        genericRequestFailed: 'request failed',
      },
    },
  }),
}));

const mockedDeleteSession = deleteSessionRequest as jest.MockedFunction<typeof deleteSessionRequest>;
const mockedListSessions = listSessions as jest.MockedFunction<typeof listSessions>;

describe('hooks/useSessions', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.resetAllMocks();
    installBrowserMocks();
  });

  afterEach(async () => {
    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });
    jest.useRealTimers();
    delete (globalThis as { window?: unknown }).window;
    delete (globalThis as { document?: unknown }).document;
  });

  it('loads sessions on mount', async () => {
    mockedListSessions.mockResolvedValue([createSession('session-1')]);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      onRender: (state) => {
        latestState = state;
      },
    });
    const state = requireState(latestState);

    expect(mockedListSessions).toHaveBeenCalledTimes(1);
    expect(state.loading).toBe(false);
    expect(state.sessions.map((session) => session.id)).toEqual(['session-1']);

    await unmountRenderer(renderer);
  });

  it('silently refreshes on interval without toggling loading', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockResolvedValueOnce([createSession('session-2'), createSession('session-1', '2026-05-16T00:01:00Z')]);

    let latestState: HookState | null = null;
    const renders: HookState[] = [];
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: (state) => {
        latestState = state;
        renders.push(snapshotState(state));
      },
    });

    renders.length = 0;
    await advanceBy(1000);
    const state = requireState(latestState);

    expect(mockedListSessions).toHaveBeenCalledTimes(2);
    expect(state.sessions.map((session) => session.id)).toEqual(['session-2', 'session-1']);
    expect(renders).toHaveLength(1);
    expect(renders[0]?.loading).toBe(false);

    await unmountRenderer(renderer);
  });

  it('refreshes on focus and visible visibilitychange', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockResolvedValueOnce([createSession('session-1'), createSession('session-2')])
      .mockResolvedValueOnce([createSession('session-1'), createSession('session-2'), createSession('session-3')]);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: (state) => {
        latestState = state;
      },
    });

    dispatchWindowEvent('focus');
    await flushAsync();
    const focusState = requireState(latestState);
    expect(mockedListSessions).toHaveBeenCalledTimes(2);
    expect(focusState.sessions.map((session) => session.id)).toEqual(['session-1', 'session-2']);

    setDocumentVisibilityState('visible');
    dispatchDocumentEvent('visibilitychange');
    await flushAsync();
    const visibleState = requireState(latestState);
    expect(mockedListSessions).toHaveBeenCalledTimes(3);
    expect(visibleState.sessions.map((session) => session.id)).toEqual([
      'session-1',
      'session-2',
      'session-3',
    ]);

    await unmountRenderer(renderer);
  });

  it('keeps existing sessions when a silent refresh fails', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockRejectedValueOnce(new Error('bridge down'));
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => undefined);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: (state) => {
        latestState = state;
      },
    });

    await advanceBy(1000);
    const state = requireState(latestState);

    expect(state.sessions.map((session) => session.id)).toEqual(['session-1']);
    expect(state.error).toBe('');
    expect(consoleError).toHaveBeenCalledWith('[useSessions] silent refresh failed', expect.any(Error));

    consoleError.mockRestore();
    await unmountRenderer(renderer);
  });

  it('does not re-render when a silent refresh only changes known session metadata', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockResolvedValueOnce([createSession('session-1', '2026-05-16T00:01:00Z')]);

    let renderCount = 0;
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: () => {
        renderCount += 1;
      },
    });

    renderCount = 0;
    await advanceBy(1000);

    expect(mockedListSessions).toHaveBeenCalledTimes(2);
    expect(renderCount).toBe(0);

    await unmountRenderer(renderer);
  });

  it('keeps the existing list when a silent refresh only removes known sessions', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1'), createSession('session-2')])
      .mockResolvedValueOnce([createSession('session-1')]);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: (state) => {
        latestState = state;
      },
    });

    await advanceBy(1000);

    expect(mockedListSessions).toHaveBeenCalledTimes(2);
    expect(requireState(latestState).sessions.map((session) => session.id)).toEqual([
      'session-1',
      'session-2',
    ]);

    await unmountRenderer(renderer);
  });

  it('coalesces overlapping auto refresh requests while one is still running', async () => {
    const pendingRefresh = createDeferred<SessionMetadata[]>();
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockImplementationOnce(() => pendingRefresh.promise)
      .mockResolvedValueOnce([createSession('session-1'), createSession('session-2')]);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      autoRefresh: true,
      refreshIntervalMs: 1000,
      onRender: (state) => {
        latestState = state;
      },
    });

    await advanceBy(1000);
    dispatchWindowEvent('focus');
    setDocumentVisibilityState('visible');
    dispatchDocumentEvent('visibilitychange');
    await flushAsync();

    expect(mockedListSessions).toHaveBeenCalledTimes(2);

    await act(async () => {
      pendingRefresh.resolve([createSession('session-1')]);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockedListSessions).toHaveBeenCalledTimes(3);
    expect(requireState(latestState).sessions.map((session) => session.id)).toEqual(['session-1', 'session-2']);

    await unmountRenderer(renderer);
  });

  it('deletes a session from the local collection', async () => {
    mockedListSessions.mockResolvedValue([createSession('session-1'), createSession('session-2')]);
    mockedDeleteSession.mockResolvedValue(undefined);

    let latestState: HookState | null = null;
    const renderer = await renderHook({
      onRender: (state) => {
        latestState = state;
      },
    });
    const state = requireState(latestState);

    await act(async () => {
      await state.deleteSession('session-1');
    });

    expect(requireState(latestState).sessions.map((session) => session.id)).toEqual(['session-2']);

    await unmountRenderer(renderer);
  });
});

function HookProbe(props: HookProbeProps) {
  const state = useSessions({
    autoRefresh: props.autoRefresh,
    refreshIntervalMs: props.refreshIntervalMs,
  });
  props.onRender(state);
  return null;
}

interface HookProbeProps {
  autoRefresh?: boolean;
  refreshIntervalMs?: number;
  onRender: (state: HookState) => void;
}

interface HookState {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  loadSessions: (options?: {
    silent?: boolean;
    commitMode?: 'always' | 'when-new-session';
  }) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  createNewSession: () => void;
  setCurrentSessionId: (id: string) => void;
}

async function renderHook(props: HookProbeProps) {
  let renderer: TestRenderer.ReactTestRenderer;
  await act(async () => {
    renderer = TestRenderer.create(React.createElement(HookProbe, props));
    await Promise.resolve();
    await Promise.resolve();
  });
  return renderer!;
}

async function unmountRenderer(renderer: TestRenderer.ReactTestRenderer) {
  await act(async () => {
    renderer.unmount();
    await Promise.resolve();
  });
}

function snapshotState(state: HookState): HookState {
  return {
    ...state,
    sessions: [...state.sessions],
  };
}

function createSession(id: string, updatedAt = '2026-05-16T00:00:00Z'): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-05-16T00:00:00Z',
    updated_at: updatedAt,
    message_count: 1,
    token_count: 1,
  };
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

function requireState(state: HookState | null): HookState {
  if (!state) {
    throw new Error('hook state missing');
  }
  return state;
}

async function advanceBy(milliseconds: number) {
  await act(async () => {
    jest.advanceTimersByTime(milliseconds);
    await Promise.resolve();
    await Promise.resolve();
  });
}

async function flushAsync() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

function installBrowserMocks() {
  const windowTarget = createEventTargetMock();
  const documentTarget = createEventTargetMock();
  const documentMock = {
    ...documentTarget,
    documentElement: { lang: 'en' },
    visibilityState: 'visible' as 'visible' | 'hidden',
  };
  const windowMock = {
    ...windowTarget,
    clearInterval,
    setInterval,
  };

  (globalThis as { window?: unknown }).window = windowMock;
  (globalThis as { document?: unknown }).document = documentMock;
}

function setDocumentVisibilityState(value: 'visible' | 'hidden') {
  const documentMock = globalThis.document as unknown as BrowserDocumentMock;
  documentMock.visibilityState = value;
}

function dispatchWindowEvent(type: string) {
  const windowMock = globalThis.window as unknown as BrowserEventTargetMock;
  windowMock.dispatchEvent({ type } as Event);
}

function dispatchDocumentEvent(type: string) {
  const documentMock = globalThis.document as unknown as BrowserEventTargetMock;
  documentMock.dispatchEvent({ type } as Event);
}

function createEventTargetMock(): BrowserEventTargetMock {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    addEventListener: jest.fn((type: string, listener: EventListener) => {
      const current = listeners.get(type) ?? new Set<EventListener>();
      current.add(listener);
      listeners.set(type, current);
    }),
    removeEventListener: jest.fn((type: string, listener: EventListener) => {
      listeners.get(type)?.delete(listener);
    }),
    dispatchEvent: jest.fn((event: Event) => {
      for (const listener of listeners.get(event.type) ?? []) {
        listener(event);
      }
      return true;
    }),
  };
}

interface BrowserEventTargetMock {
  addEventListener: jest.Mock<void, [string, EventListener]>;
  removeEventListener: jest.Mock<void, [string, EventListener]>;
  dispatchEvent: jest.Mock<boolean, [Event]>;
}

interface BrowserDocumentMock extends BrowserEventTargetMock {
  documentElement: { lang: string };
  visibilityState: 'visible' | 'hidden';
}
