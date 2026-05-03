import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import {
  activatePreset,
  createPreset,
  deletePreset,
  listPresets,
  updatePreset,
} from '@/lib/api/presets/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { PresetPayload } from '@/lib/types';
import { useConfigPresets } from './useConfigPresets';

jest.mock('@/lib/api/presets/api', () => ({
  activatePreset: jest.fn(),
  listPresets: jest.fn(),
  createPreset: jest.fn(),
  updatePreset: jest.fn(),
  deletePreset: jest.fn(),
}));

const mockedActivatePreset = activatePreset as jest.MockedFunction<typeof activatePreset>;
const mockedListPresets = listPresets as jest.MockedFunction<typeof listPresets>;
const mockedCreatePreset = createPreset as jest.MockedFunction<typeof createPreset>;
const mockedUpdatePreset = updatePreset as jest.MockedFunction<typeof updatePreset>;
const mockedDeletePreset = deletePreset as jest.MockedFunction<typeof deletePreset>;

describe('hooks/useConfigPresets', () => {
  const samplePresets: PresetPayload[] = [
    {
      id: 'preset-a',
      name: 'Default',
      tool_allowlist: ['script_exec'],
      prompt_refs: {
        rule: 'rule-card',
        context: ['context-a'],
      },
    },
  ];

  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('loads presets on mount and stores the returned payload', async () => {
    const load = createDeferred<PresetPayload[]>();
    mockedListPresets.mockReturnValue(load.promise);

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              open: true,
              onRender: (state) => {
                latestState = state;
              },
            }),
          },
        ),
      );
      await Promise.resolve();
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.presetsLoading).toBe(true);
    expect(mockedListPresets).toHaveBeenCalledTimes(1);

    await act(async () => {
      load.resolve(samplePresets);
      await load.promise;
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.presetsLoading).toBe(false);
    expect(latestState!.presetError).toBe('');
    expect(latestState!.presets).toEqual(samplePresets);
  });

  it('stores load errors without clearing the preset list shape', async () => {
    const load = createDeferred<PresetPayload[]>();
    mockedListPresets.mockReturnValue(load.promise);

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              open: true,
              onRender: (state) => {
                latestState = state;
              },
            }),
          },
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      load.reject(new Error('bridge offline'));
      await load.promise.catch(() => undefined);
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.presetsLoading).toBe(false);
    expect(latestState!.presetError).toBe('bridge offline');
    expect(latestState!.presets).toEqual([]);
  });

  it('creates, updates, and deletes presets through state transitions', async () => {
    mockedListPresets.mockResolvedValue(samplePresets);

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              open: true,
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

    const createdPreset: PresetPayload = {
      id: 'preset-b',
      name: 'Research',
      tool_allowlist: ['web_search'],
      prompt_refs: {
        memory: 'memory-card',
        context: ['context-b', 'context-a'],
      },
    };
    mockedCreatePreset.mockResolvedValue(createdPreset);

    await act(async () => {
      await latestState!.createPreset({
        name: 'Research',
        tool_allowlist: ['web_search'],
        prompt_refs: { memory: 'memory-card', context: ['context-b', 'context-a'] },
      });
    });

    expect(latestState!.presets).toEqual([...samplePresets, createdPreset]);

    const updatedPreset: PresetPayload = {
      ...createdPreset,
      name: 'Research Updated',
      tool_allowlist: ['sfind'],
      prompt_refs: {
        core_job: 'core-card',
        context: ['context-a'],
      },
    };
    mockedUpdatePreset.mockResolvedValue(updatedPreset);

    await act(async () => {
      await latestState!.updatePresetByID('preset-b', {
        name: 'Research Updated',
        tool_allowlist: ['sfind'],
        prompt_refs: { core_job: 'core-card', context: ['context-a'] },
      });
    });

    expect(latestState!.presets).toEqual([samplePresets[0], updatedPreset]);

    mockedDeletePreset.mockResolvedValue(updatedPreset);

    await act(async () => {
      await latestState!.deletePresetByID('preset-b');
    });

    expect(latestState!.presets).toEqual(samplePresets);

    mockedActivatePreset.mockResolvedValue(samplePresets[0]);

    await act(async () => {
      await latestState!.activatePresetByID('preset-a');
    });

    expect(latestState!.presetSaving).toBe(false);
    expect(latestState!.presets).toEqual(samplePresets);
  });

  it('stores mutate errors and preserves the current presets', async () => {
    mockedListPresets.mockResolvedValue(samplePresets);

    let latestState: HookState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              open: true,
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

    mockedCreatePreset.mockRejectedValue(new Error('save denied'));

    await act(async () => {
      await latestState!.createPreset({
        name: 'Invalid',
        tool_allowlist: [],
        prompt_refs: {},
      });
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.presetSaving).toBe(false);
    expect(latestState!.presetError).toBe('save denied');
    expect(latestState!.presets).toEqual(samplePresets);
  });
});

function HookProbe(props: {
  open: boolean;
  onRender: (state: HookState) => void;
}) {
  const state = useConfigPresets({ open: props.open });
  props.onRender(state);
  return null;
}

interface HookState {
  presets: PresetPayload[];
  presetsLoading: boolean;
  presetSaving: boolean;
  presetError: string;
  refreshPresets: () => Promise<void>;
  createPreset: (input: {
    name: string;
    tool_allowlist?: string[];
    prompt_refs?: {
      rule?: string;
      core_job?: string;
      memory?: string;
      context?: string[];
    };
  }) => Promise<void>;
  updatePresetByID: (id: string, input: {
    name?: string;
    tool_allowlist?: string[];
    prompt_refs?: {
      rule?: string;
      core_job?: string;
      memory?: string;
      context?: string[];
    };
  }) => Promise<void>;
  deletePresetByID: (id: string) => Promise<void>;
  activatePresetByID: (id: string) => Promise<void>;
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
