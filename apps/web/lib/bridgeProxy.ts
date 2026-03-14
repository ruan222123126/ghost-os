// Local proxy utilities for forwarding frontend requests to the bridge service.

import { readFile } from 'fs/promises';
import { homedir } from 'os';
import { join } from 'path';

import { toErrorMessage } from '@/lib/errors';
import type { ApiEnvelope } from '@/lib/types';

const bridgeBaseURL =
  process.env.GHOST_BRIDGE_URL ?? process.env.NEXT_PUBLIC_API_BASE ?? 'http://127.0.0.1:8080';
const defaultBridgeConfigPath = join(homedir(), '.ghost-os', 'config.toml');

interface ForwardBridgeOptions {
  path: string;
  method: NonNullable<RequestInit['method']>;
  request?: Request;
  headers?: HeadersInit;
}

type BridgeMethod = ForwardBridgeOptions['method'];

interface BridgeRouteContext<Params extends Record<string, string>> {
  params: Params;
}

type BridgeRouteHeaders<Params extends Record<string, string>> =
  | HeadersInit
  | ((context: { params: Params; request: Request }) => HeadersInit | undefined);

interface BridgeRouteHandlerOptions<Params extends Record<string, string>> {
  headers?: BridgeRouteHeaders<Params>;
}

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

function resolveConfigPath(rawPath: string | undefined): string {
  const configuredPath = rawPath?.trim();
  if (!configuredPath) {
    return defaultBridgeConfigPath;
  }
  if (configuredPath === '~') {
    return homedir();
  }
  if (configuredPath.startsWith('~/')) {
    return join(homedir(), configuredPath.slice(2));
  }
  return configuredPath;
}

function parseTomlString(raw: string): string | undefined {
  const doubleQuoted = raw.match(/^"((?:[^"\\]|\\.)*)"$/);
  if (doubleQuoted) {
    try {
      return JSON.parse(`"${doubleQuoted[1]}"`) as string;
    } catch {
      return doubleQuoted[1];
    }
  }

  const singleQuoted = raw.match(/^'([^']*)'$/);
  if (singleQuoted) {
    return singleQuoted[1];
  }

  return undefined;
}

function parseBridgeTokenFromConfig(raw: string): string | undefined {
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) {
      continue;
    }

    const match = trimmed.match(/^api_token\s*=\s*(.+?)\s*(?:#.*)?$/);
    if (!match) {
      continue;
    }

    const token = parseTomlString(match[1].trim())?.trim();
    if (token) {
      return token;
    }
  }

  return undefined;
}

async function loadBridgeTokenFromConfig(): Promise<string | undefined> {
  try {
    const raw = await readFile(resolveConfigPath(process.env.GHOST_CONFIG_PATH), 'utf8');
    return parseBridgeTokenFromConfig(raw);
  } catch {
    return undefined;
  }
}

function forwardedAuthHeaders(request?: Request): Headers | undefined {
  if (!request) {
    return undefined;
  }

  const headers = new Headers();
  const apiToken = request.headers.get('X-API-Token')?.trim();
  const authorization = request.headers.get('Authorization')?.trim();

  if (apiToken) {
    headers.set('X-API-Token', apiToken);
  }
  if (authorization) {
    headers.set('Authorization', authorization);
  }

  return Array.from(headers.keys()).length > 0 ? headers : undefined;
}

async function resolveBridgeHeaders(headers: HeadersInit | undefined, request?: Request): Promise<Headers> {
  const merged = new Headers(headers);
  if (merged.has('X-API-Token') || merged.has('Authorization')) {
    return merged;
  }

  const forwarded = forwardedAuthHeaders(request);
  if (forwarded) {
    for (const [key, value] of forwarded.entries()) {
      merged.set(key, value);
    }
    return merged;
  }

  const envToken = process.env.GHOST_API_TOKEN?.trim();
  const token = envToken || (await loadBridgeTokenFromConfig());
  if (token) {
    merged.set('X-API-Token', token);
  }

  return merged;
}

export async function forwardBridge({ path, method, request, headers }: ForwardBridgeOptions): Promise<Response> {
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
    const response = await fetch(`${bridgeBaseURL}${path}`, {
      method: 'GET',
      cache: 'no-store',
      headers: await resolveBridgeHeaders(undefined, request),
    });
    const body = await response.arrayBuffer();
    const forwardedHeaders = new Headers();
    for (const header of ['Content-Type', 'Content-Length', 'Content-Disposition', 'ETag', 'X-Artifact-SHA256']) {
      const value = response.headers.get(header);
      if (value) {
        forwardedHeaders.set(header, value);
      }
    }
    return new Response(body, {
      status: response.status,
      headers: forwardedHeaders,
    });
  } catch (error) {
    return bridgeUnavailableResponse(error);
  }
}

export function createBridgeRouteHandler(
  method: BridgeMethod,
  path: string,
  options: BridgeRouteHandlerOptions<Record<string, never>> = {}
) {
  return createParamBridgeRouteHandler<Record<string, never>>(method, () => path, options);
}

export function createParamBridgeRouteHandler<Params extends Record<string, string>>(
  method: BridgeMethod,
  getPath: (params: Params) => string,
  options: BridgeRouteHandlerOptions<Params> = {}
) {
  return async function routeHandler(request: Request, context?: BridgeRouteContext<Params>): Promise<Response> {
    const params = (context?.params ?? {}) as Params;
    const headers = typeof options.headers === 'function' ? options.headers({ params, request }) : options.headers;

    return forwardBridge({
      path: getPath(params),
      method,
      request,
      headers,
    });
  };
}

async function passThroughToBridge(path: string, init: RequestInit): Promise<Response> {
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

function bridgeUnavailableResponse(error: unknown): Response {
  return Response.json(errorEnvelope(toErrorMessage(error, 'bridge unavailable')), { status: 502 });
}

function methodAcceptsJSONBody(method: BridgeMethod): boolean {
  return method !== 'GET' && method !== 'DELETE';
}

function invalidJSONBodyResponse(): Response {
  return Response.json(errorEnvelope('invalid JSON body'), { status: 400 });
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

async function parseJSONBody(request: Request): Promise<ParsedJSONBody> {
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
