import { parseBridgeConfig, parseProviderListResponse } from './parser';
import type { BridgeConfig, ProviderListResponse } from '@/lib/types';

describe('lib/api/config/parser', () => {
  it('ignores unknown fields in bridge config payloads', () => {
    const domain = {
      name: 'people',
      root_queries: ['viewer'],
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
      graphql_sources: [
        {
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
          domains: [domain],
        },
      ],
      graphql_mutation_policies: [
        {
          name: 'update_viewer',
          source: 'crm',
          domain: 'people',
          root_mutation: 'updateViewer',
          idempotency_mode: 'header',
          idempotency_header: 'Idempotency-Key',
        },
      ],
      web_rooter_enabled: false,
      web_rooter_api_token_set: false,
      web_search_tavily_url: 'https://proxy.example/tavily',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: true,
      web_search_exa_api_key_set: false,
    };

    expect(parseBridgeConfig({
      ...expected,
      future_field: true,
      graphql_sources: [
        {
          ...expected.graphql_sources[0],
          extra_source_field: 'ignored',
          domains: [
            {
              ...domain,
              extra_domain_field: 'ignored',
            },
          ],
        },
      ],
      graphql_mutation_policies: [
        {
          ...expected.graphql_mutation_policies[0],
          extra_policy_field: 'ignored',
        },
      ],
    })).toEqual(expected);
  });

  it('ignores unknown fields in provider lists', () => {
    const expected: ProviderListResponse = {
      active_provider: 'crs',
      providers: [
        {
          name: 'crs',
          type: 'custom',
          base_url: 'https://lldai.online/openai',
          models: ['gpt-5.4'],
          api_key_set: true,
        },
      ],
    };

    expect(parseProviderListResponse({
      ...expected,
      future_field: true,
      providers: [
        {
          ...expected.providers[0],
          extra_provider_field: 'ignored',
        },
      ],
    })).toEqual(expected);
  });
});
