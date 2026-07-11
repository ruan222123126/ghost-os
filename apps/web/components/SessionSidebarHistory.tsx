'use client';
import { type FC, type MouseEvent, useCallback, useMemo, useRef, useState } from 'react';
import { SessionSidebarHistoryBody } from '@/components/SessionSidebarHistoryBody';
import {
  PartitionCreateDialog,
  SidebarHistoryContextMenu,
} from '@/components/SessionSidebarHistoryParts';
import {
  findContextSessionID,
  resolveContextMenuStyle,
  resolvePartitionNameErrorText,
  useCloseOnEscape,
} from '@/components/sessionSidebarHistoryMenuUtils';
import {
  SessionContextMenu,
  SessionRenameDialog,
  type SessionContextMenuState,
} from '@/components/SessionSidebarHistorySessionRename';
import type { UseSessionSidebarAliasesResult } from '@/hooks/useSessionSidebarAliases';
import { useSessionSidebarHistoryUI } from '@/hooks/useSessionSidebarHistoryUI';
import type { UseSessionSidebarPartitionsResult } from '@/hooks/useSessionSidebarPartitions';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  UNCLASSIFIED_PARTITION_ID,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitions';
import { isSystemSessionPartitionID } from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarHistoryProps {
  isOpen: boolean;
  sessions: SessionMetadata[];
  searchQuery: string;
  currentSessionId: string;
  backgroundCompletedSessionIds: ReadonlySet<string>;
  focusSessionId?: string;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  resolveSessionTitle: UseSessionSidebarAliasesResult['resolveSessionTitle'];
  renameSession: UseSessionSidebarAliasesResult['renameSession'];
  partitionModel: UseSessionSidebarPartitionsResult;
  visiblePartitionViews: SessionPartitionView[];
  sessionSourcesError: string;
}
const EMPTY_RENAME_DIALOG_STATE = { sessionID: '', value: '', error: '' };
export const SessionSidebarHistory: FC<SessionSidebarHistoryProps> = (props) => {
  const { copy } = useWebLocale();
  const { renameSession, resolveSessionTitle } = props;
  const scrollElementRef = useRef<HTMLDivElement | null>(null);
  const showBlockingLoading = props.loading && props.sessions.length === 0;
  const [sessionContextMenu, setSessionContextMenu] = useState<SessionContextMenuState>();
  const [renameDialog, setRenameDialog] = useState(EMPTY_RENAME_DIALOG_STATE);
  const [collapsedPartitionIDs, setCollapsedPartitionIDs] = useState<ReadonlySet<string>>(() => new Set());
  const {
    partitionError,
    legacyMigrationAvailable,
    legacyMigrationRunning,
    addPartition,
    renamePartition,
    deletePartition,
    moveSession,
    runLegacyMigration,
    discardLegacyMigration,
  } = props.partitionModel;
  const ui = useSessionSidebarHistoryUI({
    isOpen: props.isOpen,
    addPartition,
    moveSession,
    createSuccessText: copy.chat.sidebarPartitionCreateSuccess,
    createErrorText: (error) => resolvePartitionNameErrorText(copy.chat, error),
  });
  const renameDialogOpen = renameDialog.sessionID.length > 0;
  const sessionMenuStyle = useMemo(() => {
    return resolveContextMenuStyle(sessionContextMenu);
  }, [sessionContextMenu]);
  const closeRenameDialog = useCallback(() => {
    setRenameDialog(EMPTY_RENAME_DIALOG_STATE);
  }, []);

  const closeSessionContextMenu = useCallback(() => {
    setSessionContextMenu(undefined);
  }, []);
  const onHistoryContextMenu = useCallback((event: MouseEvent<HTMLElement>) => {
    const sessionID = findContextSessionID(event.target);
    if (!sessionID) {
      closeSessionContextMenu();
      ui.onOpenContextMenu(event);
      return;
    }
    event.preventDefault();
    ui.closeContextMenu();
    setSessionContextMenu({
      x: event.clientX,
      y: event.clientY,
      sessionID,
    });
  }, [closeSessionContextMenu, ui]);
  const onOpenRenameDialog = useCallback(() => {
    if (!sessionContextMenu?.sessionID) {
      return;
    }
    const currentSession = props.sessions.find((session) => session.id === sessionContextMenu.sessionID);
    const currentTitle = currentSession ? resolveSessionTitle(currentSession) : '';
    setRenameDialog({
      sessionID: sessionContextMenu.sessionID,
      value: currentTitle,
      error: '',
    });
    closeSessionContextMenu();
  }, [closeSessionContextMenu, props.sessions, resolveSessionTitle, sessionContextMenu]);
  const onConfirmRename = useCallback(() => {
    if (!renameDialog.sessionID) {
      return;
    }
    const result = renameSession(renameDialog.sessionID, renameDialog.value);
    if (!result.ok) {
      setRenameDialog((previous) => ({
        ...previous,
        error: result.error === 'empty'
          ? copy.chat.sidebarSessionRenameRequired
          : copy.system.genericRequestFailed,
      }));
      return;
    }
    closeRenameDialog();
  }, [closeRenameDialog, copy.chat.sidebarSessionRenameRequired, copy.system.genericRequestFailed, renameDialog, renameSession]);
  const onChangeRenameValue = useCallback((value: string) => {
    setRenameDialog((previous) => ({
      ...previous,
      value,
      error: '',
    }));
  }, []);
  const onTogglePartitionCollapsed = useCallback((partitionID: string) => {
    setCollapsedPartitionIDs((previous) => {
      const next = new Set(previous);
      if (next.has(partitionID)) {
        next.delete(partitionID);
        return next;
      }
      next.add(partitionID);
      return next;
    });
  }, []);
  const onRenamePartition = useCallback(() => {
    const partitionID = ui.contextMenu?.partitionID?.trim() ?? '';
    if (!partitionID || partitionID === UNCLASSIFIED_PARTITION_ID || isSystemSessionPartitionID(partitionID)) {
      ui.closeContextMenu();
      return;
    }

    const currentName = ui.contextMenu?.partitionName ?? '';
    ui.closeContextMenu();
    const promptValue = window.prompt(copy.chat.sidebarPartitionRenamePrompt(currentName), currentName);
    if (promptValue === null) {
      return;
    }

    const result = renamePartition(partitionID, promptValue);
    if (!result.ok) {
      const errorText = (result.error === 'empty' || result.error === 'duplicate')
        ? resolvePartitionNameErrorText(copy.chat, result.error)
        : copy.system.genericRequestFailed;
      window.alert(errorText);
      return;
    }

  }, [copy.chat, copy.system.genericRequestFailed, renamePartition, ui]);
  const onDeletePartition = useCallback(() => {
    const partitionID = ui.contextMenu?.partitionID?.trim() ?? '';
    if (!partitionID || partitionID === UNCLASSIFIED_PARTITION_ID || isSystemSessionPartitionID(partitionID)) {
      ui.closeContextMenu();
      return;
    }

    const partitionName = ui.contextMenu?.partitionName || partitionID;
    ui.closeContextMenu();
    const confirmed = window.confirm(copy.chat.sidebarPartitionDeleteConfirm(partitionName));
    if (!confirmed) {
      return;
    }

    const result = deletePartition(partitionID);
    if (!result.ok) {
      window.alert(copy.system.genericRequestFailed);
      return;
    }

  }, [copy.chat, copy.system.genericRequestFailed, deletePartition, ui]);
  const contextPartitionID = ui.contextMenu?.partitionID ?? '';
  const canManageContextPartition = Boolean(contextPartitionID)
    && contextPartitionID !== UNCLASSIFIED_PARTITION_ID
    && !isSystemSessionPartitionID(contextPartitionID);
  useCloseOnEscape(
    Boolean(ui.contextMenu) || ui.createDialogOpen || Boolean(sessionContextMenu) || renameDialogOpen,
    () => {
      ui.closeContextMenu();
      ui.closeCreateDialog();
      closeSessionContextMenu();
      closeRenameDialog();
    },
  );
  if (!props.isOpen) {
    return null;
  }
  return (
    <div
      ref={scrollElementRef}
      className="session-sidebar-history-scroll mt-6 flex-1 overflow-y-auto px-3"
      onContextMenu={onHistoryContextMenu}
    >
      <div className="mb-4 flex items-center gap-2 border-b border-black/5 px-1 pb-1" onContextMenu={onHistoryContextMenu}>
        <span className="text-[10px] font-black uppercase tracking-[0.2em]">{copy.chat.sidebarHistory}</span>
      </div>
      {ui.statusMessage ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{ui.statusMessage}</div> : null}
      {legacyMigrationAvailable ? (
        <div className="mb-3 border border-black/10 bg-white px-3 py-3 text-xs text-neutral-700">
          <p className="font-semibold text-black">{copy.chat.sidebarPartitionLegacyMigrationTitle}</p>
          <p className="mt-1">{copy.chat.sidebarPartitionLegacyMigrationDescription}</p>
          <div className="mt-3 flex gap-2">
            <button
              type="button"
              className="bg-black px-3 py-1.5 text-xs font-semibold text-white hover:bg-neutral-800 disabled:cursor-not-allowed disabled:bg-neutral-400"
              disabled={legacyMigrationRunning}
              onClick={() => { void runLegacyMigration(); }}
            >
              {legacyMigrationRunning
                ? copy.chat.sidebarPartitionLegacyMigrationRunning
                : copy.chat.sidebarPartitionLegacyMigrationAction}
            </button>
            <button
              type="button"
              className="border border-black/10 px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-100"
              onClick={() => {
                if (!window.confirm(copy.chat.sidebarPartitionLegacyDiscardConfirm)) {
                  return;
                }
                discardLegacyMigration();
              }}
            >
              {copy.chat.sidebarPartitionLegacyDiscard}
            </button>
          </div>
        </div>
      ) : null}
      {partitionError ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{partitionError}</div> : null}
      {props.sessionSourcesError ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{props.sessionSourcesError}</div> : null}
      {props.error && !showBlockingLoading ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{props.error}</div> : null}
      <SessionSidebarHistoryBody
        copy={copy.chat}
        scrollElementRef={scrollElementRef}
        resetKey={`${props.searchQuery}:${partitionCollapseResetKey(collapsedPartitionIDs)}`}
        loading={showBlockingLoading}
        empty={props.visiblePartitionViews.length === 0}
        partitionViews={props.visiblePartitionViews}
        collapsedPartitionIDs={collapsedPartitionIDs}
        currentSessionId={props.currentSessionId}
        backgroundCompletedSessionIds={props.backgroundCompletedSessionIds}
        focusSessionId={props.focusSessionId}
        dragState={ui.dragState}
        resolveSessionTitle={resolveSessionTitle}
        onSelect={props.onSelect}
        onDelete={props.onDelete}
        onTogglePartitionCollapsed={onTogglePartitionCollapsed}
        onDragStartSession={ui.onDragStartSession}
        onDragOverSession={ui.onDragOverSession}
        onDragOverPartition={ui.onDragOverPartition}
        onDropPartition={ui.onDropPartition}
        onDragEndSession={ui.onDragEndSession}
      />
      <SidebarHistoryContextMenu
        menu={ui.contextMenu}
        menuStyle={ui.menuStyle}
        createLabel={copy.chat.sidebarPartitionAdd}
        renameLabel={copy.chat.sidebarPartitionRename}
        deleteLabel={copy.chat.sidebarPartitionDelete}
        showPartitionActions={canManageContextPartition}
        onClose={ui.closeContextMenu}
        onCreate={() => {
          closeSessionContextMenu();
          ui.openCreateDialog();
        }}
        onRename={onRenamePartition}
        onDelete={onDeletePartition}
      />
      <SessionContextMenu
        menu={sessionContextMenu}
        menuStyle={sessionMenuStyle}
        renameLabel={copy.chat.sidebarSessionRename}
        onClose={closeSessionContextMenu}
        onRename={onOpenRenameDialog}
      />
      <PartitionCreateDialog
        copy={copy.chat}
        open={ui.createDialogOpen}
        value={ui.partitionNameInput}
        error={ui.createError}
        onChange={ui.onChangePartitionName}
        onClose={ui.closeCreateDialog}
        onCreate={ui.onCreatePartition}
      />
      <SessionRenameDialog
        copy={copy.chat}
        open={renameDialogOpen}
        value={renameDialog.value}
        error={renameDialog.error}
        onChange={onChangeRenameValue}
        onClose={closeRenameDialog}
        onConfirm={onConfirmRename}
      />
    </div>
  );
};

function partitionCollapseResetKey(collapsedPartitionIDs: ReadonlySet<string>): string {
  return [...collapsedPartitionIDs].sort().join(',');
}
