import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { BridgeConfig, PresetPayload, SystemPromptPayload, ToolPayload } from '@/lib/types';
import { PresetSettingsSection } from './PresetSettingsSection';

describe('components/config/PresetSettingsSection', () => {
  const config: BridgeConfig = {
    provider: 'openai',
    provider_type: 'openai',
    base_url: 'https://api.openai.com/v1',
    model: 'gpt-5.4',
    chat_path: '',
    project_root: '/tmp',
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
    session_system_prompt_visible_enabled: false,
    assistant_markdown_enabled: true,
    tool_call_compact_output_enabled: true,
    memory_mode_enabled: false,
    microcompact_enabled: false,
    session_title_mode: 'session_id',
    web_search_tavily_url: '',
    web_search_exa_url: '',
    web_search_tavily_api_key_set: false,
    web_search_exa_api_key_set: false,
  };
  const tools: ToolPayload[] = [
    { name: 'script_exec', enabled: true },
    { name: 'web_search', enabled: false },
    { name: 'sfind', enabled: true },
  ];
  const prompts: SystemPromptPayload = {
    core_prompt: 'core content',
    rendered_prompt: 'rendered prompt',
    prompt_library: [
      { id: 'rule-card', name: 'Rule Card', insert_point: 'rule', content: 'rule', active: true },
      { id: 'core-card', name: 'Core Card', insert_point: 'core_job', content: 'core', active: true },
      { id: 'context-a', name: 'Context A', insert_point: 'context', content: 'context a', active: true },
      { id: 'context-b', name: 'Context B', insert_point: 'context', content: 'context b', active: true },
    ],
    tool_definitions: [],
  };
  const presets: PresetPayload[] = [
    {
      id: 'preset-a',
      name: 'Default',
      tool_allowlist: ['script_exec'],
      prompt_refs: {
        rule: 'rule-card',
        core_job: 'core-card',
        context: ['context-b', 'context-a'],
      },
    },
  ];

  it('renders empty state and blocks save for empty name', async () => {
    const onCreatePreset = jest.fn<Promise<void>, [unknown]>(async () => undefined);
    const renderer = renderSection({
      presets: [],
      config,
      loading: false,
      saving: false,
      tools,
      toolsLoading: false,
      prompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset,
      onUpdatePreset: async () => undefined,
      onDeletePreset: async () => undefined,
      onActivatePreset: async () => undefined,
    });

    expect(textContent(renderer.root.findByType('h1'))).toBe('Presets');
    expect(textContent(renderer.root)).toContain('No presets configured yet.');

    await act(async () => {
      findByTestID(renderer.root, 'preset-new-card').props.onClick();
    });

    expect(findByTestID(renderer.root, 'preset-save').props.disabled).toBe(true);

    await act(async () => {
      findByTestID(renderer.root, 'preset-save').props.onClick?.();
    });

    expect(onCreatePreset).not.toHaveBeenCalled();
  });

  it('creates a preset with selected tools and ordered context refs', async () => {
    const onCreatePreset = jest.fn<Promise<void>, [unknown]>(async () => undefined);
    const renderer = renderSection({
      presets: [],
      config,
      loading: false,
      saving: false,
      tools,
      toolsLoading: false,
      prompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset,
      onUpdatePreset: async () => undefined,
      onDeletePreset: async () => undefined,
      onActivatePreset: async () => undefined,
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-new-card').props.onClick();
    });

    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'preset-prompt-ref-memory-select')).toHaveLength(0);

    await act(async () => {
      findByTestID(renderer.root, 'preset-name-input').props.onChange({ target: { value: 'Research' } });
      findByTestID(renderer.root, 'preset-tool-script_exec').props.onClick();
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-tool-web_search').props.onClick();
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-rule-select').props.onChange({ target: { value: 'rule-card' } });
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-b').props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-a').props.onChange({ target: { checked: true } });
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-save').props.onClick();
    });

    expect(onCreatePreset).toHaveBeenCalledWith({
      name: 'Research',
      tool_allowlist: ['script_exec', 'web_search'],
      prompt_refs: {
        rule: 'rule-card',
        context: ['context-b', 'context-a'],
      },
    });
  });

  it('edits an existing preset and filters prompt options per slot', async () => {
    const onUpdatePreset = jest.fn<Promise<void>, [string, unknown]>(async () => undefined);
    const renderer = renderSection({
      presets,
      config,
      loading: false,
      saving: false,
      tools,
      toolsLoading: false,
      prompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset: async () => undefined,
      onUpdatePreset,
      onDeletePreset: async () => undefined,
      onActivatePreset: async () => undefined,
    });

    expect(textContent(renderer.root)).toContain('rule: Rule Card');
    expect(textContent(renderer.root)).toContain('core_job: Core Card');
    expect(textContent(renderer.root)).toContain('context: Context B, Context A');

    await act(async () => {
      findByTestID(renderer.root, 'preset-card-edit-0').props.onClick();
    });

    const ruleOptions = findByTestID(renderer.root, 'preset-prompt-ref-rule-select')
      .findAllByType('option')
      .map((node) => textContent(node));
    expect(ruleOptions).toContain('Rule Card');
    expect(ruleOptions).not.toContain('Core Card');
    expect(textContent(renderer.root.findByProps({ 'data-testid': 'preset-prompt-ref-context-context-a' }).parent as TestRenderer.ReactTestInstance)).toContain('Context A');

    await act(async () => {
      findByTestID(renderer.root, 'preset-name-input').props.onChange({ target: { value: 'Updated' } });
      findByTestID(renderer.root, 'preset-tool-script_exec').props.onClick();
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-tool-sfind').props.onClick();
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-rule-select').props.onChange({ target: { value: '' } });
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-b').props.onChange({ target: { checked: false } });
    });
    await act(async () => {
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-a').props.onChange({ target: { checked: false } });
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-a').props.onChange({ target: { checked: true } });
      findByTestID(renderer.root, 'preset-prompt-ref-context-context-b').props.onChange({ target: { checked: true } });
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-save').props.onClick();
    });

    expect(onUpdatePreset).toHaveBeenCalledWith('preset-a', {
      name: 'Updated',
      tool_allowlist: ['sfind'],
      prompt_refs: {
        core_job: 'core-card',
        context: ['context-a', 'context-b'],
      },
    });
  });

  it('deletes a preset', async () => {
    const onDeletePreset = jest.fn<Promise<void>, [string]>(async () => undefined);
    const renderer = renderSection({
      presets,
      config,
      loading: false,
      saving: false,
      tools,
      toolsLoading: false,
      prompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset: async () => undefined,
      onUpdatePreset: async () => undefined,
      onDeletePreset,
      onActivatePreset: async () => undefined,
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-card-delete-0').props.onClick();
    });

    expect(onDeletePreset).toHaveBeenCalledWith('preset-a');
  });

  it('marks the matching preset as active and disables re-activation', () => {
    const activePrompts: SystemPromptPayload = {
      ...prompts,
      prompt_library: [
        prompts.prompt_library[0],
        prompts.prompt_library[1],
        prompts.prompt_library[3],
        prompts.prompt_library[2],
      ],
    };
    const renderer = renderSection({
      presets,
      config,
      loading: false,
      saving: false,
      tools: [
        { name: 'script_exec', enabled: true },
        { name: 'web_search', enabled: false },
        { name: 'sfind', enabled: false },
      ],
      toolsLoading: false,
      prompts: activePrompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset: async () => undefined,
      onUpdatePreset: async () => undefined,
      onDeletePreset: async () => undefined,
      onActivatePreset: async () => undefined,
    });

    expect(textContent(renderer.root)).toContain('Active');
    expect(findByTestID(renderer.root, 'preset-card-activate-0').props.disabled).toBe(true);
  });

  it('activates a preset from the card action', async () => {
    const onActivatePreset = jest.fn<Promise<void>, [string]>(async () => undefined);
    const renderer = renderSection({
      presets,
      config,
      loading: false,
      saving: false,
      tools,
      toolsLoading: false,
      prompts,
      promptsLoading: false,
      onRefresh: async () => undefined,
      onCreatePreset: async () => undefined,
      onUpdatePreset: async () => undefined,
      onDeletePreset: async () => undefined,
      onActivatePreset,
    });

    await act(async () => {
      findByTestID(renderer.root, 'preset-card-activate-0').props.onClick();
    });

    expect(onActivatePreset).toHaveBeenCalledWith('preset-a');
  });
});

function renderSection(
  props: React.ComponentProps<typeof PresetSettingsSection>,
  locale: 'en-US' | 'zh-CN' = 'en-US',
): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: locale,
      }, React.createElement(PresetSettingsSection, props)),
    );
  });

  return renderer;
}

function findByTestID(root: TestRenderer.ReactTestInstance, testID: string): TestRenderer.ReactTestInstance {
  return root.find((node) => node.props['data-testid'] === testID);
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
