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

interface WorkflowCanvasDragRefs {
  committedMoveRef: MutableRefObject<PendingNodeMove | undefined>;
  frameRequestRef: MutableRefObject<number | undefined>;
  pendingMoveRef: MutableRefObject<PendingNodeMove | undefined>;
}

export function useWorkflowCanvasDragCommit(
  options: WorkflowCanvasDragCommitOptions,
): WorkflowCanvasDragCommitControls {
  const dragRefs = useWorkflowCanvasDragRefs();
  const releaseDrag = useReleaseCanvasDrag({ ...options, dragRefs });
  const onDragMouseMove = useCanvasDragMouseMove({ ...options, dragRefs });

  return { onDragMouseMove, releaseDrag };
}

function useWorkflowCanvasDragRefs(): WorkflowCanvasDragRefs {
  const dragRefs = {
    committedMoveRef: useRef<PendingNodeMove>(),
    frameRequestRef: useRef<number>(),
    pendingMoveRef: useRef<PendingNodeMove>(),
  };

  useCancelQueuedMoveOnUnmount(dragRefs.frameRequestRef);

  return dragRefs;
}

function useReleaseCanvasDrag(options: WorkflowCanvasDragCommitOptions & {
  dragRefs: WorkflowCanvasDragRefs;
}): () => void {
  const { dragRefs, onMoveNode, setDragPreview, setDragState } = options;
  const { committedMoveRef, frameRequestRef, pendingMoveRef } = dragRefs;

  return useCallback(() => {
    cancelQueuedFrame(frameRequestRef);
    stagePendingMoveForCommit(pendingMoveRef, committedMoveRef);
    commitStagedMove(committedMoveRef, onMoveNode);
    setDragPreview(undefined);
    setDragState(undefined);
  }, [committedMoveRef, frameRequestRef, onMoveNode, pendingMoveRef, setDragPreview, setDragState]);
}

function useCanvasDragMouseMove(options: WorkflowCanvasDragCommitOptions & {
  dragRefs: WorkflowCanvasDragRefs;
}): (event: ReactMouseEvent<HTMLElement>) => void {
  const { canvasRef, dragRefs, dragState, setDragPreview, viewport } = options;
  const queueMove = useQueuedCanvasNodeMove(dragRefs, setDragPreview);

  return useCallback((event: ReactMouseEvent<HTMLElement>) => {
    handleCanvasMouseMove({
      event,
      dragState,
      canvas: canvasRef.current,
      viewport,
      onQueueMove: queueMove,
    });
  }, [canvasRef, dragState, queueMove, viewport]);
}

function useQueuedCanvasNodeMove(
  dragRefs: WorkflowCanvasDragRefs,
  setDragPreview: Dispatch<SetStateAction<PendingNodeMove | undefined>>,
): (nodeID: string, position: WorkflowCanvasPosition) => void {
  const { committedMoveRef, frameRequestRef, pendingMoveRef } = dragRefs;

  return useCallback((nodeID, position) => {
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
  }, [committedMoveRef, frameRequestRef, pendingMoveRef, setDragPreview]);
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
