'use client';

import {
  useCallback,
  useState,
  type Dispatch,
  type MouseEvent as ReactMouseEvent,
  type RefObject,
  type SetStateAction,
  type WheelEvent as ReactWheelEvent,
} from 'react';
import {
  panViewportByWheel,
  zoomViewport,
  type CanvasViewport,
} from '@/components/workflow/workflowCanvasViewport';
import {
  DEFAULT_VIEWPORT,
  shouldStartPan,
  type PanState,
} from '@/components/workflow/workflowCanvasStageInteractionUtils';

interface WorkflowCanvasViewportInteractionOptions {
  canvasRef: RefObject<HTMLDivElement>;
  clearContextMenu: () => void;
  onSelectNode: (nodeID?: string) => void;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}

interface WorkflowCanvasViewportInteractionControls {
  isPanning: boolean;
  onCanvasClick: () => void;
  onCanvasMouseDown: (event: ReactMouseEvent<HTMLElement>) => void;
  onCanvasWheel: (event: ReactWheelEvent<HTMLElement>) => void;
  onPanMouseMove: (event: ReactMouseEvent<HTMLElement>) => boolean;
  releasePan: () => void;
  viewport: CanvasViewport;
}

export function useWorkflowCanvasViewportInteractions(
  options: WorkflowCanvasViewportInteractionOptions,
): WorkflowCanvasViewportInteractionControls {
  const { canvasRef, clearContextMenu, onSelectNode, setConnectingSourceNodeID } = options;
  const [viewport, setViewport] = useState<CanvasViewport>(DEFAULT_VIEWPORT);
  const [panState, setPanState] = useState<PanState>();
  const onPanMouseMove = usePanMouseMove(panState, setViewport);
  const onCanvasMouseDown = useCanvasMouseDown({
    clearContextMenu,
    setPanState,
    viewport,
  });
  const onCanvasWheel = useCanvasWheel(canvasRef, setViewport);
  const onCanvasClick = useCanvasClick({
    clearContextMenu,
    isPanning: Boolean(panState),
    onSelectNode,
    setConnectingSourceNodeID,
  });
  const releasePan = useCallback(() => {
    setPanState(undefined);
  }, []);

  return {
    isPanning: Boolean(panState),
    onCanvasClick,
    onCanvasMouseDown,
    onCanvasWheel,
    onPanMouseMove,
    releasePan,
    viewport,
  };
}

export function useWorkflowCanvasMouseMove(
  onPanMouseMove: (event: ReactMouseEvent<HTMLElement>) => boolean,
  onDragMouseMove: (event: ReactMouseEvent<HTMLElement>) => void,
): (event: ReactMouseEvent<HTMLElement>) => void {
  return useCallback((event) => {
    if (onPanMouseMove(event)) {
      return;
    }
    onDragMouseMove(event);
  }, [onDragMouseMove, onPanMouseMove]);
}

function useCanvasMouseDown(options: {
  clearContextMenu: () => void;
  setPanState: Dispatch<SetStateAction<PanState | undefined>>;
  viewport: CanvasViewport;
}): (event: ReactMouseEvent<HTMLElement>) => void {
  const { clearContextMenu, setPanState, viewport } = options;
  return useCallback((event) => {
    clearContextMenu();
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
  }, [clearContextMenu, setPanState, viewport]);
}

function usePanMouseMove(
  panState: PanState | undefined,
  setViewport: Dispatch<SetStateAction<CanvasViewport>>,
): (event: ReactMouseEvent<HTMLElement>) => boolean {
  return useCallback((event) => {
    if (!panState) {
      return false;
    }
    setViewport((current) => ({
      ...current,
      offsetX: panState.startOffsetX + (event.clientX - panState.startClientX),
      offsetY: panState.startOffsetY + (event.clientY - panState.startClientY),
    }));
    return true;
  }, [panState, setViewport]);
}

function useCanvasWheel(
  canvasRef: RefObject<HTMLDivElement>,
  setViewport: Dispatch<SetStateAction<CanvasViewport>>,
): (event: ReactWheelEvent<HTMLElement>) => void {
  return useCallback((event) => {
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
  }, [canvasRef, setViewport]);
}

function useCanvasClick(options: {
  clearContextMenu: () => void;
  isPanning: boolean;
  onSelectNode: (nodeID?: string) => void;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}): () => void {
  const { clearContextMenu, isPanning, onSelectNode, setConnectingSourceNodeID } = options;
  return useCallback(() => {
    clearContextMenu();
    if (isPanning) {
      return;
    }
    onSelectNode(undefined);
    setConnectingSourceNodeID(undefined);
  }, [clearContextMenu, isPanning, onSelectNode, setConnectingSourceNodeID]);
}
