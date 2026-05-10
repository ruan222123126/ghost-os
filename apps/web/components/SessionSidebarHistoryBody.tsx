'use client';

import { type DragEvent, type FC } from 'react';
import { SessionSidebarFlatList } from '@/components/SessionSidebarFlatList';
import {
  PartitionSection,
  type DropTargetState,
} from '@/components/SessionSidebarHistoryPartitionSection';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarHistoryBodyProps {
  copy: ChatCopy;
  loading: boolean;
  groupingEnabled: boolean;
  empty: boolean;
  flatSessions: SessionMetadata[];
  partitionViews: SessionPartitionView[];
  currentSessionId: string;
  dragState: {
    draggingSessionID: string;
    dropTarget?: DropTargetState;
  };
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragOverPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDropPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragEndSession: () => void;
}

export const SessionSidebarHistoryBody: FC<SessionSidebarHistoryBodyProps> = ({
  copy,
  loading,
  groupingEnabled,
  empty,
  flatSessions,
  partitionViews,
  currentSessionId,
  dragState,
  resolveSessionTitle,
  onSelect,
  onDelete,
  onDragStartSession,
  onDragOverSession,
  onDragOverPartition,
  onDropPartition,
  onDragEndSession,
}) => {
  if (loading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 6 }).map((_, index) => (
          <div key={`session-skeleton-${index}`} className="h-10 w-full bg-white" />
        ))}
      </div>
    );
  }

  if (empty) {
    return <div className="border border-black/10 bg-white px-3 py-3 text-xs text-neutral-600">{copy.sidebarNoSessions}</div>;
  }

  if (!groupingEnabled) {
    return (
      <SessionSidebarFlatList
        copy={copy}
        sessions={flatSessions}
        currentSessionId={currentSessionId}
        resolveSessionTitle={resolveSessionTitle}
        onSelect={onSelect}
        onDelete={onDelete}
      />
    );
  }

  return (
    <div className="space-y-4 pb-3">
      {partitionViews.map((partition) => (
        <PartitionSection
          key={partition.id}
          copy={copy}
          partitionID={partition.id}
          partitionName={partition.name}
          readOnly={partition.readOnly}
          sessions={partition.sessions}
          childPartitions={partition.childPartitions}
          resolveSessionTitle={resolveSessionTitle}
          currentSessionId={currentSessionId}
          draggingSessionID={dragState.draggingSessionID}
          dropTarget={dragState.dropTarget}
          onSelect={onSelect}
          onDelete={onDelete}
          onDropPartition={onDropPartition}
          onDragOverPartition={onDragOverPartition}
          onDragOverSession={onDragOverSession}
          onDragStartSession={onDragStartSession}
          onDragEndSession={onDragEndSession}
        />
      ))}
    </div>
  );
};
