// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { Dispatch, FC, FormEvent, KeyboardEvent, ReactNode, RefObject, SetStateAction } from 'react';
import { useEffect, useRef, useState } from 'react';
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
  attachmentMenuOpen: boolean;
  ariaLabel: string;
  disabled: boolean;
  expanded: boolean;
  onActionClick: () => void;
  onAttachmentClick: () => void;
  onPhotoClick: () => void;
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
  attachmentAriaLabel: string;
  uploadDisabled: boolean;
  attachmentTitle: string;
  menuPhotoLabel: string;
  menuFileLabel: string;
  menuSkillLabel: string;
  menuUnavailableLabel: string;
  value: string;
}

const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX = 48;

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
  const composerShellRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const composingRef = useRef(false);
  const [inputExpanded, setInputExpanded] = useState(false);
  const [attachmentMenuOpen, setAttachmentMenuOpen] = useState(false);
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

  useAutosizeTextarea(textareaRef, value, setInputExpanded);
  useCloseAttachmentMenuOnOutsideClick({
    enabled: attachmentMenuOpen,
    onClose: () => setAttachmentMenuOpen(false),
    rootRef: composerShellRef,
  });

  useEffect(() => {
    if (uploadDisabled) {
      setAttachmentMenuOpen(false);
    }
  }, [uploadDisabled]);

  async function submit() {
    if (!canSend || sending || disabled) {
      return;
    }

    setAttachmentMenuOpen(false);
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

    setAttachmentMenuOpen((current) => !current);
  }

  function handlePhotoClick() {
    if (uploadDisabled) {
      return;
    }

    setAttachmentMenuOpen(false);
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
      <div ref={composerShellRef} className={buildComposerShellClassName(sending, disabled)}>
        {preview}

        <ComposerInputRow
          action={action}
          attachmentMenuOpen={attachmentMenuOpen}
          ariaLabel={effectiveAriaLabel}
          disabled={disabled}
          expanded={inputExpanded}
          onActionClick={handleActionClick}
          onAttachmentClick={handleAttachmentClick}
          onPhotoClick={handlePhotoClick}
          onChange={onChange}
          onCompositionEnd={() => {
            composingRef.current = false;
          }}
          onCompositionStart={() => {
            composingRef.current = true;
          }}
          onKeyDown={handleKeyDown}
          onTextareaInput={() => {
            syncTextareaHeight(textareaRef.current, setInputExpanded);
          }}
          placeholder={effectivePlaceholder}
          rows={rows}
          textareaRef={textareaRef}
          toolbar={toolbar}
          toolbarAriaLabel={copy.chat.composerToolsAria}
          attachmentAriaLabel={copy.chat.composerAddContent}
          uploadDisabled={uploadDisabled}
          attachmentTitle={copy.chat.composerAddContent}
          menuPhotoLabel={copy.chat.composerAttachmentPhoto}
          menuFileLabel={copy.chat.composerAttachmentFile}
          menuSkillLabel={copy.chat.composerAttachmentSkill}
          menuUnavailableLabel={copy.chat.composerAttachmentUnavailable}
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
  attachmentMenuOpen,
  ariaLabel,
  disabled,
  expanded,
  onActionClick,
  onAttachmentClick,
  onPhotoClick,
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
  attachmentAriaLabel,
  uploadDisabled,
  attachmentTitle,
  menuPhotoLabel,
  menuFileLabel,
  menuSkillLabel,
  menuUnavailableLabel,
  value,
}: ComposerInputRowProps) {
  const hasToolbar = toolbar !== undefined && toolbar !== null;

  return (
    <div className={`composer-row${expanded ? ' is-expanded' : ''}`}>
      <div className="composer-attachment-anchor">
        <button
          type="button"
          className="composer-plus-btn"
          aria-expanded={attachmentMenuOpen}
          aria-haspopup="menu"
          aria-label={attachmentAriaLabel}
          title={attachmentTitle}
          disabled={uploadDisabled}
          onClick={onAttachmentClick}
        >
          <PlusIcon />
        </button>

        {attachmentMenuOpen ? (
          <AttachmentMenu
            fileLabel={menuFileLabel}
            onPhotoClick={onPhotoClick}
            photoLabel={menuPhotoLabel}
            skillLabel={menuSkillLabel}
            unavailableLabel={menuUnavailableLabel}
          />
        ) : null}
      </div>

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

function AttachmentMenu({
  fileLabel,
  onPhotoClick,
  photoLabel,
  skillLabel,
  unavailableLabel,
}: {
  fileLabel: string;
  onPhotoClick: () => void;
  photoLabel: string;
  skillLabel: string;
  unavailableLabel: string;
}) {
  return (
    <div className="composer-attachment-menu" role="menu">
      <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onPhotoClick}>
        {photoLabel}
      </button>
      <DisabledAttachmentMenuItem label={fileLabel} unavailableLabel={unavailableLabel} />
      <DisabledAttachmentMenuItem label={skillLabel} unavailableLabel={unavailableLabel} />
    </div>
  );
}

function DisabledAttachmentMenuItem({
  label,
  unavailableLabel,
}: {
  label: string;
  unavailableLabel: string;
}) {
  return (
    <button
      type="button"
      className="composer-attachment-menu-item"
      role="menuitem"
      disabled
      title={unavailableLabel}
      aria-label={`${label}: ${unavailableLabel}`}
    >
      {label}
    </button>
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

function useAutosizeTextarea(
  textareaRef: RefObject<HTMLTextAreaElement>,
  value: string,
  setExpanded: Dispatch<SetStateAction<boolean>>,
) {
  useEffect(() => {
    syncTextareaHeight(textareaRef.current, setExpanded);
  }, [setExpanded, textareaRef, value]);
}

function useCloseAttachmentMenuOnOutsideClick({
  enabled,
  onClose,
  rootRef,
}: {
  enabled: boolean;
  onClose: () => void;
  rootRef: RefObject<HTMLElement>;
}) {
  useEffect(() => {
    if (!enabled || typeof document === 'undefined') {
      return;
    }

    function handlePointerDown(event: PointerEvent) {
      const root = rootRef.current;
      if (!root || !(event.target instanceof Node) || root.contains(event.target)) {
        return;
      }

      onClose();
    }

    document.addEventListener('pointerdown', handlePointerDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
    };
  }, [enabled, onClose, rootRef]);
}

function syncTextareaHeight(
  textarea: HTMLTextAreaElement | null,
  setExpanded?: Dispatch<SetStateAction<boolean>>,
) {
  if (!textarea) {
    return;
  }

  textarea.style.height = 'auto';
  const nextHeight = Math.min(textarea.scrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX);
  textarea.style.height = `${nextHeight}px`;
  const nextExpanded = nextHeight > COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX;
  setExpanded?.((current) => current === nextExpanded ? current : nextExpanded);
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
