import type { ApiEnvelope } from '@/lib/types';

export type PayloadParser<TPayload> = (payload: unknown) => TPayload;

function buildHeaders(headers: HeadersInit | undefined): Headers {
  const merged = new Headers(headers);
  if (!merged.has('Content-Type')) {
    merged.set('Content-Type', 'application/json');
  }

  return merged;
}

function extractEnvelopeError(
  envelope: ApiEnvelope<unknown>,
  status: number,
): Error {
  return new Error(envelope.error || `Request failed with status ${status}`);
}

function successEnvelope(payload: unknown): ApiEnvelope<unknown> {
  return {
    status: 'success',
    payload: payload ?? {},
    error: '',
  };
}

function normalizeErrorPayload(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  return {};
}

function errorEnvelope(error: string, payload: unknown = {}): ApiEnvelope<unknown> {
  return {
    status: 'error',
    payload: normalizeErrorPayload(payload),
    error,
  };
}

async function parseEnvelope(
  response: Response,
): Promise<ApiEnvelope<unknown>> {
  const raw = await response.text();
  const trimmed = raw.trim();
  if (trimmed.length === 0) {
    if (response.ok) {
      return successEnvelope({});
    }
    return errorEnvelope(`empty response body (status ${response.status})`);
  }

  try {
    const decoded = JSON.parse(trimmed) as Partial<ApiEnvelope<unknown>>;
    if (decoded.status === 'success') {
      return successEnvelope(decoded.payload ?? {});
    }
    if (decoded.status === 'error') {
      const message = typeof decoded.error === 'string' ? decoded.error : '';
      return errorEnvelope(message, decoded.payload ?? {});
    }
    if (response.ok) {
      return successEnvelope(decoded.payload ?? {});
    }
    const message = typeof decoded.error === 'string'
      ? decoded.error
      : `invalid response envelope (status ${response.status})`;
    return errorEnvelope(message, decoded.payload ?? {});
  } catch {
    const preview = trimmed.length > 200 ? `${trimmed.slice(0, 200)}...` : trimmed;
    return errorEnvelope(`bridge returned non-JSON response (status ${response.status}): ${preview}`);
  }
}

export async function requestJSON<TPayload>(
  path: string,
  init: RequestInit = {},
  parser?: PayloadParser<TPayload>,
): Promise<TPayload> {
  const response = await fetch(path, {
    ...init,
    headers: buildHeaders(init.headers),
    cache: 'no-store',
  });
  const envelope = await parseEnvelope(response);

  if (!response.ok || envelope.status === 'error') {
    throw extractEnvelopeError(envelope, response.status);
  }

  return parser ? parser(envelope.payload) : (envelope.payload as TPayload);
}

export async function requestOptionalJSON<TPayload>(
  path: string,
  init: RequestInit = {},
  parser?: PayloadParser<TPayload>,
): Promise<TPayload | null> {
  const response = await fetch(path, {
    ...init,
    headers: buildHeaders(init.headers),
    cache: 'no-store',
  });
  const envelope = await parseEnvelope(response);

  if (response.status === 404) {
    return null;
  }
  if (!response.ok || envelope.status === 'error') {
    throw extractEnvelopeError(envelope, response.status);
  }

  return parser ? parser(envelope.payload) : (envelope.payload as TPayload);
}
