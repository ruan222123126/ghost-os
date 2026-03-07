// ConfigPanel component used by the web console chat/session interface.

'use client';

import type { FC, FormEvent } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { createProvider, deleteProvider, getProviders, setActiveProvider, updateProvider } from '@/lib/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { BridgeConfig, ConfigUpdate, ProviderConfig, ProviderConfigInput, ProviderListResponse } from '@/lib/types';

interface ConfigPanelProps {
  open: boolean;
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  error: string;
  onClose: () => void;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onReload: () => Promise<void>;
}

type EditorMode = 'create' | 'edit';

interface ProviderEditorState {
  name: string;
  providerType: ProviderConfig['type'];
  baseURL: string;
  apiKey: string;
  models: string;
}

const providerTypeOptions: Array<{ value: ProviderConfig['type']; label: string; defaultBaseURL: string | null }> = [
  { value: 'openai', label: 'OpenAI', defaultBaseURL: 'https://api.openai.com/v1' },
  { value: 'anthropic', label: 'Anthropic', defaultBaseURL: 'https://api.anthropic.com' },
  { value: 'custom', label: 'OpenAI-Compatible', defaultBaseURL: null },
];

const emptyEditorState: ProviderEditorState = {
  name: '',
  providerType: 'openai',
  baseURL: '',
  apiKey: '',
  models: '',
};

function defaultBaseURLForProviderType(providerType: ProviderConfig['type']): string {
  return providerTypeOptions.find((option) => option.value === providerType)?.defaultBaseURL ?? '';
}

function labelForProviderType(providerType: ProviderConfig['type']): string {
  return providerTypeOptions.find((option) => option.value === providerType)?.label ?? providerType;
}

