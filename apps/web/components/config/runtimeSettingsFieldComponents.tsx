import type { ReactNode } from 'react';
import { SoftDropdownSelect, type DropdownSelectOption } from '@/components/SoftDropdownSelect';
import { useWebLocale } from '@/lib/i18n/provider';

export function Card(props: { title: string; copy: string; children: ReactNode }) {
  const { title, copy, children } = props;

  return (
    <section className="border-b border-[#E5E5E5] py-8">
      <div className="mb-6">
        <h3 className="text-[18px] font-semibold tracking-tight text-[#111111]">{title}</h3>
        <p className="mt-1 text-[13px] text-[#737373]">{copy}</p>
      </div>
      <div className="space-y-5 border-l-2 border-[#E5E5E5]/80 pl-5">{children}</div>
    </section>
  );
}

export function TextField(props: {
  label: string;
  description?: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: 'text' | 'password' | 'number';
  mono?: boolean;
}) {
  const { label, description, value, disabled, onChange, placeholder, type = 'text', mono = false } = props;
  const className = 'w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none' + (mono ? ' font-mono' : '');

  return (
    <Field label={label} description={description}>
      <input
        type={type}
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        className={className}
      />
    </Field>
  );
}

export function TextAreaField(props: {
  label: string;
  description?: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
  placeholder?: string;
}) {
  const { label, description, value, disabled, onChange, placeholder } = props;

  return (
    <Field label={label} description={description}>
      <textarea
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        rows={8}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 font-mono text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
      />
    </Field>
  );
}

export function ToggleField(props: {
  label: string;
  description?: string;
  checked: boolean;
  disabled: boolean;
  onChange: (checked: boolean) => void;
}) {
  const { label, description, checked, disabled, onChange } = props;
  const { copy } = useWebLocale();

  return (
    <Field label={label} description={description}>
      <label className="inline-flex items-center gap-2 text-[13px] text-[#111111]">
        <input
          type="checkbox"
          checked={checked}
          disabled={disabled}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span>{checked ? copy.settings.runtimeToggleEnabled : copy.settings.runtimeToggleDisabled}</span>
      </label>
    </Field>
  );
}

export function SelectField(props: {
  label: string;
  description?: string;
  value: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  options: readonly DropdownSelectOption[];
  testId?: string;
}) {
  const { label, description, value, disabled = false, onChange, options, testId } = props;

  return (
    <Field label={label} description={description}>
      <SoftDropdownSelect
        testId={testId}
        value={value}
        disabled={disabled}
        onChange={onChange}
        options={options}
      />
    </Field>
  );
}

function Field(props: { label: string; description?: string; children: ReactNode }) {
  const { label, description, children } = props;
  const showDescription = description !== undefined && description.trim() !== '';

  return (
    <label className="block">
      <span className="mb-1 block text-[13px] font-medium text-[#111111]">{label}</span>
      {showDescription ? <span className="mb-2 block text-[12px] text-[#737373]">{description}</span> : null}
      {children}
    </label>
  );
}
