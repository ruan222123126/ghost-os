import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { ConfigPanelSectionContent } from './ConfigPanelSectionContent';

jest.mock('@/components/config/ProviderSettingsSection', () => ({
  ProviderSettingsSection: () => React.createElement('div', { 'data-testid': 'provider-settings-section' }),
}));

jest.mock('@/components/config/RuntimeSettingsSection', () => ({
  RuntimeSettingsSection: () => React.createElement('div', { 'data-testid': 'runtime-settings-section' }),
}));

jest.mock('@/components/config/LoopSettingsSection', () => ({
  LoopSettingsSection: () => React.createElement('div', { 'data-testid': 'loop-settings-section' }),
}));

jest.mock('@/components/config/TaskSettingsSection', () => ({
  TaskSettingsSection: () => React.createElement('div', { 'data-testid': 'task-settings-section' }),
}));

jest.mock('@/components/config/OrchestrationSettingsSection', () => ({
  OrchestrationSettingsSection: () => React.createElement('div', { 'data-testid': 'orchestration-settings-section' }),
}));

jest.mock('@/components/config/SkillSettingsSection', () => ({
  SkillSettingsSection: () => React.createElement('div', { 'data-testid': 'skill-settings-section' }),
}));

jest.mock('@/components/config/ToolSettingsSection', () => ({
  ToolSettingsSection: () => React.createElement('div', { 'data-testid': 'tool-settings-section' }),
}));

jest.mock('@/components/config/PresetSettingsSection', () => ({
  PresetSettingsSection: () => React.createElement('div', { 'data-testid': 'preset-settings-section' }),
}));

jest.mock('@/components/config/PromptsLibrarySettingsSection', () => ({
  PromptsLibrarySettingsSection: () => React.createElement('div', { 'data-testid': 'prompts-library-settings-section' }),
}));

jest.mock('@/components/config/PromptsPreviewSettingsSection', () => ({
  PromptsPreviewSettingsSection: () => React.createElement('div', { 'data-testid': 'prompts-preview-settings-section' }),
}));

describe('components/config/ConfigPanelSectionContent', () => {
  it('renders the orchestration section only for the orchestration tab', () => {
    const orchestrationRenderer = renderSection('orchestration');
    expect(findByTestID(orchestrationRenderer.root, 'orchestration-settings-section')).toBeDefined();
    expect(orchestrationRenderer.root.findAll((node) => node.props['data-testid'] === 'task-settings-section')).toHaveLength(0);

    const tasksRenderer = renderSection('tasks');
    expect(findByTestID(tasksRenderer.root, 'task-settings-section')).toBeDefined();
    expect(tasksRenderer.root.findAll((node) => node.props['data-testid'] === 'orchestration-settings-section')).toHaveLength(0);

    const relayRenderer = renderSection('relay');
    expect(findByTestID(relayRenderer.root, 'loop-settings-section')).toBeDefined();
    expect(relayRenderer.root.findAll((node) => node.props['data-testid'] === 'task-settings-section')).toHaveLength(0);
  });
});

function renderSection(activeTab: React.ComponentProps<typeof ConfigPanelSectionContent>['activeTab']): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(ConfigPanelSectionContent, buildProps(activeTab)),
    );
  });

  return renderer;
}

function buildProps(
  activeTab: React.ComponentProps<typeof ConfigPanelSectionContent>['activeTab'],
): React.ComponentProps<typeof ConfigPanelSectionContent> {
  return {
    activeTab,
    loading: false,
    saving: false,
    config: null,
    onSave: async () => false,
    onRefreshConfig: async () => undefined,
    onOpenWorkflowCreate: () => undefined,
    onOpenWorkflowEdit: () => undefined,
    providersState: {} as React.ComponentProps<typeof ConfigPanelSectionContent>['providersState'],
    presetsState: {} as React.ComponentProps<typeof ConfigPanelSectionContent>['presetsState'],
    promptsState: {} as React.ComponentProps<typeof ConfigPanelSectionContent>['promptsState'],
    skillsState: {} as React.ComponentProps<typeof ConfigPanelSectionContent>['skillsState'],
    tasksState: {
      tasks: [],
      tasksLoading: false,
      taskSaving: false,
      taskError: '',
    } as unknown as React.ComponentProps<typeof ConfigPanelSectionContent>['tasksState'],
    toolsState: {} as React.ComponentProps<typeof ConfigPanelSectionContent>['toolsState'],
  };
}

function findByTestID(root: TestRenderer.ReactTestInstance, testID: string): TestRenderer.ReactTestInstance {
  return root.find((node) => node.props['data-testid'] === testID);
}
