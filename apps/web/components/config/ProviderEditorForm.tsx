'use client';

import type { FormEvent, ReactNode } from 'react';
import { ignorePromise } from '@/lib/errors';
import {
  type ProviderEditorState,
  defaultBaseURLForProviderType,
  providerTypeOptions,
} from '@/lib/configProviders';
import type { ProviderConfig } from '@/lib/types';

interface ProviderEditorFormProps {
  editorMode: 'create' | 'edit';
  editor: ProviderEditorState;
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<ProviderEditorState>) => void;
  onSelectProviderType: (providerType: ProviderConfig['type']) => void;
  onSubmit: () => Promise<void>;
  onCancelEditing: () => void;
}

interface FormFieldProps {
  label: string;
  description: string;
  children: ReactNode;
}

export function ProviderEditorForm(props: ProviderEditorFormProps) {
  const {
    editorMode,
    editor,
    controlsDisabled,
    saving,
    onChangeEditor,
    onSelectProviderType,
    onSubmit,
    onCancelEditing,
  } = props;
  const title = editorMode === 'edit' ? editor.name || 'Provider' : 'New Provider';
  const submitLabel = saving ? 'Saving...' : editorMode === 'edit' ? 'Save' : 'Create';

  return (
    <form className="space-y-8" onSubmit={(event) => handleProviderSubmit(event, onSubmit)}>
      <div>
        <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
          {editorMode === 'edit' ? 'Editing Connection' : 'New Connection'}
        </p>
        <h3 className="text-[20px] font-medium text-[#111111]">{title}</h3>
      </div>

      <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <div className="space-y-5">
          <FormField label="Provider Name" description="Unique identifier used in runtime config.">
            <input
              value={editor.name}
              disabled={controlsDisabled}
              onChange={(event) => onChangeEditor({ name: event.target.value })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>

          <FormField label="Provider Type" description="Driver contract for this endpoint.">
            <select
              value={editor.providerType}
              disabled={controlsDisabled}
              onChange={(event) => onSelectProviderType(event.target.value as ProviderConfig['type'])}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
            >
              {providerTypeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </FormField>

          <FormField label="Endpoint URL" description="Leave blank to use the provider default endpoint.">
            <input
              value={editor.baseURL}
              disabled={controlsDisabled}
              placeholder={
                editor.providerType === 'custom'
                  ? 'https://example.com/v1'
                  : defaultBaseURLForProviderType(editor.providerType)
              }
              onChange={(event) => onChangeEditor({ baseURL: event.target.value })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>

          <FormField label="API Key" description="Blank keeps existing key when editing.">
            <input
              type="password"
              value={editor.apiKey}
              disabled={controlsDisabled}
              placeholder={editorMode === 'edit' ? 'Leave blank to keep current key' : 'Enter API key'}
              onChange={(event) => onChangeEditor({ apiKey: event.target.value })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>

          <FormField label="Models" description="Comma-separated model IDs.">
            <input
              value={editor.models}
              disabled={controlsDisabled}
              onChange={(event) => onChangeEditor({ models: event.target.value })}
              placeholder="gpt-4o, claude-3-7-sonnet-latest"
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>
        </div>
      </div>

      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onCancelEditing}
          className="rounded-full border border-[#E5E5E5] px-6 py-2.5 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={controlsDisabled}
          className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {submitLabel}
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

function handleProviderSubmit(event: FormEvent<HTMLFormElement>, onSubmit: () => Promise<void>) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
