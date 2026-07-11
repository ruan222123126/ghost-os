import type { BridgeConfig } from '@/lib/types';
import type { RuntimeFormState } from '@/components/config/runtimeSettingsForm';
import {
  Card,
  SelectField,
  TextField,
  ToggleField,
} from '@/components/config/runtimeSettingsFieldComponents';
import {
  booleanFieldPatch,
  buildCommonNumberFieldSpecs,
  buildSessionToggleSpecs,
  buildWebSearchFieldSpecs,
  localize,
  stringFieldPatch,
  type RuntimeSettingsLocale,
} from '@/components/config/runtimeSettingsSectionSpecs';
import { useWebLocale } from '@/lib/i18n/provider';

interface SectionProps {
  formState: RuntimeFormState;
  controlsDisabled: boolean;
  onChange: (patch: Partial<RuntimeFormState>) => void;
}

interface ConfigSectionProps extends SectionProps {
  config: BridgeConfig | null;
}

export function CommonSettingsSection(props: SectionProps & {
  locale: RuntimeSettingsLocale;
  onLocaleChange: (value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { formState, controlsDisabled, locale, onChange, onLocaleChange } = props;
  const languageOptions = [
    { value: 'zh-CN', label: copy.settings.languageOptionZh },
    { value: 'en-US', label: copy.settings.languageOptionEn },
  ] as const;
  const numberFields = buildCommonNumberFieldSpecs(copy.settings);

  return (
    <Card title={copy.settings.runtimeCommonTitle} copy={copy.settings.runtimeCommonCopy}>
      <SelectField
        label={copy.settings.languageLabel}
        description={copy.settings.languageDescription}
        value={locale}
        onChange={onLocaleChange}
        options={languageOptions}
      />
      {numberFields.map((field) => (
        <TextField
          key={field.field}
          label={field.label}
          description={field.description}
          value={formState[field.field]}
          disabled={controlsDisabled}
          type="number"
          onChange={(value) => onChange(stringFieldPatch(field.field, value))}
        />
      ))}
    </Card>
  );
}

export function ConversationSettingsSection(props: SectionProps & { modelSelectionEnabled: boolean }) {
  const { copy } = useWebLocale();
  const { formState, controlsDisabled, modelSelectionEnabled, onChange } = props;

  return (
    <Card title={copy.settings.conversationConfigTitle} copy={copy.settings.conversationConfigCopy}>
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
        label={copy.settings.runtimeProjectRootLabel}
        description={copy.settings.runtimeProjectRootDescription}
        value={formState.projectRoot}
        disabled={controlsDisabled}
        mono
        placeholder={copy.settings.runtimeProjectRootPlaceholder}
        onChange={(value) => onChange({ projectRoot: value })}
      />
    </Card>
  );
}

export function SessionSection(props: SectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange } = props;
  const isZh = locale === 'zh-CN';
  const sessionToggleSpecs = buildSessionToggleSpecs(isZh);
  const titleModeOptions = [
    { value: 'session_id', label: localize(isZh, '会话 ID', 'Session ID') },
    { value: 'first_message', label: localize(isZh, '首条消息', 'First Message') },
    { value: 'ai_generated', label: localize(isZh, 'AI 生成', 'AI Generated') },
  ] as const;

  return (
    <Card title={copy.settings.runtimeSessionTitle} copy={copy.settings.runtimeSessionCopy}>
      <SelectField
        label={localize(isZh, '会话标题策略', 'Session Title Strategy')}
        description={localize(
          isZh,
          '只影响新建会话。AI 生成会在后台运行，聊天不会等待标题。',
          'Only affects new sessions. AI-generated titles run in the background without blocking chat.',
        )}
        value={formState.sessionTitleMode}
        disabled={controlsDisabled}
        options={titleModeOptions}
        onChange={(value) => onChange({ sessionTitleMode: value as RuntimeFormState['sessionTitleMode'] })}
      />
      {sessionToggleSpecs.map((toggle) => (
        <ToggleField
          key={toggle.field}
          label={toggle.label}
          description={toggle.description}
          checked={formState[toggle.field]}
          disabled={controlsDisabled}
          onChange={(checked) => onChange(booleanFieldPatch(toggle.field, checked))}
        />
      ))}
    </Card>
  );
}

export function WebSearchSection(props: ConfigSectionProps) {
  const { locale, copy } = useWebLocale();
  const { formState, controlsDisabled, onChange, config } = props;
  const isZh = locale === 'zh-CN';
  const fields = buildWebSearchFieldSpecs({ config, isZh, locale });

  return (
    <Card title={copy.settings.runtimeWebSearchTitle} copy={copy.settings.runtimeWebSearchCopy}>
      {fields.map((field) => (
        <TextField
          key={field.field}
          label={field.label}
          description={field.description}
          value={formState[field.field]}
          disabled={controlsDisabled}
          type={field.type}
          mono={field.mono}
          placeholder={field.placeholder}
          onChange={(value) => onChange(stringFieldPatch(field.field, value))}
        />
      ))}
    </Card>
  );
}
