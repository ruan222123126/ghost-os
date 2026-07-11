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

interface WorkflowEdgeConnection {
  fromNodeID: string;
  toNodeID: string;
}

interface WorkflowEdgeConnectionContext {
  graph: Omit<WorkflowGraphData, 'nodeMap'>;
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  edgeSet: Set<string>;
  errors: string[];
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
  const context = { graph, nodeMap, edgeSet, errors };
  for (const edge of edges) {
    connectEdge({ fromNodeID: edge.from_node_id, toNodeID: edge.to_node_id }, context);
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

function connectEdge(edge: WorkflowEdgeConnection, context: WorkflowEdgeConnectionContext): void {
  const edgeKey = workflowEdgeKey(edge);
  if (!validateEdgeEndpoints(edge, context, edgeKey)) {
    return;
  }
  context.edgeSet.add(edgeKey);
  applyEdgeConnection(edge, context.graph);
}

function validateEdgeEndpoints(
  edge: WorkflowEdgeConnection,
  context: WorkflowEdgeConnectionContext,
  edgeKey: string,
): boolean {
  if (edge.fromNodeID.length === 0 || edge.toNodeID.length === 0) {
    context.errors.push('workflow edge endpoints are required');
    return false;
  }
  if (edge.fromNodeID === edge.toNodeID) {
    context.errors.push(`workflow does not allow self-loop edge "${edge.fromNodeID}"`);
    return false;
  }
  if (!context.nodeMap.has(edge.fromNodeID)) {
    context.errors.push(`workflow edge references unknown from_node_id "${edge.fromNodeID}"`);
    return false;
  }
  if (!context.nodeMap.has(edge.toNodeID)) {
    context.errors.push(`workflow edge references unknown to_node_id "${edge.toNodeID}"`);
    return false;
  }
  if (context.edgeSet.has(edgeKey)) {
    context.errors.push(`duplicate workflow edge "${edgeKey}"`);
    return false;
  }
  return true;
}

function applyEdgeConnection(
  edge: WorkflowEdgeConnection,
  graph: Omit<WorkflowGraphData, 'nodeMap'>,
): void {
  graph.outgoing.set(edge.fromNodeID, [...(graph.outgoing.get(edge.fromNodeID) ?? []), edge.toNodeID]);
  graph.incoming.set(edge.toNodeID, [...(graph.incoming.get(edge.toNodeID) ?? []), edge.fromNodeID]);
  graph.outdegree.set(edge.fromNodeID, (graph.outdegree.get(edge.fromNodeID) ?? 0) + 1);
  graph.indegree.set(edge.toNodeID, (graph.indegree.get(edge.toNodeID) ?? 0) + 1);
}

function workflowEdgeKey(edge: WorkflowEdgeConnection): string {
  return `${edge.fromNodeID}->${edge.toNodeID}`;
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
