import type { BridgeConfig } from '@/lib/types';
import { buildRelaySettingsUpdate, createRelaySettingsFormState } from './relaySettingsForm';

function buildBridgeConfig(overrides: Partial<BridgeConfig> = {}): BridgeConfig {
  return {
    provider: 'custom',
    provider_type: 'custom',
    base_url: 'https://example.com/v1',
    model: 'gpt-5.4',
    chat_path: '',
    project_root: '',
    max_turns: 20,
    task_execution_timeout_ms: 300000,
    relay_default_stop_policy: 'max_rounds',
    relay_default_max_rounds: 8,
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

describe('components/config/relaySettingsForm', () => {
  it('reads relay defaults from bridge config', () => {
    expect(createRelaySettingsFormState(buildBridgeConfig())).toEqual({
      stopPolicy: 'max_rounds',
      maxRounds: '8',
      executionTimeoutMS: '0',
    });
  });

  it('builds relay config update payload', () => {
    expect(buildRelaySettingsUpdate({
      stopPolicy: 'ai_decides',
      maxRounds: '20',
      executionTimeoutMS: '0',
    })).toEqual({
      relay_default_stop_policy: 'ai_decides',
      relay_default_max_rounds: 20,
      relay_default_execution_timeout_ms: 0,
    });
  });

  it('rejects invalid relay numbers', () => {
    expect(() => buildRelaySettingsUpdate({
      stopPolicy: 'max_rounds',
      maxRounds: '0',
      executionTimeoutMS: '0',
    })).toThrow('relay_default_max_rounds');

    expect(() => buildRelaySettingsUpdate({
      stopPolicy: 'max_rounds',
      maxRounds: '3',
      executionTimeoutMS: '-1',
    })).toThrow('relay_default_execution_timeout_ms');
  });
});
