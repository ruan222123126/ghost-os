import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { updateConfig } from '@/lib/api/config/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { BridgeConfig, ProviderConfig } from '@/lib/types';
import { useBridgeConfig } from './useBridgeConfig';
import { useBridgeConfigLoaders } from './useBridgeConfigLoaders';

jest.mock('@/lib/api/config/api', () => ({
  updateConfig: jest.fn(),
}));

jest.mock('./useBridgeConfigLoaders', () => ({
  useBridgeConfigLoaders: jest.fn(),
}));

const mockedUpdateConfig = updateConfig as jest.MockedFunction<typeof updateConfig>;
const mockedUseBridgeConfigLoaders = useBridgeConfigLoaders as jest.MockedFunction<typeof useBridgeConfigLoaders>;

let capturedLoaderOptions: Parameters<typeof useBridgeConfigLoaders>[0] | null = null;
let loadConfig: jest.MockedFunction<() => Promise<void>>;
let loadProviders: jest.MockedFunction<(silent?: boolean) => Promise<void>>;

describe('hooks/useBridgeConfig', () => {
  beforeEach(() => {
    jest.resetAllMocks();
    capturedLoaderOptions = null;
    loadConfig = jest.fn().mockResolvedValue(undefined);
    loadProviders = jest.fn().mockResolvedValue(undefined);
    mockedUseBridgeConfigLoaders.mockImplementation((options) => {
      capturedLoaderOptions = options;
      return { loadConfig, loadProviders };
    });
  });

  it('saves config updates into the local config state', async () => {
    const latest = renderBridgeConfig();
    const initial = buildConfig({ model: 'gpt-5' });
    const updated = buildConfig({ model: 'gpt-4.1' });
    mockedUpdateConfig.mockResolvedValue(updated);

    act(() => {
      capturedLoaderOptions!.setConfig(initial);
    });

    await act(async () => {
      await latest.current.saveConfig({ model: 'gpt-4.1' });
    });

    expect(mockedUpdateConfig).toHaveBeenCalledWith({ model: 'gpt-4.1' });
    expect(latest.current.config?.model).toBe('gpt-4.1');
    expect(latest.current.savingConfig).toBe(false);
  });

  it('selects an active provider model and refreshes providers silently', async () => {
    const latest = renderBridgeConfig();
    const initial = buildConfig({ provider: 'openai', model: 'gpt-5' });
    const updated = buildConfig({ provider: 'anthropic', provider_type: 'anthropic', model: 'claude-3.7' });
    mockedUpdateConfig.mockResolvedValue(updated);

    act(() => {
      capturedLoaderOptions!.setConfig(initial);
      capturedLoaderOptions!.setProviders([
        buildProvider({ name: 'openai', models: ['gpt-5'] }),
        buildProvider({ name: 'anthropic', type: 'anthropic', models: ['claude-3.7'] }),
      ]);
    });

    await act(async () => {
      const saved = await latest.current.selectActiveModel({
        providerName: 'anthropic',
        providerType: 'anthropic',
        model: 'claude-3.7',
      });
      expect(saved).toBe(true);
    });

    expect(mockedUpdateConfig).toHaveBeenCalledWith({
      provider: 'anthropic',
      model: 'claude-3.7',
    });
    expect(loadProviders).toHaveBeenCalledWith(true);
    expect(latest.current.config?.provider).toBe('anthropic');
  });

  it('refreshes config and providers together', async () => {
    const latest = renderBridgeConfig();

    await act(async () => {
      await latest.current.refreshConfig();
    });

    expect(loadConfig).toHaveBeenCalledTimes(1);
    expect(loadProviders).toHaveBeenCalledTimes(1);
  });
});

function renderBridgeConfig() {
  const latest: { current: ReturnType<typeof useBridgeConfig> } = {
    current: null as unknown as ReturnType<typeof useBridgeConfig>,
  };

  act(() => {
    TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(BridgeConfigProbe, {
          onRender: (state) => {
            latest.current = state;
          },
        }),
      }),
    );
  });

  return latest;
}

function BridgeConfigProbe(props: {
  onRender: (state: ReturnType<typeof useBridgeConfig>) => void;
}) {
  const state = useBridgeConfig();
  props.onRender(state);
  return null;
}

function buildConfig(overrides: Partial<BridgeConfig> = {}): BridgeConfig {
  return {
    provider: 'openai',
    provider_type: 'openai',
    base_url: 'https://api.openai.com/v1',
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
    ...overrides,
  };
}

function buildProvider(overrides: Partial<ProviderConfig> = {}): ProviderConfig {
  return {
    name: 'openai',
    type: 'openai',
    base_url: 'https://api.openai.com/v1',
    provider_id: 'provider-openai',
    updated_at: '2026-06-27T00:00:00Z',
    models: ['gpt-5'],
    api_key_set: true,
    ...overrides,
  };
}
