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
      <WorkflowCanvasSidebar
        isOpen={isSidebarOpen}
        autosaveState={props.autosaveState}
        workflowCopy={defaults.workflowCopy}
        nodeLibraryTypes={props.nodeLibraryTypes}
        localizeValidationError={props.localizeValidationError}
        onToggle={toggleSidebar}
        onAddNode={props.onAddNode}
        onOpenSettings={openSettings}
        onBack={props.onBack}
        onSave={props.onSave}
      />
      <WorkflowCanvasStage
        editorKind={props.editorKind}
        draft={props.draft}
        actionError={props.actionError}
        validationErrors={props.validationErrors}
        workflowCopy={defaults.workflowCopy}
        localizeValidationError={props.localizeValidationError}
        onSelectNode={props.onSelectNode}
        onMoveNode={props.onMoveNode}
        onConnectNodes={props.onConnectNodes}
        onDeleteEdge={props.onDeleteEdge}
        onDuplicateNode={props.onDuplicateNode}
        onDeleteNode={props.onDeleteNode}
      />
      <WorkflowCanvasPropertiesPanel
        editorKind={props.editorKind}
        draft={props.draft}
        selectedNode={selectedNode}
        agentRuntimeCatalog={props.agentRuntimeCatalog}
        agentRuntimeLoading={props.agentRuntimeLoading}
        agentRuntimeError={props.agentRuntimeError}
        presets={props.presets}
        presetLoading={defaults.presetLoading}
        presetError={defaults.presetError}
        onClose={closePropertiesPanel}
        onUpdateNode={props.onUpdateNode}
        onDeleteNode={props.onDeleteNode}
      />
      <WorkflowCanvasSettingsModal
        open={isSettingsOpen}
        schedule={props.draft.schedule}
        importControls={props.importControls}
        workflowCopy={defaults.workflowCopy}
        onClose={closeSettings}
        onScheduleChange={props.onScheduleChange}
      />
    </main>
  );
}

interface WorkflowCanvasWorkbenchDefaults {
  presetError: string;
  presetLoading: boolean;
  workflowCopy: WorkflowCopy;
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
