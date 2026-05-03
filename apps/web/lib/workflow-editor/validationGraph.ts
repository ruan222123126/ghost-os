import {
  END_NODE_TYPE,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
  START_NODE_TYPE,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';
import type { WorkflowGraphData } from '@/lib/workflow-editor/validation';

interface LoopPair {
  start?: WorkflowCanvasNodeDraft;
  end?: WorkflowCanvasNodeDraft;
  startCount: number;
  endCount: number;
}

export function validateWorkflowGraphRules(
  nodes: WorkflowCanvasNodeDraft[],
  graph: WorkflowGraphData,
  errors: string[],
): void {
  validateDegreeRules(nodes, graph.indegree, graph.outdegree, errors);
  validateIfNodeEdgeTargets(nodes, graph.outgoing, errors);
  validateLoopPairs(nodes, graph.outgoing, errors);
  validateConnectivity(nodes, graph.outgoing, graph.incoming, errors);
  validateCycles(nodes, graph.outgoing, errors);
}

function validateDegreeRules(
  nodes: WorkflowCanvasNodeDraft[],
  indegree: Map<string, number>,
  outdegree: Map<string, number>,
  errors: string[],
): void {
  for (const node of nodes) {
    const inDegree = indegree.get(node.id) ?? 0;
    const outDegree = outdegree.get(node.id) ?? 0;
    if (node.type === START_NODE_TYPE && (inDegree !== 0 || outDegree < 1)) {
      errors.push('start node must have in=0 and out>=1');
      continue;
    }
    if (node.type === END_NODE_TYPE && (inDegree < 1 || outDegree !== 0)) {
      errors.push('end node must have in>=1 and out=0');
      continue;
    }
    if (node.type === 'if') {
      validateIfNodeDegree(node.id, inDegree, outDegree, errors);
      continue;
    }
    if (node.type === 'loop') {
      validateLoopNodeDegree(node.id, node.loop?.role, inDegree, outDegree, errors);
      continue;
    }
    if (node.type !== START_NODE_TYPE && node.type !== END_NODE_TYPE && (inDegree < 1 || outDegree !== 1)) {
      errors.push(`workflow node "${node.id}" must have in>=1 and out=1`);
    }
  }
}

function validateIfNodeDegree(
  nodeID: string,
  inDegree: number,
  outDegree: number,
  errors: string[],
): void {
  if (inDegree < 1 || outDegree !== 2) {
    errors.push(`workflow node "${nodeID}" must have in>=1 and out=2`);
  }
}

function validateLoopNodeDegree(
  nodeID: string,
  role: string | undefined,
  inDegree: number,
  outDegree: number,
  errors: string[],
): void {
  if (role === LOOP_ROLE_END) {
    if (inDegree < 1 || outDegree !== 1) {
      errors.push(`workflow node "${nodeID}" must have in>=1 and out=1`);
    }
    return;
  }
  if (role === LOOP_ROLE_START && (inDegree < 1 || outDegree !== 1)) {
    errors.push(`workflow node "${nodeID}" must have in>=1 and out=1`);
  }
}

function validateIfNodeEdgeTargets(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  for (const node of nodes) {
    if (node.type !== 'if' || !node.if) {
      continue;
    }
    const outgoingSet = new Set(outgoing.get(node.id) ?? []);
    const trueID = node.if.true_node_id?.trim() ?? '';
    const falseID = node.if.false_node_id?.trim() ?? '';
    if (trueID && falseID && (!outgoingSet.has(trueID) || !outgoingSet.has(falseID))) {
      errors.push(`workflow if node "${node.id}" outgoing edges must match true_node_id/false_node_id`);
    }
  }
}

function validateLoopPairs(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  const pairs = collectLoopPairs(nodes);
  for (const [loopID, pair] of pairs.entries()) {
    if (pair.startCount !== 1 || pair.endCount !== 1 || !pair.start || !pair.end) {
      errors.push(`loop "${loopID}" must contain one start node and one end node`);
      continue;
    }
    const expectedStartID = `${loopID}-${LOOP_ROLE_START}`;
    const expectedEndID = `${loopID}-${LOOP_ROLE_END}`;
    if (pair.start.id !== expectedStartID || pair.end.id !== expectedEndID) {
      errors.push(`loop "${loopID}" must use fixed node ids "${expectedStartID}" and "${expectedEndID}"`);
    }
    validateLoopPairCycle(loopID, pair.start, pair.end, outgoing, errors);
  }
}

function collectLoopPairs(nodes: WorkflowCanvasNodeDraft[]): Map<string, LoopPair> {
  const pairs = new Map<string, LoopPair>();
  for (const node of nodes) {
    if (node.type !== 'loop' || !node.loop?.loop_id?.trim()) {
      continue;
    }
    const loopID = node.loop.loop_id.trim();
    const pair = pairs.get(loopID) ?? { startCount: 0, endCount: 0 };
    if (node.loop.role === LOOP_ROLE_END) {
      pair.endCount += 1;
      pair.end = pair.end ?? node;
    } else {
      pair.startCount += 1;
      pair.start = pair.start ?? node;
    }
    pairs.set(loopID, pair);
  }
  return pairs;
}

function validateLoopPairCycle(
  loopID: string,
  start: WorkflowCanvasNodeDraft,
  end: WorkflowCanvasNodeDraft,
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  if (start.loop?.body_node_id?.trim() || start.loop?.exit_node_id?.trim()) {
    errors.push(`loop "${loopID}" start/end pair should not set body_node_id/exit_node_id directly`);
    return;
  }
  const startOutgoing = outgoing.get(start.id) ?? [];
  const endOutgoing = outgoing.get(end.id) ?? [];
  if (startOutgoing.length !== 1 || !pathExists(outgoing, startOutgoing[0], end.id)) {
    errors.push(`loop "${loopID}" start branch must reach loop end node`);
  }
  if (endOutgoing.length !== 1 || endOutgoing[0] === start.id) {
    errors.push(`loop "${loopID}" end node must contain exactly one exit edge`);
  }
}

function validateConnectivity(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  incoming: Map<string, string[]>,
  errors: string[],
): void {
  const startNode = nodes.find((node) => node.type === START_NODE_TYPE);
  const endNode = nodes.find((node) => node.type === END_NODE_TYPE);
  if (!startNode || !endNode) {
    return;
  }
  const fromStart = walkGraph(startNode.id, outgoing);
  if (fromStart.size !== nodes.length) {
    errors.push('workflow must be fully connected from start node');
  }
  const toEnd = walkGraph(endNode.id, incoming);
  for (const node of nodes) {
    if (!toEnd.has(node.id)) {
      errors.push(`workflow node "${node.id}" cannot reach end node`);
    }
  }
}

function validateCycles(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  const color = new Map<string, number>();
  const stack: string[] = [];
  const position = new Map<string, number>();
  for (const node of nodes) {
    if ((color.get(node.id) ?? 0) !== 0) {
      continue;
    }
    if (walkCycle(node.id, outgoing, nodeMap, color, stack, position, errors)) {
      return;
    }
  }
}

function walkCycle(
  nodeID: string,
  outgoing: Map<string, string[]>,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
  color: Map<string, number>,
  stack: string[],
  position: Map<string, number>,
  errors: string[],
): boolean {
  color.set(nodeID, 1);
  position.set(nodeID, stack.length);
  stack.push(nodeID);
  for (const nextID of outgoing.get(nodeID) ?? []) {
    const state = color.get(nextID) ?? 0;
    if (state === 0 && walkCycle(nextID, outgoing, nodeMap, color, stack, position, errors)) {
      return true;
    }
    if (state === 1) {
      const cycle = stack.slice(position.get(nextID) ?? 0);
      if (!cycle.some((id) => nodeMap.get(id)?.type === 'loop')) {
        errors.push(`workflow cycle must include a loop node: ${cycle.join(' -> ')}`);
        return true;
      }
    }
  }
  color.set(nodeID, 2);
  stack.pop();
  position.delete(nodeID);
  return false;
}

function walkGraph(startID: string, adjacency: Map<string, string[]>): Set<string> {
  const seen = new Set<string>();
  const queue = [startID];
  while (queue.length > 0) {
    const nodeID = queue.shift();
    if (!nodeID || seen.has(nodeID)) {
      continue;
    }
    seen.add(nodeID);
    queue.push(...(adjacency.get(nodeID) ?? []));
  }
  return seen;
}

function pathExists(adjacency: Map<string, string[]>, startID: string, targetID: string): boolean {
  if (startID === targetID) {
    return true;
  }
  const seen = new Set<string>();
  const queue = [startID];
  while (queue.length > 0) {
    const nodeID = queue.shift();
    if (!nodeID || seen.has(nodeID)) {
      continue;
    }
    if (nodeID === targetID) {
      return true;
    }
    seen.add(nodeID);
    queue.push(...(adjacency.get(nodeID) ?? []));
  }
  return false;
}
