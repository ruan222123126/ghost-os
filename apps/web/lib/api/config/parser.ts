import type {
  BridgeConfig,
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
const RELAY_STOP_POLICIES = ['ai_decides', 'max_rounds'] as const;
const SESSION_TITLE_MODES = ['session_id', 'first_message', 'ai_generated'] as const;
const BRIDGE_CONFIG_KEYS = [
  'provider',
  'provider_type',
  'base_url',
  'model',
  'chat_path',
  'project_root',
  'max_turns',
  'task_execution_timeout_ms',
  'relay_default_stop_policy',
  'relay_default_max_rounds',
  'relay_default_execution_timeout_ms',
  'llm_completion_retry_count',
  'llm_completion_retry_interval_ms',
  'api_key_set',
  'model_selection_enabled',
  'session_human_log_full_enabled',
  'session_system_prompt_visible_enabled',
  'assistant_markdown_enabled',
  'tool_call_compact_output_enabled',
  'memory_mode_enabled',
  'microcompact_enabled',
  'session_title_mode',
  'web_search_tavily_url',
  'web_search_exa_url',
  'web_search_tavily_api_key_set',
  'web_search_exa_api_key_set',
] as const;
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

export function parseBridgeConfig(payload: unknown): BridgeConfig {
  const record = pickKnownKeys(expectRecord(payload, 'bridge config'), BRIDGE_CONFIG_KEYS);

  return {
    provider: expectString(record.provider, 'bridge config.provider'),
    provider_type: expectStringEnum(record.provider_type, PROVIDER_TYPES, 'bridge config.provider_type'),
    base_url: expectString(record.base_url, 'bridge config.base_url'),
    model: expectString(record.model, 'bridge config.model'),
    chat_path: expectString(record.chat_path, 'bridge config.chat_path'),
    project_root: expectString(record.project_root, 'bridge config.project_root'),
    max_turns: expectNumber(record.max_turns, 'bridge config.max_turns'),
    task_execution_timeout_ms: expectNumber(
      record.task_execution_timeout_ms,
      'bridge config.task_execution_timeout_ms',
    ),
    relay_default_stop_policy: expectStringEnum(
      record.relay_default_stop_policy,
      RELAY_STOP_POLICIES,
      'bridge config.relay_default_stop_policy',
    ),
    relay_default_max_rounds: expectNumber(
      record.relay_default_max_rounds,
      'bridge config.relay_default_max_rounds',
    ),
    relay_default_execution_timeout_ms: expectNumber(
      record.relay_default_execution_timeout_ms,
      'bridge config.relay_default_execution_timeout_ms',
    ),
    llm_completion_retry_count: expectNumber(
      record.llm_completion_retry_count,
      'bridge config.llm_completion_retry_count',
    ),
    llm_completion_retry_interval_ms: expectNumber(
      record.llm_completion_retry_interval_ms,
      'bridge config.llm_completion_retry_interval_ms',
    ),
    api_key_set: expectBoolean(record.api_key_set, 'bridge config.api_key_set'),
    model_selection_enabled: expectBoolean(
      record.model_selection_enabled,
      'bridge config.model_selection_enabled',
    ),
    session_human_log_full_enabled: expectBoolean(
      record.session_human_log_full_enabled,
      'bridge config.session_human_log_full_enabled',
    ),
    session_system_prompt_visible_enabled: expectBoolean(
      record.session_system_prompt_visible_enabled,
      'bridge config.session_system_prompt_visible_enabled',
    ),
    assistant_markdown_enabled: expectBoolean(
      record.assistant_markdown_enabled,
      'bridge config.assistant_markdown_enabled',
    ),
    tool_call_compact_output_enabled: expectBoolean(
      record.tool_call_compact_output_enabled,
      'bridge config.tool_call_compact_output_enabled',
    ),
    memory_mode_enabled: expectBoolean(
      record.memory_mode_enabled,
      'bridge config.memory_mode_enabled',
    ),
    microcompact_enabled: expectBoolean(
      record.microcompact_enabled,
      'bridge config.microcompact_enabled',
    ),
    session_title_mode: expectStringEnum(
      record.session_title_mode,
      SESSION_TITLE_MODES,
      'bridge config.session_title_mode',
    ),
    web_search_tavily_url: expectString(
      record.web_search_tavily_url,
      'bridge config.web_search_tavily_url',
    ),
    web_search_exa_url: expectString(
      record.web_search_exa_url,
      'bridge config.web_search_exa_url',
    ),
    web_search_tavily_api_key_set: expectBoolean(
      record.web_search_tavily_api_key_set,
      'bridge config.web_search_tavily_api_key_set',
    ),
    web_search_exa_api_key_set: expectBoolean(
      record.web_search_exa_api_key_set,
      'bridge config.web_search_exa_api_key_set',
    ),
  };
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
