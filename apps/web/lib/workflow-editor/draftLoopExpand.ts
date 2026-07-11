import type {
  WorkflowDefinition,
  WorkflowNode,
} from '@/lib/types';
import {
  LOOP_ROLE_END,
  LOOP_ROLE_START,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeSource } from '@/lib/workflow-editor/draftNodeSource';

type WorkflowNodeToCanvasSource = (node: WorkflowNode) => WorkflowCanvasNodeSource;
type WorkflowEdge = WorkflowDefinition['edges'][number];
type WorkflowEdges = WorkflowDefinition['edges'];
type WorkflowLoopPayload = NonNullable<WorkflowNode['loop']>;
type LoopNodeMap = Map<string, WorkflowLoopPayload>;
type LoopBackSources = Map<string, Set<string>>;

interface ReachableQueueOptions {
  blockedID: string;
  nextNodeIDs?: string[];
  queue: string[];
  reachable: Set<string>;
}

export function expandLoopPairsForDraft(
  workflow: WorkflowDefinition,
  workflowNodeToCanvasSource: WorkflowNodeToCanvasSource,
): {
  nodes: WorkflowCanvasNodeSource[];
  edges: WorkflowDefinition['edges'];
} {
  const loopNodes: LoopNodeMap = new Map();
  const outgoing = buildOutgoingMap(workflow.edges);
  const nodes: WorkflowCanvasNodeSource[] = [];
  for (const node of workflow.nodes) {
    if (node.type !== 'loop' || !node.loop) {
      nodes.push(workflowNodeToCanvasSource(node));
      continue;
    }
    loopNodes.set(node.id, node.loop);
    nodes.push(buildLoopStartNodeSource(node.id, node.loop));
    nodes.push(buildLoopEndNodeSource(node.id));
  }
  return {
    nodes,
    edges: rewriteLoopEdgesForDraft(workflow.edges, loopNodes, outgoing),
  };
}

function buildLoopStartNodeSource(
  loopID: string,
  loop: WorkflowLoopPayload,
): WorkflowCanvasNodeSource {
  return {
    id: loopStartNodeID(loopID),
    type: 'loop',
    loop: {
      role: LOOP_ROLE_START,
      loop_id: loopID,
      max_iterations: loop.max_iterations,
    },
  };
}

function buildLoopEndNodeSource(loopID: string): WorkflowCanvasNodeSource {
  return {
    id: loopEndNodeID(loopID),
    type: 'loop',
    loop: {
      role: LOOP_ROLE_END,
      loop_id: loopID,
    },
  };
}

function rewriteLoopEdgesForDraft(
  edges: WorkflowEdges,
  loopNodes: LoopNodeMap,
  outgoing: Map<string, string[]>,
): WorkflowEdges {
  const loopBackSources = collectLoopBackSources(loopNodes, edges, outgoing);
  const rewritten: WorkflowEdges = [];
  for (const edge of edges) {
    rewritten.push(...rewriteLoopEdgeForDraft(edge, loopNodes, loopBackSources));
  }
  for (const [loopID, loop] of loopNodes.entries()) {
    rewritten.push(buildLoopExitEdge(loopID, loop));
  }
  return dedupeWorkflowEdges(rewritten);
}

function rewriteLoopEdgeForDraft(
  edge: WorkflowEdge,
  loopNodes: LoopNodeMap,
  loopBackSources: LoopBackSources,
): WorkflowEdges {
  const fromLoop = loopNodes.get(edge.from_node_id);
  if (fromLoop) {
    return rewriteLoopSourceEdge(edge, fromLoop);
  }

  if (!loopNodes.has(edge.to_node_id)) {
    return [edge];
  }

  return [rewriteLoopTargetEdge(edge, loopBackSources)];
}

