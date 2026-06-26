'use client';

import { nodeLibraryForCopy } from '@/components/workflow/workflowNodeMeta';
import {
  IconPanelLeftClose,
  IconPanelLeftOpen,
  IconSettings,
} from '@/components/workflow/WorkflowCanvasSidebarIcons';
import { localizeWorkflowValidationError } from '@/lib/i18n/workflowValidation';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type {
  AutosaveState,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

interface WorkflowCanvasSidebarProps {
  isOpen: boolean;
  autosaveState: AutosaveState;
  workflowCopy: WorkflowCopy;
  nodeLibraryTypes?: readonly WorkflowNodeType[];
  localizeValidationError?: (message: string, locale: WebLocale) => string;
  onToggle: () => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onOpenSettings: () => void;
  onBack: () => void;
  onSave: () => void;
}

interface SidebarTopProps {
  isOpen: boolean;
  workflowCopy: WorkflowCopy;
  onOpenSettings: () => void;
  onToggle: () => void;
}

interface NodeLibraryProps {
  isOpen: boolean;
  nodeLibrary: WorkflowNodeLibraryItem[];
  workflowCopy: WorkflowCopy;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
}

interface SidebarFooterProps {
  autosaveLabel: string;
  autosaveState: AutosaveState;
  isOpen: boolean;
  workflowCopy: WorkflowCopy;
  onBack: () => void;
  onSave: () => void;
}

interface ResolveAutosaveLabelOptions {
  copy: WorkflowCopy;
  locale: WebLocale;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  state: AutosaveState;
}

type WorkflowNodeLibraryItem = ReturnType<typeof nodeLibraryForCopy>[number];

const DEFAULT_NEW_NODE_POSITION: WorkflowCanvasPosition = { x: 350, y: 250 };

export function WorkflowCanvasSidebar(props: WorkflowCanvasSidebarProps) {
  const { locale } = useWebLocale();
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
  const autosaveLabel = resolveAutosaveLabel({
    state: autosaveState,
    copy: workflowCopy,
    locale,
    localizeValidationError: resolveValidationError,
  });

  return (
    <aside className={`workflow-arch-sidebar ${isOpen ? 'workflow-arch-sidebar--open' : ''}`}>
      <SidebarTop
        isOpen={isOpen}
        workflowCopy={workflowCopy}
        onOpenSettings={onOpenSettings}
        onToggle={onToggle}
      />
      <NodeLibrary
        isOpen={isOpen}
        nodeLibrary={nodeLibrary}
        workflowCopy={workflowCopy}
        onAddNode={onAddNode}
      />
      <SidebarFooter
        autosaveLabel={autosaveLabel}
        autosaveState={autosaveState}
        isOpen={isOpen}
        workflowCopy={workflowCopy}
        onBack={onBack}
        onSave={onSave}
      />
    </aside>
  );
}

function SidebarTop(props: SidebarTopProps) {
  const { isOpen, workflowCopy, onOpenSettings, onToggle } = props;

  return (
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
          className={settingsTriggerClass(isOpen)}
          onClick={onOpenSettings}
          aria-label={workflowCopy.sidebarOpenSettingsAria}
        >
          <IconSettings />
        </button>
      </div>
      <div className={sidebarTitleClass(isOpen)}>
        <h1>
          {workflowCopy.sidebarTitleMain}
          <br />
          <span>{workflowCopy.sidebarTitleSub}</span>
        </h1>
        <p>{workflowCopy.sidebarLibraryTitle}</p>
      </div>
    </div>
  );
}

function NodeLibrary(props: NodeLibraryProps) {
  const { isOpen, nodeLibrary, workflowCopy, onAddNode } = props;

  return (
    <div className="workflow-arch-library">
      {nodeLibrary.map((item) => (
        <NodeLibraryButton
          key={item.id}
          isOpen={isOpen}
          item={item}
          workflowCopy={workflowCopy}
          onAddNode={onAddNode}
        />
      ))}
    </div>
  );
}

function NodeLibraryButton(props: {
  isOpen: boolean;
  item: WorkflowNodeLibraryItem;
  workflowCopy: WorkflowCopy;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
}) {
  const { isOpen, item, workflowCopy, onAddNode } = props;

  return (
    <button
      type="button"
      className="workflow-arch-library-item"
      title={isOpen ? undefined : item.label}
      onClick={() => onAddNode(item.id, DEFAULT_NEW_NODE_POSITION)}
    >
      <span className={libraryInitialClass(isOpen)} aria-hidden>
        {item.label.slice(0, 1)}
      </span>
      <span className={libraryCopyClass(isOpen)}>
        <strong>{item.label}</strong>
        <small>{workflowCopy.sidebarAddToFlow}</small>
      </span>
    </button>
  );
}

function SidebarFooter(props: SidebarFooterProps) {
  const {
    autosaveLabel,
    autosaveState,
    isOpen,
    workflowCopy,
    onBack,
    onSave,
  } = props;

  return (
    <div className="workflow-arch-sidebar-footer">
      <button
        type="button"
        className={`workflow-arch-save-status workflow-arch-save-status--${autosaveState.phase}`}
        onClick={onSave}
        aria-label={workflowCopy.sidebarSaveAria(autosaveLabel)}
        title={autosaveLabel}
        aria-live="polite"
      >
        <span className="workflow-arch-save-status-dot" aria-hidden />
        <p>{autosaveLabel}</p>
      </button>
      <button type="button" className="workflow-arch-footer-action" onClick={onBack}>
        <span aria-hidden>←</span>
        <span className={footerCopyClass(isOpen)}>
          {workflowCopy.sidebarReturn}
        </span>
      </button>
    </div>
  );
}

function resolveAutosaveLabel(options: ResolveAutosaveLabelOptions): string {
  const { state, copy, locale, localizeValidationError } = options;

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

function settingsTriggerClass(isOpen: boolean): string {
  if (isOpen) {
    return 'workflow-arch-toggle workflow-arch-settings-trigger';
  }
  return 'workflow-arch-toggle workflow-arch-settings-trigger workflow-arch-settings-trigger--hidden';
}

function sidebarTitleClass(isOpen: boolean): string {
  if (isOpen) {
    return 'workflow-arch-sidebar-title workflow-arch-sidebar-title--open';
  }
  return 'workflow-arch-sidebar-title';
}

function libraryInitialClass(isOpen: boolean): string {
  if (isOpen) {
    return 'workflow-arch-library-initial workflow-arch-library-initial--hidden';
  }
  return 'workflow-arch-library-initial';
}

function libraryCopyClass(isOpen: boolean): string {
  if (isOpen) {
    return 'workflow-arch-library-copy workflow-arch-library-copy--open';
  }
  return 'workflow-arch-library-copy';
}

function footerCopyClass(isOpen: boolean): string {
  if (isOpen) {
    return 'workflow-arch-footer-copy workflow-arch-footer-copy--open';
  }
  return 'workflow-arch-footer-copy';
}
