import type { SessionDetail, SessionMetadata } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import {
  parseSessionDetail,
  parseSessionMetadataList,
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
