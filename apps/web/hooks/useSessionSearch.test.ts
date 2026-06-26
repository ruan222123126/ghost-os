import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { searchSessions } from '@/lib/api/sessions/api';
import type { SessionMetadata } from '@/lib/types';
import { useSessionSearch } from './useSessionSearch';

jest.mock('@/lib/api/sessions/api', () => ({
  searchSessions: jest.fn(),
}));

const mockedSearchSessions = searchSessions as jest.MockedFunction<typeof searchSessions>;

describe('hooks/useSessionSearch', () => {
  beforeEach(() => {
    jest.resetAllMocks();
    jest.useFakeTimers();
    installWindowTimers();
  });

  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
    jest.useRealTimers();
  });

  it('loads debounced remote search results', async () => {
    const session = buildSession('session-1');
    mockedSearchSessions.mockResolvedValue([session]);

    let latestState: HookState | null = null;
    await act(async () => {
      TestRenderer.create(React.createElement(HookProbe, {
        open: true,
        query: 'hello',
        onRender: (state) => {
          latestState = state;
        },
      }));
      await Promise.resolve();
    });

    expect(latestState!.loading).toBe(true);
    expect(mockedSearchSessions).not.toHaveBeenCalled();

    await act(async () => {
      jest.advanceTimersByTime(150);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockedSearchSessions).toHaveBeenCalledWith('hello');
    expect(latestState).toEqual({
      results: [session],
      loading: false,
      error: '',
    });
  });

  it('cancels pending search when closed', async () => {
    let latestState: HookState | null = null;
    let renderer!: TestRenderer.ReactTestRenderer;
    await act(async () => {
      renderer = TestRenderer.create(React.createElement(HookProbe, {
        open: true,
        query: 'hello',
        onRender: (state) => {
          latestState = state;
        },
      }));
      await Promise.resolve();
    });
    expect(latestState!.loading).toBe(true);

    await act(async () => {
      renderer.update(React.createElement(HookProbe, {
        open: false,
        query: 'hello',
        onRender: (state) => {
          latestState = state;
        },
      }));
      await Promise.resolve();
    });
    await act(async () => {
      jest.advanceTimersByTime(150);
      await Promise.resolve();
    });

    expect(mockedSearchSessions).not.toHaveBeenCalled();
    expect(latestState).toEqual({
      results: [],
      loading: false,
      error: '',
    });
  });
});

function HookProbe(props: {
  open: boolean;
  query: string;
  onRender: (state: HookState) => void;
}) {
  const state = useSessionSearch({
    open: props.open,
    query: props.query,
    fallbackMessage: 'request failed',
  });
  props.onRender(state);
  return null;
}

type HookState = ReturnType<typeof useSessionSearch>;

function buildSession(id: string): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z',
    message_count: 0,
    token_count: 0,
  };
}

function installWindowTimers(): void {
  (globalThis as { window?: unknown }).window = {
    setTimeout: globalThis.setTimeout,
    clearTimeout: globalThis.clearTimeout,
  };
}
