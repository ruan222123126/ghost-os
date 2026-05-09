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
  const sourceNode = draft.nodes.find((item) => item.id === connectingSourceNodeID);

  return (
    <div className="workflow-arch-node-layer">
      {draft.nodes.map((node) => (
        <WorkflowCanvasNode
          key={node.id}
          editorKind={editorKind}
          node={node}
          metadata={metadataForNodeType(node.type, workflowCopy)}
          selected={draft.selectedNodeId === node.id}
          targetable={Boolean(
            connectingSourceNodeID
            && connectingSourceNodeID !== node.id
            && (
              editorKind === 'workflow'
                ? node.type !== 'start'
                : canCreateOrchestrationEdge(sourceNode, node)
            )
          )}
          connectingSourceNodeID={connectingSourceNodeID}
          onSelectNode={onSelectNode}
          onOpenContextMenu={(event, nodeID) => {
            if (isProtectedBoundaryNode(node)) {
              event.preventDefault();
              event.stopPropagation();
              onSelectNode(nodeID);
              return;
            }
            onOpenContextMenu(event, nodeID);
          }}
          onStartDrag={(event, nodeID) =>
            onChangeDragState(startDrag({
              event,
              nodeID,
              nodeMap,
              canvas: canvasRef.current,
              viewport,
            }))}
          onClickSourcePort={(event, nodeID) => handleSourcePortClick(event, nodeID, onChangeConnectingSourceNodeID)}
          onClickTargetPort={(event, nodeID) =>
            handleTargetPortClick({
              event,
              nodeID,
              connectingSourceNodeID,
              onConnectNodes,
              onChangeConnectingSourceNodeID,
            })}
        />
      ))}
    </div>
  );
}
