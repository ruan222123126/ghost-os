'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { isWebLocale } from '@/lib/i18n/locale';
import { useWebLocale } from '@/lib/i18n/provider';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';
import {
  buildRuntimeUpdate,
  createRuntimeFormState,
  type RuntimeFormState,
} from '@/components/config/runtimeSettingsForm';
import {
  Card,
  SelectField,
} from '@/components/config/runtimeSettingsFieldComponents';
import {
  RuntimeCoreSection,
  SessionSection,
  WebSearchSection,
} from '@/components/config/runtimeSettingsSections';

const AUTO_SAVE_DEBOUNCE_MS = 500;

type LocaleValue = ReturnType<typeof useWebLocale>['locale'];

interface RuntimeSettingsSectionProps {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
}

interface RuntimeSettingsState {
  formState: RuntimeFormState;
  submitError: string;
  modelSelectionEnabled: boolean;
  controlsDisabled: boolean;
  updateForm: (patch: Partial<RuntimeFormState>) => void;
}

export function RuntimeSettingsSection(props: RuntimeSettingsSectionProps) {
  const { locale, setLocale, copy } = useWebLocale();
  const runtimeState = useRuntimeSettingsState({
    loading: props.loading,
    saving: props.saving,
    config: props.config,
    onSave: props.onSave,
    invalidRuntimeMessage: copy.system.invalidRuntimeSettings,
  });

  return (
    <section>
      <LanguageCard locale={locale} setLocale={setLocale} />
      <RuntimePanels config={props.config} state={runtimeState} loading={props.loading} />
    </section>
  );
}

function useRuntimeSettingsState(options: {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  invalidRuntimeMessage: string;
}): RuntimeSettingsState {
  const { loading, saving, config, onSave, invalidRuntimeMessage } = options;
  const baselineFormState = useMemo(() => createRuntimeFormState(config), [config]);
  const [formState, setFormState] = useState<RuntimeFormState>(() => baselineFormState);
  const [submitError, setSubmitError] = useState('');
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;

  useEffect(() => {
    setFormState(baselineFormState);
    setSubmitError('');
  }, [baselineFormState]);

  const baselineSignature = useMemo(() => runtimeFormSignature(baselineFormState), [baselineFormState]);
  const formSignature = useMemo(() => runtimeFormSignature(formState), [formState]);

  useRuntimeAutoSave({
    loading,
    saving,
    config,
    formState,
    formSignature,
    baselineSignature,
    modelSelectionEnabled,
    onSave,
    setSubmitError,
    invalidRuntimeMessage,
  });

  return {
    formState,
    submitError,
    modelSelectionEnabled,
    controlsDisabled: loading,
    updateForm: (patch) => setFormState((prev) => ({ ...prev, ...patch })),
  };
}

function useRuntimeAutoSave(options: {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  formState: RuntimeFormState;
  formSignature: string;
  baselineSignature: string;
  modelSelectionEnabled: boolean;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  setSubmitError: (message: string) => void;
  invalidRuntimeMessage: string;
}) {
  const { loading, saving, config, formState, formSignature, baselineSignature, modelSelectionEnabled, onSave, setSubmitError, invalidRuntimeMessage } = options;
  const lastAttemptedFormSignatureRef = useRef('');

  useEffect(() => {
    if (loading || saving || config === null) {
      return;
    }
    if (formSignature === baselineSignature) {
      lastAttemptedFormSignatureRef.current = '';
      return;
    }
    if (formSignature === lastAttemptedFormSignatureRef.current) {
      return;
    }

    let update: ConfigUpdate;
    try {
      update = buildRuntimeUpdate(modelSelectionEnabled, formState);
      setSubmitError('');
    } catch (error) {
      setSubmitError(toErrorMessage(error, invalidRuntimeMessage));
      return;
    }

    const timer = window.setTimeout(() => {
      lastAttemptedFormSignatureRef.current = formSignature;
      ignorePromise(onSave(update));
    }, AUTO_SAVE_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [baselineSignature, config, formSignature, formState, invalidRuntimeMessage, loading, modelSelectionEnabled, onSave, saving, setSubmitError]);
}

function LanguageCard(props: {
  locale: LocaleValue;
  setLocale: ReturnType<typeof useWebLocale>['setLocale'];
}) {
  const { copy } = useWebLocale();
  const { locale, setLocale } = props;
  const options = [
    { value: 'zh-CN', label: copy.settings.languageOptionZh },
    { value: 'en-US', label: copy.settings.languageOptionEn },
  ] as const;

  return (
    <Card title={copy.settings.languageTitle} copy={copy.settings.languageDescription}>
      <SelectField
        label={copy.settings.languageLabel}
        value={locale}
        onChange={(value) => handleLocaleChange(value, setLocale)}
        options={options}
      />
    </Card>
  );
}

function handleLocaleChange(value: string, setLocale: ReturnType<typeof useWebLocale>['setLocale']) {
  if (isWebLocale(value)) {
    setLocale(value);
  }
}

function RuntimePanels(props: { config: BridgeConfig | null; state: RuntimeSettingsState; loading: boolean }) {
  const { copy } = useWebLocale();
  const { config, state, loading } = props;

  return (
    <div className="space-y-0">
      {loading ? <InlineNotice text={copy.settings.loadingRuntimeConfig} /> : null}
      {state.submitError ? <InlineError text={state.submitError} /> : null}
      <RuntimeCoreSection
        formState={state.formState}
        controlsDisabled={state.controlsDisabled}
        modelSelectionEnabled={state.modelSelectionEnabled}
        onChange={state.updateForm}
        config={config}
      />
      <SessionSection formState={state.formState} controlsDisabled={state.controlsDisabled} onChange={state.updateForm} />
      <WebSearchSection
        formState={state.formState}
        controlsDisabled={state.controlsDisabled}
        onChange={state.updateForm}
        config={config}
      />
    </div>
  );
}

function runtimeFormSignature(formState: RuntimeFormState): string {
  return JSON.stringify(formState);
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
