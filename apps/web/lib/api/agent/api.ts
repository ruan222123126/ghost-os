import type {
  AgentRequest,
  AgentSendResponse,
  AgentStopRequest,
  AgentStopResponsePayload,
  ApiRequest,
  HumanResponseRequest,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseAgentSendResponse, parseAgentStopResponse } from '@/lib/api/agent/parser';
import { createClientTraceId } from '@/lib/api/trace';

export async function sendMessage(
  message: string,
  images?: AgentRequest['images'],
  sessionId?: string,
  traceId?: string,
): Promise<AgentSendResponse> {
  const body: AgentRequest = { message };
  if (images?.length) {
    body.images = images.map((image) => ({ ...image }));
  }
  if (sessionId?.trim()) {
    body.session_id = sessionId.trim();
  }
  if (traceId?.trim()) {
    body.trace_id = traceId.trim();
  }

  return requestJSON('/api/agent', {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseAgentSendResponse);
}

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

export async function sendHumanResponse(
  sessionId: string,
  questionId: string,
  answer: string,
  cancelled = false,
): Promise<AgentSendResponse> {
  const body: HumanResponseRequest = {
    session_id: sessionId.trim(),
    question_id: questionId.trim(),
    answer: answer.trim(),
  };
  if (cancelled) {
    body.cancelled = true;
  }

  return requestJSON('/api/questions/answer', {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseAgentSendResponse);
}

function buildAgentStopParams(
  sessionId?: string,
  traceId?: string,
): AgentStopRequest {
  const normalizedSessionId = sessionId?.trim() ?? '';
  const normalizedTraceId = traceId?.trim() ?? '';
  if (!normalizedSessionId && !normalizedTraceId) {
    throw new Error('session_id or trace_id is required');
  }

  return {
    ...(normalizedSessionId ? { session_id: normalizedSessionId } : {}),
    ...(normalizedTraceId ? { trace_id: normalizedTraceId } : {}),
  };
}
