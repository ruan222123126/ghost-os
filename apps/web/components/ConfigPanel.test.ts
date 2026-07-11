import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { ConfigPanel } from './ConfigPanel';

let mockProvidersMachine = buildProvidersMachine();
let mockTasksMachine = buildTasksMachine();

jest.mock('@/hooks/useConfigProviders', () => ({
  useConfigProviders: () => mockProvidersMachine,
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
  useConfigTasks: () => mockTasksMachine,
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
  beforeEach(() => {
    mockProvidersMachine = buildProvidersMachine();
    mockTasksMachine = buildTasksMachine();
  });

  it('keeps the same shell and content widths for prompts tabs', () => {
    const providerPanel = renderPanel('provider');
    const promptsPanel = renderPanel('prompts_library');

    expect(findByTestID(providerPanel.root, 'config-panel-shell').props.className).toContain('max-w-4xl');
    expect(findByTestID(promptsPanel.root, 'config-panel-shell').props.className).toContain('max-w-4xl');
    expect(findByTestID(providerPanel.root, 'config-panel-content').props.className).toContain('max-w-2xl');
    expect(findByTestID(promptsPanel.root, 'config-panel-content').props.className).toContain('max-w-2xl');
    expect(findByTestID(promptsPanel.root, 'config-panel-shell').props.className).not.toContain('max-w-[1320px]');
    expect(findByTestID(promptsPanel.root, 'config-panel-content').props.className).not.toContain('max-w-[1160px]');
  });

  it('renders task success feedback from the task machine', () => {
    mockTasksMachine = buildTasksMachine({ success: 'Run started successfully' });
    const renderer = renderPanel('tasks');

    expect(textContent(renderer.root)).toContain('Run started successfully');
  });
});

function renderPanel(initialTab: React.ComponentProps<typeof ConfigPanel>['initialTab']): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
      }, React.createElement(ConfigPanel, {
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
      ),
    );
  });

  return renderer;
}

function findByTestID(root: TestRenderer.ReactTestInstance, testID: string): TestRenderer.ReactTestInstance {
  return root.find((node) => node.props['data-testid'] === testID);
}

function buildProvidersMachine(overrides?: Record<string, unknown>) {
  return {
    state: {
      error: '',
      ...overrides,
    },
    actions: {
      cancelEditing: jest.fn(),
    },
  };
}

function buildTasksMachine(overrides?: Record<string, unknown>) {
  return {
    state: {
      error: '',
      success: '',
      ...overrides,
    },
    actions: {
      cancelEditing: jest.fn(),
    },
  };
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
