import type {
  SessionDetail,
  SessionMetadata,
  SessionMessage,
  SessionSourceResolution,
  SessionSidebarPartitionState,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { createClientTraceId } from '@/lib/api/trace';
import {
  parseSessionDetail,
  parseSessionMetadataList,
  parseSessionSourceResolution,
  parseSessionSidebarPartitionState,
} from '@/lib/api/sessions/parser';

export interface GetSessionOptions {
  before?: number;
  limit?: number;
}

export async function listSessions(): Promise<SessionMetadata[]> {
  return requestJSON('/api/sessions', {}, parseSessionMetadataList);
}

export async function searchSessions(query: string): Promise<SessionMetadata[]> {
  const params = new URLSearchParams();
  const normalizedQuery = query.trim();
  if (normalizedQuery) {
    params.set('q', normalizedQuery);
  }
  const suffix = params.size > 0 ? `?${params.toString()}` : '';
  return requestJSON(`/api/sessions/search${suffix}`, {}, parseSessionMetadataList);
}

export async function getSessionSources(): Promise<SessionSourceResolution> {
  return requestJSON('/api/sessions/sources', {}, parseSessionSourceResolution);
}

export async function getSession(id: string, options: GetSessionOptions = {}): Promise<SessionDetail> {
  return requestJSON(buildSessionPath(id, options), {}, parseSessionDetail);
}

export async function getFullSession(id: string, pageLimit = 100): Promise<SessionDetail> {
  const latest = await getSession(id, { limit: pageLimit });
  if (!latest.page.has_more_before || latest.page.next_before === null) {
    return latest;
  }

  const messages = await collectFullSessionMessages(id, latest, pageLimit);
  return {
    ...latest,
    messages,
    page: {
      ...latest.page,
      has_more_before: false,
      next_before: null,
      start_index: messages[0]?.index ?? latest.page.start_index,
      end_index: messages.at(-1)?.index ?? latest.page.end_index,
    },
  };
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

async function collectFullSessionMessages(
  id: string,
  latest: SessionDetail,
  pageLimit: number,
): Promise<SessionMessage[]> {
  let messages = [...latest.messages];
  let hasMoreBefore = latest.page.has_more_before;
  let nextBefore = latest.page.next_before ?? null;

  while (hasMoreBefore && nextBefore !== null) {
    const page = await getSession(id, { before: nextBefore, limit: pageLimit });
    messages = [...page.messages, ...messages];
    hasMoreBefore = page.page.has_more_before;
    nextBefore = page.page.next_before ?? null;
  }

  return messages;
}
