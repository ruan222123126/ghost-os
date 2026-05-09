import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { defaultOrchestrationGroupSharedContext } from '@/lib/orchestration-editor/groupDefaults';
import type { PresetPayload, ProviderConfig, ToolPayload } from '@/lib/types';
import type { WorkflowAgentRuntimeCatalog, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor';
import { WorkflowCanvasNodeEditorContent } from './WorkflowCanvasNodeEditorContent';

jest.mock('@/hooks/workflow/useWorkflowToolOptions', () => ({
  useWorkflowToolOptions: () => ({
    options: [
      {
        name: 'script_exec',
        label: 'script_exec',
        disabled: false,
        inputSchema: {
          type: 'object',
          properties: {
            query: { type: 'string', description: 'query text' },
          },
        },
      },
    ],
    loading: false,
    error: '',
  }),
}));

describe('components/workflow/WorkflowCanvasNodeEditorContent', () => {
  it('renders autocomplete textareas for workflow editable text fields', () => {
    const renderer = renderEditor({
      editorKind: 'workflow',
      selectedNode: createNode('agent-node', 'agent', {
        agent: { message: 'hello' },
      }),
    });

    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(2);
    expect(renderer.root.findAllByType('textarea')).toHaveLength(2);
  });

  it('renders plain textareas for orchestration editable text fields', () => {
    const renderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: createNode('agent-node', 'agent', {
        agent: { message: 'hello' },
      }),
    });

    expect(renderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(0);
    expect(renderer.root.findAllByType('textarea')).toHaveLength(2);
  });

  it('mounts orchestration presets by syncing tools and disabling manual system prompt', () => {
    const onUpdateNode = jest.fn();
    const renderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: createNode('agent-node', 'agent', {
        agent: {
          title: 'Alpha',
          message: 'hello',
          runtime_overrides: {
            system_prompt: 'manual prompt',
          },
        },
      }),
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [],
        tools: [createTool('ghost_tool', false), createTool('script_exec', true)],
      },
      presets: [createPreset('preset-a', ['ghost_tool'])],
      onUpdateNode,
    });

    expect(textContent(renderer.root)).toContain('ghost_tool');

    const presetSelect = renderer.root.findByProps({ 'data-testid': 'orchestration-agent-preset-select' });
    act(() => {
      presetSelect.props.onChange({ target: { value: 'preset-a' } });
    });

    const updatedNode = onUpdateNode.mock.calls.at(-1)?.[0] as WorkflowCanvasNodeDraft;
    expect(updatedNode.agent?.runtime_overrides).toEqual({
      preset_id: 'preset-a',
      tool_allowlist_only: true,
      tool_allowlist: ['ghost_tool'],
    });

    const updatedRenderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: updatedNode,
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [],
        tools: [createTool('ghost_tool', false), createTool('script_exec', true)],
      },
      presets: [createPreset('preset-a', ['ghost_tool'])],
    });

    const systemPromptField = updatedRenderer.root.findAllByType('textarea')[1];
    expect(systemPromptField.props.disabled).toBe(true);
  });

  it('renders autocomplete field for if compare value only in workflow mode', () => {
    const workflowRenderer = renderEditor({
      editorKind: 'workflow',
      selectedNode: createNode('if-node', 'if', {
        if: {
          source_node_id: 'tool-a',
          operator: 'equals',
          value: '',
          true_node_id: 'end',
          false_node_id: 'end',
        },
      }),
    });
    const orchestrationRenderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: createNode('if-node', 'if', {
        if: {
          source_node_id: 'tool-a',
          operator: 'equals',
          value: '',
          true_node_id: 'end',
          false_node_id: 'end',
        },
      }),
    });

    expect(workflowRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(1);
    expect(orchestrationRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(0);
  });

  it('renders autocomplete tool schema inputs only in workflow mode', () => {
    const workflowRenderer = renderEditor({
      editorKind: 'workflow',
      selectedNode: createNode('tool-node', 'tool', {
        tool: {
          tool_name: 'script_exec',
          arguments: { query: '' },
        },
      }),
    });
    const orchestrationRenderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: createNode('tool-node', 'tool', {
        tool: {
          tool_name: 'script_exec',
          arguments: { query: '' },
        },
      }),
    });

    expect(workflowRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(1);
    expect(orchestrationRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete')).toHaveLength(0);
  });

  it('keeps invalid provider/model options visible as disabled entries', () => {
    const renderer = renderEditor({
      editorKind: 'workflow',
      selectedNode: createNode('agent-node', 'agent', {
        agent: {
          message: 'hello',
          runtime_overrides: {
            provider_name: 'missing-provider',
            model: 'missing-model',
            tool_allowlist_only: true,
            tool_allowlist: ['ghost_tool'],
          },
        },
      }),
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [createProvider()],
        tools: [createTool('script_exec', true), createTool('ghost_tool', false)],
      },
    });

    const options = renderer.root.findAllByType('option');
    expect(options.some((node) => node.props.value === 'missing-provider' && node.props.disabled)).toBe(true);
    expect(options.some((node) => node.props.value === 'missing-model' && node.props.disabled)).toBe(true);
  });

  it('renders group owner prompt copy for orchestration group editors', () => {
    const renderer = renderEditor({
      editorKind: 'orchestration',
      draft: {
        mode: 'edit',
        schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
        nodes: [
          createNode('group-node', 'group', {
            group: {
              title: '群组 1',
              shared_context: '',
              speaking_mode: 'owner',
              owner_agent_id: 'agent-1',
              max_rounds: 3,
            },
          }),
          createNode('agent-1', 'agent', {
            agent: { title: 'Owner', message: 'dispatch' },
          }),
        ],
        edges: [
          { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-node', kind: 'member' },
        ],
      },
      selectedNode: createNode('group-node', 'group', {
        group: {
          title: '群组 1',
          shared_context: '',
          speaking_mode: 'owner',
          owner_agent_id: 'agent-1',
          max_rounds: 3,
        },
      }),
    });

    expect(textContent(renderer.root)).toContain('Group Owner Initial Prompt');
    expect(textContent(renderer.root)).toContain('This text is injected to every member as the group owner opening prompt and shared context.');
    expect(textContent(renderer.root)).toContain('Owner Member');
    expect(textContent(renderer.root)).toContain('The owner gets a runtime-only orchestration_dispatch tool');
    expect(renderer.root.findByType('textarea').props.placeholder).toBe(defaultOrchestrationGroupSharedContext('en-US'));
  });
});

