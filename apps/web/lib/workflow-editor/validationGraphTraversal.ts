import {
  END_NODE_TYPE,
  START_NODE_TYPE,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

interface ConnectivityInput {
  nodes: WorkflowCanvasNodeDraft[];
  outgoing: Map<string, string[]>;
  incoming: Map<string, string[]>;
  errors: string[];
}

interface BoundaryNodes {
  endNode: WorkflowCanvasNodeDraft;
  startNode: WorkflowCanvasNodeDraft;
}

interface CycleWalkContext {
  outgoing: Map<string, string[]>;
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  color: Map<string, number>;
  stack: string[];
  position: Map<string, number>;
  errors: string[];
}

export function validateConnectivity(input: ConnectivityInput): void {
  const boundaryNodes = findBoundaryNodes(input.nodes);
  if (!boundaryNodes) {
    return;
  }

  validateReachableFromStart(input, boundaryNodes.startNode);
  validateCanReachEnd(input, boundaryNodes.endNode);
}

function findBoundaryNodes(nodes: WorkflowCanvasNodeDraft[]): BoundaryNodes | undefined {
  const startNode = nodes.find((node) => node.type === START_NODE_TYPE);
  const endNode = nodes.find((node) => node.type === END_NODE_TYPE);
  if (!startNode || !endNode) {
    return undefined;
  }
  return { endNode, startNode };
}

function validateReachableFromStart(input: ConnectivityInput, startNode: WorkflowCanvasNodeDraft): void {
  const fromStart = walkGraph(startNode.id, input.outgoing);
  if (fromStart.size !== input.nodes.length) {
    input.errors.push('workflow must be fully connected from start node');
  }
}

function validateCanReachEnd(input: ConnectivityInput, endNode: WorkflowCanvasNodeDraft): void {
  const toEnd = walkGraph(endNode.id, input.incoming);
  for (const node of input.nodes) {
    if (!toEnd.has(node.id)) {
      input.errors.push(`workflow node "${node.id}" cannot reach end node`);
    }
  }
}

export function validateCycles(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  const context = createCycleWalkContext(nodes, outgoing, errors);
  for (const node of nodes) {
    if ((context.color.get(node.id) ?? 0) !== 0) {
      continue;
    }
    if (walkCycle(node.id, context)) {
      return;
    }
  }
}

export function pathExists(
  adjacency: Map<string, string[]>,
  startID: string,
  targetID: string,
): boolean {
  return walkGraph(startID, adjacency).has(targetID);
}

function createCycleWalkContext(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): CycleWalkContext {
  return {
    outgoing,
    nodeMap: new Map(nodes.map((node) => [node.id, node])),
    color: new Map<string, number>(),
    stack: [],
    position: new Map<string, number>(),
    errors,
  };
}

function walkCycle(nodeID: string, context: CycleWalkContext): boolean {
  enterCycleNode(nodeID, context);
  for (const nextID of context.outgoing.get(nodeID) ?? []) {
    if (walkCycleEdge(nextID, context)) {
      return true;
    }
  }
  leaveCycleNode(nodeID, context);
  return false;
}

function enterCycleNode(nodeID: string, context: CycleWalkContext): void {
  context.color.set(nodeID, 1);
  context.position.set(nodeID, context.stack.length);
  context.stack.push(nodeID);
}

function leaveCycleNode(nodeID: string, context: CycleWalkContext): void {
  context.color.set(nodeID, 2);
  context.stack.pop();
  context.position.delete(nodeID);
}

function walkCycleEdge(nextID: string, context: CycleWalkContext): boolean {
  const state = context.color.get(nextID) ?? 0;
  if (state === 0) {
    return walkCycle(nextID, context);
  }
  return state === 1 && rejectInvalidCycle(nextID, context);
}

function rejectInvalidCycle(nextID: string, context: CycleWalkContext): boolean {
  const cycle = context.stack.slice(context.position.get(nextID) ?? 0);
  if (cycle.some((id) => context.nodeMap.get(id)?.type === 'loop')) {
    return false;
  }
  context.errors.push(`workflow cycle must include a loop node: ${cycle.join(' -> ')}`);
  return true;
}

function walkGraph(startID: string, adjacency: Map<string, string[]>): Set<string> {
  const seen = new Set<string>();
  const queue = [startID];
  while (queue.length > 0) {
    const nodeID = queue.shift();
    if (!shouldVisitGraphNode(nodeID, seen)) {
      continue;
    }
    seen.add(nodeID);
    queue.push(...(adjacency.get(nodeID) ?? []));
  }
  return seen;
}

function shouldVisitGraphNode(
  nodeID: string | undefined,
  seen: Set<string>,
): nodeID is string {
  if (!nodeID) {
    return false;
  }
  return !seen.has(nodeID);
}
