import { toErrorMessage } from '@/lib/errors';
import type { ApiEnvelope } from '@/lib/types';

function errorEnvelope(message: string): ApiEnvelope<Record<string, never>> {
  return {
    status: 'error',
    payload: {},
    error: message,
  };
}

export function bridgeUnavailableResponse(error: unknown): Response {
  return Response.json(errorEnvelope(toErrorMessage(error, 'bridge unavailable')), { status: 502 });
}

export function invalidJSONBodyResponse(): Response {
  return Response.json(errorEnvelope('invalid JSON body'), { status: 400 });
}
