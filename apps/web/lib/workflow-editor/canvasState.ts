import { isProtectedBoundaryNodeType } from '@/lib/workflow-editor/boundaryNodes';
import { DEFAULT_LOOP_MAX_ITERATIONS, LOOP_ROLE_END, LOOP_ROLE_START } from '@/lib/workflow-editor/constants';
import {
  createDefaultNodePosition,
  createDraftNode,
  createNextNodePosition,
} from '@/lib/workflow-editor/nodeFactory';
import {
  assertDistinctWorkflowNodeIDs,
  assertWorkflowLoopID,
  assertWorkflowNodeID,
} from '@/lib/workflow-editor/nodeIDs';
import { duplicateWorkflowNode, type DuplicateNodeOptions } from '@/lib/workflow-editor/nodeDuplicate';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor/types';

const EDGE_ID_PREFIX = 'edge';

interface AddRegularNodeOptions {
  type: Exclude<WorkflowNodeType, 'loop'>;
  id: string;
  position?: WorkflowCanvasPosition;
  source?: Partial<WorkflowCanvasNodeDraft>;
}

interface AddLoopNodePairOptions {
  type: 'loop';
  loopId: string;
  startNodeId: string;
  endNodeId: string;
  position?: WorkflowCanvasPosition;
}

interface ConnectNodesOptions {
  sourceNodeID: string;
  targetNodeID: string;
}

interface LoopNodePair {
  endNode: WorkflowCanvasNodeDraft;
  startNode: WorkflowCanvasNodeDraft;
}

interface LoopNodePairPositions {
  endPosition: WorkflowCanvasPosition;
  startPosition: WorkflowCanvasPosition;
}

interface CreateLoopNodeOptions {
  id: string;
  index: number;
  loopId: string;
  position: WorkflowCanvasPosition;
}

export type AddNodeOptions = AddRegularNodeOptions | AddLoopNodePairOptions;

export function addNode(draft: WorkflowCanvasDraft, options: AddNodeOptions): WorkflowCanvasDraft {
  if (isProtectedBoundaryNodeType(options.type)) {
    return draft;
  }
  if (options.type === 'loop') {
    return addLoopNodePair(draft, options);
  }

  assertNodeIDAvailable(draft, options.id, 'workflow node id');
  const index = draft.nodes.length;
  const node = createDraftNode({
    id: options.id,
    type: options.type,
    index,
    source: { position: options.position, ...options.source },
  });

  return {
    ...draft,
    nodes: [...draft.nodes, node],
    selectedNodeId: node.id,
  };
}

export function duplicateNode(draft: WorkflowCanvasDraft, options: DuplicateNodeOptions): WorkflowCanvasDraft {
  const source = draft.nodes.find((node) => node.id === options.nodeID);
  if (!source || isProtectedBoundaryNodeType(source.type)) {
    return draft;
  }
  return duplicateWorkflowNode(draft, options);
}

export function removeNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  const target = draft.nodes.find((node) => node.id === nodeID);
  if (!target || isProtectedBoundaryNodeType(target.type)) {
    return draft;
  }
  const removableNodeIDs = findRemovableNodeIDs(draft.nodes, nodeID);
  return {
    ...draft,
    nodes: draft.nodes.filter((node) => !removableNodeIDs.has(node.id)),
    edges: draft.edges.filter(
      (edge) => !removableNodeIDs.has(edge.from_node_id) && !removableNodeIDs.has(edge.to_node_id),
    ),
    selectedNodeId: draft.selectedNodeId && removableNodeIDs.has(draft.selectedNodeId)
      ? undefined
      : draft.selectedNodeId,
  };
}

