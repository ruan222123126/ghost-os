// ChatComposer provides the React-based shell for the web console message composer.

'use client';

import type { Dispatch, FC, FormEvent, KeyboardEvent, ReactNode, RefObject, SetStateAction } from 'react';
import { useEffect, useRef, useState } from 'react';
import { ComposerInputRow, type ComposerActionState } from '@/components/ChatComposerInputRow';
import { ComposerMetaRow } from '@/components/ComposerMetaRow';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentModeSelection, ChatSelectedSkill, SkillPayload } from '@/lib/types';

interface ChatComposerProps {
  agentMode: AgentModeSelection;
  canEnableCodexMode?: boolean;
  codexModeDisabledMessage?: string;
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
  onChangeAgentMode?: (mode: AgentModeSelection) => void;
}

const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX = 48;
const COMPOSER_ROW_COMPACT_COLUMN_GAPS = 2;
const COMPOSER_ROW_COLUMN_GAP_PX = 4;
type ComposerMenuView = 'attachment' | 'features' | 'skills' | null;

export const ChatComposer: FC<ChatComposerProps> = ({
  agentMode,
  canEnableCodexMode = false,
  codexModeDisabledMessage,
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
  onChangeAgentMode,
}) => {
  const { copy } = useWebLocale();
  const composerShellRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const composingRef = useRef(false);
  const [inputExpanded, setInputExpanded] = useState(false);
  const [menuView, setMenuView] = useState<ComposerMenuView>(null);
  const effectiveAriaLabel = ariaLabel ?? copy.chat.composerMessageInputAria;
  const effectivePlaceholder = placeholder ?? copy.chat.composerInputPlaceholder;
  const canSend = canSubmit ?? value.trim().length > 0;
  const attachmentMenuOpen = menuView === 'attachment';
  const featureMenuOpen = menuView === 'features';
  const skillMenuOpen = menuView === 'skills';
  const agentModeActive = agentMode !== null;
  const codexModeToggleEnabled = agentModeActive || canEnableCodexMode;
  const codexModeDescription = agentMode === 'normal'
    ? copy.chat.composerCodexNormalEnabled
    : agentMode === 'plan'
    ? copy.chat.composerCodexPlanEnabled
    : codexModeToggleEnabled
    ? copy.chat.composerCodexInactive
    : codexModeDisabledMessage ?? copy.chat.composerCodexUnavailable;
  const fileDisabled = disabled || sending || onSelectFiles === undefined;
  const attachmentDisabled = disabled || sending || (
    onSelectFiles === undefined
    && onSelectSkill === undefined
    && onChangeAgentMode === undefined
  );
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
    enabled: menuView !== null,
    onClose: () => setMenuView(null),
    rootRef: composerShellRef,
  });

  useEffect(() => {
    if (attachmentDisabled) {
      setMenuView(null);
    }
  }, [attachmentDisabled]);

  async function submit() {
    if (!canSend || sending || disabled) {
      return;
    }

    setMenuView(null);
    const submitPromise = onSubmit();
    focusTextareaAfterSubmit(textareaRef.current);
    await submitPromise;
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Backspace' && value === '' && selectedSkill && onClearSelectedSkill) {
      event.preventDefault();
      setMenuView(null);
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

    setMenuView((current) => current === 'attachment' ? null : 'attachment');
  }

  function handleFileClick() {
    if (fileDisabled) {
      return;
    }

    setMenuView(null);
    fileInputRef.current?.click();
  }

  function handleSkillClick() {
    if (disabled || sending || onSelectSkill === undefined) {
      return;
    }

    setMenuView('skills');
    ignorePromise(Promise.resolve(onRefreshSkills?.()));
    textareaRef.current?.focus();
  }

  function handleFeatureClick() {
    if (disabled || sending || onChangeAgentMode === undefined) {
      return;
    }

    setMenuView('features');
    textareaRef.current?.focus();
  }

  function handleSelectSkill(skill: SkillPayload) {
    onSelectSkill?.(skill);
    setMenuView(null);
    textareaRef.current?.focus();
  }

  function handleChangeAgentMode(mode: AgentModeSelection) {
    if (!onChangeAgentMode) {
      textareaRef.current?.focus();
      return;
    }

    if (mode !== null && mode !== agentMode && !canEnableCodexMode) {
      return;
    }

    onChangeAgentMode(mode);
    setMenuView(null);
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
          agentMode={agentMode}
          attachmentMenuOpen={attachmentMenuOpen}
          ariaLabel={effectiveAriaLabel}
          codexModeDescription={codexModeDescription}
          codexModeLabel={copy.chat.composerCodexModeBadge}
          codexModeTitle={copy.chat.composerCodexModeTitle}
          codexToggleEnabled={codexModeToggleEnabled}
          disabled={disabled}
          expanded={inputExpanded}
          featureMenuOpen={featureMenuOpen}
          featureMenuTitle={copy.chat.composerFeatureMenuTitle}
          onClearSelectedSkill={onClearSelectedSkill}
          onActionClick={handleActionClick}
          onAttachmentClick={handleAttachmentClick}
          onFileClick={handleFileClick}
          onRefreshSkills={onRefreshSkills}
          onSelectSkill={onSelectSkill ? handleSelectSkill : undefined}
          onChangeAgentMode={onChangeAgentMode ? handleChangeAgentMode : undefined}
          onFeatureClick={handleFeatureClick}
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
          fileDisabled={fileDisabled}
          attachmentTitle={copy.chat.composerAddContent}
          menuFileLabel={copy.chat.composerAttachmentFile}
          menuFeatureLabel={copy.chat.composerAttachmentFeature}
          normalModeLabel={copy.chat.composerRuntimeNormal}
          planModeLabel={copy.chat.composerRuntimePlan}
          menuSkillLabel={copy.chat.composerAttachmentSkill}
          selectedSkillClearLabel={selectedSkill ? copy.chat.composerClearSelectedSkill(selectedSkill.name) : undefined}
          menuUnavailableLabel={copy.chat.composerAttachmentUnavailable}
          fileUnavailableLabel={copy.chat.composerAttachmentUnavailable}
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
  const currentScrollHeight = textarea.scrollHeight;
  const compactScrollHeight = measureTextareaCompactScrollHeight(textarea) ?? currentScrollHeight;
  const nextHeight = Math.min(currentScrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX);
  textarea.style.height = `${nextHeight}px`;
  const textareaValue = typeof textarea.value === 'string' ? textarea.value : '';
  const nextExpanded = textareaValue.length > 0
    && (
      textareaValue.includes('\n')
      || compactScrollHeight > COMPOSER_TEXTAREA_EXPANDED_HEIGHT_PX
    );
  setExpanded?.((current) => current === nextExpanded ? current : nextExpanded);
}

