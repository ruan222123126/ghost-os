// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { Dispatch, FC, FormEvent, KeyboardEvent, ReactNode, RefObject, SetStateAction } from 'react';
import { useEffect, useRef, useState } from 'react';
import { ComposerInputRow, type ComposerActionState } from '@/components/ChatComposerInputRow';
import { ComposerMetaRow } from '@/components/ComposerMetaRow';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ChatSelectedSkill, SkillPayload } from '@/lib/types';

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
  selectedSkill?: ChatSelectedSkill | null;
  status?: ReactNode;
  skillError?: string;
  skills?: SkillPayload[];
  skillsLoading?: boolean;
  toolbar?: ReactNode;
  onClearSelectedSkill?: () => void;
  onSelectFiles?: (files: FileList) => Promise<void> | void;
  onRefreshSkills?: () => Promise<void> | void;
  onSelectSkill?: (skill: SkillPayload) => void;
}

const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX = 48;

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
  selectedSkill = null,
  status,
  skillError = '',
  skills = [],
  skillsLoading = false,
  toolbar,
  onClearSelectedSkill,
  onSelectFiles,
  onRefreshSkills,
  onSelectSkill,
}) => {
  const { copy } = useWebLocale();
  const composerShellRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const composingRef = useRef(false);
  const [inputExpanded, setInputExpanded] = useState(false);
  const [attachmentMenuOpen, setAttachmentMenuOpen] = useState(false);
  const [skillMenuOpen, setSkillMenuOpen] = useState(false);
  const effectiveAriaLabel = ariaLabel ?? copy.chat.composerMessageInputAria;
  const effectivePlaceholder = placeholder ?? copy.chat.composerInputPlaceholder;
  const canSend = canSubmit ?? value.trim().length > 0;
  const uploadDisabled = disabled || sending || onSelectFiles === undefined;
  const attachmentDisabled = disabled || sending || (onSelectFiles === undefined && onSelectSkill === undefined);
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
    enabled: attachmentMenuOpen || skillMenuOpen,
    onClose: () => {
      setAttachmentMenuOpen(false);
      setSkillMenuOpen(false);
    },
    rootRef: composerShellRef,
  });

  useEffect(() => {
    if (attachmentDisabled) {
      setAttachmentMenuOpen(false);
      setSkillMenuOpen(false);
    }
  }, [attachmentDisabled]);

  async function submit() {
    if (!canSend || sending || disabled) {
      return;
    }

    setAttachmentMenuOpen(false);
    setSkillMenuOpen(false);
    await onSubmit();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Backspace' && value === '' && selectedSkill && onClearSelectedSkill) {
      event.preventDefault();
      setSkillMenuOpen(false);
      onClearSelectedSkill();
      return;
    }

    if (!shouldSubmitOnEnter(event, composingRef.current)) {
      return;
    }

    event.preventDefault();
    ignorePromise(submit());
  }

  function handleAttachmentClick() {
    if (attachmentDisabled) {
      return;
    }

    setAttachmentMenuOpen((current) => {
      const nextOpen = !current;
      if (nextOpen) {
        setSkillMenuOpen(false);
      }
      return nextOpen;
    });
  }

  function handlePhotoClick() {
    if (uploadDisabled) {
      return;
    }

    setAttachmentMenuOpen(false);
    fileInputRef.current?.click();
  }

  function handleSkillClick() {
    if (disabled || sending || onSelectSkill === undefined) {
      return;
    }

    setAttachmentMenuOpen(false);
    setSkillMenuOpen(true);
    ignorePromise(Promise.resolve(onRefreshSkills?.()));
    textareaRef.current?.focus();
  }

  function handleSelectSkill(skill: SkillPayload) {
    onSelectSkill?.(skill);
    setSkillMenuOpen(false);
    setAttachmentMenuOpen(false);
    textareaRef.current?.focus();
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
          onClearSelectedSkill={onClearSelectedSkill}
          onActionClick={handleActionClick}
          onAttachmentClick={handleAttachmentClick}
          onPhotoClick={handlePhotoClick}
          onRefreshSkills={onRefreshSkills}
          onSelectSkill={onSelectSkill ? handleSelectSkill : undefined}
          onSkillClick={handleSkillClick}
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
          placeholder={selectedSkill ? '' : effectivePlaceholder}
          rows={rows}
          selectedSkill={selectedSkill}
          skillError={skillError}
          skillMenuEmptyLabel={copy.chat.composerSkillMenuEmpty}
          skillMenuLoadingLabel={copy.chat.composerSkillMenuLoading}
          skillMenuOpen={skillMenuOpen}
          skillMenuRefreshLabel={copy.chat.composerSkillMenuRefresh}
          skillMenuTitle={copy.chat.composerSkillMenuTitle}
          skills={skills}
          skillsLoading={skillsLoading}
          textareaRef={textareaRef}
          toolbar={toolbar}
          toolbarAriaLabel={copy.chat.composerToolsAria}
          attachmentAriaLabel={copy.chat.composerAddContent}
          attachmentDisabled={attachmentDisabled}
          photoDisabled={uploadDisabled}
          attachmentTitle={copy.chat.composerAddContent}
          menuPhotoLabel={copy.chat.composerAttachmentPhoto}
          menuSkillLabel={copy.chat.composerAttachmentSkill}
          selectedSkillClearLabel={selectedSkill ? copy.chat.composerClearSelectedSkill(selectedSkill.name) : undefined}
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
