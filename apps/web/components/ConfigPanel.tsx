'use client';

import type { FC, FormEvent } from 'react';
import { useEffect, useRef, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

const PROVIDERS = [
  { value: 'openai', label: 'openai' },
  { value: 'anthropic', label: 'anthropic' },
  { value: 'custom', label: 'openai compatibility' },
] as const;

interface ConfigPanelProps {
  open: boolean;
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  error: string;
  onClose: () => void;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
}

export const ConfigPanel: FC<ConfigPanelProps> = ({ open, loading, saving, config, error, onClose, onSave }) => {
  const [provider, setProvider] = useState('openai');
  const [baseURL, setBaseURL] = useState('');
  const [model, setModel] = useState('');
  const [chatPath, setChatPath] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const firstInputRef = useRef<HTMLInputElement>(null);
  const controlsDisabled = saving || loading;

  useEffect(() => {
    if (!config) {
      return;
    }
    setProvider(config.provider);
    setBaseURL(config.base_url);
    setModel(config.model);
    setChatPath(config.chat_path || '');
    setAPIKey('');
  }, [config, open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    firstInputRef.current?.focus();
  }, [open]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const payload: ConfigUpdate = {
      provider,
      base_url: baseURL,
      model,
      chat_path: chatPath,
    };
    if (apiKey.trim()) {
      payload.api_key = apiKey.trim();
    }

    ignorePromise(
      onSave(payload).then((saved) => {
        if (saved) {
          onClose();
        }
      }),
    );
  }

  if (!open) {
    return null;
  }

  return (
    <div className="animate-rise rounded-2xl border border-app-border bg-app-panel/95 p-4 shadow-xl backdrop-blur">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-app-text">Runtime Config</h2>
        <button
          type="button"
          onClick={onClose}
          className="rounded-md border border-app-border px-2 py-1 text-xs text-app-muted transition hover:text-app-text"
        >
          Close
        </button>
      </div>

      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={handleSubmit}
      >
        {loading && <p className="sm:col-span-2 text-sm text-app-muted">Loading runtime config...</p>}

        <label className="text-sm text-app-muted">
          Provider
          <select
            value={provider}
            disabled={controlsDisabled}
            onChange={(event) => setProvider(event.target.value)}
            aria-label="Provider"
            className="mt-1 w-full rounded-lg border border-app-border bg-[#070c17] px-3 py-2 text-app-text outline-none transition focus:border-app-accent"
          >
            {PROVIDERS.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
        </label>

        <label className="text-sm text-app-muted">
          Model
          <input
            ref={firstInputRef}
            value={model}
            disabled={controlsDisabled}
            onChange={(event) => setModel(event.target.value)}
            className="mono mt-1 w-full rounded-lg border border-app-border bg-[#070c17] px-3 py-2 text-app-text outline-none transition focus:border-app-accent"
          />
        </label>

        <label className="text-sm text-app-muted sm:col-span-2">
          Base URL
          <input
            value={baseURL}
            disabled={controlsDisabled}
            onChange={(event) => setBaseURL(event.target.value)}
            className="mono mt-1 w-full rounded-lg border border-app-border bg-[#070c17] px-3 py-2 text-app-text outline-none transition focus:border-app-accent"
          />
        </label>

        <label className="text-sm text-app-muted sm:col-span-2">
          Chat Path (optional)
          <input
            value={chatPath}
            disabled={controlsDisabled}
            onChange={(event) => setChatPath(event.target.value)}
            className="mono mt-1 w-full rounded-lg border border-app-border bg-[#070c17] px-3 py-2 text-app-text outline-none transition focus:border-app-accent"
          />
        </label>

        <label className="text-sm text-app-muted sm:col-span-2">
          API Key (hidden after save)
          <input
            type="password"
            value={apiKey}
            disabled={controlsDisabled}
            placeholder={config?.api_key_set ? 'Configured (enter to replace)' : 'Enter API key'}
            onChange={(event) => setAPIKey(event.target.value)}
            className="mt-1 w-full rounded-lg border border-app-border bg-[#070c17] px-3 py-2 text-app-text outline-none transition focus:border-app-accent"
          />
        </label>

        {error && <p className="text-sm text-rose-300 sm:col-span-2">{error}</p>}

        <div className="sm:col-span-2 flex justify-end">
          <button
            type="submit"
            disabled={controlsDisabled}
            className="rounded-lg border border-app-accent/40 bg-app-accent/20 px-4 py-2 text-sm font-medium text-app-text transition hover:bg-app-accent/30 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {saving ? 'Saving...' : 'Save Config'}
          </button>
        </div>
      </form>
    </div>
  );
};
