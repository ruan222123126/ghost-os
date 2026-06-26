import {
  END_NODE_TYPE,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
  START_NODE_TYPE,
} from '@/lib/workflow-editor/constants';
import {
  pathExists,
  validateConnectivity,
  validateCycles,
} from '@/lib/workflow-editor/validationGraphTraversal';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';
import type { WorkflowGraphData } from '@/lib/workflow-editor/validation';

interface LoopPair {
  start?: WorkflowCanvasNodeDraft;
  end?: WorkflowCanvasNodeDraft;
  startCount: number;
  endCount: number;
}

interface DegreeContext {
  indegree: Map<string, number>;
  outdegree: Map<string, number>;
  errors: string[];
}

interface NodeDegree {
  node: WorkflowCanvasNodeDraft;
  inDegree: number;
  outDegree: number;
}

interface LoopPairEntry {
  loopID: string;
  role: string | undefined;
  node: WorkflowCanvasNodeDraft;
}

interface LoopPairCycleInput {
  loopID: string;
  start: WorkflowCanvasNodeDraft;
  end: WorkflowCanvasNodeDraft;
  outgoing: Map<string, string[]>;
  errors: string[];
}

export function validateWorkflowGraphRules(
  nodes: WorkflowCanvasNodeDraft[],
  graph: WorkflowGraphData,
  errors: string[],
): void {
  validateDegreeRules(nodes, {
    indegree: graph.indegree,
    outdegree: graph.outdegree,
    errors,
  });
  validateIfNodeEdgeTargets(nodes, graph.outgoing, errors);
  validateLoopPairs(nodes, graph.outgoing, errors);
  validateConnectivity({
    nodes,
    outgoing: graph.outgoing,
    incoming: graph.incoming,
    errors,
  });
  validateCycles(nodes, graph.outgoing, errors);
}

function validateDegreeRules(
  nodes: WorkflowCanvasNodeDraft[],
  context: DegreeContext,
): void {
  for (const node of nodes) {
    const degree = nodeDegree(node, context);
    if (validateBoundaryNodeDegree(degree, context.errors)) {
      continue;
    }
    if (node.type === 'if') {
      validateIfNodeDegree(degree, context.errors);
      continue;
    }
    if (node.type === 'loop') {
      validateLoopNodeDegree(degree, node.loop?.role, context.errors);
      continue;
    }
    validateRegularNodeDegree(degree, context.errors);
  }
}

function validateIfNodeDegree(
  degree: NodeDegree,
  errors: string[],
): void {
  if (degree.inDegree < 1 || degree.outDegree !== 2) {
    errors.push(`workflow node "${degree.node.id}" must have in>=1 and out=2`);
  }
}

function validateLoopNodeDegree(
  degree: NodeDegree,
  role: string | undefined,
  errors: string[],
): void {
  if (role === LOOP_ROLE_END) {
    validateSingleInSingleOutDegree(degree, errors);
    return;
  }
  if (role === LOOP_ROLE_START) {
    validateSingleInSingleOutDegree(degree, errors);
  }
}

function nodeDegree(node: WorkflowCanvasNodeDraft, context: DegreeContext): NodeDegree {
  return {
    node,
    inDegree: context.indegree.get(node.id) ?? 0,
    outDegree: context.outdegree.get(node.id) ?? 0,
  };
}

function validateBoundaryNodeDegree(degree: NodeDegree, errors: string[]): boolean {
  if (degree.node.type === START_NODE_TYPE) {
    validateStartNodeDegree(degree, errors);
    return true;
  }
  if (degree.node.type === END_NODE_TYPE) {
    validateEndNodeDegree(degree, errors);
    return true;
  }
  return false;
}

function validateStartNodeDegree(degree: NodeDegree, errors: string[]): void {
  if (degree.inDegree !== 0 || degree.outDegree < 1) {
    errors.push('start node must have in=0 and out>=1');
  }
}

function validateEndNodeDegree(degree: NodeDegree, errors: string[]): void {
  if (degree.inDegree < 1 || degree.outDegree !== 0) {
    errors.push('end node must have in>=1 and out=0');
  }
}

