import type {
  BridgeConfig,
  ProviderConfig,
  ProviderListResponse,
} from '@/lib/types';
import {
  ensureKnownKeys,
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalRecord,
  parseOptionalStringArray,
} from '@/lib/api/shared';

const PROVIDER_TYPES = ['openai', 'anthropic', 'custom', 'codex'] as const;
const BRIDGE_CONFIG_KEYS = [
  'provider',
  'provider_type',
  'base_url',
  'model',
  'chat_path',
  'api_key_set',
  'model_selection_enabled',
] as const;
const PROVIDER_CONFIG_KEYS = [
  'name',
  'type',
  'base_url',
  'models',
  'context_window_tokens',
  'response_reserve_tokens',
  'model_context_window_tokens',
  'model_response_reserve_tokens',
  'api_key_set',
] as const;
const PROVIDER_LIST_KEYS = ['providers', 'active_provider'] as const;

function parseProviderConfig(value: unknown, label: string): ProviderConfig {
  const record = expectRecord(value, label);
  ensureKnownKeys(record, PROVIDER_CONFIG_KEYS, label);

  return {
    name: expectString(record.name, `${label}.name`),
    type: expectStringEnum(record.type, PROVIDER_TYPES, `${label}.type`),
    base_url: expectString(record.base_url, `${label}.base_url`),
    models: parseOptionalStringArray(record.models, `${label}.models`),
    context_window_tokens: record.context_window_tokens === undefined
      ? undefined
      : expectNumber(record.context_window_tokens, `${label}.context_window_tokens`),
    response_reserve_tokens: record.response_reserve_tokens === undefined
      ? undefined
      : expectNumber(record.response_reserve_tokens, `${label}.response_reserve_tokens`),
    model_context_window_tokens: parseOptionalRecord(
      record.model_context_window_tokens,
      `${label}.model_context_window_tokens`,
    ),
    model_response_reserve_tokens: parseOptionalRecord(
      record.model_response_reserve_tokens,
      `${label}.model_response_reserve_tokens`,
    ),
    api_key_set: expectBoolean(record.api_key_set, `${label}.api_key_set`),
  };
}

export function parseBridgeConfig(payload: unknown): BridgeConfig {
  const record = expectRecord(payload, 'bridge config');
  ensureKnownKeys(record, BRIDGE_CONFIG_KEYS, 'bridge config');

  return {
    provider: expectString(record.provider, 'bridge config.provider'),
    provider_type: expectStringEnum(record.provider_type, PROVIDER_TYPES, 'bridge config.provider_type'),
    base_url: expectString(record.base_url, 'bridge config.base_url'),
    model: expectString(record.model, 'bridge config.model'),
    chat_path: expectString(record.chat_path, 'bridge config.chat_path'),
    api_key_set: expectBoolean(record.api_key_set, 'bridge config.api_key_set'),
    model_selection_enabled: expectBoolean(
      record.model_selection_enabled,
      'bridge config.model_selection_enabled',
    ),
  };
}

export function parseProviderListResponse(payload: unknown): ProviderListResponse {
  const record = expectRecord(payload, 'provider list');
  ensureKnownKeys(record, PROVIDER_LIST_KEYS, 'provider list');
  if (!Array.isArray(record.providers)) {
    throw new Error('Invalid provider list.providers: expected array');
  }

  return {
    active_provider: expectString(record.active_provider, 'provider list.active_provider'),
    providers: record.providers.map((provider, index) => {
      return parseProviderConfig(provider, `provider list.providers[${index}]`);
    }),
  };
}
