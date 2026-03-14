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

export function RuntimeSettingsSection(props: RuntimeSettingsSectionProps) {
  const { loading, saving, config, onSave, onReload } = props;
  const [model, setModel] = useState('');
  const [chatPath, setChatPath] = useState('');
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;
  const runtimeControlsDisabled = loading || saving;

  useEffect(() => {
    if (!config) {
      return;
    }

    setModel(config.model);
    setChatPath(config.chat_path || '');
  }, [config]);

  return (
    <section className="settings-section">
      <div className="section-head-copy">
        <h3>Runtime</h3>
        <p className="section-copy">These values stay inside the current Ghost-OS runtime and do not add any new settings surface beyond what already exists.</p>
      </div>

      <div className="settings-stack">
        <form className="settings-form" onSubmit={(event) => handleRuntimeSubmit(event, modelSelectionEnabled, model, chatPath, onSave)}>
          {loading ? <div className="status-line info">Loading runtime config...</div> : null}

          <label className="field-label">
            Model
            <input
              value={model}
              disabled={runtimeControlsDisabled || !modelSelectionEnabled}
              onChange={(event) => setModel(event.target.value)}
              className="input mono"
            />
          </label>

          <label className="field-label">
            Chat Path (optional)
            <input
              value={chatPath}
              disabled={runtimeControlsDisabled}
              onChange={(event) => setChatPath(event.target.value)}
              className="input mono"
            />
          </label>

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
        </form>
      </div>
    </section>
  );
}

function handleRuntimeSubmit(
  event: FormEvent<HTMLFormElement>,
  modelSelectionEnabled: boolean,
  model: string,
  chatPath: string,
  onSave: (update: ConfigUpdate) => Promise<boolean>,
) {
  event.preventDefault();

  const update: ConfigUpdate = { chat_path: chatPath };
  if (modelSelectionEnabled) {
    update.model = model;
  }

  ignorePromise(onSave(update));
}
