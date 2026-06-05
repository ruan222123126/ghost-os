// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { FC, FormEvent, KeyboardEvent, ReactNode, RefObject } from 'react';
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

interface ComposerActionState {
  disabled: boolean;
  label: string;
  showStop: boolean;
  visible: boolean;
}

interface ComposerInputRowProps {
  action: ComposerActionState;
  ariaLabel: string;
  disabled: boolean;
  onActionClick: () => void;
  onAttachmentClick: () => void;
  onChange: (value: string) => void;
  onCompositionEnd: () => void;
  onCompositionStart: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  onTextareaInput: () => void;
  placeholder: string;
  rows: number;
  textareaRef: RefObject<HTMLTextAreaElement>;
  toolbar?: ReactNode;
  toolbarAriaLabel: string;
  uploadAriaLabel: string;
  uploadDisabled: boolean;
  uploadTitle: string;
  value: string;
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
  const composingRef = useRef(false);
  const effectiveAriaLabel = ariaLabel ?? copy.chat.composerMessageInputAria;
  const effectivePlaceholder = placeholder ?? copy.chat.composerInputPlaceholder;
  const canSend = canSubmit ?? value.trim().length > 0;
  const uploadDisabled = disabled || sending || onSelectFiles === undefined;
  const action = buildComposerActionState({
    canSend,
    canStop,
    disabled,
    hasStopHandler: onStop !== undefined,
    sending,
    sendingLabel: copy.chat.composerSending,
    sendLabel: copy.chat.composerSend,
    stopLabel: copy.chat.composerStopRun,
    stoppingLabel: copy.chat.composerStoppingRun,
  });

  useAutosizeTextarea(textareaRef, value);

