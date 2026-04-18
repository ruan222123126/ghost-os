import {
  createProvider,
  deleteProvider,
  getConfig,
  getProviders,
  setActiveProvider,
  updateConfig,
  updateProvider,
} from './api';
import { installFetchMock, mockFetchJSON, fetchMock } from '@/lib/api.test.helpers';
import type {
  BridgeConfig,
  ConfigUpdate,
  ProviderConfigInput,
  ProviderListResponse,
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
      api_key_set: true,
      model_selection_enabled: true,
      graphql_default_source: 'crm',
      graphql_tool_runtime_enabled: false,
      graphql_text_sanitize_enabled: true,
      graphql_sources: [{
        name: 'crm',
        endpoint: 'https://crm.example/graphql',
        schema_path: '/schemas/crm.json',
        timeout_ms: 5000,
        max_response_bytes: 8192,
        max_depth: 5,
        max_fields: 32,
        max_root_fields: 2,
        max_fragments: 4,
        api_key_set: true,
      }],
      graphql_mutation_policies: [{
        name: 'update_viewer',
        source: 'crm',
        domain: 'people',
        root_mutation: 'updateViewer',
        idempotency_mode: 'header',
        idempotency_header: 'Idempotency-Key',
      }],
      session_human_log_full_enabled: false,
      web_rooter_enabled: false,
      web_rooter_base_url: 'http://127.0.0.1:8765',
      web_rooter_timeout_ms: 90000,
      web_rooter_api_token_set: false,
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
      api_key_set: true,
      model_selection_enabled: true,
      graphql_default_source: 'crm',
      graphql_tool_runtime_enabled: false,
      graphql_text_sanitize_enabled: true,
      graphql_sources: [{
        name: 'crm',
        endpoint: 'https://crm.example/graphql',
        schema_path: '/schemas/crm.json',
        timeout_ms: 5000,
        max_response_bytes: 8192,
        max_depth: 5,
        max_fields: 32,
        max_root_fields: 2,
        max_fragments: 4,
        api_key_set: true,
      }],
      graphql_mutation_policies: [{
        name: 'update_viewer',
        source: 'crm',
        domain: 'people',
        root_mutation: 'updateViewer',
        idempotency_mode: 'header',
        idempotency_header: 'Idempotency-Key',
      }],
      session_human_log_full_enabled: false,
      web_rooter_enabled: false,
      web_rooter_base_url: 'http://127.0.0.1:8765',
      web_rooter_timeout_ms: 90000,
      web_rooter_api_token_set: false,
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
          models: ['gpt-5.4', 'gpt-4'],
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
