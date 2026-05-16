import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { listOrchestrations, listOrchestrationLogs } from '@/lib/api/orchestrations/api';
import { listTaskLogs, listTasks } from '@/lib/api/tasks/api';
import type { SessionMetadata, TaskPayload, TaskRunLog } from '@/lib/types';
import { useSessionSidebarSessionSources } from './useSessionSidebarSessionSources';

jest.mock('@/lib/api/tasks/api', () => ({
  listTaskLogs: jest.fn(),
  listTasks: jest.fn(),
}));

jest.mock('@/lib/api/orchestrations/api', () => ({
  listOrchestrationLogs: jest.fn(),
  listOrchestrations: jest.fn(),
}));

const mockedListTaskLogs = listTaskLogs as jest.MockedFunction<typeof listTaskLogs>;
const mockedListTasks = listTasks as jest.MockedFunction<typeof listTasks>;
const mockedListOrchestrationLogs = listOrchestrationLogs as jest.MockedFunction<typeof listOrchestrationLogs>;
const mockedListOrchestrations = listOrchestrations as jest.MockedFunction<typeof listOrchestrations>;

describe('hooks/useSessionSidebarSessionSources', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.resetAllMocks();
    installBrowserMocks();
    mockedListOrchestrations.mockResolvedValue([]);
    mockedListTaskLogs.mockResolvedValue([runLog({ session_id_output: 'loop-session' })]);
    mockedListTasks.mockResolvedValue([loopTask('loop-task')]);
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

  it('keeps polling and resolves known source assignments for visible sessions', async () => {
    let latestState: HookState | null = null;

    const renderer = await renderHook({
      enabled: true,
      refreshIntervalMs: 1000,
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
    expect(mockedListTasks).toHaveBeenCalledTimes(1);
    expect(mockedListTaskLogs).toHaveBeenCalledTimes(1);

    await advanceBy(1000);

    expect(mockedListTasks).toHaveBeenCalledTimes(2);
    expect(mockedListTaskLogs).toHaveBeenCalledTimes(2);

    await unmountRenderer(renderer);
  });

  it('filters out run sessions that are not present in the loaded sidebar list', async () => {
    let latestState: HookState | null = null;

    const renderer = await renderHook({
      enabled: true,
      refreshIntervalMs: 1000,
      sessions: [],
      requestFailedText: 'request failed',
      onRender: (state) => {
        latestState = state;
      },
    });

    expect(requireState(latestState)).toMatchObject({
      assignments: {},
      hiddenSessionIDs: [],
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
  refreshIntervalMs?: number;
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

function loopTask(id: string): TaskPayload {
  return {
    id,
    task_kind: 'agent_message',
    message: 'Loop task',
    agent_mode: 'relay',
    relay: {
      stop_policy: 'ai_decides',
      max_rounds: 3,
      execution_timeout_ms: 0,
    },
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: true,
    created_at: '2026-05-16T00:00:00Z',
    updated_at: '2026-05-16T00:00:00Z',
  };
}

function runLog(patch: Partial<TaskRunLog>): TaskRunLog {
  return {
    task_id: 'loop-task',
    run_id: 'run-1',
    trace_id: 'trace-1',
    task_kind: 'agent_message',
    scheduled_at: '2026-05-16T00:00:00Z',
    status: 'success',
    ...patch,
  };
}

function installBrowserMocks() {
  const windowTarget = createEventTargetMock();
  const documentTarget = createEventTargetMock();
  const documentMock = {
    ...documentTarget,
    documentElement: { lang: 'zh-CN' },
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
