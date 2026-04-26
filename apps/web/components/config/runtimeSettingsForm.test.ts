import type { BridgeConfig } from '@/lib/types';
import { buildRuntimeUpdate, createRuntimeFormState } from './runtimeSettingsForm';

function buildBridgeConfig(overrides: Partial<BridgeConfig> = {}): BridgeConfig {
  return {
    provider: 'custom',
    provider_type: 'custom',
    base_url: 'https://example.com/v1',
    model: 'gpt-5.4',
    chat_path: '/v1/chat',
    api_key_set: true,
    model_selection_enabled: true,
    graphql_default_source: '',
    graphql_tool_runtime_enabled: false,
    graphql_text_sanitize_enabled: true,
    graphql_sources: [],
    graphql_mutation_policies: [],
    session_human_log_full_enabled: false,
    assistant_markdown_enabled: true,
    memory_mode_enabled: false,
    web_rooter_enabled: false,
    web_rooter_base_url: 'http://127.0.0.1:8765',
    web_rooter_timeout_ms: 90000,
    web_rooter_api_token_set: false,
    web_search_tavily_url: '',
    web_search_exa_url: '',
    web_search_tavily_api_key_set: false,
    web_search_exa_api_key_set: false,
    ...overrides,
  };
}

describe('components/config/runtimeSettingsForm', () => {
  it('defaults assistant markdown toggle to true when config is absent', () => {
    const form = createRuntimeFormState(null);
    expect(form.assistantMarkdownEnabled).toBe(true);
    expect(form.memoryModeEnabled).toBe(false);
  });

  it('reads assistant markdown toggle from bridge config', () => {
    const form = createRuntimeFormState(buildBridgeConfig({
      assistant_markdown_enabled: false,
      memory_mode_enabled: true,
    }));
    expect(form.assistantMarkdownEnabled).toBe(false);
    expect(form.memoryModeEnabled).toBe(true);
  });

  it('includes assistant markdown toggle in config update payload', () => {
    const update = buildRuntimeUpdate(true, {
      provider: 'custom',
      apiKey: '',
      baseURL: 'https://example.com/v1',
      model: 'gpt-5.4',
      chatPath: '/v1/chat',
      graphqlToolRuntimeEnabled: false,
      graphqlTextSanitizeEnabled: true,
      sessionHumanLogFullEnabled: false,
      assistantMarkdownEnabled: false,
      memoryModeEnabled: true,
      webRooterEnabled: false,
      webRooterBaseURL: '',
      webRooterAPIToken: '',
      webRooterTimeoutMS: '',
      webSearchTavilyURL: '',
      webSearchExaURL: '',
      webSearchTavilyAPIKey: '',
      webSearchExaAPIKey: '',
    }, 'en-US');

    expect(update.assistant_markdown_enabled).toBe(false);
    expect(update.memory_mode_enabled).toBe(true);
  });
});
