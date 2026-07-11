import { readFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

const defaultBridgeConfigPath = path.join(os.homedir(), '.ghost-os', 'config.toml');
const apiTokenPattern = /^api_token\s*=\s*"([^"\n]*)"/m;

type BridgeConfigPath = {
  explicit: boolean;
  value: string;
};

function bridgeConfigPath(): BridgeConfigPath {
  const configured = process.env.GHOST_CONFIG_PATH?.trim();
  if (configured) {
    return { explicit: true, value: configured };
  }
  return { explicit: false, value: defaultBridgeConfigPath };
}

function parseAPIToken(configText: string): string | undefined {
  const match = configText.match(apiTokenPattern);
  const token = match?.[1]?.trim();
  if (!token) {
    return undefined;
  }
  return token;
}

function isMissingFile(error: unknown): boolean {
  return typeof error === 'object' && error !== null && 'code' in error && error.code === 'ENOENT';
}

export async function readBridgeAPIToken(): Promise<string | undefined> {
  const configPath = bridgeConfigPath();
  try {
    const configText = await readFile(configPath.value, 'utf8');
    return parseAPIToken(configText);
  } catch (error) {
    if (!configPath.explicit && isMissingFile(error)) {
      return undefined;
    }
    throw error;
  }
}
