'use client';

import type { FC, KeyboardEvent, ReactNode, RefObject } from 'react';
import { ComposerSkillMenu } from '@/components/ComposerSkillMenu';
import { ComposerToolbar } from '@/components/ComposerToolbar';
import { ignorePromise } from '@/lib/errors';
import type { AgentRuntimeType, ChatSelectedSkill, SkillPayload } from '@/lib/types';

export interface ComposerActionState {
  disabled: boolean;
  label: string;
  showStop: boolean;
  visible: boolean;
}

export interface ComposerInputRowProps {
  action: ComposerActionState;
  agentRuntime: AgentRuntimeType;
  attachmentMenuOpen: boolean;
  ariaLabel: string;
  codexModeDescription: string;
  codexModeTitle: string;
  codexToggleEnabled: boolean;
  disabled: boolean;
  expanded: boolean;
  featureMenuOpen: boolean;
  featureMenuTitle: string;
  onClearSelectedSkill?: () => void;
  onActionClick: () => void;
  onAttachmentClick: () => void;
  onFileClick: () => void;
  onFeatureClick: () => void;
  onRefreshSkills?: () => Promise<void> | void;
  onSelectSkill?: (skill: SkillPayload) => void;
  onSwitchAgentRuntime?: (runtime: AgentRuntimeType) => void;
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
  fileDisabled: boolean;
  fileUnavailableLabel: string;
  attachmentTitle: string;
  codexRuntimeLabel: string;
  ghostRuntimeLabel: string;
  menuFileLabel: string;
  menuFeatureLabel: string;
  menuSkillLabel: string;
  selectedSkillClearLabel?: string;
  menuUnavailableLabel: string;
  value: string;
}

export const ComposerInputRow: FC<ComposerInputRowProps> = ({
  action,
  agentRuntime,
  attachmentMenuOpen,
  ariaLabel,
  codexModeDescription,
  codexModeTitle,
  codexToggleEnabled,
  disabled,
  expanded,
  featureMenuOpen,
  featureMenuTitle,
  onClearSelectedSkill,
  onActionClick,
  onAttachmentClick,
  onFileClick,
  onFeatureClick,
  onRefreshSkills,
  onSelectSkill,
  onSwitchAgentRuntime,
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
  fileDisabled,
  fileUnavailableLabel,
  attachmentTitle,
  codexRuntimeLabel,
  ghostRuntimeLabel,
  menuFileLabel,
  menuFeatureLabel,
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
          aria-expanded={attachmentMenuOpen || skillMenuOpen || featureMenuOpen}
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
            onFileClick={fileDisabled ? undefined : onFileClick}
            onFeatureClick={onSwitchAgentRuntime ? onFeatureClick : undefined}
            onSkillClick={onSelectSkill ? onSkillClick : undefined}
            fileLabel={menuFileLabel}
            featureLabel={menuFeatureLabel}
            fileUnavailableLabel={fileUnavailableLabel}
            skillLabel={menuSkillLabel}
            unavailableLabel={menuUnavailableLabel}
          />
        ) : null}

        {featureMenuOpen ? (
          <ComposerFeatureMenu
            agentRuntime={agentRuntime}
            codexModeDescription={codexModeDescription}
            codexModeTitle={codexModeTitle}
            codexRuntimeLabel={codexRuntimeLabel}
            codexToggleEnabled={codexToggleEnabled}
            ghostRuntimeLabel={ghostRuntimeLabel}
            onSwitchAgentRuntime={onSwitchAgentRuntime}
            title={featureMenuTitle}
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
  onFileClick,
  onFeatureClick,
  onSkillClick,
  fileLabel,
  featureLabel,
  fileUnavailableLabel,
  skillLabel,
  unavailableLabel,
}: {
  onFileClick?: () => void;
  onFeatureClick?: () => void;
  onSkillClick?: () => void;
  fileLabel: string;
  featureLabel: string;
  fileUnavailableLabel: string;
  skillLabel: string;
  unavailableLabel: string;
}) {
  return (
    <div className="composer-attachment-menu" role="menu">
      {onFileClick ? (
        <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onFileClick}>
          <AttachmentMenuIcon kind="file" />
          <span className="composer-attachment-menu-label">{fileLabel}</span>
        </button>
      ) : (
        <DisabledAttachmentMenuItem icon="file" label={fileLabel} unavailableLabel={fileUnavailableLabel} />
      )}
      {onFeatureClick ? (
        <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onFeatureClick}>
          <AttachmentMenuIcon kind="feature" />
          <span className="composer-attachment-menu-label">{featureLabel}</span>
        </button>
      ) : (
        <DisabledAttachmentMenuItem icon="feature" label={featureLabel} unavailableLabel={unavailableLabel} />
      )}
      {onSkillClick ? (
        <button type="button" className="composer-attachment-menu-item" role="menuitem" onClick={onSkillClick}>
          <AttachmentMenuIcon kind="skill" />
          <span className="composer-attachment-menu-label">{skillLabel}</span>
        </button>
      ) : (
        <DisabledAttachmentMenuItem icon="skill" label={skillLabel} unavailableLabel={unavailableLabel} />
      )}
    </div>
  );
}

