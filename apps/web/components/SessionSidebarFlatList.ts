import {
  type SessionSidebarHistoryRow,
} from '@/components/SessionSidebarHistoryPartitionSection';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { SessionMetadata } from '@/lib/types';
import type { DropTargetState } from './SessionSidebarHistoryPartitionSection';

export function buildFlatSessionList(
  sessions: SessionMetadata[],
  searchQuery: string,
): SessionMetadata[] {
  const query = searchQuery.trim().toLowerCase();
  return sessions
    .filter((session) => session.id.toLowerCase().includes(query))
    .sort(compareSessionsByRecentActivity);
}

export function buildFlatSessionRows(
  sessions: SessionMetadata[],
): SessionSidebarHistoryRow[] {
  return sessions.map((session) => ({
    kind: 'session',
    key: `flat:${session.id}`,
    session,
  }));
}

export function buildPartitionSessionRows(input: {
  partitionViews: SessionPartitionView[];
  draggingSessionID: string;
  dropTarget?: DropTargetState;
}): SessionSidebarHistoryRow[] {
  return input.partitionViews.flatMap((partition) => {
    return rowsForPartition(partition, {
      draggingSessionID: input.draggingSessionID,
      dropTarget: input.dropTarget,
      level: 0,
    });
  });
}

function rowsForPartition(
  partition: SessionPartitionView,
  options: {
    draggingSessionID: string;
    dropTarget?: DropTargetState;
    level: number;
  },
): SessionSidebarHistoryRow[] {
  const rows: SessionSidebarHistoryRow[] = [{
    kind: 'partition-header',
    key: `partition:${partition.id}`,
    partitionID: partition.id,
    partitionName: partition.name,
    level: options.level,
  }];

  if (partition.childPartitions?.length) {
    return rows.concat(partition.childPartitions.flatMap((child) => {
      return rowsForPartition(child, { ...options, level: options.level + 1 });
    }));
  }

  return rows.concat(rowsForPartitionSessions(partition, options));
}

function rowsForPartitionSessions(
  partition: SessionPartitionView,
  options: {
    draggingSessionID: string;
    dropTarget?: DropTargetState;
  },
): SessionSidebarHistoryRow[] {
  if (partition.sessions.length === 0 && !partition.readOnly) {
    return [{
      kind: 'partition-empty-hint',
      key: `empty:${partition.id}`,
      partitionID: partition.id,
    }];
  }

  const rows = partition.sessions.flatMap((session, index) => {
    const sessionRows: SessionSidebarHistoryRow[] = [];
    if (shouldShowDropLine(partition, options.dropTarget, index)) {
      sessionRows.push(dropLineRow(partition.id, index));
    }
    sessionRows.push({
      kind: 'session',
      key: `partition:${partition.id}:session:${session.id}`,
      session,
      partitionID: partition.id,
      itemIndex: index,
      readOnly: partition.readOnly,
    });
    return sessionRows;
  });

  if (shouldShowDropLine(partition, options.dropTarget, partition.sessions.length)) {
    rows.push(dropLineRow(partition.id, partition.sessions.length));
  }
  return rows;
}

function shouldShowDropLine(
  partition: SessionPartitionView,
  dropTarget: DropTargetState | undefined,
  insertIndex: number,
): boolean {
  return !partition.readOnly
    && dropTarget?.partitionID === partition.id
    && dropTarget.insertIndex === insertIndex;
}

function dropLineRow(partitionID: string, insertIndex: number): SessionSidebarHistoryRow {
  return {
    kind: 'drop-line',
    key: `drop:${partitionID}:${insertIndex}`,
    partitionID,
  };
}
