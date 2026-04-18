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
    const compiledEdges = compileSingleLoopPair(loopID, pair.start, pair.end, transportEdges);
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
    if (node.type !== 'loop') {
      continue;
    }
    const loopID = node.loop?.loop_id?.trim() ?? '';
    if (loopID.length === 0) {
      throw new Error(`loop node "${node.id}" requires loop_id`);
    }
    const pair = pairs.get(loopID) ?? {};
    if (node.loop?.role === LOOP_ROLE_END) {
      if (pair.end) {
        throw new Error(`loop "${loopID}" has multiple end nodes`);
      }
      pair.end = node;
    } else {
      if (pair.start) {
        throw new Error(`loop "${loopID}" has multiple start nodes`);
      }
      pair.start = node;
    }
    pairs.set(loopID, pair);
  }
  return pairs;
}

function compileSingleLoopPair(
  loopID: string,
  startNode: WorkflowCanvasNodeDraft,
  endNode: WorkflowCanvasNodeDraft,
  edges: WorkflowCanvasEdgeDraft[],
): WorkflowCanvasEdgeDraft[] {
  const originalStartNodeID = startNode.id;
  const expectedStartID = `${loopID}-${LOOP_ROLE_START}`;
  const expectedEndID = `${loopID}-${LOOP_ROLE_END}`;
  if (startNode.id !== expectedStartID || endNode.id !== expectedEndID) {
    throw new Error(`loop "${loopID}" nodes must use fixed ids "${expectedStartID}" / "${expectedEndID}"`);
  }
  const startOutgoing = collectOutgoingEdges(edges, startNode.id);
  const endOutgoing = collectOutgoingEdges(edges, endNode.id);
  const incomingToEnd = collectIncomingEdges(edges, endNode.id);
  if (startOutgoing.length !== 1) {
    throw new Error(`loop "${loopID}" start node must have exactly one outgoing edge`);
  }
  if (endOutgoing.length !== 2) {
    throw new Error(`loop "${loopID}" end node must have exactly two outgoing edges`);
  }
  if (incomingToEnd.length === 0) {
    throw new Error(`loop "${loopID}" end node must have at least one incoming edge`);
  }
  const loopBackEdge = endOutgoing.find((edge) => edge.to_node_id === startNode.id);
  if (!loopBackEdge) {
    throw new Error(`loop "${loopID}" end node must connect back to the loop start node`);
  }
  const exitEdge = endOutgoing.find((edge) => edge.to_node_id !== startNode.id);
  if (!exitEdge) {
    throw new Error(`loop "${loopID}" end node must contain an exit edge`);
  }

  startNode.id = loopID;
  const bodyNodeID = startOutgoing[0].to_node_id;
  startNode.loop = {
    role: LOOP_ROLE_START,
    loop_id: loopID,
    max_iterations: resolveLoopIterations(startNode.loop?.max_iterations, endNode.loop?.max_iterations),
    body_node_id: bodyNodeID,
    exit_node_id: exitEdge.to_node_id,
  };

  const rewired = edges
    .filter((edge) => edge.from_node_id !== endNode.id && edge.to_node_id !== endNode.id)
    .map((edge) => ({
      ...edge,
      from_node_id: edge.from_node_id === originalStartNodeID ? loopID : edge.from_node_id,
      to_node_id: edge.to_node_id === originalStartNodeID ? loopID : edge.to_node_id,
    }));
  for (const edge of incomingToEnd) {
    rewired.push({
      ...edge,
      id: `${edge.id}-loop-rewire`,
      to_node_id: loopID,
    });
  }
  rewired.push({
    id: `edge-loop-exit-${loopID}-${loopID}-${exitEdge.to_node_id}`,
    from_node_id: loopID,
    to_node_id: exitEdge.to_node_id,
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
    ui: { ...node.ui },
    start: node.start ? { inputs: node.start.inputs?.map((input) => ({ ...input })) } : undefined,
    tool: node.tool ? { ...node.tool, arguments: node.tool.arguments ? { ...node.tool.arguments } : undefined } : undefined,
    llm: node.llm ? { ...node.llm } : undefined,
    agent: node.agent ? { ...node.agent } : undefined,
    if: node.if ? { ...node.if } : undefined,
    loop: node.loop ? { ...node.loop } : undefined,
  };
}
