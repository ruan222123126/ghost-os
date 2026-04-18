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
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor';

interface WorkflowCanvasStageNodesProps {
  draft: WorkflowCanvasDraft;
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
  const { copy } = useWebLocale();
  const {
    draft,
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

  return (
    <div className="workflow-arch-node-layer">
      {draft.nodes.map((node) => (
        <WorkflowCanvasNode
          key={node.id}
          node={node}
          metadata={metadataForNodeType(node.type, copy)}
          selected={draft.selectedNodeId === node.id}
          targetable={Boolean(connectingSourceNodeID && connectingSourceNodeID !== node.id && node.type !== 'start')}
          connectingSourceNodeID={connectingSourceNodeID}
          onSelectNode={onSelectNode}
          onOpenContextMenu={onOpenContextMenu}
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
