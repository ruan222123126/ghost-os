import type { SessionMetadata } from '@/lib/types';

export function compareSessionsByRecentActivity(
  left: SessionMetadata,
  right: SessionMetadata,
): number {
  const rightTime = resolveRecentActivityTimestamp(right);
  const leftTime = resolveRecentActivityTimestamp(left);
  if (rightTime !== leftTime) {
    return rightTime - leftTime;
  }
  return right.id.localeCompare(left.id);
}

export function resolveRecentActivityTimestamp(session: SessionMetadata): number {
  return parseTimestamp(session.updated_at)
    ?? parseTimestamp(session.created_at)
    ?? 0;
}

export function sortSessionIDsByRecentActivity(
  sessionIDs: string[],
  sessionByID: Map<string, SessionMetadata>,
): string[] {
  return [...sessionIDs].sort((leftID, rightID) => {
    const left = sessionByID.get(leftID);
    const right = sessionByID.get(rightID);
    if (!left && !right) {
      return rightID.localeCompare(leftID);
    }
    if (!left) {
      return 1;
    }
    if (!right) {
      return -1;
    }
    return compareSessionsByRecentActivity(left, right);
  });
}

function parseTimestamp(value: string | undefined): number | undefined {
  if (!value) {
    return undefined;
  }
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? undefined : parsed;
}
