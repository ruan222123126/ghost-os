import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
} from '@/lib/api/shared';
import type { BridgeConfig } from '@/lib/types';

const PROVIDER_TYPES = ['openai', 'anthropic', 'custom', 'codex'] as const;
const RELAY_STOP_POLICIES = ['ai_decides', 'max_rounds'] as const;
const SESSION_TITLE_MODES = ['session_id', 'first_message', 'ai_generated'] as const;
const EXTERNAL_CODEX_PERMISSION_MODES = ['read-only', 'default', 'safe-yolo', 'yolo'] as const;
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
  'external_codex_permission_mode',
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

type BridgeConfigRecord = Partial<Record<(typeof BRIDGE_CONFIG_KEYS)[number], unknown>>;

export function parseBridgeConfig(payload: unknown): BridgeConfig {
  const record = pickKnownKeys(expectRecord(payload, 'bridge config'), BRIDGE_CONFIG_KEYS);

  return {
    ...parseBridgeProviderFields(record),
    ...parseBridgeExecutionFields(record),
    ...parseBridgeFeatureFields(record),
    ...parseBridgeWebSearchFields(record),
  };
}

function parseBridgeProviderFields(record: BridgeConfigRecord): Pick<
  BridgeConfig,
  'provider' | 'provider_type' | 'base_url' | 'model' | 'chat_path' | 'project_root'
> {
  return {
    provider: expectString(record.provider, 'bridge config.provider'),
    provider_type: expectStringEnum(record.provider_type, PROVIDER_TYPES, 'bridge config.provider_type'),
    base_url: expectString(record.base_url, 'bridge config.base_url'),
    model: expectString(record.model, 'bridge config.model'),
    chat_path: expectString(record.chat_path, 'bridge config.chat_path'),
    project_root: expectString(record.project_root, 'bridge config.project_root'),
  };
}

function parseBridgeExecutionFields(record: BridgeConfigRecord): Pick<
  BridgeConfig,
  | 'max_turns'
  | 'task_execution_timeout_ms'
  | 'relay_default_stop_policy'
  | 'relay_default_max_rounds'
  | 'relay_default_execution_timeout_ms'
  | 'external_codex_permission_mode'
  | 'llm_completion_retry_count'
  | 'llm_completion_retry_interval_ms'
> {
  return {
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
    external_codex_permission_mode: expectStringEnum(
      record.external_codex_permission_mode,
      EXTERNAL_CODEX_PERMISSION_MODES,
      'bridge config.external_codex_permission_mode',
    ),
    llm_completion_retry_count: expectNumber(
      record.llm_completion_retry_count,
      'bridge config.llm_completion_retry_count',
    ),
    llm_completion_retry_interval_ms: expectNumber(
      record.llm_completion_retry_interval_ms,
      'bridge config.llm_completion_retry_interval_ms',
    ),
  };
}

function parseBridgeFeatureFields(record: BridgeConfigRecord): Pick<
  BridgeConfig,
  | 'api_key_set'
  | 'model_selection_enabled'
  | 'session_human_log_full_enabled'
  | 'session_system_prompt_visible_enabled'
  | 'assistant_markdown_enabled'
  | 'tool_call_compact_output_enabled'
  | 'memory_mode_enabled'
  | 'microcompact_enabled'
  | 'session_title_mode'
> {
  return {
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
  };
}

function parseBridgeWebSearchFields(record: BridgeConfigRecord): Pick<
  BridgeConfig,
  | 'web_search_tavily_url'
  | 'web_search_exa_url'
  | 'web_search_tavily_api_key_set'
  | 'web_search_exa_api_key_set'
> {
  return {
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
