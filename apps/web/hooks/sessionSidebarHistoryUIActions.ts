import { type DragEvent, type MouseEvent } from 'react';
import { type ContextMenuState } from '@/components/SessionSidebarHistoryParts';
import { type DropTargetState } from '@/components/SessionSidebarHistoryPartitionSection';

const SESSION_DRAG_MIME = 'application/x-ghost-session-id';

type PartitionNameError = 'empty' | 'duplicate';

interface PartitionCreationOptions {
  addPartition: (name: string) => { ok: boolean; error?: PartitionNameError; createdName?: string };
  createErrorText: (error?: PartitionNameError) => string;
  createSuccessText: (name: string) => string;
  input: string;
  setCreateDialogOpen: (value: boolean) => void;
  setCreateError: (value: string) => void;
  setPartitionNameInput: (value: string) => void;
  setStatusMessage: (value: string) => void;
}

export function openPartitionContextMenu(
  event: MouseEvent<HTMLElement>,
  setContextMenu: (value: ContextMenuState) => void,
): void {
  const target = event.target as HTMLElement;
  if (isSessionItemTarget(target)) {
    return;
  }

  event.preventDefault();
  setContextMenu({
    x: event.clientX,
    y: event.clientY,
    ...resolveContextPartition(target),
  });
}

export function createPartitionFromInput(options: PartitionCreationOptions): void {
  const result = options.addPartition(options.input);
  if (!result.ok) {
    options.setCreateError(options.createErrorText(result.error));
    return;
  }

  options.setPartitionNameInput('');
  options.setCreateError('');
  options.setCreateDialogOpen(false);
  options.setStatusMessage(options.createSuccessText(result.createdName ?? ''));
}

export function openPartitionCreateDialog(
  setContextMenu: (value: ContextMenuState | undefined) => void,
  setCreateError: (value: string) => void,
  setCreateDialogOpen: (value: boolean) => void,
): void {
  setContextMenu(undefined);
  setCreateError('');
  setCreateDialogOpen(true);
}

export function closePartitionCreateDialog(
  setCreateDialogOpen: (value: boolean) => void,
  setCreateError: (value: string) => void,
): void {
  setCreateDialogOpen(false);
  setCreateError('');
}

export function startSessionDrag(options: {
  event: DragEvent<HTMLDivElement>;
  sessionID: string;
  setDraggingSessionID: (value: string) => void;
  setDropTarget: (value: DropTargetState | undefined) => void;
}): void {
  options.event.dataTransfer.effectAllowed = 'move';
  options.event.dataTransfer.setData(SESSION_DRAG_MIME, options.sessionID);
  options.setDraggingSessionID(options.sessionID);
  options.setDropTarget(undefined);
}

export function dragOverSession(options: {
  draggingSessionID: string;
  event: DragEvent<HTMLDivElement>;
  itemIndex: number;
  partitionID: string;
  setDropTarget: (value: DropTargetState) => void;
}): void {
  if (!options.draggingSessionID) {
    return;
  }

  options.event.preventDefault();
  options.event.stopPropagation();
  options.setDropTarget(resolveSessionDropTarget(options.event, options.partitionID, options.itemIndex));
}

export function dragOverPartition(options: {
  draggingSessionID: string;
  dropTarget?: DropTargetState;
  event: DragEvent<HTMLDivElement>;
  itemCount: number;
  partitionID: string;
  setDropTarget: (value: DropTargetState) => void;
}): void {
  if (!options.draggingSessionID) {
    return;
  }

  options.event.preventDefault();
  const nextTarget = resolvePartitionDropTarget(options.partitionID, options.itemCount);
  if (!isSameDropTarget(options.dropTarget, nextTarget)) {
    options.setDropTarget(nextTarget);
  }
}

export function dropSessionOnPartition(options: {
  draggingSessionID: string;
  dropTarget?: DropTargetState;
  event: DragEvent<HTMLDivElement>;
  itemCount: number;
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
  partitionID: string;
  setDraggingSessionID: (value: string) => void;
  setDropTarget: (value: DropTargetState | undefined) => void;
}): void {
  if (!options.draggingSessionID) {
    return;
  }

  options.event.preventDefault();
  options.moveSession(
    resolveDroppedSessionID(options.event, options.draggingSessionID),
    options.partitionID,
    resolveDropIndex(options.dropTarget, options.partitionID, options.itemCount),
  );
  resetSessionDrag(options.setDraggingSessionID, options.setDropTarget);
}

export function resetSessionDrag(
  setDraggingSessionID: (value: string) => void,
  setDropTarget: (value: DropTargetState | undefined) => void,
): void {
  setDraggingSessionID('');
  setDropTarget(undefined);
}

function isSessionItemTarget(target: HTMLElement): boolean {
  return Boolean(target.closest('[data-session-item="true"]'));
}

function resolveContextPartition(target: HTMLElement): Pick<ContextMenuState, 'partitionID' | 'partitionName'> {
  const partition = target.closest<HTMLElement>('[data-partition-item="true"]');

  return {
    partitionID: partition?.dataset.partitionId?.trim() ?? '',
    partitionName: partition?.dataset.partitionName?.trim() ?? '',
  };
}

function resolveSessionDropTarget(
  event: DragEvent<HTMLDivElement>,
  partitionID: string,
  itemIndex: number,
): DropTargetState {
  const bounds = event.currentTarget.getBoundingClientRect();
  const insertIndex = event.clientY < bounds.top + bounds.height / 2 ? itemIndex : itemIndex + 1;

  return { partitionID, insertIndex };
}

function resolvePartitionDropTarget(partitionID: string, itemCount: number): DropTargetState {
  return {
    partitionID,
    insertIndex: itemCount === 0 ? 0 : itemCount,
  };
}

function isSameDropTarget(previous: DropTargetState | undefined, next: DropTargetState): boolean {
  if (!previous) {
    return false;
  }

  return previous.partitionID === next.partitionID
    && previous.insertIndex === next.insertIndex;
}

function resolveDroppedSessionID(event: DragEvent<HTMLDivElement>, draggingSessionID: string): string {
  return event.dataTransfer.getData(SESSION_DRAG_MIME).trim() || draggingSessionID;
}

function resolveDropIndex(
  dropTarget: DropTargetState | undefined,
  partitionID: string,
  itemCount: number,
): number {
  return dropTarget?.partitionID === partitionID ? dropTarget.insertIndex : itemCount;
}
