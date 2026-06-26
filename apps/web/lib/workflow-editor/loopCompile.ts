import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
} from '@/lib/workflow-editor/constants';
import type {
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

interface LoopTransportResult {
  nodes: WorkflowCanvasNodeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
}

interface LoopNodePair {
  start?: WorkflowCanvasNodeDraft;
  end?: WorkflowCanvasNodeDraft;
}

interface LoopPairTopology {
  bodyNodeID: string;
  exitNodeID: string;
  incomingToEnd: WorkflowCanvasEdgeDraft[];
}

interface LoopNodePairEntry {
  loopID: string;
  role: string | undefined;
  node: WorkflowCanvasNodeDraft;
}

interface CompileLoopPairInput {
  loopID: string;
  startNode: WorkflowCanvasNodeDraft;
  endNode: WorkflowCanvasNodeDraft;
  edges: WorkflowCanvasEdgeDraft[];
}

interface ResolveLoopPairTopologyInput extends CompileLoopPairInput {}

export function compileLoopPairsForTransport(
  nodes: WorkflowCanvasNodeDraft[],
  edges: WorkflowCanvasEdgeDraft[],
): LoopTransportResult {
  const transportNodes = nodes.map((node) => cloneNode(node));
  const transportEdges = edges.map((edge) => ({ ...edge }));
  const loopPairs = indexLoopNodePairs(transportNodes);
  for (const [loopID, pair] of loopPairs.entries()) {
    if (!pair.start || !pair.end) {
      throw new Error(`loop "${loopID}" must contain one start node and one end node`);
    }
    const compiledEdges = compileSingleLoopPair({
      loopID,
      startNode: pair.start,
      endNode: pair.end,
      edges: transportEdges,
    });
    transportEdges.splice(0, transportEdges.length, ...compiledEdges);
  }
  return {
    nodes: transportNodes.filter((node) => !(node.type === 'loop' && node.loop?.role === LOOP_ROLE_END)),
    edges: transportEdges,
  };
}

function indexLoopNodePairs(nodes: WorkflowCanvasNodeDraft[]): Map<string, LoopNodePair> {
  const pairs = new Map<string, LoopNodePair>();
  for (const node of nodes) {
    const entry = loopNodePairEntry(node);
    if (!entry) {
      continue;
    }
    recordLoopNodePair(pairs, entry);
  }
  return pairs;
}

function loopNodePairEntry(node: WorkflowCanvasNodeDraft): LoopNodePairEntry | undefined {
  if (node.type !== 'loop') {
    return undefined;
  }
  const loopID = node.loop?.loop_id?.trim() ?? '';
  if (loopID.length === 0) {
    throw new Error(`loop node "${node.id}" requires loop_id`);
  }
  return { loopID, role: node.loop?.role, node };
}

function recordLoopNodePair(pairs: Map<string, LoopNodePair>, entry: LoopNodePairEntry): void {
  const pair = pairs.get(entry.loopID) ?? {};
  if (entry.role === LOOP_ROLE_END) {
    assertLoopNodeSlotEmpty(pair.end, entry.loopID, 'end');
    pair.end = entry.node;
  } else {
    assertLoopNodeSlotEmpty(pair.start, entry.loopID, 'start');
    pair.start = entry.node;
  }
  pairs.set(entry.loopID, pair);
}

function assertLoopNodeSlotEmpty(
  existing: WorkflowCanvasNodeDraft | undefined,
  loopID: string,
  role: 'start' | 'end',
): void {
  if (existing) {
    throw new Error(`loop "${loopID}" has multiple ${role} nodes`);
  }
}

function compileSingleLoopPair(input: CompileLoopPairInput): WorkflowCanvasEdgeDraft[] {
  const { edges, endNode, loopID, startNode } = input;
  const originalStartNodeID = startNode.id;
  assertLoopPairIDs(loopID, startNode.id, endNode.id);
  const topology = resolveLoopPairTopology(input);

  startNode.id = loopID;
  startNode.loop = {
    role: LOOP_ROLE_START,
    loop_id: loopID,
    max_iterations: resolveLoopIterations(startNode.loop?.max_iterations, endNode.loop?.max_iterations),
    body_node_id: topology.bodyNodeID,
    exit_node_id: topology.exitNodeID,
  };

  return compileLoopPairEdges({
    loopID,
    loopStartID: originalStartNodeID,
    loopEndID: endNode.id,
    exitNodeID: topology.exitNodeID,
    incomingToEnd: topology.incomingToEnd,
    edges,
  });
}

function assertLoopPairIDs(loopID: string, startNodeID: string, endNodeID: string): void {
  const expectedStartID = `${loopID}-${LOOP_ROLE_START}`;
  const expectedEndID = `${loopID}-${LOOP_ROLE_END}`;
  if (startNodeID !== expectedStartID || endNodeID !== expectedEndID) {
    throw new Error(`loop "${loopID}" nodes must use fixed ids "${expectedStartID}" / "${expectedEndID}"`);
  }
}

function resolveLoopPairTopology(input: ResolveLoopPairTopologyInput): LoopPairTopology {
  const { edges, endNode, loopID, startNode } = input;
  const startOutgoing = collectOutgoingEdges(edges, startNode.id);
  const endOutgoing = collectOutgoingEdges(edges, endNode.id);
  const incomingToEnd = collectIncomingEdges(edges, endNode.id);
  if (startOutgoing.length !== 1) {
    throw new Error(`loop "${loopID}" start node must have exactly one outgoing edge`);
  }
  if (endOutgoing.length !== 1 || endOutgoing[0].to_node_id === startNode.id) {
    throw new Error(`loop "${loopID}" end node must contain exactly one exit edge`);
  }
  if (incomingToEnd.length === 0) {
    throw new Error(`loop "${loopID}" end node must have at least one incoming edge`);
  }
  return {
    bodyNodeID: startOutgoing[0].to_node_id,
    exitNodeID: endOutgoing[0].to_node_id,
    incomingToEnd,
  };
}

function compileLoopPairEdges(options: {
  loopID: string;
  loopStartID: string;
  loopEndID: string;
  exitNodeID: string;
  incomingToEnd: WorkflowCanvasEdgeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
}): WorkflowCanvasEdgeDraft[] {
  const { loopID, loopStartID, loopEndID, exitNodeID, incomingToEnd, edges } = options;
  const rewired = edges
    .filter((edge) => edge.from_node_id !== loopEndID && edge.to_node_id !== loopEndID)
    .map((edge) => ({
      ...edge,
      from_node_id: edge.from_node_id === loopStartID ? loopID : edge.from_node_id,
      to_node_id: edge.to_node_id === loopStartID ? loopID : edge.to_node_id,
    }));
  for (const edge of incomingToEnd) {
    rewired.push({
      ...edge,
      id: `${edge.id}-loop-rewire`,
      to_node_id: loopID,
    });
  }
  rewired.push({
    id: `edge-loop-exit-${loopID}-${loopID}-${exitNodeID}`,
    from_node_id: loopID,
    to_node_id: exitNodeID,
  });
  return dedupeEdges(rewired);
}

function collectOutgoingEdges(edges: WorkflowCanvasEdgeDraft[], nodeID: string): WorkflowCanvasEdgeDraft[] {
  return edges.filter((edge) => edge.from_node_id === nodeID);
}

function collectIncomingEdges(edges: WorkflowCanvasEdgeDraft[], nodeID: string): WorkflowCanvasEdgeDraft[] {
  return edges.filter((edge) => edge.to_node_id === nodeID && edge.from_node_id !== nodeID);
}

function dedupeEdges(edges: WorkflowCanvasEdgeDraft[]): WorkflowCanvasEdgeDraft[] {
  const seen = new Set<string>();
  const unique: WorkflowCanvasEdgeDraft[] = [];
  for (const edge of edges) {
    const key = `${edge.from_node_id}->${edge.to_node_id}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    unique.push(edge);
  }
  return unique;
}

function resolveLoopIterations(...values: Array<number | undefined>): number {
  for (const value of values) {
    if (value !== undefined && Number.isFinite(value) && value > 0) {
      return Math.floor(value);
    }
  }
  return DEFAULT_LOOP_MAX_ITERATIONS;
}

function cloneNode(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  return {
    ...node,
    ui: cloneNodeUI(node),
    start: cloneStartNodePayload(node),
    tool: cloneToolNodePayload(node),
    llm: cloneLLMNodePayload(node),
    agent: cloneAgentNodePayload(node),
    if: cloneIfNodePayload(node),
    loop: cloneLoopNodePayload(node),
  };
}

function cloneNodeUI(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['ui'] {
  return { ...node.ui };
}

function cloneStartNodePayload(
  node: WorkflowCanvasNodeDraft,
): WorkflowCanvasNodeDraft['start'] {
  return node.start ? { inputs: node.start.inputs?.map((input) => ({ ...input })) } : undefined;
}

function cloneToolNodePayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['tool'] {
  if (!node.tool) {
    return undefined;
  }
  return {
    ...node.tool,
    arguments: node.tool.arguments ? { ...node.tool.arguments } : undefined,
  };
}

function cloneLLMNodePayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['llm'] {
  return node.llm ? { ...node.llm } : undefined;
}

function cloneAgentNodePayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['agent'] {
  if (!node.agent) {
    return undefined;
  }
  return {
    ...node.agent,
    runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent.runtime_overrides),
  };
}

function cloneIfNodePayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['if'] {
  return node.if ? { ...node.if } : undefined;
}

function cloneLoopNodePayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['loop'] {
  return node.loop ? { ...node.loop } : undefined;
}
