import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

const DEFAULT_RELAY_STOP_POLICY: RelayDefaultStopPolicy = 'ai_decides';
const DEFAULT_RELAY_MAX_ROUNDS = 20;
const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = 0;

export type RelayDefaultStopPolicy = BridgeConfig['relay_default_stop_policy'];

export interface RelaySettingsFormState {
  stopPolicy: RelayDefaultStopPolicy;
  maxRounds: string;
  executionTimeoutMS: string;
}

export function createRelaySettingsFormState(config: BridgeConfig | null): RelaySettingsFormState {
  return {
    stopPolicy: config?.relay_default_stop_policy ?? DEFAULT_RELAY_STOP_POLICY,
    maxRounds: String(config?.relay_default_max_rounds ?? DEFAULT_RELAY_MAX_ROUNDS),
    executionTimeoutMS: String(
      config?.relay_default_execution_timeout_ms ?? DEFAULT_RELAY_EXECUTION_TIMEOUT_MS,
    ),
  };
}

export function buildRelaySettingsUpdate(formState: RelaySettingsFormState): ConfigUpdate {
  return {
    relay_default_stop_policy: formState.stopPolicy,
    relay_default_max_rounds: parsePositiveInteger(
      formState.maxRounds,
      'relay_default_max_rounds',
    ),
    relay_default_execution_timeout_ms: parseNonNegativeInteger(
      formState.executionTimeoutMS,
      'relay_default_execution_timeout_ms',
    ),
  };
}

function parsePositiveInteger(raw: string, fieldName: string): number {
  const trimmed = raw.trim();
  if (!/^[1-9]\d*$/.test(trimmed)) {
    throw new Error(`${fieldName} must be a positive integer`);
  }
  return Number(trimmed);
}

function parseNonNegativeInteger(raw: string, fieldName: string): number {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    throw new Error(`${fieldName} must be a non-negative integer`);
  }
  return Number(trimmed);
}
