import type { BridgeConfig, ProviderConfig } from '@/lib/types';
import { buildProviderModelOptions } from './bridgeConfigModelOptions';

function buildConfig(overrides: Partial<BridgeConfig> = {}): BridgeConfig {
  return {
    provider: 'anthropic-main',
    provider_type: 'anthropic',
    base_url: 'https://api.anthropic.com',
    model: 'claude-3-7-sonnet',
    chat_path: '/v1/messages',
    project_root: '',
    max_turns: 20,
    task_execution_timeout_ms: 300000,
    relay_default_stop_policy: 'ai_decides',
    relay_default_max_rounds: 20,
    relay_default_execution_timeout_ms: 0,
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

function buildProvider(overrides: Partial<ProviderConfig> & Pick<ProviderConfig, 'name' | 'type'>): ProviderConfig {
  return {
    name: overrides.name,
    type: overrides.type,
    base_url: overrides.base_url ?? '',
    provider_id: overrides.provider_id ?? overrides.name,
    updated_at: overrides.updated_at ?? '2026-01-01T00:00:00Z',
    deleted_at: overrides.deleted_at,
    models: overrides.models,
    context_window_tokens: overrides.context_window_tokens,
    response_reserve_tokens: overrides.response_reserve_tokens,
    model_context_window_tokens: overrides.model_context_window_tokens,
    model_response_reserve_tokens: overrides.model_response_reserve_tokens,
    api_key_set: overrides.api_key_set ?? true,
  };
}

describe('hooks/bridgeConfigModelOptions', () => {
  it('only exposes models from the active provider group', () => {
    const options = buildProviderModelOptions(buildConfig(), [
      buildProvider({ name: 'openai-main', type: 'openai', models: ['gpt-5.4'] }),
      buildProvider({ name: 'anthropic-main', type: 'anthropic', models: ['claude-3-7-sonnet', 'claude-3-5-haiku'] }),
    ]);

    expect(options).toEqual([
      {
        providerName: 'anthropic-main',
        providerType: 'anthropic',
        model: 'claude-3-7-sonnet',
      },
      {
        providerName: 'anthropic-main',
        providerType: 'anthropic',
        model: 'claude-3-5-haiku',
      },
    ]);
  });

  it('keeps the active runtime model selectable when the active provider card omits it', () => {
    const options = buildProviderModelOptions(buildConfig({
      model: 'claude-opus-4',
    }), [
      buildProvider({ name: 'anthropic-main', type: 'anthropic', models: ['claude-3-5-haiku'] }),
      buildProvider({ name: 'openai-main', type: 'openai', models: ['gpt-5.4'] }),
    ]);

    expect(options).toEqual([
      {
        providerName: 'anthropic-main',
        providerType: 'anthropic',
        model: 'claude-3-5-haiku',
      },
      {
        providerName: 'anthropic-main',
        providerType: 'anthropic',
        model: 'claude-opus-4',
      },
    ]);
  });
});
