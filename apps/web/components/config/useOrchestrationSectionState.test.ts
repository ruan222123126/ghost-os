import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import {
  createOrchestration,
  deleteOrchestration,
  listOrchestrations,
  runOrchestrationNow,
  updateOrchestration,
} from '@/lib/api/orchestrations/api';
import { WebLocaleProvider, useWebLocale } from '@/lib/i18n/provider';
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

describe('components/config/useOrchestrationSectionState', () => {
  const sampleTask: OrchestrationTaskPayload = {
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
  };

  beforeEach(() => {
    jest.resetAllMocks();
    mockedMigrateLegacyOrchestrations.mockResolvedValue(false);
    mockedCreateOrchestration.mockResolvedValue(sampleTask);
    mockedDeleteOrchestration.mockResolvedValue(undefined);
    mockedRunOrchestrationNow.mockResolvedValue(undefined);
  });

  it('loads orchestrations on mount and updates enabled state in place', async () => {
    mockedListOrchestrations.mockResolvedValue([sampleTask]);
    mockedUpdateOrchestration.mockResolvedValue({ ...sampleTask, enabled: false });

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              onRender: (state) => {
                latestState = state;
              },
            }),
          },
        ),
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockedMigrateLegacyOrchestrations).toHaveBeenCalledTimes(1);
    expect(mockedListOrchestrations).toHaveBeenCalledTimes(1);
    expect(latestState).not.toBeNull();
    expect(latestState!.loading).toBe(false);
    expect(latestState!.orchestrations).toEqual([sampleTask]);

    await act(async () => {
      await latestState!.setEnabledByID('orch_1', false);
    });

    expect(mockedUpdateOrchestration).toHaveBeenCalledWith('orch_1', { enabled: false });
    expect(latestState!.orchestrations).toEqual([{ ...sampleTask, enabled: false }]);
    expect(latestState!.error).toBe('');
  });

  it('removes orchestration after successful delete', async () => {
    mockedListOrchestrations.mockResolvedValue([sampleTask]);

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              onRender: (state) => {
                latestState = state;
              },
            }),
          },
        ),
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    await act(async () => {
      await latestState!.deleteByID('orch_1');
    });

    expect(mockedDeleteOrchestration).toHaveBeenCalledWith('orch_1');
    expect(latestState!.orchestrations).toEqual([]);
    expect(latestState!.error).toBe('');
  });

  it('tracks running orchestration id while run is in flight', async () => {
    mockedListOrchestrations.mockResolvedValue([sampleTask]);
    let resolveRun: (() => void) | undefined;
    mockedRunOrchestrationNow.mockImplementation(() => new Promise<void>((resolve) => {
      resolveRun = resolve;
    }));

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              onRender: (state) => {
                latestState = state;
              },
            }),
          },
        ),
      );
      await Promise.resolve();
      await Promise.resolve();
    });

    let runPromise: Promise<void> | undefined;
    await act(async () => {
      runPromise = latestState!.runByID('orch_1');
      await Promise.resolve();
    });

    expect(latestState!.runningOrchestrationID).toBe('orch_1');

    await act(async () => {
      resolveRun?.();
      await runPromise;
      await Promise.resolve();
    });

    expect(latestState!.runningOrchestrationID).toBe('');
  });
});

function HookProbe(props: {
  onRender: (state: HookState) => void;
}) {
  const { copy } = useWebLocale();
  const state = useOrchestrationSectionState(copy);
  props.onRender(state);
  return null;
}

type HookState = ReturnType<typeof useOrchestrationSectionState>;
