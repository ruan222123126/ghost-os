import type { WebLocale } from '@/lib/i18n/locale';
import type { BridgeConfig } from '@/lib/types';
import { buildSecretPlaceholder, type RuntimeFormState } from '@/components/config/runtimeSettingsForm';

export type RuntimeSettingsLocale = WebLocale;
export type RuntimeBooleanField = {
  [Key in keyof RuntimeFormState]: RuntimeFormState[Key] extends boolean ? Key : never;
}[keyof RuntimeFormState];
export type RuntimeStringField = {
  [Key in keyof RuntimeFormState]: RuntimeFormState[Key] extends string ? Key : never;
}[keyof RuntimeFormState];

interface CommonSettingsCopy {
  runtimeMaxTurnsLabel: string;
  runtimeMaxTurnsDescription: string;
  runtimeTaskExecutionTimeoutMSLabel: string;
  runtimeTaskExecutionTimeoutMSDescription: string;
  runtimeLLMCompletionRetryCountLabel: string;
  runtimeLLMCompletionRetryCountDescription: string;
  runtimeLLMCompletionRetryIntervalMSLabel: string;
  runtimeLLMCompletionRetryIntervalMSDescription: string;
}

export interface SessionToggleSpec {
  field: RuntimeBooleanField;
  label: string;
  description: string;
}

interface LocalizedText {
  zh: string;
  en: string;
}

interface SessionToggleTextSpec {
  field: RuntimeBooleanField;
  label: LocalizedText;
  description: LocalizedText;
}

export interface RuntimeTextFieldSpec {
  field: RuntimeStringField;
  label: string;
  description: string;
  placeholder?: string;
  type?: 'text' | 'password' | 'number';
  mono?: boolean;
}

const SESSION_TOGGLE_TEXTS = [
  {
    field: 'assistantMarkdownEnabled',
    label: { zh: 'Assistant Markdown 渲染', en: 'Assistant Markdown Rendering' },
    description: {
      zh: '关闭后，assistant 文本始终按纯文本显示，不进行 Markdown 解析。',
      en: 'When disabled, assistant messages are always shown as plain text without Markdown parsing.',
    },
  },
  {
    field: 'sessionSystemPromptVisibleEnabled',
    label: { zh: '显示系统提示词', en: 'Show System Prompt' },
    description: {
      zh: '关闭后，会话首条 system 提示词不再显示在消息区，但仍保留在真实会话历史里。',
      en: 'When disabled, the first system prompt is hidden from the message list but still kept in session history.',
    },
  },
  {
    field: 'toolCallCompactOutputEnabled',
    label: { zh: '精简工具调用输出', en: 'Compact Tool Call Output' },
    description: {
      zh: '启用后，工具详情仅显示 step 序列与失败 error 行。',
      en: 'When enabled, tool details only show ordered steps plus a failure error line.',
    },
  },
  {
    field: 'memoryModeEnabled',
    label: { zh: '记忆模式', en: 'Memory Mode' },
    description: {
      zh: '启用后仅确保当天记忆文档存在，不再向系统提示词注入 Memory 段。',
      en: 'When enabled, only ensures today\'s memory file exists and no longer injects a Memory section into the system prompt.',
    },
  },
  {
    field: 'microcompactEnabled',
    label: { zh: 'Microcompact 请求压缩', en: 'Microcompact Request Compression' },
    description: {
      zh: '启用后，请求前会压缩较旧的高膨胀工具结果视图，但不会改写会话持久化历史。',
      en: 'When enabled, older high-expansion tool-result spans are compacted before requests without rewriting persisted session history.',
    },
  },
  {
    field: 'sessionHumanLogFullEnabled',
    label: { zh: '会话人类日志（完整工具输出）', en: 'Session Human Log (Full Tool Output)' },
    description: {
      zh: '启用后，工具消息将以完整 output/error 写入会话 markdown 日志。',
      en: 'When enabled, tool messages are written with full output/error in session markdown logs.',
    },
  },
] as const satisfies readonly SessionToggleTextSpec[];

export function buildCommonNumberFieldSpecs(copy: CommonSettingsCopy): RuntimeTextFieldSpec[] {
  return [
    {
      field: 'maxTurns',
      label: copy.runtimeMaxTurnsLabel,
      description: copy.runtimeMaxTurnsDescription,
    },
    {
      field: 'taskExecutionTimeoutMS',
      label: copy.runtimeTaskExecutionTimeoutMSLabel,
      description: copy.runtimeTaskExecutionTimeoutMSDescription,
    },
    {
      field: 'llmCompletionRetryCount',
      label: copy.runtimeLLMCompletionRetryCountLabel,
      description: copy.runtimeLLMCompletionRetryCountDescription,
    },
    {
      field: 'llmCompletionRetryIntervalMS',
      label: copy.runtimeLLMCompletionRetryIntervalMSLabel,
      description: copy.runtimeLLMCompletionRetryIntervalMSDescription,
    },
  ];
}

export function buildSessionToggleSpecs(isZh: boolean): SessionToggleSpec[] {
  return SESSION_TOGGLE_TEXTS.map((spec) => ({
    field: spec.field,
    label: localizeText(isZh, spec.label),
    description: localizeText(isZh, spec.description),
  }));
}

export function buildWebSearchFieldSpecs(options: {
  config: BridgeConfig | null;
  isZh: boolean;
  locale: RuntimeSettingsLocale;
}): RuntimeTextFieldSpec[] {
  const { config, isZh, locale } = options;
  return [
    {
      field: 'webSearchTavilyURL',
      label: localize(isZh, 'Tavily 自定义 URL', 'Tavily Custom URL'),
      description: localize(isZh, '可选 Tavily 端点覆盖。', 'Optional Tavily endpoint override.'),
      placeholder: localize(isZh, '留空使用 Tavily 官方端点', 'Leave blank to use the official Tavily endpoint'),
      mono: true,
    },
    {
      field: 'webSearchTavilyAPIKey',
      label: 'Tavily API Key',
      description: localize(isZh, '留空会保留当前已保存 key。', 'Blank keeps currently saved key.'),
      placeholder: buildSecretPlaceholder(config?.web_search_tavily_api_key_set, 'Tavily key', locale),
      type: 'password',
    },
    {
      field: 'webSearchExaURL',
      label: localize(isZh, 'Exa 自定义 URL', 'Exa Custom URL'),
      description: localize(isZh, '可选 Exa 端点覆盖。', 'Optional Exa endpoint override.'),
      placeholder: localize(isZh, '留空使用 Exa 官方端点', 'Leave blank to use the official Exa endpoint'),
      mono: true,
    },
    {
      field: 'webSearchExaAPIKey',
      label: 'Exa API Key',
      description: localize(isZh, '留空会保留当前已保存 key。', 'Blank keeps currently saved key.'),
      placeholder: buildSecretPlaceholder(config?.web_search_exa_api_key_set, 'Exa key', locale),
      type: 'password',
    },
  ];
}

export function booleanFieldPatch(field: RuntimeBooleanField, value: boolean): Partial<RuntimeFormState> {
  return { [field]: value };
}

export function stringFieldPatch(field: RuntimeStringField, value: string): Partial<RuntimeFormState> {
  return { [field]: value };
}

export function localize(isZh: boolean, zh: string, en: string): string {
  return isZh ? zh : en;
}

function localizeText(isZh: boolean, text: LocalizedText): string {
  return localize(isZh, text.zh, text.en);
}
