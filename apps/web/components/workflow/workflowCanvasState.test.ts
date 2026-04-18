import {
  addNode,
  connectNodesByID,
  duplicateNode,
  removeNode,
  updateNode,
} from '@/components/workflow/workflowCanvasState';
import { LOOP_ROLE_END, LOOP_ROLE_START } from '@/lib/workflow-editor/constants';
import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';

const DUPLICATE_OFFSET = 36;

describe('components/workflow/workflowCanvasState', () => {
  it('adds loop start/end node pair with shared loop_id in one click', () => {
    const draft = createEmptyWorkflowDraft();
    const next = addNode(draft, 'loop', { position: { x: 320, y: 180 } });
    const loopNodes = next.nodes.filter((node) => node.type === 'loop');
    const startNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_START);
    const endNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_END);

    expect(loopNodes).toHaveLength(2);
    expect(startNode).toBeDefined();
    expect(endNode).toBeDefined();
    expect(startNode?.loop?.loop_id).toBeTruthy();
    expect(startNode?.loop?.loop_id).toBe(endNode?.loop?.loop_id);
    expect(next.selectedNodeId).toBe(startNode?.id);
  });

  it('removes paired loop start/end together when deleting either side', () => {
    const draft = createEmptyWorkflowDraft();
    const withPair = addNode(draft, 'loop');
    const loopNodes = withPair.nodes.filter((node) => node.type === 'loop');
    const startNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_START);
    const endNode = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_END);
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
    const withPair = addNode(draft, 'loop');
    const startNode = withPair.nodes.find((node) => node.type === 'loop' && node.loop?.role === LOOP_ROLE_START);
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

  it('keeps if node as regular single node', () => {
    const draft = createEmptyWorkflowDraft();
    const withIf = addNode(draft, 'if');
    const ifNodes = withIf.nodes.filter((node) => node.type === 'if');

    expect(ifNodes).toHaveLength(1);
    expect(ifNodes[0]?.if?.operator).toBe('equals');
  });

  it('duplicates a regular node with copied payload and shifted position', () => {
    const draft = createEmptyWorkflowDraft();
    const withNode = addNode(draft, 'agent', { position: { x: 120, y: 90 } });
    const sourceNode = withNode.nodes.find((node) => node.type === 'agent');
    if (!sourceNode) {
      throw new Error('agent node missing');
    }
    const configured = updateNode(withNode, {
      ...sourceNode,
      agent: { message: 'planner' },
    });

    const duplicated = duplicateNode(configured, sourceNode.id);
    const copiedNode = duplicated.nodes.find((node) => node.id !== sourceNode.id && node.type === 'agent');
    if (!copiedNode) {
      throw new Error('copied node missing');
    }

    expect(copiedNode.agent?.message).toBe('planner');
    expect(copiedNode.position.x).toBe(sourceNode.position.x + DUPLICATE_OFFSET);
    expect(copiedNode.position.y).toBe(sourceNode.position.y + DUPLICATE_OFFSET);
    expect(duplicated.selectedNodeId).toBe(copiedNode.id);
  });

  it('duplicates loop node as paired start/end with a new loop_id', () => {
    const draft = createEmptyWorkflowDraft();
    const withLoopPair = addNode(draft, 'loop', { position: { x: 200, y: 140 } });
    const loopNodes = withLoopPair.nodes.filter((node) => node.type === 'loop');
    const sourceStart = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_START);
    const sourceEnd = loopNodes.find((node) => node.loop?.role === LOOP_ROLE_END);
    if (!sourceStart || !sourceEnd || !sourceStart.loop?.loop_id) {
      throw new Error('loop node pair missing');
    }

    const duplicated = duplicateNode(withLoopPair, sourceEnd.id);
    const duplicatedLoopNodes = duplicated.nodes.filter((node) =>
      node.type === 'loop' && node.loop?.loop_id !== sourceStart.loop?.loop_id);
    const copiedStart = duplicatedLoopNodes.find((node) => node.loop?.role === LOOP_ROLE_START);
    const copiedEnd = duplicatedLoopNodes.find((node) => node.loop?.role === LOOP_ROLE_END);
    if (!copiedStart || !copiedEnd || !copiedStart.loop?.loop_id) {
      throw new Error('copied loop node pair missing');
    }

    expect(duplicatedLoopNodes).toHaveLength(2);
    expect(copiedStart.loop?.loop_id).toBe(copiedEnd.loop?.loop_id);
    expect(copiedStart.loop?.loop_id).not.toBe(sourceStart.loop.loop_id);
    expect(copiedStart.position.x).toBe(sourceStart.position.x + DUPLICATE_OFFSET);
    expect(copiedStart.position.y).toBe(sourceStart.position.y + DUPLICATE_OFFSET);
    expect(copiedEnd.position.x).toBe(sourceEnd.position.x + DUPLICATE_OFFSET);
    expect(copiedEnd.position.y).toBe(sourceEnd.position.y + DUPLICATE_OFFSET);
    expect(duplicated.selectedNodeId).toBe(copiedEnd.id);
  });

  it('keeps screen control composer steps when duplicating a tool node', () => {
    const draft = createEmptyWorkflowDraft();
    const withToolNode = addNode(draft, 'tool', { position: { x: 180, y: 120 } });
    const toolNode = withToolNode.nodes.find((node) => node.type === 'tool');
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
    const duplicated = duplicateNode(configured, toolNode.id);
    const copiedNode = duplicated.nodes.find((node) => node.id !== toolNode.id && node.type === 'tool');
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
