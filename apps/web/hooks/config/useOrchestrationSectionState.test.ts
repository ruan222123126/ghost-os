import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import {
  createOrchestration,
  deleteOrchestration,
  listOrchestrations,
  runOrchestrationNow,
  updateOrchestration,
} from '@/lib/api/orchestrations/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { migrateLegacyOrchestrations } from '@/lib/orchestration-editor/legacyMigration';
import type { OrchestrationTaskPayload } from '@/lib/types';
import { useOrchestrationSectionState } from './useOrchestrationSectionState';

jest.mock('@/lib/api/orchestrations/api', () => ({
  createOrchestration: jest.fn(),
  deleteOrchestration: jest.fn(),
  listOrchestrations: jest.fn(),
  runOrchestrationNow: jest.fn(),
  updateOrchestration: jest.fn(),
}));

jest.mock('@/lib/orchestration-editor/legacyMigration', () => ({
  migrateLegacyOrchestrations: jest.fn(),
}));

const mockedCreateOrchestration = createOrchestration as jest.MockedFunction<typeof createOrchestration>;
const mockedDeleteOrchestration = deleteOrchestration as jest.MockedFunction<typeof deleteOrchestration>;
const mockedListOrchestrations = listOrchestrations as jest.MockedFunction<typeof listOrchestrations>;
const mockedRunOrchestrationNow = runOrchestrationNow as jest.MockedFunction<typeof runOrchestrationNow>;
const mockedUpdateOrchestration = updateOrchestration as jest.MockedFunction<typeof updateOrchestration>;
const mockedMigrateLegacyOrchestrations = migrateLegacyOrchestrations as jest.MockedFunction<typeof migrateLegacyOrchestrations>;

describe('hooks/config/useOrchestrationSectionState', () => {
  const sampleTask = buildTask();

  beforeEach(() => {
    jest.resetAllMocks();
    mockedCreateOrchestration.mockResolvedValue(sampleTask);
    mockedDeleteOrchestration.mockResolvedValue(undefined);
    mockedListOrchestrations.mockResolvedValue([sampleTask]);
    mockedMigrateLegacyOrchestrations.mockResolvedValue(false);
    mockedRunOrchestrationNow.mockResolvedValue(undefined);
  });

  it('migrates legacy drafts, loads orchestrations, and updates enabled state', async () => {
    mockedUpdateOrchestration.mockResolvedValue({ ...sampleTask, enabled: false });
    const latest = await renderHookMachine();

    expect(mockedMigrateLegacyOrchestrations).toHaveBeenCalledTimes(1);
    expect(mockedListOrchestrations).toHaveBeenCalledTimes(1);
    expect(latest.current.state.loading).toBe(false);
    expect(latest.current.state.orchestrations).toEqual([sampleTask]);

    await act(async () => {
      await latest.current.actions.setEnabledByID('orch_1', false);
    });

    expect(mockedUpdateOrchestration).toHaveBeenCalledWith('orch_1', { enabled: false });
    expect(latest.current.state.orchestrations).toEqual([{ ...sampleTask, enabled: false }]);
    expect(latest.current.state.error).toBe('');
  });

  it('removes orchestration after successful delete', async () => {
    const latest = await renderHookMachine();

    await act(async () => {
      await latest.current.actions.deleteByID('orch_1');
    });

    expect(mockedDeleteOrchestration).toHaveBeenCalledWith('orch_1');
    expect(latest.current.state.orchestrations).toEqual([]);
    expect(latest.current.state.error).toBe('');
  });

  it('tracks running orchestration id while run is in flight', async () => {
    let resolveRun: (() => void) | undefined;
    mockedRunOrchestrationNow.mockImplementation(() => new Promise<void>((resolve) => {
      resolveRun = resolve;
    }));
    const latest = await renderHookMachine();

    let runPromise: Promise<void> | undefined;
    await act(async () => {
      runPromise = latest.current.actions.runByID('orch_1');
      await flushPromises();
    });

    expect(latest.current.state.runningOrchestrationID).toBe('orch_1');

    await act(async () => {
      resolveRun?.();
      await runPromise;
      await flushPromises();
    });

    expect(latest.current.state.runningOrchestrationID).toBe('');
  });

  it('surfaces success and refreshes after launch request succeeds', async () => {
    const latest = await renderHookMachine();

    await act(async () => {
      await latest.current.actions.runByID('orch_1');
    });

    expect(mockedRunOrchestrationNow).toHaveBeenCalledWith('orch_1');
    expect(latest.current.state.success).toBe('Run started successfully');
    expect(latest.current.state.error).toBe('');
    expect(mockedListOrchestrations).toHaveBeenCalledTimes(2);
  });

  it('creates a named orchestration and returns to list view', async () => {
    const latest = await renderHookMachine();

    await act(async () => {
      latest.current.actions.startCreate();
      latest.current.actions.setName('Daily');
    });
    await act(async () => {
      await latest.current.actions.submitCreate();
    });

    expect(mockedCreateOrchestration).toHaveBeenCalledTimes(1);
    expect(latest.current.state.view).toBe('list');
    expect(latest.current.state.name).toBe('');
  });
});

function HookProbe(props: {
  onRender: (machine: HookMachine) => void;
}) {
  const machine = useOrchestrationSectionState();
  props.onRender(machine);
  return null;
}

async function renderHookMachine() {
  const latest: { current: HookMachine } = {
    current: null as unknown as HookMachine,
  };

  await act(async () => {
    TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(HookProbe, {
          onRender: (machine) => {
            latest.current = machine;
          },
        }),
      }),
    );
    await flushPromises();
  });

  return latest;
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

function buildTask(overrides?: Partial<OrchestrationTaskPayload>): OrchestrationTaskPayload {
  return {
    id: 'orch_1',
    name: '日报编排',
    task_kind: 'orchestration',
    orchestration: {
      nodes: [
        { id: 'group-1', type: 'group', group: { title: '群组 1', shared_context: '', speaking_mode: 'sequential', max_rounds: 1 } },
        { id: 'agent-1', type: 'agent', agent: { title: '角色 1', message: 'member' } },
      ],
      edges: [],
    },
    schedule_type: 'interval',
    interval_seconds: 60,
    enabled: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

type HookMachine = ReturnType<typeof useOrchestrationSectionState>;
