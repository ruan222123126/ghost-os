import {
  createProvider,
  deleteProvider,
  getConfig,
  getProviders,
  getSystemPrompts,
  setActiveProvider,
  updateConfig,
  updateProvider,
  updateSystemPrompts,
} from './api';
import { installFetchMock, mockFetchJSON, fetchMock } from '@/lib/api.test.helpers';
import type {
  BridgeConfig,
  ConfigUpdate,
  ProviderConfigInput,
  ProviderListResponse,
  SystemPromptPayload,
  SystemPromptUpdateRequest,
} from '@/lib/types';

describe('lib/api/config/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('getConfig reads /api/config with GET', async () => {
    const expected: BridgeConfig = {
      provider: 'crs',
      provider_type: 'custom',
      base_url: 'https://lldai.online/openai',
      model: 'gpt-5.4',
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
      web_search_tavily_url: 'https://proxy.example/tavily',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: true,
      web_search_exa_api_key_set: false,
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const config = await getConfig();

    expect(config).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/config', expect.any(Object));
  });

  it('updateConfig posts payload and returns updated config', async () => {
    const update: ConfigUpdate = {
      provider: 'crs',
      model: 'gpt-5.4',
      base_url: 'https://lldai.online/openai',
      api_key: 'secret',
      chat_path: '/v1/chat',
      project_root: '/tmp/ghost-os',
      max_turns: 9,
      task_execution_timeout_ms: 600000,
      relay_default_stop_policy: 'max_rounds',
      relay_default_max_rounds: 12,
      relay_default_execution_timeout_ms: 0,
      external_codex_permission_mode: 'safe-yolo',
      web_search_tavily_url: 'https://proxy.example/tavily',
      web_search_exa_url: 'https://proxy.example/exa',
      web_search_exa_api_key: 'exa-secret',
    };

    const expected: BridgeConfig = {
      provider: 'crs',
      provider_type: 'custom',
      base_url: 'https://lldai.online/openai',
      model: 'gpt-5.4',
      chat_path: '/v1/chat',
      project_root: '/tmp/ghost-os',
      max_turns: 9,
      task_execution_timeout_ms: 600000,
      relay_default_stop_policy: 'max_rounds',
      relay_default_max_rounds: 12,
      relay_default_execution_timeout_ms: 0,
      external_codex_permission_mode: 'safe-yolo',
      llm_completion_retry_count: 2,
      llm_completion_retry_interval_ms: 300,
      api_key_set: true,
      model_selection_enabled: true,
      session_human_log_full_enabled: false,
      session_system_prompt_visible_enabled: false,
      assistant_markdown_enabled: false,
      tool_call_compact_output_enabled: true,
      memory_mode_enabled: true,
      microcompact_enabled: true,
      session_title_mode: 'first_message',
      web_search_tavily_url: 'https://proxy.example/tavily',
      web_search_exa_url: 'https://proxy.example/exa',
      web_search_tavily_api_key_set: true,
      web_search_exa_api_key_set: true,
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const config = await updateConfig(update);

    expect(config).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(update),
      }),
    );
  });

  it('getProviders reads provider list with GET', async () => {
    const expected: ProviderListResponse = {
      active_provider: 'crs',
      providers: [
        {
          name: 'crs',
          type: 'custom',
          base_url: 'https://lldai.online/openai',
          provider_id: 'provider-crs',
          updated_at: '2026-06-26T00:00:00Z',
          models: ['gpt-5.4', 'gpt-4'],
          context_window_tokens: 1000000,
          api_key_set: true,
        },
      ],
      provider_sync_records: [
        {
          provider_id: 'provider-crs',
          updated_at: '2026-06-26T00:00:00Z',
          name: 'crs',
          type: 'custom',
          base_url: 'https://lldai.online/openai',
          models: ['gpt-5.4', 'gpt-4'],
          context_window_tokens: 1000000,
          api_key_set: true,
        },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const providers = await getProviders();

    expect(providers).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/config/providers', expect.any(Object));
  });

  it('createProvider posts payload and returns provider list', async () => {
    const input: ProviderConfigInput = {
      name: 'openai',
      type: 'openai',
      base_url: 'https://api.openai.com/v1',
      api_key: 'sk-yyy',
      models: ['gpt-5.4'],
      context_window_tokens: 128000,
    };

    mockFetchJSON({
      status: 'success',
      payload: { active_provider: 'openai', providers: [] },
      error: '',
    });

    await createProvider(input);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config/providers',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      }),
    );
  });

  it('updateProvider targets the provider route', async () => {
    const input: ProviderConfigInput = {
      name: 'crs',
      type: 'custom',
      base_url: 'https://lldai.online/openai',
      models: ['gpt-5.4'],
      context_window_tokens: 1000000,
    };

    mockFetchJSON({
      status: 'success',
      payload: { active_provider: 'crs', providers: [] },
      error: '',
    });

    await updateProvider('crs', input);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config/providers/crs',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(input),
      }),
    );
  });

  it('deleteProvider targets the provider route', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { active_provider: '', providers: [] },
      error: '',
    });

    await deleteProvider('crs');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config/providers/crs',
      expect.objectContaining({
        method: 'DELETE',
      }),
    );
  });

  it('setActiveProvider updates the active provider route', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { active_provider: 'crs', providers: [] },
      error: '',
    });

    await setActiveProvider('crs');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/config/active-provider',
      expect.objectContaining({
        method: 'PUT',
      body: JSON.stringify({ name: 'crs' }),
      }),
    );
  });

  it('getSystemPrompts reads /api/prompts/system with GET', async () => {
    const expected: SystemPromptPayload = {
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
      tool_definitions: [
        {
          name: 'script_exec',
          description: 'Run a script.',
          parameters: { type: 'object' },
        },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const prompts = await getSystemPrompts();

    expect(prompts).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/prompts/system', expect.any(Object));
  });

  it('updateSystemPrompts patches a single prompt field and preserves trace id', async () => {
    const update: SystemPromptUpdateRequest = {
      core_prompt: 'updated core guidance',
      trace_id: 'trace-123',
    };
    const expected: SystemPromptPayload = {
      core_prompt: 'updated core guidance',
      rendered_prompt: 'base prompt with updated core guidance',
      prompt_library: [
        {
          id: 'core-job',
          name: 'Core Job',
          insert_point: 'core_job',
          content: 'updated core guidance',
          active: true,
        },
      ],
      tool_definitions: [
        {
          name: 'script_exec',
          description: 'Run a script.',
          parameters: { type: 'object' },
        },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const prompts = await updateSystemPrompts(update);

    expect(prompts).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/prompts/system',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify(update),
      }),
    );
  });

  it('throws fallback status message when error field is empty', async () => {
    mockFetchJSON(
      {
        status: 'error',
        payload: {},
        error: '',
      },
      { ok: false, status: 502 },
    );

    await expect(getConfig()).rejects.toThrow('Request failed with status 502');
  });
});
