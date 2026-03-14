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

async function parseEnvelope(
  response: Response,
): Promise<ApiEnvelope<unknown>> {
  return (await response.json()) as ApiEnvelope<unknown>;
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
