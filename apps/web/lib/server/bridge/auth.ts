import { readFile } from 'fs/promises';
import { homedir } from 'os';
import { join } from 'path';

const DEFAULT_BRIDGE_CONFIG_PATH = join(homedir(), '.ghost-os', 'config.toml');

function resolveConfigPath(rawPath: string | undefined): string {
  const configuredPath = rawPath?.trim();
  if (!configuredPath) {
    return DEFAULT_BRIDGE_CONFIG_PATH;
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

export async function resolveBridgeHeaders(headers: HeadersInit | undefined, request?: Request): Promise<Headers> {
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

  const token = process.env.GHOST_API_TOKEN?.trim() || (await loadBridgeTokenFromConfig());
  if (token) {
    merged.set('X-API-Token', token);
  }
  return merged;
}
