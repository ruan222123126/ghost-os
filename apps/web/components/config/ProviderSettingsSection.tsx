'use client';

import { useState } from 'react';
import { ProviderEditorForm } from '@/components/config/ProviderEditorForm';
import { ProviderList } from '@/components/config/ProviderList';
import { ignorePromise } from '@/lib/errors';
import type { ProviderEditorState } from '@/lib/configProviders';
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
  onSubmit: () => Promise<boolean>;
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
  const [editorOpen, setEditorOpen] = useState(false);
  const controlsDisabled = loading || saving;

  const handleBeginCreate = () => {
    onBeginCreate();
    setEditorOpen(true);
  };

  const handleEdit = (provider: ProviderConfig) => {
    onEdit(provider);
    setEditorOpen(true);
  };

  const handleCancel = () => {
    onCancelEditing();
    setEditorOpen(false);
  };

  const handleSubmit = async () => {
    if (await onSubmit()) {
      setEditorOpen(false);
    }
  };

  if (editorOpen) {
    return (
      <ProviderEditorForm
        editorMode={editorMode}
        editor={editor}
        controlsDisabled={controlsDisabled}
        saving={saving}
        onChangeEditor={onChangeEditor}
        onSelectProviderType={onSelectProviderType}
        onSubmit={handleSubmit}
        onCancelEditing={handleCancel}
      />
    );
  }

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">Provider</h1>
          <p className="text-[14px] text-[#737373]">Manage your service connections.</p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onRefresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Refresh
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={handleBeginCreate}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Add
          </button>
        </div>
      </header>

      <ProviderList
        providers={providers}
        activeProvider={activeProvider}
        loading={loading}
        controlsDisabled={controlsDisabled}
        onActivate={onActivate}
        onDelete={onDelete}
        onEdit={handleEdit}
      />
    </section>
  );
}
