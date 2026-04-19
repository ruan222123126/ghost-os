import type {
  BridgeConfig,
  GraphQLDomainResponse,
  GraphQLMutationPolicyResponse,
  GraphQLSourceResponse,
  ProviderConfig,
  ProviderListResponse,
  SystemPromptPayload,
} from '@/lib/types';
import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringArray,
  expectStringEnum,
  parseOptionalNumberRecord,
  pickKnownKeys,
  parseOptionalStringRecord,
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
  'graphql_default_source',
  'graphql_tool_runtime_enabled',
  'graphql_text_sanitize_enabled',
  'graphql_sources',
  'graphql_mutation_policies',
  'session_human_log_full_enabled',
  'web_rooter_enabled',
  'web_rooter_base_url',
  'web_rooter_timeout_ms',
  'web_rooter_api_token_set',
  'web_search_tavily_url',
  'web_search_exa_url',
  'web_search_tavily_api_key_set',
  'web_search_exa_api_key_set',
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
const SYSTEM_PROMPT_KEYS = [
  'global_template',
  'core_prompt',
  'tool_prompt',
  'tool_key_spec',
  'rendered_prompt',
] as const;
const PROVIDER_LIST_KEYS = ['providers', 'active_provider'] as const;
const GRAPHQL_DOMAIN_KEYS = [
  'name',
  'description',
  'root_queries',
  'types',
  'max_depth',
  'max_fields',
  'max_root_fields',
] as const;
const GRAPHQL_SOURCE_KEYS = [
  'name',
  'description',
  'endpoint',
  'schema_path',
  'timeout_ms',
  'max_response_bytes',
  'max_depth',
  'max_fields',
  'max_root_fields',
  'max_fragments',
  'headers',
  'api_key_set',
  'domains',
] as const;
const GRAPHQL_MUTATION_POLICY_KEYS = [
  'name',
  'description',
  'source',
  'domain',
  'root_mutation',
  'approval_required',
  'idempotency_mode',
  'idempotency_header',
  'idempotency_variable_path',
  'max_depth',
  'max_fields',
  'max_root_fields',
  'max_fragments',
] as const;
const GRAPHQL_IDEMPOTENCY_MODES = ['header', 'variable_path'] as const;

