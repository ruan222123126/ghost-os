import type { MouseEvent as ReactMouseEvent } from 'react';
import type { PendingNodeMove } from '@/components/workflow/workflowCanvasStageHelpers';
import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';
import type { WorkflowCanvasDraft } from '@/lib/workflow-editor';

export interface PanState {
  startClientX: number;
  startClientY: number;
  startOffsetX: number;
  startOffsetY: number;
}

const PRIMARY_MOUSE_BUTTON = 0;
const MIDDLE_MOUSE_BUTTON = 1;

export const DEFAULT_VIEWPORT: CanvasViewport = {
  scale: 1,
  offsetX: 0,
  offsetY: 0,
};

export function shouldStartPan(event: ReactMouseEvent<HTMLElement>): boolean {
  if (event.button === MIDDLE_MOUSE_BUTTON) {
    return true;
  }
  if (event.button !== PRIMARY_MOUSE_BUTTON) {
    return false;
  }

  const target = event.target;
  if (!(target instanceof Element)) {
    return false;
  }
  return !target.closest('.workflow-arch-node, .workflow-arch-edge-hitbox');
}

export function applyPendingNodeMoveToDraft(
  draft: WorkflowCanvasDraft,
  pendingMove?: PendingNodeMove,
): WorkflowCanvasDraft {
  if (!pendingMove) {
    return draft;
  }

  let changed = false;
  const nodes = draft.nodes.map((node) => {
    if (node.id !== pendingMove.nodeID) {
      return node;
    }
    changed = true;
    return { ...node, position: pendingMove.position };
  });

  if (!changed) {
    return draft;
  }
  return { ...draft, nodes };
}

export function resolveCanvasClassName(isPanning: boolean): string {
  return isPanning
    ? 'workflow-arch-canvas workflow-arch-canvas--panning'
    : 'workflow-arch-canvas';
}
