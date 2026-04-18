import type { SessionMetadata } from '@/lib/types';

export const SESSION_ALIAS_STORAGE_KEY = 'ghost.web.session.aliases.v1';
export const SESSION_ALIAS_VERSION = 1 as const;

export interface SessionAliasStoreV1 {
  version: 1;
  aliases: Record<string, string>;
}

export function createInitialSessionAliasStore(): SessionAliasStoreV1 {
  return {
    version: SESSION_ALIAS_VERSION,
    aliases: {},
  };
}

export function stringifySessionAliasStore(store: SessionAliasStoreV1): string {
  return JSON.stringify(store);
}

export function parseSessionAliasStore(raw: string): SessionAliasStoreV1 {
  const payload = parseSessionAliasPayload(raw);
  if (payload.version !== SESSION_ALIAS_VERSION) {
    throw new Error(`Invalid session alias version: ${payload.version}`);
  }

  return {
    version: SESSION_ALIAS_VERSION,
    aliases: payload.aliases,
  };
}

export function sanitizeSessionAliasStore(
  store: SessionAliasStoreV1,
  sessions: SessionMetadata[],
): SessionAliasStoreV1 {
  const knownSessionIDs = new Set(sessions.map((session) => session.id));
  const aliases: Record<string, string> = {};

  for (const [sessionID, rawAlias] of Object.entries(store.aliases)) {
    const normalizedID = sessionID.trim();
    if (!normalizedID || !knownSessionIDs.has(normalizedID)) {
      continue;
    }

    const alias = rawAlias.trim();
    if (!alias) {
      continue;
    }
    aliases[normalizedID] = alias;
  }

  return {
    version: store.version,
    aliases,
  };
}

export function setSessionAlias(
  store: SessionAliasStoreV1,
  sessionID: string,
  alias: string,
): SessionAliasStoreV1 {
  const normalizedID = sessionID.trim();
  const normalizedAlias = alias.trim();
  if (!normalizedID) {
    throw new Error('Session id cannot be empty');
  }
  if (!normalizedAlias) {
    throw new Error('Session alias cannot be empty');
  }

  return {
    version: store.version,
    aliases: {
      ...store.aliases,
      [normalizedID]: normalizedAlias,
    },
  };
}

interface ParsedSessionAliasPayload {
  version: number;
  aliases: Record<string, string>;
}

function parseSessionAliasPayload(raw: string): ParsedSessionAliasPayload {
  let decoded: unknown;
  try {
    decoded = JSON.parse(raw);
  } catch (error) {
    throw new Error(`Invalid session alias payload JSON: ${String(error)}`);
  }

  const root = expectRecord(decoded, 'session alias payload');
  return {
    version: expectNumber(root.version, 'session alias payload.version'),
    aliases: parseAliases(root.aliases),
  };
}

function parseAliases(value: unknown): Record<string, string> {
  const record = expectRecord(value, 'session alias payload.aliases');
  const parsed: Record<string, string> = {};

  for (const [sessionID, rawAlias] of Object.entries(record)) {
    const normalizedID = sessionID.trim();
    const alias = expectString(rawAlias, `session alias payload.aliases.${sessionID}`).trim();
    if (!normalizedID || !alias) {
      continue;
    }
    parsed[normalizedID] = alias;
  }

  return parsed;
}

function expectRecord(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected object`);
  }
  return value as Record<string, unknown>;
}

function expectString(value: unknown, label: string): string {
  if (typeof value !== 'string') {
    throw new Error(`Invalid ${label}: expected string`);
  }
  return value;
}

function expectNumber(value: unknown, label: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Invalid ${label}: expected number`);
  }
  return value;
}