function rewriteLoopSourceEdge(edge: WorkflowEdge, loop: WorkflowLoopPayload): WorkflowEdges {
  if (edge.to_node_id !== loop.body_node_id) {
    return [];
  }

  return [{
    from_node_id: loopStartNodeID(edge.from_node_id),
    to_node_id: loop.body_node_id,
  }];
}

function rewriteLoopTargetEdge(edge: WorkflowEdge, loopBackSources: LoopBackSources): WorkflowEdge {
  const loopID = edge.to_node_id;
  return {
    from_node_id: edge.from_node_id,
    to_node_id: isLoopBackSource(edge, loopBackSources) ? loopEndNodeID(loopID) : loopStartNodeID(loopID),
  };
}

function buildLoopExitEdge(loopID: string, loop: WorkflowLoopPayload): WorkflowEdge {
  return { from_node_id: loopEndNodeID(loopID), to_node_id: loop.exit_node_id };
}

function isLoopBackSource(edge: WorkflowEdge, loopBackSources: LoopBackSources): boolean {
  return loopBackSources.get(edge.to_node_id)?.has(edge.from_node_id) ?? false;
}

function collectLoopBackSources(
  loopNodes: LoopNodeMap,
  edges: WorkflowEdges,
  outgoing: Map<string, string[]>,
): LoopBackSources {
  const loopBackSources: LoopBackSources = new Map();
  for (const [loopID, loop] of loopNodes.entries()) {
    const reachableFromBody = collectReachableNodes(outgoing, loop.body_node_id, loopID);
    loopBackSources.set(
      loopID,
      collectReachableLoopIncomingSources(edges, loopID, reachableFromBody),
    );
  }
  return loopBackSources;
}

function collectReachableLoopIncomingSources(
  edges: WorkflowEdges,
  loopID: string,
  reachableFromBody: Set<string>,
): Set<string> {
  const sources = new Set<string>();
  for (const edge of edges) {
    if (edge.to_node_id === loopID && edge.from_node_id !== loopID && reachableFromBody.has(edge.from_node_id)) {
      sources.add(edge.from_node_id);
    }
  }
  return sources;
}

function collectReachableNodes(
  outgoing: Map<string, string[]>,
  startID: string,
  blockedID: string,
): Set<string> {
  const reachable = new Set<string>();
  const queue = [startID];
  while (queue.length > 0) {
    const nodeID = queue.shift();
    if (!shouldVisitReachableNode(nodeID, blockedID, reachable)) {
      continue;
    }
    reachable.add(nodeID);
    enqueueReachableNextNodes({
      blockedID,
      nextNodeIDs: outgoing.get(nodeID),
      queue,
      reachable,
    });
  }
  return reachable;
}

function shouldVisitReachableNode(
  nodeID: string | undefined,
  blockedID: string,
  reachable: Set<string>,
): nodeID is string {
  if (!nodeID) {
    return false;
  }
  return nodeID !== blockedID && !reachable.has(nodeID);
}

function enqueueReachableNextNodes(options: ReachableQueueOptions): void {
  const { blockedID, nextNodeIDs, queue, reachable } = options;

  for (const nextID of nextNodeIDs ?? []) {
    if (nextID !== blockedID && !reachable.has(nextID)) {
      queue.push(nextID);
    }
  }
}

function buildOutgoingMap(edges: WorkflowEdges): Map<string, string[]> {
  const outgoing = new Map<string, string[]>();
  for (const edge of edges) {
    outgoing.set(edge.from_node_id, [...(outgoing.get(edge.from_node_id) ?? []), edge.to_node_id]);
  }
  return outgoing;
}

function dedupeWorkflowEdges(edges: WorkflowEdges): WorkflowEdges {
  const seen = new Set<string>();
  const unique: WorkflowEdges = [];
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

function loopStartNodeID(loopID: string): string {
  return `${loopID}-${LOOP_ROLE_START}`;
}

function loopEndNodeID(loopID: string): string {
  return `${loopID}-${LOOP_ROLE_END}`;
}