function validateRegularNodeDegree(degree: NodeDegree, errors: string[]): void {
  if (degree.node.type !== START_NODE_TYPE && degree.node.type !== END_NODE_TYPE) {
    validateSingleInSingleOutDegree(degree, errors);
  }
}

function validateSingleInSingleOutDegree(degree: NodeDegree, errors: string[]): void {
  if (degree.inDegree < 1 || degree.outDegree !== 1) {
    errors.push(`workflow node "${degree.node.id}" must have in>=1 and out=1`);
  }
}

function validateIfNodeEdgeTargets(
  nodes: WorkflowCanvasNodeDraft[],
  outgoing: Map<string, string[]>,
  errors: string[],
): void {
  for (const node of nodes) {
    const ifConfig = node.type === 'if' ? node.if : undefined;
    if (!ifConfig) {
      continue;
    }
    if (ifEdgeTargetsMismatch(ifConfig, outgoing.get(node.id) ?? [])) {
      errors.push(`workflow if node "${node.id}" outgoing edges must match true_node_id/false_node_id`);
    }
  }
}

function ifEdgeTargetsMismatch(
  ifConfig: NonNullable<WorkflowCanvasNodeDraft['if']>,
  outgoingIDs: string[],
): boolean {
  const outgoingSet = new Set(outgoingIDs);
  const trueID = ifConfig.true_node_id?.trim() ?? '';
  const falseID = ifConfig.false_node_id?.trim() ?? '';
  return Boolean(trueID && falseID && (!outgoingSet.has(trueID) || !outgoingSet.has(falseID)));
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
    validateLoopPairCycle({
      loopID,
      start: pair.start,
      end: pair.end,
      outgoing,
      errors,
    });
  }
}

function collectLoopPairs(nodes: WorkflowCanvasNodeDraft[]): Map<string, LoopPair> {
  const pairs = new Map<string, LoopPair>();
  for (const node of nodes) {
    const entry = loopPairEntry(node);
    if (!entry) {
      continue;
    }
    recordLoopPair(pairs, entry);
  }
  return pairs;
}

function loopPairEntry(node: WorkflowCanvasNodeDraft): LoopPairEntry | undefined {
  if (node.type !== 'loop') {
    return undefined;
  }
  const loopID = node.loop?.loop_id?.trim() ?? '';
  if (!loopID) {
    return undefined;
  }
  return { loopID, role: node.loop?.role, node };
}

function recordLoopPair(pairs: Map<string, LoopPair>, entry: LoopPairEntry): void {
  const pair = pairs.get(entry.loopID) ?? { startCount: 0, endCount: 0 };
  if (entry.role === LOOP_ROLE_END) {
    pair.endCount += 1;
    pair.end = pair.end ?? entry.node;
  } else {
    pair.startCount += 1;
    pair.start = pair.start ?? entry.node;
  }
  pairs.set(entry.loopID, pair);
}

function validateLoopPairCycle(input: LoopPairCycleInput): void {
  if (hasCompiledLoopEdges(input.start)) {
    input.errors.push(`loop "${input.loopID}" start/end pair should not set body_node_id/exit_node_id directly`);
    return;
  }
  validateLoopStartBranch(input);
  validateLoopEndExit(input);
}

function hasCompiledLoopEdges(start: WorkflowCanvasNodeDraft): boolean {
  return Boolean(start.loop?.body_node_id?.trim() || start.loop?.exit_node_id?.trim());
}

function validateLoopStartBranch(input: LoopPairCycleInput): void {
  const startOutgoing = input.outgoing.get(input.start.id) ?? [];
  if (startOutgoing.length !== 1 || !pathExists(input.outgoing, startOutgoing[0], input.end.id)) {
    input.errors.push(`loop "${input.loopID}" start branch must reach loop end node`);
  }
}

function validateLoopEndExit(input: LoopPairCycleInput): void {
  const endOutgoing = input.outgoing.get(input.end.id) ?? [];
  if (endOutgoing.length !== 1 || endOutgoing[0] === input.start.id) {
    input.errors.push(`loop "${input.loopID}" end node must contain exactly one exit edge`);
  }
}
