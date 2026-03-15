import { resolveBridgeHeaders } from './auth';
import { bridgeUnavailableResponse } from './error';
import { methodAcceptsJSONBody, parseJSONBody, withJSONHeaders } from './request';
import type { ForwardBridgeOptions } from './types';

const BRIDGE_BASE_URL =
  process.env.GHOST_BRIDGE_URL ?? process.env.NEXT_PUBLIC_API_BASE ?? 'http://127.0.0.1:8080';

async function fetchBridge(path: string, init: RequestInit): Promise<Response> {
  return fetch(`${BRIDGE_BASE_URL}${path}`, {
    ...init,
    cache: 'no-store',
  });
}

async function passThroughToBridge(path: string, init: RequestInit): Promise<Response> {
  return fetchBridge(path, {
    ...init,
    headers: withJSONHeaders(init.headers),
  });
}

export async function forwardBridge(options: ForwardBridgeOptions): Promise<Response> {
  const { path, method, request, headers } = options;
  const init: RequestInit = {
    method,
    headers: await resolveBridgeHeaders(headers, request),
  };

  if (request && methodAcceptsJSONBody(method)) {
    const parsed = await parseJSONBody(request);
    if (!parsed.ok) {
      return parsed.response;
    }
    init.body = JSON.stringify(parsed.body);
  }

  try {
    return await passThroughToBridge(path, init);
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}

export async function forwardBridgeDownload(path: string, request: Request): Promise<Response> {
  try {
    return await fetchBridge(path, {
      method: 'GET',
      headers: await resolveBridgeHeaders(undefined, request),
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
