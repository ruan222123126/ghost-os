'use client';

import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type MouseEvent as ReactMouseEvent,
  type SetStateAction,
} from 'react';
import type { WorkflowNodeContextMenuState } from '@/components/workflow/WorkflowCanvasNodeContextMenu';
import type { WorkflowCanvasDraft } from '@/lib/workflow-editor';

interface WorkflowNodeContextMenuOptions {
  draft: WorkflowCanvasDraft;
  onSelectNode: (nodeID?: string) => void;
  setConnectingSourceNodeID: Dispatch<SetStateAction<string | undefined>>;
}

interface WorkflowNodeContextMenuControls {
  clearContextMenu: () => void;
  closeContextMenu: () => void;
  nodeContextMenu?: WorkflowNodeContextMenuState;
  onOpenContextMenu: (event: ReactMouseEvent<HTMLElement>, nodeID: string) => void;
}

export function useWorkflowNodeContextMenu(
  options: WorkflowNodeContextMenuOptions,
): WorkflowNodeContextMenuControls {
  const { draft, onSelectNode, setConnectingSourceNodeID } = options;
  const [nodeContextMenu, setNodeContextMenu] = useState<WorkflowNodeContextMenuState>();

  useEffect(() => {
    if (!nodeContextMenu) {
      return;
    }
    const exists = draft.nodes.some((node) => node.id === nodeContextMenu.nodeID);
    if (!exists) {
      setNodeContextMenu(undefined);
    }
  }, [draft.nodes, nodeContextMenu]);

  const clearContextMenu = useCallback(() => {
    setNodeContextMenu(undefined);
  }, []);

  const onOpenContextMenu = useCallback((event: ReactMouseEvent<HTMLElement>, nodeID: string) => {
    event.preventDefault();
    event.stopPropagation();
    onSelectNode(nodeID);
    setConnectingSourceNodeID(undefined);
    setNodeContextMenu({
      nodeID,
      clientX: event.clientX,
      clientY: event.clientY,
    });
  }, [onSelectNode, setConnectingSourceNodeID]);

  return {
    clearContextMenu,
    closeContextMenu: clearContextMenu,
    nodeContextMenu,
    onOpenContextMenu,
  };
}
