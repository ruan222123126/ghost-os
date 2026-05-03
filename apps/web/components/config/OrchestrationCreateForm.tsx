'use client';

import type { FormEvent, ReactNode } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';

interface OrchestrationCreateFormProps {
  name: string;
  controlsDisabled: boolean;
  saving: boolean;
  onChangeName: (value: string) => void;
  onSubmit: () => Promise<void>;
  onCancel: () => void;
}

interface FormFieldProps {
  label: string;
  description: string;
  children: ReactNode;
}

export function OrchestrationCreateForm(props: OrchestrationCreateFormProps) {
  const { copy } = useWebLocale();
  const {
    name,
    controlsDisabled,
    saving,
    onChangeName,
    onSubmit,
    onCancel,
  } = props;

  return (
    <form className="space-y-8" onSubmit={(event) => handleSubmit(event, onSubmit)}>
      <div>
        <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
          {copy.settings.orchestrationNew}
        </p>
        <h3 className="text-[20px] font-medium text-[#111111]">
          {copy.settings.orchestrationCreateTitle}
        </h3>
      </div>

      <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <FormField
          label={copy.settings.orchestrationNameLabel}
          description={copy.settings.orchestrationCreateDescription}
        >
          <input
            value={name}
            disabled={controlsDisabled}
            onChange={(event) => onChangeName(event.target.value)}
            placeholder={copy.settings.orchestrationNamePlaceholder}
            className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
          />
        </FormField>
      </div>

      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-full border border-[#E5E5E5] px-6 py-2.5 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
        >
          {copy.settings.cancel}
        </button>
        <button
          type="submit"
          disabled={controlsDisabled}
          className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? copy.settings.saving : copy.settings.orchestrationCreateSubmit}
        </button>
      </div>
    </form>
  );
}

function FormField(props: FormFieldProps) {
  const { label, description, children } = props;

  return (
    <label className="block">
      <span className="mb-1 block text-[13px] font-medium text-[#111111]">{label}</span>
      <span className="mb-2 block text-[12px] text-[#737373]">{description}</span>
      {children}
    </label>
  );
}

function handleSubmit(event: FormEvent<HTMLFormElement>, onSubmit: () => Promise<void>) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
