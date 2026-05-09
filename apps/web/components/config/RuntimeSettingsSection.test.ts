import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { RuntimeSettingsSection } from '@/components/config/RuntimeSettingsSection';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { BridgeConfig } from '@/lib/types';

jest.mock('@/hooks/useSessionSidebarGroupingPreference', () => ({
  useSessionSidebarGroupingPreference: () => ({
    enabled: true,
    setEnabled: () => undefined,
  }),
}));

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
    web_search_tavily_url: '',
    web_search_exa_url: '',
    web_search_tavily_api_key_set: false,
    web_search_exa_api_key_set: false,
    ...overrides,
  };
}

describe('components/config/RuntimeSettingsSection', () => {
  it('renders language and sidebar grouping with the shared lined section layout', () => {
    const renderer = renderSection(buildBridgeConfig());
    const languageSection = findSectionByTitle(renderer.root, 'Interface Language');
    const groupingSection = findSectionByTitle(renderer.root, 'Session History Grouping');

    expect(languageSection.props.className).toContain('border-b');
    expect(groupingSection.props.className).toContain('border-b');
    expect(findBorderLine(languageSection).props.className).toContain('border-l-2');
    expect(findBorderLine(groupingSection).props.className).toContain('border-l-2');
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

function findSectionByTitle(root: TestRenderer.ReactTestInstance, title: string): TestRenderer.ReactTestInstance {
  return root.find((node) => (
    node.type === 'section'
    && typeof node.props.className === 'string'
    && node.props.className.includes('border-b')
    && textContent(node).includes(title)
  ));
}

function findBorderLine(section: TestRenderer.ReactTestInstance): TestRenderer.ReactTestInstance {
  return section.find((node) => (
    typeof node.props.className === 'string'
    && node.props.className.includes('border-l-2')
  ));
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
