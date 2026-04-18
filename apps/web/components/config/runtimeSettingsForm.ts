import type {
  BridgeConfig,
  ConfigUpdate,
} from '@/lib/types';
import type { WebLocale } from '@/lib/i18n/locale';

export interface RuntimeFormState {
  provider: string;
  apiKey: string;
  baseURL: string;
  model: string;
  chatPath: string;
  graphqlToolRuntimeEnabled: boolean;
  graphqlTextSanitizeEnabled: boolean;
  sessionHumanLogFullEnabled: boolean;
  webRooterEnabled: boolean;
  webRooterBaseURL: string;
  webRooterAPIToken: string;
  webRooterTimeoutMS: string;
  webSearchTavilyURL: string;
  webSearchExaURL: string;
  webSearchTavilyAPIKey: string;
  webSearchExaAPIKey: string;
}

export function createRuntimeFormState(config: BridgeConfig | null): RuntimeFormState {
  return {
    provider: config?.provider ?? '',
    apiKey: '',
    baseURL: config?.base_url ?? '',
    model: config?.model ?? '',
    chatPath: config?.chat_path ?? '',
    graphqlToolRuntimeEnabled: config?.graphql_tool_runtime_enabled ?? true,
    graphqlTextSanitizeEnabled: config?.graphql_text_sanitize_enabled ?? true,
    sessionHumanLogFullEnabled: config?.session_human_log_full_enabled ?? false,
    webRooterEnabled: config?.web_rooter_enabled ?? false,
    webRooterBaseURL: config?.web_rooter_base_url ?? '',
    webRooterAPIToken: '',
    webRooterTimeoutMS: config ? String(config.web_rooter_timeout_ms) : '',
    webSearchTavilyURL: config?.web_search_tavily_url ?? '',
    webSearchExaURL: config?.web_search_exa_url ?? '',
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
  locale: WebLocale,
): ConfigUpdate {
  const update = buildRuntimeScalarUpdate(modelSelectionEnabled, formState);
  applySecretUpdate(update, formState);
  applyWebRooterUpdate(update, formState, locale);
  return update;
}

function buildRuntimeScalarUpdate(
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
): ConfigUpdate {
  const update: ConfigUpdate = {
    provider: formState.provider,
    base_url: formState.baseURL,
    chat_path: formState.chatPath,
    graphql_tool_runtime_enabled: formState.graphqlToolRuntimeEnabled,
    graphql_text_sanitize_enabled: formState.graphqlTextSanitizeEnabled,
    session_human_log_full_enabled: formState.sessionHumanLogFullEnabled,
    web_rooter_enabled: formState.webRooterEnabled,
    web_search_tavily_url: formState.webSearchTavilyURL,
    web_search_exa_url: formState.webSearchExaURL,
  };

  if (modelSelectionEnabled) {
    update.model = formState.model;
  }

  return update;
}

function applySecretUpdate(update: ConfigUpdate, formState: RuntimeFormState) {
  if (formState.apiKey.trim()) {
    update.api_key = formState.apiKey;
  }
  if (formState.webRooterAPIToken.trim()) {
    update.web_rooter_api_token = formState.webRooterAPIToken;
  }
  if (formState.webSearchTavilyAPIKey.trim()) {
    update.web_search_tavily_api_key = formState.webSearchTavilyAPIKey;
  }
  if (formState.webSearchExaAPIKey.trim()) {
    update.web_search_exa_api_key = formState.webSearchExaAPIKey;
  }
}

function applyWebRooterUpdate(update: ConfigUpdate, formState: RuntimeFormState, locale: WebLocale) {
  const baseURL = formState.webRooterBaseURL.trim();
  if (baseURL) {
    update.web_rooter_base_url = baseURL;
  }

  const timeoutValue = formState.webRooterTimeoutMS.trim();
  if (timeoutValue) {
    update.web_rooter_timeout_ms = parsePositiveInteger(
      timeoutValue,
      locale === 'zh-CN' ? 'Web Rooter 超时（毫秒）' : 'Web Rooter Timeout (ms)',
      locale,
    );
  }
}

function parsePositiveInteger(value: string, label: string, locale: WebLocale): number {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isInteger(parsed) || parsed < 1) {
    throw new Error(locale === 'zh-CN' ? `${label} 必须为正整数。` : `${label} must be a positive integer.`);
  }

  return parsed;
}