function measureTextareaCompactScrollHeight(textarea: HTMLTextAreaElement): number | null {
  const compactWidth = getCompactTextareaWidth(textarea);
  if (compactWidth === null) {
    return null;
  }

  const previousWidth = textarea.style.width;
  textarea.style.width = `${compactWidth}px`;
  textarea.style.height = 'auto';
  const compactScrollHeight = textarea.scrollHeight;
  textarea.style.width = previousWidth;
  return compactScrollHeight;
}

function getCompactTextareaWidth(textarea: HTMLTextAreaElement): number | null {
  if (typeof textarea.closest !== 'function') {
    return null;
  }

  const row = textarea.closest('.composer-row') as HTMLElement | null;
  if (!row) {
    return null;
  }

  const attachment = row.querySelector<HTMLElement>('.composer-attachment-anchor');
  const actions = row.querySelector<HTMLElement>('.composer-actions');
  if (!attachment || !actions) {
    return null;
  }

  const rowWidth = getElementWidth(row);
  if (rowWidth <= 0) {
    return null;
  }

  const reservedWidth = getElementWidth(attachment)
    + getElementWidth(actions)
    + getColumnGapPx(row) * COMPOSER_ROW_COMPACT_COLUMN_GAPS;
  const compactWidth = rowWidth - reservedWidth;
  return compactWidth > 0 ? compactWidth : null;
}

function getElementWidth(element: HTMLElement): number {
  const rect = typeof element.getBoundingClientRect === 'function'
    ? element.getBoundingClientRect()
    : null;
  const rectWidth = rect?.width ?? 0;
  if (rectWidth > 0) {
    return rectWidth;
  }

  return element.clientWidth || element.offsetWidth || 0;
}

function getColumnGapPx(element: HTMLElement): number {
  if (typeof window === 'undefined' || typeof window.getComputedStyle !== 'function') {
    return COMPOSER_ROW_COLUMN_GAP_PX;
  }

  const columnGap = window.getComputedStyle(element).columnGap;
  const parsedGap = Number.parseFloat(columnGap);
  return Number.isFinite(parsedGap) ? parsedGap : COMPOSER_ROW_COLUMN_GAP_PX;
}

function focusTextareaAfterSubmit(textarea: HTMLTextAreaElement | null) {
  textarea?.focus({ preventScroll: true });
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
