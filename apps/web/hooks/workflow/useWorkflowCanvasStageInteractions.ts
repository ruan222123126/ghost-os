'use client';

import {
  type Dispatch,
  useCallback,
  useMemo,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type RefObject,
  type SetStateAction,
} from 'react';
import {
  buildNodeMap,
  type DragState,
  type PendingNodeMove,
} from '@/components/workflow/workflowCanvasStageHelpers';
import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';
import {
  type WorkflowNodeContextMenuState,
} from '@/components/workflow/WorkflowCanvasNodeContextMenu';
import {
  applyPendingNodeMoveToDraft,
  resolveCanvasClassName,
} from '@/components/workflow/workflowCanvasStageInteractionUtils';
import type { WorkflowCanvasDraft, WorkflowCanvasPosition } from '@/lib/workflow-editor';
import { useWorkflowCanvasBackground } from '@/hooks/workflow/useWorkflowCanvasBackground';
import { useWorkflowCanvasDragCommit } from '@/hooks/workflow/useWorkflowCanvasDragCommit';
import {
  useWorkflowCanvasMouseMove,
  useWorkflowCanvasViewportInteractions,
} from '@/hooks/workflow/useWorkflowCanvasViewportInteractions';
import { useWorkflowNodeContextMenu } from '@/hooks/workflow/useWorkflowNodeContextMenu';

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
  onCanvasWheel: ReturnType<typeof useWorkflowCanvasViewportInteractions>['onCanvasWheel'];
  onCanvasClick: () => void;
  onOpenContextMenu: (event: ReactMouseEvent<HTMLElement>, nodeID: string) => void;
}

interface WorkflowCanvasRefs {
  backgroundCanvasRef: RefObject<HTMLCanvasElement>;
  canvasRef: RefObject<HTMLDivElement>;
}

interface WorkflowCanvasRenderState {
  canvasClassName: string;
  nodeMap: ReturnType<typeof buildNodeMap>;
  renderDraft: WorkflowCanvasDraft;
  worldTransform: { transform: string };
}

interface StageInteractionResultOptions {
  closeContextMenu: () => void;
  connectingSourceNodeID?: string;
  nodeContextMenu?: WorkflowNodeContextMenuState;
  onCanvasMouseMove: (event: ReactMouseEvent<HTMLElement>) => void;
  onOpenContextMenu: (event: ReactMouseEvent<HTMLElement>, nodeID: string) => void;
  refs: WorkflowCanvasRefs;
  releaseInteractions: () => void;
  renderState: WorkflowCanvasRenderState;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
  setDragState: (state?: DragState) => void;
  viewport: CanvasViewport;
  viewportInteractions: ReturnType<typeof useWorkflowCanvasViewportInteractions>;
}

interface WorkflowCanvasConnectionState extends ReturnType<typeof useWorkflowNodeContextMenu> {
  connectingSourceNodeID?: string;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}

interface WorkflowCanvasDragState {
  dragPreview?: PendingNodeMove;
  onDragMouseMove: (event: ReactMouseEvent<HTMLElement>) => void;
  releaseDrag: () => void;
  setDragState: (state?: DragState) => void;
}

export function useWorkflowCanvasStageInteractions(
  options: UseWorkflowCanvasStageInteractionsOptions,
): UseWorkflowCanvasStageInteractionsResult {
  const { draft, onMoveNode, onSelectNode } = options;
  const refs = useWorkflowCanvasRefs();
  const connectionState = useWorkflowCanvasConnectionState({ draft, onSelectNode });
  const viewportInteractions = useWorkflowCanvasViewportInteractions({
    canvasRef: refs.canvasRef,
    clearContextMenu: connectionState.clearContextMenu,
    onSelectNode,
    setConnectingSourceNodeID: connectionState.setConnectingSourceNodeID,
  });
  const dragState = useWorkflowCanvasDragState({
    canvasRef: refs.canvasRef,
    onMoveNode,
    viewport: viewportInteractions.viewport,
  });
  const onCanvasMouseMove = useWorkflowCanvasMouseMove(
    viewportInteractions.onPanMouseMove,
    dragState.onDragMouseMove,
  );
  useWorkflowCanvasBackground({ ...refs, viewport: viewportInteractions.viewport });
  const renderState = useWorkflowCanvasRenderState({
    draft,
    dragPreview: dragState.dragPreview,
    isPanning: viewportInteractions.isPanning,
    viewport: viewportInteractions.viewport,
  });
  const releaseInteractions = useWorkflowCanvasReleaseInteractions(
    dragState.releaseDrag,
    viewportInteractions.releasePan,
  );

  return buildStageInteractionResult({
    closeContextMenu: connectionState.closeContextMenu,
    connectingSourceNodeID: connectionState.connectingSourceNodeID,
    nodeContextMenu: connectionState.nodeContextMenu,
    onCanvasMouseMove,
    onOpenContextMenu: connectionState.onOpenContextMenu,
    refs,
    releaseInteractions,
    renderState,
    setConnectingSourceNodeID: connectionState.setConnectingSourceNodeID,
    setDragState: dragState.setDragState,
    viewport: viewportInteractions.viewport,
    viewportInteractions,
  });
}

