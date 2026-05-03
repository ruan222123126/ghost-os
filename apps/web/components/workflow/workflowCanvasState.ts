import type {
  ScreenControlComposerStep,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor';
import { isProtectedBoundaryNodeType } from '@/lib/workflow-editor/boundaryNodes';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
} from '@/lib/workflow-editor/constants';
import { duplicateWorkflowNode } from '@/components/workflow/workflowCanvasStateDuplicate';

const NEW_NODE_X_GAP = 260;
const NEW_NODE_Y = 250;
const EDGE_ID_PREFIX = 'edge';
const LOOP_NODE_ID_PREFIX = 'loop';
const LOOP_START_NODE_SUFFIX = 'start';
const LOOP_END_NODE_SUFFIX = 'end';

interface AddNodeOptions {
  position?: WorkflowCanvasPosition;
}

interface ConnectNodesOptions {
  sourceNodeID: string;
  targetNodeID: string;
}

export function addNode(
  draft: WorkflowCanvasDraft,
  type: WorkflowNodeType,
  options?: AddNodeOptions,
): WorkflowCanvasDraft {
  if (isProtectedBoundaryNodeType(type)) {
    return draft;
  }
  if (type === 'loop') {
    return addLoopNodePair(draft, options?.position);
  }

  const index = draft.nodes.length;
  const node = createDraftNode(buildNodeID(type, index), type, index, {
    position: options?.position,
  });

  return {
    ...draft,
    nodes: [...draft.nodes, node],
    selectedNodeId: node.id,
  };
}

export function duplicateNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  const source = draft.nodes.find((node) => node.id === nodeID);
  if (!source || isProtectedBoundaryNodeType(source.type)) {
    return draft;
  }
  return duplicateWorkflowNode(draft, nodeID);
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

export function createDraftNode(
  id: string,
  type: WorkflowNodeType,
  index: number,
  source?: Partial<WorkflowCanvasNodeDraft>,
): WorkflowCanvasNodeDraft {
  return {
    id,
    type,
    position: source?.position ?? { x: index * NEW_NODE_X_GAP, y: NEW_NODE_Y },
    ui: buildNodeUI(source?.ui),
    start: type === 'start' ? source?.start ?? { inputs: [] } : undefined,
    tool: type === 'tool' ? source?.tool ?? { tool_name: '', arguments: {} } : undefined,
    llm: type === 'llm' ? source?.llm ?? { prompt: '', system_prompt: '' } : undefined,
    agent: type === 'agent' ? source?.agent ?? { message: '' } : undefined,
    if: type === 'if'
      ? source?.if ?? buildDefaultIfConfig()
      : undefined,
    loop: type === 'loop'
      ? source?.loop ?? {
        role: LOOP_ROLE_START,
        loop_id: '',
        max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
      }
      : undefined,
  };
}

function buildEdgeID(sourceNodeID: string, targetNodeID: string, index: number): string {
  return `${EDGE_ID_PREFIX}-${index}-${sourceNodeID}-${targetNodeID}`;
}

function buildNodeID(type: WorkflowNodeType, index: number): string {
  return `${type}-${Date.now()}-${index}`;
}

function addLoopNodePair(
  draft: WorkflowCanvasDraft,
  position?: WorkflowCanvasPosition,
): WorkflowCanvasDraft {
  const loopID = `${LOOP_NODE_ID_PREFIX}-${Date.now()}-${draft.nodes.length}`;
  const startNodeID = `${loopID}-${LOOP_START_NODE_SUFFIX}`;
  const endNodeID = `${loopID}-${LOOP_END_NODE_SUFFIX}`;
  const index = draft.nodes.length;
  const startPosition = position ?? { x: index * NEW_NODE_X_GAP, y: NEW_NODE_Y };
  const endPosition = { x: startPosition.x + NEW_NODE_X_GAP, y: startPosition.y };
  const startNode = createDraftNode(startNodeID, 'loop', index, {
    position: startPosition,
    loop: {
      role: LOOP_ROLE_START,
      loop_id: loopID,
      max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
    },
  });
  const endNode = createDraftNode(endNodeID, 'loop', index + 1, {
    position: endPosition,
    loop: {
      role: LOOP_ROLE_END,
      loop_id: loopID,
    },
  });

  return {
    ...draft,
    nodes: [...draft.nodes, startNode, endNode],
    selectedNodeId: startNode.id,
  };
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

function buildDefaultIfConfig(): NonNullable<WorkflowCanvasNodeDraft['if']> {
  return {
    source_node_id: '',
    operator: 'equals',
    value: '',
    true_node_id: '',
    false_node_id: '',
  };
}

function buildNodeUI(sourceUI?: WorkflowCanvasNodeDraft['ui']): WorkflowCanvasNodeDraft['ui'] {
  return {
    toolArgumentsMode: sourceUI?.toolArgumentsMode ?? 'kv',
    screenControlComposer: cloneScreenControlComposer(sourceUI?.screenControlComposer),
  };
}

function cloneScreenControlComposer(
  composer?: WorkflowCanvasNodeDraft['ui']['screenControlComposer'],
): WorkflowCanvasNodeDraft['ui']['screenControlComposer'] {
  if (!composer || composer.steps.length === 0) {
    return undefined;
  }
  return {
    steps: cloneScreenControlComposerSteps(composer.steps),
  };
}

function cloneScreenControlComposerSteps(
  steps: ScreenControlComposerStep[],
): ScreenControlComposerStep[] {
  return steps.map((step) => ({
    action: normalizeScreenControlComposerAction(step.action),
    params: step.params ? { ...step.params } : undefined,
  }));
}