function parseModels(raw: string): string[] {
  return raw
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function editorStateFromProvider(provider: ProviderConfig): ProviderEditorState {
  return {
    name: provider.name,
    providerType: provider.type,
    baseURL: provider.base_url,
    apiKey: '',
    models: provider.models?.join(', ') ?? '',
  };
}

export const ConfigPanel: FC<ConfigPanelProps> = ({ open, loading, saving, config, error, onClose, onSave, onReload }) => {
  const [model, setModel] = useState('');
  const [chatPath, setChatPath] = useState('');
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [activeProvider, setActiveProviderName] = useState('');
  const [providersLoading, setProvidersLoading] = useState(false);
  const [providerSaving, setProviderSaving] = useState(false);
  const [providerError, setProviderError] = useState('');
  const [editorMode, setEditorMode] = useState<EditorMode>('create');
  const [editingName, setEditingName] = useState('');
  const [editor, setEditor] = useState<ProviderEditorState>(emptyEditorState);

  const runtimeControlsDisabled = saving || loading;
  const providerControlsDisabled = providerSaving || providersLoading;
  const combinedError = useMemo(() => providerError || error, [error, providerError]);

  useEffect(() => {
    if (!config) {
      return;
    }
    setModel(config.model);
    setChatPath(config.chat_path || '');
  }, [config, open]);

  useEffect(() => {
    if (!open) {
      return;
    }

    ignorePromise(
      (async () => {
        setProvidersLoading(true);
        try {
          const result = await getProviders();
          applyProviderList(result);
          setProviderError('');
        } catch (loadError) {
          setProviderError(toErrorMessage(loadError, 'failed to load providers'));
        } finally {
          setProvidersLoading(false);
        }
      })(),
    );
  }, [open]);

  function applyProviderList(payload: ProviderListResponse) {
    setProviders(payload.providers);
    setActiveProviderName(payload.active_provider);
  }

  function resetEditor(mode: EditorMode = 'create') {
    setEditorMode(mode);
    setEditingName('');
    setEditor(emptyEditorState);
  }

  function handleRuntimeSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    ignorePromise(onSave({ model, chat_path: chatPath }));
  }

  async function runProviderMutation(
    action: () => Promise<ProviderListResponse>,
    errorMessage: string,
    onSuccess?: () => void | Promise<void>
  ) {
    setProviderSaving(true);
    setProviderError('');
    try {
      const result = await action();
      applyProviderList(result);
      await onReload();
      await onSuccess?.();
    } catch (operationError) {
      setProviderError(toErrorMessage(operationError, errorMessage));
    } finally {
      setProviderSaving(false);
    }
  }

  function handleProviderSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input: ProviderConfigInput = {
      name: editor.name.trim(),
      type: editor.providerType,
      models: parseModels(editor.models),
    };
    if (editor.baseURL.trim()) {
      input.base_url = editor.baseURL.trim();
    }
    if (editor.apiKey.trim()) {
      input.api_key = editor.apiKey.trim();
    }

    ignorePromise(
      runProviderMutation(
        () => (editorMode === 'edit' ? updateProvider(editingName, input) : createProvider(input)),
        'failed to save provider',
        () => {
          resetEditor();
        }
      ),
    );
  }

  function handleEditProvider(provider: ProviderConfig) {
    setEditorMode('edit');
    setEditingName(provider.name);
    setEditor(editorStateFromProvider(provider));
    setProviderError('');
  }

  function handleActivateProvider(name: string) {
    ignorePromise(runProviderMutation(() => setActiveProvider(name), 'failed to switch provider'));
  }

  function handleDeleteProvider(name: string) {
    if (!window.confirm(`Delete provider \"${name}\"?`)) {
      return;
    }

    ignorePromise(
      runProviderMutation(() => deleteProvider(name), 'failed to delete provider', () => {
        if (stringsEqualIgnoreCase(editingName, name)) {
          resetEditor();
        }
      }),
    );
  }

  if (!open) {
    return null;
  }

  return (
    <div className="ui-panel animate-riseSoft p-4">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-app-text">Model Providers</h2>
        <button
          type="button"
          onClick={onClose}
          className="ui-btn-secondary px-2 py-1 text-xs text-app-muted hover:text-app-text"
        >
          Close
        </button>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
        <section className="ui-panel-soft p-3">
          <div className="mb-3 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-medium text-app-text">Providers</h3>
              <p className="ui-hint">Switch the active model backend and manage saved endpoints.</p>
            </div>
            <button
              type="button"
              disabled={providerControlsDisabled}
              onClick={() => resetEditor('create')}
              className="ui-btn px-3 py-1.5 text-xs"
            >
              Add Provider
            </button>
          </div>

          {providersLoading && <p className="text-sm text-app-muted">Loading providers...</p>}
          {!providersLoading && providers.length === 0 && (
            <p className="ui-empty px-3 py-4 text-sm">
              No providers configured yet. Add one to create `~/.ghost-os/config.toml` provider entries.
            </p>
          )}

          <div className="grid gap-2">
            {providers.map((provider) => {
              const isActive = stringsEqualIgnoreCase(activeProvider, provider.name);
              return (
                <div
                  key={provider.name}
                  className={`ui-panel-soft px-3 py-3 transition ${
                    isActive
                      ? 'border-app-accent/60 bg-app-accent/10 shadow-lift'
                      : 'hover:border-app-fieldBorderHover/80 hover:shadow-lift'
                  }`}
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="mono text-sm font-medium text-app-text">{provider.name}</span>
                        <span className="rounded-full border border-app-border/90 px-2 py-0.5 text-[11px] text-app-muted">{labelForProviderType(provider.type)}</span>
                        {isActive && (
                          <span className="rounded-full border border-app-accent/40 bg-app-accent/15 px-2 py-0.5 text-[11px] text-app-text">
                            active
                          </span>
                        )}
                        {provider.api_key_set && (
                          <span className="rounded-full border border-app-border/90 px-2 py-0.5 text-[11px] text-app-muted">key set</span>
                        )}
                      </div>
                      <p className="mono mt-1 truncate text-xs text-app-muted">{provider.base_url}</p>
                      {!!provider.models?.length && (
                        <p className="mono mt-1 text-xs text-app-muted">models: {provider.models.join(', ')}</p>
                      )}
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        disabled={providerControlsDisabled || isActive}
                        onClick={() => handleActivateProvider(provider.name)}
                        className="ui-btn-secondary px-3 py-1.5 text-xs"
                      >
                        {isActive ? 'Active' : 'Use'}
                      </button>
                      <button
                        type="button"
                        disabled={providerControlsDisabled}
                        onClick={() => handleEditProvider(provider)}
                        className="ui-btn-secondary px-3 py-1.5 text-xs"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        disabled={providerControlsDisabled}
                        onClick={() => handleDeleteProvider(provider.name)}
                        className="ui-btn-secondary border-rose-400/30 px-3 py-1.5 text-xs text-rose-200 hover:border-rose-300/60 hover:text-rose-100"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        <section className="grid gap-4">
          <form className="ui-panel-soft grid gap-3 p-3" onSubmit={handleProviderSubmit}>
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-medium text-app-text">{editorMode === 'edit' ? 'Edit Provider' : 'Add Provider'}</h3>
                <p className="ui-hint">Provider names must be unique. API keys stay hidden after save.</p>
              </div>
              {editorMode === 'edit' && (
                <button
                  type="button"
                  onClick={() => resetEditor()}
                  className="ui-btn-secondary px-2 py-1 text-xs text-app-muted hover:text-app-text"
                >
                  Cancel
                </button>
              )}
            </div>

            <label className="ui-label">
              Name
              <input
                value={editor.name}
                disabled={providerControlsDisabled}
                onChange={(event) => setEditor((state) => ({ ...state, name: event.target.value }))}
                className="ui-input mono"
              />
            </label>

            <label className="ui-label">
              Provider Type
              <select
                value={editor.providerType}
                disabled={providerControlsDisabled}
                onChange={(event) => {
                  const providerType = event.target.value as ProviderConfig['type'];
                  setEditor((state) => {
                    const nextBaseURL = state.baseURL.trim();
                    const currentDefault = defaultBaseURLForProviderType(state.providerType);
                    const shouldReplaceBaseURL = nextBaseURL === '' || nextBaseURL === currentDefault;
                    return {
                      ...state,
                      providerType,
                      baseURL: shouldReplaceBaseURL ? defaultBaseURLForProviderType(providerType) : state.baseURL,
                    };
                  });
                }}
                className="ui-select"
              >
                {providerTypeOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="ui-label">
              Base URL
              <input
                value={editor.baseURL}
                disabled={providerControlsDisabled}
                placeholder={editor.providerType === 'custom' ? 'https://example.com/v1' : defaultBaseURLForProviderType(editor.providerType)}
                onChange={(event) => setEditor((state) => ({ ...state, baseURL: event.target.value }))}
                className="ui-input mono"
              />
            </label>

            <label className="ui-label">
              API Key (optional)
              <input
                type="password"
                value={editor.apiKey}
                disabled={providerControlsDisabled}
                placeholder={editorMode === 'edit' ? 'Leave blank to keep current key' : 'Enter API key'}
                onChange={(event) => setEditor((state) => ({ ...state, apiKey: event.target.value }))}
                className="ui-input"
              />
            </label>

            <label className="ui-label">
              Models (optional)
              <input
                value={editor.models}
                disabled={providerControlsDisabled}
                onChange={(event) => setEditor((state) => ({ ...state, models: event.target.value }))}
                placeholder="gpt-5.4, gpt-4.1"
                className="ui-input mono"
              />
            </label>

            <div className="flex justify-end">
              <button
                type="submit"
                disabled={providerControlsDisabled}
                className="ui-btn px-4 py-2 text-sm"
              >
                {providerSaving ? 'Saving...' : editorMode === 'edit' ? 'Update Provider' : 'Create Provider'}
              </button>
            </div>
          </form>

          <form className="ui-panel-soft grid gap-3 p-3" onSubmit={handleRuntimeSubmit}>
            <div>
              <h3 className="text-sm font-medium text-app-text">Runtime</h3>
              <p className="ui-hint">Model and chat path stay on the active runtime config.</p>
            </div>

            {loading && <p className="text-sm text-app-muted">Loading runtime config...</p>}

            <label className="ui-label">
              Model
              <input
                value={model}
                disabled={runtimeControlsDisabled}
                onChange={(event) => setModel(event.target.value)}
                className="ui-input mono"
              />
            </label>

            <label className="ui-label">
              Chat Path (optional)
              <input
                value={chatPath}
                disabled={runtimeControlsDisabled}
                onChange={(event) => setChatPath(event.target.value)}
                className="ui-input mono"
              />
            </label>

            <div className="flex justify-end">
              <button
                type="submit"
                disabled={runtimeControlsDisabled}
                className="ui-btn px-4 py-2 text-sm"
              >
                {saving ? 'Saving...' : 'Save Runtime'}
              </button>
            </div>
          </form>
        </section>
      </div>

      {combinedError && (
        <p className="mt-4 rounded-xl border border-rose-300/30 bg-rose-300/10 px-3 py-2 text-sm text-rose-200">{combinedError}</p>
      )}
    </div>
  );
};

function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}
