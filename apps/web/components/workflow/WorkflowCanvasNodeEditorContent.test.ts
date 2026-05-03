import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { WorkflowCanvasDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor';
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
  it('renders variable autocomplete fields for workflow editable template fields', () => {
    const renderer = renderEditor({
      editorKind: 'workflow',
      selectedNode: createNode('agent-node', 'agent', {
        agent: { message: 'hello' },
      }),
    });

    expect(
      renderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete'),
    ).toHaveLength(1);
    expect(renderer.root.findAllByType('textarea')).toHaveLength(1);
  });

  it('keeps plain inputs for orchestration mode', () => {
    const renderer = renderEditor({
      editorKind: 'orchestration',
      selectedNode: createNode('agent-node', 'agent', {
        agent: { message: 'hello' },
      }),
    });

    expect(
      renderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete'),
    ).toHaveLength(0);
    expect(renderer.root.findAllByType('textarea')).toHaveLength(1);
  });

  it('uses variable autocomplete for if compare value and tool schema values only in workflow mode', () => {
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

    expect(
      workflowRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete'),
    ).toHaveLength(1);
  });

  it('renders tool schema value autocomplete in workflow mode but not orchestration mode', () => {
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

    expect(
      workflowRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete'),
    ).toHaveLength(1);
    expect(
      orchestrationRenderer.root.findAll((node) => node.props['data-testid'] === 'workflow-variable-autocomplete'),
    ).toHaveLength(0);
  });
});

function renderEditor(input: {
  editorKind: 'workflow' | 'orchestration';
  selectedNode: WorkflowCanvasNodeDraft;
}) {
  let renderer!: TestRenderer.ReactTestRenderer;
  const draft = createDraft(input.selectedNode);

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'zh-CN',
        children: React.createElement(WorkflowCanvasNodeEditorContent, {
          editorKind: input.editorKind,
          draft,
          selectedNode: input.selectedNode,
          onUpdateNode: jest.fn(),
        }),
      }),
    );
  });

  return renderer;
}

function createDraft(selectedNode: WorkflowCanvasNodeDraft): WorkflowCanvasDraft {
  return {
    mode: 'edit',
    selectedNodeId: selectedNode.id,
    schedule: {
      mode: 'interval',
      intervalSeconds: '60',
      cronExpr: '',
    },
    nodes: [
      createNode('start-node', 'start', {
        start: {
          inputs: [
            { name: 'user_name', type: 'string', description: 'user name' },
          ],
        },
      }),
      createNode('tool-a', 'tool', {
        tool: {
          tool_name: 'script_exec',
          arguments: { query: '${inputs.user_name}' },
        },
      }),
      selectedNode,
      createNode('end-node', 'end'),
    ],
    edges: [
      { id: 'edge-1', from_node_id: 'start-node', to_node_id: 'tool-a' },
      { id: 'edge-2', from_node_id: 'tool-a', to_node_id: selectedNode.id },
      { id: 'edge-3', from_node_id: selectedNode.id, to_node_id: 'end-node' },
    ],
  };
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
