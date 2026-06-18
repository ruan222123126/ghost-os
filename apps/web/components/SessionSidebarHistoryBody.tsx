'use client';

import { useVirtualizer } from '@tanstack/react-virtual';
import { type DragEvent, type FC, type RefObject, useEffect, useMemo } from 'react';
import {
  buildFlatSessionRows,
  buildPartitionSessionRows,
  countSessionsInPartitionViews,
  limitPartitionViewsBySessionCount,
} from '@/components/SessionSidebarFlatList';
import {
  SessionSidebarHistoryRowView,
  type DropTargetState,
  type SessionSidebarHistoryRow,
} from '@/components/SessionSidebarHistoryPartitionSection';
import { useSessionSidebarVisibleCount } from '@/hooks/useSessionSidebarVisibleCount';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarHistoryBodyProps {
  copy: ChatCopy;
  scrollElementRef: RefObject<HTMLDivElement>;
  resetKey: string;
  loading: boolean;
  groupingEnabled: boolean;
  empty: boolean;
  flatSessions: SessionMetadata[];
  partitionViews: SessionPartitionView[];
  collapsedPartitionIDs: ReadonlySet<string>;
  currentSessionId: string;
  focusSessionId?: string;
  dragState: {
    draggingSessionID: string;
    dropTarget?: DropTargetState;
  };
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onTogglePartitionCollapsed: (partitionID: string) => void;
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
  resetKey,
  loading,
  groupingEnabled,
  empty,
  flatSessions,
  partitionViews,
  collapsedPartitionIDs,
  currentSessionId,
  focusSessionId,
  dragState,
  resolveSessionTitle,
  onSelect,
  onDelete,
  onTogglePartitionCollapsed,
  onDragStartSession,
  onDragOverSession,
  onDragOverPartition,
  onDropPartition,
  onDragEndSession,
}) => {
  const totalSessions = useMemo(() => {
    if (!groupingEnabled) {
      return flatSessions.length;
    }
    return countSessionsInPartitionViews(partitionViews, collapsedPartitionIDs);
  }, [collapsedPartitionIDs, flatSessions.length, groupingEnabled, partitionViews]);
  const focusSessionIndex = useMemo(() => {
    const id = focusSessionId?.trim();
    if (!id) {
      return undefined;
    }
    if (!groupingEnabled) {
      return flatSessions.findIndex((session) => session.id === id);
    }
    return findSessionIndexInPartitionViews(partitionViews, id, collapsedPartitionIDs);
  }, [collapsedPartitionIDs, flatSessions, focusSessionId, groupingEnabled, partitionViews]);
  const visibleSessionCount = useSessionSidebarVisibleCount({
    scrollElementRef,
    totalSessions,
    resetKey,
    focusSessionIndex,
  });
  const visibleFlatSessions = useMemo(() => {
    return flatSessions.slice(0, visibleSessionCount);
  }, [flatSessions, visibleSessionCount]);
  const visiblePartitionViews = useMemo(() => {
    return limitPartitionViewsBySessionCount(partitionViews, visibleSessionCount, collapsedPartitionIDs);
  }, [collapsedPartitionIDs, partitionViews, visibleSessionCount]);

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
        rows={buildFlatSessionRows(visibleFlatSessions)}
        scrollElementRef={scrollElementRef}
        currentSessionId={currentSessionId}
        focusSessionId={focusSessionId}
        resolveSessionTitle={resolveSessionTitle}
        onSelect={onSelect}
        onDelete={onDelete}
        onTogglePartitionCollapsed={onTogglePartitionCollapsed}
        onDragStartSession={onDragStartSession}
        onDragOverSession={onDragOverSession}
        onDropPartition={onDropPartition}
        onDragOverPartition={onDragOverPartition}
        onDragEndSession={onDragEndSession}
      />
    );
  }

  const rows = buildPartitionSessionRows({
    partitionViews: visiblePartitionViews,
    draggingSessionID: dragState.draggingSessionID,
    dropTarget: dragState.dropTarget,
    collapsedPartitionIDs,
  });

  return (
    <SessionSidebarVirtualRows
      copy={copy}
      rows={rows}
      scrollElementRef={scrollElementRef}
      currentSessionId={currentSessionId}
      focusSessionId={focusSessionId}
      resolveSessionTitle={resolveSessionTitle}
      onSelect={onSelect}
      onDelete={onDelete}
      onTogglePartitionCollapsed={onTogglePartitionCollapsed}
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
  focusSessionId?: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onTogglePartitionCollapsed: (partitionID: string) => void;
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
  const focusedRowIndex = useMemo(() => {
    const id = props.focusSessionId?.trim();
    if (!id) {
      return -1;
    }
    return props.rows.findIndex((row) => row.kind === 'session' && row.session.id === id);
  }, [props.focusSessionId, props.rows]);

  const partitionCounts = useMemo(() => {
    return buildPartitionRowCounts(props.rows);
  }, [props.rows]);

  useEffect(() => {
    if (focusedRowIndex < 0) {
      return;
    }
    rowVirtualizer.scrollToIndex(focusedRowIndex, { align: 'center' });
  }, [focusedRowIndex, rowVirtualizer]);

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
              onTogglePartitionCollapsed={props.onTogglePartitionCollapsed}
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

function findSessionIndexInPartitionViews(
  partitionViews: SessionPartitionView[],
  sessionID: string,
  collapsedPartitionIDs: ReadonlySet<string>,
): number {
  let index = 0;
  for (const partition of partitionViews) {
    const found = findSessionIndexInPartition(partition, sessionID, index, collapsedPartitionIDs);
    if (found.found) {
      return found.index;
    }
    index = found.nextIndex;
  }
  return -1;
}

function findSessionIndexInPartition(
  partition: SessionPartitionView,
  sessionID: string,
  startIndex: number,
  collapsedPartitionIDs: ReadonlySet<string>,
): { found: boolean; index: number; nextIndex: number } {
  if (collapsedPartitionIDs.has(partition.id)) {
    return { found: false, index: -1, nextIndex: startIndex };
  }

  if (partition.childPartitions?.length) {
    let nextIndex = startIndex;
    for (const child of partition.childPartitions) {
      const found = findSessionIndexInPartition(child, sessionID, nextIndex, collapsedPartitionIDs);
      if (found.found) {
        return found;
      }
      nextIndex = found.nextIndex;
    }
    return { found: false, index: -1, nextIndex };
  }

  const localIndex = partition.sessions.findIndex((session) => session.id === sessionID);
  if (localIndex >= 0) {
    return { found: true, index: startIndex + localIndex, nextIndex: startIndex + partition.sessions.length };
  }
  return { found: false, index: -1, nextIndex: startIndex + partition.sessions.length };
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
