'use client';

import { useEffect, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SystemPromptPayload } from '@/lib/types';
import type { SystemPromptField } from '@/hooks/useConfigPrompts';

interface PromptsSettingsSectionProps {
  prompts: SystemPromptPayload | null;
  loading: boolean;
  saving: boolean;
  onRefresh: () => Promise<void>;
  onSaveField: (field: SystemPromptField, value: string) => Promise<void>;
}

interface PromptEditorCardProps {
  field: SystemPromptField;
  title: string;
  description: string;
  savedValue: string;
  controlsDisabled: boolean;
  rows?: number;
  monospace?: boolean;
  onSaveField: (field: SystemPromptField, value: string) => Promise<void>;
}

export function PromptsSettingsSection(props: PromptsSettingsSectionProps) {
  const { copy } = useWebLocale();
  const { prompts, loading, saving, onRefresh, onSaveField } = props;
  const controlsDisabled = loading || saving;

  return (
    <section>
      <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.tabPrompts}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.promptsDescription}</p>
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

      {loading && prompts === null ? <LoadingPromptsNotice text={copy.settings.promptsLoading} /> : null}

      <div className="space-y-4">
        <PromptEditorCard
          field="global_template"
          title={copy.settings.promptsGlobalTemplateLabel}
          description={copy.settings.promptsGlobalTemplateDescription}
          savedValue={prompts?.global_template ?? ''}
          controlsDisabled={controlsDisabled || prompts === null}
          rows={10}
          monospace
          onSaveField={onSaveField}
        />

        <PromptEditorCard
          field="core_prompt"
          title={copy.settings.promptsCorePromptLabel}
          description={copy.settings.promptsCorePromptDescription}
          savedValue={prompts?.core_prompt ?? ''}
          controlsDisabled={controlsDisabled || prompts === null}
          rows={8}
          onSaveField={onSaveField}
        />

        <PromptEditorCard
          field="tool_prompt"
          title={copy.settings.promptsToolPromptLabel}
          description={copy.settings.promptsToolPromptDescription}
          savedValue={prompts?.tool_prompt ?? ''}
          controlsDisabled={controlsDisabled || prompts === null}
          rows={8}
          onSaveField={onSaveField}
        />

        <PromptEditorCard
          field="tool_key_spec"
          title={copy.settings.promptsToolKeySpecLabel}
          description={copy.settings.promptsToolKeySpecDescription}
          savedValue={prompts?.tool_key_spec ?? ''}
          controlsDisabled={controlsDisabled || prompts === null}
          rows={8}
          onSaveField={onSaveField}
        />

        <RenderedPromptCard
          title={copy.settings.promptsRenderedPromptLabel}
          description={copy.settings.promptsRenderedPromptDescription}
          value={prompts?.rendered_prompt ?? ''}
        />
      </div>
    </section>
  );
}

function PromptEditorCard(props: PromptEditorCardProps) {
  const { copy } = useWebLocale();
  const {
    field,
    title,
    description,
    savedValue,
    controlsDisabled,
    rows = 8,
    monospace = false,
    onSaveField,
  } = props;
  const [draft, setDraft] = useState(savedValue);

  useEffect(() => {
    setDraft(savedValue);
  }, [savedValue]);

  const dirty = draft !== savedValue;

  return (
    <section className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
      <div className="mb-3">
        <h2 className="text-[16px] font-semibold text-[#111111]">{title}</h2>
        <p className="mt-1 text-[13px] text-[#737373]">{description}</p>
      </div>
      <textarea
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        disabled={controlsDisabled}
        rows={rows}
        className={`mb-4 w-full rounded-[12px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60 ${
          monospace ? 'font-mono' : ''
        }`}
      />
      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={controlsDisabled || !dirty}
          onClick={() => {
            ignorePromise(onSaveField(field, draft));
          }}
          className="rounded-full bg-[#111111] px-4 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.save}
        </button>
        <button
          type="button"
          disabled={controlsDisabled || !dirty}
          onClick={() => setDraft(savedValue)}
          className="rounded-full border border-[#E5E5E5] px-4 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.promptsReset}
        </button>
      </div>
    </section>
  );
}

function RenderedPromptCard(props: {
  title: string;
  description: string;
  value: string;
}) {
  const { copy } = useWebLocale();
  const { title, description, value } = props;

  return (
    <section className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
      <div className="mb-3">
        <h2 className="text-[16px] font-semibold text-[#111111]">{title}</h2>
        <p className="mt-1 text-[13px] text-[#737373]">{description}</p>
      </div>
      <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap break-words rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-3 py-2 font-mono text-[12px] leading-6 text-[#111111]">
        {value.trim() === '' ? copy.settings.promptsRenderedEmpty : value}
      </pre>
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
