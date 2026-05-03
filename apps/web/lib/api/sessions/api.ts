import type {
  SessionDetail,
  SessionMetadata,
  SessionSidebarPartitionState,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { createClientTraceId } from '@/lib/api/trace';
import {
  parseSessionDetail,
  parseSessionMetadataList,
  parseSessionSidebarPartitionState,
} from '@/lib/api/sessions/parser';

export interface GetSessionOptions {
  before?: number;
  limit?: number;
}

export async function listSessions(): Promise<SessionMetadata[]> {
  return requestJSON('/api/sessions', {}, parseSessionMetadataList);
}

export async function getSession(id: string, options: GetSessionOptions = {}): Promise<SessionDetail> {
  return requestJSON(buildSessionPath(id, options), {}, parseSessionDetail);
}

export async function deleteSession(id: string): Promise<void> {
  await requestJSON<Record<string, unknown>>(`/api/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export async function getSessionSidebarPartitions(): Promise<SessionSidebarPartitionState> {
  return requestJSON('/api/sessions/partitions', {}, parseSessionSidebarPartitionState);
}

export async function putSessionSidebarPartitions(
  state: SessionSidebarPartitionState,
  traceId = createClientTraceId('session-partitions'),
): Promise<SessionSidebarPartitionState> {
  return requestJSON('/api/sessions/partitions', {
    method: 'PUT',
    body: JSON.stringify({
      ...state,
      trace_id: traceId,
    }),
  }, parseSessionSidebarPartitionState);
}

function buildSessionPath(id: string, options: GetSessionOptions): string {
  const query = new URLSearchParams();
  if (options.limit !== undefined) {
    query.set('limit', String(options.limit));
  }
  if (options.before !== undefined) {
    query.set('before', String(options.before));
  }

  const suffix = query.size > 0 ? `?${query.toString()}` : '';
  return `/api/sessions/${encodeURIComponent(id)}${suffix}`;
}
