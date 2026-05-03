'use client';

import { nodeLibraryForCopy } from '@/components/workflow/workflowNodeMeta';
import { localizeWorkflowValidationError } from '@/lib/i18n/workflowValidation';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type {
  AutosaveState,
  WorkflowCanvasDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

interface WorkflowCanvasSidebarProps {
  isOpen: boolean;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  importSessionID: string;
  importLoading: boolean;
  workflowCopy: WorkflowCopy;
  nodeLibraryTypes?: readonly WorkflowNodeType[];
  localizeValidationError?: (message: string, locale: WebLocale) => string;
  onToggle: () => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onOpenSettings: () => void;
  onBack: () => void;
  onSave: () => void;
}

const DEFAULT_NEW_NODE_POSITION: WorkflowCanvasPosition = { x: 350, y: 250 };

function IconPanelLeftClose(props: { size?: number }) {
  const { size = 20 } = props;
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M12.7 7.2l-2.5 2.8 2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function IconPanelLeftOpen(props: { size?: number }) {
  const { size = 20 } = props;
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M10.2 7.2l2.5 2.8-2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function IconSettings(props: { size?: number }) {
  const { size = 18 } = props;
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path
        d="M10 3.25 11.35 2.5l1.1 1.9 1.6.4 1.55-1.15 1.4 1.4-1.15 1.55.4 1.6 1.9 1.1-.75 1.35.75 1.35-1.9 1.1-.4 1.6 1.15 1.55-1.4 1.4-1.55-1.15-1.6.4-1.1 1.9L10 16.75l-1.35.75-1.1-1.9-1.6-.4-1.55 1.15-1.4-1.4 1.15-1.55-.4-1.6-1.9-1.1.75-1.35-.75-1.35 1.9-1.1.4-1.6L3 5.05l1.4-1.4 1.55 1.15 1.6-.4 1.1-1.9L10 3.25Z"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinejoin="round"
      />
      <circle cx="10" cy="10" r="2.6" stroke="currentColor" strokeWidth="1.3" />
    </svg>
  );
}

export function WorkflowCanvasSidebar(props: WorkflowCanvasSidebarProps) {
  const { copy, locale } = useWebLocale();
  const {
    isOpen,
    autosaveState,
    workflowCopy,
    nodeLibraryTypes,
    localizeValidationError,
    onToggle,
    onAddNode,
    onOpenSettings,
    onBack,
    onSave,
  } = props;
  const nodeLibrary = nodeLibraryForCopy(workflowCopy, nodeLibraryTypes);
  const resolveValidationError = localizeValidationError ?? localizeWorkflowValidationError;

  return (
    <aside className={`workflow-arch-sidebar ${isOpen ? 'workflow-arch-sidebar--open' : ''}`}>
      <div className="workflow-arch-sidebar-top">
        <div className="workflow-arch-sidebar-controls">
          <button
            type="button"
            className="workflow-arch-toggle"
            onClick={onToggle}
            aria-label={isOpen ? workflowCopy.sidebarCollapseAria : workflowCopy.sidebarExpandAria}
          >
            {isOpen ? <IconPanelLeftClose /> : <IconPanelLeftOpen />}
          </button>
          <button
            type="button"
            className={isOpen ? 'workflow-arch-toggle workflow-arch-settings-trigger' : 'workflow-arch-toggle workflow-arch-settings-trigger workflow-arch-settings-trigger--hidden'}
            onClick={onOpenSettings}
            aria-label={workflowCopy.sidebarOpenSettingsAria}
          >
            <IconSettings />
          </button>
        </div>
        <div className={`workflow-arch-sidebar-title ${isOpen ? 'workflow-arch-sidebar-title--open' : ''}`}>
          <h1>
            {workflowCopy.sidebarTitleMain}
            <br />
            <span>{workflowCopy.sidebarTitleSub}</span>
          </h1>
          <p>{workflowCopy.sidebarLibraryTitle}</p>
        </div>
      </div>

      <div className="workflow-arch-library">
        {nodeLibrary.map((item) => (
          <button
            type="button"
            key={item.id}
            className="workflow-arch-library-item"
            title={isOpen ? undefined : item.label}
            onClick={() => onAddNode(item.id, DEFAULT_NEW_NODE_POSITION)}
          >
            <span
              className={isOpen ? 'workflow-arch-library-initial workflow-arch-library-initial--hidden' : 'workflow-arch-library-initial'}
              aria-hidden
            >
              {item.label.slice(0, 1)}
            </span>
            <span className={`workflow-arch-library-copy ${isOpen ? 'workflow-arch-library-copy--open' : ''}`}>
              <strong>{item.label}</strong>
              <small>{workflowCopy.sidebarAddToFlow}</small>
            </span>
          </button>
        ))}
      </div>

      <div className="workflow-arch-sidebar-footer">
        <button
          type="button"
          className={`workflow-arch-save-status workflow-arch-save-status--${autosaveState.phase}`}
          onClick={onSave}
          aria-label={workflowCopy.sidebarSaveAria(resolveAutosaveLabel(autosaveState, workflowCopy, locale, resolveValidationError))}
          title={resolveAutosaveLabel(autosaveState, workflowCopy, locale, resolveValidationError)}
          aria-live="polite"
        >
          <span className="workflow-arch-save-status-dot" aria-hidden />
          <p>{resolveAutosaveLabel(autosaveState, workflowCopy, locale, resolveValidationError)}</p>
        </button>
        <button type="button" className="workflow-arch-footer-action" onClick={onBack}>
          <span aria-hidden>←</span>
          <span className={isOpen ? 'workflow-arch-footer-copy workflow-arch-footer-copy--open' : 'workflow-arch-footer-copy'}>
            {workflowCopy.sidebarReturn}
          </span>
        </button>
      </div>
    </aside>
  );
}

function resolveAutosaveLabel(
  state: AutosaveState,
  copy: WorkflowCopy,
  locale: WebLocale,
  localizeValidationError: (message: string, locale: WebLocale) => string,
): string {
  if (state.phase === 'saving') {
    return copy.autosaveSaving;
  }
  if (state.phase === 'saved') {
    return copy.autosaveSaved;
  }
  if (state.phase === 'error') {
    return state.message?.trim() || copy.autosaveErrorRetry;
  }
  if (state.phase === 'blocked') {
    if (state.message?.trim()) {
      return localizeValidationError(state.message, locale);
    }
    return copy.autosaveValidationBlocked;
  }
  return copy.autosaveIdle;
}
