import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { RuntimeSettingsSection } from '@/components/config/RuntimeSettingsSection';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { BridgeConfig } from '@/lib/types';

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

describe('components/config/RuntimeSettingsSection', () => {
  it('renders common settings first without lined section gutters', () => {
    const renderer = renderSection(buildBridgeConfig());
    const sections = renderer.root.findAll((node) => (
      node.type === 'section'
      && typeof node.props.className === 'string'
      && node.props.className.includes('border-b')
    ));

    expect(textContent(sections[0])).toContain('Common Settings');
    expect(textContent(sections[0])).toContain('Language');
    expect(textContent(sections[0])).toContain('Max Turns');
    expect(textContent(sections[0])).toContain('Task Total Timeout (ms)');
    expect(renderer.root.findAll((node) => (
      typeof node.props.className === 'string'
      && node.props.className.includes('border-l-2')
    ))).toHaveLength(0);
  });
});

function renderSection(config: BridgeConfig): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      // eslint-disable-next-line react/no-children-prop
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(RuntimeSettingsSection, {
          loading: false,
          saving: false,
          config,
          onSave: async () => false,
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
