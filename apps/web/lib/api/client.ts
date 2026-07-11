import type { ApiEnvelope } from '@/lib/types';

export type PayloadParser<TPayload> = (payload: unknown) => TPayload;

const NON_JSON_PREVIEW_LIMIT = 200;

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
  if (!trimmed) {
    return envelopeFromEmptyBody(response);
  }

  try {
    const decoded = JSON.parse(trimmed) as Partial<ApiEnvelope<unknown>>;
    return envelopeFromDecodedJSON(response, decoded);
  } catch {
    return envelopeFromNonJSONBody(response, trimmed);
  }
}

function envelopeFromEmptyBody(response: Response): ApiEnvelope<unknown> {
  if (response.ok) {
    return successEnvelope({});
  }
  return errorEnvelope(`empty response body (status ${response.status})`);
}

function envelopeFromDecodedJSON(
  response: Response,
  decoded: Partial<ApiEnvelope<unknown>>,
): ApiEnvelope<unknown> {
  const payload = decodedEnvelopePayload(decoded);
  switch (decoded.status) {
    case 'success':
      return successEnvelope(payload);
    case 'error':
      return errorEnvelope(readEnvelopeError(decoded), payload);
    default:
      return envelopeFromUnknownStatus(response, decoded, payload);
  }
}

function envelopeFromUnknownStatus(
  response: Response,
  decoded: Partial<ApiEnvelope<unknown>>,
  payload: unknown,
): ApiEnvelope<unknown> {
  if (response.ok) {
    return successEnvelope(payload);
  }
  return errorEnvelope(invalidEnvelopeMessage(response, decoded), payload);
}

function decodedEnvelopePayload(envelope: Partial<ApiEnvelope<unknown>>): unknown {
  return envelope.payload ?? {};
}

function readEnvelopeError(envelope: Partial<ApiEnvelope<unknown>>): string {
  return typeof envelope.error === 'string' ? envelope.error : '';
}

function invalidEnvelopeMessage(
  response: Response,
  envelope: Partial<ApiEnvelope<unknown>>,
): string {
  return readEnvelopeError(envelope) || `invalid response envelope (status ${response.status})`;
}

function envelopeFromNonJSONBody(response: Response, body: string): ApiEnvelope<unknown> {
  return errorEnvelope(`bridge returned non-JSON response (status ${response.status}): ${bodyPreview(body)}`);
}

function bodyPreview(body: string): string {
  if (body.length <= NON_JSON_PREVIEW_LIMIT) {
    return body;
  }
  return `${body.slice(0, NON_JSON_PREVIEW_LIMIT)}...`;
}

const DEFAULT_TIMEOUT_MS = 10_000;

async function fetchWithTimeout(path: string, init: RequestInit, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(path, { ...init, signal: controller.signal });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error(`request timed out after ${timeoutMs}ms`);
    }
    throw error;
  } finally {
    clearTimeout(timer);
  }
}

export async function requestJSON<TPayload>(
  path: string,
  init: RequestInit = {},
  parser?: PayloadParser<TPayload>,
): Promise<TPayload> {
  const response = await fetchWithTimeout(path, {
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
