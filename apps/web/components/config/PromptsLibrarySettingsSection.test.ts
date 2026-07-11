import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { PromptLibraryItem, SystemPromptPayload } from '@/lib/types';
import { PromptsLibrarySettingsSection } from './PromptsLibrarySettingsSection';

describe('components/config/PromptsLibrarySettingsSection', () => {
  const samplePrompts: SystemPromptPayload = {
    core_prompt: 'card b content',
    rendered_prompt: 'rendered prompt',
    prompt_library: [
      {
        id: 'card-a',
        name: 'Card A',
        insert_point: 'core_job',
        content: 'card a content',
        active: false,
      },
      {
        id: 'card-b',
        name: 'Card B',
        insert_point: 'core_job',
        content: 'card b content',
        active: true,
      },
    ],
    tool_definitions: [],
  };

  it('renders card list and disables actions when prompts are absent', () => {
    const renderer = renderSection({
      prompts: null,
      loading: true,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary: async () => undefined,
    });

    expect(textContent(renderer.root.findByType('h1'))).toBe('Library');
    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'prompt-library-card-0')).toHaveLength(0);
    expect(findByTestID(renderer.root, 'prompt-library-new-card').props.disabled).toBe(true);
  });

  it('activates cards with same insert_point mutually exclusively', async () => {
    const onSavePromptLibrary = jest.fn<Promise<void>, [PromptLibraryItem[]]>(async () => undefined);
    const renderer = renderSection({
      prompts: samplePrompts,
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary,
    });

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-toggle-0').props.onClick();
    });

    expect(onSavePromptLibrary).toHaveBeenCalledWith([
      {
        id: 'card-a',
        name: 'Card A',
        insert_point: 'core_job',
        content: 'card a content',
        active: true,
      },
      {
        id: 'card-b',
        name: 'Card B',
        insert_point: 'core_job',
        content: 'card b content',
        active: false,
      },
    ]);
  });

  it('keeps multiple active context cards and appends newly activated card to the saved context order', async () => {
    const onSavePromptLibrary = jest.fn<Promise<void>, [PromptLibraryItem[]]>(async () => undefined);
    const renderer = renderSection({
      prompts: {
        ...samplePrompts,
        prompt_library: [
          {
            id: 'context-a',
            name: 'Context A',
            insert_point: 'context',
            content: 'context a',
            active: true,
          },
          {
            id: 'rule-card',
            name: 'Rule Card',
            insert_point: 'rule',
            content: 'rule content',
            active: true,
          },
          {
            id: 'context-b',
            name: 'Context B',
            insert_point: 'context',
            content: 'context b',
            active: false,
          },
        ],
      },
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary,
    });

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-toggle-2').props.onClick();
    });

    expect(onSavePromptLibrary).toHaveBeenCalledWith([
      {
        id: 'context-a',
        name: 'Context A',
        insert_point: 'context',
        content: 'context a',
        active: true,
      },
      {
        id: 'context-b',
        name: 'Context B',
        insert_point: 'context',
        content: 'context b',
        active: true,
      },
      {
        id: 'rule-card',
        name: 'Rule Card',
        insert_point: 'rule',
        content: 'rule content',
        active: true,
      },
    ]);
  });

  it('supports create/edit/delete flow and exposes insert_point selector in editor', async () => {
    const onSavePromptLibrary = jest.fn<Promise<void>, [PromptLibraryItem[]]>(async () => undefined);
    const renderer = renderSection({
      prompts: samplePrompts,
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary,
    });

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-new-card').props.onClick();
    });
    expect(textContent(findByTestID(renderer.root, 'prompt-card-editor-title'))).toBe('Create Prompt Card');
    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'prompt-card-reset')).toHaveLength(0);
    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'prompt-card-delete')).toHaveLength(0);
    const insertPointSelect = findByTestID(renderer.root, 'prompt-card-insert-point-select');
    expect(insertPointSelect.props.value).toBe('core_job');
    const insertPointOptions = insertPointSelect.findAllByType('option').map((node) => String(node.props.value));
    expect(insertPointOptions).toContain('rule');
    expect(insertPointOptions).toContain('core_job');
    expect(insertPointOptions).toContain('context');
    expect(insertPointOptions).not.toContain('memory');

    await act(async () => {
      findByTestID(renderer.root, 'prompt-card-name-input').props.onChange({ target: { value: 'New Card' } });
      findByTestID(renderer.root, 'prompt-card-content-input').props.onChange({ target: { value: 'new content' } });
    });
    await act(async () => {
      findByTestID(renderer.root, 'prompt-card-save').props.onClick();
    });

    const createCall = onSavePromptLibrary.mock.calls[0][0] as PromptLibraryItem[];
    expect(createCall).toHaveLength(3);
    expect(createCall[2]).toMatchObject({
      name: 'New Card',
      insert_point: 'core_job',
      content: 'new content',
      active: false,
    });
    expect(createCall[2].id).not.toBe('');

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-edit-0').props.onClick();
    });
    expect(textContent(findByTestID(renderer.root, 'prompt-card-editor-title'))).toBe('Edit Prompt Card');
    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'prompt-card-reset')).toHaveLength(0);
    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'prompt-card-delete')).toHaveLength(1);

    await act(async () => {
      findByTestID(renderer.root, 'prompt-card-name-input').props.onChange({ target: { value: 'Edited A' } });
      findByTestID(renderer.root, 'prompt-card-content-input').props.onChange({ target: { value: 'edited content' } });
    });
    await act(async () => {
      findByTestID(renderer.root, 'prompt-card-save').props.onClick();
    });
    expect(onSavePromptLibrary.mock.calls[1][0]).toEqual([
      {
        id: 'card-a',
        name: 'Edited A',
        insert_point: 'core_job',
        content: 'edited content',
        active: false,
      },
      {
        id: 'card-b',
        name: 'Card B',
        insert_point: 'core_job',
        content: 'card b content',
        active: true,
      },
    ]);

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-delete-0').props.onClick();
    });
    expect(onSavePromptLibrary.mock.calls[2][0]).toEqual([
      {
        id: 'card-b',
        name: 'Card B',
        insert_point: 'core_job',
        content: 'card b content',
        active: true,
      },
    ]);
  });

  it('keeps prompt library action buttons on one line and hides memory from new-card insert points', async () => {
    const renderer = renderSection({
      prompts: {
        ...samplePrompts,
        prompt_library: [
          {
            id: 'memory-card',
            name: 'Memory Card',
            insert_point: 'memory',
            content: 'memory content',
            active: true,
          },
        ],
      },
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary: async () => undefined,
    }, 'zh-CN');

    expect(findByTestID(renderer.root, 'prompt-library-toggle-0').props.className).toContain('whitespace-nowrap');
    expect(findByTestID(renderer.root, 'prompt-library-edit-0').props.className).toContain('whitespace-nowrap');
    expect(findByTestID(renderer.root, 'prompt-library-delete-0').props.className).toContain('whitespace-nowrap');

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-new-card').props.onClick();
    });

    const insertPointOptions = findByTestID(renderer.root, 'prompt-card-insert-point-select')
      .findAllByType('option')
      .map((node) => ({
        value: String(node.props.value),
        label: textContent(node),
      }));

    expect(insertPointOptions).toContainEqual({ value: 'context', label: 'context' });
    expect(insertPointOptions).toContainEqual({ value: 'rule', label: 'rule' });
    expect(insertPointOptions).not.toContainEqual({ value: 'memory', label: 'memory' });
  });

  it('keeps legacy memory insert_point visible while editing an existing memory card', async () => {
    const renderer = renderSection({
      prompts: {
        ...samplePrompts,
        prompt_library: [
          {
            id: 'memory-card',
            name: 'Memory Card',
            insert_point: 'memory',
            content: 'memory content',
            active: true,
          },
        ],
      },
      loading: false,
      saving: false,
      onRefresh: async () => undefined,
      onSavePromptLibrary: async () => undefined,
    }, 'zh-CN');

    await act(async () => {
      findByTestID(renderer.root, 'prompt-library-edit-0').props.onClick();
    });

    const insertPointOptions = findByTestID(renderer.root, 'prompt-card-insert-point-select')
      .findAllByType('option')
      .map((node) => ({
        value: String(node.props.value),
        label: textContent(node),
      }));

    expect(insertPointOptions).toContainEqual({ value: 'memory', label: 'memory' });
  });
});

function renderSection(
  props: React.ComponentProps<typeof PromptsLibrarySettingsSection>,
  locale: 'en-US' | 'zh-CN' = 'en-US',
): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(renderSectionNode(props, locale));
  });

  return renderer;
}

function renderSectionNode(
  props: React.ComponentProps<typeof PromptsLibrarySettingsSection>,
  locale: 'en-US' | 'zh-CN',
) {
  // eslint-disable-next-line react/no-children-prop
  return React.createElement(
    WebLocaleProvider,
    {
      initialLocale: locale,
      children: React.createElement(PromptsLibrarySettingsSection, props),
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
