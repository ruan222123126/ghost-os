'use client';

import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SystemPromptPayload } from '@/lib/types';

interface PromptsPreviewSettingsSectionProps {
  prompts: SystemPromptPayload | null;
  loading: boolean;
  saving: boolean;
  onRefresh: () => Promise<void>;
}

export function PromptsPreviewSettingsSection(props: PromptsPreviewSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { prompts, loading, saving, onRefresh } = props;
  const controlsDisabled = loading || saving;

  return (
    <section>
      <header className="mb-8 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.tabPromptsPreview}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.promptsPreviewDescription}</p>
        </div>
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
      </header>

      {loading && prompts === null ? <LoadingPromptsNotice text={copy.settings.promptsLoading} /> : null}

      <section className="rounded-[16px] border border-[#E5E5E5] bg-white p-5">
        <div className="mb-4">
          <h2 className="text-[18px] font-semibold text-[#111111]">{copy.settings.promptsPreviewTitle}</h2>
        </div>
        <pre
          data-testid="prompts-rendered-preview"
          className="max-h-[620px] overflow-auto whitespace-pre-wrap break-words rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-3 py-2 font-mono text-[12px] leading-6 text-[#111111]"
        >
          {(prompts?.rendered_prompt ?? '').trim() === '' ? copy.settings.promptsRenderedEmpty : prompts?.rendered_prompt}
        </pre>
      </section>
    </section>
  );
}

function LoadingPromptsNotice(props: { text: string }) {
  return (
    <div className="mb-4 rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">
      {props.text}
    </div>
  );
}
