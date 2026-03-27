'use client';

import type { FormEvent } from 'react';
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

export function RuntimeSettingsSection(props: RuntimeSettingsSectionProps) {
  const { loading, saving, config, onSave, onReload } = props;
  const [formState, setFormState] = useState<RuntimeFormState>(() => createRuntimeFormState(null));

  useEffect(() => {
    setFormState(createRuntimeFormState(config));
  }, [config]);

  return (
    <section className="settings-section">
      <div className="section-head-copy">
        <h3>Runtime</h3>
        <p className="section-copy">Update runtime defaults and configure Tavily / Exa search endpoints and keys.</p>
      </div>

      <div className="settings-stack">
        <RuntimeSettingsForm
          loading={loading}
          saving={saving}
          config={config}
          formState={formState}
          onChange={setFormState}
          onSave={onSave}
          onReload={onReload}
        />
      </div>
    </section>
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

interface RuntimeFieldProps {
  modelSelectionEnabled: boolean;
  runtimeControlsDisabled: boolean;
  formState: RuntimeFormState;
  onChange: (updater: (prev: RuntimeFormState) => RuntimeFormState) => void;
}

interface RuntimeSettingsFormProps {
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  formState: RuntimeFormState;
  onChange: (updater: (prev: RuntimeFormState) => RuntimeFormState) => void;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onReload: () => Promise<void>;
}

function RuntimeSettingsForm(props: RuntimeSettingsFormProps) {
  const { loading, saving, config, formState, onChange, onSave, onReload } = props;
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;
  const runtimeControlsDisabled = loading || saving;

  return (
    <form className="settings-form" onSubmit={(event) => handleRuntimeSubmit(event, modelSelectionEnabled, formState, onSave)}>
      {loading ? <div className="status-line info">Loading runtime config...</div> : null}

      <RuntimeCoreFields
        modelSelectionEnabled={modelSelectionEnabled}
        runtimeControlsDisabled={runtimeControlsDisabled}
        formState={formState}
        onChange={onChange}
      />

      <WebSearchFields
        runtimeControlsDisabled={runtimeControlsDisabled}
        formState={formState}
        config={config}
        onChange={onChange}
      />

      <RuntimeFormActions runtimeControlsDisabled={runtimeControlsDisabled} saving={saving} onReload={onReload} />
    </form>
  );
}

function RuntimeCoreFields(props: RuntimeFieldProps) {
  const { modelSelectionEnabled, runtimeControlsDisabled, formState, onChange } = props;

  return (
    <>
      <label className="field-label">
        Model
        <input
          value={formState.model}
          disabled={runtimeControlsDisabled || !modelSelectionEnabled}
          onChange={(event) => onChange((prev) => ({ ...prev, model: event.target.value }))}
          className="input mono"
        />
      </label>

      <label className="field-label">
        Chat Path (optional)
        <input
          value={formState.chatPath}
          disabled={runtimeControlsDisabled}
          onChange={(event) => onChange((prev) => ({ ...prev, chatPath: event.target.value }))}
          className="input mono"
        />
      </label>
    </>
  );
}

interface WebSearchFieldsProps {
  runtimeControlsDisabled: boolean;
  formState: RuntimeFormState;
  config: BridgeConfig | null;
  onChange: (updater: (prev: RuntimeFormState) => RuntimeFormState) => void;
}

function WebSearchFields(props: WebSearchFieldsProps) {
  const { runtimeControlsDisabled, formState, config, onChange } = props;
  const tavilyPlaceholder = buildAPIKeyPlaceholder(config?.web_search_tavily_api_key_set, 'Tavily');
  const exaPlaceholder = buildAPIKeyPlaceholder(config?.web_search_exa_api_key_set, 'Exa');

  return (
    <>
      <label className="field-label">
        Tavily Custom URL (optional)
        <input
          value={formState.webSearchTavilyURL}
          disabled={runtimeControlsDisabled}
          placeholder="Leave blank to use the official Tavily endpoint"
          onChange={(event) => onChange((prev) => ({ ...prev, webSearchTavilyURL: event.target.value }))}
          className="input mono"
        />
      </label>

      <label className="field-label">
        Tavily API Key (optional)
        <input
          type="password"
          value={formState.webSearchTavilyAPIKey}
          disabled={runtimeControlsDisabled}
          placeholder={tavilyPlaceholder}
          onChange={(event) => onChange((prev) => ({ ...prev, webSearchTavilyAPIKey: event.target.value }))}
          className="input"
        />
      </label>

      <label className="field-label">
        Exa Custom URL (optional)
        <input
          value={formState.webSearchExaURL}
          disabled={runtimeControlsDisabled}
          placeholder="Leave blank to use the official Exa endpoint"
          onChange={(event) => onChange((prev) => ({ ...prev, webSearchExaURL: event.target.value }))}
          className="input mono"
        />
      </label>

      <label className="field-label">
        Exa API Key (optional)
        <input
          type="password"
          value={formState.webSearchExaAPIKey}
          disabled={runtimeControlsDisabled}
          placeholder={exaPlaceholder}
          onChange={(event) => onChange((prev) => ({ ...prev, webSearchExaAPIKey: event.target.value }))}
          className="input"
        />
      </label>
    </>
  );
}

interface RuntimeFormActionsProps {
  runtimeControlsDisabled: boolean;
  saving: boolean;
  onReload: () => Promise<void>;
}

function RuntimeFormActions(props: RuntimeFormActionsProps) {
  const { runtimeControlsDisabled, saving, onReload } = props;

  return (
    <div className="settings-actions">
      <button
        type="button"
        disabled={runtimeControlsDisabled}
        onClick={() => {
          ignorePromise(onReload());
        }}
        className="button-secondary"
      >
        Reload
      </button>
      <button type="submit" disabled={runtimeControlsDisabled} className="button">
        {saving ? 'Saving...' : 'Save Runtime'}
      </button>
    </div>
  );
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
  const update = buildRuntimeUpdate(modelSelectionEnabled, formState);
  ignorePromise(onSave(update));
}

function buildRuntimeUpdate(
  modelSelectionEnabled: boolean,
  formState: RuntimeFormState,
): ConfigUpdate {
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