function parseProviderConfig(value: unknown, label: string): ProviderConfig {
  const record = pickKnownKeys(expectRecord(value, label), PROVIDER_CONFIG_KEYS);

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

function parseGraphQLDomainResponse(value: unknown, label: string): GraphQLDomainResponse {
  const record = pickKnownKeys(expectRecord(value, label), GRAPHQL_DOMAIN_KEYS);

  return {
    name: expectString(record.name, `${label}.name`),
    description: record.description === undefined
      ? undefined
      : expectString(record.description, `${label}.description`),
    root_queries: expectStringArray(record.root_queries, `${label}.root_queries`),
    types: parseOptionalStringArray(record.types, `${label}.types`),
    max_depth: record.max_depth === undefined ? undefined : expectNumber(record.max_depth, `${label}.max_depth`),
    max_fields: record.max_fields === undefined ? undefined : expectNumber(record.max_fields, `${label}.max_fields`),
    max_root_fields: record.max_root_fields === undefined
      ? undefined
      : expectNumber(record.max_root_fields, `${label}.max_root_fields`),
  };
}

function parseGraphQLSourceResponse(value: unknown, label: string): GraphQLSourceResponse {
  const record = pickKnownKeys(expectRecord(value, label), GRAPHQL_SOURCE_KEYS);

  let domains: GraphQLDomainResponse[] | undefined;
  if (record.domains !== undefined) {
    if (!Array.isArray(record.domains)) {
      throw new Error(`Invalid ${label}.domains: expected array`);
    }
    domains = record.domains.map((entry, index) => {
      return parseGraphQLDomainResponse(entry, `${label}.domains[${index}]`);
    });
  }

  return {
    name: expectString(record.name, `${label}.name`),
    description: record.description === undefined
      ? undefined
      : expectString(record.description, `${label}.description`),
    endpoint: expectString(record.endpoint, `${label}.endpoint`),
    schema_path: expectString(record.schema_path, `${label}.schema_path`),
    timeout_ms: expectNumber(record.timeout_ms, `${label}.timeout_ms`),
    max_response_bytes: expectNumber(record.max_response_bytes, `${label}.max_response_bytes`),
    max_depth: expectNumber(record.max_depth, `${label}.max_depth`),
    max_fields: expectNumber(record.max_fields, `${label}.max_fields`),
    max_root_fields: expectNumber(record.max_root_fields, `${label}.max_root_fields`),
    max_fragments: expectNumber(record.max_fragments, `${label}.max_fragments`),
    headers: parseOptionalStringRecord(record.headers, `${label}.headers`),
    api_key_set: expectBoolean(record.api_key_set, `${label}.api_key_set`),
    domains,
  };
}

function parseGraphQLMutationPolicyResponse(
  value: unknown,
  label: string,
): GraphQLMutationPolicyResponse {
  const record = pickKnownKeys(expectRecord(value, label), GRAPHQL_MUTATION_POLICY_KEYS);
  const approvalRequired = record.approval_required === undefined
    ? undefined
    : expectBoolean(record.approval_required, `${label}.approval_required`);

  return {
    name: expectString(record.name, `${label}.name`),
    description: record.description === undefined
      ? undefined
      : expectString(record.description, `${label}.description`),
    source: expectString(record.source, `${label}.source`),
    domain: expectString(record.domain, `${label}.domain`),
    root_mutation: expectString(record.root_mutation, `${label}.root_mutation`),
    ...(approvalRequired === undefined ? {} : { approval_required: approvalRequired }),
    idempotency_mode: expectStringEnum(
      record.idempotency_mode,
      GRAPHQL_IDEMPOTENCY_MODES,
      `${label}.idempotency_mode`,
    ),
    idempotency_header: record.idempotency_header === undefined
      ? undefined
      : expectString(record.idempotency_header, `${label}.idempotency_header`),
    idempotency_variable_path: record.idempotency_variable_path === undefined
      ? undefined
      : expectString(record.idempotency_variable_path, `${label}.idempotency_variable_path`),
    max_depth: record.max_depth === undefined ? undefined : expectNumber(record.max_depth, `${label}.max_depth`),
    max_fields: record.max_fields === undefined ? undefined : expectNumber(record.max_fields, `${label}.max_fields`),
    max_root_fields: record.max_root_fields === undefined
      ? undefined
      : expectNumber(record.max_root_fields, `${label}.max_root_fields`),
    max_fragments: record.max_fragments === undefined
      ? undefined
      : expectNumber(record.max_fragments, `${label}.max_fragments`),
  };
}

export function parseBridgeConfig(payload: unknown): BridgeConfig {
  const record = pickKnownKeys(expectRecord(payload, 'bridge config'), BRIDGE_CONFIG_KEYS);
  if (!Array.isArray(record.graphql_sources)) {
    throw new Error('Invalid bridge config.graphql_sources: expected array');
  }
  if (!Array.isArray(record.graphql_mutation_policies)) {
    throw new Error('Invalid bridge config.graphql_mutation_policies: expected array');
  }

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
    graphql_default_source: expectString(record.graphql_default_source, 'bridge config.graphql_default_source'),
    graphql_tool_runtime_enabled: expectBoolean(
      record.graphql_tool_runtime_enabled,
      'bridge config.graphql_tool_runtime_enabled',
    ),
    graphql_text_sanitize_enabled: expectBoolean(
      record.graphql_text_sanitize_enabled,
      'bridge config.graphql_text_sanitize_enabled',
    ),
    graphql_sources: record.graphql_sources.map((entry, index) => {
      return parseGraphQLSourceResponse(entry, `bridge config.graphql_sources[${index}]`);
    }),
    graphql_mutation_policies: record.graphql_mutation_policies.map((entry, index) => {
      return parseGraphQLMutationPolicyResponse(
        entry,
        `bridge config.graphql_mutation_policies[${index}]`,
      );
    }),
    session_human_log_full_enabled: expectBoolean(
      record.session_human_log_full_enabled,
      'bridge config.session_human_log_full_enabled',
    ),
    web_rooter_enabled: expectBoolean(
      record.web_rooter_enabled,
      'bridge config.web_rooter_enabled',
    ),
    web_rooter_base_url: expectString(
      record.web_rooter_base_url,
      'bridge config.web_rooter_base_url',
    ),
    web_rooter_timeout_ms: expectNumber(
      record.web_rooter_timeout_ms,
      'bridge config.web_rooter_timeout_ms',
    ),
    web_rooter_api_token_set: expectBoolean(
      record.web_rooter_api_token_set,
      'bridge config.web_rooter_api_token_set',
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

  return {
    global_template: expectString(record.global_template, 'system prompt.global_template'),
    core_prompt: expectString(record.core_prompt, 'system prompt.core_prompt'),
    tool_prompt: expectString(record.tool_prompt, 'system prompt.tool_prompt'),
    tool_key_spec: expectString(record.tool_key_spec, 'system prompt.tool_key_spec'),
    rendered_prompt: expectString(record.rendered_prompt, 'system prompt.rendered_prompt'),
  };
}

export function parseProviderListResponse(payload: unknown): ProviderListResponse {
  const record = pickKnownKeys(expectRecord(payload, 'provider list'), PROVIDER_LIST_KEYS);
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
