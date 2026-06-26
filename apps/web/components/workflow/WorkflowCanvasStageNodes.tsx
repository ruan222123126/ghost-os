import type {
  Dispatch,
  MouseEvent,
  RefObject,
  SetStateAction,
} from 'react';
import { WorkflowCanvasNode } from '@/components/workflow/WorkflowCanvasNode';
import {
  handleSourcePortClick,
  handleTargetPortClick,
  startDrag,
  type DragState,
} from '@/components/workflow/workflowCanvasStageHelpers';
import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';
import { metadataForNodeType } from '@/components/workflow/workflowNodeMeta';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowEditorKind,
} from '@/lib/workflow-editor';
import { isProtectedBoundaryNode } from '@/lib/workflow-editor';
import { canCreateOrchestrationEdge } from '@/lib/orchestration-editor/graph';

interface WorkflowCanvasStageNodesProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  workflowCopy: WorkflowCopy;
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  canvasRef: RefObject<HTMLDivElement>;
  viewport: CanvasViewport;
  connectingSourceNodeID?: string;
  onSelectNode: (nodeID?: string) => void;
  onOpenContextMenu: (event: MouseEvent<HTMLElement>, nodeID: string) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onChangeDragState: (state?: DragState) => void;
  onChangeConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}

interface WorkflowCanvasStageNodeItemProps {
  context: WorkflowCanvasStageNodeContext;
  node: WorkflowCanvasNodeDraft;
  selected: boolean;
}

interface WorkflowCanvasStageNodeContext {
  editorKind: WorkflowEditorKind;
  workflowCopy: WorkflowCopy;
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  canvasRef: RefObject<HTMLDivElement>;
  viewport: CanvasViewport;
  connectingSourceNodeID?: string;
  sourceNode?: WorkflowCanvasNodeDraft;
  onSelectNode: (nodeID?: string) => void;
  onOpenContextMenu: (event: MouseEvent<HTMLElement>, nodeID: string) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onChangeDragState: (state?: DragState) => void;
  onChangeConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}

export function WorkflowCanvasStageNodes(props: WorkflowCanvasStageNodesProps) {
  const {
    draft,
    editorKind,
    workflowCopy,
    nodeMap,
    canvasRef,
    viewport,
    connectingSourceNodeID,
    onSelectNode,
    onOpenContextMenu,
    onConnectNodes,
    onChangeDragState,
    onChangeConnectingSourceNodeID,
  } = props;
  const context: WorkflowCanvasStageNodeContext = {
    editorKind,
    workflowCopy,
    nodeMap,
    canvasRef,
    viewport,
    connectingSourceNodeID,
    sourceNode: findConnectingSourceNode(draft.nodes, connectingSourceNodeID),
    onSelectNode,
    onOpenContextMenu,
    onConnectNodes,
    onChangeDragState,
    onChangeConnectingSourceNodeID,
  };

  return (
    <div className="workflow-arch-node-layer">
      {draft.nodes.map((node) => (
        <WorkflowCanvasStageNodeItem
          key={node.id}
          context={context}
          node={node}
          selected={draft.selectedNodeId === node.id}
        />
      ))}
    </div>
  );
}

function WorkflowCanvasStageNodeItem(props: WorkflowCanvasStageNodeItemProps) {
  const { context, node, selected } = props;

  return (
    <WorkflowCanvasNode
      editorKind={context.editorKind}
      node={node}
      metadata={metadataForNodeType(node.type, context.workflowCopy)}
      selected={selected}
      targetable={isNodeTargetable(node, context)}
      connectingSourceNodeID={context.connectingSourceNodeID}
      onSelectNode={context.onSelectNode}
      onOpenContextMenu={(event, nodeID) => handleNodeContextMenu({ event, node, nodeID, context })}
      onStartDrag={(event, nodeID) => handleNodeDragStart({ event, nodeID, context })}
      onClickSourcePort={(event, nodeID) =>
        handleSourcePortClick(event, nodeID, context.onChangeConnectingSourceNodeID)}
      onClickTargetPort={(event, nodeID) =>
        handleTargetPortClick({
          event,
          nodeID,
          connectingSourceNodeID: context.connectingSourceNodeID,
          onConnectNodes: context.onConnectNodes,
          onChangeConnectingSourceNodeID: context.onChangeConnectingSourceNodeID,
        })}
    />
  );
}

function findConnectingSourceNode(
  nodes: WorkflowCanvasNodeDraft[],
  connectingSourceNodeID: string | undefined,
): WorkflowCanvasNodeDraft | undefined {
  return nodes.find((item) => item.id === connectingSourceNodeID);
}

function isNodeTargetable(
  node: WorkflowCanvasNodeDraft,
  context: WorkflowCanvasStageNodeContext,
): boolean {
  if (!context.connectingSourceNodeID || context.connectingSourceNodeID === node.id) {
    return false;
  }
  if (context.editorKind === 'workflow') {
    return node.type !== 'start';
  }
  return canCreateOrchestrationEdge(context.sourceNode, node);
}

function handleNodeContextMenu(options: {
  event: MouseEvent<HTMLElement>;
  node: WorkflowCanvasNodeDraft;
  nodeID: string;
  context: WorkflowCanvasStageNodeContext;
}): void {
  const { event, node, nodeID, context } = options;

  if (isProtectedBoundaryNode(node)) {
    event.preventDefault();
    event.stopPropagation();
    context.onSelectNode(nodeID);
    return;
  }

  context.onOpenContextMenu(event, nodeID);
}

function handleNodeDragStart(options: {
  event: MouseEvent<HTMLElement>;
  nodeID: string;
  context: WorkflowCanvasStageNodeContext;
}): void {
  const { event, nodeID, context } = options;

  context.onChangeDragState(startDrag({
    event,
    nodeID,
    nodeMap: context.nodeMap,
    canvas: context.canvasRef.current,
    viewport: context.viewport,
  }));
}
