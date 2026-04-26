import type { BridgeConfig } from '@/lib/types';
import {
  buildSecretPlaceholder,
  type RuntimeFormState,
} from '@/components/config/runtimeSettingsForm';
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
        label={copy.settings.runtimeProviderApiKeyLabel}
        description={copy.settings.runtimeProviderApiKeyDescription}
        value={formState.apiKey}
        disabled={controlsDisabled}
        type="password"
        placeholder={buildSecretPlaceholder(config?.api_key_set, copy.settings.runtimeProviderApiKeyLabel, locale)}
        onChange={(value) => onChange({ apiKey: value })}
      />
      <TextField
        label={copy.settings.runtimeBaseURLLabel}
        description={copy.settings.runtimeBaseURLDescription}
        value={formState.baseURL}
        disabled={controlsDisabled}
        mono
        onChange={(value) => onChange({ baseURL: value })}
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
        onChange={(value) => onChange({ chatPath: value })}
      />
    </Card>
  );
}

export function GraphQLSection(props: SectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange } = props;
  const isZh = locale === 'zh-CN';

  return (
    <Card title={copy.settings.runtimeGraphQLTitle} copy={copy.settings.runtimeGraphQLCopy}>
      <ToggleField
        label={isZh ? '启用 GraphQL 工具运行时' : 'GraphQL Tool Runtime Enabled'}
        description={isZh
          ? '启用后，Assistant 文本工具调用切换为 GraphQL 文档协议。'
          : 'When enabled, assistant text tool-calls switch to the GraphQL document protocol.'}
        checked={formState.graphqlToolRuntimeEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ graphqlToolRuntimeEnabled: checked })}
      />
      <ToggleField
        label={isZh ? '启用 GraphQL 文本清洗' : 'GraphQL Text Sanitize Enabled'}
        description={isZh
          ? '仅在 GraphQL 工具运行时生效，用于清理模型输出中的已知格式噪声。'
          : 'Only applies in GraphQL tool runtime mode; cleans known formatting artifacts from model output.'}
        checked={formState.graphqlTextSanitizeEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ graphqlTextSanitizeEnabled: checked })}
      />
    </Card>
  );
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
        label={isZh ? '记忆模式' : 'Memory Mode'}
        description={isZh
          ? '启用后在系统提示词注入 Memory 段，并自动确保当天记忆文档存在。'
          : 'When enabled, injects a Memory section into the system prompt and ensures today\'s memory file exists.'}
        checked={formState.memoryModeEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ memoryModeEnabled: checked })}
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

export function WebRooterSection(props: ConfigSectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange, config } = props;
  const isZh = locale === 'zh-CN';

  return (
    <Card title={copy.settings.runtimeWebRooterTitle} copy={copy.settings.runtimeWebRooterCopy}>
      <ToggleField
        label={isZh ? '启用 Web Rooter' : 'Web Rooter Enabled'}
        description={isZh ? '启用 web_rooter 工具集成。' : 'Enable web_rooter tool integration.'}
        checked={formState.webRooterEnabled}
        disabled={controlsDisabled}
        onChange={(checked) => onChange({ webRooterEnabled: checked })}
      />
      <TextField
        label={isZh ? 'Web Rooter Base URL' : 'Web Rooter Base URL'}
        description={isZh ? 'web_rooter 服务基础 URL。' : 'Base URL for the web_rooter service.'}
        value={formState.webRooterBaseURL}
        disabled={controlsDisabled}
        mono
        placeholder="http://127.0.0.1:8765"
        onChange={(value) => onChange({ webRooterBaseURL: value })}
      />
      <TextField
        label={isZh ? 'Web Rooter API Token' : 'Web Rooter API Token'}
        description={isZh ? '留空会保留当前已保存 token。' : 'Blank keeps currently saved token.'}
        value={formState.webRooterAPIToken}
        disabled={controlsDisabled}
        type="password"
        placeholder={buildSecretPlaceholder(config?.web_rooter_api_token_set, isZh ? 'Web Rooter API token' : 'Web Rooter API token', locale)}
        onChange={(value) => onChange({ webRooterAPIToken: value })}
      />
      <TextField
        label={isZh ? 'Web Rooter Timeout（毫秒）' : 'Web Rooter Timeout (ms)'}
        description={isZh ? '正整数超时，单位毫秒。' : 'Positive timeout in milliseconds.'}
        value={formState.webRooterTimeoutMS}
        disabled={controlsDisabled}
        type="number"
        placeholder="90000"
        onChange={(value) => onChange({ webRooterTimeoutMS: value })}
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
