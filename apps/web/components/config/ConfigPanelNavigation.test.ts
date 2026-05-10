import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { SettingsNavigation } from './ConfigPanelNavigation';

describe('components/config/ConfigPanelNavigation', () => {
  it('renders orchestration and presets under system and library/preview only under prompts group', () => {
    const renderer = renderNavigation();

    const systemGroup = findByTestID(renderer.root, 'settings-nav-group-system');
    const promptsGroup = findByTestID(renderer.root, 'settings-nav-group-prompts');

    const systemText = textContent(systemGroup);
    const promptsText = textContent(promptsGroup);

    expect(systemText).toContain('General');
    expect(systemText).toContain('Provider');
    expect(systemText).toContain('Relay');
    expect(systemText).toContain('Tasks');
    expect(systemText).toContain('Orchestration');
    expect(systemText).toContain('Skills');
    expect(systemText).toContain('Tools');
    expect(systemText).toContain('Presets');
    expect(systemText).not.toContain('Library');
    expect(systemText).not.toContain('Preview');

    expect(promptsText).toContain('Library');
    expect(promptsText).toContain('Preview');
    expect(promptsText).not.toContain('Presets');
    expect(promptsText).not.toContain('General');
    expect(promptsText).not.toContain('Provider');
  });
});

function renderNavigation(): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      // eslint-disable-next-line react/no-children-prop
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(SettingsNavigation, {
          activeTab: 'provider',
          onSelectTab: () => undefined,
        }),
      }),
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
