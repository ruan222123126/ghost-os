'use client';

import { useState } from 'react';
import { WorkflowCanvasPropertiesPanel } from '@/components/workflow/WorkflowCanvasPropertiesPanel';
import {
  WorkflowCanvasSettingsModal,
  type WorkflowImportControls,
} from '@/components/workflow/WorkflowCanvasSettingsModal';
import { WorkflowCanvasSidebar } from '@/components/workflow/WorkflowCanvasSidebar';
import { WorkflowCanvasStage } from '@/components/workflow/WorkflowCanvasStage';
import { useWorkflowSaveShortcut } from '@/hooks/workflow/useWorkflowSaveShortcut';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WebLocale } from '@/lib/i18n/locale';
import type { PresetPayload } from '@/lib/types';
import type {
  AutosaveState,
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasDraft,
  WorkflowEditorKind,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

interface WorkflowCanvasWorkbenchProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  validationErrors: string[];
  importControls?: WorkflowImportControls;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading?: boolean;
  presetError?: string;
  workflowCopy?: WorkflowCopy;
  nodeLibraryTypes?: readonly WorkflowNodeType[];
  localizeValidationError?: (message: string, locale: WebLocale) => string;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
  onSave: () => void;
  onBack: () => void;
}

export function WorkflowCanvasWorkbench(props: WorkflowCanvasWorkbenchProps) {
  const { copy } = useWebLocale();
  const defaults = resolveWorkbenchDefaults(props, copy.workflow);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const selectedNode = findSelectedWorkflowNode(props.draft);
  useWorkflowSaveShortcut(props.onSave);

  function toggleSidebar() {
    setIsSidebarOpen(invertBoolean);
  }

  function openSettings() {
    setIsSettingsOpen(true);
  }

  function closeSettings() {
    setIsSettingsOpen(false);
  }

  function closePropertiesPanel() {
    props.onSelectNode(undefined);
  }

  return (
    <main className="workflow-arch-root">
      <div className="workflow-arch-watermark">{defaults.workflowCopy.watermark}</div>
      <WorkbenchSidebarPanel
        defaults={defaults}
        isOpen={isSidebarOpen}
        onToggle={toggleSidebar}
        onOpenSettings={openSettings}
        workbench={props}
      />
      <WorkbenchStagePanel defaults={defaults} workbench={props} />
      <WorkbenchPropertiesPanel
        defaults={defaults}
        onClose={closePropertiesPanel}
        selectedNode={selectedNode}
        workbench={props}
      />
      <WorkbenchSettingsPanel
        defaults={defaults}
        open={isSettingsOpen}
        onClose={closeSettings}
        workbench={props}
      />
    </main>
  );
}

interface WorkflowCanvasWorkbenchDefaults {
  presetError: string;
  presetLoading: boolean;
  workflowCopy: WorkflowCopy;
}

interface WorkbenchPanelProps {
  defaults: WorkflowCanvasWorkbenchDefaults;
  workbench: WorkflowCanvasWorkbenchProps;
}

interface WorkbenchSidebarPanelProps extends WorkbenchPanelProps {
  isOpen: boolean;
  onOpenSettings: () => void;
  onToggle: () => void;
}

function WorkbenchSidebarPanel(props: WorkbenchSidebarPanelProps) {
  const { defaults, isOpen, onOpenSettings, onToggle, workbench } = props;
  return (
    <WorkflowCanvasSidebar
      isOpen={isOpen}
      autosaveState={workbench.autosaveState}
      workflowCopy={defaults.workflowCopy}
      nodeLibraryTypes={workbench.nodeLibraryTypes}
      localizeValidationError={workbench.localizeValidationError}
      onToggle={onToggle}
      onAddNode={workbench.onAddNode}
      onOpenSettings={onOpenSettings}
      onBack={workbench.onBack}
      onSave={workbench.onSave}
    />
  );
}

function WorkbenchStagePanel(props: WorkbenchPanelProps) {
  const { defaults, workbench } = props;
  return (
    <WorkflowCanvasStage
      editorKind={workbench.editorKind}
      draft={workbench.draft}
      actionError={workbench.actionError}
      validationErrors={workbench.validationErrors}
      workflowCopy={defaults.workflowCopy}
      localizeValidationError={workbench.localizeValidationError}
      onSelectNode={workbench.onSelectNode}
      onMoveNode={workbench.onMoveNode}
      onConnectNodes={workbench.onConnectNodes}
      onDeleteEdge={workbench.onDeleteEdge}
      onDuplicateNode={workbench.onDuplicateNode}
      onDeleteNode={workbench.onDeleteNode}
    />
  );
}

interface WorkbenchPropertiesPanelProps extends WorkbenchPanelProps {
  onClose: () => void;
  selectedNode?: WorkflowCanvasNodeDraft;
}

function WorkbenchPropertiesPanel(props: WorkbenchPropertiesPanelProps) {
  const { defaults, onClose, selectedNode, workbench } = props;
  return (
    <WorkflowCanvasPropertiesPanel
      editorKind={workbench.editorKind}
      draft={workbench.draft}
      selectedNode={selectedNode}
      agentRuntimeCatalog={workbench.agentRuntimeCatalog}
      agentRuntimeLoading={workbench.agentRuntimeLoading}
      agentRuntimeError={workbench.agentRuntimeError}
      presets={workbench.presets}
      presetLoading={defaults.presetLoading}
      presetError={defaults.presetError}
      onClose={onClose}
      onUpdateNode={workbench.onUpdateNode}
      onDeleteNode={workbench.onDeleteNode}
    />
  );
}

interface WorkbenchSettingsPanelProps extends WorkbenchPanelProps {
  onClose: () => void;
  open: boolean;
}

function WorkbenchSettingsPanel(props: WorkbenchSettingsPanelProps) {
  const { defaults, onClose, open, workbench } = props;
  return (
    <WorkflowCanvasSettingsModal
      open={open}
      schedule={workbench.draft.schedule}
      importControls={workbench.importControls}
      workflowCopy={defaults.workflowCopy}
      onClose={onClose}
      onScheduleChange={workbench.onScheduleChange}
    />
  );
}

function resolveWorkbenchDefaults(
  props: WorkflowCanvasWorkbenchProps,
  fallbackWorkflowCopy: WorkflowCopy,
): WorkflowCanvasWorkbenchDefaults {
  return {
    presetError: props.presetError ?? '',
    presetLoading: props.presetLoading ?? false,
    workflowCopy: props.workflowCopy ?? fallbackWorkflowCopy,
  };
}

function findSelectedWorkflowNode(draft: WorkflowCanvasDraft): WorkflowCanvasNodeDraft | undefined {
  return draft.nodes.find((node) => node.id === draft.selectedNodeId);
}

function invertBoolean(value: boolean): boolean {
  return !value;
}
