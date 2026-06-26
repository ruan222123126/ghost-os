'use client';

import { type DragEvent, type MouseEvent, useEffect, useState } from 'react';
import { type ContextMenuState } from '@/components/SessionSidebarHistoryParts';
import { type DropTargetState } from '@/components/SessionSidebarHistoryPartitionSection';
import { resolveContextMenuStyle } from '@/components/sessionSidebarHistoryMenuUtils';
import { UNCLASSIFIED_PARTITION_ID } from '@/lib/sessionSidebarPartitions';
import { isSystemSessionPartitionID } from '@/lib/sessionSidebarSessionSources';
import {
  closePartitionCreateDialog,
  createPartitionFromInput,
  dragOverPartition,
  dragOverSession,
  dropSessionOnPartition,
  openPartitionContextMenu,
  openPartitionCreateDialog,
  resetSessionDrag,
  startSessionDrag,
} from './sessionSidebarHistoryUIActions';

const STATUS_TIMEOUT_MS = 1800;
const PARTITION_MENU_ACTION_HEIGHT = 116;
const PARTITION_MENU_CREATE_ONLY_HEIGHT = 48;

interface UseSessionSidebarHistoryUIOptions {
  isOpen: boolean;
  addPartition: (name: string) => { ok: boolean; error?: 'empty' | 'duplicate'; createdName?: string };
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
  createSuccessText: (name: string) => string;
  createErrorText: (error?: 'empty' | 'duplicate') => string;
}

interface UseSessionSidebarHistoryUIResult {
  contextMenu?: ContextMenuState;
  menuStyle: ReturnType<typeof resolveContextMenuStyle>;
  createDialogOpen: boolean;
  partitionNameInput: string;
  createError: string;
  statusMessage: string;
  dragState: {
    draggingSessionID: string;
    dropTarget?: DropTargetState;
  };
  onOpenContextMenu: (event: MouseEvent<HTMLElement>) => void;
  onCreatePartition: () => void;
  onChangePartitionName: (value: string) => void;
  closeContextMenu: () => void;
  openCreateDialog: () => void;
  closeCreateDialog: () => void;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragOverPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDropPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragEndSession: () => void;
}

interface PartitionMenuState {
  contextMenu?: ContextMenuState;
  createDialogOpen: boolean;
  partitionNameInput: string;
  createError: string;
  statusMessage: string;
  onOpenContextMenu: (event: MouseEvent<HTMLElement>) => void;
  onCreatePartition: () => void;
  onChangePartitionName: (value: string) => void;
  closeContextMenu: () => void;
  openCreateDialog: () => void;
  closeCreateDialog: () => void;
}

interface DragState {
  draggingSessionID: string;
  dropTarget?: DropTargetState;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragOverPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDropPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragEndSession: () => void;
}

interface ResetMenuWhenClosedOptions {
  isOpen: boolean;
  setContextMenu: (value: ContextMenuState | undefined) => void;
  setCreateDialogOpen: (value: boolean) => void;
  setCreateError: (value: string) => void;
}

export function useSessionSidebarHistoryUI(
  options: UseSessionSidebarHistoryUIOptions,
): UseSessionSidebarHistoryUIResult {
  const partitionMenu = usePartitionMenuState(options);
  const drag = useSessionDragState(options.moveSession);

  return {
    contextMenu: partitionMenu.contextMenu,
    menuStyle: resolveContextMenuStyle(partitionMenu.contextMenu, resolvePartitionMenuHeight(partitionMenu.contextMenu)),
    createDialogOpen: partitionMenu.createDialogOpen,
    partitionNameInput: partitionMenu.partitionNameInput,
    createError: partitionMenu.createError,
    statusMessage: partitionMenu.statusMessage,
    dragState: { draggingSessionID: drag.draggingSessionID, dropTarget: drag.dropTarget },
    onOpenContextMenu: partitionMenu.onOpenContextMenu,
    onCreatePartition: partitionMenu.onCreatePartition,
    onChangePartitionName: partitionMenu.onChangePartitionName,
    closeContextMenu: partitionMenu.closeContextMenu,
    openCreateDialog: partitionMenu.openCreateDialog,
    closeCreateDialog: partitionMenu.closeCreateDialog,
    onDragStartSession: drag.onDragStartSession,
    onDragOverSession: drag.onDragOverSession,
    onDragOverPartition: drag.onDragOverPartition,
    onDropPartition: drag.onDropPartition,
    onDragEndSession: drag.onDragEndSession,
  };
}

