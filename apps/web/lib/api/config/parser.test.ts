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
      web_search_tavily_url: '',
      web_search_exa_url: '',
      web_search_tavily_api_key_set: false,
      web_search_exa_api_key_set: false,
    });

    expect(parsed.session_system_prompt_visible_enabled).toBe(false);
    expect(parsed.assistant_markdown_enabled).toBe(false);
    expect(parsed.max_turns).toBe(7);
    expect(parsed.llm_completion_retry_count).toBe(0);
    expect(parsed.llm_completion_retry_interval_ms).toBe(150);
    expect(parsed.tool_call_compact_output_enabled).toBe(true);
    expect(parsed.microcompact_enabled).toBe(true);
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
          models: ['deepseek-v4-pro'],
          context_window_tokens: 1000000,
          api_key_set: true,
        },
      ],
    });

    expect(parsed.providers[0].context_window_tokens).toBe(1000000);
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
