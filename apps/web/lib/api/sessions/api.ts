import type { SessionDetail, SessionMetadata } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import {
  parseSessionDetail,
  parseSessionMetadataList,
} from '@/lib/api/sessions/parser';

export async function listSessions(): Promise<SessionMetadata[]> {
  return requestJSON('/api/sessions', {}, parseSessionMetadataList);
}

export async function getSession(id: string): Promise<SessionDetail> {
  return requestJSON(`/api/sessions/${encodeURIComponent(id)}`, {}, parseSessionDetail);
}

export async function deleteSession(id: string): Promise<void> {
  await requestJSON<Record<string, unknown>>(`/api/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}
