import { toErrorMessage } from '@/lib/errors';
import type { ApiEnvelope } from '@/lib/types';

const bridgeBaseURL =
  process.env.GHOST_BRIDGE_URL ?? process.env.NEXT_PUBLIC_API_BASE ?? 'http://127.0.0.1:8080';

function errorEnvelope(message: string): ApiEnvelope<Record<string, never>> {
  return {
    status: 'error',
    payload: {},
    error: message,
  };
}

function withJSONHeaders(headers: HeadersInit | undefined): Headers {
  const merged = new Headers(headers);
  if (!merged.has('Content-Type')) {
    merged.set('Content-Type', 'application/json');
  }
  return merged;
}

export async function passThroughToBridge(path: string, init: RequestInit): Promise<Response> {
  const response = await fetch(`${bridgeBaseURL}${path}`, {
    ...init,
    cache: 'no-store',
    headers: withJSONHeaders(init.headers),
  });

  const text = await response.text();
  return new Response(text, {
    status: response.status,
    headers: {
      'Content-Type': response.headers.get('Content-Type') ?? 'application/json',
    },
  });
}

export function bridgeUnavailableResponse(error: unknown): Response {
  return Response.json(errorEnvelope(toErrorMessage(error, 'bridge unavailable')), { status: 502 });
}

export function invalidJSONBodyResponse(): Response {
  return Response.json(errorEnvelope('invalid JSON body'), { status: 400 });
}

export type ParsedJSONBody =
  | {
      ok: true;
      body: unknown;
    }
  | {
      ok: false;
      response: Response;
    };

export async function parseJSONBody(request: Request): Promise<ParsedJSONBody> {
  try {
    return {
      ok: true,
      body: await request.json(),
    };
  } catch {
    return {
      ok: false,
      response: invalidJSONBodyResponse(),
    };
  }
}
