import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { SettingsNavigation } from './ConfigPanelNavigation';

describe('components/config/ConfigPanelNavigation', () => {
  it('renders all settings entries in the single system menu', () => {
    const renderer = renderNavigation();

    const systemGroup = findByTestID(renderer.root, 'settings-nav-group-system');
    const systemText = textContent(systemGroup);

    expect(systemText).toContain('General');
    expect(systemText).toContain('Provider');
    expect(systemText).toContain('Tasks');
    expect(systemText).toContain('Orchestration');
    expect(systemText).toContain('Skills');
    expect(systemText).toContain('Tools');
    expect(systemText).toContain('Presets');
    expect(systemText).toContain('Prompts');
    expect(systemText).toContain('Library');
    expect(systemText).toContain('Preview');
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
