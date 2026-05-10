import type { BridgeConfig } from '@/lib/types';
import { buildRuntimeUpdate, createRuntimeFormState } from './runtimeSettingsForm';

function buildBridgeConfig(overrides: Partial<BridgeConfig> = {}): BridgeConfig {
  return {
    provider: 'custom',
    provider_type: 'custom',
    base_url: 'https://example.com/v1',
    model: 'gpt-5.4',
    chat_path: '/v1/chat',
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

describe('components/config/runtimeSettingsForm', () => {
  it('defaults assistant markdown toggle to true when config is absent', () => {
    const form = createRuntimeFormState(null);
    expect(form.maxTurns).toBe('20');
    expect(form.taskExecutionTimeoutMS).toBe('300000');
    expect(form.llmCompletionRetryCount).toBe('1');
    expect(form.llmCompletionRetryIntervalMS).toBe('200');
    expect(form.sessionSystemPromptVisibleEnabled).toBe(true);
    expect(form.assistantMarkdownEnabled).toBe(true);
    expect(form.toolCallCompactOutputEnabled).toBe(false);
    expect(form.memoryModeEnabled).toBe(false);
    expect(form.microcompactEnabled).toBe(false);
    expect(form.sessionTitleMode).toBe('session_id');
  });

  it('reads assistant markdown toggle from bridge config', () => {
    const form = createRuntimeFormState(buildBridgeConfig({
      session_system_prompt_visible_enabled: false,
      assistant_markdown_enabled: false,
      tool_call_compact_output_enabled: true,
      memory_mode_enabled: true,
      microcompact_enabled: true,
      session_title_mode: 'ai_generated',
      llm_completion_retry_count: 0,
      llm_completion_retry_interval_ms: 0,
    }));
    expect(form.llmCompletionRetryCount).toBe('0');
    expect(form.llmCompletionRetryIntervalMS).toBe('0');
    expect(form.sessionSystemPromptVisibleEnabled).toBe(false);
    expect(form.assistantMarkdownEnabled).toBe(false);
    expect(form.toolCallCompactOutputEnabled).toBe(true);
    expect(form.memoryModeEnabled).toBe(true);
    expect(form.microcompactEnabled).toBe(true);
    expect(form.sessionTitleMode).toBe('ai_generated');
  });

  it('includes assistant markdown toggle in config update payload', () => {
    const update = buildRuntimeUpdate(true, {
      provider: 'custom',
      model: 'gpt-5.4',
      chatPath: '/v1/chat',
      projectRoot: '/tmp/runtime-root',
      maxTurns: '9',
      taskExecutionTimeoutMS: '600000',
      llmCompletionRetryCount: '0',
      llmCompletionRetryIntervalMS: '250',
      sessionHumanLogFullEnabled: false,
      sessionSystemPromptVisibleEnabled: false,
      assistantMarkdownEnabled: false,
      toolCallCompactOutputEnabled: true,
      memoryModeEnabled: true,
      microcompactEnabled: true,
      sessionTitleMode: 'first_message',
      webSearchTavilyURL: '',
      webSearchExaURL: '',
      webSearchTavilyAPIKey: '',
      webSearchExaAPIKey: '',
    });

    expect(update.session_system_prompt_visible_enabled).toBe(false);
    expect(update.assistant_markdown_enabled).toBe(false);
    expect(update.project_root).toBe('/tmp/runtime-root');
    expect(update.max_turns).toBe(9);
    expect(update.task_execution_timeout_ms).toBe(600000);
    expect(update.llm_completion_retry_count).toBe(0);
    expect(update.llm_completion_retry_interval_ms).toBe(250);
    expect(update.tool_call_compact_output_enabled).toBe(true);
    expect(update.memory_mode_enabled).toBe(true);
    expect(update.microcompact_enabled).toBe(true);
    expect(update.session_title_mode).toBe('first_message');
    expect(update).not.toHaveProperty('api_key');
    expect(update).not.toHaveProperty('base_url');
  });
});
