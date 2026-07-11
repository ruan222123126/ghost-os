import { readBridgeAPIToken } from './config';

const placeholderAPITokens = new Set(['change-me']);

function hasAuthHeaders(headers: Headers): boolean {
  return headers.has('X-API-Token') || headers.has('Authorization');
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

async function configuredBridgeToken(): Promise<string | undefined> {
  const envToken = process.env.GHOST_API_TOKEN?.trim();
  if (envToken && !placeholderAPITokens.has(envToken.toLowerCase())) {
    return envToken;
  }
  return readBridgeAPIToken();
}

export async function resolveBridgeHeaders(headers: HeadersInit | undefined, request?: Request): Promise<Headers> {
  const merged = new Headers(headers);
  if (hasAuthHeaders(merged)) {
    return merged;
  }

  const forwarded = forwardedAuthHeaders(request);
  if (forwarded) {
    for (const [key, value] of forwarded.entries()) {
      merged.set(key, value);
    }
    return merged;
  }

  const token = await configuredBridgeToken();
  if (token) {
    merged.set('X-API-Token', token);
  }
  return merged;
}
