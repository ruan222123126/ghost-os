'use client';

import type { FC, KeyboardEvent, ReactNode, RefObject } from 'react';
import { ComposerSkillMenu } from '@/components/ComposerSkillMenu';
import { ComposerToolbar } from '@/components/ComposerToolbar';
import { ignorePromise } from '@/lib/errors';
import type { ChatSelectedSkill, SkillPayload } from '@/lib/types';

export interface ComposerActionState {
  disabled: boolean;
  label: string;
  showStop: boolean;
  visible: boolean;
}

export interface ComposerInputRowProps {
  action: ComposerActionState;
  attachmentMenuOpen: boolean;
  ariaLabel: string;
  disabled: boolean;
  expanded: boolean;
  onClearSelectedSkill?: () => void;
  onActionClick: () => void;
  onAttachmentClick: () => void;
  onPhotoClick: () => void;
  onRefreshSkills?: () => Promise<void> | void;
  onSelectSkill?: (skill: SkillPayload) => void;
  onSkillClick: () => void;
  onChange: (value: string) => void;
  onCompositionEnd: () => void;
  onCompositionStart: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  onTextareaInput: () => void;
  placeholder: string;
  rows: number;
  selectedSkill?: ChatSelectedSkill | null;
  skillError: string;
  skillMenuOpen: boolean;
  skillMenuEmptyLabel: string;
  skillMenuLoadingLabel: string;
  skillMenuRefreshLabel: string;
  skillMenuTitle: string;
  skills: SkillPayload[];
  skillsLoading: boolean;
  textareaRef: RefObject<HTMLTextAreaElement>;
  toolbar?: ReactNode;
  toolbarAriaLabel: string;
  attachmentAriaLabel: string;
  attachmentDisabled: boolean;
  photoDisabled: boolean;
  attachmentTitle: string;
  menuPhotoLabel: string;
  menuFileLabel: string;
  menuSkillLabel: string;
  selectedSkillClearLabel?: string;
  menuUnavailableLabel: string;
  value: string;
}

export const ComposerInputRow: FC<ComposerInputRowProps> = ({
  action,
  attachmentMenuOpen,
  ariaLabel,
  disabled,
  expanded,
  onClearSelectedSkill,
  onActionClick,
  onAttachmentClick,
  onPhotoClick,
  onRefreshSkills,
  onSelectSkill,
  onSkillClick,
  onChange,
  onCompositionEnd,
  onCompositionStart,
  onKeyDown,
  onTextareaInput,
  placeholder,
  rows,
  selectedSkill,
  selectedSkillClearLabel,
  skillError,
  skillMenuEmptyLabel,
  skillMenuLoadingLabel,
  skillMenuOpen,
  skillMenuRefreshLabel,
  skillMenuTitle,
  skills,
  skillsLoading,
  textareaRef,
  toolbar,
  toolbarAriaLabel,
  attachmentAriaLabel,
  attachmentDisabled,
  photoDisabled,
  attachmentTitle,
  menuPhotoLabel,
  menuFileLabel,
  menuSkillLabel,
  menuUnavailableLabel,
  value,
}) => {
  const hasToolbar = toolbar !== undefined && toolbar !== null;

  return (
    <div className={`composer-row${expanded ? ' is-expanded' : ''}`}>
      <div className="composer-attachment-anchor">
        <button
          type="button"
          className="composer-plus-btn"
          aria-expanded={attachmentMenuOpen || skillMenuOpen}
          aria-haspopup="menu"
          aria-label={attachmentAriaLabel}
          title={attachmentTitle}
          disabled={attachmentDisabled}
          onClick={onAttachmentClick}
        >
          <PlusIcon />
        </button>

        {selectedSkill ? (
          <button
            type="button"
            className="composer-selected-skill"
            title={selectedSkillClearLabel}
            aria-label={selectedSkillClearLabel ?? selectedSkill.name}
            onClick={onClearSelectedSkill}
          >
            {selectedSkill.name}
          </button>
        ) : null}

        {attachmentMenuOpen ? (
          <AttachmentMenu
            fileLabel={menuFileLabel}
            onPhotoClick={photoDisabled ? undefined : onPhotoClick}
            onSkillClick={onSelectSkill ? onSkillClick : undefined}
            photoLabel={menuPhotoLabel}
            skillLabel={menuSkillLabel}
            unavailableLabel={menuUnavailableLabel}
          />
        ) : null}

        {skillMenuOpen ? (
          <ComposerSkillMenu
            emptyLabel={skillMenuEmptyLabel}
            error={skillError}
            loading={skillsLoading}
            loadingLabel={skillMenuLoadingLabel}
            onRefresh={() => {
              ignorePromise(Promise.resolve(onRefreshSkills?.()));
            }}
            onSelect={onSelectSkill ?? (() => undefined)}
            refreshLabel={skillMenuRefreshLabel}
            skills={skills}
            title={skillMenuTitle}
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
};

function AttachmentMenu({
  fileLabel,
  onPhotoClick,
  onSkillClick,
  photoLabel,
  skillLabel,
  unavailableLabel,
}: {
  fileLabel: string;
  onPhotoClick?: () => void;
  onSkillClick?: () => void;
  photoLabel: string;
  skillLabel: string;
  unavailableLabel: string;
}) {
  return (
    <div className="composer-attachment-menu" role="menu">
      {onPhotoClick ? (
        <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onPhotoClick}>
          {photoLabel}
        </button>
      ) : (
        <DisabledAttachmentMenuItem label={photoLabel} unavailableLabel={unavailableLabel} />
      )}
      <DisabledAttachmentMenuItem label={fileLabel} unavailableLabel={unavailableLabel} />
      {onSkillClick ? (
        <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onSkillClick}>
          {skillLabel}
        </button>
      ) : (
        <DisabledAttachmentMenuItem label={skillLabel} unavailableLabel={unavailableLabel} />
      )}
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
