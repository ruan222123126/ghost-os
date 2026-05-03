import {
  buildWorkflowVariableSuggestions,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor';

describe('lib/workflow-editor/variableSuggestions', () => {
  it('includes start variables builtins and only real upstream action nodes in a linear chain', () => {
    const draft = createDraft({
      nodes: [
        createNode('start-node', 'start', {
          start: {
            inputs: [
              { name: 'ticket_id', type: 'string', description: 'ticket identifier' },
            ],
          },
        }),
        createNode('tool-fetch', 'tool'),
        createNode('llm-summarize', 'llm'),
        createNode('agent-review', 'agent'),
        createNode('end-node', 'end'),
      ],
      edges: [
        createEdge('start-node', 'tool-fetch'),
        createEdge('tool-fetch', 'llm-summarize'),
        createEdge('llm-summarize', 'agent-review'),
        createEdge('agent-review', 'end-node'),
      ],
    });

    const options = buildWorkflowVariableSuggestions(draft, draft.nodes[3]);

    expect(options.map((option) => option.token)).toEqual([
      '${inputs.ticket_id}',
      '${last}',
      '${last_output}',
      '${last_node_id}',
      '${outputs.tool-fetch}',
      '${nodes.tool-fetch.output}',
      '${nodes.tool-fetch.text}',
      '${nodes.tool-fetch.status}',
      '${outputs.llm-summarize}',
      '${nodes.llm-summarize.output}',
      '${nodes.llm-summarize.text}',
      '${nodes.llm-summarize.status}',
    ]);
  });

  it('limits upstream suggestions to the selected node branch in a parallel graph', () => {
    const draft = createDraft({
      nodes: [
        createNode('start-node', 'start', {
          start: { inputs: [{ name: 'branch_input', type: 'string' }] },
        }),
        createNode('tool-branch-a', 'tool'),
        createNode('agent-branch-a', 'agent'),
        createNode('tool-branch-b', 'tool'),
        createNode('end-node', 'end'),
      ],
      edges: [
        createEdge('start-node', 'tool-branch-a'),
        createEdge('tool-branch-a', 'agent-branch-a'),
        createEdge('agent-branch-a', 'end-node'),
        createEdge('start-node', 'tool-branch-b'),
        createEdge('tool-branch-b', 'end-node'),
      ],
    });

    const options = buildWorkflowVariableSuggestions(draft, draft.nodes[2]);

    expect(options.map((option) => option.token)).toContain('${outputs.tool-branch-a}');
    expect(options.map((option) => option.token)).not.toContain('${outputs.tool-branch-b}');
  });

  it('never exposes start end if or loop nodes as upstream node variables', () => {
    const draft = createDraft({
      nodes: [
        createNode('start-node', 'start', {
          start: { inputs: [{ name: 'loop_input', type: 'number' }] },
        }),
        createNode('if-node', 'if'),
        createNode('loop-start', 'loop', {
          loop: { role: 'start', loop_id: 'loop-1', max_iterations: 3 },
        }),
        createNode('tool-node', 'tool'),
        createNode('loop-end', 'loop', {
          loop: { role: 'end', loop_id: 'loop-1' },
        }),
        createNode('agent-node', 'agent'),
        createNode('end-node', 'end'),
      ],
      edges: [
        createEdge('start-node', 'if-node'),
        createEdge('if-node', 'loop-start'),
        createEdge('loop-start', 'tool-node'),
        createEdge('tool-node', 'loop-end'),
        createEdge('loop-end', 'agent-node'),
        createEdge('agent-node', 'end-node'),
      ],
    });

    const options = buildWorkflowVariableSuggestions(draft, draft.nodes[5]);
    const tokens = options.map((option) => option.token);

    expect(tokens).toContain('${outputs.tool-node}');
    expect(tokens).not.toContain('${outputs.if-node}');
    expect(tokens).not.toContain('${outputs.loop-start}');
    expect(tokens).not.toContain('${outputs.loop-end}');
    expect(tokens).not.toContain('${outputs.start-node}');
    expect(tokens).not.toContain('${outputs.end-node}');
  });
});

function createDraft(input: {
  nodes: WorkflowCanvasNodeDraft[];
  edges: Array<{ from_node_id: string; to_node_id: string }>;
}): WorkflowCanvasDraft {
  return {
    mode: 'edit',
    schedule: {
      mode: 'interval',
      intervalSeconds: '60',
      cronExpr: '',
    },
    nodes: input.nodes,
    edges: input.edges.map((edge, index) => ({
      id: `edge-${index}`,
      ...edge,
    })),
  };
}

function createEdge(from_node_id: string, to_node_id: string) {
  return { from_node_id, to_node_id };
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
