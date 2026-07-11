import type { ReactNode } from 'react';
import { SoftDropdownSelect, type DropdownSelectOption } from '@/components/SoftDropdownSelect';
import { useWebLocale } from '@/lib/i18n/provider';

export function Card(props: { title: string; copy: string; children: ReactNode }) {
  const { title, copy, children } = props;

  return (
    <section className="mb-10 border-b border-gray-100 pb-10 last:mb-0 last:border-b-0 last:pb-0">
      <div className="mb-5">
        <h3 className="mb-1 text-lg font-bold text-gray-900">{title}</h3>
        <p className="text-sm text-gray-500">{copy}</p>
      </div>
      <div className="space-y-5">{children}</div>
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
  const className = 'block w-full rounded-lg border border-gray-200 bg-gray-50 px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 transition-colors focus:bg-white focus:outline-none focus:ring-2 focus:ring-gray-900 disabled:cursor-not-allowed disabled:opacity-60' + (mono ? ' font-mono' : '');

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
      <div className="flex items-center space-x-3">
        <button
          type="button"
          role="switch"
          aria-checked={checked}
          aria-label={label}
          disabled={disabled}
          className={`${checked ? 'bg-gray-900' : 'bg-gray-200'} relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60`}
          onClick={() => onChange(!checked)}
        >
          <span
            aria-hidden="true"
            className={`${checked ? 'translate-x-4' : 'translate-x-0'} pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out`}
          />
        </button>
        <span className={`text-sm font-medium ${checked ? 'text-gray-900' : 'text-gray-500'}`}>
          {checked ? copy.settings.runtimeToggleEnabled : copy.settings.runtimeToggleDisabled}
        </span>
      </div>
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
    <div className="block">
      <span className="mb-1 block text-sm text-gray-700">{label}</span>
      {showDescription ? <span className="mb-2 block text-xs text-gray-400">{description}</span> : null}
      {children}
    </div>
  );
}
