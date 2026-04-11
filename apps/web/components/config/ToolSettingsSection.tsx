'use client';

import { ToolList } from '@/components/config/ToolList';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolUpdateRequest, ToolPayload } from '@/lib/types';

interface ToolSettingsSectionProps {
  tools: ToolPayload[];
  loading: boolean;
  saving: boolean;
  onRefresh: () => Promise<void>;
  onUpdate: (name: string, input: ToolUpdateRequest) => Promise<void>;
}

export function ToolSettingsSection(props: ToolSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { tools, loading, saving, onRefresh, onUpdate } = props;
  const controlsDisabled = loading || saving;

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.toolsTitle}</h1>
          <p className="text-[14px] text-[#737373]">
            {copy.settings.toolsDescription}
          </p>
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
            {copy.settings.refresh}
          </button>
        </div>
      </header>

      <ToolList
        tools={tools}
        loading={loading}
        controlsDisabled={controlsDisabled}
        onUpdate={onUpdate}
      />
    </section>
  );
}
