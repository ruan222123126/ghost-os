import type {
  ProviderConfig,
  ProviderListResponse,
  SystemPromptPayload,
  SystemPromptToolDefinition,
} from '@/lib/types';
import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalNumberRecord,
  pickKnownKeys,
  parseOptionalStringArray,
} from '@/lib/api/shared';

const PROVIDER_TYPES = ['openai', 'anthropic', 'custom', 'codex'] as const;
const PROMPT_INSERT_POINTS = ['rule', 'core_job', 'memory', 'context'] as const;
const PROVIDER_CONFIG_KEYS = [
  'name',
  'type',
  'base_url',
  'provider_id',
  'updated_at',
  'deleted_at',
  'models',
  'context_window_tokens',
  'response_reserve_tokens',
  'model_context_window_tokens',
  'model_response_reserve_tokens',
  'api_key_set',
] as const;
const SYSTEM_PROMPT_KEYS = [
  'core_prompt',
  'rendered_prompt',
  'prompt_library',
  'tool_definitions',
] as const;
const PROMPT_LIBRARY_ITEM_KEYS = [
  'id',
  'name',
  'insert_point',
  'content',
  'active',
] as const;
const SYSTEM_PROMPT_TOOL_DEFINITION_KEYS = [
  'name',
  'description',
  'parameters',
] as const;
const PROVIDER_SYNC_RECORD_KEYS = [
  'provider_id',
  'updated_at',
  'deleted_at',
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
const PROVIDER_LIST_KEYS = ['providers', 'active_provider', 'provider_sync_records'] as const;

export { parseBridgeConfig } from './parser.bridgeConfig';

function parseProviderConfig(value: unknown, label: string): ProviderConfig {
  const record = pickKnownKeys(expectRecord(value, label), PROVIDER_CONFIG_KEYS);

  return {
    name: expectString(record.name, `${label}.name`),
    type: expectStringEnum(record.type, PROVIDER_TYPES, `${label}.type`),
    base_url: expectString(record.base_url, `${label}.base_url`),
    provider_id: expectString(record.provider_id, `${label}.provider_id`),
    updated_at: expectString(record.updated_at, `${label}.updated_at`),
    deleted_at: record.deleted_at === undefined
      ? undefined
      : expectString(record.deleted_at, `${label}.deleted_at`),
    models: parseOptionalStringArray(record.models, `${label}.models`),
    context_window_tokens: record.context_window_tokens === undefined
      ? undefined
      : expectNumber(record.context_window_tokens, `${label}.context_window_tokens`),
    response_reserve_tokens: record.response_reserve_tokens === undefined
      ? undefined
      : expectNumber(record.response_reserve_tokens, `${label}.response_reserve_tokens`),
    model_context_window_tokens: parseOptionalNumberRecord(
      record.model_context_window_tokens,
      `${label}.model_context_window_tokens`,
    ),
    model_response_reserve_tokens: parseOptionalNumberRecord(
      record.model_response_reserve_tokens,
      `${label}.model_response_reserve_tokens`,
    ),
    api_key_set: expectBoolean(record.api_key_set, `${label}.api_key_set`),
  };
}

function parseProviderSyncRecord(value: unknown, label: string) {
  const record = pickKnownKeys(expectRecord(value, label), PROVIDER_SYNC_RECORD_KEYS);

  return {
    provider_id: expectString(record.provider_id, `${label}.provider_id`),
    updated_at: expectString(record.updated_at, `${label}.updated_at`),
    deleted_at: record.deleted_at === undefined
      ? undefined
      : expectString(record.deleted_at, `${label}.deleted_at`),
    name: record.name === undefined
      ? undefined
      : expectString(record.name, `${label}.name`),
    type: record.type === undefined
      ? undefined
      : expectStringEnum(record.type, PROVIDER_TYPES, `${label}.type`),
    base_url: record.base_url === undefined
      ? undefined
      : expectString(record.base_url, `${label}.base_url`),
    models: parseOptionalStringArray(record.models, `${label}.models`),
    context_window_tokens: record.context_window_tokens === undefined
      ? undefined
      : expectNumber(record.context_window_tokens, `${label}.context_window_tokens`),
    response_reserve_tokens: record.response_reserve_tokens === undefined
      ? undefined
      : expectNumber(record.response_reserve_tokens, `${label}.response_reserve_tokens`),
    model_context_window_tokens: parseOptionalNumberRecord(
      record.model_context_window_tokens,
      `${label}.model_context_window_tokens`,
    ),
    model_response_reserve_tokens: parseOptionalNumberRecord(
      record.model_response_reserve_tokens,
      `${label}.model_response_reserve_tokens`,
    ),
    api_key_set: record.api_key_set === undefined
      ? undefined
      : expectBoolean(record.api_key_set, `${label}.api_key_set`),
  };
}

function parseProviderSyncRecords(value: unknown) {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error('Invalid provider list.provider_sync_records: expected array');
  }

  return value.map((record, index) => {
    return parseProviderSyncRecord(record, `provider list.provider_sync_records[${index}]`);
  });
}

