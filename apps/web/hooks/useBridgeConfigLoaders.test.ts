import React, { useRef } from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getConfig, getProviders } from '@/lib/api/config/api';
import type { BridgeConfig, ProviderConfig, ProviderListResponse } from '@/lib/types';
import { useBridgeConfigLoaders } from './useBridgeConfigLoaders';

jest.mock('@/lib/api/config/api', () => ({
  getConfig: jest.fn(),
  getProviders: jest.fn(),
}));

const mockedGetConfig = getConfig as jest.MockedFunction<typeof getConfig>;
const mockedGetProviders = getProviders as jest.MockedFunction<typeof getProviders>;

describe('hooks/useBridgeConfigLoaders', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('loads config and providers on mount through shared mounted loader', async () => {
    const calls = createLoaderCalls();
    const config = buildBridgeConfig();
    const provider = buildProvider();
    mockedGetConfig.mockResolvedValue(config);
    mockedGetProviders.mockResolvedValue(buildProviderList([provider]));

    await renderLoaders(calls);

    expect(calls.setConfigLoading).toHaveBeenCalledWith(true);
    expect(calls.setModelOptionsLoading).toHaveBeenCalledWith(true);
    expect(calls.setConfig).toHaveBeenCalledWith(config);
    expect(calls.setProviders).toHaveBeenCalledWith([provider]);
    expect(calls.setConfigLoadError).toHaveBeenCalledWith('');
    expect(calls.setProviderLoadError).toHaveBeenCalledWith('');
    expect(calls.setConfigLoading).toHaveBeenLastCalledWith(false);
    expect(calls.setModelOptionsLoading).toHaveBeenLastCalledWith(false);
  });

  it('keeps silent reload failures out of visible loading and error state', async () => {
    const calls = createLoaderCalls();
    mockedGetConfig.mockResolvedValue(buildBridgeConfig());
    mockedGetProviders.mockResolvedValue(buildProviderList([buildProvider()]));
    const latest = await renderLoaders(calls);
    calls.setConfigLoading.mockClear();
    calls.setModelOptionsLoading.mockClear();
    calls.setConfigLoadError.mockClear();
    calls.setProviderLoadError.mockClear();
    mockedGetConfig.mockRejectedValueOnce(new Error('config down'));
    mockedGetProviders.mockRejectedValueOnce(new Error('providers down'));

    await act(async () => {
      await Promise.all([latest.loadConfig(true), latest.loadProviders(true)]);
    });

    expect(calls.setConfigLoading).not.toHaveBeenCalled();
    expect(calls.setModelOptionsLoading).not.toHaveBeenCalled();
    expect(calls.setConfigLoadError).not.toHaveBeenCalled();
    expect(calls.setProviderLoadError).not.toHaveBeenCalled();
  });
});

async function renderLoaders(calls: LoaderCalls) {
  let latest!: ReturnType<typeof useBridgeConfigLoaders>;

  await act(async () => {
    TestRenderer.create(
      React.createElement(BridgeConfigLoadersProbe, {
        calls,
        onRender: (loaders) => {
          latest = loaders;
        },
      }),
    );
    await flushPromises();
  });

  return latest;
}

function BridgeConfigLoadersProbe(props: {
  calls: LoaderCalls;
  onRender: (loaders: ReturnType<typeof useBridgeConfigLoaders>) => void;
}) {
  const mountedRef = useRef(true);
  const loaders = useBridgeConfigLoaders({
    autoRefresh: false,
    mountedRef,
    loadConfigFallbackMessage: 'failed config',
    loadProvidersFallbackMessage: 'failed providers',
    setConfig: props.calls.setConfig,
    setProviders: props.calls.setProviders,
    setConfigLoading: props.calls.setConfigLoading,
    setModelOptionsLoading: props.calls.setModelOptionsLoading,
    setConfigLoadError: props.calls.setConfigLoadError,
    setProviderLoadError: props.calls.setProviderLoadError,
  });
  props.onRender(loaders);
  return null;
}

interface LoaderCalls {
  setConfig: jest.MockedFunction<(config: BridgeConfig) => void>;
  setProviders: jest.MockedFunction<(providers: ProviderConfig[]) => void>;
  setConfigLoading: jest.MockedFunction<(loading: boolean) => void>;
  setModelOptionsLoading: jest.MockedFunction<(loading: boolean) => void>;
  setConfigLoadError: jest.MockedFunction<(error: string) => void>;
  setProviderLoadError: jest.MockedFunction<(error: string) => void>;
}

function createLoaderCalls(): LoaderCalls {
  return {
    setConfig: jest.fn(),
    setProviders: jest.fn(),
    setConfigLoading: jest.fn(),
    setModelOptionsLoading: jest.fn(),
    setConfigLoadError: jest.fn(),
    setProviderLoadError: jest.fn(),
  };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

function buildBridgeConfig(): BridgeConfig {
  return {
    provider: 'crs',
    provider_type: 'custom',
    base_url: 'https://proxy.example/openai',
    model: 'gpt-5',
    chat_path: '',
    project_root: '',
    max_turns: 20,
    task_execution_timeout_ms: 300000,
    relay_default_stop_policy: 'ai_decides',
    relay_default_max_rounds: 20,
    relay_default_execution_timeout_ms: 0,
    external_codex_permission_mode: 'default',
    llm_completion_retry_count: 1,
    llm_completion_retry_interval_ms: 200,
    api_key_set: true,
    model_selection_enabled: true,
    session_human_log_full_enabled: false,
    session_system_prompt_visible_enabled: true,
    assistant_markdown_enabled: true,
    tool_call_compact_output_enabled: false,
    memory_mode_enabled: false,
    microcompact_enabled: false,
    session_title_mode: 'session_id',
    web_search_tavily_url: '',
    web_search_exa_url: '',
    web_search_tavily_api_key_set: false,
    web_search_exa_api_key_set: false,
  };
}

function buildProvider(): ProviderConfig {
  return {
    name: 'crs',
    type: 'custom',
    base_url: 'https://proxy.example/openai',
    provider_id: 'provider-crs',
    updated_at: '2026-06-27T00:00:00Z',
    models: ['gpt-5'],
    api_key_set: true,
  };
}

function buildProviderList(providers: ProviderConfig[]): ProviderListResponse {
  return {
    active_provider: providers[0]?.name ?? '',
    providers,
  };
}