function usePartitionMenuState(options: UseSessionSidebarHistoryUIOptions): PartitionMenuState {
  const [contextMenu, setContextMenu] = useState<ContextMenuState>();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [partitionNameInput, setPartitionNameInput] = useState('');
  const [createError, setCreateError] = useState('');
  const [statusMessage, setStatusMessage] = useState('');

  useAutoClearStatus(statusMessage, setStatusMessage);
  useResetMenuWhenClosed({
    isOpen: options.isOpen,
    setContextMenu,
    setCreateDialogOpen,
    setCreateError,
  });

  const onOpenContextMenu = (event: MouseEvent<HTMLElement>) => {
    openPartitionContextMenu(event, setContextMenu);
  };

  const onCreatePartition = () => {
    createPartitionFromInput({
      addPartition: options.addPartition,
      createErrorText: options.createErrorText,
      createSuccessText: options.createSuccessText,
      input: partitionNameInput,
      setCreateDialogOpen,
      setCreateError,
      setPartitionNameInput,
      setStatusMessage,
    });
  };

  return {
    contextMenu,
    createDialogOpen,
    partitionNameInput,
    createError,
    statusMessage,
    onOpenContextMenu,
    onCreatePartition,
    onChangePartitionName: setPartitionNameInput,
    closeContextMenu: () => setContextMenu(undefined),
    openCreateDialog: () => {
      openPartitionCreateDialog(setContextMenu, setCreateError, setCreateDialogOpen);
    },
    closeCreateDialog: () => {
      closePartitionCreateDialog(setCreateDialogOpen, setCreateError);
    },
  };
}

function resolvePartitionMenuHeight(menu?: ContextMenuState): number {
  if (!menu || !canManagePartition(menu.partitionID)) {
    return PARTITION_MENU_CREATE_ONLY_HEIGHT;
  }
  return PARTITION_MENU_ACTION_HEIGHT;
}

function canManagePartition(partitionID: string): boolean {
  return Boolean(partitionID)
    && partitionID !== UNCLASSIFIED_PARTITION_ID
    && !isSystemSessionPartitionID(partitionID);
}

function useAutoClearStatus(
  statusMessage: string,
  setStatusMessage: (value: string) => void,
) {
  useEffect(() => {
    if (!statusMessage) {
      return;
    }
    const timer = setTimeout(() => setStatusMessage(''), STATUS_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [statusMessage, setStatusMessage]);
}

function useResetMenuWhenClosed(options: ResetMenuWhenClosedOptions) {
  const { isOpen, setContextMenu, setCreateDialogOpen, setCreateError } = options;

  useEffect(() => {
    if (isOpen) {
      return;
    }
    setContextMenu(undefined);
    setCreateDialogOpen(false);
    setCreateError('');
  }, [isOpen, setContextMenu, setCreateDialogOpen, setCreateError]);
}

function useSessionDragState(
  moveSession: UseSessionSidebarHistoryUIOptions['moveSession'],
): DragState {
  const [draggingSessionID, setDraggingSessionID] = useState('');
  const [dropTarget, setDropTarget] = useState<DropTargetState>();

  const onDragStartSession = (event: DragEvent<HTMLDivElement>, sessionID: string) => {
    startSessionDrag({ event, sessionID, setDraggingSessionID, setDropTarget });
  };

  const onDragOverSession = (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => {
    dragOverSession({ draggingSessionID, event, itemIndex, partitionID, setDropTarget });
  };

  const onDragOverPartition = (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => {
    dragOverPartition({ draggingSessionID, dropTarget, event, itemCount, partitionID, setDropTarget });
  };

  const onDropPartition = (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => {
    dropSessionOnPartition({
      draggingSessionID,
      dropTarget,
      event,
      itemCount,
      moveSession,
      partitionID,
      setDraggingSessionID,
      setDropTarget,
    });
  };

  return {
    draggingSessionID,
    dropTarget,
    onDragStartSession,
    onDragOverSession,
    onDragOverPartition,
    onDropPartition,
    onDragEndSession: () => resetSessionDrag(setDraggingSessionID, setDropTarget),
  };
}
