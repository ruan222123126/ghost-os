import {
  addNode,
  connectNodesByID,
  duplicateNode,
  removeNode,
  updateNode,
} from '@/lib/workflow-editor/canvasState';
import { LOOP_ROLE_END, LOOP_ROLE_START } from '@/lib/workflow-editor/constants';
import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor/types';

const DUPLICATE_OFFSET = 36;
const TEST_POSITION = { x: 120, y: 90 };

describe('lib/workflow-editor/canvasState', () => {
  it('does not add extra start or end nodes', () => {
    const draft = createEmptyWorkflowDraft();

    expect(addTestNode({ draft, type: 'start', id: 'extra-start' })).toBe(draft);
    expect(addTestNode({ draft, type: 'end', id: 'extra-end' })).toBe(draft);
  });

  it('throws explicit errors for invalid generated node ids', () => {
    const draft = createEmptyWorkflowDraft();

    expect(() => addTestNode({ draft, type: 'tool', id: '' })).toThrow('workflow node id is required');
    expect(() => addTestLoopNodePair({
      draft,
      loopId: '',
      startNodeId: 'loop-empty-start',
      endNodeId: 'loop-empty-end',
    })).toThrow('workflow loop id is required');
  });

  it('does not remove or duplicate protected start/end nodes', () => {
    const draft = createEmptyWorkflowDraft();
    const startNode = draft.nodes.find((node) => node.type === 'start');
    const endNode = draft.nodes.find((node) => node.type === 'end');
    if (!startNode || !endNode) {
      throw new Error('protected boundary nodes missing');
    }

    expect(removeNode(draft, startNode.id)).toBe(draft);
    expect(removeNode(draft, endNode.id)).toBe(draft);
    expect(duplicateNode(draft, { nodeID: startNode.id, newNodeId: 'copy-start' })).toBe(draft);
    expect(duplicateNode(draft, { nodeID: endNode.id, newNodeId: 'copy-end' })).toBe(draft);
  });

  it('adds loop start/end node pair with explicit ids', () => {
    const draft = createEmptyWorkflowDraft();
    const next = addTestLoopNodePair({
      draft,
      loopId: 'loop-a',
      startNodeId: 'loop-a-start',
      endNodeId: 'loop-a-end',
      position: { x: 320, y: 180 },
    });
    const loopNodes = next.nodes.filter((node) => node.type === 'loop');
    const startNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_START);
    const endNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_END);

    expect(loopNodes).toHaveLength(2);
    expect(startNode?.id).toBe('loop-a-start');
    expect(endNode?.id).toBe('loop-a-end');
    expect(startNode?.loop?.loop_id).toBe('loop-a');
    expect(endNode?.loop?.loop_id).toBe('loop-a');
    expect(next.selectedNodeId).toBe('loop-a-start');
  });

  it('removes paired loop start/end together when deleting either side', () => {
    const draft = createEmptyWorkflowDraft();
    const withPair = addTestLoopNodePair({
      draft,
      loopId: 'loop-remove',
      startNodeId: 'loop-remove-start',
      endNodeId: 'loop-remove-end',
    });
    const startNode = withPair.nodes.find((node) => node.id === 'loop-remove-start');
    const endNode = withPair.nodes.find((node) => node.id === 'loop-remove-end');
    if (!startNode || !endNode) {
      throw new Error('loop node pair missing');
    }

    const connected = connectNodesByID(withPair, {
      sourceNodeID: startNode.id,
      targetNodeID: endNode.id,
    });
    const removedFromStart = removeNode(connected, startNode.id);
    const removedFromEnd = removeNode(connected, endNode.id);

    expect(removedFromStart.nodes.some((node) => node.id === startNode.id)).toBe(false);
    expect(removedFromStart.nodes.some((node) => node.id === endNode.id)).toBe(false);
    expect(removedFromEnd.nodes.some((node) => node.id === startNode.id)).toBe(false);
    expect(removedFromEnd.nodes.some((node) => node.id === endNode.id)).toBe(false);
  });

  it('locks paired loop node id/role/loop_id from manual updates', () => {
    const draft = createEmptyWorkflowDraft();
    const withPair = addTestLoopNodePair({
      draft,
      loopId: 'loop-locked',
      startNodeId: 'loop-locked-start',
      endNodeId: 'loop-locked-end',
    });
    const startNode = withPair.nodes.find((node) => node.id === 'loop-locked-start');
    if (!startNode || !startNode.loop?.loop_id) {
      throw new Error('loop start node missing');
    }

    const updated = updateNode(withPair, {
      ...startNode,
      id: 'manual-override',
      loop: {
        ...startNode.loop,
        role: LOOP_ROLE_END,
        loop_id: 'manual-loop-id',
        max_iterations: 9,
      },
    });
    const persisted = updated.nodes.find((node) => node.id === startNode.id);

    expect(updated.nodes.some((node) => node.id === 'manual-override')).toBe(false);
    expect(persisted?.loop?.role).toBe(LOOP_ROLE_START);
    expect(persisted?.loop?.loop_id).toBe(startNode.loop.loop_id);
    expect(persisted?.loop?.max_iterations).toBe(9);
  });

  it('adds if node as a regular single node with explicit id', () => {
    const draft = createEmptyWorkflowDraft();
    const withIf = addTestNode({ draft, type: 'if', id: 'if-explicit' });
    const ifNodes = withIf.nodes.filter((node) => node.type === 'if');

    expect(ifNodes).toHaveLength(1);
    expect(ifNodes[0]?.id).toBe('if-explicit');
    expect(ifNodes[0]?.if?.operator).toBe('equals');
  });

  it('duplicates a regular node with copied payload and explicit id', () => {
    const draft = createEmptyWorkflowDraft();
    const withNode = addTestNode({
      draft,
      type: 'agent',
      id: 'agent-source',
      position: TEST_POSITION,
    });
    const sourceNode = withNode.nodes.find((node) => node.id === 'agent-source');
    if (!sourceNode) {
      throw new Error('agent node missing');
    }
    const configured = updateNode(withNode, {
      ...sourceNode,
      agent: { message: 'planner' },
    });

    const duplicated = duplicateNode(configured, {
      nodeID: sourceNode.id,
      newNodeId: 'agent-copy',
    });
    const copiedNode = duplicated.nodes.find((node) => node.id === 'agent-copy');
    if (!copiedNode) {
      throw new Error('copied node missing');
    }

    expect(copiedNode.agent?.message).toBe('planner');
    expect(copiedNode.position.x).toBe(sourceNode.position.x + DUPLICATE_OFFSET);
    expect(copiedNode.position.y).toBe(sourceNode.position.y + DUPLICATE_OFFSET);
    expect(duplicated.selectedNodeId).toBe('agent-copy');
  });

  it('duplicates loop node as paired start/end with explicit ids', () => {
    const draft = createEmptyWorkflowDraft();
    const withLoopPair = addTestLoopNodePair({
      draft,
      loopId: 'loop-source',
      startNodeId: 'loop-source-start',
      endNodeId: 'loop-source-end',
      position: { x: 200, y: 140 },
    });
    const sourceStart = withLoopPair.nodes.find((node) => node.id === 'loop-source-start');
    const sourceEnd = withLoopPair.nodes.find((node) => node.id === 'loop-source-end');
    if (!sourceStart || !sourceEnd) {
      throw new Error('loop node pair missing');
    }

    const duplicated = duplicateNode(withLoopPair, {
      nodeID: sourceEnd.id,
      loopId: 'loop-copy',
      startNodeId: 'loop-copy-start',
      endNodeId: 'loop-copy-end',
    });
    const copiedStart = duplicated.nodes.find((node) => node.id === 'loop-copy-start');
    const copiedEnd = duplicated.nodes.find((node) => node.id === 'loop-copy-end');
    if (!copiedStart || !copiedEnd) {
      throw new Error('copied loop node pair missing');
    }

    expect(copiedStart.loop?.loop_id).toBe('loop-copy');
    expect(copiedEnd.loop?.loop_id).toBe('loop-copy');
    expect(copiedStart.position.x).toBe(sourceStart.position.x + DUPLICATE_OFFSET);
    expect(copiedStart.position.y).toBe(sourceStart.position.y + DUPLICATE_OFFSET);
    expect(copiedEnd.position.x).toBe(sourceEnd.position.x + DUPLICATE_OFFSET);
    expect(copiedEnd.position.y).toBe(sourceEnd.position.y + DUPLICATE_OFFSET);
    expect(duplicated.selectedNodeId).toBe('loop-copy-end');
  });

  it('throws explicit errors when duplicate ids are missing', () => {
    const draft = createEmptyWorkflowDraft();
    const withToolNode = addTestNode({ draft, type: 'tool', id: 'tool-source' });

    expect(() => duplicateNode(withToolNode, { nodeID: 'tool-source' })).toThrow(
      'duplicate workflow node id is required',
    );
  });

  it('keeps screen control composer steps when duplicating a tool node', () => {
    const draft = createEmptyWorkflowDraft();
    const withToolNode = addTestNode({
      draft,
      type: 'tool',
      id: 'tool-source',
      position: { x: 180, y: 120 },
    });
    const toolNode = withToolNode.nodes.find((node) => node.id === 'tool-source');
    if (!toolNode) {
      throw new Error('tool node missing');
    }

    const configured = updateNode(withToolNode, {
      ...toolNode,
      tool: { tool_name: 'screen_control', arguments: {} },
      ui: {
        ...toolNode.ui,
        screenControlComposer: {
          steps: [{ action: 'screenshot' }, { action: 'find_text' }],
        },
      },
    });
    const duplicated = duplicateNode(configured, {
      nodeID: toolNode.id,
      newNodeId: 'tool-copy',
    });
    const copiedNode = duplicated.nodes.find((node) => node.id === 'tool-copy');
    if (!copiedNode) {
      throw new Error('copied tool node missing');
    }

    expect(copiedNode.ui.screenControlComposer?.steps).toEqual([
      { action: 'screenshot' },
      { action: 'find_text' },
    ]);
    expect(copiedNode.ui.screenControlComposer?.steps).not.toBe(
      configured.nodes.find((node) => node.id === toolNode.id)?.ui.screenControlComposer?.steps,
    );
  });
});

interface AddTestNodeOptions {
  draft: WorkflowCanvasDraft;
  type: Exclude<WorkflowNodeType, 'loop'>;
  id: string;
  position?: WorkflowCanvasPosition;
}

function addTestNode(options: AddTestNodeOptions): WorkflowCanvasDraft {
  const { draft, type, id, position } = options;
  return addNode(draft, { type, id, position });
}

interface AddTestLoopOptions {
  draft: WorkflowCanvasDraft;
  loopId: string;
  startNodeId: string;
  endNodeId: string;
  position?: WorkflowCanvasPosition;
}

function addTestLoopNodePair(options: AddTestLoopOptions): WorkflowCanvasDraft {
  return addNode(options.draft, {
    type: 'loop',
    loopId: options.loopId,
    startNodeId: options.startNodeId,
    endNodeId: options.endNodeId,
    position: options.position,
  });
}
