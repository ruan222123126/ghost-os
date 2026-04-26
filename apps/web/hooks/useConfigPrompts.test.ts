import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getSystemPrompts, updateSystemPrompts } from '@/lib/api/config/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { SystemPromptPayload } from '@/lib/types';
import { useConfigPrompts } from './useConfigPrompts';

jest.mock('@/lib/api/config/api', () => ({
  getSystemPrompts: jest.fn(),
  updateSystemPrompts: jest.fn(),
}));

const mockedGetSystemPrompts = getSystemPrompts as jest.MockedFunction<typeof getSystemPrompts>;
const mockedUpdateSystemPrompts = updateSystemPrompts as jest.MockedFunction<typeof updateSystemPrompts>;

describe('hooks/useConfigPrompts', () => {
  const samplePrompts: SystemPromptPayload = {
    core_prompt: 'core guidance',
    rendered_prompt: 'base prompt with core guidance',
    prompt_library: [
      {
        id: 'core-job',
        name: 'Core Job',
        insert_point: 'core_job',
        content: 'core guidance',
        active: true,
      },
    ],
  };

  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('loads prompts on mount and stores the returned payload', async () => {
    const load = createDeferred<SystemPromptPayload>();
    mockedGetSystemPrompts.mockReturnValue(load.promise);

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
    expect(latestState!.promptsLoading).toBe(true);
    expect(mockedGetSystemPrompts).toHaveBeenCalledTimes(1);

    await act(async () => {
      load.resolve(samplePrompts);
      await load.promise;
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.promptsLoading).toBe(false);
    expect(latestState!.promptError).toBe('');
    expect(latestState!.prompts).toEqual(samplePrompts);
  });

  it('stores load errors without clearing the prompt state shape', async () => {
    const load = createDeferred<SystemPromptPayload>();
    mockedGetSystemPrompts.mockReturnValue(load.promise);

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
    expect(latestState!.promptsLoading).toBe(false);
    expect(latestState!.promptError).toBe('bridge offline');
    expect(latestState!.prompts).toBeNull();
  });

  it('saves prompt_library and replaces the payload with the update response', async () => {
    const load = createDeferred<SystemPromptPayload>();
    mockedGetSystemPrompts.mockReturnValue(load.promise);

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
      load.resolve(samplePrompts);
      await load.promise;
    });

    const save = createDeferred<SystemPromptPayload>();
    mockedUpdateSystemPrompts.mockReturnValue(save.promise);

    let saveRun: Promise<void> | undefined;
    const updatedLibrary = [
      {
        id: 'core-job',
        name: 'Core Job',
        insert_point: 'core_job' as const,
        content: 'updated core guidance',
        active: true,
      },
    ];
    act(() => {
      saveRun = latestState!.savePromptLibrary(updatedLibrary);
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.promptSaving).toBe(true);
    expect(mockedUpdateSystemPrompts).toHaveBeenCalledWith({
      prompt_library: updatedLibrary,
    });

    const updatedPrompts: SystemPromptPayload = {
      ...samplePrompts,
      core_prompt: 'updated core guidance',
      rendered_prompt: 'base prompt with updated core guidance',
      prompt_library: updatedLibrary,
    };

    await act(async () => {
      save.resolve(updatedPrompts);
      await saveRun;
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.promptSaving).toBe(false);
    expect(latestState!.promptError).toBe('');
    expect(latestState!.prompts).toEqual(updatedPrompts);
  });

  it('stores save errors and preserves the loaded prompt payload', async () => {
    const load = createDeferred<SystemPromptPayload>();
    mockedGetSystemPrompts.mockReturnValue(load.promise);

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
      load.resolve(samplePrompts);
      await load.promise;
    });

    const save = createDeferred<SystemPromptPayload>();
    mockedUpdateSystemPrompts.mockReturnValue(save.promise);

    let saveRun: Promise<void> | undefined;
    const updatedLibrary = [
      {
        id: 'core-job',
        name: 'Core Job',
        insert_point: 'core_job' as const,
        content: 'updated core guidance',
        active: true,
      },
    ];
    act(() => {
      saveRun = latestState!.savePromptLibrary(updatedLibrary);
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.promptSaving).toBe(true);

    await act(async () => {
      save.reject(new Error('save denied'));
      await save.promise.catch(() => undefined);
      await saveRun;
    });

    expect(latestState).not.toBeNull();
    expect(latestState!.promptSaving).toBe(false);
    expect(latestState!.promptError).toBe('save denied');
    expect(latestState!.prompts).toEqual(samplePrompts);
  });
});

function HookProbe(props: {
  open: boolean;
  onRender: (state: HookState) => void;
}) {
  const state = useConfigPrompts({ open: props.open });
  props.onRender(state);
  return null;
}

interface HookState {
  prompts: SystemPromptPayload | null;
  promptsLoading: boolean;
  promptSaving: boolean;
  promptError: string;
  refreshPrompts: () => Promise<void>;
  savePromptLibrary: (promptLibrary: SystemPromptPayload['prompt_library']) => Promise<void>;
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve;
    reject = nextReject;
  });

  return {
    promise,
    resolve,
    reject,
  };
}
