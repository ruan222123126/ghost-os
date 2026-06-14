import type {
  Dispatch,
  MutableRefObject,
  MouseEvent,
  SetStateAction,
} from 'react';
import type {
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
} from '@/lib/workflow-editor';
import {
  pointerToWorldPoint,
  type CanvasViewport,
} from '@/components/workflow/workflowCanvasViewport';

export interface DragState {
  nodeID: string;
  offsetX: number;
  offsetY: number;
}

export interface PendingNodeMove {
  nodeID: string;
  position: WorkflowCanvasPosition;
}

const EDGE_HITBOX_WIDTH = 10;
const NODE_WIDTH = 256;
const PORT_Y_OFFSET = 52;

export function buildNodeMap(nodes: WorkflowCanvasNodeDraft[]): Map<string, WorkflowCanvasNodeDraft> {
  const map = new Map<string, WorkflowCanvasNodeDraft>();
  for (const node of nodes) {
    map.set(node.id, node);
  }
  return map;
}

export function handleSourcePortClick(
  event: MouseEvent<HTMLDivElement>,
  nodeID: string,
  onChangeConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>,
) {
  event.preventDefault();
  event.stopPropagation();
  onChangeConnectingSourceNodeID((current) => (current === nodeID ? undefined : nodeID));
}

export function handleTargetPortClick(options: {
  event: MouseEvent<HTMLDivElement>;
  nodeID: string;
  connectingSourceNodeID: string | undefined;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onChangeConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}) {
  const {
    event,
    nodeID,
    connectingSourceNodeID,
    onConnectNodes,
    onChangeConnectingSourceNodeID,
  } = options;
  event.preventDefault();
  event.stopPropagation();
  if (!connectingSourceNodeID || connectingSourceNodeID === nodeID) {
    return;
  }
  onConnectNodes(connectingSourceNodeID, nodeID);
  onChangeConnectingSourceNodeID(undefined);
}

export function startDrag(options: {
  event: MouseEvent<HTMLElement>;
  nodeID: string;
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  canvas: HTMLDivElement | null;
  viewport: CanvasViewport;
}): DragState | undefined {
  const {
    event,
    nodeID,
    nodeMap,
    canvas,
    viewport,
  } = options;
  const node = nodeMap.get(nodeID);
  if (!node || !canvas) {
    return undefined;
  }
  const position = pointerInCanvas({
    clientX: event.clientX,
    clientY: event.clientY,
    canvas,
    viewport,
  });
  return {
    nodeID,
    offsetX: position.x - node.position.x,
    offsetY: position.y - node.position.y,
  };
}

export function handleCanvasMouseMove(options: {
  event: MouseEvent<HTMLElement>;
  dragState: DragState | undefined;
  canvas: HTMLDivElement | null;
  viewport: CanvasViewport;
  onQueueMove: (nodeID: string, position: WorkflowCanvasPosition) => void;
}) {
  const {
    event,
    dragState,
    canvas,
    viewport,
    onQueueMove,
  } = options;
  if (!dragState || !canvas) {
    return;
  }
  const cursor = pointerInCanvas({
    clientX: event.clientX,
    clientY: event.clientY,
    canvas,
    viewport,
  });
  onQueueMove(dragState.nodeID, {
    x: cursor.x - dragState.offsetX,
    y: cursor.y - dragState.offsetY,
  });
}

export function queueNodeMove(options: {
  nodeID: string;
  position: WorkflowCanvasPosition;
  frameRequestRef: MutableRefObject<number | undefined>;
  pendingMoveRef: MutableRefObject<PendingNodeMove | undefined>;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
}) {
  const { nodeID, position, frameRequestRef, pendingMoveRef, onMoveNode } = options;
  pendingMoveRef.current = { nodeID, position };
  if (frameRequestRef.current !== undefined) {
    return;
  }
  frameRequestRef.current = requestAnimationFrame(() => {
    frameRequestRef.current = undefined;
    const pending = pendingMoveRef.current;
    if (!pending) {
      return;
    }
    pendingMoveRef.current = undefined;
    onMoveNode(pending.nodeID, pending.position);
  });
}

export function renderEdge(
  edge: WorkflowCanvasEdgeDraft,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
  onDeleteEdge: (edgeID: string) => void,
) {
  const source = nodeMap.get(edge.from_node_id);
  const target = nodeMap.get(edge.to_node_id);
  if (!source || !target) {
    return null;
  }
  const path = buildEdgePath(source.position, target.position);

  return (
    <g key={edge.id} className="workflow-arch-edge-group">
      <path
        d={path}
        className="workflow-arch-edge-curve"
        style={edge.kind === 'member' ? { strokeDasharray: '7 5' } : undefined}
      />
      <path
        d={path}
        className="workflow-arch-edge-hitbox"
        style={{ strokeWidth: EDGE_HITBOX_WIDTH }}
        onClick={(event) => {
          event.stopPropagation();
          onDeleteEdge(edge.id);
        }}
      />
    </g>
  );
}

export function pointerInCanvas(options: {
  clientX: number;
  clientY: number;
  canvas: HTMLDivElement;
  viewport: CanvasViewport;
}): WorkflowCanvasPosition {
  const {
    clientX,
    clientY,
    canvas,
    viewport,
  } = options;
  const rect = canvas.getBoundingClientRect();
  return pointerToWorldPoint({
    clientX,
    clientY,
    rect,
    viewport,
  });
}

function buildEdgePath(source: WorkflowCanvasPosition, target: WorkflowCanvasPosition): string {
  const startX = source.x + NODE_WIDTH;
  const startY = source.y + PORT_Y_OFFSET;
  const endX = target.x;
  const endY = target.y + PORT_Y_OFFSET;
  const cpX = startX + (endX - startX) * 0.5;
  return `M ${startX} ${startY} C ${cpX} ${startY}, ${cpX} ${endY}, ${endX} ${endY}`;
}
