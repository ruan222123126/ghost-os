'use client';

import { ProviderEditorForm } from '@/components/config/ProviderEditorForm';
import { ProviderList } from '@/components/config/ProviderList';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { useConfigProviders } from '@/hooks/useConfigProviders';

interface ProviderSettingsSectionProps {
  machine: ReturnType<typeof useConfigProviders>;
}

export function ProviderSettingsSection(props: ProviderSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { machine } = props;
  const { state, actions } = machine;
  const controlsDisabled = state.loading || state.saving;

  if (state.view === 'editor') {
    return (
      <ProviderEditorForm
        editorMode={state.editorMode}
        editor={state.editor}
        controlsDisabled={controlsDisabled}
        saving={state.saving}
        onChangeEditor={actions.updateEditor}
        onSelectProviderType={actions.selectProviderType}
        onSubmit={async () => {
          await actions.submit();
        }}
        onCancelEditing={actions.cancelEditing}
      />
    );
  }

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.providerTitle}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.providerDescription}</p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(actions.refresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.refresh}
          </button>
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={actions.startCreate}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.providerAdd}
          </button>
        </div>
      </header>

      <ProviderList
        providers={state.providers}
        activeProvider={state.activeProvider}
        loading={state.loading}
        controlsDisabled={controlsDisabled}
        onActivate={actions.activate}
        onDelete={actions.delete}
        onEdit={actions.startEdit}
      />
    </section>
  );
}