function renderEditor(input: {
  editorKind: 'workflow' | 'orchestration';
  selectedNode: WorkflowCanvasNodeDraft;
  draft?: {
    mode: 'create' | 'edit';
    schedule: { mode: 'interval' | 'cron'; intervalSeconds: string; cronExpr: string };
    nodes: WorkflowCanvasNodeDraft[];
    edges: Array<{ id: string; from_node_id: string; to_node_id: string; kind?: 'control' | 'member' }>;
  };
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  presets?: PresetPayload[];
  onUpdateNode?: jest.Mock;
}) {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(
        WebLocaleProvider,
        { initialLocale: 'zh-CN' } as React.ComponentProps<typeof WebLocaleProvider>,
        React.createElement(WorkflowCanvasNodeEditorContent, {
          editorKind: input.editorKind,
          draft: input.draft,
          selectedNode: input.selectedNode,
          agentRuntimeCatalog: input.agentRuntimeCatalog,
          agentRuntimeLoading: false,
          agentRuntimeError: '',
          presets: input.presets,
          presetLoading: false,
          presetError: '',
          onUpdateNode: input.onUpdateNode ?? jest.fn(),
        }),
      ),
    );
  });

  return renderer;
}

function createNode(
  id: string,
  type: WorkflowCanvasNodeDraft['type'],
  patch: Partial<WorkflowCanvasNodeDraft> = {},
): WorkflowCanvasNodeDraft {
  return {
    id,
    type,
    position: { x: 0, y: 0 },
    ui: { toolArgumentsMode: 'kv' },
    ...patch,
  };
}

function createProvider(): ProviderConfig {
  return {
    name: 'openai-main',
    type: 'openai',
    base_url: 'https://api.openai.com/v1',
    models: ['gpt-5.4'],
    api_key_set: true,
  };
}

function createTool(name: string, enabled: boolean): ToolPayload {
  return {
    name,
    enabled,
  };
}

function createPreset(id: string, tool_allowlist: string[]): PresetPayload {
  return {
    id,
    name: id,
    tool_allowlist,
    prompt_refs: {},
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
