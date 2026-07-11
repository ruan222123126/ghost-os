import { copyForWorkflow } from '@/lib/i18n/messages/workflow';
import type { ProviderConfig, ToolPayload } from '@/lib/types';
import type { WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor';
import { buildModelOptions, buildToolOptions } from './workflowAgentNodeEditorState';

const copy = copyForWorkflow('en-US');

describe('components/workflow/workflowAgentNodeEditorState', () => {
  it('keeps the current model selectable when it is no longer provided by the catalog', () => {
    const options = buildModelOptions({
      catalog: buildCatalog({
        providers: [buildProvider({ name: 'main', models: ['gpt-5'] })],
      }),
      providerName: 'main',
      currentValue: 'legacy-model',
      copy,
    });

    expect(options).toEqual([
      { value: 'legacy-model', label: 'legacy-model (unavailable)', disabled: true },
      { value: 'gpt-5', label: 'gpt-5', disabled: false },
    ]);
  });

  it('filters disabled tools unless the editor asks to show all tools', () => {
    const tools: ToolPayload[] = [
      { name: 'enabled_tool', enabled: true },
      { name: 'disabled_tool', enabled: false },
    ];

    expect(buildToolOptions({
      tools,
      selectedNames: ['disabled_tool'],
      copy,
      showAllTools: false,
    })).toEqual([
      { value: 'disabled_tool', label: 'disabled_tool (unavailable)', disabled: true },
      { value: 'enabled_tool', label: 'enabled_tool', disabled: false },
    ]);

    expect(buildToolOptions({
      tools,
      selectedNames: ['disabled_tool'],
      copy,
      showAllTools: true,
    })).toEqual([
      { value: 'disabled_tool', label: 'disabled_tool', disabled: false },
      { value: 'enabled_tool', label: 'enabled_tool', disabled: false },
    ]);
  });
});

function buildCatalog(overrides: Partial<WorkflowAgentRuntimeCatalog> = {}): WorkflowAgentRuntimeCatalog {
  return {
    activeProvider: 'main',
    providers: [],
    tools: [],
    ...overrides,
  };
}

function buildProvider(overrides: Partial<ProviderConfig>): ProviderConfig {
  const name = overrides.name ?? 'main';
  return {
    name,
    type: overrides.type ?? 'openai',
    base_url: overrides.base_url ?? 'https://api.example.test/v1',
    provider_id: overrides.provider_id ?? name,
    updated_at: overrides.updated_at ?? '2026-01-01T00:00:00Z',
    models: overrides.models,
    api_key_set: overrides.api_key_set ?? true,
  };
}
