import type {
  BridgeConfig,
  ConfigUpdate,
} from '@/lib/types';
import type { WebLocale } from '@/lib/i18n/locale';

const DEFAULT_MAX_TURNS = 20;
const DEFAULT_TASK_EXECUTION_TIMEOUT_MS = 300000;
const DEFAULT_LLM_COMPLETION_RETRY_COUNT = 1;
const DEFAULT_LLM_COMPLETION_RETRY_INTERVAL_MS = 200;
const DEFAULT_SESSION_TITLE_MODE = 'session_id';

export interface RuntimeFormState {
  provider: string;
  model: string;
  projectRoot: string;
  maxTurns: string;
  taskExecutionTimeoutMS: string;
  llmCompletionRetryCount: string;
  llmCompletionRetryIntervalMS: string;
  sessionHumanLogFullEnabled: boolean;
  sessionSystemPromptVisibleEnabled: boolean;
  assistantMarkdownEnabled: boolean;
  memoryModeEnabled: boolean;
  microcompactEnabled: boolean;
  sessionTitleMode: BridgeConfig['session_title_mode'];
  webSearchTavilyURL: string;
  webSearchExaURL: string;
  webSearchTavilyAPIKey: string;
  webSearchExaAPIKey: string;
}

const DEFAULT_RUNTIME_FORM_STATE: RuntimeFormState = {
  provider: '',
  model: '',
  projectRoot: '',
  maxTurns: String(DEFAULT_MAX_TURNS),
  taskExecutionTimeoutMS: String(DEFAULT_TASK_EXECUTION_TIMEOUT_MS),
  llmCompletionRetryCount: String(DEFAULT_LLM_COMPLETION_RETRY_COUNT),
  llmCompletionRetryIntervalMS: String(DEFAULT_LLM_COMPLETION_RETRY_INTERVAL_MS),
  sessionHumanLogFullEnabled: false,
  sessionSystemPromptVisibleEnabled: true,
  assistantMarkdownEnabled: true,
  memoryModeEnabled: false,
  microcompactEnabled: false,
  sessionTitleMode: DEFAULT_SESSION_TITLE_MODE,
  webSearchTavilyURL: '',
  webSearchExaURL: '',
  webSearchTavilyAPIKey: '',
  webSearchExaAPIKey: '',
};

export function createRuntimeFormState(config: BridgeConfig | null): RuntimeFormState {
  if (!config) {
    return { ...DEFAULT_RUNTIME_FORM_STATE };
  }

  return {
    provider: config.provider,
    model: config.model,
    projectRoot: config.project_root,
    maxTurns: String(config.max_turns),
    taskExecutionTimeoutMS: String(config.task_execution_timeout_ms),
    llmCompletionRetryCount: String(config.llm_completion_retry_count),
    llmCompletionRetryIntervalMS: String(config.llm_completion_retry_interval_ms),
    sessionHumanLogFullEnabled: config.session_human_log_full_enabled,
    sessionSystemPromptVisibleEnabled: config.session_system_prompt_visible_enabled,
    assistantMarkdownEnabled: config.assistant_markdown_enabled,
    memoryModeEnabled: config.memory_mode_enabled,
    microcompactEnabled: config.microcompact_enabled,
    sessionTitleMode: config.session_title_mode,
    webSearchTavilyURL: config.web_search_tavily_url,
    webSearchExaURL: config.web_search_exa_url,
    webSearchTavilyAPIKey: '',
    webSearchExaAPIKey: '',
  };
}

export function buildSecretPlaceholder(
  secretSet: boolean | undefined,
  secretName: string,
  locale: WebLocale,
): string {
  if (secretSet) {
    return locale === 'zh-CN'
      ? `留空以保留已保存的${secretName}`
      : `Leave blank to keep saved ${secretName}`;
  }

  return locale === 'zh-CN' ? `输入${secretName}` : `Enter ${secretName}`;
}

export function buildRuntimeUpdate(
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
): ConfigUpdate {
  const update = buildRuntimeScalarUpdate(modelSelectionEnabled, formState);
  applySecretUpdate(update, formState);
  return update;
}

function buildRuntimeScalarUpdate(
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
): ConfigUpdate {
  const update: ConfigUpdate = {
    provider: formState.provider,
    project_root: formState.projectRoot,
    max_turns: parsePositiveInteger(formState.maxTurns, 'max_turns'),
    task_execution_timeout_ms: parsePositiveInteger(
      formState.taskExecutionTimeoutMS,
      'task_execution_timeout_ms',
    ),
    llm_completion_retry_count: parseNonNegativeInteger(
      formState.llmCompletionRetryCount,
      'llm_completion_retry_count',
    ),
    llm_completion_retry_interval_ms: parseNonNegativeInteger(
      formState.llmCompletionRetryIntervalMS,
      'llm_completion_retry_interval_ms',
    ),
    session_human_log_full_enabled: formState.sessionHumanLogFullEnabled,
    session_system_prompt_visible_enabled: formState.sessionSystemPromptVisibleEnabled,
    assistant_markdown_enabled: formState.assistantMarkdownEnabled,
    memory_mode_enabled: formState.memoryModeEnabled,
    microcompact_enabled: formState.microcompactEnabled,
    session_title_mode: formState.sessionTitleMode,
    web_search_tavily_url: formState.webSearchTavilyURL,
    web_search_exa_url: formState.webSearchExaURL,
  };

  if (modelSelectionEnabled) {
    update.model = formState.model;
  }

  return update;
}

function applySecretUpdate(update: ConfigUpdate, formState: RuntimeFormState) {
  if (formState.webSearchTavilyAPIKey.trim()) {
    update.web_search_tavily_api_key = formState.webSearchTavilyAPIKey;
  }
  if (formState.webSearchExaAPIKey.trim()) {
    update.web_search_exa_api_key = formState.webSearchExaAPIKey;
  }
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
