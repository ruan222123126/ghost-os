import type { WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor/agentRuntime';
import {
  END_NODE_TYPE,
  START_NODE_TYPE,
} from '@/lib/workflow-editor/constants';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowValidationResult,
} from '@/lib/workflow-editor/types';
import { validateWorkflowGraphRules } from '@/lib/workflow-editor/validationGraph';
import { validateNodePayloads } from '@/lib/workflow-editor/validationPayload';

export interface WorkflowGraphData {
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  outgoing: Map<string, string[]>;
  incoming: Map<string, string[]>;
  indegree: Map<string, number>;
  outdegree: Map<string, number>;
}

export interface WorkflowValidationOptions {
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
}

export function validateWorkflowDraft(
  draft: WorkflowCanvasDraft,
  options?: WorkflowValidationOptions,
): WorkflowValidationResult {
  const errors: string[] = [];
  const graph = buildWorkflowGraphData(draft.nodes, draft.edges, errors);
  validateNodeKinds(draft.nodes, errors);
  validateNodePayloads(draft.nodes, errors, options?.agentRuntimeCatalog);
  validateWorkflowGraphRules(draft.nodes, graph, errors);
  return { valid: errors.length === 0, errors };
}

function buildWorkflowGraphData(
  nodes: WorkflowCanvasNodeDraft[],
  edges: WorkflowCanvasDraft['edges'],
  errors: string[],
): WorkflowGraphData {
  const nodeMap = new Map<string, WorkflowCanvasNodeDraft>();
  for (const node of nodes) {
    if (node.id.trim().length === 0) {
      errors.push('workflow node id is required');
      continue;
    }
    if (nodeMap.has(node.id)) {
      errors.push(`duplicate workflow node id "${node.id}"`);
      continue;
    }
    nodeMap.set(node.id, node);
  }
  const graph = createEmptyGraphData(nodes);
  const edgeSet = new Set<string>();
  for (const edge of edges) {
    connectEdge(edge.from_node_id, edge.to_node_id, graph, nodeMap, edgeSet, errors);
  }
  return { nodeMap, ...graph };
}

function createEmptyGraphData(nodes: WorkflowCanvasNodeDraft[]): Omit<WorkflowGraphData, 'nodeMap'> {
  const outgoing = new Map<string, string[]>();
  const incoming = new Map<string, string[]>();
  const indegree = new Map<string, number>();
  const outdegree = new Map<string, number>();
  for (const node of nodes) {
    outgoing.set(node.id, []);
    incoming.set(node.id, []);
    indegree.set(node.id, 0);
    outdegree.set(node.id, 0);
  }
  return { outgoing, incoming, indegree, outdegree };
}

function connectEdge(
  fromNodeID: string,
  toNodeID: string,
  graph: Omit<WorkflowGraphData, 'nodeMap'>,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
  edgeSet: Set<string>,
  errors: string[],
): void {
  if (fromNodeID.length === 0 || toNodeID.length === 0) {
    errors.push('workflow edge endpoints are required');
    return;
  }
  if (fromNodeID === toNodeID) {
    errors.push(`workflow does not allow self-loop edge "${fromNodeID}"`);
    return;
  }
  if (!nodeMap.has(fromNodeID)) {
    errors.push(`workflow edge references unknown from_node_id "${fromNodeID}"`);
    return;
  }
  if (!nodeMap.has(toNodeID)) {
    errors.push(`workflow edge references unknown to_node_id "${toNodeID}"`);
    return;
  }
  const edgeKey = `${fromNodeID}->${toNodeID}`;
  if (edgeSet.has(edgeKey)) {
    errors.push(`duplicate workflow edge "${edgeKey}"`);
    return;
  }
  edgeSet.add(edgeKey);
  graph.outgoing.set(fromNodeID, [...(graph.outgoing.get(fromNodeID) ?? []), toNodeID]);
  graph.incoming.set(toNodeID, [...(graph.incoming.get(toNodeID) ?? []), fromNodeID]);
  graph.outdegree.set(fromNodeID, (graph.outdegree.get(fromNodeID) ?? 0) + 1);
  graph.indegree.set(toNodeID, (graph.indegree.get(toNodeID) ?? 0) + 1);
}

function validateNodeKinds(nodes: WorkflowCanvasNodeDraft[], errors: string[]): void {
  const startCount = nodes.filter((node) => node.type === START_NODE_TYPE).length;
  const endCount = nodes.filter((node) => node.type === END_NODE_TYPE).length;
  if (startCount !== 1) {
    errors.push('workflow requires exactly 1 start node');
  }
  if (endCount !== 1) {
    errors.push('workflow requires exactly 1 end node');
  }
}
