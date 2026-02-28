import type {
  AgentSendResponse,
  ApiEnvelope,
  BridgeConfig,
  ConfigUpdate,
  HumanResponseRequest,
  SessionDetail,
  SessionMetadata,
} from '@/lib/types';

async function request<TPayload>(path: string, init: RequestInit = {}): Promise<TPayload> {
  const headers = {
    'Content-Type': 'application/json',
    ...(init.headers ?? {}),
  };

  const response = await fetch(path, {
    ...init,
    headers,
    cache: 'no-store',
  });

  const data = (await response.json()) as ApiEnvelope<TPayload>;
  if (!response.ok) {
    throw new Error(data.error || `Request failed with status ${response.status}`);
  }

  if (data.status === 'error') {
    throw new Error(data.error || `Request failed with status ${response.status}`);
  }

  return data.payload;
}

export async function sendMessage(message: string, sessionId?: string): Promise<AgentSendResponse> {
  const body: Record<string, string> = { message };
  if (sessionId && sessionId.trim()) {
    body.session_id = sessionId.trim();
  }

  return request<AgentSendResponse>('/api/agent', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export async function sendHumanResponse(sessionId: string, questionId: string, answer: string): Promise<void> {
  const body: HumanResponseRequest = {
    session_id: sessionId.trim(),
    question_id: questionId.trim(),
    answer: answer.trim(),
  };

  await request<{ accepted: boolean }>('/api/bus', {
    method: 'POST',
    body: JSON.stringify({
      action: 'HUMAN_RESPONSE',
      params: body,
      trace_id: `web-${Date.now()}`,
    }),
  });
}

export async function getConfig(): Promise<BridgeConfig> {
  return request<BridgeConfig>('/api/config');
}

export async function updateConfig(update: ConfigUpdate): Promise<BridgeConfig> {
  return request<BridgeConfig>('/api/config', {
    method: 'POST',
    body: JSON.stringify(update),
  });
}

export async function listSessions(): Promise<SessionMetadata[]> {
  return request<SessionMetadata[]>('/api/sessions');
}

export async function getSession(id: string): Promise<SessionDetail> {
  return request<SessionDetail>(`/api/sessions/${encodeURIComponent(id)}`);
}

export async function deleteSession(id: string): Promise<void> {
  await request<Record<string, unknown>>(`/api/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}
