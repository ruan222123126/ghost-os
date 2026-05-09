import type { WorkflowCanvasDraft, WorkflowCanvasEdgeDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

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
  if (!sourceNode || !targetNode || sourceNode.id === targetNode.id) {
    return undefined;
  }
  if (sourceNode.type === 'agent' && targetNode.type === 'group') {
    return 'member';
  }
  if (sourceNode.type === 'group' && targetNode.type === 'group') {
    return 'control';
  }
  return undefined;
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
  const duplicate = draft.edges.some(
    (edge) => edge.from_node_id === sourceNodeID && edge.to_node_id === targetNodeID && edge.kind === kind,
  );
  if (duplicate) {
    return draft;
  }
  return {
    ...draft,
    edges: [
      ...draft.edges,
      {
        id: `edge-${draft.edges.length + 1}-${sourceNodeID}-${targetNodeID}-${kind}`,
        from_node_id: sourceNodeID,
        to_node_id: targetNodeID,
        kind,
      },
    ],
  };
}
