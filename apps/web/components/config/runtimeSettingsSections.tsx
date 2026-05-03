import type { BridgeConfig } from '@/lib/types';
import { buildSecretPlaceholder, type RuntimeFormState } from '@/components/config/runtimeSettingsForm';
import {
  Card,
  TextField,
  ToggleField,
} from '@/components/config/runtimeSettingsFieldComponents';
import { useWebLocale } from '@/lib/i18n/provider';

interface SectionProps {
  formState: RuntimeFormState;
  controlsDisabled: boolean;
  onChange: (patch: Partial<RuntimeFormState>) => void;
}

interface ConfigSectionProps extends SectionProps {
  config: BridgeConfig | null;
}

export function RuntimeCoreSection(props: ConfigSectionProps & { modelSelectionEnabled: boolean }) {
  const { copy, locale } = useWebLocale();
  const { formState, controlsDisabled, modelSelectionEnabled, onChange, config } = props;
  const chatPathPlaceholder = runtimeChatPathPlaceholder(config, locale);

  return (
    <Card title={copy.settings.runtimeCoreTitle} copy={copy.settings.runtimeCoreCopy}>
      <TextField
        label={copy.settings.runtimeProviderNameLabel}
        description={copy.settings.runtimeProviderNameDescription}
        value={formState.provider}
        disabled={controlsDisabled}
        mono
        onChange={(value) => onChange({ provider: value })}
      />
      <TextField
        label={copy.settings.runtimeModelLabel}
        description={modelSelectionEnabled
          ? copy.settings.runtimeModelEnabledDescription
          : copy.settings.runtimeModelDisabledDescription}
        value={formState.model}
        disabled={controlsDisabled || !modelSelectionEnabled}
        mono
        onChange={(value) => onChange({ model: value })}
      />
      <TextField
        label={copy.settings.runtimeChatPathLabel}
        description={copy.settings.runtimeChatPathDescription}
        value={formState.chatPath}
        disabled={controlsDisabled}
        mono
        placeholder={chatPathPlaceholder}
        onChange={(value) => onChange({ chatPath: value })}
      />
      <TextField
        label={copy.settings.runtimeProjectRootLabel}
        description={copy.settings.runtimeProjectRootDescription}
        value={formState.projectRoot}
        disabled={controlsDisabled}
        mono
        placeholder={copy.settings.runtimeProjectRootPlaceholder}
        onChange={(value) => onChange({ projectRoot: value })}
      />
      <TextField
        label={copy.settings.runtimeMaxTurnsLabel}
        description={copy.settings.runtimeMaxTurnsDescription}
        value={formState.maxTurns}
        disabled={controlsDisabled}
        type="number"
        onChange={(value) => onChange({ maxTurns: value })}
      />
      <TextField
        label={copy.settings.runtimeLLMCompletionRetryCountLabel}
        description={copy.settings.runtimeLLMCompletionRetryCountDescription}
        value={formState.llmCompletionRetryCount}
        disabled={controlsDisabled}
        type="number"
        onChange={(value) => onChange({ llmCompletionRetryCount: value })}
      />
      <TextField
        label={copy.settings.runtimeLLMCompletionRetryIntervalMSLabel}
        description={copy.settings.runtimeLLMCompletionRetryIntervalMSDescription}
        value={formState.llmCompletionRetryIntervalMS}
        disabled={controlsDisabled}
        type="number"
        onChange={(value) => onChange({ llmCompletionRetryIntervalMS: value })}
      />
    </Card>
  );
}

function runtimeChatPathPlaceholder(config: BridgeConfig | null, locale: string): string {
  const defaultPath = defaultChatPath(config?.provider_type);
  if (defaultPath === '') {
    return '';
  }
  if (locale === 'zh-CN') {
    return `留空使用默认: ${defaultPath}`;
  }
  return `Leave blank to use default: ${defaultPath}`;
}

function defaultChatPath(providerType: BridgeConfig['provider_type'] | undefined): string {
  switch (providerType) {
    case 'anthropic':
      return '/v1/messages';
    case 'codex':
      return '/responses';
    case 'openai':
    case 'custom':
      return '/chat/completions';
    default:
      return '';
  }
}

