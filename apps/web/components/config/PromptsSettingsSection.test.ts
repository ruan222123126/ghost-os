import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { SystemPromptPayload } from '@/lib/types';
import { PromptsSettingsSection } from './PromptsSettingsSection';

describe('components/config/PromptsSettingsSection', () => {
  const samplePrompts: SystemPromptPayload = {
    global_template: '{{base_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}',
    core_prompt: 'core guidance',
    tool_prompt: 'tool guidance',
    tool_key_spec: 'json key rules',
    rendered_prompt: 'base prompt with core guidance\ntool guidance\njson key rules',
  };

  it('renders loading state and empty preview when prompts are absent', () => {
    const renderer = renderSection({
      prompts: null,
      loading: true,
      saving: false,
      onRefresh: async () => undefined,
      onSaveField: async () => undefined,
    });

    expect(textContent(renderer.root.findByType('h1'))).toBe('Prompts');
    expect(textContent(renderer.root.findAllByType('p')[0])).toContain('Edit the visible system prompt blocks.');
    expect(textContent(renderer.root)).toContain('Core Job');
    expect(textContent(renderer.root)).not.toContain('Core Prompt');
    expect(textContent(renderer.root.findByType('pre'))).toBe('Rendered prompt is empty.');
    const sectionTitles = renderer.root.findAllByType('h2').map(textContent);
    expect(sectionTitles).toEqual(['Rendered Prompt', 'Core Job']);
    expect(textContent(renderer.root)).not.toContain('Global Template');

    for (const button of renderer.root.findAllByType('button')) {
      expect(button.props.disabled).toBe(true);
    }
  });

  it('tracks dirty edits, reset, and save sync for prompt cards', async () => {
    const onRefresh = jest.fn(async () => undefined);
    const onSaveField = jest.fn(async () => undefined);
    const renderer = renderSection({
      prompts: samplePrompts,
      loading: false,
      saving: false,
      onRefresh,
      onSaveField,
    });

    expect(textContent(renderer.root.findByType('pre'))).toBe(samplePrompts.rendered_prompt);

    const initialButtons = renderer.root.findAllByType('button');
    const initialTextareas = renderer.root.findAllByType('textarea');
    expect(initialTextareas).toHaveLength(1);
    expect(initialButtons[0].props.disabled).toBe(false);
    expect(initialButtons[1].props.disabled).toBe(true);
    expect(initialButtons[2].props.disabled).toBe(true);
    expect(textContent(renderer.root)).not.toContain('Global Template');
    expect(textContent(renderer.root)).not.toContain('Tool Prompt');
    expect(textContent(renderer.root)).not.toContain('Tool Key Spec');

    await act(async () => {
      initialTextareas[0].props.onChange({
        target: { value: 'updated core job' },
      });
    });

    const dirtyTextareas = renderer.root.findAllByType('textarea');
    const dirtyButtons = renderer.root.findAllByType('button');
    expect(dirtyTextareas[0].props.value).toBe('updated core job');
    expect(dirtyButtons[1].props.disabled).toBe(false);
    expect(dirtyButtons[2].props.disabled).toBe(false);

    await act(async () => {
      dirtyButtons[2].props.onClick();
    });

    const resetTextareas = renderer.root.findAllByType('textarea');
    const resetButtons = renderer.root.findAllByType('button');
    expect(resetTextareas[0].props.value).toBe(samplePrompts.core_prompt);
    expect(resetButtons[1].props.disabled).toBe(true);
    expect(resetButtons[2].props.disabled).toBe(true);

    await act(async () => {
      resetTextareas[0].props.onChange({
        target: { value: 'updated core job again' },
      });
    });

    const saveButtons = renderer.root.findAllByType('button');

    await act(async () => {
      saveButtons[1].props.onClick();
    });

    expect(onSaveField).toHaveBeenCalledWith('core_prompt', 'updated core job again');

    const updatedPrompts: SystemPromptPayload = {
      ...samplePrompts,
      core_prompt: 'updated core job again',
      rendered_prompt: 'base prompt with updated core job again\ntool guidance\njson key rules',
    };

    await act(async () => {
      renderer.update(renderSectionNode({
        prompts: updatedPrompts,
        loading: false,
        saving: false,
        onRefresh,
        onSaveField,
      }));
    });

    expect(renderer.root.findAllByType('textarea')[0].props.value).toBe('updated core job again');
    expect(renderer.root.findAllByType('button')[1].props.disabled).toBe(true);
    expect(renderer.root.findAllByType('button')[2].props.disabled).toBe(true);
  });
});

function renderSection(props: React.ComponentProps<typeof PromptsSettingsSection>): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(renderSectionNode(props));
  });

  return renderer;
}

function renderSectionNode(props: React.ComponentProps<typeof PromptsSettingsSection>) {
  return React.createElement(
    WebLocaleProvider,
    {
      initialLocale: 'en-US',
      children: React.createElement(PromptsSettingsSection, props),
    },
  );
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
