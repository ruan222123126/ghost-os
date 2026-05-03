import type { WorkflowCanvasNodeDraft, WorkflowNodeType } from '@/lib/workflow-editor/types';

export function isProtectedBoundaryNodeType(type: WorkflowNodeType): boolean {
  return type === 'start' || type === 'end';
}

export function isProtectedBoundaryNode(
  node: WorkflowCanvasNodeDraft | undefined,
): boolean {
  return Boolean(node && isProtectedBoundaryNodeType(node.type));
}
