import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';
import { validateWorkflowDraft } from '@/lib/workflow-editor/validation';

describe('lib/workflow-editor/validation', () => {
  it('passes validation for linear workflow chain', () => {
    const draft = createEmptyWorkflowDraft();
    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('rejects branching, cycle, isolated node, and payload mismatch', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'tool-1',
        type: 'tool',
        position: { x: 200, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: '' },
      },
      {
        id: 'agent-1',
        type: 'agent',
        position: { x: 420, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
      },
      {
        id: 'isolated',
        type: 'llm',
        position: { x: 640, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        llm: { prompt: 'x' },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'edge-a', from_node_id: 'start-node', to_node_id: 'tool-1' },
      { id: 'edge-b', from_node_id: 'start-node', to_node_id: 'agent-1' },
      { id: 'edge-c', from_node_id: 'agent-1', to_node_id: 'tool-1' },
      { id: 'edge-d', from_node_id: 'tool-1', to_node_id: 'agent-1' },
    ];

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        'workflow branching is not supported',
        'workflow must not contain cycles',
        'workflow tool node "tool-1" requires tool_name',
        'workflow node "agent-1" payload does not match type "agent"',
      ]),
    );
  });

  it('validates start.inputs name uniqueness and default type', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes[0] = {
      ...draft.nodes[0],
      start: {
        inputs: [
          { name: '1invalid', type: 'string', default: 'x' },
          { name: 'dup_name', type: 'number', default: '3' as unknown as number },
          { name: 'dup_name', type: 'boolean', default: true },
        ],
      },
    };

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        'workflow start node "start-node" input[0] name "1invalid" must match ^[A-Za-z_][A-Za-z0-9_]*$',
        'workflow start node "start-node" input[1] default must match declared type "number"',
        'workflow start node "start-node" input[2] duplicate name "dup_name"',
      ]),
    );
  });
});
