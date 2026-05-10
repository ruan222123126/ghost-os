'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Card, SelectField, TextField } from '@/components/config/runtimeSettingsFieldComponents';
import {
  buildRelaySettingsUpdate,
  createRelaySettingsFormState,
  type RelayDefaultStopPolicy,
  type RelaySettingsFormState,
} from '@/components/config/relaySettingsForm';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

const AUTO_SAVE_DEBOUNCE_MS = 500;

interface RelaySettingsAutoSaveOptions {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  formState: RelaySettingsFormState;
  baselineSignature: string;
  currentSignature: string;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  invalidSettingsMessage: string;
  setSubmitError: (message: string) => void;
}

interface RelayAutoSaveGuard {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  baselineSignature: string;
  currentSignature: string;
  lastAttemptedSignature: string;
}

interface RelaySettingsSectionProps {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
}

export function RelaySettingsSection(props: RelaySettingsSectionProps) {
  const { copy } = useWebLocale();
  const state = useRelaySettingsState({
    loading: props.loading,
    saving: props.saving,
    config: props.config,
    onSave: props.onSave,
    invalidSettingsMessage: copy.system.invalidRuntimeSettings,
  });

  return (
    <section>
      <header className="mb-10">
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.relayTitle}</h1>
        <p className="text-[14px] text-[#737373]">{copy.settings.relayDescription}</p>
      </header>
      {props.loading ? <InlineNotice text={copy.settings.loadingRuntimeConfig} /> : null}
      {state.submitError ? <InlineError text={state.submitError} /> : null}
      <RelayDefaultsPanel
        formState={state.formState}
        controlsDisabled={props.loading}
        onChange={state.updateForm}
      />
    </section>
  );
}

function useRelaySettingsState(options: {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  invalidSettingsMessage: string;
}) {
  const baselineFormState = useMemo(() => createRelaySettingsFormState(options.config), [options.config]);
  const [formState, setFormState] = useState<RelaySettingsFormState>(() => baselineFormState);
  const [submitError, setSubmitError] = useState('');
  const baselineSignature = useMemo(() => formSignature(baselineFormState), [baselineFormState]);
  const currentSignature = useMemo(() => formSignature(formState), [formState]);

  useEffect(() => {
    setFormState(baselineFormState);
    setSubmitError('');
  }, [baselineFormState]);

  useRelaySettingsAutoSave({
    ...options,
    formState,
    baselineSignature,
    currentSignature,
    setSubmitError,
  });

  return {
    formState,
    submitError,
    updateForm: (patch: Partial<RelaySettingsFormState>) => setFormState((prev) => ({ ...prev, ...patch })),
  };
}

function useRelaySettingsAutoSave(options: RelaySettingsAutoSaveOptions) {
  const {
    loading,
    saving,
    config,
    formState,
    baselineSignature,
    currentSignature,
    onSave,
    invalidSettingsMessage,
    setSubmitError,
  } = options;
  const lastAttemptedSignatureRef = useRef('');

  useEffect(() => {
    if (currentSignature === baselineSignature) {
      lastAttemptedSignatureRef.current = '';
    }
    if (shouldSkipRelayAutoSave({
      loading,
      saving,
      config,
      baselineSignature,
      currentSignature,
      lastAttemptedSignature: lastAttemptedSignatureRef.current,
    })) {
      return;
    }

    const result = buildRelayAutoSaveUpdate(formState, invalidSettingsMessage);
    setSubmitError(result.error);
    if (result.update === null) {
      return;
    }

    const timer = window.setTimeout(() => {
      lastAttemptedSignatureRef.current = currentSignature;
      ignorePromise(onSave(result.update));
    }, AUTO_SAVE_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [
    baselineSignature,
    config,
    currentSignature,
    formState,
    invalidSettingsMessage,
    loading,
    onSave,
    saving,
    setSubmitError,
  ]);
}

function shouldSkipRelayAutoSave(options: RelayAutoSaveGuard): boolean {
  if (options.loading || options.saving || options.config === null) {
    return true;
  }
  if (options.currentSignature === options.baselineSignature) {
    return true;
  }
  return options.currentSignature === options.lastAttemptedSignature;
}

function buildRelayAutoSaveUpdate(
  formState: RelaySettingsFormState,
  invalidSettingsMessage: string,
): { update: ConfigUpdate; error: string } | { update: null; error: string } {
  try {
    return { update: buildRelaySettingsUpdate(formState), error: '' };
  } catch (error) {
    return { update: null, error: toErrorMessage(error, invalidSettingsMessage) };
  }
}

function RelayDefaultsPanel(props: {
  formState: RelaySettingsFormState;
  controlsDisabled: boolean;
  onChange: (patch: Partial<RelaySettingsFormState>) => void;
}) {
  const { copy } = useWebLocale();
  const { formState, controlsDisabled, onChange } = props;
  const stopPolicyOptions: Array<{ value: RelayDefaultStopPolicy; label: string }> = [
    { value: 'ai_decides', label: copy.settings.relayStopAIDecides },
    { value: 'max_rounds', label: copy.settings.relayStopMaxRounds },
  ];

  return (
    <Card title={copy.settings.relayDefaultsTitle} copy={copy.settings.relayDefaultsDescription}>
      <SelectField
        label={copy.settings.relayStopPolicyLabel}
        description={copy.settings.relayStopPolicyDescription}
        value={formState.stopPolicy}
        disabled={controlsDisabled}
        options={stopPolicyOptions}
        onChange={(value) => onChange({ stopPolicy: value as RelayDefaultStopPolicy })}
      />
      <TextField
        label={copy.settings.relayMaxRoundsLabel}
        description={copy.settings.relayMaxRoundsDescription}
        value={formState.maxRounds}
        disabled={controlsDisabled}
        type="number"
        onChange={(value) => onChange({ maxRounds: value })}
      />
      <TextField
        label={copy.settings.relayTimeoutLabel}
        description={copy.settings.relayTimeoutDescription}
        value={formState.executionTimeoutMS}
        disabled={controlsDisabled}
        type="number"
        onChange={(value) => onChange({ executionTimeoutMS: value })}
      />
    </Card>
  );
}

function formSignature(formState: RelaySettingsFormState): string {
  return JSON.stringify(formState);
}

function InlineNotice(props: { text: string }) {
  return <div className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">{props.text}</div>;
}

function InlineError(props: { text: string }) {
  return <div className="rounded-[12px] border border-red-200 bg-red-50 px-4 py-3 text-[13px] text-red-700">{props.text}</div>;
}
