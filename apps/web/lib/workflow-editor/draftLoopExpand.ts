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

export function expandLoopPairsForDraft(
  workflow: WorkflowDefinition,
  workflowNodeToCanvasSource: WorkflowNodeToCanvasSource,
): {
  nodes: WorkflowCanvasNodeSource[];
  edges: WorkflowDefinition['edges'];
} {
  const loopNodes = new Map<string, NonNullable<WorkflowNode['loop']>>();
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
  loop: NonNullable<WorkflowNode['loop']>,
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
  edges: WorkflowDefinition['edges'],
  loopNodes: Map<string, NonNullable<WorkflowNode['loop']>>,
  outgoing: Map<string, string[]>,
): WorkflowDefinition['edges'] {
  const loopBackSources = collectLoopBackSources(loopNodes, edges, outgoing);
  const rewritten: WorkflowDefinition['edges'] = [];
  for (const edge of edges) {
    const fromLoop = loopNodes.get(edge.from_node_id);
    if (fromLoop) {
      if (edge.to_node_id === fromLoop.body_node_id) {
        rewritten.push({
          from_node_id: loopStartNodeID(edge.from_node_id),
          to_node_id: fromLoop.body_node_id,
        });
      }
      continue;
    }
    const toLoop = loopNodes.get(edge.to_node_id);
    if (!toLoop) {
      rewritten.push(edge);
      continue;
    }
    const loopID = edge.to_node_id;
    const isLoopBack = loopBackSources.get(loopID)?.has(edge.from_node_id) ?? false;
    rewritten.push({
      from_node_id: edge.from_node_id,
      to_node_id: isLoopBack ? loopEndNodeID(loopID) : loopStartNodeID(loopID),
    });
  }
  for (const [loopID, loop] of loopNodes.entries()) {
    rewritten.push({ from_node_id: loopEndNodeID(loopID), to_node_id: loop.exit_node_id });
  }
  return dedupeWorkflowEdges(rewritten);
}

function collectLoopBackSources(
  loopNodes: Map<string, NonNullable<WorkflowNode['loop']>>,
  edges: WorkflowDefinition['edges'],
  outgoing: Map<string, string[]>,
): Map<string, Set<string>> {
  const loopBackSources = new Map<string, Set<string>>();
  for (const [loopID, loop] of loopNodes.entries()) {
    const incoming = edges
      .filter((edge) => edge.to_node_id === loopID)
      .map((edge) => edge.from_node_id)
      .filter((sourceID) => sourceID !== loopID);
    const reachableFromBody = collectReachableNodes(outgoing, loop.body_node_id, loopID);
    loopBackSources.set(
      loopID,
      new Set(incoming.filter((sourceID) => reachableFromBody.has(sourceID))),
    );
  }
  return loopBackSources;
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
    if (!nodeID || nodeID === blockedID || reachable.has(nodeID)) {
      continue;
    }
    reachable.add(nodeID);
    for (const nextID of outgoing.get(nodeID) ?? []) {
      if (nextID !== blockedID && !reachable.has(nextID)) {
        queue.push(nextID);
      }
    }
  }
  return reachable;
}

function buildOutgoingMap(edges: WorkflowDefinition['edges']): Map<string, string[]> {
  const outgoing = new Map<string, string[]>();
  for (const edge of edges) {
    outgoing.set(edge.from_node_id, [...(outgoing.get(edge.from_node_id) ?? []), edge.to_node_id]);
  }
  return outgoing;
}

function dedupeWorkflowEdges(edges: WorkflowDefinition['edges']): WorkflowDefinition['edges'] {
  const seen = new Set<string>();
  const unique: WorkflowDefinition['edges'] = [];
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
