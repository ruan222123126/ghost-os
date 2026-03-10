// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { FC, KeyboardEvent, ReactNode } from 'react';
import { useEffect, useRef } from 'react';
import { ComposerMetaRow } from '@/components/ComposerMetaRow';
import { ComposerToolbar } from '@/components/ComposerToolbar';
import { ignorePromise } from '@/lib/errors';

interface ChatComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (value: string) => Promise<void>;
  onStop?: () => Promise<void>;
  sending: boolean;
  canStop?: boolean;
  disabled?: boolean;
  ariaLabel?: string;
  placeholder?: string;
  rows?: number;
  hint?: ReactNode;
  status?: ReactNode;
  toolbar?: ReactNode;
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M10 4.5V15.5" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M4.5 10H15.5" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

function SendIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M10 4.25V15.75" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M5.75 8.5L10 4.25L14.25 8.5" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export const ChatComposer: FC<ChatComposerProps> = ({
  value,
  onChange,
  onSubmit,
  onStop,
  sending,
  canStop = false,
  disabled = false,
  ariaLabel = 'Message input',
  placeholder = 'Type a task for Ghost-OS...',
  hint,
  status,
  toolbar,
}) => {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const hasToolbar = toolbar !== undefined && toolbar !== null;
  const showStopAction = sending && onStop !== undefined;
  const submitDisabled = sending || disabled || value.trim().length === 0;
  const actionDisabled = showStopAction ? !canStop : submitDisabled;
  const actionLabel = showStopAction
    ? canStop
      ? 'Stop agent run'
      : 'Stopping agent run'
    : sending
      ? 'Sending message'
      : 'Send message';

  useEffect(() => {
    syncTextareaHeight();
  }, [value]);

  async function submit() {
    const nextValue = value.trim();
    if (!nextValue || sending || disabled) {
      return;
    }

    await onSubmit(nextValue);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (
      event.key !== 'Enter'
      || event.shiftKey
      || event.ctrlKey
      || event.metaKey
      || event.altKey
      || event.nativeEvent.isComposing
    ) {
      return;
    }

    event.preventDefault();
    ignorePromise(submit());
  }

  function syncTextareaHeight() {
    const textarea = textareaRef.current;
    if (!textarea) {
      return;
    }

    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
  }

  return (
    <form
      className="composer"
      onSubmit={(event) => {
        event.preventDefault();
        ignorePromise(submit());
      }}
    >
      <div className={`composer-shell${sending ? ' is-sending' : ''}${disabled ? ' is-disabled' : ''}`}>
        <textarea
          ref={textareaRef}
          value={value}
          disabled={disabled}
          aria-label={ariaLabel}
          onChange={(event) => onChange(event.target.value)}
          onInput={syncTextareaHeight}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          rows={1}
          className="composer-textarea"
        />

        <div className="composer-footer">
          <div
            className="composer-footer-start"
            role={hasToolbar ? 'toolbar' : undefined}
            aria-label={hasToolbar ? 'Composer tools' : undefined}
          >
            <button
              type="button"
              className="composer-plus-btn is-placeholder"
              aria-label="Attachments not available in this view"
              title="Attachments not available in this view"
              disabled
            >
              <PlusIcon />
            </button>

            <ComposerToolbar>{toolbar}</ComposerToolbar>
          </div>

          <button
            type={showStopAction ? 'button' : 'submit'}
            disabled={actionDisabled}
            onClick={showStopAction
              ? () => {
                ignorePromise(onStop());
              }
              : undefined}
            className={`composer-send-btn${showStopAction ? ' is-stop' : ''}`}
            aria-label={actionLabel}
          >
            <span className="sr-only">{actionLabel}</span>
            {showStopAction ? <span className="composer-stop-glyph" aria-hidden="true" /> : <SendIcon />}
          </button>
        </div>
      </div>

      <ComposerMetaRow hint={hint} status={status} />
    </form>
  );
};
