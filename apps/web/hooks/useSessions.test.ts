import { act } from 'react-test-renderer';
import {
  deleteSession as deleteSessionRequest,
  listSessions,
} from '@/lib/api/sessions/api';
import type { SessionMetadata } from '@/lib/types';
import {
  advanceBy,
  createDeferred,
  createSession,
  dispatchDocumentEvent,
  dispatchWindowEvent,
  flushAsync,
  installBrowserMocks,
  renderHook,
  requireState,
  setDocumentVisibilityState,
  snapshotState,
  unmountRenderer,
  type HookState,
} from './useSessions.testHelpers';

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

  it('does not silently refresh on elapsed time', async () => {
    mockedListSessions
      .mockResolvedValueOnce([createSession('session-1')])
      .mockResolvedValueOnce([createSession('session-2'), createSession('session-1', '2026-05-16T00:01:00Z')]);

    let latestState: HookState | null = null;
    const renders: HookState[] = [];
    const renderer = await renderHook({
      autoRefresh: true,
      onRender: (state) => {
        latestState = state;
        renders.push(snapshotState(state));
      },
    });

    renders.length = 0;
    await advanceBy(1000);
    const state = requireState(latestState);

    expect(mockedListSessions).toHaveBeenCalledTimes(1);
    expect(state.sessions.map((session) => session.id)).toEqual(['session-1']);
    expect(renders).toHaveLength(0);

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
      onRender: (state) => {
        latestState = state;
      },
    });

    dispatchWindowEvent('focus');
    await flushAsync();
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
      onRender: () => {
        renderCount += 1;
      },
    });

    renderCount = 0;
    dispatchWindowEvent('focus');
    await flushAsync();

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
      onRender: (state) => {
        latestState = state;
      },
    });

    dispatchWindowEvent('focus');
    await flushAsync();

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
      onRender: (state) => {
        latestState = state;
      },
    });

    dispatchWindowEvent('focus');
    await flushAsync();
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
