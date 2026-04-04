'use client';

import type { FormEvent, ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

interface RuntimeSettingsSectionProps {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onReload: () => Promise<void>;
}

interface RuntimeFormState {
  model: string;
  chatPath: string;
  webSearchTavilyURL: string;
  webSearchExaURL: string;
  webSearchTavilyAPIKey: string;
  webSearchExaAPIKey: string;
}

interface FieldProps {
  label: string;
  description: string;
  children: ReactNode;
}

export function RuntimeSettingsSection(props: RuntimeSettingsSectionProps) {
  const { loading, saving, config, onSave, onReload } = props;
  const [formState, setFormState] = useState<RuntimeFormState>(() => createRuntimeFormState(null));
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;
  const controlsDisabled = loading || saving;

  useEffect(() => {
    setFormState(createRuntimeFormState(config));
  }, [config]);

  return (
    <section>
      <header className="mb-10">
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">General</h1>
        <p className="text-[14px] text-[#737373]">
          Update runtime defaults, chat path, and web search credentials.
        </p>
      </header>

      <form className="space-y-4" onSubmit={(event) => handleRuntimeSubmit(event, modelSelectionEnabled, formState, onSave)}>
        {loading ? (
          <div className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">
            Loading runtime config...
          </div>
        ) : null}

        <section className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
          <div className="mb-5">
            <h3 className="text-[18px] font-medium text-[#111111]">Runtime Core</h3>
            <p className="mt-1 text-[13px] text-[#737373]">Core defaults used when starting new runs.</p>
          </div>

          <div className="space-y-5">
            <Field
              label="Model"
              description={
                modelSelectionEnabled
                  ? 'Default model ID used when the console starts a run.'
                  : 'Model selection is disabled by current runtime config.'
              }
            >
              <input
                value={formState.model}
                disabled={controlsDisabled || !modelSelectionEnabled}
                onChange={(event) => setFormState((prev) => ({ ...prev, model: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>

            <Field label="Chat Path" description="Optional chat path override for the active runtime.">
              <input
                value={formState.chatPath}
                disabled={controlsDisabled}
                onChange={(event) => setFormState((prev) => ({ ...prev, chatPath: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>
          </div>
        </section>

        <section className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
          <div className="mb-5">
            <h3 className="text-[18px] font-medium text-[#111111]">Web Search</h3>
            <p className="mt-1 text-[13px] text-[#737373]">
              Configure Tavily and Exa providers. Blank URLs keep official endpoints.
            </p>
          </div>

          <div className="space-y-5">
            <Field label="Tavily Custom URL" description="Optional Tavily endpoint override.">
              <input
                value={formState.webSearchTavilyURL}
                disabled={controlsDisabled}
                placeholder="Leave blank to use the official Tavily endpoint"
                onChange={(event) => setFormState((prev) => ({ ...prev, webSearchTavilyURL: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>

            <Field label="Tavily API Key" description="Blank keeps currently saved key.">
              <input
                type="password"
                value={formState.webSearchTavilyAPIKey}
                disabled={controlsDisabled}
                placeholder={buildAPIKeyPlaceholder(config?.web_search_tavily_api_key_set, 'Tavily')}
                onChange={(event) => setFormState((prev) => ({ ...prev, webSearchTavilyAPIKey: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>

            <Field label="Exa Custom URL" description="Optional Exa endpoint override.">
              <input
                value={formState.webSearchExaURL}
                disabled={controlsDisabled}
                placeholder="Leave blank to use the official Exa endpoint"
                onChange={(event) => setFormState((prev) => ({ ...prev, webSearchExaURL: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>

            <Field label="Exa API Key" description="Blank keeps currently saved key.">
              <input
                type="password"
                value={formState.webSearchExaAPIKey}
                disabled={controlsDisabled}
                placeholder={buildAPIKeyPlaceholder(config?.web_search_exa_api_key_set, 'Exa')}
                onChange={(event) => setFormState((prev) => ({ ...prev, webSearchExaAPIKey: event.target.value }))}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </Field>
          </div>
        </section>

        <div className="flex items-center justify-end gap-3 pt-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onReload());
            }}
            className="rounded-full border border-[#E5E5E5] px-6 py-2.5 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Reload
          </button>
          <button
            type="submit"
            disabled={controlsDisabled}
            className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </form>
    </section>
  );
}

function Field(props: FieldProps) {
  const { label, description, children } = props;

  return (
    <label className="block">
      <span className="mb-1 block text-[13px] font-medium text-[#111111]">{label}</span>
      <span className="mb-2 block text-[12px] text-[#737373]">{description}</span>
      {children}
    </label>
  );
}

function createRuntimeFormState(config: BridgeConfig | null): RuntimeFormState {
  return {
    model: config?.model ?? '',
    chatPath: config?.chat_path ?? '',
    webSearchTavilyURL: config?.web_search_tavily_url ?? '',
    webSearchExaURL: config?.web_search_exa_url ?? '',
    webSearchTavilyAPIKey: '',
    webSearchExaAPIKey: '',
  };
}

function buildAPIKeyPlaceholder(apiKeySet: boolean | undefined, providerName: string): string {
  if (apiKeySet) {
    return `Leave blank to keep saved ${providerName} key`;
  }

  return `Enter ${providerName} API key`;
}

function handleRuntimeSubmit(
  event: FormEvent<HTMLFormElement>,
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
  onSave: (update: ConfigUpdate) => Promise<boolean>,
) {
  event.preventDefault();
  ignorePromise(onSave(buildRuntimeUpdate(modelSelectionEnabled, formState)));
}

function buildRuntimeUpdate(modelSelectionEnabled: boolean, formState: RuntimeFormState): ConfigUpdate {
  const update: ConfigUpdate = {
    chat_path: formState.chatPath,
    web_search_tavily_url: formState.webSearchTavilyURL,
    web_search_exa_url: formState.webSearchExaURL,
  };

  if (modelSelectionEnabled) {
    update.model = formState.model;
  }
  if (formState.webSearchTavilyAPIKey.trim()) {
    update.web_search_tavily_api_key = formState.webSearchTavilyAPIKey;
  }
  if (formState.webSearchExaAPIKey.trim()) {
    update.web_search_exa_api_key = formState.webSearchExaAPIKey;
  }

  return update;
}
