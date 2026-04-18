// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { FC, KeyboardEvent, ReactNode } from 'react';
import { useEffect, useRef } from 'react';
import { ComposerMetaRow } from '@/components/ComposerMetaRow';
import { ComposerToolbar } from '@/components/ComposerToolbar';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';

interface ChatComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => Promise<void>;
  onStop?: () => Promise<void>;
  sending: boolean;
  canStop?: boolean;
  canSubmit?: boolean;
  disabled?: boolean;
  ariaLabel?: string;
  placeholder?: string;
  rows?: number;
  hint?: ReactNode;
  preview?: ReactNode;
  status?: ReactNode;
  toolbar?: ReactNode;
  onSelectFiles?: (files: FileList) => Promise<void> | void;
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
  canSubmit,
  disabled = false,
  ariaLabel,
  placeholder,
  rows = 1,
  hint,
  preview,
  status,
  toolbar,
  onSelectFiles,
}) => {
  const { copy } = useWebLocale();
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const effectiveAriaLabel = ariaLabel ?? copy.chat.composerMessageInputAria;
  const effectivePlaceholder = placeholder ?? copy.chat.composerInputPlaceholder;
  const hasToolbar = toolbar !== undefined && toolbar !== null;
  const showStopAction = sending && onStop !== undefined;
  const canSend = canSubmit ?? value.trim().length > 0;
  const submitDisabled = sending || disabled || !canSend;
  const actionDisabled = showStopAction ? !canStop : submitDisabled;
  const uploadDisabled = disabled || sending || onSelectFiles === undefined;
  const actionLabel = showStopAction
    ? canStop
      ? copy.chat.composerStopRun
      : copy.chat.composerStoppingRun
    : sending
      ? copy.chat.composerSending
      : copy.chat.composerSend;

  useEffect(() => {
    syncTextareaHeight();
  }, [value]);

  async function submit() {
    if (!canSend || sending || disabled) {
      return;
    }

    await onSubmit();
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

  function handleAttachmentClick() {
    if (uploadDisabled) {
      return;
    }

    fileInputRef.current?.click();
  }

  function handleFileChange(files: FileList | null) {
    if (!files || files.length === 0 || onSelectFiles === undefined) {
      return;
    }

    ignorePromise(Promise.resolve(onSelectFiles(files)));
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
        {preview}

        <textarea
          ref={textareaRef}
          value={value}
          disabled={disabled}
          aria-label={effectiveAriaLabel}
          onChange={(event) => onChange(event.target.value)}
          onInput={syncTextareaHeight}
          onKeyDown={handleKeyDown}
          placeholder={effectivePlaceholder}
          rows={rows}
          className="composer-textarea"
        />

        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          multiple
          hidden
          onChange={(event) => {
            handleFileChange(event.currentTarget.files);
            event.currentTarget.value = '';
          }}
        />

        <div className="composer-footer">
          <div
            className="composer-footer-start"
            role={hasToolbar ? 'toolbar' : undefined}
            aria-label={hasToolbar ? copy.chat.composerToolsAria : undefined}
          >
            <button
              type="button"
              className={`composer-plus-btn${uploadDisabled ? ' is-placeholder' : ''}`}
              aria-label={copy.chat.composerUploadImagesAria}
              title={copy.chat.composerUploadImagesTitle}
              disabled={uploadDisabled}
              onClick={handleAttachmentClick}
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
