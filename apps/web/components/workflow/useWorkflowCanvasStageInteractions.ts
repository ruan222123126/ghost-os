'use client';

import {
  type Dispatch,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type RefObject,
  type SetStateAction,
  type WheelEvent as ReactWheelEvent,
} from 'react';
import {
  buildNodeMap,
  handleCanvasMouseMove,
  queueNodeMove,
  type DragState,
  type PendingNodeMove,
} from '@/components/workflow/workflowCanvasStageHelpers';
import {
  panViewportByWheel,
  zoomViewport,
  type CanvasViewport,
} from '@/components/workflow/workflowCanvasViewport';
import {
  type WorkflowNodeContextMenuState,
} from '@/components/workflow/WorkflowCanvasNodeContextMenu';
import { drawWorkflowCanvasBackground } from '@/components/workflow/workflowCanvasBackground';
import {
  DEFAULT_VIEWPORT,
  applyPendingNodeMoveToDraft,
  resolveCanvasClassName,
  shouldStartPan,
  type PanState,
} from '@/components/workflow/workflowCanvasStageInteractionUtils';
import type { WorkflowCanvasDraft, WorkflowCanvasPosition } from '@/lib/workflow-editor';

interface UseWorkflowCanvasStageInteractionsOptions {
  draft: WorkflowCanvasDraft;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
}

interface UseWorkflowCanvasStageInteractionsResult {
  canvasRef: RefObject<HTMLDivElement>;
  backgroundCanvasRef: RefObject<HTMLCanvasElement>;
  viewport: CanvasViewport;
  renderDraft: WorkflowCanvasDraft;
  nodeMap: ReturnType<typeof buildNodeMap>;
  canvasClassName: string;
  worldTransform: { transform: string };
  connectingSourceNodeID?: string;
  nodeContextMenu?: WorkflowNodeContextMenuState;
  setDragState: (state?: DragState) => void;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
  closeContextMenu: () => void;
  onCanvasMouseDown: (event: ReactMouseEvent<HTMLElement>) => void;
  onCanvasMouseMove: (event: ReactMouseEvent<HTMLElement>) => void;
  onCanvasMouseUp: () => void;
  onCanvasMouseLeave: () => void;
  onCanvasWheel: (event: ReactWheelEvent<HTMLElement>) => void;
  onCanvasClick: () => void;
  onOpenContextMenu: (event: ReactMouseEvent<HTMLElement>, nodeID: string) => void;
}

