import type { AgentStreamEvent } from '@/lib/types';

export function resolveEventSessionId(event: AgentStreamEvent): string {
  if (event.session_id?.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case 'run_started':
    case 'message':
    case 'done':
    case 'error':
      return readPayloadSessionId(event.payload);
    default:
      return '';
  }
}

function readPayloadSessionId(payload: unknown): string {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return '';
  }

  const sessionId = (payload as Record<string, unknown>).session_id;
  return typeof sessionId === 'string' ? sessionId.trim() : '';
}
