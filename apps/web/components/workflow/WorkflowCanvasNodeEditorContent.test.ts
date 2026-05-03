import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { ProviderConfig, ToolPayload } from '@/lib/types';
import type { WorkflowAgentRuntimeCatalog, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor';
import { WorkflowCanvasNodeEditorContent } from './WorkflowCanvasNodeEditorContent';

jest.mock('@/components/workflow/useWorkflowToolOptions', () => ({
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
});

function renderEditor(input: {
  editorKind: 'workflow' | 'orchestration';
  selectedNode: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
}) {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'zh-CN',
        children: React.createElement(WorkflowCanvasNodeEditorContent, {
          editorKind: input.editorKind,
          selectedNode: input.selectedNode,
          agentRuntimeCatalog: input.agentRuntimeCatalog,
          agentRuntimeLoading: false,
          agentRuntimeError: '',
          onUpdateNode: jest.fn(),
        }),
      }),
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
