import { resolveBridgeHeaders } from './auth';
import { bridgeUnavailableResponse } from './error';
import { methodAcceptsJSONBody, parseJSONBody, withJSONHeaders } from './request';
import type { ForwardBridgeOptions } from './types';

const BRIDGE_BASE_URL =
  process.env.GHOST_BRIDGE_URL ?? process.env.NEXT_PUBLIC_API_BASE ?? 'http://127.0.0.1:8080';

const DOWNLOAD_HEADERS = ['Content-Type', 'Content-Length', 'Content-Disposition', 'ETag', 'X-Artifact-SHA256'];

async function passThroughToBridge(path: string, init: RequestInit): Promise<Response> {
  const response = await fetch(`${BRIDGE_BASE_URL}${path}`, {
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
    const response = await fetch(`${BRIDGE_BASE_URL}${path}`, {
      method: 'GET',
      cache: 'no-store',
      headers: await resolveBridgeHeaders(undefined, request),
    });
    const body = await response.arrayBuffer();
    const headers = new Headers();
    for (const header of DOWNLOAD_HEADERS) {
      const value = response.headers.get(header);
      if (value) {
        headers.set(header, value);
      }
    }
    return new Response(body, {
      status: response.status,
      headers,
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}
