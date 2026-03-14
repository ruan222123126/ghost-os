// ConfigPanel component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useMemo, useState } from 'react';
import { ProviderSettingsSection } from '@/components/config/ProviderSettingsSection';
import { RuntimeSettingsSection } from '@/components/config/RuntimeSettingsSection';
import { useConfigProviders } from '@/hooks/useConfigProviders';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

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

type SettingsSection = 'providers' | 'runtime';

export const ConfigPanel: FC<ConfigPanelProps> = ({ open, loading, saving, config, error, onClose, onSave, onReload }) => {
  const [activeSection, setActiveSection] = useState<SettingsSection>('providers');
  const {
    providers,
    activeProvider,
    providersLoading,
    providerSaving,
    providerError,
    editorMode,
    editor,
    refreshProviders,
    beginCreateProvider,
    editProvider,
    updateEditor,
    selectProviderType,
    submitProvider,
    activateProvider,
    deleteProviderByName,
    cancelEditing,
  } = useConfigProviders({
    open,
    onReloadConfig: onReload,
  });
  const combinedError = useMemo(() => providerError || error, [error, providerError]);

  if (!open) {
    return null;
  }

  return (
    <div className="settings-popover" role="dialog" aria-modal="true" aria-labelledby="settings-title">
      <button type="button" className="settings-backdrop" onClick={onClose} aria-label="Close settings" />

      <section className="settings-panel">
        <div className="settings-panel-head">
          <div>
            <p className="kicker">Runtime Settings</p>
            <h2 id="settings-title">Control Center</h2>
          </div>

          <div className="panel-head-actions">
            <button type="button" onClick={onClose} className="button-secondary">
              Close
            </button>
          </div>
        </div>

        {combinedError ? <div className="status-line error">{combinedError}</div> : null}

        <div className="settings-layout">
          <nav className="settings-nav" aria-label="Settings sections">
            <button
              type="button"
              onClick={() => setActiveSection('providers')}
              className={`settings-nav-item${activeSection === 'providers' ? ' is-active' : ''}`}
            >
              Providers
              <span>Manage saved model endpoints and active backend selection.</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveSection('runtime')}
              className={`settings-nav-item${activeSection === 'runtime' ? ' is-active' : ''}`}
            >
              Runtime
              <span>Update default model and optional chat path for the current runtime.</span>
            </button>
          </nav>

          <div className="settings-sections ui-scroll">
            {activeSection === 'providers' ? (
              <ProviderSettingsSection
                providers={providers}
                activeProvider={activeProvider}
                loading={providersLoading}
                saving={providerSaving}
                editorMode={editorMode}
                editor={editor}
                onRefresh={refreshProviders}
                onBeginCreate={beginCreateProvider}
                onEdit={editProvider}
                onChangeEditor={updateEditor}
                onSelectProviderType={selectProviderType}
                onSubmit={submitProvider}
                onActivate={activateProvider}
                onDelete={deleteProviderByName}
                onCancelEditing={cancelEditing}
              />
            ) : (
              <RuntimeSettingsSection
                loading={loading}
                saving={saving}
                config={config}
                onSave={onSave}
                onReload={onReload}
              />
            )}
          </div>
        </div>
      </section>
    </div>
  );
};
