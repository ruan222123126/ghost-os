'use client';

import { ProviderEditorForm } from '@/components/config/ProviderEditorForm';
import { ProviderList } from '@/components/config/ProviderList';
import { ignorePromise } from '@/lib/errors';
import { ProviderEditorState } from '@/lib/configProviders';
import type { ProviderConfig } from '@/lib/types';

interface ProviderSettingsSectionProps {
  providers: ProviderConfig[];
  activeProvider: string;
  loading: boolean;
  saving: boolean;
  editorMode: 'create' | 'edit';
  editor: ProviderEditorState;
  onRefresh: () => Promise<void>;
  onBeginCreate: () => void;
  onEdit: (provider: ProviderConfig) => void;
  onChangeEditor: (patch: Partial<ProviderEditorState>) => void;
  onSelectProviderType: (providerType: ProviderConfig['type']) => void;
  onSubmit: () => Promise<void>;
  onActivate: (name: string) => Promise<void>;
  onDelete: (name: string) => Promise<void>;
  onCancelEditing: () => void;
}

export function ProviderSettingsSection(props: ProviderSettingsSectionProps) {
  const {
    providers,
    activeProvider,
    loading,
    saving,
    editorMode,
    editor,
    onRefresh,
    onBeginCreate,
    onEdit,
    onChangeEditor,
    onSelectProviderType,
    onSubmit,
    onActivate,
    onDelete,
    onCancelEditing,
  } = props;
  const controlsDisabled = loading || saving;

  return (
    <section className="settings-section">
      <div className="section-head">
        <div className="section-head-copy">
          <h3>Providers</h3>
          <p className="section-copy">Keep the visual structure aligned with NextAI, while only exposing Ghost-OS features that already exist today.</p>
        </div>

        <div className="panel-head-actions">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onRefresh());
            }}
            className="button-secondary"
          >
            Refresh
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={onBeginCreate}
            className="button"
          >
            Add Provider
          </button>
        </div>
      </div>

      <div className="settings-grid">
        <ProviderList
          providers={providers}
          activeProvider={activeProvider}
          loading={loading}
          controlsDisabled={controlsDisabled}
          onActivate={onActivate}
          onDelete={onDelete}
          onEdit={onEdit}
        />

        <ProviderEditorForm
          editorMode={editorMode}
          editor={editor}
          controlsDisabled={controlsDisabled}
          saving={saving}
          onChangeEditor={onChangeEditor}
          onSelectProviderType={onSelectProviderType}
          onSubmit={onSubmit}
          onCancelEditing={onCancelEditing}
        />
      </div>
    </section>
  );
}