function parsePromptLibraryItem(value: unknown, label: string) {
  const record = pickKnownKeys(expectRecord(value, label), PROMPT_LIBRARY_ITEM_KEYS);

  return {
    id: expectString(record.id, `${label}.id`),
    name: expectString(record.name, `${label}.name`),
    insert_point: expectStringEnum(record.insert_point, PROMPT_INSERT_POINTS, `${label}.insert_point`),
    content: expectString(record.content, `${label}.content`),
    active: expectBoolean(record.active, `${label}.active`),
  };
}

function parseSystemPromptToolDefinition(
  value: unknown,
  label: string,
): SystemPromptToolDefinition {
  const record = pickKnownKeys(expectRecord(value, label), SYSTEM_PROMPT_TOOL_DEFINITION_KEYS);

  return {
    name: expectString(record.name, `${label}.name`),
    description: expectString(record.description, `${label}.description`),
    parameters: parseOptionalToolDefinitionParameters(record.parameters, `${label}.parameters`),
  };
}

function parseOptionalToolDefinitionParameters(
  value: unknown,
  label: string,
): Record<string, unknown> | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  return expectRecord(value, label);
}

export function parseSystemPromptResponse(payload: unknown): SystemPromptPayload {
  const record = pickKnownKeys(expectRecord(payload, 'system prompt'), SYSTEM_PROMPT_KEYS);
  if (!Array.isArray(record.prompt_library)) {
    throw new Error('Invalid system prompt.prompt_library: expected array');
  }
  const rawToolDefinitions = record.tool_definitions;
  if (rawToolDefinitions !== undefined && !Array.isArray(rawToolDefinitions)) {
    throw new Error('Invalid system prompt.tool_definitions: expected array');
  }

  return {
    core_prompt: expectString(record.core_prompt, 'system prompt.core_prompt'),
    rendered_prompt: expectString(record.rendered_prompt, 'system prompt.rendered_prompt'),
    prompt_library: record.prompt_library.map((entry, index) => {
      return parsePromptLibraryItem(entry, `system prompt.prompt_library[${index}]`);
    }),
    tool_definitions: (rawToolDefinitions ?? []).map((entry, index) => {
      return parseSystemPromptToolDefinition(entry, `system prompt.tool_definitions[${index}]`);
    }),
  };
}

export function parseProviderListResponse(payload: unknown): ProviderListResponse {
  const record = pickKnownKeys(expectRecord(payload, 'provider list'), PROVIDER_LIST_KEYS);
  if (!Array.isArray(record.providers)) {
    throw new Error('Invalid provider list.providers: expected array');
  }

  return {
    active_provider: expectString(record.active_provider, 'provider list.active_provider'),
    provider_sync_records: parseProviderSyncRecords(record.provider_sync_records),
    providers: record.providers.map((provider, index) => {
      return parseProviderConfig(provider, `provider list.providers[${index}]`);
    }),
  };
}
