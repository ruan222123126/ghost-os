'use client';

import { type DragEvent, type FC, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { IconTrash } from '@/components/sessionSidebarIcons';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionMetadata } from '@/lib/types';

export interface DropTargetState {
  partitionID: string;
  insertIndex: number;
}

export interface SessionSidebarPartitionHeaderRow {
  kind: 'partition-header';
  key: string;
  partitionID: string;
  partitionName: string;
  level: number;
}

export interface SessionSidebarPartitionHintRow {
  kind: 'partition-empty-hint';
  key: string;
  partitionID: string;
}

export interface SessionSidebarDropLineRow {
  kind: 'drop-line';
  key: string;
  partitionID: string;
}

export interface SessionSidebarSessionRow {
  kind: 'session';
  key: string;
  session: SessionMetadata;
  partitionID?: string;
  itemIndex?: number;
  readOnly?: boolean;
}

export type SessionSidebarHistoryRow =
  | SessionSidebarPartitionHeaderRow
  | SessionSidebarPartitionHintRow
  | SessionSidebarDropLineRow
  | SessionSidebarSessionRow;

interface SessionRowProps {
  copy: ChatCopy;
  session: SessionMetadata;
  resolveSessionTitle: (session: SessionMetadata) => string;
  isActive: boolean;
  partitionID?: string;
  itemIndex?: number;
  readOnly?: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDragStart?: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOver?: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragEnd?: () => void;
}

interface SessionSidebarHistoryRowViewProps {
  copy: ChatCopy;
  row: SessionSidebarHistoryRow;
  currentSessionId: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragEndSession: () => void;
}

export const SessionSidebarHistoryRowView: FC<SessionSidebarHistoryRowViewProps> = ({
  copy,
  row,
  currentSessionId,
  resolveSessionTitle,
  onSelect,
  onDelete,
  onDragStartSession,
  onDragOverSession,
  onDragEndSession,
}) => {
  if (row.kind === 'partition-header') {
    return (
      <div
        data-partition-item="true"
        data-partition-id={row.partitionID}
        data-partition-name={row.partitionName}
        className={row.level === 0
          ? 'mb-2 px-1 pt-1 text-[10px] font-bold uppercase tracking-[0.16em] text-neutral-500'
          : 'mb-1 border-l border-black/10 px-2 pt-1 text-[10px] font-semibold text-neutral-500'}
      >
        {row.partitionName}
      </div>
    );
  }

  if (row.kind === 'partition-empty-hint') {
    return (
      <div
        data-partition-item="true"
        data-partition-id={row.partitionID}
        className="px-2 py-2 text-[11px] text-neutral-400"
      >
        {copy.sidebarPartitionDropHint}
      </div>
    );
  }

  if (row.kind === 'drop-line') {
    return <DropLine partitionID={row.partitionID} />;
  }

  return (
    <SessionRow
      copy={copy}
      session={row.session}
      resolveSessionTitle={resolveSessionTitle}
      isActive={row.session.id === currentSessionId}
      onSelect={onSelect}
      onDelete={onDelete}
      onDragStart={onDragStartSession}
      onDragOver={onDragOverSession}
      onDragEnd={onDragEndSession}
      partitionID={row.partitionID}
      itemIndex={row.itemIndex}
      readOnly={row.readOnly}
    />
  );
};

const SessionRow: FC<SessionRowProps> = ({
  copy,
  session,
  resolveSessionTitle,
  isActive,
  partitionID,
  itemIndex,
  readOnly,
  onSelect,
  onDelete,
  onDragStart,
  onDragOver,
  onDragEnd,
}) => {
  const shortID = session.id.slice(0, 8);
  const sessionTitle = resolveSessionTitle(session);
  const canDrag = !readOnly && partitionID !== undefined && itemIndex !== undefined;

  return (
    <div
      data-session-item="true"
      data-session-id={session.id}
      draggable={canDrag}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={(event) => onSessionRowKeyDown(event, session.id, onSelect)}
      onContextMenu={(event) => {
        event.preventDefault();
      }}
      onDragStart={canDrag ? (event) => onDragStart?.(event, session.id) : undefined}
      onDragOver={canDrag ? (event) => onDragOver?.(event, partitionID, itemIndex) : undefined}
      onDragEnd={canDrag ? onDragEnd : undefined}
      className={`group relative flex w-full items-center gap-3 px-3 py-3 text-left text-xs transition-colors ${rowClassName(isActive)}`}
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
        <span className="flex h-6 w-6 items-center justify-center text-black">
          <IconTrash />
        </span>
      </button>
    </div>
  );
};

function onSessionRowKeyDown(
  event: ReactKeyboardEvent<HTMLDivElement>,
  sessionID: string,
  onSelect: (id: string) => void,
): void {
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }
  event.preventDefault();
  onSelect(sessionID);
}

function rowClassName(isActive: boolean): string {
  if (isActive) {
    return 'bg-white font-bold text-black shadow-sm ring-1 ring-black/5';
  }
  return 'text-neutral-500 hover:bg-white hover:text-black';
}

const DropLine: FC<{ partitionID: string }> = ({ partitionID }) => (
  <div data-partition-id={partitionID} className="mx-2 h-0.5 bg-black/55" />
);
