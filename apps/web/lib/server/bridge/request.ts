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