export function updateNode(draft: WorkflowCanvasDraft, node: WorkflowCanvasNodeDraft): WorkflowCanvasDraft {
  const currentID = draft.selectedNodeId ?? node.id;
  const currentNode = draft.nodes.find((item) => item.id === currentID);
  if (!currentNode) {
    return draft;
  }

  const nextNode = sanitizeNodeUpdate(currentNode, node);
  const nodes = draft.nodes.map((item) => (item.id === currentID ? nextNode : item));
  if (currentID === nextNode.id) {
    return { ...draft, nodes };
  }

  return applyNodeIDChange({ draft, nodes, currentID, nextID: nextNode.id });
}

export function moveNode(
  draft: WorkflowCanvasDraft,
  nodeID: string,
  position: WorkflowCanvasPosition,
): WorkflowCanvasDraft {
  return {
    ...draft,
    nodes: draft.nodes.map((node) => (node.id === nodeID ? { ...node, position } : node)),
  };
}

export function connectNodesByID(
  draft: WorkflowCanvasDraft,
  options: ConnectNodesOptions,
): WorkflowCanvasDraft {
  const { sourceNodeID, targetNodeID } = options;
  if (sourceNodeID === targetNodeID) {
    return draft;
  }
  const duplicate = draft.edges.some(
    (edge) => edge.from_node_id === sourceNodeID && edge.to_node_id === targetNodeID,
  );
  if (duplicate) {
    return draft;
  }

  return {
    ...draft,
    edges: [
      ...draft.edges,
      {
        id: buildEdgeID(sourceNodeID, targetNodeID, draft.edges.length + 1),
        from_node_id: sourceNodeID,
        to_node_id: targetNodeID,
      },
    ],
  };
}

export function removeEdge(draft: WorkflowCanvasDraft, edgeID: string): WorkflowCanvasDraft {
  return {
    ...draft,
    edges: draft.edges.filter((edge) => edge.id !== edgeID),
  };
}

function buildEdgeID(sourceNodeID: string, targetNodeID: string, index: number): string {
  return `${EDGE_ID_PREFIX}-${index}-${sourceNodeID}-${targetNodeID}`;
}

function applyNodeIDChange(input: {
  draft: WorkflowCanvasDraft;
  nodes: WorkflowCanvasNodeDraft[];
  currentID: string;
  nextID: string;
}): WorkflowCanvasDraft {
  const { currentID, draft, nextID, nodes } = input;
  return {
    ...draft,
    nodes,
    edges: draft.edges.map((edge) => ({
      ...edge,
      from_node_id: edge.from_node_id === currentID ? nextID : edge.from_node_id,
      to_node_id: edge.to_node_id === currentID ? nextID : edge.to_node_id,
    })),
    selectedNodeId: nextID,
  };
}

function addLoopNodePair(
  draft: WorkflowCanvasDraft,
  options: AddLoopNodePairOptions,
): WorkflowCanvasDraft {
  assertLoopNodeIDsAvailable(draft, options);
  const pair = buildLoopNodePair(draft.nodes.length, options);

  return {
    ...draft,
    nodes: [...draft.nodes, pair.startNode, pair.endNode],
    selectedNodeId: pair.startNode.id,
  };
}

function buildLoopNodePair(index: number, options: AddLoopNodePairOptions): LoopNodePair {
  const { startPosition, endPosition } = buildLoopNodePairPositions(index, options.position);

  return {
    startNode: createLoopStartNode({
      id: options.startNodeId,
      index,
      loopId: options.loopId,
      position: startPosition,
    }),
    endNode: createLoopEndNode({
      id: options.endNodeId,
      index: index + 1,
      loopId: options.loopId,
      position: endPosition,
    }),
  };
}

function buildLoopNodePairPositions(
  index: number,
  position: WorkflowCanvasPosition | undefined,
): LoopNodePairPositions {
  const startPosition = position ?? createDefaultNodePosition(index);
  return {
    startPosition,
    endPosition: createNextNodePosition(startPosition),
  };
}

