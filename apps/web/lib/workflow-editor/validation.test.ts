import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';
import { validateWorkflowDraft } from '@/lib/workflow-editor/validation';

describe('lib/workflow-editor/validation', () => {
  it('passes validation for linear workflow chain', () => {
    const draft = createEmptyWorkflowDraft();
    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('passes validation when start branches to two nodes', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'tool-a',
        type: 'tool',
        position: { x: 200, y: 20 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: 'script_exec' },
      },
      {
        id: 'tool-b',
        type: 'tool',
        position: { x: 200, y: 140 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: 'script_exec' },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'edge-a', from_node_id: 'start-node', to_node_id: 'tool-a' },
      { id: 'edge-b', from_node_id: 'start-node', to_node_id: 'tool-b' },
      { id: 'edge-c', from_node_id: 'tool-a', to_node_id: 'end-node' },
      { id: 'edge-d', from_node_id: 'tool-b', to_node_id: 'end-node' },
    ];

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('passes validation for regular if + loop start/end pair', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'if-1',
        type: 'if',
        position: { x: 200, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        if: {
          operator: 'is_empty',
          true_node_id: 'sync-loop-start',
          false_node_id: 'end-node',
        },
      },
      {
        id: 'sync-loop-start',
        type: 'loop',
        position: { x: 420, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        loop: {
          role: 'start',
          loop_id: 'sync-loop',
          max_iterations: 2,
        },
      },
      {
        id: 'tool-1',
        type: 'tool',
        position: { x: 640, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: 'script_exec' },
      },
      {
        id: 'sync-loop-end',
        type: 'loop',
        position: { x: 860, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        loop: {
          role: 'end',
          loop_id: 'sync-loop',
        },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'edge-a', from_node_id: 'start-node', to_node_id: 'if-1' },
      { id: 'edge-b', from_node_id: 'if-1', to_node_id: 'sync-loop-start' },
      { id: 'edge-c', from_node_id: 'if-1', to_node_id: 'end-node' },
      { id: 'edge-d', from_node_id: 'sync-loop-start', to_node_id: 'tool-1' },
      { id: 'edge-e', from_node_id: 'tool-1', to_node_id: 'sync-loop-end' },
      { id: 'edge-g', from_node_id: 'sync-loop-end', to_node_id: 'end-node' },
    ];

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('rejects loop end node with non-exit back edge', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'sync-loop-start',
        type: 'loop',
        position: { x: 260, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        loop: {
          role: 'start',
          loop_id: 'sync-loop',
          max_iterations: 2,
        },
      },
      {
        id: 'tool-1',
        type: 'tool',
        position: { x: 480, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: 'script_exec' },
      },
      {
        id: 'sync-loop-end',
        type: 'loop',
        position: { x: 700, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        loop: {
          role: 'end',
          loop_id: 'sync-loop',
        },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'edge-a', from_node_id: 'start-node', to_node_id: 'sync-loop-start' },
      { id: 'edge-b', from_node_id: 'sync-loop-start', to_node_id: 'tool-1' },
      { id: 'edge-c', from_node_id: 'tool-1', to_node_id: 'sync-loop-end' },
      { id: 'edge-d', from_node_id: 'sync-loop-end', to_node_id: 'sync-loop-start' },
      { id: 'edge-e', from_node_id: 'sync-loop-end', to_node_id: 'end-node' },
    ];

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        'workflow node "sync-loop-end" must have in>=1 and out=1',
      ]),
    );
  });

  it('rejects invalid if/loop setup and payload mismatch', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'if-1',
        type: 'if',
        position: { x: 200, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        if: {
          operator: 'contains',
          true_node_id: 'tool-1',
          false_node_id: 'end-node',
        },
      },
      {
        id: 'sync-loop-start',
        type: 'loop',
        position: { x: 420, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        loop: {
          role: 'start',
          loop_id: 'sync-loop',
          max_iterations: 0,
        },
      },
      {
        id: 'tool-1',
        type: 'tool',
        position: { x: 640, y: 80 },
        ui: { toolArgumentsMode: 'kv' },
        tool: { tool_name: '' },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'edge-a', from_node_id: 'start-node', to_node_id: 'if-1' },
      { id: 'edge-b', from_node_id: 'if-1', to_node_id: 'sync-loop-start' },
      { id: 'edge-c', from_node_id: 'if-1', to_node_id: 'end-node' },
      { id: 'edge-d', from_node_id: 'sync-loop-start', to_node_id: 'tool-1' },
      { id: 'edge-e', from_node_id: 'tool-1', to_node_id: 'sync-loop-start' },
    ];

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        'workflow tool node "tool-1" requires tool_name',
        'workflow loop node "sync-loop-start" start role requires max_iterations > 0',
        'workflow if node "if-1" outgoing edges must match true_node_id/false_node_id',
        'loop "sync-loop" must contain one start node and one end node',
      ]),
    );
  });

  it('validates start.inputs name constraints and value type', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes[0] = {
      ...draft.nodes[0],
      start: {
        inputs: [
          { name: '', type: 'string', default: 'x' },
          { name: '1invalid', type: 'string', default: 'x' },
          { name: 'x'.repeat(101), type: 'number', default: '3' as unknown as number },
          { name: 'dup_name', type: 'boolean', default: true },
          { name: 'dup_name', type: 'string', default: 'v' },
        ],
      },
    };

    const result = validateWorkflowDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        'workflow start node "start-node" input[0] name must be a non-empty string',
        'workflow start node "start-node" input[1] name must match ^[A-Za-z_][A-Za-z0-9_]*$',
        'workflow start node "start-node" input[2] name length must be <= 100',
        'workflow start node "start-node" input[2] value must match declared type "number"',
        'workflow start node "start-node" input[4] duplicate name "dup_name"',
      ]),
    );
  });
});
