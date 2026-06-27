import { parseBridgeConfig, parseProviderListResponse, parseSystemPromptResponse } from './parser';
import type { BridgeConfig, ProviderListResponse, SystemPromptPayload } from '@/lib/types';

describe('lib/api/config/parser', () => {
  it('rejects unknown fields in bridge config payloads', () => {
    const expected: BridgeConfig = {
      provider: 'crs',
      provider_type: 'custom',
      base_url: 'https://lldai.online/openai',
      model: 'gpt-5.4',
      chat_path: '/v1/chat',
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

    expect(() => {
      parseBridgeConfig({
        ...expected,
        future_field: true,
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
      project_root: '/tmp/ghost-os',
      max_turns: 7,
      task_execution_timeout_ms: 600000,
      relay_default_stop_policy: 'max_rounds',
      relay_default_max_rounds: 12,
      relay_default_execution_timeout_ms: 0,
      external_codex_permission_mode: 'safe-yolo',
      llm_completion_retry_count: 0,
      llm_completion_retry_interval_ms: 150,
      api_key_set: true,
      model_selection_enabled: true,
      session_human_log_full_enabled: false,
      session_system_prompt_visible_enabled: false,
      assistant_markdown_enabled: false,
      tool_call_compact_output_enabled: true,
      memory_mode_enabled: true,
      microcompact_enabled: true,
      session_title_mode: 'ai_generated',
      web_search_tavily_url: '',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: false,
      web_search_exa_api_key_set: false,
    });

    expect(parsed.session_system_prompt_visible_enabled).toBe(false);
    expect(parsed.assistant_markdown_enabled).toBe(false);
    expect(parsed.max_turns).toBe(7);
    expect(parsed.task_execution_timeout_ms).toBe(600000);
    expect(parsed.relay_default_stop_policy).toBe('max_rounds');
    expect(parsed.relay_default_max_rounds).toBe(12);
    expect(parsed.relay_default_execution_timeout_ms).toBe(0);
    expect(parsed.external_codex_permission_mode).toBe('safe-yolo');
    expect(parsed.llm_completion_retry_count).toBe(0);
    expect(parsed.llm_completion_retry_interval_ms).toBe(150);
    expect(parsed.tool_call_compact_output_enabled).toBe(true);
    expect(parsed.microcompact_enabled).toBe(true);
    expect(parsed.session_title_mode).toBe('ai_generated');
  });

  it('rejects unknown fields in provider lists', () => {
    const expected: ProviderListResponse = {
      active_provider: 'crs',
      providers: [
        {
          name: 'crs',
          type: 'custom',
          base_url: 'https://lldai.online/openai',
          provider_id: 'provider-crs',
          updated_at: '2026-06-26T00:00:00Z',
          models: ['gpt-5.4'],
          context_window_tokens: 1000000,
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

  it('parses provider context window tokens', () => {
    const parsed = parseProviderListResponse({
      active_provider: 'deepseekv4',
      providers: [
        {
          name: 'deepseekv4',
          type: 'custom',
          base_url: 'https://api.deepseek.com/v1',
          provider_id: 'provider-deepseekv4',
          updated_at: '2026-06-26T00:00:00Z',
          models: ['deepseek-v4-pro'],
          context_window_tokens: 1000000,
          api_key_set: true,
        },
      ],
    });

    expect(parsed.providers[0].context_window_tokens).toBe(1000000);
  });

  it('parses provider sync records from provider lists', () => {
    const parsed = parseProviderListResponse({
      active_provider: 'crs',
      providers: [
        {
          name: 'crs',
          type: 'custom',
          base_url: 'https://lldai.online/openai',
          provider_id: 'provider-crs',
          updated_at: '2026-06-26T00:00:00Z',
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
          models: ['gpt-5.4'],
          api_key_set: true,
        },
      ],
    });

    expect(parsed.provider_sync_records).toEqual([
      {
        provider_id: 'provider-crs',
        updated_at: '2026-06-26T00:00:00Z',
        deleted_at: undefined,
        name: 'crs',
        type: 'custom',
        base_url: 'https://lldai.online/openai',
        models: ['gpt-5.4'],
        context_window_tokens: undefined,
        response_reserve_tokens: undefined,
        model_context_window_tokens: undefined,
        model_response_reserve_tokens: undefined,
        api_key_set: true,
      },
    ]);
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
      tool_definitions: [
        {
          name: 'script_exec',
          description: 'Run a script.',
          parameters: { type: 'object' },
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
    expect(parsed.tool_definitions).toEqual([]);
  });

  it('accepts rule prompt_library insert_point', () => {
    const parsed = parseSystemPromptResponse({
      core_prompt: '',
      rendered_prompt: 'rendered',
      prompt_library: [
        {
          id: 'rule-card',
          name: 'Rule',
          insert_point: 'rule',
          content: 'rule guidance',
          active: true,
        },
      ],
    });

    expect(parsed.prompt_library[0].insert_point).toBe('rule');
    expect(parsed.tool_definitions).toEqual([]);
  });

  it('accepts context prompt_library insert_point', () => {
    const parsed = parseSystemPromptResponse({
      core_prompt: '',
      rendered_prompt: 'rendered',
      prompt_library: [
        {
          id: 'context-card',
          name: 'Context',
          insert_point: 'context',
          content: 'context guidance',
          active: true,
        },
      ],
    });

    expect(parsed.prompt_library[0].insert_point).toBe('context');
    expect(parsed.tool_definitions).toEqual([]);
  });

  it('rejects invalid tool_definitions payloads', () => {
    expect(() => {
      parseSystemPromptResponse({
        core_prompt: '',
        rendered_prompt: '',
        prompt_library: [],
        tool_definitions: {},
      });
    }).toThrow('Invalid system prompt.tool_definitions: expected array');

    expect(() => {
      parseSystemPromptResponse({
        core_prompt: '',
        rendered_prompt: '',
        prompt_library: [],
        tool_definitions: [
          {
            name: 'script_exec',
            description: 'Run a script.',
            parameters: 'invalid',
          },
        ],
      });
    }).toThrow('Invalid system prompt.tool_definitions[0].parameters: expected object');
  });
});
