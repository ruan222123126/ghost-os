'use client';

import {
  useCallback,
  useEffect,
  useRef,
  type Dispatch,
  type MouseEvent as ReactMouseEvent,
  type MutableRefObject,
  type RefObject,
  type SetStateAction,
} from 'react';
import {
  handleCanvasMouseMove,
  queueNodeMove,
  type DragState,
  type PendingNodeMove,
} from '@/components/workflow/workflowCanvasStageHelpers';
import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';
import type { WorkflowCanvasPosition } from '@/lib/workflow-editor';

interface WorkflowCanvasDragCommitOptions {
  canvasRef: RefObject<HTMLDivElement>;
  dragState?: DragState;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  setDragPreview: Dispatch<SetStateAction<PendingNodeMove | undefined>>;
  setDragState: Dispatch<SetStateAction<DragState | undefined>>;
  viewport: CanvasViewport;
}

interface WorkflowCanvasDragCommitControls {
  onDragMouseMove: (event: ReactMouseEvent<HTMLElement>) => void;
  releaseDrag: () => void;
}

export function useWorkflowCanvasDragCommit(
  options: WorkflowCanvasDragCommitOptions,
): WorkflowCanvasDragCommitControls {
  const {
    canvasRef,
    dragState,
    onMoveNode,
    setDragPreview,
    setDragState,
    viewport,
  } = options;
  const frameRequestRef = useRef<number>();
  const pendingMoveRef = useRef<PendingNodeMove>();
  const committedMoveRef = useRef<PendingNodeMove>();

  useCancelQueuedMoveOnUnmount(frameRequestRef);

  const releaseDrag = useCallback(() => {
    cancelQueuedFrame(frameRequestRef);
    stagePendingMoveForCommit(pendingMoveRef, committedMoveRef);
    commitStagedMove(committedMoveRef, onMoveNode);
    setDragPreview(undefined);
    setDragState(undefined);
  }, [onMoveNode, setDragPreview, setDragState]);

  const onDragMouseMove = useCallback((event: ReactMouseEvent<HTMLElement>) => {
    handleCanvasMouseMove({
      event,
      dragState,
      canvas: canvasRef.current,
      viewport,
      onQueueMove: (nodeID, position) => {
        committedMoveRef.current = { nodeID, position };
        queueNodeMove({
          nodeID,
          position,
          frameRequestRef,
          pendingMoveRef,
          onMoveNode: (previewNodeID, previewPosition) => {
            setDragPreview({
              nodeID: previewNodeID,
              position: previewPosition,
            });
          },
        });
      },
    });
  }, [canvasRef, dragState, setDragPreview, viewport]);

  return { onDragMouseMove, releaseDrag };
}

function useCancelQueuedMoveOnUnmount(frameRequestRef: MutableRefObject<number | undefined>): void {
  useEffect(() => {
    const frameRequest = frameRequestRef;
    return () => {
      cancelQueuedFrame(frameRequest);
    };
  }, [frameRequestRef]);
}

function cancelQueuedFrame(frameRequestRef: MutableRefObject<number | undefined>): void {
  const frameID = frameRequestRef.current;
  if (frameID === undefined) {
    return;
  }
  cancelAnimationFrame(frameID);
  frameRequestRef.current = undefined;
}

function stagePendingMoveForCommit(
  pendingMoveRef: MutableRefObject<PendingNodeMove | undefined>,
  committedMoveRef: MutableRefObject<PendingNodeMove | undefined>,
): void {
  const pendingMove = pendingMoveRef.current;
  if (pendingMove) {
    committedMoveRef.current = pendingMove;
  }
  pendingMoveRef.current = undefined;
}

function commitStagedMove(
  committedMoveRef: MutableRefObject<PendingNodeMove | undefined>,
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void,
): void {
  const committedMove = committedMoveRef.current;
  if (committedMove) {
    onMoveNode(committedMove.nodeID, committedMove.position);
  }
  committedMoveRef.current = undefined;
}
