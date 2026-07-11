import type { WorkflowCanvasDraft, WorkflowCanvasEdgeDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

type OrchestrationEdgeKind = NonNullable<WorkflowCanvasEdgeDraft['kind']>;

interface OrchestrationEdgeRule {
  kind: OrchestrationEdgeKind;
  sourceType: WorkflowCanvasNodeDraft['type'];
  targetType: WorkflowCanvasNodeDraft['type'];
}

interface OrchestrationEdgeRequest {
  kind: OrchestrationEdgeKind;
  sourceNodeID: string;
  targetNodeID: string;
}

interface OrchestrationEdgeNodes {
  sourceNode: WorkflowCanvasNodeDraft;
  targetNode: WorkflowCanvasNodeDraft;
}

const ORCHESTRATION_EDGE_RULES = [
  { sourceType: 'agent', targetType: 'group', kind: 'member' },
  { sourceType: 'group', targetType: 'group', kind: 'control' },
] as const satisfies readonly OrchestrationEdgeRule[];

export function canCreateOrchestrationEdge(
  sourceNode: WorkflowCanvasNodeDraft | undefined,
  targetNode: WorkflowCanvasNodeDraft | undefined,
): boolean {
  return resolveOrchestrationEdgeKind(sourceNode, targetNode) !== undefined;
}

export function resolveOrchestrationEdgeKind(
  sourceNode: WorkflowCanvasNodeDraft | undefined,
  targetNode: WorkflowCanvasNodeDraft | undefined,
): WorkflowCanvasEdgeDraft['kind'] | undefined {
  const nodes = resolveOrchestrationEdgeNodes(sourceNode, targetNode);
  if (!nodes) {
    return undefined;
  }

  return ORCHESTRATION_EDGE_RULES.find(
    (rule) => matchesOrchestrationEdgeRule(rule, nodes),
  )?.kind;
}

export function connectOrchestrationNodes(
  draft: WorkflowCanvasDraft,
  sourceNodeID: string,
  targetNodeID: string,
): WorkflowCanvasDraft {
  const sourceNode = draft.nodes.find((node) => node.id === sourceNodeID);
  const targetNode = draft.nodes.find((node) => node.id === targetNodeID);
  const kind = resolveOrchestrationEdgeKind(sourceNode, targetNode);
  if (!kind) {
    return draft;
  }
  const request = { sourceNodeID, targetNodeID, kind };
  if (hasOrchestrationEdge(draft, request)) {
    return draft;
  }

  return {
    ...draft,
    edges: [...draft.edges, buildOrchestrationEdge(draft, request)],
  };
}

function resolveOrchestrationEdgeNodes(
  sourceNode: WorkflowCanvasNodeDraft | undefined,
  targetNode: WorkflowCanvasNodeDraft | undefined,
): OrchestrationEdgeNodes | undefined {
  if (!sourceNode || !targetNode || sourceNode.id === targetNode.id) {
    return undefined;
  }
  return { sourceNode, targetNode };
}

function matchesOrchestrationEdgeRule(
  rule: OrchestrationEdgeRule,
  nodes: OrchestrationEdgeNodes,
): boolean {
  return (
    nodes.sourceNode.type === rule.sourceType &&
    nodes.targetNode.type === rule.targetType
  );
}

function hasOrchestrationEdge(
  draft: WorkflowCanvasDraft,
  request: OrchestrationEdgeRequest,
): boolean {
  return draft.edges.some((edge) => (
    edge.from_node_id === request.sourceNodeID &&
    edge.to_node_id === request.targetNodeID &&
    edge.kind === request.kind
  ));
}

function buildOrchestrationEdge(
  draft: WorkflowCanvasDraft,
  request: OrchestrationEdgeRequest,
): WorkflowCanvasEdgeDraft {
  return {
    id: `edge-${draft.edges.length + 1}-${request.sourceNodeID}-${request.targetNodeID}-${request.kind}`,
    from_node_id: request.sourceNodeID,
    to_node_id: request.targetNodeID,
    kind: request.kind,
  };
}
