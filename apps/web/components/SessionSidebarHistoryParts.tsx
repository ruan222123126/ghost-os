'use client';

import { Fragment, type CSSProperties, type DragEvent, type FC, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionMetadata } from '@/lib/types';

export interface ContextMenuState {
  x: number;
  y: number;
  partitionID: string;
  partitionName: string;
}

export interface DropTargetState {
  partitionID: string;
  insertIndex: number;
}

interface PartitionSectionProps {
  copy: ChatCopy;
  partitionID: string;
  partitionName: string;
  sessions: SessionMetadata[];
  resolveSessionTitle: (session: SessionMetadata) => string;
  currentSessionId: string;
  draggingSessionID: string;
  dropTarget?: DropTargetState;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDropPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragOverPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragEndSession: () => void;
}

export const PartitionSection: FC<PartitionSectionProps> = ({
  copy,
  partitionID,
  partitionName,
  sessions,
  resolveSessionTitle,
  currentSessionId,
  draggingSessionID,
  dropTarget,
  onSelect,
  onDelete,
  onDropPartition,
  onDragOverPartition,
  onDragOverSession,
  onDragStartSession,
  onDragEndSession,
}) => {
  const showTailDrop = dropTarget?.partitionID === partitionID && dropTarget.insertIndex === sessions.length;

  return (
    <section data-partition-item="true" data-partition-id={partitionID} data-partition-name={partitionName}>
      <div className="mb-2 px-1 text-[10px] font-bold uppercase tracking-[0.16em] text-neutral-500">{partitionName}</div>
      <div
        className={`space-y-0.5 rounded-sm ${draggingSessionID ? 'border border-dashed border-black/25 p-1' : ''}`}
        onDragOver={(event) => onDragOverPartition(event, partitionID, sessions.length)}
        onDrop={(event) => onDropPartition(event, partitionID, sessions.length)}
      >
        {sessions.length === 0 ? <div className="px-2 py-2 text-[11px] text-neutral-400">{copy.sidebarPartitionDropHint}</div> : null}
        {sessions.map((session, index) => (
          <Fragment key={session.id}>
            {dropTarget?.partitionID === partitionID && dropTarget.insertIndex === index ? <DropLine /> : null}
            <SessionRow
              copy={copy}
              session={session}
              resolveSessionTitle={resolveSessionTitle}
              isActive={session.id === currentSessionId}
              onSelect={onSelect}
              onDelete={onDelete}
              onDragStart={onDragStartSession}
              onDragOver={onDragOverSession}
              onDragEnd={onDragEndSession}
              partitionID={partitionID}
              itemIndex={index}
            />
          </Fragment>
        ))}
        {showTailDrop ? <DropLine /> : null}
      </div>
    </section>
  );
};

interface SessionRowProps {
  copy: ChatCopy;
  session: SessionMetadata;
  resolveSessionTitle: (session: SessionMetadata) => string;
  isActive: boolean;
  partitionID: string;
  itemIndex: number;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDragStart: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOver: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragEnd: () => void;
}

const SessionRow: FC<SessionRowProps> = ({
  copy,
  session,
  resolveSessionTitle,
  isActive,
  partitionID,
  itemIndex,
  onSelect,
  onDelete,
  onDragStart,
  onDragOver,
  onDragEnd,
}) => {
  const shortID = session.id.slice(0, 8);
  const sessionTitle = resolveSessionTitle(session);
  const onRowKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    onSelect(session.id);
  };

  return (
    <div
      data-session-item="true"
      data-session-id={session.id}
      draggable
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={onRowKeyDown}
      onContextMenu={(event) => {
        event.preventDefault();
      }}
      onDragStart={(event) => onDragStart(event, session.id)}
      onDragOver={(event) => onDragOver(event, partitionID, itemIndex)}
      onDragEnd={onDragEnd}
      className={`group relative flex w-full items-center gap-3 px-3 py-3 text-left text-xs transition-colors ${
        isActive
          ? 'bg-white font-bold text-black shadow-sm ring-1 ring-black/5'
          : 'text-neutral-500 hover:bg-white hover:text-black'
      }`}
    >
      <span className="flex-1 truncate">{sessionTitle}</span>
      <button
        type="button"
        onClick={(event) => {
          event.stopPropagation();
          onDelete(session.id);
        }}
        className="opacity-0 transition-opacity group-hover:opacity-100"
        aria-label={copy.sidebarDeleteSessionAria(shortID)}
        title={copy.sessionDelete}
      >
        <span className="text-[10px] font-black uppercase tracking-widest text-black">{copy.sidebarDeleteShort}</span>
      </button>
    </div>
  );
};

const DropLine: FC = () => <div className="mx-2 h-0.5 bg-black/55" />;

interface SidebarHistoryContextMenuProps {
  menu?: ContextMenuState;
  menuStyle: CSSProperties;
  createLabel: string;
  renameLabel: string;
  deleteLabel: string;
  showPartitionActions: boolean;
  onClose: () => void;
  onCreate: () => void;
  onRename: () => void;
  onDelete: () => void;
}

export const SidebarHistoryContextMenu: FC<SidebarHistoryContextMenuProps> = ({
  menu,
  menuStyle,
  createLabel,
  renameLabel,
  deleteLabel,
  showPartitionActions,
  onClose,
  onCreate,
  onRename,
  onDelete,
}) => {
  if (!menu) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 z-40" onMouseDown={onClose} />
      <div
        className="fixed z-50 w-40 border border-black/15 bg-white p-1 shadow-lg"
        role="menu"
        style={menuStyle}
        onMouseDown={(event) => event.stopPropagation()}
        onContextMenu={(event) => event.preventDefault()}
      >
        <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold hover:bg-neutral-100" onClick={onCreate}>
          {createLabel}
        </button>
        {showPartitionActions ? (
          <>
            <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold hover:bg-neutral-100" onClick={onRename}>
              {renameLabel}
            </button>
            <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold text-red-700 hover:bg-red-50" onClick={onDelete}>
              {deleteLabel}
            </button>
          </>
        ) : null}
      </div>
    </>
  );
};

interface PartitionCreateDialogProps {
  copy: ChatCopy;
  open: boolean;
  value: string;
  error: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onCreate: () => void;
}

export const PartitionCreateDialog: FC<PartitionCreateDialogProps> = ({ copy, open, value, error, onChange, onClose, onCreate }) => {
  if (!open) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/35" onMouseDown={onClose} />
      <div className="fixed left-1/2 top-1/2 z-50 w-[320px] -translate-x-1/2 -translate-y-1/2 border border-black/10 bg-white p-4 shadow-xl">
        <p className="mb-2 text-sm font-bold text-black">{copy.sidebarPartitionCreateTitle}</p>
        <input
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={copy.sidebarPartitionNamePlaceholder}
          className="w-full border border-black/20 px-3 py-2 text-xs outline-none focus:border-black"
          aria-label={copy.sidebarPartitionNameLabel}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              onCreate();
            }
          }}
        />
        {error ? <p className="mt-2 text-xs text-red-700">{error}</p> : null}
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" className="px-3 py-1.5 text-xs font-semibold text-neutral-600 hover:bg-neutral-100" onClick={onClose}>{copy.sidebarPartitionCancel}</button>
          <button type="button" className="bg-black px-3 py-1.5 text-xs font-semibold text-white hover:bg-neutral-800" onClick={onCreate}>{copy.sidebarPartitionCreateConfirm}</button>
        </div>
      </div>
    </>
  );
};
