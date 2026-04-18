'use client';

import { type DragEvent, type MouseEvent, useEffect, useState } from 'react';
import {
  type ContextMenuState,
  type DropTargetState,
} from '@/components/SessionSidebarHistoryParts';
import { resolveContextMenuStyle } from '@/components/sessionSidebarHistoryMenuUtils';
import { UNCLASSIFIED_PARTITION_ID } from '@/lib/sessionSidebarPartitions';

const SESSION_DRAG_MIME = 'application/x-ghost-session-id';
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
  useResetMenuWhenClosed(options.isOpen, setContextMenu, setCreateDialogOpen, setCreateError);

  const onOpenContextMenu = (event: MouseEvent<HTMLElement>) => {
    const target = event.target as HTMLElement;
    if (target.closest('[data-session-item="true"]')) {
      return;
    }
    event.preventDefault();
    const partitionID = target.closest<HTMLElement>('[data-partition-item="true"]')?.dataset.partitionId?.trim() ?? '';
    const partitionName = target.closest<HTMLElement>('[data-partition-item="true"]')?.dataset.partitionName?.trim() ?? '';
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      partitionID,
      partitionName,
    });
  };

  const onCreatePartition = () => {
    const result = options.addPartition(partitionNameInput);
    if (!result.ok) {
      setCreateError(options.createErrorText(result.error));
      return;
    }
    setPartitionNameInput('');
    setCreateError('');
    setCreateDialogOpen(false);
    setStatusMessage(options.createSuccessText(result.createdName ?? ''));
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
      setContextMenu(undefined);
      setCreateError('');
      setCreateDialogOpen(true);
    },
    closeCreateDialog: () => {
      setCreateDialogOpen(false);
      setCreateError('');
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
  return Boolean(partitionID) && partitionID !== UNCLASSIFIED_PARTITION_ID;
}

function useAutoClearStatus(
  statusMessage: string,
  setStatusMessage: (value: string) => void,
) {
  useEffect(() => {
    if (!statusMessage) {
      return;
    }
    const timer = window.setTimeout(() => setStatusMessage(''), STATUS_TIMEOUT_MS);
    return () => window.clearTimeout(timer);
  }, [statusMessage, setStatusMessage]);
}

function useResetMenuWhenClosed(
  isOpen: boolean,
  setContextMenu: (value: ContextMenuState | undefined) => void,
  setCreateDialogOpen: (value: boolean) => void,
  setCreateError: (value: string) => void,
) {
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
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData(SESSION_DRAG_MIME, sessionID);
    setDraggingSessionID(sessionID);
    setDropTarget(undefined);
  };

  const onDragOverSession = (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => {
    if (!draggingSessionID) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    const bounds = event.currentTarget.getBoundingClientRect();
    const insertIndex = event.clientY < bounds.top + bounds.height / 2 ? itemIndex : itemIndex + 1;
    setDropTarget({ partitionID, insertIndex });
  };

  const onDragOverPartition = (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => {
    if (!draggingSessionID) {
      return;
    }
    event.preventDefault();
    const insertIndex = itemCount === 0 ? 0 : itemCount;
    if (!dropTarget || dropTarget.partitionID !== partitionID || dropTarget.insertIndex !== insertIndex) {
      setDropTarget({ partitionID, insertIndex });
    }
  };

  const onDropPartition = (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => {
    if (!draggingSessionID) {
      return;
    }
    event.preventDefault();
    const fallback = event.dataTransfer.getData(SESSION_DRAG_MIME).trim();
    const sessionID = fallback || draggingSessionID;
    const index = dropTarget?.partitionID === partitionID ? dropTarget.insertIndex : itemCount;
    moveSession(sessionID, partitionID, index);
    setDraggingSessionID('');
    setDropTarget(undefined);
  };

  return {
    draggingSessionID,
    dropTarget,
    onDragStartSession,
    onDragOverSession,
    onDragOverPartition,
    onDropPartition,
    onDragEndSession: () => {
      setDraggingSessionID('');
      setDropTarget(undefined);
    },
  };
}