function useWorkflowCanvasRefs(): WorkflowCanvasRefs {
  return {
    backgroundCanvasRef: useRef<HTMLCanvasElement>(null),
    canvasRef: useRef<HTMLDivElement>(null),
  };
}

function useWorkflowCanvasConnectionState(options: {
  draft: WorkflowCanvasDraft;
  onSelectNode: (nodeID?: string) => void;
}): WorkflowCanvasConnectionState {
  const { draft, onSelectNode } = options;
  const [connectingSourceNodeID, setConnectingSourceNodeID] = useState<string>();
  const contextMenu = useWorkflowNodeContextMenu({
    draft,
    onSelectNode,
    setConnectingSourceNodeID,
  });
  return { ...contextMenu, connectingSourceNodeID, setConnectingSourceNodeID };
}

function useWorkflowCanvasDragState(options: {
  canvasRef: RefObject<HTMLDivElement>;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  viewport: CanvasViewport;
}): WorkflowCanvasDragState {
  const { canvasRef, onMoveNode, viewport } = options;
  const [dragState, setDragState] = useState<DragState>();
  const [dragPreview, setDragPreview] = useState<PendingNodeMove>();
  const dragCommit = useWorkflowCanvasDragCommit({
    canvasRef,
    dragState,
    onMoveNode,
    setDragPreview,
    setDragState,
    viewport,
  });
  const handleChangeDragState = useCallback((state?: DragState) => {
    setDragState(state);
  }, []);
  return {
    dragPreview,
    onDragMouseMove: dragCommit.onDragMouseMove,
    releaseDrag: dragCommit.releaseDrag,
    setDragState: handleChangeDragState,
  };
}

function useWorkflowCanvasReleaseInteractions(
  releaseDrag: () => void,
  releasePan: () => void,
): () => void {
  return useCallback(() => {
    releaseDrag();
    releasePan();
  }, [releaseDrag, releasePan]);
}

function useWorkflowCanvasRenderState(options: {
  draft: WorkflowCanvasDraft;
  dragPreview?: PendingNodeMove;
  isPanning: boolean;
  viewport: CanvasViewport;
}): WorkflowCanvasRenderState {
  const { draft, dragPreview, isPanning, viewport } = options;
  const renderDraft = useMemo(() => {
    return applyPendingNodeMoveToDraft(draft, dragPreview);
  }, [draft, dragPreview]);
  const nodeMap = useMemo(() => buildNodeMap(renderDraft.nodes), [renderDraft.nodes]);
  const worldTransform = useMemo(
    () => ({ transform: `translate(${viewport.offsetX}px, ${viewport.offsetY}px) scale(${viewport.scale})` }),
    [viewport],
  );
  return {
    canvasClassName: resolveCanvasClassName(isPanning),
    nodeMap,
    renderDraft,
    worldTransform,
  };
}

function buildStageInteractionResult(
  options: StageInteractionResultOptions,
): UseWorkflowCanvasStageInteractionsResult {
  const {
    closeContextMenu,
    connectingSourceNodeID,
    nodeContextMenu,
    onCanvasMouseMove,
    onOpenContextMenu,
    refs,
    releaseInteractions,
    renderState,
    setConnectingSourceNodeID,
    setDragState,
    viewport,
    viewportInteractions,
  } = options;
  return {
    canvasRef: refs.canvasRef,
    backgroundCanvasRef: refs.backgroundCanvasRef,
    viewport,
    renderDraft: renderState.renderDraft,
    nodeMap: renderState.nodeMap,
    canvasClassName: renderState.canvasClassName,
    worldTransform: renderState.worldTransform,
    connectingSourceNodeID,
    nodeContextMenu,
    setDragState,
    setConnectingSourceNodeID,
    closeContextMenu,
    onCanvasMouseDown: viewportInteractions.onCanvasMouseDown,
    onCanvasMouseMove,
    onCanvasMouseUp: releaseInteractions,
    onCanvasMouseLeave: releaseInteractions,
    onCanvasWheel: viewportInteractions.onCanvasWheel,
    onCanvasClick: viewportInteractions.onCanvasClick,
    onOpenContextMenu,
  };
}
