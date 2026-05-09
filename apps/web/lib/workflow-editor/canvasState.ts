import { isProtectedBoundaryNodeType } from '@/lib/workflow-editor/boundaryNodes';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
} from '@/lib/workflow-editor/constants';
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

  return {
    ...draft,
    nodes,
    edges: draft.edges.map((edge) => ({
      ...edge,
      from_node_id: edge.from_node_id === currentID ? nextNode.id : edge.from_node_id,
      to_node_id: edge.to_node_id === currentID ? nextNode.id : edge.to_node_id,
    })),
    selectedNodeId: nextNode.id,
  };
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

function addLoopNodePair(
  draft: WorkflowCanvasDraft,
  options: AddLoopNodePairOptions,
): WorkflowCanvasDraft {
  assertLoopNodeIDsAvailable(draft, options);
  const index = draft.nodes.length;
  const startPosition = options.position ?? createDefaultNodePosition(index);
  const endPosition = createNextNodePosition(startPosition);
  const startNode = createDraftNode({
    id: options.startNodeId,
    type: 'loop',
    index,
    source: {
      position: startPosition,
      loop: {
        role: LOOP_ROLE_START,
        loop_id: options.loopId,
        max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
      },
    },
  });
  const endNode = createDraftNode({
    id: options.endNodeId,
    type: 'loop',
    index: index + 1,
    source: {
      position: endPosition,
      loop: {
        role: LOOP_ROLE_END,
        loop_id: options.loopId,
      },
    },
  });

  return {
    ...draft,
    nodes: [...draft.nodes, startNode, endNode],
    selectedNodeId: startNode.id,
  };
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
  const target = nodes.find((node) => node.id === nodeID);
  const loopID = target?.type === 'loop' ? target.loop?.loop_id?.trim() : '';
  if (!loopID) {
    return removable;
  }

  for (const node of nodes) {
    if (node.type !== 'loop' || node.loop?.loop_id !== loopID) {
      continue;
    }
    removable.add(node.id);
  }
  return removable;
}

function sanitizeNodeUpdate(currentNode: WorkflowCanvasNodeDraft, nextNode: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  if (currentNode.type !== 'loop' || !currentNode.loop?.loop_id?.trim()) {
    return nextNode;
  }
  const nextLoop: Partial<NonNullable<WorkflowCanvasNodeDraft['loop']>> = nextNode.loop ?? {};
  const role = currentNode.loop.role;
  const maxIterations = role === LOOP_ROLE_START
    ? nextLoop.max_iterations ?? currentNode.loop.max_iterations ?? DEFAULT_LOOP_MAX_ITERATIONS
    : undefined;
  return {
    ...nextNode,
    id: currentNode.id,
    type: 'loop',
    loop: {
      ...nextLoop,
      role,
      loop_id: currentNode.loop.loop_id,
      max_iterations: maxIterations,
    },
  };
}
