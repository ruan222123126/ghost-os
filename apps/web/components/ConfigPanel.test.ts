import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { ConfigPanel } from './ConfigPanel';

jest.mock('@/hooks/useConfigProviders', () => ({
  useConfigProviders: () => ({
    providerError: '',
    cancelEditing: () => undefined,
  }),
}));

jest.mock('@/hooks/useConfigPrompts', () => ({
  useConfigPrompts: () => ({
    promptError: '',
  }),
}));

jest.mock('@/hooks/useConfigPresets', () => ({
  useConfigPresets: () => ({
    presetError: '',
  }),
}));

jest.mock('@/hooks/useConfigSkills', () => ({
  useConfigSkills: () => ({
    skillError: '',
  }),
}));

jest.mock('@/hooks/useConfigTasks', () => ({
  useConfigTasks: () => ({
    taskError: '',
    cancelEditing: () => undefined,
  }),
}));

jest.mock('@/hooks/useConfigTools', () => ({
  useConfigTools: () => ({
    toolError: '',
  }),
}));

jest.mock('@/components/config/ConfigPanelSectionContent', () => ({
  ConfigPanelSectionContent: () => React.createElement('div', { 'data-testid': 'config-panel-section-content' }),
}));

describe('components/ConfigPanel', () => {
  it('keeps the same shell and content widths for prompts tabs', () => {
    const providerPanel = renderPanel('provider');
    const promptsPanel = renderPanel('prompts_library');

    expect(findByTestID(providerPanel.root, 'config-panel-shell').props.className).toContain('max-w-[1000px]');
    expect(findByTestID(promptsPanel.root, 'config-panel-shell').props.className).toContain('max-w-[1000px]');
    expect(findByTestID(providerPanel.root, 'config-panel-content').props.className).toContain('max-w-2xl');
    expect(findByTestID(promptsPanel.root, 'config-panel-content').props.className).toContain('max-w-2xl');
    expect(findByTestID(promptsPanel.root, 'config-panel-shell').props.className).not.toContain('max-w-[1320px]');
    expect(findByTestID(promptsPanel.root, 'config-panel-content').props.className).not.toContain('max-w-[1160px]');
  });
});

function renderPanel(initialTab: React.ComponentProps<typeof ConfigPanel>['initialTab']): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(ConfigPanel, {
          open: true,
          initialTab,
          loading: false,
          saving: false,
          config: null,
          error: '',
          onClose: () => undefined,
          onOpenWorkflowCreate: () => undefined,
          onOpenWorkflowEdit: () => undefined,
          onSave: async () => false,
          onReload: async () => undefined,
        }),
      }),
    );
  });

  return renderer;
}

function findByTestID(root: TestRenderer.ReactTestInstance, testID: string): TestRenderer.ReactTestInstance {
  return root.find((node) => node.props['data-testid'] === testID);
}
