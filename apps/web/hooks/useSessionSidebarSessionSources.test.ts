import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getSessionSources } from '@/lib/api/sessions/api';
import type { SessionMetadata, SessionSourceResolution } from '@/lib/types';
import { useSessionSidebarSessionSources } from './useSessionSidebarSessionSources';

jest.mock('@/lib/api/sessions/api', () => ({
  getSessionSources: jest.fn(),
}));

const mockedGetSessionSources = getSessionSources as jest.MockedFunction<typeof getSessionSources>;

describe('hooks/useSessionSidebarSessionSources', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.resetAllMocks();
    mockedGetSessionSources.mockResolvedValue(sessionSources({
      assignments: {
        'loop-session': {
          kind: 'loop',
          owner_id: 'loop-task',
          owner_name: 'Loop task',
        },
      },
      hidden_session_ids: [],
    }));
  });

  afterEach(async () => {
    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });
    jest.useRealTimers();
  });

  it('loads source assignments through the aggregated session sources endpoint', async () => {
    let latestState: HookState | null = null;

    const renderer = await renderHook({
      enabled: true,
      sessions: [session('loop-session')],
      requestFailedText: 'request failed',
      onRender: (state) => {
        latestState = state;
      },
    });

    expect(requireState(latestState)).toMatchObject({
      assignments: {
        'loop-session': {
          kind: 'loop',
          ownerID: 'loop-task',
          ownerName: 'Loop task',
        },
      },
      hiddenSessionIDs: [],
      error: '',
    });
    expect(mockedGetSessionSources).toHaveBeenCalledTimes(1);

    await advanceBy(1000);

    expect(mockedGetSessionSources).toHaveBeenCalledTimes(1);

    await unmountRenderer(renderer);
  });

  it('filters source assignments and hidden ids to the loaded sidebar list', async () => {
    mockedGetSessionSources.mockResolvedValue(sessionSources({
      assignments: {
        'visible-session': {
          kind: 'workflow',
          owner_id: 'workflow-task',
          owner_name: 'Workflow task',
        },
        'missing-session': {
          kind: 'task',
          owner_id: 'task-1',
          owner_name: 'Task 1',
        },
      },
      hidden_session_ids: ['visible-session', 'missing-session'],
    }));
    let latestState: HookState | null = null;

    const renderer = await renderHook({
      enabled: true,
      sessions: [session('visible-session')],
      requestFailedText: 'request failed',
      onRender: (state) => {
        latestState = state;
      },
    });

    expect(requireState(latestState)).toMatchObject({
      assignments: {
        'visible-session': {
          kind: 'workflow',
          ownerID: 'workflow-task',
          ownerName: 'Workflow task',
        },
      },
      hiddenSessionIDs: ['visible-session'],
      error: '',
    });

    await unmountRenderer(renderer);
  });
});

function HookProbe(props: HookProbeProps) {
  const state = useSessionSidebarSessionSources(props);
  props.onRender(state);
  return null;
}

interface HookProbeProps {
  enabled: boolean;
  sessions: SessionMetadata[];
  requestFailedText: string;
  onRender: (state: HookState) => void;
}

interface HookState {
  assignments: Record<string, { kind: string; ownerID: string; ownerName: string }>;
  hiddenSessionIDs: string[];
  error: string;
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

function requireState(state: HookState | null): HookState {
  if (!state) {
    throw new Error('hook state missing');
  }
  return state;
}

async function unmountRenderer(renderer: TestRenderer.ReactTestRenderer) {
  await act(async () => {
    renderer.unmount();
    await Promise.resolve();
  });
}

async function advanceBy(milliseconds: number) {
  await act(async () => {
    jest.advanceTimersByTime(milliseconds);
    await Promise.resolve();
    await Promise.resolve();
  });
}

function session(id: string): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-05-16T00:00:00Z',
    updated_at: '2026-05-16T00:00:00Z',
    message_count: 1,
    token_count: 1,
  };
}

function sessionSources(resolution: SessionSourceResolution): SessionSourceResolution {
  return resolution;
}
