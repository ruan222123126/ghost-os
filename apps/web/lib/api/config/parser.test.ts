import { parseBridgeConfig, parseProviderListResponse, parseSystemPromptResponse } from './parser';
import type { BridgeConfig, ProviderListResponse, SystemPromptPayload } from '@/lib/types';

describe('lib/api/config/parser', () => {
  it('rejects unknown fields in bridge config payloads', () => {
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
      session_human_log_full_enabled: false,
      assistant_markdown_enabled: true,
      memory_mode_enabled: false,
      web_rooter_enabled: false,
      web_rooter_base_url: 'http://127.0.0.1:8765',
      web_rooter_timeout_ms: 90000,
      web_rooter_api_token_set: false,
      web_search_tavily_url: 'https://proxy.example/tavily',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: true,
      web_search_exa_api_key_set: false,
    };

    expect(() => {
      parseBridgeConfig({
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
      });
    }).toThrow('Invalid payload: unexpected field "future_field"');
  });

  it('parses assistant_markdown_enabled from bridge config payloads', () => {
    const parsed = parseBridgeConfig({
      provider: 'crs',
      provider_type: 'custom',
      base_url: 'https://lldai.online/openai',
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
      assistant_markdown_enabled: false,
      memory_mode_enabled: true,
      web_rooter_enabled: false,
      web_rooter_base_url: 'http://127.0.0.1:8765',
      web_rooter_timeout_ms: 90000,
      web_rooter_api_token_set: false,
      web_search_tavily_url: '',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: false,
      web_search_exa_api_key_set: false,
    });

    expect(parsed.assistant_markdown_enabled).toBe(false);
  });

  it('rejects unknown fields in provider lists', () => {
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

    expect(() => {
      parseProviderListResponse({
        ...expected,
        future_field: true,
        providers: [
          {
            ...expected.providers[0],
            extra_provider_field: 'ignored',
          },
        ],
      });
    }).toThrow('Invalid payload: unexpected field "future_field"');
  });

  it('parses system prompt payloads and rejects unknown fields', () => {
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
    };

    expect(parseSystemPromptResponse(expected)).toEqual(expected);

    expect(() => {
      parseSystemPromptResponse({
        ...expected,
        future_field: true,
      });
    }).toThrow('Invalid payload: unexpected field "future_field"');
  });

  it('rejects invalid prompt_library insert_point and unknown item fields', () => {
    expect(() => {
      parseSystemPromptResponse({
        core_prompt: '',
        rendered_prompt: '',
        prompt_library: [
          {
            id: 'core-job',
            name: 'Core Job',
            insert_point: 'invalid',
            content: '',
            active: true,
          },
        ],
      });
    }).toThrow('Invalid system prompt.prompt_library[0].insert_point: unexpected value "invalid"');

    expect(() => {
      parseSystemPromptResponse({
        core_prompt: '',
        rendered_prompt: '',
        prompt_library: [
          {
            id: 'core-job',
            name: 'Core Job',
            insert_point: 'core_job',
            content: '',
            active: true,
            extra: true,
          },
        ],
      });
    }).toThrow('Invalid payload: unexpected field "extra"');
  });

  it('accepts memory prompt_library insert_point', () => {
    const parsed = parseSystemPromptResponse({
      core_prompt: '',
      rendered_prompt: 'rendered',
      prompt_library: [
        {
          id: 'memory-card',
          name: 'Memory',
          insert_point: 'memory',
          content: 'memory guidance',
          active: true,
        },
      ],
    });

    expect(parsed.prompt_library[0].insert_point).toBe('memory');
  });
});
