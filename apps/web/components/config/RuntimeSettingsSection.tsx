'use client';

import type { FormEvent } from 'react';
import { useEffect, useState } from 'react';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useSessionSidebarGroupingPreference } from '@/hooks/useSessionSidebarGroupingPreference';
import { isWebLocale } from '@/lib/i18n/locale';
import { useWebLocale } from '@/lib/i18n/provider';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';
import {
  buildRuntimeUpdate,
  createRuntimeFormState,
  type RuntimeFormState,
} from '@/components/config/runtimeSettingsForm';
import {
  GraphQLSection,
  RuntimeCoreSection,
  SessionSection,
  WebRooterSection,
  WebSearchSection,
} from '@/components/config/runtimeSettingsSections';

interface RuntimeSettingsSectionProps {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onReload: () => Promise<void>;
}

export function RuntimeSettingsSection(props: RuntimeSettingsSectionProps) {
  const { locale, setLocale, copy } = useWebLocale();
  const groupingPreference = useSessionSidebarGroupingPreference();
  const { loading, saving, config, onSave, onReload } = props;
  const [formState, setFormState] = useState<RuntimeFormState>(() => createRuntimeFormState(null));
  const [submitError, setSubmitError] = useState('');
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;
  const controlsDisabled = loading || saving;

  useEffect(() => {
    setFormState(createRuntimeFormState(config));
    setSubmitError('');
  }, [config]);

  const updateForm = (patch: Partial<RuntimeFormState>) => {
    setFormState((prev) => ({ ...prev, ...patch }));
  };

  return (
    <section>
      <header className="mb-10">
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.generalTitle}</h1>
        <p className="text-[14px] text-[#737373]">
          {copy.settings.generalDescription}
        </p>
      </header>

      <section className="mb-8 rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <div className="mb-3">
          <h2 className="text-[16px] font-semibold text-[#111111]">{copy.settings.languageTitle}</h2>
          <p className="mt-1 text-[13px] text-[#737373]">{copy.settings.languageDescription}</p>
        </div>
        <label className="block">
          <span className="mb-2 block text-[13px] font-medium text-[#111111]">{copy.settings.languageLabel}</span>
          <select
            value={locale}
            onChange={(event) => {
              const nextLocale = event.target.value;
              if (isWebLocale(nextLocale)) {
                setLocale(nextLocale);
              }
            }}
            className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
          >
            <option value="zh-CN">{copy.settings.languageOptionZh}</option>
            <option value="en-US">{copy.settings.languageOptionEn}</option>
          </select>
        </label>
      </section>

      <SidebarGroupingPreferenceCard
        title={copy.settings.sidebarGroupingTitle}
        description={copy.settings.sidebarGroupingDescription}
        label={copy.settings.sidebarGroupingLabel}
        enabled={groupingPreference.enabled}
        onChange={groupingPreference.setEnabled}
      />

      <form
        className="space-y-0"
        onSubmit={(event) =>
          handleRuntimeSubmit(
            event,
            modelSelectionEnabled,
            formState,
            locale,
            onSave,
            setSubmitError,
            copy.system.invalidRuntimeSettings,
          )}
      >
        {loading ? <InlineNotice text={copy.settings.loadingRuntimeConfig} /> : null}
        {submitError ? <InlineError text={submitError} /> : null}

        <RuntimeCoreSection
          formState={formState}
          controlsDisabled={controlsDisabled}
          modelSelectionEnabled={modelSelectionEnabled}
          onChange={updateForm}
          config={config}
        />

        <GraphQLSection
          formState={formState}
          controlsDisabled={controlsDisabled}
          onChange={updateForm}
        />

        <SessionSection
          formState={formState}
          controlsDisabled={controlsDisabled}
          onChange={updateForm}
        />

        <WebRooterSection
          formState={formState}
          controlsDisabled={controlsDisabled}
          onChange={updateForm}
          config={config}
        />

        <WebSearchSection
          formState={formState}
          controlsDisabled={controlsDisabled}
          onChange={updateForm}
          config={config}
        />

        <div className="flex items-center justify-end gap-3 pt-8">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onReload());
            }}
            className="rounded-full border border-[#E5E5E5] px-6 py-2.5 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.reload}
          </button>
          <button
            type="submit"
            disabled={controlsDisabled}
            className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? copy.settings.saving : copy.settings.save}
          </button>
        </div>
      </form>
    </section>
  );
}

function SidebarGroupingPreferenceCard(props: {
  title: string;
  description: string;
  label: string;
  enabled: boolean;
  onChange: (enabled: boolean) => void;
}) {
  const { copy } = useWebLocale();
  const { title, description, label, enabled, onChange } = props;

  return (
    <section className="mb-8 rounded-[16px] border border-[#E5E5E5] bg-white p-6">
      <div className="mb-3">
        <h2 className="text-[16px] font-semibold text-[#111111]">{title}</h2>
        <p className="mt-1 text-[13px] text-[#737373]">{description}</p>
      </div>
      <label className="inline-flex items-center gap-2 text-[13px] text-[#111111]">
        <input
          type="checkbox"
          checked={enabled}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span>{label}: {enabled ? copy.settings.runtimeToggleEnabled : copy.settings.runtimeToggleDisabled}</span>
      </label>
    </section>
  );
}

function InlineNotice(props: { text: string }) {
  return (
    <div className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">
      {props.text}
    </div>
  );
}

function InlineError(props: { text: string }) {
  return (
    <div className="rounded-[12px] border border-red-200 bg-red-50 px-4 py-3 text-[13px] text-red-700">
      {props.text}
    </div>
  );
}

function handleRuntimeSubmit(
  event: FormEvent<HTMLFormElement>,
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
  locale: ReturnType<typeof useWebLocale>['locale'],
  onSave: (update: ConfigUpdate) => Promise<boolean>,
  setSubmitError: (message: string) => void,
  fallbackMessage: string,
) {
  event.preventDefault();
  setSubmitError('');

  let update: ConfigUpdate;
  try {
    update = buildRuntimeUpdate(modelSelectionEnabled, formState, locale);
  } catch (error) {
    setSubmitError(toErrorMessage(error, fallbackMessage));
    return;
  }

  ignorePromise(onSave(update));
}