export function useWorkflowCanvasStageInteractions(
  options: UseWorkflowCanvasStageInteractionsOptions,
): UseWorkflowCanvasStageInteractionsResult {
  const { draft, onMoveNode, onSelectNode } = options;
  const [dragState, setDragState] = useState<DragState>();
  const [panState, setPanState] = useState<PanState>();
  const [viewport, setViewport] = useState<CanvasViewport>(DEFAULT_VIEWPORT);
  const [connectingSourceNodeID, setConnectingSourceNodeID] = useState<string>();
  const [nodeContextMenu, setNodeContextMenu] = useState<WorkflowNodeContextMenuState>();
  const [dragPreview, setDragPreview] = useState<PendingNodeMove>();
  const canvasRef = useRef<HTMLDivElement>(null);
  const backgroundCanvasRef = useRef<HTMLCanvasElement>(null);
  const frameRequestRef = useRef<number>();
  const pendingMoveRef = useRef<PendingNodeMove>();
  const committedMoveRef = useRef<PendingNodeMove>();
  const viewportRef = useRef<CanvasViewport>(DEFAULT_VIEWPORT);
  const renderDraft = useMemo(() => {
    return applyPendingNodeMoveToDraft(draft, dragPreview);
  }, [draft, dragPreview]);
  const nodeMap = useMemo(() => buildNodeMap(renderDraft.nodes), [renderDraft.nodes]);
  const worldTransform = useMemo(
    () => ({ transform: `translate(${viewport.offsetX}px, ${viewport.offsetY}px) scale(${viewport.scale})` }),
    [viewport],
  );

  useEffect(() => {
    const frameRequest = frameRequestRef;
    return () => {
      const frameID = frameRequest.current;
      if (frameID !== undefined) {
        cancelAnimationFrame(frameID);
      }
    };
  }, []);

  useEffect(() => {
    viewportRef.current = viewport;
    const canvas = canvasRef.current;
    const background = backgroundCanvasRef.current;
    if (!canvas || !background) {
      return;
    }
    drawWorkflowCanvasBackground(background, canvas, viewport);
  }, [viewport]);

  useEffect(() => {
    const canvas = canvasRef.current;
    const background = backgroundCanvasRef.current;
    if (!canvas || !background) {
      return;
    }

    const observer = new ResizeObserver(() => {
      drawWorkflowCanvasBackground(background, canvas, viewportRef.current);
    });
    observer.observe(canvas);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!nodeContextMenu) {
      return;
    }
    const exists = draft.nodes.some((node) => node.id === nodeContextMenu.nodeID);
    if (!exists) {
      setNodeContextMenu(undefined);
    }
  }, [draft.nodes, nodeContextMenu]);

  const releaseInteractions = useCallback(() => {
    const frameID = frameRequestRef.current;
    if (frameID !== undefined) {
      cancelAnimationFrame(frameID);
      frameRequestRef.current = undefined;
    }

    const pendingMove = pendingMoveRef.current;
    if (pendingMove) {
      committedMoveRef.current = pendingMove;
    }
    pendingMoveRef.current = undefined;

    const committedMove = committedMoveRef.current;
    if (committedMove) {
      onMoveNode(committedMove.nodeID, committedMove.position);
    }
    committedMoveRef.current = undefined;

    setDragPreview(undefined);
    setDragState(undefined);
    setPanState(undefined);
  }, [onMoveNode]);

  const handleCanvasMouseDown = useCallback((event: ReactMouseEvent<HTMLElement>) => {
    setNodeContextMenu(undefined);
    if (!shouldStartPan(event)) {
      return;
    }
    event.preventDefault();
    setPanState({
      startClientX: event.clientX,
      startClientY: event.clientY,
      startOffsetX: viewport.offsetX,
      startOffsetY: viewport.offsetY,
    });
  }, [viewport]);

  const handleCanvasMouseMoveEvent = useCallback((event: ReactMouseEvent<HTMLElement>) => {
    if (panState) {
      setViewport((current) => ({
        ...current,
        offsetX: panState.startOffsetX + (event.clientX - panState.startClientX),
        offsetY: panState.startOffsetY + (event.clientY - panState.startClientY),
      }));
      return;
    }

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
  }, [dragState, panState, viewport]);

  const handleCanvasWheel = useCallback((event: ReactWheelEvent<HTMLElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }

    event.preventDefault();
    if (event.shiftKey) {
      setViewport((current) => panViewportByWheel(current, event.deltaX, event.deltaY));
      return;
    }

    const rect = canvas.getBoundingClientRect();
    setViewport((current) => {
      return zoomViewport({
        viewport: current,
        clientX: event.clientX,
        clientY: event.clientY,
        rect,
        deltaY: event.deltaY,
      });
    });
  }, []);

  const handleCanvasClick = useCallback(() => {
    setNodeContextMenu(undefined);
    if (panState) {
      return;
    }
    onSelectNode(undefined);
    setConnectingSourceNodeID(undefined);
  }, [onSelectNode, panState]);

  const handleOpenContextMenu = useCallback((event: ReactMouseEvent<HTMLElement>, nodeID: string) => {
    event.preventDefault();
    event.stopPropagation();
    onSelectNode(nodeID);
    setConnectingSourceNodeID(undefined);
    setNodeContextMenu({
      nodeID,
      clientX: event.clientX,
      clientY: event.clientY,
    });
  }, [onSelectNode]);

  const handleChangeDragState = useCallback((state?: DragState) => {
    setDragState(state);
  }, []);

  const closeContextMenu = useCallback(() => {
    setNodeContextMenu(undefined);
  }, []);

  return {
    canvasRef,
    backgroundCanvasRef,
    viewport,
    renderDraft,
    nodeMap,
    canvasClassName: resolveCanvasClassName(Boolean(panState)),
    worldTransform,
    connectingSourceNodeID,
    nodeContextMenu,
    setDragState: handleChangeDragState,
    setConnectingSourceNodeID,
    closeContextMenu,
    onCanvasMouseDown: handleCanvasMouseDown,
    onCanvasMouseMove: handleCanvasMouseMoveEvent,
    onCanvasMouseUp: releaseInteractions,
    onCanvasMouseLeave: releaseInteractions,
    onCanvasWheel: handleCanvasWheel,
    onCanvasClick: handleCanvasClick,
    onOpenContextMenu: handleOpenContextMenu,
  };
}
