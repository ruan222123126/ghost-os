import type { BridgeMethod } from './types';
import { invalidJSONBodyResponse } from './error';

export function withJSONHeaders(headers: HeadersInit | undefined): Headers {
  const merged = new Headers(headers);
  if (!merged.has('Content-Type')) {
    merged.set('Content-Type', 'application/json');
  }
  return merged;
}

export function methodAcceptsJSONBody(method: BridgeMethod): boolean {
  return method !== 'GET' && method !== 'DELETE';
}

type ParsedJSONBody =
  | {
      ok: true;
      hasBody: boolean;
      body?: unknown;
    }
  | {
      ok: false;
      response: Response;
    };

export async function parseJSONBody(request: Request): Promise<ParsedJSONBody> {
  const raw = await request.text();
  if (raw.trim() === '') {
    return {
      ok: true,
      hasBody: false,
    };
  }

  try {
    return {
      ok: true,
      hasBody: true,
      body: JSON.parse(raw),
    };
  } catch {
    return {
      ok: false,
      response: invalidJSONBodyResponse(),
    };
  }
}