  async function submit() {
    if (!canSend || sending || disabled) {
      return;
    }

    await onSubmit();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (!shouldSubmitOnEnter(event, composingRef.current)) {
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

  function handleActionClick() {
    if (onStop === undefined) {
      return;
    }

    ignorePromise(onStop());
  }

  function handleFileChange(files: FileList | null) {
    if (!files || files.length === 0 || onSelectFiles === undefined) {
      return;
    }

    ignorePromise(Promise.resolve(onSelectFiles(files)));
  }

  function handleFormSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    ignorePromise(submit());
  }

  return (
    <form className="composer" onSubmit={handleFormSubmit}>
      <div className={buildComposerShellClassName(sending, disabled)}>
        {preview}

        <ComposerInputRow
          action={action}
          ariaLabel={effectiveAriaLabel}
          disabled={disabled}
          onActionClick={handleActionClick}
          onAttachmentClick={handleAttachmentClick}
          onChange={onChange}
          onCompositionEnd={() => {
            composingRef.current = false;
          }}
          onCompositionStart={() => {
            composingRef.current = true;
          }}
          onKeyDown={handleKeyDown}
          onTextareaInput={() => {
            syncTextareaHeight(textareaRef.current);
          }}
          placeholder={effectivePlaceholder}
          rows={rows}
          textareaRef={textareaRef}
          toolbar={toolbar}
          toolbarAriaLabel={copy.chat.composerToolsAria}
          uploadAriaLabel={copy.chat.composerUploadImagesAria}
          uploadDisabled={uploadDisabled}
          uploadTitle={copy.chat.composerUploadImagesTitle}
          value={value}
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
      </div>

      <ComposerMetaRow hint={hint} status={status} />
    </form>
  );
};

function ComposerInputRow({
  action,
  ariaLabel,
  disabled,
  onActionClick,
  onAttachmentClick,
  onChange,
  onCompositionEnd,
  onCompositionStart,
  onKeyDown,
  onTextareaInput,
  placeholder,
  rows,
  textareaRef,
  toolbar,
  toolbarAriaLabel,
  uploadAriaLabel,
  uploadDisabled,
  uploadTitle,
  value,
}: ComposerInputRowProps) {
  const hasToolbar = toolbar !== undefined && toolbar !== null;

  return (
    <div className="composer-row">
      <button
        type="button"
        className="composer-plus-btn"
        aria-label={uploadAriaLabel}
        title={uploadTitle}
        disabled={uploadDisabled}
        onClick={onAttachmentClick}
      >
        <PlusIcon />
      </button>

      <textarea
        ref={textareaRef}
        value={value}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={(event) => onChange(event.target.value)}
        onInput={onTextareaInput}
        onKeyDown={onKeyDown}
        onCompositionStart={onCompositionStart}
        onCompositionEnd={onCompositionEnd}
        placeholder={placeholder}
        rows={rows}
        className="composer-textarea"
      />

      <div className="composer-actions">
        <div
          className="composer-toolbar"
          role={hasToolbar ? 'toolbar' : undefined}
          aria-label={hasToolbar ? toolbarAriaLabel : undefined}
        >
          <ComposerToolbar>{toolbar}</ComposerToolbar>
        </div>
        <ComposerActionButton action={action} onClick={onActionClick} />
      </div>
    </div>
  );
}

function ComposerActionButton({
  action,
  onClick,
}: {
  action: ComposerActionState;
  onClick: () => void;
}) {
  if (!action.visible) {
    return null;
  }

  return (
    <button
      type={action.showStop ? 'button' : 'submit'}
      disabled={action.disabled}
      onClick={action.showStop ? onClick : undefined}
      className={`composer-send-btn${action.showStop ? ' is-stop' : ''}`}
      aria-label={action.label}
    >
      <span className="sr-only">{action.label}</span>
      {action.showStop ? <span className="composer-stop-glyph" aria-hidden="true" /> : <SendIcon />}
    </button>
  );
}

function useAutosizeTextarea(textareaRef: RefObject<HTMLTextAreaElement>, value: string) {
  useEffect(() => {
    syncTextareaHeight(textareaRef.current);
  }, [textareaRef, value]);
}

function syncTextareaHeight(textarea: HTMLTextAreaElement | null) {
  if (!textarea) {
    return;
  }

  textarea.style.height = 'auto';
  textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
}

function buildComposerShellClassName(sending: boolean, disabled: boolean): string {
  return `composer-shell${sending ? ' is-sending' : ''}${disabled ? ' is-disabled' : ''}`;
}

function buildComposerActionState({
  canSend,
  canStop,
  disabled,
  hasStopHandler,
  sending,
  sendingLabel,
  sendLabel,
  stopLabel,
  stoppingLabel,
}: {
  canSend: boolean;
  canStop: boolean;
  disabled: boolean;
  hasStopHandler: boolean;
  sending: boolean;
  sendingLabel: string;
  sendLabel: string;
  stopLabel: string;
  stoppingLabel: string;
}): ComposerActionState {
  const showStop = sending && hasStopHandler;
  if (showStop) {
    return {
      disabled: !canStop,
      label: canStop ? stopLabel : stoppingLabel,
      showStop: true,
      visible: true,
    };
  }

  return {
    disabled: sending || disabled || !canSend,
    label: sending ? sendingLabel : sendLabel,
    showStop: false,
    visible: canSend,
  };
}

interface ComposerNativeKeyboardEvent {
  isComposing?: boolean;
  keyCode?: number;
  which?: number;
}

function shouldSubmitOnEnter(
  event: KeyboardEvent<HTMLTextAreaElement>,
  composing: boolean,
): boolean {
  if (
    event.key !== 'Enter'
    || event.shiftKey
    || event.ctrlKey
    || event.metaKey
    || event.altKey
  ) {
    return false;
  }

  const nativeEvent = event.nativeEvent as ComposerNativeKeyboardEvent;
  return !composing
    && nativeEvent.isComposing !== true
    && nativeEvent.keyCode !== 229
    && nativeEvent.which !== 229;
}