export function SessionSection(props: SectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange } = props;
  const isZh = locale === 'zh-CN';

  return (
    <Card title={copy.settings.runtimeSessionTitle} copy={copy.settings.runtimeSessionCopy}>
      <ToggleField
        label={isZh ? 'Assistant Markdown 渲染' : 'Assistant Markdown Rendering'}
        description={isZh
          ? '关闭后，assistant 文本始终按纯文本显示，不进行 Markdown 解析。'
          : 'When disabled, assistant messages are always shown as plain text without Markdown parsing.'}
        checked={formState.assistantMarkdownEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ assistantMarkdownEnabled: checked })}
      />
      <ToggleField
        label={isZh ? '显示系统提示词' : 'Show System Prompt'}
        description={isZh
          ? '关闭后，会话首条 system 提示词不再显示在消息区，但仍保留在真实会话历史里。'
          : 'When disabled, the first system prompt is hidden from the message list but still kept in session history.'}
        checked={formState.sessionSystemPromptVisibleEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ sessionSystemPromptVisibleEnabled: checked })}
      />
      <ToggleField
        label={isZh ? '精简工具调用输出' : 'Compact Tool Call Output'}
        description={isZh
          ? '启用后，工具详情仅显示 step 序列与失败 error 行。'
          : 'When enabled, tool details only show ordered steps plus a failure error line.'}
        checked={formState.toolCallCompactOutputEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ toolCallCompactOutputEnabled: checked })}
      />
      <ToggleField
        label={isZh ? '记忆模式' : 'Memory Mode'}
        description={isZh
          ? '启用后仅确保当天记忆文档存在，不再向系统提示词注入 Memory 段。'
          : 'When enabled, only ensures today\'s memory file exists and no longer injects a Memory section into the system prompt.'}
        checked={formState.memoryModeEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ memoryModeEnabled: checked })}
      />
      <ToggleField
        label={isZh ? 'Microcompact 请求压缩' : 'Microcompact Request Compression'}
        description={isZh
          ? '启用后，请求前会压缩较旧的高膨胀工具结果视图，但不会改写会话持久化历史。'
          : 'When enabled, older high-expansion tool-result spans are compacted before requests without rewriting persisted session history.'}
        checked={formState.microcompactEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ microcompactEnabled: checked })}
      />
      <ToggleField
        label={isZh ? '会话人类日志（完整工具输出）' : 'Session Human Log (Full Tool Output)'}
        description={isZh
          ? '启用后，工具消息将以完整 output/error 写入会话 markdown 日志。'
          : 'When enabled, tool messages are written with full output/error in session markdown logs.'}
        checked={formState.sessionHumanLogFullEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ sessionHumanLogFullEnabled: checked })}
      />
    </Card>
  );
}

export function WebSearchSection(props: ConfigSectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange, config } = props;
  const isZh = locale === 'zh-CN';

  return (
    <Card title={copy.settings.runtimeWebSearchTitle} copy={copy.settings.runtimeWebSearchCopy}>
      <TextField
        label={isZh ? 'Tavily 自定义 URL' : 'Tavily Custom URL'}
        description={isZh ? '可选 Tavily 端点覆盖。' : 'Optional Tavily endpoint override.'}
        value={formState.webSearchTavilyURL}
        disabled={controlsDisabled}
        mono
        placeholder={isZh ? '留空使用 Tavily 官方端点' : 'Leave blank to use the official Tavily endpoint'}
        onChange={(value) => onChange({ webSearchTavilyURL: value })}
      />
      <TextField
        label={isZh ? 'Tavily API Key' : 'Tavily API Key'}
        description={isZh ? '留空会保留当前已保存 key。' : 'Blank keeps currently saved key.'}
        value={formState.webSearchTavilyAPIKey}
        disabled={controlsDisabled}
        type="password"
        placeholder={buildSecretPlaceholder(config?.web_search_tavily_api_key_set, isZh ? 'Tavily key' : 'Tavily key', locale)}
        onChange={(value) => onChange({ webSearchTavilyAPIKey: value })}
      />
      <TextField
        label={isZh ? 'Exa 自定义 URL' : 'Exa Custom URL'}
        description={isZh ? '可选 Exa 端点覆盖。' : 'Optional Exa endpoint override.'}
        value={formState.webSearchExaURL}
        disabled={controlsDisabled}
        mono
        placeholder={isZh ? '留空使用 Exa 官方端点' : 'Leave blank to use the official Exa endpoint'}
        onChange={(value) => onChange({ webSearchExaURL: value })}
      />
      <TextField
        label={isZh ? 'Exa API Key' : 'Exa API Key'}
        description={isZh ? '留空会保留当前已保存 key。' : 'Blank keeps currently saved key.'}
        value={formState.webSearchExaAPIKey}
        disabled={controlsDisabled}
        type="password"
        placeholder={buildSecretPlaceholder(config?.web_search_exa_api_key_set, isZh ? 'Exa key' : 'Exa key', locale)}
        onChange={(value) => onChange({ webSearchExaAPIKey: value })}
      />
    </Card>
  );
}
