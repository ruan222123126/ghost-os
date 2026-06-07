import {
  type SessionSidebarHistoryRow,
} from '@/components/SessionSidebarHistoryPartitionSection';
import type { SessionPartitionView, SessionSearchMatcher } from '@/lib/sessionSidebarPartitions';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { SessionMetadata } from '@/lib/types';
import type { DropTargetState } from './SessionSidebarHistoryPartitionSection';

export function buildFlatSessionList(
  sessions: SessionMetadata[],
  searchQuery: string,
  matchesSearch: SessionSearchMatcher = defaultSessionSearchMatcher,
): SessionMetadata[] {
  const query = searchQuery.trim().toLowerCase();
  return sessions
    .filter((session) => matchesSearch(session, query))
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

export function countSessionsInPartitionViews(
  partitionViews: SessionPartitionView[],
): number {
  return partitionViews.reduce((count, partition) => {
    return count + countSessionsInPartition(partition);
  }, 0);
}

export function limitPartitionViewsBySessionCount(
  partitionViews: SessionPartitionView[],
  sessionLimit: number,
): SessionPartitionView[] {
  let remaining = Math.max(sessionLimit, 0);
  const limited: SessionPartitionView[] = [];

  for (const partition of partitionViews) {
    if (remaining <= 0) {
      break;
    }

    const next = limitPartitionViewBySessionCount(partition, remaining);
    if (!next.partition) {
      continue;
    }

    limited.push(next.partition);
    remaining = next.remaining;
  }

  return limited;
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

function countSessionsInPartition(partition: SessionPartitionView): number {
  if (partition.childPartitions?.length) {
    return partition.childPartitions.reduce((count, child) => {
      return count + countSessionsInPartition(child);
    }, 0);
  }
  return partition.sessions.length;
}

function limitPartitionViewBySessionCount(
  partition: SessionPartitionView,
  remaining: number,
): { partition: SessionPartitionView | null; remaining: number } {
  if (remaining <= 0) {
    return { partition: null, remaining: 0 };
  }

  if (partition.childPartitions?.length) {
    const childPartitions: SessionPartitionView[] = [];
    let nextRemaining = remaining;

    for (const child of partition.childPartitions) {
      if (nextRemaining <= 0) {
        break;
      }

      const nextChild = limitPartitionViewBySessionCount(child, nextRemaining);
      if (!nextChild.partition) {
        continue;
      }

      childPartitions.push(nextChild.partition);
      nextRemaining = nextChild.remaining;
    }

    if (childPartitions.length === 0) {
      return { partition: null, remaining };
    }

    return {
      partition: {
        ...partition,
        childPartitions,
      },
      remaining: nextRemaining,
    };
  }

  const sessions = partition.sessions.slice(0, remaining);
  if (sessions.length === 0) {
    return { partition: null, remaining };
  }

  return {
    partition: {
      ...partition,
      sessions,
    },
    remaining: remaining - sessions.length,
  };
}

function defaultSessionSearchMatcher(
  session: SessionMetadata,
  normalizedQuery: string,
): boolean {
  return !normalizedQuery || session.id.toLowerCase().includes(normalizedQuery);
}
