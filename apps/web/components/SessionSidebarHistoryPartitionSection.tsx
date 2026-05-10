'use client';

import { Fragment, type DragEvent, type FC, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

export interface DropTargetState {
  partitionID: string;
  insertIndex: number;
}

interface PartitionSectionProps {
  copy: ChatCopy;
  partitionID: string;
  partitionName: string;
  readOnly?: boolean;
  sessions: SessionMetadata[];
  childPartitions?: SessionPartitionView[];
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
  childPartitions,
  ...props
}) => {
  const children = childPartitions ?? [];

  return (
    <section data-partition-item="true" data-partition-id={props.partitionID} data-partition-name={props.partitionName}>
      <div className="mb-2 px-1 text-[10px] font-bold uppercase tracking-[0.16em] text-neutral-500">{props.partitionName}</div>
      {children.length > 0 ? <ChildPartitionSections childPartitions={children} {...props} /> : <PartitionSessionList {...props} />}
    </section>
  );
};

interface ChildPartitionSectionsProps extends Omit<PartitionSectionProps, 'partitionID' | 'partitionName' | 'sessions' | 'childPartitions' | 'readOnly'> {
  childPartitions: SessionPartitionView[];
}

const ChildPartitionSections: FC<ChildPartitionSectionsProps> = ({
  childPartitions,
  ...props
}) => (
  <div className="space-y-3">
    {childPartitions.map((partition) => (
      <section
        key={partition.id}
        data-partition-item="true"
        data-partition-id={partition.id}
        data-partition-name={partition.name}
        className="border-l border-black/10 pl-2"
      >
        <div className="mb-1 px-1 text-[10px] font-semibold text-neutral-500">{partition.name}</div>
        <PartitionSessionList
          {...props}
          partitionID={partition.id}
          readOnly={partition.readOnly}
          sessions={partition.sessions}
        />
      </section>
    ))}
  </div>
);

type PartitionSessionListProps = Omit<PartitionSectionProps, 'partitionName' | 'childPartitions'>;

const PartitionSessionList: FC<PartitionSessionListProps> = ({
  copy,
  partitionID,
  readOnly,
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
  const showTailDrop = !readOnly && dropTarget?.partitionID === partitionID && dropTarget.insertIndex === sessions.length;

  return (
    <div
      className={`space-y-0.5 rounded-sm ${draggingSessionID && !readOnly ? 'border border-dashed border-black/25 p-1' : ''}`}
      onDragOver={readOnly ? undefined : (event) => onDragOverPartition(event, partitionID, sessions.length)}
      onDrop={readOnly ? undefined : (event) => onDropPartition(event, partitionID, sessions.length)}
    >
      {sessions.length === 0 && !readOnly ? <div className="px-2 py-2 text-[11px] text-neutral-400">{copy.sidebarPartitionDropHint}</div> : null}
      {sessions.map((session, index) => (
        <Fragment key={session.id}>
          {!readOnly && dropTarget?.partitionID === partitionID && dropTarget.insertIndex === index ? <DropLine /> : null}
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
            readOnly={readOnly}
          />
        </Fragment>
      ))}
      {showTailDrop ? <DropLine /> : null}
    </div>
  );
};

interface SessionRowProps {
  copy: ChatCopy;
  session: SessionMetadata;
  resolveSessionTitle: (session: SessionMetadata) => string;
  isActive: boolean;
  partitionID: string;
  itemIndex: number;
  readOnly?: boolean;
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
  readOnly,
  onSelect,
  onDelete,
  onDragStart,
  onDragOver,
  onDragEnd,
}) => {
  const shortID = session.id.slice(0, 8);
  const sessionTitle = resolveSessionTitle(session);

  return (
    <div
      data-session-item="true"
      data-session-id={session.id}
      draggable={!readOnly}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={(event) => onSessionRowKeyDown(event, session.id, onSelect)}
      onContextMenu={(event) => {
        event.preventDefault();
      }}
      onDragStart={readOnly ? undefined : (event) => onDragStart(event, session.id)}
      onDragOver={readOnly ? undefined : (event) => onDragOver(event, partitionID, itemIndex)}
      onDragEnd={readOnly ? undefined : onDragEnd}
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
        <span className="text-[10px] font-black uppercase tracking-widest text-black">{copy.sidebarDeleteShort}</span>
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

const DropLine: FC = () => <div className="mx-2 h-0.5 bg-black/55" />;
