import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { createRuntimeFormState } from '@/components/config/runtimeSettingsForm';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { BridgeConfig } from '@/lib/types';
import { RuntimeCoreSection } from './runtimeSettingsSections';

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

describe('components/config/runtimeSettingsSections', () => {
  it('keeps provider api key and base url out of general runtime settings', () => {
    const config = buildBridgeConfig();
    const renderer = renderRuntimeCore(config);
    const content = textContent(renderer.root);

    expect(content).toContain('Provider Name');
    expect(content).toContain('Model');
    expect(content).toContain('Max Turns');
    expect(content).toContain('Task Execution Timeout (ms)');
    expect(content).toContain('LLM Retry Count');
    expect(content).toContain('LLM Retry Interval (ms)');
    expect(content).not.toContain('Provider API Key');
    expect(content).not.toContain('Base URL');
  });
});

function renderRuntimeCore(config: BridgeConfig): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      // eslint-disable-next-line react/no-children-prop
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(RuntimeCoreSection, {
          formState: createRuntimeFormState(config),
          controlsDisabled: false,
          modelSelectionEnabled: true,
          onChange: () => undefined,
          config,
        }),
      }),
    );
  });

  return renderer;
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
