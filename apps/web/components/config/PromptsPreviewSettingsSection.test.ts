import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { SystemPromptPayload } from '@/lib/types';
import { PromptsPreviewSettingsSection } from './PromptsPreviewSettingsSection';

describe('components/config/PromptsPreviewSettingsSection', () => {
  it('renders read-only preview without textarea editors', () => {
    const prompts: SystemPromptPayload = {
      core_prompt: '## Role\nOperator',
      rendered_prompt: 'Rendered final prompt',
      prompt_library: [],
    };

    const renderer = renderSection({
      prompts,
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
    });

    expect(textContent(renderer.root.findByType('h1'))).toBe('Preview');
    expect(textContent(findByTestID(renderer.root, 'prompts-rendered-preview'))).toBe('Rendered final prompt');
    expect(renderer.root.findAllByType('textarea')).toHaveLength(0);
    expect(renderer.root.findAll((node) => String(node.props['data-testid'] ?? '').startsWith('prompt-save-'))).toHaveLength(0);
  });

  it('shows empty preview copy when rendered prompt is missing', () => {
    const renderer = renderSection({
      prompts: null,
      loading: true,
      saving: false,
      onRefresh: async () => undefined,
    });

    expect(textContent(findByTestID(renderer.root, 'prompts-rendered-preview'))).toBe('Rendered prompt is empty.');
    expect(renderer.root.findAllByType('textarea')).toHaveLength(0);
  });
});

function renderSection(props: React.ComponentProps<typeof PromptsPreviewSettingsSection>): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(renderSectionNode(props));
  });

  return renderer;
}

function renderSectionNode(props: React.ComponentProps<typeof PromptsPreviewSettingsSection>) {
  // eslint-disable-next-line react/no-children-prop
  return React.createElement(
    WebLocaleProvider,
    {
      initialLocale: 'en-US',
      children: React.createElement(PromptsPreviewSettingsSection, props),
    },
  );
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
