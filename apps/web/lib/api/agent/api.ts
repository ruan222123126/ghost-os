import type {
  AgentStopRequest,
  AgentStopResponsePayload,
  ApiRequest,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseAgentStopResponse } from '@/lib/api/agent/parser';
import { createClientTraceId } from '@/lib/api/trace';

export async function stopAgent(
  sessionId?: string,
  traceId?: string,
): Promise<AgentStopResponsePayload> {
  const params = buildAgentStopParams(sessionId, traceId);
  const body: ApiRequest<AgentStopRequest> = {
    action: 'AGENT_STOP',
    params,
    trace_id: createClientTraceId('agent-stop'),
  };

  return requestJSON('/api/bus', {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseAgentStopResponse);
}

function buildAgentStopParams(
  sessionId?: string,
  traceId?: string,
): AgentStopRequest {
  const params: AgentStopRequest = {};
  const normalizedSessionId = normalizeOptionalId(sessionId);
  const normalizedTraceId = normalizeOptionalId(traceId);
  if (normalizedSessionId) {
    params.session_id = normalizedSessionId;
  }
  if (normalizedTraceId) {
    params.trace_id = normalizedTraceId;
  }
  if (!params.session_id && !params.trace_id) {
    throw new Error('session_id or trace_id is required');
  }
  return params;
}

function normalizeOptionalId(value?: string): string {
  return value?.trim() ?? '';
}