function DisabledAttachmentMenuItem({
  icon,
  label,
  unavailableLabel,
}: {
  icon: 'feature' | 'file' | 'skill';
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
      <AttachmentMenuIcon kind={icon} />
      <span className="composer-attachment-menu-label">{label}</span>
    </button>
  );
}

function AttachmentMenuIcon({ kind }: { kind: 'feature' | 'file' | 'skill' }) {
  return (
    <span className="composer-attachment-menu-icon" aria-hidden="true">
      {kind === 'feature' ? <FeatureIcon /> : kind === 'file' ? <FileIcon /> : <SkillIcon />}
    </span>
  );
}

function ComposerFeatureMenu({
  agentRuntime,
  codexModeDescription,
  codexModeTitle,
  codexRuntimeLabel,
  codexToggleEnabled,
  ghostRuntimeLabel,
  onSwitchAgentRuntime,
  title,
}: {
  agentRuntime: AgentRuntimeType;
  codexModeDescription: string;
  codexModeTitle: string;
  codexRuntimeLabel: string;
  codexToggleEnabled: boolean;
  ghostRuntimeLabel: string;
  onSwitchAgentRuntime?: (runtime: AgentRuntimeType) => void;
  title: string;
}) {
  const codexModeEnabled = agentRuntime === 'codex';

  return (
    <div className="composer-feature-menu" role="menu" aria-label={title}>
      <div className="composer-skill-menu-head">
        <span>{title}</span>
      </div>
      <div className="composer-feature-mode-switch" role="group" aria-label={codexModeTitle}>
        <button
          className={!codexModeEnabled ? 'is-active' : ''}
          type="button"
          aria-pressed={!codexModeEnabled}
          onClick={() => onSwitchAgentRuntime?.('ghost')}
        >
          {ghostRuntimeLabel}
        </button>
        <button
          className={codexModeEnabled ? 'is-active' : ''}
          type="button"
          aria-pressed={codexModeEnabled}
          disabled={!codexToggleEnabled}
          onClick={() => onSwitchAgentRuntime?.('codex')}
        >
          {codexRuntimeLabel}
        </button>
      </div>
      <div className="composer-feature-status" role="status">{codexModeDescription}</div>
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

function FileIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path
        d="M7.4 10.8L10.95 7.25C11.8 6.4 13.15 6.4 14 7.25C14.85 8.1 14.85 9.45 14 10.3L9.25 15.05C7.95 16.35 5.85 16.35 4.55 15.05C3.25 13.75 3.25 11.65 4.55 10.35L9.4 5.5C11.15 3.75 14 3.75 15.75 5.5C17.5 7.25 17.5 10.1 15.75 11.85L11.1 16.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
    </svg>
  );
}

function FeatureIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path
        d="M10 4.25L10.9 7.1L13.75 8L10.9 8.9L10 11.75L9.1 8.9L6.25 8L9.1 7.1L10 4.25Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
      <path
        d="M14.5 11.75L15.05 13.45L16.75 14L15.05 14.55L14.5 16.25L13.95 14.55L12.25 14L13.95 13.45L14.5 11.75Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
      <path
        d="M5.5 11.5L5.95 12.95L7.4 13.4L5.95 13.85L5.5 15.3L5.05 13.85L3.6 13.4L5.05 12.95L5.5 11.5Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
    </svg>
  );
}

function SkillIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path
        d="M8 3.75H5.25C4.42 3.75 3.75 4.42 3.75 5.25V8H8V3.75Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
      <path
        d="M16.25 8V5.25C16.25 4.42 15.58 3.75 14.75 3.75H12V8H16.25Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
      <path
        d="M8 16.25V12H3.75V14.75C3.75 15.58 4.42 16.25 5.25 16.25H8Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
      <path
        d="M12 12H16.25V14.75C16.25 15.58 15.58 16.25 14.75 16.25H12V12Z"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="1.55"
      />
    </svg>
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
