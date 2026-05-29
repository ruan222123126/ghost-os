'use client';

import { useVirtualizer } from '@tanstack/react-virtual';
import { type DragEvent, type FC, type RefObject, useMemo } from 'react';
import {
  buildFlatSessionRows,
  buildPartitionSessionRows,
} from '@/components/SessionSidebarFlatList';
import {
  SessionSidebarHistoryRowView,
  type DropTargetState,
  type SessionSidebarHistoryRow,
} from '@/components/SessionSidebarHistoryPartitionSection';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarHistoryBodyProps {
  copy: ChatCopy;
  scrollElementRef: RefObject<HTMLDivElement>;
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

const SESSION_ROW_OVERSCAN = 10;

export const SessionSidebarHistoryBody: FC<SessionSidebarHistoryBodyProps> = ({
  copy,
  scrollElementRef,
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
          <div key={`session-skeleton-${index}`} className="h-10 w-full animate-pulse rounded bg-neutral-100" />
        ))}
      </div>
    );
  }

  if (empty) {
    return <div className="border border-black/10 bg-white px-3 py-3 text-xs text-neutral-600">{copy.sidebarNoSessions}</div>;
  }

  if (!groupingEnabled) {
    return (
      <SessionSidebarVirtualRows
        copy={copy}
        rows={buildFlatSessionRows(flatSessions)}
        scrollElementRef={scrollElementRef}
        currentSessionId={currentSessionId}
        resolveSessionTitle={resolveSessionTitle}
        onSelect={onSelect}
        onDelete={onDelete}
        onDragStartSession={onDragStartSession}
        onDragOverSession={onDragOverSession}
        onDropPartition={onDropPartition}
        onDragOverPartition={onDragOverPartition}
        onDragEndSession={onDragEndSession}
      />
    );
  }

  const rows = buildPartitionSessionRows({
    partitionViews,
    draggingSessionID: dragState.draggingSessionID,
    dropTarget: dragState.dropTarget,
  });

  return (
    <SessionSidebarVirtualRows
      copy={copy}
      rows={rows}
      scrollElementRef={scrollElementRef}
      currentSessionId={currentSessionId}
      resolveSessionTitle={resolveSessionTitle}
      onSelect={onSelect}
      onDelete={onDelete}
      onDragStartSession={onDragStartSession}
      onDragOverSession={onDragOverSession}
      onDragOverPartition={onDragOverPartition}
      onDropPartition={onDropPartition}
      onDragEndSession={onDragEndSession}
    />
  );
};

const SessionSidebarVirtualRows: FC<{
  copy: ChatCopy;
  rows: SessionSidebarHistoryRow[];
  scrollElementRef: RefObject<HTMLDivElement>;
  currentSessionId: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onDragStartSession: (event: DragEvent<HTMLDivElement>, sessionID: string) => void;
  onDragOverSession: (event: DragEvent<HTMLDivElement>, partitionID: string, itemIndex: number) => void;
  onDragOverPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDropPartition: (event: DragEvent<HTMLDivElement>, partitionID: string, itemCount: number) => void;
  onDragEndSession: () => void;
}> = (props) => {
  const rowVirtualizer = useVirtualizer({
    count: props.rows.length,
    estimateSize: (index) => estimateRowSize(props.rows[index]),
    getItemKey: (index) => props.rows[index]?.key ?? index,
    getScrollElement: () => props.scrollElementRef.current,
    overscan: SESSION_ROW_OVERSCAN,
  });
  const virtualItems = rowVirtualizer.getVirtualItems();

  const partitionCounts = useMemo(() => {
    return buildPartitionRowCounts(props.rows);
  }, [props.rows]);

  return (
    <div className="relative pb-3" style={{ height: rowVirtualizer.getTotalSize() }}>
      {virtualItems.map((virtualItem) => {
        const row = props.rows[virtualItem.index];
        if (!row) {
          return null;
        }
        return (
          <div
            key={virtualItem.key}
            data-index={virtualItem.index}
            ref={rowVirtualizer.measureElement}
            className="absolute left-0 top-0 w-full"
            style={{ transform: `translateY(${virtualItem.start}px)` }}
            onDragOver={dragOverHandler(row, props.onDragOverPartition, partitionCounts)}
            onDrop={dropHandler(row, props.onDropPartition, partitionCounts)}
          >
            <SessionSidebarHistoryRowView
              copy={props.copy}
              row={row}
              currentSessionId={props.currentSessionId}
              resolveSessionTitle={props.resolveSessionTitle}
              onSelect={props.onSelect}
              onDelete={props.onDelete}
              onDragStartSession={props.onDragStartSession}
              onDragOverSession={props.onDragOverSession}
              onDragEndSession={props.onDragEndSession}
            />
          </div>
        );
      })}
    </div>
  );
};

function estimateRowSize(row: SessionSidebarHistoryRow | undefined): number {
  if (!row) {
    return 44;
  }
  if (row.kind === 'partition-header') {
    return row.level === 0 ? 26 : 24;
  }
  if (row.kind === 'partition-empty-hint') {
    return 30;
  }
  if (row.kind === 'drop-line') {
    return 8;
  }
  return 44;
}

function buildPartitionRowCounts(rows: SessionSidebarHistoryRow[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const row of rows) {
    if (row.kind === 'session' && row.partitionID) {
      counts[row.partitionID] = (counts[row.partitionID] ?? 0) + 1;
    }
  }
  return counts;
}

function dragOverHandler(
  row: SessionSidebarHistoryRow,
  onDragOverPartition: SessionSidebarHistoryBodyProps['onDragOverPartition'],
  counts: Record<string, number>,
) {
  if (row.kind !== 'partition-empty-hint' && row.kind !== 'drop-line') {
    return undefined;
  }
  return (event: DragEvent<HTMLDivElement>) => {
    onDragOverPartition(event, row.partitionID, counts[row.partitionID] ?? 0);
  };
}

function dropHandler(
  row: SessionSidebarHistoryRow,
  onDropPartition: SessionSidebarHistoryBodyProps['onDropPartition'],
  counts: Record<string, number>,
) {
  if (row.kind !== 'partition-empty-hint' && row.kind !== 'drop-line') {
    return undefined;
  }
  return (event: DragEvent<HTMLDivElement>) => {
    onDropPartition(event, row.partitionID, counts[row.partitionID] ?? 0);
  };
}
