// Shared textarea + submit control used by chat and question answer inputs.

'use client';

import type { FC, ReactNode } from 'react';
import { useState } from 'react';
import { ignorePromise } from '@/lib/errors';

interface TextareaSubmitInputProps {
  loading: boolean;
  disabled?: boolean;
  onSubmit: (value: string) => Promise<void>;
  ariaLabel: string;
  placeholder: string;
  rows: number;
  hint: ReactNode;
  submitLabel: string;
  submittingLabel: string;
  containerClassName: string;
  textareaClassName: string;
  actionsClassName: string;
  hintClassName: string;
  buttonClassName: string;
}

export const TextareaSubmitInput: FC<TextareaSubmitInputProps> = ({
  loading,
  disabled = false,
  onSubmit,
  ariaLabel,
  placeholder,
  rows,
  hint,
  submitLabel,
  submittingLabel,
  containerClassName,
  textareaClassName,
  actionsClassName,
  hintClassName,
  buttonClassName,
}) => {
  const [value, setValue] = useState('');

  async function submit() {
    const nextValue = value.trim();
    if (!nextValue || loading || disabled) {
      return;
    }

    setValue('');
    await onSubmit(nextValue);
  }

  const submitDisabled = loading || disabled || value.trim().length === 0;

  return (
    <div className={containerClassName}>
      <textarea
        value={value}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            ignorePromise(submit());
          }
        }}
        placeholder={placeholder}
        rows={rows}
        className={textareaClassName}
      />
      <div className={actionsClassName}>
        <span className={hintClassName}>{hint}</span>
        <button
          type="button"
          onClick={() => {
            ignorePromise(submit());
          }}
          disabled={submitDisabled}
          className={buttonClassName}
        >
          {loading ? submittingLabel : submitLabel}
        </button>
      </div>
    </div>
  );
};
