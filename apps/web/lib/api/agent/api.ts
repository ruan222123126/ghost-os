import type {
  AgentStopRequest,
  AgentStopResponsePayload,
  ApiRequest,
  ExternalAgentResponse,
  ExternalAgentStopParams,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseAgentStopResponse, parseExternalAgentResponse } from '@/lib/api/agent/parser';
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

export async function stopExternalAgent(
  sessionId: string,
): Promise<ExternalAgentResponse> {
  const params = buildExternalAgentStopParams(sessionId);
  const body: ApiRequest<ExternalAgentStopParams> = {
    action: 'EXTERNAL_AGENT_STOP',
    params,
    trace_id: createClientTraceId('external-agent-stop'),
  };

  return requestJSON('/api/bus', {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseExternalAgentResponse);
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

function buildExternalAgentStopParams(sessionId: string): ExternalAgentStopParams {
  const normalizedSessionId = normalizeOptionalId(sessionId);
  if (!normalizedSessionId) {
    throw new Error('session_id is required');
  }
  return { session_id: normalizedSessionId };
}

function normalizeOptionalId(value?: string): string {
  return value?.trim() ?? '';
}