function createLoopStartNode(options: CreateLoopNodeOptions): WorkflowCanvasNodeDraft {
  return createDraftNode({
    id: options.id,
    type: 'loop',
    index: options.index,
    source: {
      position: options.position,
      loop: {
        role: LOOP_ROLE_START,
        loop_id: options.loopId,
        max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
      },
    },
  });
}

function createLoopEndNode(options: CreateLoopNodeOptions): WorkflowCanvasNodeDraft {
  return createDraftNode({
    id: options.id,
    type: 'loop',
    index: options.index,
    source: {
      position: options.position,
      loop: {
        role: LOOP_ROLE_END,
        loop_id: options.loopId,
      },
    },
  });
}

function assertNodeIDAvailable(draft: WorkflowCanvasDraft, nodeID: string, label: string): void {
  assertWorkflowNodeID(nodeID, label);
  if (draft.nodes.some((node) => node.id === nodeID)) {
    throw new Error(`workflow node id already exists: ${nodeID}`);
  }
}

function assertLoopNodeIDsAvailable(draft: WorkflowCanvasDraft, options: AddLoopNodePairOptions): void {
  assertWorkflowLoopID(options.loopId);
  assertNodeIDAvailable(draft, options.startNodeId, 'workflow loop start node id');
  assertNodeIDAvailable(draft, options.endNodeId, 'workflow loop end node id');
  assertDistinctWorkflowNodeIDs([options.startNodeId, options.endNodeId]);
  if (draft.nodes.some((node) => node.type === 'loop' && node.loop?.loop_id === options.loopId)) {
    throw new Error(`workflow loop id already exists: ${options.loopId}`);
  }
}

function findRemovableNodeIDs(nodes: WorkflowCanvasNodeDraft[], nodeID: string): Set<string> {
  const removable = new Set<string>([nodeID]);
  const loopID = removableLoopID(nodes, nodeID);
  if (!loopID) {
    return removable;
  }

  for (const node of nodes) {
    if (isLoopNodeInLoop(node, loopID)) {
      removable.add(node.id);
    }
  }
  return removable;
}

function sanitizeNodeUpdate(currentNode: WorkflowCanvasNodeDraft, nextNode: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  if (!shouldLockLoopIdentity(currentNode)) {
    return nextNode;
  }
  const nextLoop: Partial<NonNullable<WorkflowCanvasNodeDraft['loop']>> = nextNode.loop ?? {};
  return {
    ...nextNode,
    id: currentNode.id,
    type: 'loop',
    loop: {
      ...nextLoop,
      role: currentNode.loop.role,
      loop_id: currentNode.loop.loop_id,
      max_iterations: loopMaxIterationsForUpdate(currentNode, nextLoop),
    },
  };
}

function removableLoopID(nodes: WorkflowCanvasNodeDraft[], nodeID: string): string {
  const target = nodes.find((node) => node.id === nodeID);
  if (target?.type !== 'loop') {
    return '';
  }
  return target.loop?.loop_id?.trim() ?? '';
}

function isLoopNodeInLoop(node: WorkflowCanvasNodeDraft, loopID: string): boolean {
  return node.type === 'loop' && node.loop?.loop_id === loopID;
}

function shouldLockLoopIdentity(node: WorkflowCanvasNodeDraft): node is WorkflowCanvasNodeDraft & {
  loop: NonNullable<WorkflowCanvasNodeDraft['loop']>;
} {
  return node.type === 'loop' && Boolean(node.loop?.loop_id?.trim());
}

function loopMaxIterationsForUpdate(
  currentNode: WorkflowCanvasNodeDraft & { loop: NonNullable<WorkflowCanvasNodeDraft['loop']> },
  nextLoop: Partial<NonNullable<WorkflowCanvasNodeDraft['loop']>>,
): number | undefined {
  if (currentNode.loop.role !== LOOP_ROLE_START) {
    return undefined;
  }
  return nextLoop.max_iterations ?? currentNode.loop.max_iterations ?? DEFAULT_LOOP_MAX_ITERATIONS;
}
