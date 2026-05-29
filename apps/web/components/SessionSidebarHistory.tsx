'use client';
import { type FC, type MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { buildFlatSessionList } from '@/components/SessionSidebarFlatList';
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
import { useSessionSidebarGroupingPreference } from '@/hooks/useSessionSidebarGroupingPreference';
import { useSessionSidebarHistoryUI } from '@/hooks/useSessionSidebarHistoryUI';
import { useSessionSidebarPartitions } from '@/hooks/useSessionSidebarPartitions';
import { useSessionSidebarSessionSources } from '@/hooks/useSessionSidebarSessionSources';
import { useWebLocale } from '@/lib/i18n/provider';
import { UNCLASSIFIED_PARTITION_ID } from '@/lib/sessionSidebarPartitions';
import {
  isSystemSessionPartitionID,
  mergeSessionSourcePartitionViews,
} from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarHistoryProps {
  isOpen: boolean;
  sessions: SessionMetadata[];
  searchQuery: string;
  currentSessionId: string;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  resolveSessionTitle: UseSessionSidebarAliasesResult['resolveSessionTitle'];
  renameSession: UseSessionSidebarAliasesResult['renameSession'];
}
const EMPTY_RENAME_DIALOG_STATE = { sessionID: '', value: '', error: '' };
export const SessionSidebarHistory: FC<SessionSidebarHistoryProps> = (props) => {
  const { copy } = useWebLocale();
  const { renameSession, resolveSessionTitle } = props;
  const { enabled: groupingEnabled } = useSessionSidebarGroupingPreference();
  const scrollElementRef = useRef<HTMLDivElement | null>(null);
  const showBlockingLoading = props.loading && props.sessions.length === 0;
  const [sessionContextMenu, setSessionContextMenu] = useState<SessionContextMenuState>();
  const [renameDialog, setRenameDialog] = useState(EMPTY_RENAME_DIALOG_STATE);
  const {
    partitionViews,
    partitionError,
    addPartition,
    renamePartition,
    deletePartition,
    moveSession,
  } = useSessionSidebarPartitions({
    sessions: props.sessions,
    sessionsLoaded: !showBlockingLoading,
    searchQuery: props.searchQuery,
    unclassifiedName: copy.chat.sidebarPartitionUnclassified,
    requestFailedText: copy.system.genericRequestFailed,
  });
  const sessionSources = useSessionSidebarSessionSources({
    enabled: groupingEnabled,
    sessions: props.sessions,
    requestFailedText: copy.system.genericRequestFailed,
  });
  const visiblePartitionViews = useMemo(() => {
    return mergeSessionSourcePartitionViews({
      manualViews: partitionViews,
      sessions: props.sessions,
      sourceAssignments: sessionSources.assignments,
      hiddenSessionIDs: sessionSources.hiddenSessionIDs,
      searchQuery: props.searchQuery,
      copy: copy.chat,
    });
  }, [copy.chat, partitionViews, props.searchQuery, props.sessions, sessionSources.assignments, sessionSources.hiddenSessionIDs]);
  const ui = useSessionSidebarHistoryUI({
    isOpen: props.isOpen,
    addPartition,
    moveSession,
    createSuccessText: copy.chat.sidebarPartitionCreateSuccess,
    createErrorText: (error) => resolvePartitionNameErrorText(copy.chat, error),
  });
  const renameDialogOpen = renameDialog.sessionID.length > 0;
  const flatSessions = useMemo(() => {
    return buildFlatSessionList(props.sessions, props.searchQuery);
  }, [props.searchQuery, props.sessions]);
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
      if (groupingEnabled) {
        ui.onOpenContextMenu(event);
      }
      return;
    }
    event.preventDefault();
    ui.closeContextMenu();
    setSessionContextMenu({
      x: event.clientX,
      y: event.clientY,
      sessionID,
    });
  }, [closeSessionContextMenu, groupingEnabled, ui]);
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
  useEffect(() => {
    if (groupingEnabled) {
      return;
    }
    ui.closeContextMenu();
    ui.closeCreateDialog();
  }, [groupingEnabled, ui]);
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
    <div ref={scrollElementRef} className="mt-6 flex-1 overflow-y-auto px-3" onContextMenu={onHistoryContextMenu}>
      <div className="mb-4 flex items-center gap-2 border-b border-black/5 px-1 pb-1" onContextMenu={onHistoryContextMenu}>
        <span className="text-[10px] font-black uppercase tracking-[0.2em]">{copy.chat.sidebarHistory}</span>
      </div>
      {ui.statusMessage ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{ui.statusMessage}</div> : null}
      {partitionError ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{partitionError}</div> : null}
      {groupingEnabled && sessionSources.error ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{sessionSources.error}</div> : null}
      {props.error && !showBlockingLoading ? <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{props.error}</div> : null}
      <SessionSidebarHistoryBody
        copy={copy.chat}
        scrollElementRef={scrollElementRef}
        loading={showBlockingLoading}
        groupingEnabled={groupingEnabled}
        empty={groupingEnabled ? visiblePartitionViews.length === 0 : flatSessions.length === 0}
        flatSessions={flatSessions}
        partitionViews={visiblePartitionViews}
        currentSessionId={props.currentSessionId}
        dragState={ui.dragState}
        resolveSessionTitle={resolveSessionTitle}
        onSelect={props.onSelect}
        onDelete={props.onDelete}
        onDragStartSession={ui.onDragStartSession}
        onDragOverSession={ui.onDragOverSession}
        onDragOverPartition={ui.onDragOverPartition}
        onDropPartition={ui.onDropPartition}
        onDragEndSession={ui.onDragEndSession}
      />
      {groupingEnabled ? (
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
      ) : null}
      <SessionContextMenu
        menu={sessionContextMenu}
        menuStyle={sessionMenuStyle}
        renameLabel={copy.chat.sidebarSessionRename}
        onClose={closeSessionContextMenu}
        onRename={onOpenRenameDialog}
      />
      {groupingEnabled ? (
        <PartitionCreateDialog
          copy={copy.chat}
          open={ui.createDialogOpen}
          value={ui.partitionNameInput}
          error={ui.createError}
          onChange={ui.onChangePartitionName}
          onClose={ui.closeCreateDialog}
          onCreate={ui.onCreatePartition}
        />
      ) : null}
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
